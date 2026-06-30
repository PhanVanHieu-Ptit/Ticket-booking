package ticket

import (
	"context"
)

// Repository defines data store operations for tickets.
type Repository interface {
	GetAvailability(ctx context.Context) ([]Ticket, error)
	GetActiveHold(ctx context.Context, sessionID string) (*Ticket, error)
	ReserveTicket(ctx context.Context, sessionID string, category string) (*Ticket, error)
	CancelHold(ctx context.Context, sessionID string) error
}

// Service defines business logic operations for tickets.
type Service interface {
	GetAvailability(ctx context.Context) ([]Ticket, error)
	GetActiveHold(ctx context.Context, sessionID string) (*Ticket, error)
	ReserveTicket(ctx context.Context, sessionID string, category string) (*Ticket, error)
	CancelHold(ctx context.Context, sessionID string) error
}
