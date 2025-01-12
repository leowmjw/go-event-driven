package inventory

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Repository struct {
	db *mongo.Database
}

func NewRepository(db *mongo.Database) *Repository {
	return &Repository{db: db}
}

// SaveInventory creates or updates an inventory item
func (r *Repository) SaveInventory(ctx context.Context, inv *Inventory) error {
	if inv.ID.IsZero() {
		inv.ID = primitive.NewObjectID()
	}
	
	_, err := r.db.Collection("inventory").UpdateOne(
		ctx,
		bson.M{"_id": inv.ID},
		bson.M{"$set": inv},
		options.Update().SetUpsert(true),
	)
	return err
}

// GetInventory retrieves an inventory item by ID
func (r *Repository) GetInventory(ctx context.Context, id primitive.ObjectID) (*Inventory, error) {
	var inv Inventory
	err := r.db.Collection("inventory").FindOne(ctx, bson.M{"_id": id}).Decode(&inv)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// SaveOutboxEvent saves an event to the outbox
func (r *Repository) SaveOutboxEvent(ctx context.Context, event OutboxEvent) error {
	if event.ID.IsZero() {
		event.ID = primitive.NewObjectID()
	}
	_, err := r.db.Collection("inventory_outbox").InsertOne(ctx, event)
	return err
}

// GetPendingOutboxEvents retrieves all pending events from the outbox
func (r *Repository) GetPendingOutboxEvents(ctx context.Context) ([]OutboxEvent, error) {
	cursor, err := r.db.Collection("inventory_outbox").Find(ctx, bson.M{
		"status": OutboxStatusPending,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var events []OutboxEvent
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}

// UpdateOutboxEvent updates the status of an outbox event
func (r *Repository) UpdateOutboxEvent(ctx context.Context, event OutboxEvent) error {
	_, err := r.db.Collection("inventory_outbox").UpdateOne(
		ctx,
		bson.M{"_id": event.ID},
		bson.M{"$set": bson.M{"status": event.Status}},
	)
	return err
}

// UpsertProjection updates or creates an inventory projection
func (r *Repository) UpsertProjection(ctx context.Context, proj *InventoryProjection) error {
	_, err := r.db.Collection("inventory_projections").UpdateOne(
		ctx,
		bson.M{"product_id": proj.ProductID},
		bson.M{"$set": proj},
		options.Update().SetUpsert(true),
	)
	return err
}
