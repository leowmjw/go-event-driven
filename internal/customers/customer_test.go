package customers

import (
	"context"
	"testing"
	"time"

	"github.com/qiniu/qmgo"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CustomerTestSuite struct {
	suite.Suite
	customersDB *qmgo.Database
	orderingDB  *qmgo.Database
	repository  *MongoRepository
	forwarder   *EventForwarder
}

func (s *CustomerTestSuite) SetupSuite() {
	ctx := context.Background()

	// Connect to CustomersDB
	clientCustomers, err := qmgo.NewClient(ctx, &qmgo.Config{Uri: "mongodb://localhost:27017"})
	s.Require().NoError(err)
	s.customersDB = clientCustomers.Database("CustomersDB")

	// Connect to OrderingDB
	clientOrdering, err := qmgo.NewClient(ctx, &qmgo.Config{Uri: "mongodb://localhost:27017"})
	s.Require().NoError(err)
	s.orderingDB = clientOrdering.Database("OrderingDB")

	// Initialize repository and forwarder
	s.repository = NewMongoRepository(s.customersDB)
	s.forwarder = NewEventForwarder(s.customersDB, s.orderingDB)
	s.forwarder.Start()
}

func (s *CustomerTestSuite) TearDownSuite() {
	s.forwarder.Stop()
	s.customersDB.DropDatabase(context.Background())
	s.orderingDB.DropDatabase(context.Background())
}

func (s *CustomerTestSuite) SetupTest() {
	s.customersDB.Collection("customers").DropCollection(context.Background())
	s.customersDB.Collection("outbox").DropCollection(context.Background())
	s.orderingDB.Collection("projection_customers").DropCollection(context.Background())
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
	time.Sleep(2 * time.Second)

	// Verify customer was created
	saved, err := s.repository.FindByID(customer.ID)
	s.Require().NoError(err)
	s.Equal(customer.Name, saved.Name)
	s.Equal(customer.Email, saved.Email)

	// Verify outbox event was created
	var event OutboxEvent
	err = s.customersDB.Collection("outbox").Find(context.Background(),
		bson.M{"event_type": "CustomerCreated"}).One(&event)
	s.Require().NoError(err)
	s.Equal("processed", event.Status)

	// Verify projection was created
	var projection CustomerProjection
	err = s.orderingDB.Collection("projection_customers").Find(context.Background(),
		bson.M{"_id": customer.ID}).One(&projection)
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

	// Wait for initial event processing
	time.Sleep(2 * time.Second)

	// Update the customer
	customer.Name = "Jane Smith"
	err = s.repository.Update(customer)
	s.Require().NoError(err)

	// Wait for update event processing
	time.Sleep(2 * time.Second)

	// Verify customer was updated
	updated, err := s.repository.FindByID(customer.ID)
	s.Require().NoError(err)
	s.Equal("Jane Smith", updated.Name)

	// Verify update event was created
	var event OutboxEvent
	err = s.customersDB.Collection("outbox").Find(context.Background(),
		bson.M{"event_type": "CustomerUpdated"}).One(&event)
	s.Require().NoError(err)
	s.Equal("processed", event.Status)

	// Verify projection was updated
	var projection CustomerProjection
	err = s.orderingDB.Collection("projection_customers").Find(context.Background(),
		bson.M{"_id": customer.ID}).One(&projection)
	s.Require().NoError(err)
	s.Equal("Jane Smith", projection.Name)
}

func (s *CustomerTestSuite) Test_FindByEmail() {
	// Create a customer
	customer := &Customer{
		ID:    primitive.NewObjectID(),
		Name:  "Alice Brown",
		Email: "alice@example.com",
	}
	err := s.repository.Create(customer)
	s.Require().NoError(err)

	// Find by email
	found, err := s.repository.FindByEmail("alice@example.com")
	s.Require().NoError(err)
	s.Equal(customer.ID, found.ID)
	s.Equal(customer.Name, found.Name)
}
