package payment

import (
	"context"
	"database/sql"
)

// Repository defines data store operations for payments/orders.
type Repository interface {
	GetDB() *sql.DB
	LockTicket(ctx context.Context, tx *sql.Tx, ticketID int64) (*Ticket, error)
	UpdateTicketStatus(ctx context.Context, tx *sql.Tx, ticketID int64, status string) error
	CreateOrder(ctx context.Context, tx *sql.Tx, order *Order) error
}

// Service defines business logic operations for payments/orders.
type Service interface {
	Checkout(ctx context.Context, sessionID string, ticketID int64, email string, cardHolderName string, simulateStatus string) (*Order, error)
}
