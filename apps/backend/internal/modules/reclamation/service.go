package reclamation

import (
	"context"
)

type reclamationService struct {
	repo Repository
}

// NewReclamationService creates a new instance of reclamation Service.
func NewReclamationService(repo Repository) Service {
	return &reclamationService{
		repo: repo,
	}
}

func (s *reclamationService) StartReclamationWorker(ctx context.Context) error {
	return nil
}

func (s *reclamationService) StartReconciliationCron(ctx context.Context) error {
	return nil
}
