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

	_, err := s.env.ExecuteActivity(CreateOrder, input)

	// Verify activity returns "not implemented" error
	s.Error(err)
	s.True(strings.Contains(err.Error(), "CreateOrder not implemented"))
}

func (s *OrderActivitiesTestSuite) Test_ProcessPayment() {
	orderID := "test-order-1"

	// Do a pre-check of the stock before submitting

	// Finalize the amount including any special discounts + rules ..
	_, err := s.env.ExecuteActivity(ProcessPayment, orderID)

	// Verify activity returns "not implemented" error
	s.Error(err)
	s.True(strings.Contains(err.Error(), "ProcessPayment not implemented"))
}

func (s *OrderActivitiesTestSuite) Test_ProcessFulfillment() {
	orderID := "test-order-1"

	_, err := s.env.ExecuteActivity(ProcessFulfillment, orderID)

	// Verify activity returns "not implemented" error
	s.Error(err)
	s.True(strings.Contains(err.Error(), "ProcessFulfillment not implemented"))
}

func (s *OrderActivitiesTestSuite) Test_ProcessDelivery() {
	orderID := "test-order-1"

	_, err := s.env.ExecuteActivity(ProcessDelivery, orderID)

	// Verify activity returns "not implemented" error
	s.Error(err)
	s.True(strings.Contains(err.Error(), "ProcessDelivery not implemented"))
}
