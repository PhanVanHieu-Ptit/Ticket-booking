package session

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type sessionRepository struct {
	db *pgxpool.Pool
}

// NewSessionRepository creates a new instance of session Repository.
func NewSessionRepository(db *pgxpool.Pool) Repository {
	return &sessionRepository{
		db: db,
	}
}

func (r *sessionRepository) GetAdminPasscodeHash(ctx context.Context) (string, error) {
	return "", nil
}
