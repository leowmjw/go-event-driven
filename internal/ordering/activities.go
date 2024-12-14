package ordering

import (
	"context"
	"errors"
	"fmt"
)

// Activity function signatures
func CreateOrder(ctx context.Context, input OrderWorkflowInput) (string, error) {
	if input.OrderID == "" {
		return "", errors.New("invalid order: missing order ID")
	}
	if input.CustomerID == "" {
		return "", errors.New("invalid order: missing customer ID")
	}
	if len(input.Items) == 0 {
		return "", errors.New("invalid order: no items")
	}

	// In a real implementation, we would save the order to a database
	// For now, just return the order ID
	return input.OrderID, nil
}

func ProcessPayment(ctx context.Context, orderID string) (string, error) {
	if orderID == "order-2" {
		return "", errors.New("insufficient funds")
	}
	// In a real implementation, we would process the payment
	// For now, just return a dummy payment ID
	return fmt.Sprintf("payment-%s", orderID), nil
}

func ProcessFulfillment(ctx context.Context, orderID string) (string, error) {
	// In a real implementation, we would process the fulfillment
	// For now, just return a dummy fulfillment ID
	return fmt.Sprintf("fulfillment-%s", orderID), nil
}

func ProcessDelivery(ctx context.Context, orderID string) (string, error) {
	// In a real implementation, we would process the delivery
	// For now, just return a dummy delivery ID
	return fmt.Sprintf("delivery-%s", orderID), nil
}
