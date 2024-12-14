package customers

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Customer represents a customer in the system
type Customer struct {
	ID        primitive.ObjectID `bson:"_id"`
	Name      string             `bson:"name"`
	Email     string             `bson:"email"`
	Deleted   bool               `bson:"deleted"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
}

// OutboxEvent represents an event in the outbox pattern
type OutboxEvent struct {
	ID         primitive.ObjectID `bson:"_id"`
	EventType  string             `bson:"event_type"`
	Payload    bson.Raw           `bson:"payload"`
	Status     string             `bson:"status"`
	RetryCount int32              `bson:"retry_count"`
	Error      string             `bson:"error,omitempty"`
	CreatedAt  time.Time          `bson:"created_at"`
	UpdatedAt  time.Time          `bson:"updated_at"`
}

// CustomerProjection represents a customer in the projection store
type CustomerProjection struct {
	ID        primitive.ObjectID `bson:"_id"`
	Name      string             `bson:"name"`
	Email     string             `bson:"email"`
	Deleted   bool               `bson:"deleted"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
}

// Repository interface for customer operations
type Repository interface {
	Create(customer *Customer) error
	Update(customer *Customer) error
	FindByID(id primitive.ObjectID) (*Customer, error)
	FindByEmail(email string) (*Customer, error)
}

// EventPublisher interface for publishing events
type EventPublisher interface {
	PublishCustomerCreated(customer *Customer) error
	PublishCustomerUpdated(customer *Customer) error
}

// TestCustomer extends Customer with test-specific fields
type TestCustomer struct {
	Customer `bson:",inline"`
	TestCase string `bson:"test_case"`
}

// CustomerAction represents the action to perform on a customer
type CustomerAction struct {
	Action string `json:"action"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}
