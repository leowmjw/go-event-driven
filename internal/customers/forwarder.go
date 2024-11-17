package customers

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type EventForwarder struct {
	customersDB     *mongo.Database
	orderingDB      *mongo.Database
	client          *mongo.Client
	stopChan        chan struct{}
	processingMutex sync.Mutex
	isProcessing    bool
}

func NewEventForwarder(mongoURI string) *EventForwarder {
	ctx := context.Background()
	clientOpts := options.Client().ApplyURI(mongoURI).
		SetServerSelectionTimeout(2 * time.Second)
	
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		panic(err)
	}

	// Verify that we can connect to MongoDB and that it's a replica set
	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		panic(fmt.Errorf("failed to ping MongoDB: %v", err))
	}

	return &EventForwarder{
		customersDB: client.Database("CustomersDB"),
		orderingDB:  client.Database("OrderingDB"),
		client:      client,
		stopChan:    make(chan struct{}),
	}
}

func (f *EventForwarder) Start() {
	go f.processEvents()
}

func (f *EventForwarder) Stop() {
	close(f.stopChan)
	if err := f.client.Disconnect(context.Background()); err != nil {
		log.Printf("Error disconnecting MongoDB client: %v", err)
	}
}

func (f *EventForwarder) processEvents() {
	ticker := time.NewTicker(100 * time.Millisecond) // Process events more frequently during testing
	defer ticker.Stop()

	for {
		select {
		case <-f.stopChan:
			return
		case <-ticker.C:
			f.processingMutex.Lock()
			if f.isProcessing {
				f.processingMutex.Unlock()
				continue
			}
			f.isProcessing = true
			f.processingMutex.Unlock()

			if err := f.forwardPendingEvents(); err != nil {
				log.Printf("Error forwarding events: %v", err)
			}

			f.processingMutex.Lock()
			f.isProcessing = false
			f.processingMutex.Unlock()
		}
	}
}

func (f *EventForwarder) forwardPendingEvents() error {
	ctx := context.Background()

	// Find pending events
	cursor, err := f.customersDB.Collection("outbox").Find(ctx, bson.M{
		"status": "pending",
	}, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return fmt.Errorf("failed to find pending events: %v", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var event OutboxEvent
		if err := cursor.Decode(&event); err != nil {
			log.Printf("Error decoding event: %v", err)
			continue
		}

		// Process the event
		if err := f.processEvent(ctx, &event); err != nil {
			log.Printf("Error processing event: %v", err)
			// Update event status to failed
			update := bson.M{
				"$set": bson.M{
					"status":     "failed",
					"error":      err.Error(),
					"updated_at": time.Now(),
				},
				"$inc": bson.M{
					"retry_count": 1,
				},
			}
			if _, err := f.customersDB.Collection("outbox").UpdateOne(ctx,
				bson.M{"_id": event.ID}, update); err != nil {
				log.Printf("Error updating failed event: %v", err)
			}
			continue
		}

		// Mark event as processed
		update := bson.M{
			"$set": bson.M{
				"status":     "processed",
				"updated_at": time.Now(),
			},
		}
		if _, err := f.customersDB.Collection("outbox").UpdateOne(ctx,
			bson.M{"_id": event.ID}, update); err != nil {
			log.Printf("Error updating processed event: %v", err)
		}
	}

	return nil
}

func (f *EventForwarder) processEvent(ctx context.Context, event *OutboxEvent) error {
	// Start a session for transaction
	session, err := f.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %v", err)
	}
	defer session.EndSession(ctx)

	// Start transaction
	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		var customer Customer
		if err := bson.Unmarshal(event.Payload, &customer); err != nil {
			return nil, fmt.Errorf("failed to unmarshal customer: %v", err)
		}

		// Create or update projection
		projection := CustomerProjection{
			ID:        customer.ID,
			Name:      customer.Name,
			Email:     customer.Email,
			CreatedAt: customer.CreatedAt,
			UpdatedAt: customer.UpdatedAt,
		}

		filter := bson.M{"_id": customer.ID}
		update := bson.M{"$set": projection}
		opts := options.Update().SetUpsert(true)

		_, err := f.orderingDB.Collection("projection_customers").UpdateOne(
			sessCtx, filter, update, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to update projection: %v", err)
		}

		return nil, nil
	})

	return err
}
