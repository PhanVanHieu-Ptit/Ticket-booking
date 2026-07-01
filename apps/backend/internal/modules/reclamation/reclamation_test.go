package reclamation

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	appDB "github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/db"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/sse"
	"github.com/redis/go-redis/v9"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestReclamationFlow(t *testing.T) {
	// 1. Connect to Redis (isolation on DB 1)
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/1"
	} else {
		redisURL = strings.Replace(redisURL, "/0", "/1", 1)
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatalf("failed to parse Redis URL: %v", err)
	}
	rdb := redis.NewClient(opts)
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("Redis is not running, skipping integration tests")
		return
	}
	defer rdb.Close()

	// 2. Connect to PostgreSQL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:postgres@localhost:5432/ticket_booking?sslmode=disable"
	}
	dbConn, err := appDB.Init(dbURL)
	if err != nil {
		t.Skipf("PostgreSQL connection failed, skipping integration tests: %v", err)
		return
	}
	defer appDB.Close()

	// 3. Set up SSE Broker
	sse.GlobalBroker = sse.NewBroker()
	sse.GlobalBroker.Start()

	// 4. Cleanup old test data
	testSessionID := "sess_reclaim_test"
	rdb.Del(ctx, "tickets:available:VIP", "tickets:available:Standard", "hold:"+testSessionID)
	_, _ = dbConn.ExecContext(ctx, "DELETE FROM tickets WHERE ticket_code LIKE 'TKT-REC-%'")
	_, _ = dbConn.ExecContext(ctx, "UPDATE tickets SET status = 'Available', session_id = NULL, held_at = NULL, expires_at = NULL WHERE status = 'Holding'")

	// 5. Insert test tickets in PostgreSQL
	var vipTicketID int64
	err = dbConn.QueryRowContext(ctx, `
		INSERT INTO tickets (ticket_code, category, price, status) 
		VALUES ('TKT-REC-VIP-001', 'VIP', 100.0, 'Available') 
		RETURNING id
	`).Scan(&vipTicketID)
	if err != nil {
		t.Fatalf("failed to insert VIP test ticket: %v", err)
	}

	// 6. Test ReclaimSessionHold: first, hold the ticket manually in PG and Redis
	heldAt := time.Now()
	expiresAt := heldAt.Add(5 * time.Minute)
	_, err = dbConn.ExecContext(ctx, `
		UPDATE tickets 
		SET status = 'Holding', session_id = $2, held_at = $3, expires_at = $4, updated_at = NOW() 
		WHERE id = $1
	`, vipTicketID, testSessionID, heldAt, expiresAt)
	if err != nil {
		t.Fatalf("failed to hold test ticket in PG: %v", err)
	}
	rdb.Set(ctx, "hold:"+testSessionID, fmt.Sprintf("%d:VIP", vipTicketID), 5*time.Minute)

	repo := NewReclamationRepository(dbConn, rdb)

	// Now run ReclaimSessionHold
	category, err := repo.ReclaimSessionHold(ctx, testSessionID)
	if err != nil {
		t.Fatalf("ReclaimSessionHold failed: %v", err)
	}
	if category != "VIP" {
		t.Errorf("expected reclaimed category VIP, got %q", category)
	}

	// Verify database was released
	var status string
	var sessionID sql.NullString
	err = dbConn.QueryRowContext(ctx, "SELECT status, session_id FROM tickets WHERE id = $1", vipTicketID).Scan(&status, &sessionID)
	if err != nil {
		t.Fatalf("failed to query ticket status: %v", err)
	}
	if status != "Available" || sessionID.Valid {
		t.Errorf("expected ticket status Available and NULL session, got %s, %v", status, sessionID)
	}

	// Verify Redis holds were cleared and available set updated
	existsVal, err := rdb.Exists(ctx, "hold:"+testSessionID).Result()
	if err != nil {
		t.Fatalf("failed to check hold in Redis: %v", err)
	}
	if existsVal != 0 {
		t.Errorf("expected hold key to be deleted from Redis")
	}

	isMember, err := rdb.SIsMember(ctx, "tickets:available:VIP", vipTicketID).Result()
	if err != nil {
		t.Fatalf("failed to check availability set in Redis: %v", err)
	}
	if !isMember {
		t.Errorf("expected ticket ID %d to be restored to Redis available set", vipTicketID)
	}

	// 7. Test ReclaimExpiredHolds (periodic DB sweeper sweep)
	// Reset ticket hold in PG, but set expires_at in the past
	expiredExpiresAt := time.Now().Add(-1 * time.Minute)
	_, err = dbConn.ExecContext(ctx, `
		UPDATE tickets 
		SET status = 'Holding', session_id = $2, held_at = $3, expires_at = $4, updated_at = NOW() 
		WHERE id = $1
	`, vipTicketID, testSessionID, expiredExpiresAt.Add(-5*time.Minute), expiredExpiresAt)
	if err != nil {
		t.Fatalf("failed to update ticket to expired hold in PG: %v", err)
	}
	rdb.Set(ctx, "hold:"+testSessionID, fmt.Sprintf("%d:VIP", vipTicketID), 1*time.Minute)
	rdb.SRem(ctx, "tickets:available:VIP", vipTicketID) // ensure not in set

	// Execute ReclaimExpiredHolds
	reclaimedCount, err := repo.ReclaimExpiredHolds(ctx)
	if err != nil {
		t.Fatalf("ReclaimExpiredHolds failed: %v", err)
	}
	if reclaimedCount != 1 {
		t.Errorf("expected reclaimed count 1, got %d", reclaimedCount)
	}

	// Verify DB is Available again
	err = dbConn.QueryRowContext(ctx, "SELECT status FROM tickets WHERE id = $1", vipTicketID).Scan(&status)
	if err != nil {
		t.Fatalf("failed to query ticket status: %v", err)
	}
	if status != "Available" {
		t.Errorf("expected ticket to be Available after sweep, got %s", status)
	}

	// Verify Redis set is updated
	isMember, err = rdb.SIsMember(ctx, "tickets:available:VIP", vipTicketID).Result()
	if err != nil {
		t.Fatalf("failed to check Redis: %v", err)
	}
	if !isMember {
		t.Errorf("expected ticket to be restored to Redis availability pool after cron sweep")
	}

	// Cleanup at the end
	rdb.Del(ctx, "tickets:available:VIP", "tickets:available:Standard", "hold:"+testSessionID)
	_, _ = dbConn.ExecContext(ctx, "DELETE FROM tickets WHERE ticket_code LIKE 'TKT-REC-%'")
}
