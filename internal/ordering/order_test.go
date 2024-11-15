package ordering

import (
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
	input := OrderWorkflowInput{
		OrderID:     "test-order-1",
		CustomerID:  "test-customer-1",
		Items: []OrderItem{
			{
				ProductID:  "test-product-1",
				Quantity:   1,
				UnitPrice: 10.00,
				TotalPrice: 10.00,
			},
		},
		TotalAmount: 10.00,
	}

	result, err := s.env.ExecuteActivity(CreateOrder, input)

	s.NoError(err)
	var orderID string
	s.NoError(result.Get(&orderID))
	s.Equal("test-order-1", orderID)
}

func (s *OrderActivitiesTestSuite) Test_ProcessPayment() {
	orderID := "test-order-1"

	result, err := s.env.ExecuteActivity(ProcessPayment, orderID)

	s.NoError(err)
	var paymentID string
	s.NoError(result.Get(&paymentID))
	s.NotEmpty(paymentID)
}

func (s *OrderActivitiesTestSuite) Test_ProcessFulfillment() {
	orderID := "test-order-1"

	result, err := s.env.ExecuteActivity(ProcessFulfillment, orderID)

	s.NoError(err)
	var fulfillmentID string
	s.NoError(result.Get(&fulfillmentID))
	s.NotEmpty(fulfillmentID)
}

func (s *OrderActivitiesTestSuite) Test_ProcessDelivery() {
	orderID := "test-order-1"

	result, err := s.env.ExecuteActivity(ProcessDelivery, orderID)

	s.NoError(err)
	var deliveryID string
	s.NoError(result.Get(&deliveryID))
	s.NotEmpty(deliveryID)
}