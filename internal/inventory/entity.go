package inventory

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Inventory struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	ProductID string            `json:"product_id" bson:"product_id"`
	Quantity  int               `json:"quantity" bson:"quantity"`
	Status    string            `json:"status" bson:"status"`
	UpdatedAt time.Time         `json:"updated_at" bson:"updated_at"`
}

type OutboxEvent struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	EventType string            `bson:"event_type"`
	Payload   []byte            `bson:"payload"`
	CreatedAt time.Time         `bson:"created_at"`
	Status    string            `bson:"status"`
}

type InventoryEvent struct {
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (e InventoryEvent) ToProjection() *InventoryProjection {
	return &InventoryProjection{
		ProductID: e.ProductID,
		Quantity:  e.Quantity,
		Status:    e.Status,
		UpdatedAt: e.UpdatedAt,
	}
}

type InventoryProjection struct {
	ProductID string    `bson:"product_id"`
	Quantity  int       `bson:"quantity"`
	Status    string    `bson:"status"`
	UpdatedAt time.Time `bson:"updated_at"`
}

const (
	OutboxStatusPending   = "pending"
	OutboxStatusProcessed = "processed"
)
