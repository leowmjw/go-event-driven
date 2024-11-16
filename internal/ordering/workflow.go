package ordering

import (
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

	// Block until payment process is initiated; or auto-cancel after 1 week
	// Process Payment
	var paymentID string
	err = workflow.ExecuteActivity(ctx, ProcessPayment, orderID).Get(ctx, &paymentID)
	// When payment is processed
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
	// If fulfilment fails; need to issue correcting statement to customer
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
	// Delivery failure will lead to operations dealing/fraud/dispute; which will kick off other failure
	if err != nil {
		state.Status = "delivery_failed"
		state.ErrorMessage = err.Error()
		return state, nil
	}
	state.DeliveryID = deliveryID
	state.Status = "completed"

	return state, nil
}
