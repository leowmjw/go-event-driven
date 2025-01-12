package customers

import (
	"context"
	"encoding/json"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Customer represents a customer in the system
type Customer struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string            `bson:"name" json:"name"`
	Email     string            `bson:"email" json:"email"`
	Deleted   bool              `bson:"deleted" json:"deleted"`
	CreatedAt time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time         `bson:"updated_at" json:"updated_at"`
}

// OutboxEvent represents an event in the outbox pattern
type OutboxEvent struct {
	ID         primitive.ObjectID `bson:"_id" json:"id"`
	EventType  string            `bson:"event_type" json:"event_type"`
	Payload    json.RawMessage   `bson:"payload" json:"payload"`
	Status     string            `bson:"status" json:"status"`
	RetryCount int32             `bson:"retry_count" json:"retry_count"`
	Error      string            `bson:"error,omitempty" json:"error,omitempty"`
	CreatedAt  time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time         `bson:"updated_at" json:"updated_at"`
}

// CustomerProjection represents a customer in the projection store
type CustomerProjection struct {
	ID        primitive.ObjectID `bson:"_id" json:"id"`
	Name      string            `bson:"name" json:"name"`
	Email     string            `bson:"email" json:"email"`
	Deleted   bool              `bson:"deleted" json:"deleted"`
	CreatedAt time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time         `bson:"updated_at" json:"updated_at"`
}

// Repository defines the interface for customer data operations
type Repository interface {
	Create(ctx context.Context, customer *Customer) error
	Update(ctx context.Context, customer *Customer) error
	FindByEmail(ctx context.Context, email string, includeDeleted bool) (*Customer, error)
	FindByID(ctx context.Context, id primitive.ObjectID, includeDeleted bool) (*Customer, error)
	SoftDelete(ctx context.Context, id primitive.ObjectID) error
}

// EventForwarder defines the interface for forwarding customer events
type EventForwarder interface {
	Forward(event OutboxEvent) error
	Start()
	Stop()
}
