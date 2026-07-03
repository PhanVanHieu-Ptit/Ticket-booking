package reclamation

import (
	"context"
	"database/sql"
	"time"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/constants"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/holdtimer"
	"github.com/redis/go-redis/v9"
)

type reclamationRepository struct {
	db    *sql.DB
	redis *redis.Client
}

// NewReclamationRepository creates a new instance of reclamation Repository.
func NewReclamationRepository(db *sql.DB, redis *redis.Client) Repository {
	return &reclamationRepository{
		db:    db,
		redis: redis,
	}
}

func (r *reclamationRepository) ReclaimExpiredHolds(ctx context.Context) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// expires_at <= NOW() is only a candidate filter here (bounds the query
	// using Postgres's own clock); each candidate still gets a second,
	// authoritative check against holdtimer's monotonic anchor below before
	// actually being reclaimed, since NOW() can drift the same way as
	// whatever OS clock change a developer is testing locally.
	rows, err := tx.QueryContext(ctx, `
		SELECT id, category, session_id, expires_at
		FROM tickets
		WHERE status = 'Holding' AND expires_at <= NOW()
		FOR UPDATE
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type reclaimedTicket struct {
		ID        int64
		Category  string
		SessionID string
		ExpiresAt time.Time
	}

	var candidates []reclaimedTicket
	for rows.Next() {
		var t reclaimedTicket
		if err := rows.Scan(&t.ID, &t.Category, &t.SessionID, &t.ExpiresAt); err != nil {
			return 0, err
		}
		candidates = append(candidates, t)
	}
	rows.Close()

	var tickets []reclaimedTicket
	for _, t := range candidates {
		if holdtimer.SecondsRemaining(t.ID, t.ExpiresAt, constants.TicketHoldTTL) > 0 {
			// The monotonic anchor says this hold genuinely still has time
			// left despite Postgres's NOW() reporting it as expired; skip
			// it this sweep, it'll be re-evaluated on the next tick.
			continue
		}
		tickets = append(tickets, t)
	}

	if len(tickets) == 0 {
		return 0, nil
	}

	// Update the tickets in PostgreSQL
	for _, t := range tickets {
		_, err = tx.ExecContext(ctx, `
			UPDATE tickets 
			SET status = 'Available', session_id = NULL, held_at = NULL, expires_at = NULL, updated_at = NOW() 
			WHERE id = $1 AND status = 'Holding'
		`, t.ID)
		if err != nil {
			return 0, err
		}
	}

	// Commit PostgreSQL transaction
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	for _, t := range tickets {
		holdtimer.Clear(t.ID)
	}

	// Sync Redis
	var count int64
	for _, t := range tickets {
		pipe := r.redis.Pipeline()
		pipe.Del(ctx, "hold:"+t.SessionID)
		pipe.SAdd(ctx, "tickets:available:"+t.Category, t.ID)
		_, err = pipe.Exec(ctx)
		if err != nil {
			// Log or handle single key sync failure, but keep going
			continue
		}
		count++
	}

	return count, nil
}

func (r *reclamationRepository) ReclaimSessionHold(ctx context.Context, sessionID string) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	var id int64
	var category string
	err = tx.QueryRowContext(ctx, `
		SELECT id, category 
		FROM tickets 
		WHERE session_id = $1 AND status = 'Holding' 
		FOR UPDATE
	`, sessionID).Scan(&id, &category)

	if err != nil {
		if err == sql.ErrNoRows {
			// No active hold found for this session, nothing to do
			return "", nil
		}
		return "", err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE tickets 
		SET status = 'Available', session_id = NULL, held_at = NULL, expires_at = NULL, updated_at = NOW() 
		WHERE id = $1 AND status = 'Holding'
	`, id)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}
	holdtimer.Clear(id)

	// Sync Redis
	pipe := r.redis.Pipeline()
	pipe.Del(ctx, "hold:"+sessionID)
	pipe.SAdd(ctx, "tickets:available:"+category, id)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return "", err
	}

	return category, nil
}
