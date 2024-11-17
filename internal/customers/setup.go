package customers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SetupHandler handles database setup for testing
type SetupHandler struct {
	mongoClient *mongo.Client
}

func NewSetupHandler(mongoURI string) (*SetupHandler, error) {
	ctx := context.Background()
	clientOpts := options.Client().ApplyURI(mongoURI).
		SetServerSelectionTimeout(2 * time.Second)
	
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	return &SetupHandler{
		mongoClient: client,
	}, nil
}

func (h *SetupHandler) ResetDatabases(ctx context.Context) error {
	// Drop existing databases
	if err := h.mongoClient.Database("CustomersDB").Drop(ctx); err != nil {
		return fmt.Errorf("failed to drop CustomersDB: %v", err)
	}
	if err := h.mongoClient.Database("OrderingDB").Drop(ctx); err != nil {
		return fmt.Errorf("failed to drop OrderingDB: %v", err)
	}

	// Create test data
	testCustomers := []TestCustomer{
		// Normal cases
		{
			Customer: Customer{
				ID:        primitive.NewObjectID(),
				Name:      "John Doe",
				Email:     "john.doe@example.com",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			TestCase: "Normal user with standard name and email",
		},
		{
			Customer: Customer{
				ID:        primitive.NewObjectID(),
				Name:      "Jane Smith",
				Email:     "jane.smith@example.com",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			TestCase: "Normal user with standard name and email",
		},
		// Unicode characters
		{
			Customer: Customer{
				ID:        primitive.NewObjectID(),
				Name:      "José García",
				Email:     "jose.garcia@example.com",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			TestCase: "Unicode characters in name",
		},
		{
			Customer: Customer{
				ID:        primitive.NewObjectID(),
				Name:      "张伟",
				Email:     "zhang.wei@example.com",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			TestCase: "Chinese characters in name",
		},
		// Special email formats
		{
			Customer: Customer{
				ID:        primitive.NewObjectID(),
				Name:      "Test User",
				Email:     "test+label@example.com",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			TestCase: "Email with plus addressing",
		},
		{
			Customer: Customer{
				ID:        primitive.NewObjectID(),
				Name:      "Domain User",
				Email:     "user@sub.example.com",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			TestCase: "Email with subdomain",
		},
		// Long names
		{
			Customer: Customer{
				ID:        primitive.NewObjectID(),
				Name:      strings.Repeat("Long ", 20) + "Name",
				Email:     "long.name@example.com",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			TestCase: "Very long name",
		},
		// Special characters in name
		{
			Customer: Customer{
				ID:        primitive.NewObjectID(),
				Name:      "O'Connor-Smith Jr.",
				Email:     "oconnor@example.com",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			TestCase: "Name with special characters",
		},
		// Multiple dots in email
		{
			Customer: Customer{
				ID:        primitive.NewObjectID(),
				Name:      "Dot User",
				Email:     "first.middle.last@example.com",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			TestCase: "Email with multiple dots",
		},
		// Minimal length
		{
			Customer: Customer{
				ID:        primitive.NewObjectID(),
				Name:      "A",
				Email:     "a@b.co",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			TestCase: "Minimal length name and email",
		},
	}

	// Insert test customers
	for _, testCustomer := range testCustomers {
		// Insert customer
		_, err := h.mongoClient.Database("CustomersDB").Collection("customers").InsertOne(ctx, testCustomer.Customer)
		if err != nil {
			return fmt.Errorf("failed to insert test customer: %v", err)
		}

		// Create outbox event
		payload, err := bson.Marshal(testCustomer.Customer)
		if err != nil {
			return fmt.Errorf("failed to marshal customer: %v", err)
		}

		event := OutboxEvent{
			ID:         primitive.NewObjectID(),
			EventType:  "CustomerCreated",
			Payload:    payload,
			Status:    "pending",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		_, err = h.mongoClient.Database("CustomersDB").Collection("outbox").InsertOne(ctx, event)
		if err != nil {
			return fmt.Errorf("failed to insert outbox event: %v", err)
		}

		// Create projection
		projection := CustomerProjection{
			ID:        testCustomer.Customer.ID,
			Name:      testCustomer.Customer.Name,
			Email:     testCustomer.Customer.Email,
			CreatedAt: testCustomer.Customer.CreatedAt,
			UpdatedAt: testCustomer.Customer.UpdatedAt,
		}

		_, err = h.mongoClient.Database("OrderingDB").Collection("projection_customers").InsertOne(ctx, projection)
		if err != nil {
			return fmt.Errorf("failed to insert customer projection: %v", err)
		}
	}

	return nil
}

func (h *SetupHandler) Close(ctx context.Context) error {
	return h.mongoClient.Disconnect(ctx)
}

// GetTestCustomers returns all test customers
func (h *SetupHandler) GetTestCustomers(ctx context.Context) ([]TestCustomer, error) {
	var customers []TestCustomer
	cursor, err := h.mongoClient.Database("CustomersDB").Collection("customers").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to get test customers: %v", err)
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &customers); err != nil {
		return nil, fmt.Errorf("failed to decode test customers: %v", err)
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
