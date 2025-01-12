package inventory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MockRepository is a mock implementation of the repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) SaveInventory(ctx context.Context, inv *Inventory) error {
	args := m.Called(ctx, inv)
	return args.Error(0)
}

func (m *MockRepository) GetInventory(ctx context.Context, id primitive.ObjectID) (*Inventory, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Inventory), args.Error(1)
}

func (m *MockRepository) SaveOutboxEvent(ctx context.Context, event OutboxEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockRepository) GetPendingOutboxEvents(ctx context.Context) ([]OutboxEvent, error) {
	args := m.Called(ctx)
	return args.Get(0).([]OutboxEvent), args.Error(1)
}

func (m *MockRepository) UpdateOutboxEvent(ctx context.Context, event OutboxEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockRepository) UpsertProjection(ctx context.Context, proj *InventoryProjection) error {
	args := m.Called(ctx, proj)
	return args.Error(0)
}

// MockPublisher is a mock implementation of the Publisher interface
type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) Publish(subject string, data []byte) error {
	args := m.Called(subject, data)
	return args.Error(0)
}

func TestCreateInventory(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPub := &MockPublisher{}

	module := &Module{
		repo:      mockRepo,
		publisher: mockPub,
	}

	// Test case: successful creation
	t.Run("successful creation", func(t *testing.T) {
		inv := &Inventory{
			ProductID: "PROD123",
			Quantity:  100,
			Status:    "available",
		}

		// Setup expectations
		mockRepo.On("SaveInventory", mock.Anything, mock.MatchedBy(func(i *Inventory) bool {
			return i.ProductID == inv.ProductID
		})).Return(nil)

		mockRepo.On("SaveOutboxEvent", mock.Anything, mock.MatchedBy(func(e OutboxEvent) bool {
			return e.EventType == "inventory.created"
		})).Return(nil)

		// Create request
		body, _ := json.Marshal(inv)
		req := httptest.NewRequest(http.MethodPost, "/inventory", strings.NewReader(string(body)))
		w := httptest.NewRecorder()

		// Execute request
		module.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusCreated, w.Code)
		mockRepo.AssertExpectations(t)
	})

	// Test case: invalid request body
	t.Run("invalid request body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/inventory", strings.NewReader("invalid json"))
		w := httptest.NewRecorder()

		module.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestProcessOutboxEvents(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPub := &MockPublisher{}

	module := &Module{
		repo:      mockRepo,
		publisher: mockPub,
	}

	t.Run("successful processing", func(t *testing.T) {
		events := []OutboxEvent{
			{
				ID:        primitive.NewObjectID(),
				EventType: "inventory.created",
				Payload:   []byte(`{"product_id":"PROD123"}`),
				Status:    OutboxStatusPending,
			},
		}

		// Setup expectations
		mockRepo.On("GetPendingOutboxEvents", mock.Anything).Return(events, nil)
		mockPub.On("Publish", "inventory.projection.update", mock.Anything).Return(nil)
		mockRepo.On("UpdateOutboxEvent", mock.Anything, mock.MatchedBy(func(e OutboxEvent) bool {
			return e.Status == OutboxStatusProcessed
		})).Return(nil)

		// Execute
		module.ProcessOutboxEvents(nil) // nil msg since we're testing internal logic

		// Assert
		mockRepo.AssertExpectations(t)
		mockPub.AssertExpectations(t)
	})
}
