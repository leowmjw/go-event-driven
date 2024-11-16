package customers

import (
	"context"
	"log"
	"time"

	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EventForwarder struct {
	outboxCollection     *qmgo.Collection
	projectionCollection *qmgo.Collection
	eventChan           chan *OutboxEvent
	stopChan            chan struct{}
}

func NewEventForwarder(customersDB, orderingDB *qmgo.Database) *EventForwarder {
	return &EventForwarder{
		outboxCollection:     customersDB.Collection("outbox"),
		projectionCollection: orderingDB.Collection("projection_customers"),
		eventChan:           make(chan *OutboxEvent, 100),
		stopChan:            make(chan struct{}),
	}
}

func (f *EventForwarder) Start() {
	go f.pollOutbox()
	go f.processEvents()
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
			events, err := f.fetchPendingEvents()
			if err != nil {
				log.Printf("Error fetching pending events: %v", err)
				continue
			}

			for _, event := range events {
				f.eventChan <- event
			}
		}
	}
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
				continue
			}

			err = f.markEventAsProcessed(event)
			if err != nil {
				log.Printf("Error marking event %s as processed: %v", event.ID.Hex(), err)
			}
		}
	}
}

func (f *EventForwarder) fetchPendingEvents() ([]*OutboxEvent, error) {
	var events []*OutboxEvent
	err := f.outboxCollection.Find(context.Background(), 
		bson.M{"status": "pending"}).All(&events)
	return events, err
}

func (f *EventForwarder) handleEvent(event *OutboxEvent) error {
	ctx := context.Background()

	// Convert payload to primitive.D
	payloadDoc, ok := event.Payload.(primitive.D)
	if !ok {
		log.Printf("Invalid payload type: %T", event.Payload)
		return nil
	}

	// Convert primitive.D to map
	payloadMap := make(map[string]interface{})
	for _, elem := range payloadDoc {
		payloadMap[elem.Key] = elem.Value
	}

	// Convert MongoDB DateTime to time.Time
	createdAt, ok := payloadMap["created_at"].(primitive.DateTime)
	if !ok {
		log.Printf("Invalid created_at type: %T", payloadMap["created_at"])
		return nil
	}
	updatedAt, ok := payloadMap["updated_at"].(primitive.DateTime)
	if !ok {
		log.Printf("Invalid updated_at type: %T", payloadMap["updated_at"])
		return nil
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
		_, err := f.projectionCollection.InsertOne(ctx, projection)
		return err

	case "CustomerUpdated":
		err := f.projectionCollection.UpdateOne(ctx, 
			bson.M{"_id": projection.ID}, 
			bson.M{"$set": projection})
		return err

	default:
		return nil
	}
}

func (f *EventForwarder) markEventAsProcessed(event *OutboxEvent) error {
	return f.outboxCollection.UpdateOne(context.Background(),
		bson.M{"_id": event.ID},
		bson.M{"$set": bson.M{
			"status":     "processed",
			"updated_at": time.Now(),
		}})
}
