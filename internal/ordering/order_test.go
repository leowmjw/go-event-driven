package ordering

import (
	"errors"
	"testing"
	"time"
)

type Order struct {
	ID            string
	CustomerID    string
	Status        string
	TotalAmount   float64
	Items         []OrderItem
	DeliveryInfo  DeliveryInfo
	BillingInfo   BillingInfo
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type OrderItem struct {
	ProductID   string
	Quantity    int
	UnitPrice   float64
	TotalPrice  float64
}

type DeliveryInfo struct {
	Address     string
	City        string
	PostalCode  string
	Country     string
}

type BillingInfo struct {
	PaymentMethod string
	Status        string
}

type OrderService interface {
	CreateOrder(order Order) error
	ProcessPayment(orderID string) error
	ProcessFulfillment(orderID string) error
	ProcessDelivery(orderID string) error
	GetOrderStatus(orderID string) (string, error)
}

type mockOrderService struct {
	orders map[string]Order
}

func newMockOrderService() *mockOrderService {
	return &mockOrderService{
		orders: make(map[string]Order),
	}
}

func (m *mockOrderService) CreateOrder(order Order) error {
	if order.ID == "" {
		return errors.New("invalid order ID")
	}
	if order.CustomerID == "" {
		return errors.New("missing customer ID")
	}
	if len(order.Items) == 0 {
		return errors.New("order must contain at least one item")
	}
	m.orders[order.ID] = order
	return nil
}

func (m *mockOrderService) ProcessPayment(orderID string) error {
	order, ok := m.orders[orderID]
	if !ok {
		return errors.New("order not found")
	}
	if order.Status != "pending" {
		return errors.New("invalid state transition: cannot process payment for non-pending order")
	}
	order.Status = "payment_processed"
	m.orders[orderID] = order
	return nil
}

func (m *mockOrderService) ProcessFulfillment(orderID string) error {
	order, ok := m.orders[orderID]
	if !ok {
		return errors.New("order not found")
	}
	if order.Status != "payment_processed" {
		return errors.New("invalid state transition: cannot process fulfillment for non-payment-processed order")
	}
	order.Status = "fulfillment_processed"
	m.orders[orderID] = order
	return nil
}

func (m *mockOrderService) ProcessDelivery(orderID string) error {
	order, ok := m.orders[orderID]
	if !ok {
		return errors.New("order not found")
	}
	if order.Status != "fulfillment_processed" {
		return errors.New("invalid state transition: cannot process delivery for non-fulfillment-processed order")
	}
	order.Status = "delivery_processed"
	m.orders[orderID] = order
	return nil
}

func (m *mockOrderService) GetOrderStatus(orderID string) (string, error) {
	order, ok := m.orders[orderID]
	if !ok {
		return "", errors.New("order not found")
	}
	return order.Status, nil
}

func TestCreateOrder(t *testing.T) {
	tests := []struct {
		name    string
		order   Order
		wantErr bool
	}{
		{
			name: "valid order creation",
			order: Order{
				CustomerID: "customer-123",
				Items: []OrderItem{
					{
						ProductID:  "prod-1",
						Quantity:   2,
						UnitPrice: 10.00,
						TotalPrice: 20.00,
					},
				},
				TotalAmount: 20.00,
				DeliveryInfo: DeliveryInfo{
					Address:    "123 Main St",
					City:      "Sample City",
					PostalCode: "12345",
					Country:   "Sample Country",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid order - no items",
			order: Order{
				CustomerID: "customer-123",
				Items:     []OrderItem{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: Implement the actual test once the Order service is created
		})
	}
}

func TestOrderWorkflow(t *testing.T) {
	tests := []struct {
		name           string
		order          Order
		paymentStatus  string
		fulfillStatus  string
		deliveryStatus string
		expectedStatus string
		wantErr        bool
		expectedError  string
	}{
		{
			name: "successful order workflow",
			order: Order{
				ID:         "order-1",
				CustomerID: "customer-123",
				Status:    "pending",
				Items: []OrderItem{
					{
						ProductID:  "prod-1",
						Quantity:   2,
						UnitPrice: 10.00,
						TotalPrice: 20.00,
					},
				},
				TotalAmount: 20.00,
			},
			paymentStatus:  "success",
			fulfillStatus:  "completed",
			deliveryStatus: "delivered",
			expectedStatus: "delivery_processed",
			wantErr:       false,
		},
		{
			name: "error - invalid order ID",
			order: Order{
				ID:         "",  // Invalid: empty ID
				CustomerID: "customer-123",
				Status:    "pending",
				Items: []OrderItem{
					{
						ProductID:  "prod-1",
						Quantity:   1,
						UnitPrice: 10.00,
						TotalPrice: 10.00,
					},
				},
				TotalAmount: 10.00,
			},
			wantErr:       true,
			expectedError: "invalid order ID",
		},
		{
			name: "error - invalid state transition",
			order: Order{
				ID:         "order-2",
				CustomerID: "customer-123",
				Status:    "completed",  // Cannot process payment for completed order
				Items: []OrderItem{
					{
						ProductID:  "prod-1",
						Quantity:   1,
						UnitPrice: 10.00,
						TotalPrice: 10.00,
					},
				},
				TotalAmount: 10.00,
			},
			paymentStatus:  "success",
			wantErr:       true,
			expectedError: "invalid state transition: cannot process payment for non-pending order",
		},
		{
			name: "error - missing customer ID",
			order: Order{
				ID:         "order-3",
				CustomerID: "",  // Invalid: empty customer ID
				Status:    "pending",
				Items: []OrderItem{
					{
						ProductID:  "prod-1",
						Quantity:   1,
						UnitPrice: 10.00,
						TotalPrice: 10.00,
					},
				},
				TotalAmount: 10.00,
			},
			wantErr:       true,
			expectedError: "missing customer ID",
		},
		{
			name: "error - no items in order",
			order: Order{
				ID:         "order-4",
				CustomerID: "customer-123",
				Status:    "pending",
				Items:     []OrderItem{},  // Invalid: empty items
				TotalAmount: 0,
			},
			wantErr:       true,
			expectedError: "order must contain at least one item",
		},
		{
			name: "business case - failed payment",
			order: Order{
				ID:         "order-5",
				CustomerID: "customer-123",
				Status:    "pending",
				Items: []OrderItem{
					{
						ProductID:  "prod-1",
						Quantity:   1,
						UnitPrice: 10.00,
						TotalPrice: 10.00,
					},
				},
				TotalAmount: 10.00,
			},
			paymentStatus:  "failed",
			expectedStatus: "payment_failed",
			wantErr:       false,  // Not an error, just a business state
		},
		{
			name: "business case - failed fulfillment",
			order: Order{
				ID:         "order-6",
				CustomerID: "customer-123",
				Status:    "pending",
				Items: []OrderItem{
					{
						ProductID:  "prod-1",
						Quantity:   1,
						UnitPrice: 10.00,
						TotalPrice: 10.00,
					},
				},
				TotalAmount: 10.00,
			},
			paymentStatus:  "success",
			fulfillStatus:  "failed",
			expectedStatus: "fulfillment_failed",
			wantErr:       false,  // Not an error, just a business state
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newMockOrderService()
			
			// Create the order
			err := svc.CreateOrder(tt.order)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("CreateOrder() unexpected error = %v", err)
					return
				}
				if err.Error() != tt.expectedError {
					t.Errorf("CreateOrder() error = %v, want %v", err, tt.expectedError)
				}
				return
			}

			// Process payment
			if tt.paymentStatus != "" {
				err = svc.ProcessPayment(tt.order.ID)
				if err != nil {
					if !tt.wantErr {
						t.Errorf("ProcessPayment() unexpected error = %v", err)
						return
					}
					if err.Error() != tt.expectedError {
						t.Errorf("ProcessPayment() error = %v, want %v", err, tt.expectedError)
					}
					return
				}
			}

			// Process fulfillment if payment was successful
			if tt.paymentStatus == "success" && tt.fulfillStatus != "" {
				err = svc.ProcessFulfillment(tt.order.ID)
				if err != nil {
					if !tt.wantErr {
						t.Errorf("ProcessFulfillment() unexpected error = %v", err)
						return
					}
					if err.Error() != tt.expectedError {
						t.Errorf("ProcessFulfillment() error = %v, want %v", err, tt.expectedError)
					}
					return
				}
			}

			// Process delivery if fulfillment was completed
			if tt.fulfillStatus == "completed" && tt.deliveryStatus != "" {
				err = svc.ProcessDelivery(tt.order.ID)
				if err != nil {
					if !tt.wantErr {
						t.Errorf("ProcessDelivery() unexpected error = %v", err)
						return
					}
					if err.Error() != tt.expectedError {
						t.Errorf("ProcessDelivery() error = %v, want %v", err, tt.expectedError)
					}
					return
				}
			}

			// Only check status if we don't expect an error
			if !tt.wantErr {
				status, err := svc.GetOrderStatus(tt.order.ID)
				if err != nil {
					t.Errorf("GetOrderStatus() unexpected error = %v", err)
					return
				}
				if status != tt.expectedStatus {
					t.Errorf("Final order status = %v, want %v", status, tt.expectedStatus)
				}
			}
		})
	}
}
