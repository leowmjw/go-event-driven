package customers

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type EventForwarderImpl struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
	stopChan   chan struct{}
}

func NewEventForwarder(mongoURI string) *EventForwarderImpl {
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	db := client.Database("CustomersDB")
	collection := db.Collection("outbox")

	return &EventForwarderImpl{
		client:     client,
		db:         db,
		collection: collection,
		stopChan:   make(chan struct{}),
	}
}

func (f *EventForwarderImpl) Forward(event OutboxEvent) error {
	_, err := f.collection.InsertOne(context.Background(), event)
	if err != nil {
		return err
	}
	return nil
}

func (f *EventForwarderImpl) Start() {
	go f.processOutboxEvents()
}

func (f *EventForwarderImpl) Stop() {
	close(f.stopChan)
	if err := f.client.Disconnect(context.Background()); err != nil {
		log.Printf("Error disconnecting from MongoDB: %v", err)
	}
}

func (f *EventForwarderImpl) processOutboxEvents() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-f.stopChan:
			return
		case <-ticker.C:
			ctx := context.Background()
			filter := bson.M{"status": "pending"}

			// Find pending events
			cursor, err := f.collection.Find(ctx, filter)
			if err != nil {
				log.Printf("Error finding pending events: %v", err)
				continue
			}

			var events []OutboxEvent
			if err := cursor.All(ctx, &events); err != nil {
				log.Printf("Error decoding events: %v", err)
				cursor.Close(ctx)
				continue
			}
			cursor.Close(ctx)

			for _, event := range events {
				// Process event (in a real application, this would publish to a message broker)
				log.Printf("Processing event: %s", event.EventType)

				// Mark event as processed
				_, err := f.collection.UpdateOne(
					ctx,
					bson.M{"_id": event.ID},
					bson.M{
						"$set": bson.M{
							"status":     "processed",
							"updatedAt": time.Now(),
						},
					},
				)
				if err != nil {
					log.Printf("Error updating event status: %v", err)
				}
			}
		}
	}
}
