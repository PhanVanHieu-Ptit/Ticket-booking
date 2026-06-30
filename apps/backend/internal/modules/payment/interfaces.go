package payment

import (
	"context"
)

// Repository defines data store operations for payments/orders.
type Repository interface {
	CreateOrder(ctx context.Context, order *Order) error
}

// Service defines business logic operations for payments/orders.
type Service interface {
	Checkout(ctx context.Context, sessionID string, ticketID int64, email string, cardHolderName string, simulateStatus string) (*Order, error)
}
