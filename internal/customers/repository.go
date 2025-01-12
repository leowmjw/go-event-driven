package customers

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// MongoRepository implements the Repository interface
type MongoRepository struct {
	db *mongo.Database
}

// NewMongoRepository creates a new MongoDB repository
func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{
		db: db,
	}
}

// Create creates a new customer
func (r *MongoRepository) Create(ctx context.Context, customer *Customer) error {
	// Set timestamps if not set
	now := time.Now()
	if customer.CreatedAt.IsZero() {
		customer.CreatedAt = now
	}
	if customer.UpdatedAt.IsZero() {
		customer.UpdatedAt = now
	}

	// Insert customer
	result, err := r.db.Collection("customers").InsertOne(ctx, customer)
	if err != nil {
		return fmt.Errorf("failed to create customer: %v", err)
	}

	// Set the generated ID
	customer.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// Update updates an existing customer
func (r *MongoRepository) Update(ctx context.Context, customer *Customer) error {
	// Set update timestamp
	customer.UpdatedAt = time.Now()

	// Update customer
	filter := bson.M{"_id": customer.ID, "deleted": bson.M{"$ne": true}}
	update := bson.M{"$set": customer}

	result, err := r.db.Collection("customers").UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update customer: %v", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("customer not found")
	}

	return nil
}

// FindByEmail finds a customer by email
func (r *MongoRepository) FindByEmail(ctx context.Context, email string, includeDeleted bool) (*Customer, error) {
	filter := bson.M{"email": email}
	if !includeDeleted {
		filter["deleted"] = bson.M{"$ne": true}
	}

	var customer Customer
	err := r.db.Collection("customers").FindOne(ctx, filter).Decode(&customer)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find customer by email: %v", err)
	}

	return &customer, nil
}

// FindByID finds a customer by ID
func (r *MongoRepository) FindByID(ctx context.Context, id primitive.ObjectID, includeDeleted bool) (*Customer, error) {
	filter := bson.M{"_id": id}
	if !includeDeleted {
		filter["deleted"] = bson.M{"$ne": true}
	}

	var customer Customer
	err := r.db.Collection("customers").FindOne(ctx, filter).Decode(&customer)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find customer by ID: %v", err)
	}

	return &customer, nil
}

// SoftDelete soft deletes a customer
func (r *MongoRepository) SoftDelete(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id, "deleted": bson.M{"$ne": true}}
	update := bson.M{
		"$set": bson.M{
			"deleted":    true,
			"updatedAt": time.Now(),
		},
	}

	result, err := r.db.Collection("customers").UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to soft delete customer: %v", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("customer not found")
	}

	return nil
}
