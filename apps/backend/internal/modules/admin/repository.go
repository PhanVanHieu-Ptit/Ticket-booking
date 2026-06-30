package admin

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type adminRepository struct {
	db *pgxpool.Pool
}

// NewAdminRepository creates a new instance of admin Repository.
func NewAdminRepository(db *pgxpool.Pool) Repository {
	return &adminRepository{
		db: db,
	}
}

func (r *adminRepository) GetSalesMetrics(ctx context.Context) (*SalesMetrics, error) {
	return &SalesMetrics{}, nil
}

func (r *adminRepository) GetActiveHolds(ctx context.Context) ([]ActiveHoldDetail, error) {
	return []ActiveHoldDetail{}, nil
}
