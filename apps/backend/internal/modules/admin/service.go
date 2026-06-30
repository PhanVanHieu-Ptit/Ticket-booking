package admin

import (
	"context"
)

type adminService struct {
	repo Repository
}

// NewAdminService creates a new instance of admin Service.
func NewAdminService(repo Repository) Service {
	return &adminService{
		repo: repo,
	}
}

func (s *adminService) GetSalesMetrics(ctx context.Context) (*SalesMetrics, error) {
	return s.repo.GetSalesMetrics(ctx)
}

func (s *adminService) GetActiveHolds(ctx context.Context) ([]ActiveHoldDetail, error) {
	return s.repo.GetActiveHolds(ctx)
}
