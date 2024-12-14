package ordering

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type OrderActivitiesTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestActivityEnvironment
}

func (s *OrderActivitiesTestSuite) SetupTest() {
	s.env = s.NewTestActivityEnvironment()
	s.env.RegisterActivity(CreateOrder)
	s.env.RegisterActivity(ProcessPayment)
	s.env.RegisterActivity(ProcessFulfillment)
	s.env.RegisterActivity(ProcessDelivery)
}

func TestOrderActivitiesTestSuite(t *testing.T) {
	suite.Run(t, new(OrderActivitiesTestSuite))
}

func (s *OrderActivitiesTestSuite) Test_CreateOrder() {
	// Test valid order
	input := OrderWorkflowInput{
		OrderID:    "test-order-1",
		CustomerID: "test-customer-1",
		Items: []OrderItem{
			{
				ProductID:  "test-product-1",
				Quantity:   1,
				UnitPrice:  10.00,
				TotalPrice: 10.00,
			},
		},
		TotalAmount: 10.00,
	}

	var orderID string
	result, err := s.env.ExecuteActivity(CreateOrder, input)
	s.NoError(err)
	s.NoError(result.Get(&orderID))
	s.Equal("test-order-1", orderID)

	// Test invalid order (missing order ID)
	input.OrderID = ""
	_, err = s.env.ExecuteActivity(CreateOrder, input)
	s.Error(err)
	s.True(strings.Contains(err.Error(), "missing order ID"))

	// Test invalid order (missing customer ID)
	input.OrderID = "test-order-1"
	input.CustomerID = ""
	_, err = s.env.ExecuteActivity(CreateOrder, input)
	s.Error(err)
	s.True(strings.Contains(err.Error(), "missing customer ID"))

	// Test invalid order (no items)
	input.CustomerID = "test-customer-1"
	input.Items = nil
	_, err = s.env.ExecuteActivity(CreateOrder, input)
	s.Error(err)
	s.True(strings.Contains(err.Error(), "no items"))
}

func (s *OrderActivitiesTestSuite) Test_ProcessPayment() {
	// Test successful payment
	var paymentID string
	result, err := s.env.ExecuteActivity(ProcessPayment, "order-1")
	s.NoError(err)
	s.NoError(result.Get(&paymentID))
	s.Equal("payment-order-1", paymentID)

	// Test failed payment (insufficient funds)
	_, err = s.env.ExecuteActivity(ProcessPayment, "order-2")
	s.Error(err)
	s.True(strings.Contains(err.Error(), "insufficient funds"))
}

func (s *OrderActivitiesTestSuite) Test_ProcessFulfillment() {
	var fulfillmentID string
	result, err := s.env.ExecuteActivity(ProcessFulfillment, "order-1")
	s.NoError(err)
	s.NoError(result.Get(&fulfillmentID))
	s.Equal("fulfillment-order-1", fulfillmentID)
}

func (s *OrderActivitiesTestSuite) Test_ProcessDelivery() {
	var deliveryID string
	result, err := s.env.ExecuteActivity(ProcessDelivery, "order-1")
	s.NoError(err)
	s.NoError(result.Get(&deliveryID))
	s.Equal("delivery-order-1", deliveryID)
}
