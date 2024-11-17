package customers

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type EventForwarder struct {
	outboxCollection     *mongo.Collection
	projectionCollection *mongo.Collection
	eventChan           chan *OutboxEvent
	stopChan            chan struct{}
	maxRetries          int
	mongoClient         *mongo.Client
}

func NewEventForwarder(mongoURI string) *EventForwarder {
	// Create MongoDB client
	clientOpts := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(context.Background(), clientOpts)
	if err != nil {
		panic(err)
	}

	return &EventForwarder{
		outboxCollection:     client.Database("CustomersDB").Collection("outbox"),
		projectionCollection: client.Database("OrderingDB").Collection("projection_customers"),
		eventChan:           make(chan *OutboxEvent, 100),
		stopChan:            make(chan struct{}),
		maxRetries:          3, // Maximum number of retry attempts
		mongoClient:         client,
	}
}

func (f *EventForwarder) Start() {
	go f.pollOutbox()
	go f.processEvents()
	go f.retryFailedEvents() // Start retry processor
}

func (f *EventForwarder) Stop() {
	close(f.stopChan)
}

func (f *EventForwarder) pollOutbox() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-f.stopChan:
			return
		case <-ticker.C:
			event, err := f.claimNextEvent()
			if err != nil {
				if err != mongo.ErrNoDocuments {
					log.Printf("Error claiming next event: %v", err)
				}
				continue
			}

			if event != nil {
				f.eventChan <- event
			}
		}
	}
}

func (f *EventForwarder) retryFailedEvents() {
	ticker := time.NewTicker(1 * time.Minute) // Check failed events every minute
	defer ticker.Stop()

	for {
		select {
		case <-f.stopChan:
			return
		case <-ticker.C:
			err := f.processFailedEvents()
			if err != nil {
				log.Printf("Error processing failed events: %v", err)
			}
		}
	}
}

func (f *EventForwarder) processFailedEvents() error {
	ctx := context.Background()

	// Find failed events that haven't exceeded retry limit
	cursor, err := f.outboxCollection.Find(ctx, bson.M{
		"status": "failed",
		"retry_count": bson.M{"$lt": f.maxRetries},
		"last_retry_at": bson.M{"$lt": time.Now().Add(-5 * time.Minute)}, // Wait 5 minutes between retries
	}, options.Find().SetSort(bson.D{{"created_at", 1}})) // Maintain ordering

	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	var failedEvents []*OutboxEvent
	if err = cursor.All(ctx, &failedEvents); err != nil {
		return err
	}

	for _, event := range failedEvents {
		// Atomically update retry count and status
		result, err := f.outboxCollection.UpdateOne(ctx,
			bson.M{
				"_id": event.ID,
				"status": "failed",
				"retry_count": event.RetryCount, // Optimistic locking
			},
			bson.M{
				"$set": bson.M{
					"status": "pending",
					"last_retry_at": time.Now(),
					"retry_count": event.RetryCount + 1,
				},
			})

		if err != nil {
			log.Printf("Failed to reset event %s for retry: %v", event.ID.Hex(), err)
			continue
		}

		if result.MatchedCount == 0 {
			log.Printf("Event %s was already processed by another forwarder", event.ID.Hex())
			continue
		}

		log.Printf("Reset failed event %s for retry attempt %d", event.ID.Hex(), event.RetryCount+1)
	}

	return nil
}

func (f *EventForwarder) processEvents() {
	for {
		select {
		case <-f.stopChan:
			return
		case event := <-f.eventChan:
			err := f.handleEvent(event)
			if err != nil {
				log.Printf("Error handling event %s: %v", event.ID.Hex(), err)
				err = f.markEventAsFailed(event, err.Error())
				if err != nil {
					log.Printf("Error marking event %s as failed: %v", event.ID.Hex(), err)
				}
				continue
			}

			err = f.markEventAsProcessed(event)
			if err != nil {
				log.Printf("Error marking event %s as processed: %v", event.ID.Hex(), err)
			}
		}
	}
}

func (f *EventForwarder) claimNextEvent() (*OutboxEvent, error) {
	ctx := context.Background()
	now := time.Now()
	lockTimeout := time.Now().Add(-5 * time.Minute)

	// Find the next event that needs processing, maintaining strict order
	lastProcessedTime := f.getLastProcessedEventTime(ctx)

	var event OutboxEvent
	err := f.outboxCollection.FindOne(ctx, bson.M{
		"$or": []bson.M{
			{
				"status": "pending",
				"processed_up_to": bson.M{"$exists": false}, // First event
			},
			{
				"status": "pending",
				"created_at": bson.M{"$gt": lastProcessedTime}, // Events after last processed
			},
			{
				"status": "processing",
				"updated_at": bson.M{"$lt": lockTimeout},
				"created_at": bson.M{"$gt": lastProcessedTime},
			},
		},
	}, options.FindOne().SetSort(bson.D{{"created_at", 1}})).Decode(&event) // Ensure ordering by creation time

	if err != nil {
		return nil, err
	}

	// Update the event status to processing
	result, err := f.outboxCollection.UpdateOne(ctx,
		bson.M{"_id": event.ID},
		bson.M{
			"$set": bson.M{
				"status":     "processing",
				"updated_at": now,
			},
		})
	if err != nil {
		return nil, err
	}

	if result.MatchedCount == 0 {
		return nil, fmt.Errorf("event %s was claimed by another forwarder", event.ID.Hex())
	}

	return &event, nil
}

func (f *EventForwarder) getLastProcessedEventTime(ctx context.Context) time.Time {
	var lastEvent OutboxEvent
	err := f.outboxCollection.FindOne(ctx, bson.M{
		"status": "processed",
	}, options.FindOne().SetSort(bson.D{{"created_at", -1}})).Decode(&lastEvent)

	if err != nil {
		return time.Time{} // Return zero time if no events processed yet
	}

	return lastEvent.CreatedAt
}

func (f *EventForwarder) handleEvent(event *OutboxEvent) error {
	ctx := context.Background()

	// Start session for transaction
	session, err := f.mongoClient.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %v", err)
	}
	defer session.EndSession(ctx)

	// Start transaction
	err = session.StartTransaction()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %v", err)
	}

	// Convert payload to primitive.D
	payloadDoc, ok := event.Payload.(primitive.D)
	if !ok {
		session.AbortTransaction(ctx)
		return fmt.Errorf("invalid payload type: %T", event.Payload)
	}

	// Convert primitive.D to map
	payloadMap := make(map[string]interface{})
	for _, elem := range payloadDoc {
		payloadMap[elem.Key] = elem.Value
	}

	// Convert MongoDB DateTime to time.Time
	createdAt, ok := payloadMap["created_at"].(primitive.DateTime)
	if !ok {
		session.AbortTransaction(ctx)
		return fmt.Errorf("invalid created_at type: %T", payloadMap["created_at"])
	}
	updatedAt, ok := payloadMap["updated_at"].(primitive.DateTime)
	if !ok {
		session.AbortTransaction(ctx)
		return fmt.Errorf("invalid updated_at type: %T", payloadMap["updated_at"])
	}

	// Create customer projection from map
	projection := &CustomerProjection{
		ID:        payloadMap["_id"].(primitive.ObjectID),
		Name:      payloadMap["name"].(string),
		Email:     payloadMap["email"].(string),
		CreatedAt: createdAt.Time(),
		UpdatedAt: updatedAt.Time(),
	}

	switch event.EventType {
	case "CustomerCreated":
		_, err := session.Client().Database("OrderingDB").Collection("projection_customers").InsertOne(ctx, projection)
		if err != nil {
			session.AbortTransaction(ctx)
			return err
		}

	case "CustomerUpdated":
		result, err := session.Client().Database("OrderingDB").Collection("projection_customers").UpdateOne(ctx, 
			bson.M{"_id": projection.ID}, 
			bson.M{"$set": projection})
		if err != nil {
			session.AbortTransaction(ctx)
			return err
		}
		if result.MatchedCount == 0 {
			session.AbortTransaction(ctx)
			return fmt.Errorf("no matching projection found for customer %s", projection.ID.Hex())
		}

	default:
		session.AbortTransaction(ctx)
		return fmt.Errorf("unknown event type: %s", event.EventType)
	}

	return session.CommitTransaction(ctx)
}

func (f *EventForwarder) markEventAsProcessed(event *OutboxEvent) error {
	result, err := f.outboxCollection.UpdateOne(context.Background(),
		bson.M{
			"_id": event.ID,
			"status": "processing", // Only update if still in processing state
		},
		bson.M{"$set": bson.M{
			"status":     "processed",
			"updated_at": time.Now(),
			"processed_up_to": event.CreatedAt, // Mark the last successfully processed event time
		}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("event %s not found or already processed", event.ID.Hex())
	}
	return nil
}

func (f *EventForwarder) markEventAsFailed(event *OutboxEvent, errorMsg string) error {
	result, err := f.outboxCollection.UpdateOne(context.Background(),
		bson.M{
			"_id": event.ID,
			"status": "processing",
		},
		bson.M{"$set": bson.M{
			"status":        "failed",
			"error":         errorMsg,
			"updated_at":    time.Now(),
			"failed_at":     time.Now(),
			"retry_count":   bson.M{"$inc": 1},
			"last_retry_at": time.Now(),
		}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("event %s not found or already processed", event.ID.Hex())
	}
	return nil
}
