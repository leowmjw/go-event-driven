package customers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SetupHandler handles test data setup
type SetupHandler struct {
	mongoClient *mongo.Client
}

// NewSetupHandler creates a new setup handler
func NewSetupHandler(mongoURI string) (*SetupHandler, error) {
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	return &SetupHandler{
		mongoClient: client,
	}, nil
}

// Close closes the MongoDB connection
func (h *SetupHandler) Close() error {
	return h.mongoClient.Disconnect(context.Background())
}

// SetupTestData sets up test data in MongoDB
func (h *SetupHandler) SetupTestData(ctx context.Context) error {
	// Drop existing collections
	if err := h.mongoClient.Database("CustomersDB").Collection("customers").Drop(ctx); err != nil {
		return fmt.Errorf("failed to drop customers collection: %v", err)
	}
	if err := h.mongoClient.Database("CustomersDB").Collection("outbox").Drop(ctx); err != nil {
		return fmt.Errorf("failed to drop outbox collection: %v", err)
	}

	// Create test customers
	testCustomers := []Customer{
		{
			ID:        primitive.NewObjectID(),
			Name:      "John Doe",
			Email:     "john.doe@example.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        primitive.NewObjectID(),
			Name:      "Jane Smith",
			Email:     "jane.smith@example.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        primitive.NewObjectID(),
			Name:      "José García",
			Email:     "jose.garcia@example.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        primitive.NewObjectID(),
			Name:      "张伟",
			Email:     "zhang.wei@example.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        primitive.NewObjectID(),
			Name:      "Test User",
			Email:     "test+label@example.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        primitive.NewObjectID(),
			Name:      "Domain User",
			Email:     "user@sub.example.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        primitive.NewObjectID(),
			Name:      strings.Repeat("Long ", 20) + "Name",
			Email:     "long.name@example.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        primitive.NewObjectID(),
			Name:      "O'Connor-Smith Jr.",
			Email:     "oconnor@example.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        primitive.NewObjectID(),
			Name:      "Dot User",
			Email:     "first.middle.last@example.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        primitive.NewObjectID(),
			Name:      "A",
			Email:     "a@b.co",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// Insert test customers and create outbox events
	for _, customer := range testCustomers {
		// Insert customer
		_, err := h.mongoClient.Database("CustomersDB").Collection("customers").InsertOne(ctx, customer)
		if err != nil {
			return fmt.Errorf("failed to insert test customer: %v", err)
		}

		// Create outbox event
		payload, err := json.Marshal(customer)
		if err != nil {
			return fmt.Errorf("failed to marshal customer: %v", err)
		}

		event := OutboxEvent{
			ID:         primitive.NewObjectID(),
			EventType:  "CustomerCreated",
			Payload:    payload,
			Status:     "pending",
			RetryCount: 0,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		_, err = h.mongoClient.Database("CustomersDB").Collection("outbox").InsertOne(ctx, event)
		if err != nil {
			return fmt.Errorf("failed to insert outbox event: %v", err)
		}
	}

	log.Println("Test data setup completed successfully")
	return nil
}

// GetTestCustomers returns all test customers
func (h *SetupHandler) GetTestCustomers(ctx context.Context) ([]Customer, error) {
	var customers []Customer
	cursor, err := h.mongoClient.Database("CustomersDB").Collection("customers").Find(ctx, bson.M{
		"deleted": bson.M{"$ne": true}, // Filter out deleted customers
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find customers: %v", err)
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &customers); err != nil {
		return nil, fmt.Errorf("failed to decode customers: %v", err)
	}

	return customers, nil
}

// GetOutboxEntries returns all outbox entries
func (h *SetupHandler) GetOutboxEntries(ctx context.Context) ([]OutboxEvent, error) {
	var entries []OutboxEvent
	cursor, err := h.mongoClient.Database("CustomersDB").Collection("outbox").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to get outbox entries: %v", err)
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &entries); err != nil {
		return nil, fmt.Errorf("failed to decode outbox entries: %v", err)
	}

	return entries, nil
}

// GetCustomerProjections returns all customer projections from OrderingDB
func (h *SetupHandler) GetCustomerProjections(ctx context.Context) ([]CustomerProjection, error) {
	var projections []CustomerProjection
	cursor, err := h.mongoClient.Database("OrderingDB").Collection("projection_customers").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to get customer projections: %v", err)
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &projections); err != nil {
		return nil, fmt.Errorf("failed to decode customer projections: %v", err)
	}

	return projections, nil
}
