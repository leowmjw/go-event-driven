package customers

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoRepository struct {
	customersCollection *mongo.Collection
	outboxCollection   *mongo.Collection
	mongoClient       *mongo.Client
}

func NewMongoRepository(mongoURI string) *MongoRepository {
	// Create MongoDB client
	clientOpts := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(context.Background(), clientOpts)
	if err != nil {
		panic(err)
	}

	return &MongoRepository{
		customersCollection: client.Database("CustomersDB").Collection("customers"),
		outboxCollection:   client.Database("CustomersDB").Collection("outbox"),
		mongoClient:       client,
	}
}

func (r *MongoRepository) Create(customer *Customer) error {
	ctx := context.Background()

	// Set timestamps
	now := time.Now()
	customer.CreatedAt = now
	customer.UpdatedAt = now

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

	// Start session for transaction
	session, err := r.mongoClient.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	// Start transaction
	err = session.StartTransaction()
	if err != nil {
		return err
	}

	// Insert customer
	_, err = session.Client().Database("CustomersDB").Collection("customers").InsertOne(ctx, customer)
	if err != nil {
		session.AbortTransaction(ctx)
		return err
	}

	// Insert outbox event
	_, err = session.Client().Database("CustomersDB").Collection("outbox").InsertOne(ctx, outboxEvent)
	if err != nil {
		session.AbortTransaction(ctx)
		return err
	}

	return session.CommitTransaction(ctx)
}

func (r *MongoRepository) Update(customer *Customer) error {
	ctx := context.Background()

	// Set update timestamp
	now := time.Now()
	customer.UpdatedAt = now

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

	// Start session for transaction
	session, err := r.mongoClient.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	// Start transaction
	err = session.StartTransaction()
	if err != nil {
		return err
	}

	// Update customer
	result, err := session.Client().Database("CustomersDB").Collection("customers").UpdateOne(ctx, 
		bson.M{"_id": customer.ID}, 
		bson.M{"$set": customer})
	if err != nil {
		session.AbortTransaction(ctx)
		return err
	}
	if result.MatchedCount == 0 {
		session.AbortTransaction(ctx)
		return mongo.ErrNoDocuments
	}

	// Insert outbox event
	_, err = session.Client().Database("CustomersDB").Collection("outbox").InsertOne(ctx, outboxEvent)
	if err != nil {
		session.AbortTransaction(ctx)
		return err
	}

	return session.CommitTransaction(ctx)
}

func (r *MongoRepository) FindByID(id primitive.ObjectID) (*Customer, error) {
	var customer Customer
	err := r.customersCollection.FindOne(context.Background(), 
		bson.M{"_id": id}).Decode(&customer)
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
	err := r.customersCollection.FindOne(context.Background(), 
		bson.M{"email": email}).Decode(&customer)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &customer, nil
}
