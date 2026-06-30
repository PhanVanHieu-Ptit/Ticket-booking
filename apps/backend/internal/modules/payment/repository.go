package payment

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type paymentRepository struct {
	db *pgxpool.Pool
}

// NewPaymentRepository creates a new instance of payment Repository.
func NewPaymentRepository(db *pgxpool.Pool) Repository {
	return &paymentRepository{
		db: db,
	}
}

func (r *paymentRepository) CreateOrder(ctx context.Context, order *Order) error {
	return nil
}
