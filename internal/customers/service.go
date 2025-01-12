package customers

import (
	"context"
	"encoding/json"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Service handles customer operations
type Service struct {
	repository Repository
	forwarder  EventForwarder
}

// NewService creates a new customer service
func NewService(repository Repository, forwarder EventForwarder) *Service {
	return &Service{
		repository: repository,
		forwarder:  forwarder,
	}
}

// CreateCustomer creates a new customer
func (s *Service) CreateCustomer(ctx context.Context, customer *Customer) error {
	// Set timestamps
	now := time.Now()
	customer.CreatedAt = now
	customer.UpdatedAt = now

	// Create customer
	if err := s.repository.Create(ctx, customer); err != nil {
		return err
	}

	// Create outbox event
	payload, err := json.Marshal(customer)
	if err != nil {
		return err
	}

	event := OutboxEvent{
		ID:         primitive.NewObjectID(),
		EventType:  "CustomerCreated",
		Payload:    payload,
		Status:     "pending",
		RetryCount: 0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	return s.forwarder.Forward(event)
}

// UpdateCustomer updates an existing customer
func (s *Service) UpdateCustomer(ctx context.Context, customer *Customer) error {
	// Set update timestamp
	customer.UpdatedAt = time.Now()

	// Update customer
	if err := s.repository.Update(ctx, customer); err != nil {
		return err
	}

	// Create outbox event
	payload, err := json.Marshal(customer)
	if err != nil {
		return err
	}

	event := OutboxEvent{
		ID:         primitive.NewObjectID(),
		EventType:  "CustomerUpdated",
		Payload:    payload,
		Status:     "pending",
		RetryCount: 0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	return s.forwarder.Forward(event)
}

// FindCustomerByEmail finds a customer by email
func (s *Service) FindCustomerByEmail(ctx context.Context, email string, includeDeleted bool) (*Customer, error) {
	return s.repository.FindByEmail(ctx, email, includeDeleted)
}

// FindCustomerByID finds a customer by ID
func (s *Service) FindCustomerByID(ctx context.Context, id primitive.ObjectID, includeDeleted bool) (*Customer, error) {
	return s.repository.FindByID(ctx, id, includeDeleted)
}

// SoftDeleteCustomer soft deletes a customer
func (s *Service) SoftDeleteCustomer(ctx context.Context, id primitive.ObjectID) error {
	// Soft delete customer
	if err := s.repository.SoftDelete(ctx, id); err != nil {
		return err
	}

	// Create outbox event
	event := OutboxEvent{
		ID:         primitive.NewObjectID(),
		EventType:  "CustomerDeleted",
		Payload:    []byte(`{"id":"` + id.Hex() + `"}`),
		Status:     "pending",
		RetryCount: 0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	return s.forwarder.Forward(event)
}
