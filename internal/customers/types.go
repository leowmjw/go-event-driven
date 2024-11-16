package customers

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Customer represents a customer in the system
type Customer struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Name      string            `bson:"name"`
	Email     string            `bson:"email"`
	CreatedAt time.Time         `bson:"created_at"`
	UpdatedAt time.Time         `bson:"updated_at"`
}

// OutboxEvent represents an event in the outbox collection
type OutboxEvent struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	EventType string            `bson:"event_type"`
	Payload   interface{}       `bson:"payload"`
	Status    string            `bson:"status"` // pending, processed
	CreatedAt time.Time         `bson:"created_at"`
	UpdatedAt time.Time         `bson:"updated_at"`
}

// CustomerProjection represents a customer in the OrderingDB
type CustomerProjection struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Name      string            `bson:"name"`
	Email     string            `bson:"email"`
	CreatedAt time.Time         `bson:"created_at"`
	UpdatedAt time.Time         `bson:"updated_at"`
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
