package customers

import (
	"context"
	"time"

	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoRepository struct {
	customersCollection *qmgo.Collection
	outboxCollection   *qmgo.Collection
}

func NewMongoRepository(customersDB *qmgo.Database) *MongoRepository {
	return &MongoRepository{
		customersCollection: customersDB.Collection("customers"),
		outboxCollection:   customersDB.Collection("outbox"),
	}
}

func (r *MongoRepository) Create(customer *Customer) error {
	ctx := context.Background()

	// Set timestamps
	now := time.Now()
	customer.CreatedAt = now
	customer.UpdatedAt = now

	// Insert customer
	_, err := r.customersCollection.InsertOne(ctx, customer)
	if err != nil {
		return err
	}

	// Create outbox event with payload as map
	payload := bson.M{
		"_id":        customer.ID,
		"name":      customer.Name,
		"email":     customer.Email,
		"created_at": customer.CreatedAt,
		"updated_at": customer.UpdatedAt,
	}

	outboxEvent := &OutboxEvent{
		ID:        primitive.NewObjectID(),
		EventType: "CustomerCreated",
		Payload:   payload,
		Status:    "pending",
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err = r.outboxCollection.InsertOne(ctx, outboxEvent)
	return err
}

func (r *MongoRepository) Update(customer *Customer) error {
	ctx := context.Background()

	// Set update timestamp
	now := time.Now()
	customer.UpdatedAt = now

	// Update customer
	err := r.customersCollection.UpdateOne(ctx, bson.M{"_id": customer.ID}, 
		bson.M{"$set": customer})
	if err != nil {
		return err
	}

	// Create outbox event with payload as map
	payload := bson.M{
		"_id":        customer.ID,
		"name":      customer.Name,
		"email":     customer.Email,
		"created_at": customer.CreatedAt,
		"updated_at": customer.UpdatedAt,
	}

	outboxEvent := &OutboxEvent{
		ID:        primitive.NewObjectID(),
		EventType: "CustomerUpdated",
		Payload:   payload,
		Status:    "pending",
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err = r.outboxCollection.InsertOne(ctx, outboxEvent)
	return err
}

func (r *MongoRepository) FindByID(id primitive.ObjectID) (*Customer, error) {
	var customer Customer
	err := r.customersCollection.Find(context.Background(), 
		bson.M{"_id": id}).One(&customer)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &customer, nil
}

func (r *MongoRepository) FindByEmail(email string) (*Customer, error) {
	var customer Customer
	err := r.customersCollection.Find(context.Background(), 
		bson.M{"email": email}).One(&customer)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &customer, nil
}
