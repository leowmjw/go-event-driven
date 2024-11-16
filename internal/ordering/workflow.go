package ordering

import (
	"context"
	"errors"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type OrderItem struct {
	ProductID  string
	Quantity   int
	UnitPrice  float64
	TotalPrice float64
}

type OrderWorkflowInput struct {
	OrderID     string
	CustomerID  string
	Items       []OrderItem
	TotalAmount float64
}

type OrderWorkflowState struct {
	Status        string
	OrderID       string
	PaymentID     string
	FulfillmentID string
	DeliveryID    string
	ErrorMessage  string
}

type OrderWorkflow struct{}

func (w OrderWorkflow) Execute(ctx workflow.Context, input OrderWorkflowInput) (OrderWorkflowState, error) {
	state := OrderWorkflowState{
		OrderID: input.OrderID,
		Status:  "pending",
	}

	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Create Order
	var orderID string
	err := workflow.ExecuteActivity(ctx, CreateOrder, input).Get(ctx, &orderID)
	if err != nil {
		state.Status = "creation_failed"
		state.ErrorMessage = err.Error()
		return state, nil
	}

	// Process Payment
	var paymentID string
	err = workflow.ExecuteActivity(ctx, ProcessPayment, orderID).Get(ctx, &paymentID)
	if err != nil {
		state.Status = "payment_failed"
		state.ErrorMessage = err.Error()
		return state, nil
	}
	state.PaymentID = paymentID
	state.Status = "payment_processed"

	// Process Fulfillment
	var fulfillmentID string
	err = workflow.ExecuteActivity(ctx, ProcessFulfillment, orderID).Get(ctx, &fulfillmentID)
	if err != nil {
		state.Status = "fulfillment_failed"
		state.ErrorMessage = err.Error()
		return state, nil
	}
	state.FulfillmentID = fulfillmentID
	state.Status = "fulfillment_processed"

	// Process Delivery
	var deliveryID string
	err = workflow.ExecuteActivity(ctx, ProcessDelivery, orderID).Get(ctx, &deliveryID)
	if err != nil {
		state.Status = "delivery_failed"
		state.ErrorMessage = err.Error()
		return state, nil
	}
	state.DeliveryID = deliveryID
	state.Status = "completed"

	return state, nil
}

// Activity function signatures
func CreateOrder(ctx context.Context, input OrderWorkflowInput) (string, error) {
	return "", errors.New("CreateOrder not implemented")
}

func ProcessPayment(ctx context.Context, orderID string) (string, error) {
	return "", errors.New("ProcessPayment not implemented")
}

func ProcessFulfillment(ctx context.Context, orderID string) (string, error) {
	return "", errors.New("ProcessFulfillment not implemented")
}

func ProcessDelivery(ctx context.Context, orderID string) (string, error) {
	return "", errors.New("ProcessDelivery not implemented")
}
