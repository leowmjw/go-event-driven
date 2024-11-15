package ordering

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type OrderWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

func (s *OrderWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
	s.env.RegisterActivity(CreateOrder)
	s.env.RegisterActivity(ProcessPayment)
	s.env.RegisterActivity(ProcessFulfillment)
	s.env.RegisterActivity(ProcessDelivery)
}

func (s *OrderWorkflowTestSuite) TearDownTest() {
	s.env.AssertExpectations(s.T())
}

func TestOrderWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(OrderWorkflowTestSuite))
}

func (s *OrderWorkflowTestSuite) Test_SuccessfulOrderWorkflow() {
	// Test input
	input := OrderWorkflowInput{
		OrderID:     "order-1",
		CustomerID:  "customer-1",
		Items: []OrderItem{
			{
				ProductID:  "prod-1",
				Quantity:   2,
				UnitPrice: 10.00,
				TotalPrice: 20.00,
			},
		},
		TotalAmount: 20.00,
	}

	// Mock activities
	s.env.OnActivity(CreateOrder, mock.Anything, input).Return("order-1", nil)
	s.env.OnActivity(ProcessPayment, mock.Anything, "order-1").Return("payment-1", nil)
	s.env.OnActivity(ProcessFulfillment, mock.Anything, "order-1").Return("fulfillment-1", nil)
	s.env.OnActivity(ProcessDelivery, mock.Anything, "order-1").Return("delivery-1", nil)

	// Execute workflow
	s.env.ExecuteWorkflow(OrderWorkflow{}.Execute, input)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result OrderWorkflowState
	s.NoError(s.env.GetWorkflowResult(&result))

	// Assert final state
	s.Equal("completed", result.Status)
	s.Equal("payment-1", result.PaymentID)
	s.Equal("fulfillment-1", result.FulfillmentID)
	s.Equal("delivery-1", result.DeliveryID)
	s.Empty(result.ErrorMessage)
}

func (s *OrderWorkflowTestSuite) Test_FailedPaymentWorkflow() {
	input := OrderWorkflowInput{
		OrderID:     "order-2",
		CustomerID:  "customer-2",
		Items: []OrderItem{
			{
				ProductID:  "prod-1",
				Quantity:   1,
				UnitPrice: 10.00,
				TotalPrice: 10.00,
			},
		},
		TotalAmount: 10.00,
	}

	paymentError := "insufficient funds"

	// Mock activities
	s.env.OnActivity(CreateOrder, mock.Anything, input).Return("order-2", nil)
	s.env.OnActivity(ProcessPayment, mock.Anything, "order-2").Return("", errors.New(paymentError))

	// Execute workflow
	s.env.ExecuteWorkflow(OrderWorkflow{}.Execute, input)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result OrderWorkflowState
	s.NoError(s.env.GetWorkflowResult(&result))

	// Assert final state
	s.Equal("payment_failed", result.Status)
	s.Contains(result.ErrorMessage, paymentError)
}

func (s *OrderWorkflowTestSuite) Test_InvalidOrderWorkflow() {
	input := OrderWorkflowInput{
		CustomerID:  "customer-3",
		Items: []OrderItem{
			{
				ProductID:  "prod-1",
				Quantity:   1,
				UnitPrice: 10.00,
				TotalPrice: 10.00,
			},
		},
		TotalAmount: 10.00,
	}

	invalidOrderError := "invalid order: missing order ID"

	// Mock activities
	s.env.OnActivity(CreateOrder, mock.Anything, input).Return("", errors.New(invalidOrderError))

	// Execute workflow
	s.env.ExecuteWorkflow(OrderWorkflow{}.Execute, input)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result OrderWorkflowState
	s.NoError(s.env.GetWorkflowResult(&result))

	// Assert final state
	s.Equal("creation_failed", result.Status)
	s.Contains(result.ErrorMessage, invalidOrderError)
}
