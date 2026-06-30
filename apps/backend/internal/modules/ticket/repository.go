package ticket

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type ticketRepository struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

// NewTicketRepository creates a new instance of ticket Repository.
func NewTicketRepository(db *pgxpool.Pool, redis *redis.Client) Repository {
	return &ticketRepository{
		db:    db,
		redis: redis,
	}
}

func (r *ticketRepository) GetAvailability(ctx context.Context) ([]Ticket, error) {
	return []Ticket{}, nil
}

func (r *ticketRepository) GetActiveHold(ctx context.Context, sessionID string) (*Ticket, error) {
	return &Ticket{}, nil
}

func (r *ticketRepository) ReserveTicket(ctx context.Context, sessionID string, category string) (*Ticket, error) {
	return &Ticket{}, nil
}

func (r *ticketRepository) CancelHold(ctx context.Context, sessionID string) error {
	return nil
}
