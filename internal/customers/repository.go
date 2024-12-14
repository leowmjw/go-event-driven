package customers

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type MongoRepository struct {
	customersCollection *mongo.Collection
	outboxCollection   *mongo.Collection
	client            *mongo.Client
}

func NewMongoRepository(mongoURI string) *MongoRepository {
	ctx := context.Background()
	clientOpts := options.Client().ApplyURI(mongoURI).
		SetServerSelectionTimeout(2 * time.Second)
	
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		panic(err)
	}

	// Verify that we can connect to MongoDB and that it's a replica set
	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		panic(fmt.Errorf("failed to ping MongoDB: %v", err))
	}

	customersDB := client.Database("CustomersDB")
	return &MongoRepository{
		customersCollection: customersDB.Collection("customers"),
		outboxCollection:   customersDB.Collection("outbox"),
		client:            client,
	}
}

func (r *MongoRepository) Create(customer *Customer) error {
	ctx := context.Background()

	// Start a session for transaction
	session, err := r.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %v", err)
	}
	defer session.EndSession(ctx)

	// Start transaction
	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Insert customer
		if customer.ID.IsZero() {
			customer.ID = primitive.NewObjectID()
		}
		now := time.Now()
		customer.CreatedAt = now
		customer.UpdatedAt = now

		_, err := r.customersCollection.InsertOne(sessCtx, customer)
		if err != nil {
			return nil, fmt.Errorf("failed to insert customer: %v", err)
		}

		// Create outbox event
		customerBSON, err := bson.Marshal(customer)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal customer: %v", err)
		}

		event := OutboxEvent{
			ID:         primitive.NewObjectID(),
			EventType:  "CustomerCreated",
			Payload:    bson.Raw(customerBSON),
			Status:    "pending",
			CreatedAt: now,
			UpdatedAt: now,
		}

		_, err = r.outboxCollection.InsertOne(sessCtx, event)
		if err != nil {
			return nil, fmt.Errorf("failed to insert outbox event: %v", err)
		}

		return nil, nil
	})

	return err
}

func (r *MongoRepository) Update(customer *Customer) error {
	ctx := context.Background()

	// Start a session for transaction
	session, err := r.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %v", err)
	}
	defer session.EndSession(ctx)

	// Start transaction
	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		now := time.Now()
		customer.UpdatedAt = now

		// Update customer
		filter := bson.M{"_id": customer.ID}
		update := bson.M{"$set": customer}
		result := r.customersCollection.FindOneAndUpdate(sessCtx, filter, update, options.FindOneAndUpdate().SetReturnDocument(options.After))
		
		var updatedCustomer Customer
		if err := result.Decode(&updatedCustomer); err != nil {
			return nil, fmt.Errorf("failed to update customer: %v", err)
		}

		// Create outbox event
		customerBSON, err := bson.Marshal(customer)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal customer: %v", err)
		}

		event := OutboxEvent{
			ID:         primitive.NewObjectID(),
			EventType:  "CustomerUpdated",
			Payload:    bson.Raw(customerBSON),
			Status:    "pending",
			CreatedAt: now,
			UpdatedAt: now,
		}

		_, err = r.outboxCollection.InsertOne(sessCtx, event)
		if err != nil {
			return nil, fmt.Errorf("failed to insert outbox event: %v", err)
		}

		return nil, nil
	})

	return err
}

func (r *MongoRepository) FindByID(id primitive.ObjectID, includeDeleted bool) (*Customer, error) {
	filter := bson.M{"_id": id}
	if !includeDeleted {
		filter["deleted"] = false
	}

	var customer Customer
	err := r.customersCollection.FindOne(context.Background(), filter).Decode(&customer)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &customer, nil
}

func (r *MongoRepository) FindByEmail(email string, includeDeleted bool) (*Customer, error) {
	filter := bson.M{"email": email}
	if !includeDeleted {
		filter["deleted"] = false
	}

	var customer Customer
	err := r.customersCollection.FindOne(context.Background(), filter).Decode(&customer)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &customer, nil
}

func (r *MongoRepository) SoftDelete(id primitive.ObjectID) error {
	session, err := r.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %v", err)
	}
	defer session.EndSession(context.Background())

	_, err = session.WithTransaction(context.Background(), func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Update customer
		update := bson.M{
			"$set": bson.M{
				"deleted":    true,
				"updated_at": time.Now(),
			},
		}
		result, err := r.customersCollection.UpdateOne(
			sessCtx,
			bson.M{"_id": id},
			update,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to update customer: %v", err)
		}
		if result.MatchedCount == 0 {
			return nil, fmt.Errorf("customer not found")
		}

		// Get the updated customer
		var customer Customer
		err = r.customersCollection.FindOne(sessCtx, bson.M{"_id": id}).Decode(&customer)
		if err != nil {
			return nil, fmt.Errorf("failed to find customer: %v", err)
		}

		// Create outbox event
		customerBytes, err := bson.Marshal(customer)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal customer: %v", err)
		}

		event := OutboxEvent{
			ID:        primitive.NewObjectID(),
			EventType: "CustomerDeleted",
			Payload:   bson.Raw(customerBytes),
			Status:    "pending",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		_, err = r.outboxCollection.InsertOne(sessCtx, event)
		if err != nil {
			return nil, fmt.Errorf("failed to create outbox event: %v", err)
		}

		return nil, nil
	})

	return err
}

func (r *MongoRepository) GetRandomCustomer(ctx context.Context) (*Customer, error) {
	// Use MongoDB's $sample to get a random document
	pipeline := []bson.D{
		{{Key: "$sample", Value: bson.D{{Key: "size", Value: 1}}}},
	}

	cursor, err := r.customersCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get random customer: %v", err)
	}
	defer cursor.Close(ctx)

	var customers []Customer
	if err := cursor.All(ctx, &customers); err != nil {
		return nil, fmt.Errorf("failed to decode random customer: %v", err)
	}

	if len(customers) == 0 {
		return nil, fmt.Errorf("no customers found")
	}

	return &customers[0], nil
}
