package reclamation

import (
	"context"
	"database/sql"

	"github.com/redis/go-redis/v9"
)

// Module coordinates the reclamation background workers.
type Module struct {
	Repo    Repository
	Service Service
}

// NewModule initializes all layers of the reclamation module.
func NewModule(db *sql.DB, redis *redis.Client) *Module {
	repo := NewReclamationRepository(db, redis)
	svc := NewReclamationService(repo, redis)

	return &Module{
		Repo:    repo,
		Service: svc,
	}
}

// StartBackgroundJobs starts the Redis subscription and cron sweeper.
func (m *Module) StartBackgroundJobs(ctx context.Context) {
	go m.Service.StartReclamationWorker(ctx)
	go m.Service.StartReconciliationCron(ctx)
}
