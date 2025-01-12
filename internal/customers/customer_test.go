package customers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, customer *Customer) error {
	args := m.Called(ctx, customer)
	return args.Error(0)
}

func (m *MockRepository) Update(ctx context.Context, customer *Customer) error {
	args := m.Called(ctx, customer)
	return args.Error(0)
}

func (m *MockRepository) FindByEmail(ctx context.Context, email string, includeDeleted bool) (*Customer, error) {
	args := m.Called(ctx, email, includeDeleted)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Customer), args.Error(1)
}

func (m *MockRepository) FindByID(ctx context.Context, id primitive.ObjectID, includeDeleted bool) (*Customer, error) {
	args := m.Called(ctx, id, includeDeleted)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Customer), args.Error(1)
}

func (m *MockRepository) SoftDelete(ctx context.Context, id primitive.ObjectID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockEventForwarder struct {
	mock.Mock
}

func (m *MockEventForwarder) Forward(event OutboxEvent) error {
	args := m.Called(event)
	return args.Error(0)
}

func (m *MockEventForwarder) Start() {
	m.Called()
}

func (m *MockEventForwarder) Stop() {
	m.Called()
}

type CustomerTestSuite struct {
	suite.Suite
	mockRepo       *MockRepository
	mockForwarder  *MockEventForwarder
	service        *Service
}

func (s *CustomerTestSuite) SetupTest() {
	s.mockRepo = new(MockRepository)
	s.mockForwarder = new(MockEventForwarder)
	s.service = NewService(s.mockRepo, s.mockForwarder)
}

func TestCustomerTestSuite(t *testing.T) {
	suite.Run(t, new(CustomerTestSuite))
}

func (s *CustomerTestSuite) TestCreateCustomer() {
	// Test data
	customer := &Customer{
		Email: "test@example.com",
		Name:  "Test User",
	}

	// Setup expectations
	s.mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*customers.Customer")).Return(nil)
	s.mockForwarder.On("Forward", mock.AnythingOfType("OutboxEvent")).Return(nil)

	// Execute test
	err := s.service.CreateCustomer(context.Background(), customer)

	// Assertions
	s.NoError(err)
	s.mockRepo.AssertExpectations(s.T())
	s.mockForwarder.AssertExpectations(s.T())
}

func (s *CustomerTestSuite) TestUpdateCustomer() {
	// Test data
	customer := &Customer{
		ID:    primitive.NewObjectID(),
		Email: "test@example.com",
		Name:  "Updated User",
	}

	// Setup expectations
	s.mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*customers.Customer")).Return(nil)
	s.mockForwarder.On("Forward", mock.AnythingOfType("OutboxEvent")).Return(nil)

	// Execute test
	err := s.service.UpdateCustomer(context.Background(), customer)

	// Assertions
	s.NoError(err)
	s.mockRepo.AssertExpectations(s.T())
	s.mockForwarder.AssertExpectations(s.T())
}

func (s *CustomerTestSuite) TestFindCustomerByEmail() {
	// Test data
	email := "test@example.com"
	expected := &Customer{
		Email: email,
		Name:  "Test User",
	}

	// Setup expectations
	s.mockRepo.On("FindByEmail", mock.Anything, email, false).Return(expected, nil)

	// Execute test
	customer, err := s.service.FindCustomerByEmail(context.Background(), email, false)

	// Assertions
	s.NoError(err)
	s.Equal(expected, customer)
	s.mockRepo.AssertExpectations(s.T())
}

func (s *CustomerTestSuite) TestFindCustomerByID() {
	// Test data
	id := primitive.NewObjectID()
	expected := &Customer{
		ID:    id,
		Email: "test@example.com",
		Name:  "Test User",
	}

	// Setup expectations
	s.mockRepo.On("FindByID", mock.Anything, id, false).Return(expected, nil)

	// Execute test
	customer, err := s.service.FindCustomerByID(context.Background(), id, false)

	// Assertions
	s.NoError(err)
	s.Equal(expected, customer)
	s.mockRepo.AssertExpectations(s.T())
}

func (s *CustomerTestSuite) TestSoftDeleteCustomer() {
	// Test data
	id := primitive.NewObjectID()

	// Setup expectations
	s.mockRepo.On("SoftDelete", mock.Anything, id).Return(nil)
	s.mockForwarder.On("Forward", mock.AnythingOfType("OutboxEvent")).Return(nil)

	// Execute test
	err := s.service.SoftDeleteCustomer(context.Background(), id)

	// Assertions
	s.NoError(err)
	s.mockRepo.AssertExpectations(s.T())
	s.mockForwarder.AssertExpectations(s.T())
}
