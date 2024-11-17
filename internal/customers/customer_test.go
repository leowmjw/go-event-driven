package customers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CustomerTestSuite struct {
	suite.Suite
	mongoURI string
	repository  *MongoRepository
	forwarder   *EventForwarder
	client      *mongo.Client
}

func (s *CustomerTestSuite) SetupSuite() {
	ctx := context.Background()
	s.mongoURI = "mongodb://localhost:27017/?replicaSet=rs0&directConnection=true"

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(s.mongoURI))
	s.Require().NoError(err)
	s.client = client

	// Initialize repository and forwarder
	s.repository = NewMongoRepository(s.mongoURI)
	s.forwarder = NewEventForwarder(s.mongoURI)
	s.forwarder.Start()
}

func (s *CustomerTestSuite) TearDownSuite() {
	s.forwarder.Stop()
	s.client.Disconnect(context.Background())
}

func (s *CustomerTestSuite) SetupTest() {
	customersDB := s.client.Database("CustomersDB")
	orderingDB := s.client.Database("OrderingDB")
	customersDB.Collection("customers").Drop(context.Background())
	customersDB.Collection("outbox").Drop(context.Background())
	orderingDB.Collection("projection_customers").Drop(context.Background())
}

func TestCustomerSuite(t *testing.T) {
	suite.Run(t, new(CustomerTestSuite))
}

func (s *CustomerTestSuite) Test_CreateCustomer() {
	// Create a new customer
	customer := &Customer{
		ID:    primitive.NewObjectID(),
		Name:  "John Doe",
		Email: "john@example.com",
	}

	// Save the customer
	err := s.repository.Create(customer)
	s.Require().NoError(err)
	s.NotEmpty(customer.ID)

	// Wait for event processing
	time.Sleep(500 * time.Millisecond)

	// Verify customer was created
	saved, err := s.repository.FindByID(customer.ID)
	s.Require().NoError(err)
	s.Equal(customer.Name, saved.Name)
	s.Equal(customer.Email, saved.Email)

	// Verify outbox event was created
	customersDB := s.client.Database("CustomersDB")
	var event OutboxEvent
	err = customersDB.Collection("outbox").FindOne(context.Background(),
		bson.M{"event_type": "CustomerCreated"}).Decode(&event)
	s.Require().NoError(err)
	s.Equal("processed", event.Status)

	// Verify projection was created
	orderingDB := s.client.Database("OrderingDB")
	var projection CustomerProjection
	err = orderingDB.Collection("projection_customers").FindOne(context.Background(),
		bson.M{"_id": customer.ID}).Decode(&projection)
	s.Require().NoError(err)
	s.Equal(customer.Name, projection.Name)
	s.Equal(customer.Email, projection.Email)
}

func (s *CustomerTestSuite) Test_UpdateCustomer() {
	// Create a customer first
	customer := &Customer{
		ID:    primitive.NewObjectID(),
		Name:  "Jane Doe",
		Email: "jane@example.com",
	}
	err := s.repository.Create(customer)
	s.Require().NoError(err)

	// Wait for event processing
	time.Sleep(500 * time.Millisecond)

	// Update the customer
	customer.Name = "Jane Smith"
	err = s.repository.Update(customer)
	s.Require().NoError(err)

	// Wait for event processing
	time.Sleep(500 * time.Millisecond)

	// Verify customer was updated
	saved, err := s.repository.FindByID(customer.ID)
	s.Require().NoError(err)
	s.Equal("Jane Smith", saved.Name)
	s.Equal(customer.Email, saved.Email)

	// Verify outbox event was created
	customersDB := s.client.Database("CustomersDB")
	var event OutboxEvent
	err = customersDB.Collection("outbox").FindOne(context.Background(),
		bson.M{"event_type": "CustomerUpdated"}).Decode(&event)
	s.Require().NoError(err)
	s.Equal("processed", event.Status)

	// Verify projection was updated
	orderingDB := s.client.Database("OrderingDB")
	var projection CustomerProjection
	err = orderingDB.Collection("projection_customers").FindOne(context.Background(),
		bson.M{"_id": customer.ID}).Decode(&projection)
	s.Require().NoError(err)
	s.Equal("Jane Smith", projection.Name)
	s.Equal(customer.Email, projection.Email)
}

func (s *CustomerTestSuite) Test_FindByEmail() {
	// Create a customer first
	customer := &Customer{
		ID:    primitive.NewObjectID(),
		Name:  "Bob Johnson",
		Email: "bob@example.com",
	}
	err := s.repository.Create(customer)
	s.Require().NoError(err)

	// Find the customer by email
	found, err := s.repository.FindByEmail("bob@example.com")
	s.Require().NoError(err)
	s.NotNil(found)
	s.Equal(customer.ID, found.ID)
	s.Equal(customer.Name, found.Name)
	s.Equal(customer.Email, found.Email)
}
