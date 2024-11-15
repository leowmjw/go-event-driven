package ordering

import (
	"context"
)

// OrderService defines the interface for order processing activities
type OrderService interface {
	CreateOrder(ctx context.Context, input OrderWorkflowInput) (string, error)
	ProcessPayment(ctx context.Context, orderID string) (string, error)
	ProcessFulfillment(ctx context.Context, orderID string) (string, error)
	ProcessDelivery(ctx context.Context, orderID string) (string, error)
}
