package inventory

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var integration = flag.Bool("integration", false, "run integration tests")

type IntegrationTestSuite struct {
	suite.Suite
	module     *Module
	natsConn   *nats.Conn
	mongoConn  *mongo.Client
	repository *Repository
}

func (s *IntegrationTestSuite) SetupSuite() {
	if !*integration {
		s.T().Skip("Skipping integration tests")
	}

	// Connect to NATS
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		s.T().Fatalf("Failed to connect to NATS: %v", err)
	}
	s.natsConn = nc

	// Connect to MongoDB
	ctx := context.Background()
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		s.T().Fatalf("Failed to connect to MongoDB: %v", err)
	}
	s.mongoConn = mongoClient

	// Initialize module
	s.module = NewModule(nc)
	s.repository = NewRepository(mongoClient.Database("inventory_test"))
	s.module.repo = s.repository
}

func (s *IntegrationTestSuite) TearDownSuite() {
	if s.natsConn != nil {
		s.natsConn.Close()
	}
	if s.mongoConn != nil {
		s.mongoConn.Disconnect(context.Background())
	}
}

func (s *IntegrationTestSuite) SetupTest() {
	if !*integration {
		return
	}
	// Clean up database before each test
	ctx := context.Background()
	s.mongoConn.Database("inventory_test").Drop(ctx)
}

func (s *IntegrationTestSuite) TestCreateInventoryIntegration() {
	inv := &Inventory{
		ProductID: "PROD123",
		Quantity:  100,
		Status:    "available",
	}

	// Create request
	body, _ := json.Marshal(inv)
	req := httptest.NewRequest(http.MethodPost, "/inventory", strings.NewReader(string(body)))
	w := httptest.NewRecorder()

	// Execute request
	s.module.ServeHTTP(w, req)

	// Assert response
	s.Equal(http.StatusCreated, w.Code)

	// Verify data in MongoDB
	var created Inventory
	err := json.NewDecoder(w.Body).Decode(&created)
	s.NoError(err)

	stored, err := s.repository.GetInventory(context.Background(), created.ID)
	s.NoError(err)
	s.Equal(inv.ProductID, stored.ProductID)
	s.Equal(inv.Quantity, stored.Quantity)

	// Wait for projection to be updated
	time.Sleep(100 * time.Millisecond)

	// Verify projection
	var proj InventoryProjection
	err = s.mongoConn.Database("inventory_test").Collection("inventory_projections").
		FindOne(context.Background(), bson.M{"product_id": inv.ProductID}).
		Decode(&proj)
	s.NoError(err)
	s.Equal(inv.ProductID, proj.ProductID)
	s.Equal(inv.Quantity, proj.Quantity)
}

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
