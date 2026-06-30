package reclamation

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type reclamationRepository struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

// NewReclamationRepository creates a new instance of reclamation Repository.
func NewReclamationRepository(db *pgxpool.Pool, redis *redis.Client) Repository {
	return &reclamationRepository{
		db:    db,
		redis: redis,
	}
}

func (r *reclamationRepository) ReclaimExpiredHolds(ctx context.Context) (int64, error) {
	return 0, nil
}

func (r *reclamationRepository) ReclaimSessionHold(ctx context.Context, sessionID string) error {
	return nil
}
