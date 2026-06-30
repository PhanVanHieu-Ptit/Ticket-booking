package reclamation

import (
	"context"
)

// Repository defines operations for reclaiming expired ticket holds.
type Repository interface {
	ReclaimExpiredHolds(ctx context.Context) (int64, error)
	ReclaimSessionHold(ctx context.Context, sessionID string) error
}

// Service defines background work operations.
type Service interface {
	StartReclamationWorker(ctx context.Context) error
	StartReconciliationCron(ctx context.Context) error
}
