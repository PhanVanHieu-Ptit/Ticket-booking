package admin

import (
	"context"
)

// Repository defines data store operations for administrative queries.
type Repository interface {
	GetSalesMetrics(ctx context.Context) (*SalesMetrics, error)
	GetActiveHolds(ctx context.Context) ([]ActiveHoldDetail, error)
}

// Service defines business logic operations for administrative queries.
type Service interface {
	GetSalesMetrics(ctx context.Context) (*SalesMetrics, error)
	GetActiveHolds(ctx context.Context) ([]ActiveHoldDetail, error)
}
