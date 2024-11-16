package ordering

import (
	"context"
	"errors"
)

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
