package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/middleware"
	appDB "github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/db"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/types"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/sse"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPaymentCheckoutFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)

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
	testSessionID := "sess_payment_test"
	rdb.Del(ctx, "tickets:available:VIP", "tickets:available:Standard", "hold:"+testSessionID, "purchased:sessions")
	_, _ = dbConn.ExecContext(ctx, "DELETE FROM orders WHERE session_id = $1", testSessionID)
	_, _ = dbConn.ExecContext(ctx, "DELETE FROM tickets WHERE ticket_code LIKE 'TKT-PMT-%'")

	// 5. Setup test variables
	idempotencyKey := uuid.New().String()

	// 6. Insert a test ticket in PostgreSQL in "Holding" state
	var ticketID int64
	err = dbConn.QueryRowContext(ctx, `
		INSERT INTO tickets (ticket_code, category, price, status, session_id, held_at, expires_at) 
		VALUES ('TKT-PMT-VIP-001', 'VIP', 150.0, 'Holding', $1, NOW(), NOW() + INTERVAL '5 minutes') 
		RETURNING id
	`, testSessionID).Scan(&ticketID)
	if err != nil {
		t.Fatalf("failed to insert holding test ticket: %v", err)
	}

	// Mock hold in Redis
	rdb.Set(ctx, "hold:"+testSessionID, fmt.Sprintf("%d:VIP", ticketID), 5*time.Minute)

	// 7. Initialize Payment module
	module := NewModule(dbConn, rdb)

	r := gin.New()
	r.Use(middleware.ErrorHandlerMiddleware(false))
	// Mock middleware to inject session_id
	r.Use(func(c *gin.Context) {
		c.Set("session_id", testSessionID)
		c.Next()
	})
	// Attach IdempotencyMiddleware
	r.Use(middleware.IdempotencyMiddleware(rdb))

	r.POST("/api/v1/payments/checkout", module.Controller.Checkout)

	// 8. Scenario 1: Successful simulated checkout
	reqBody := CheckoutRequest{
		TicketID:       ticketID,
		Email:          "test@example.com",
		CardHolderName: "John Doe",
		PaymentMethod:  "credit_card",
		SimulateStatus: "success",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/checkout", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected checkout status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var successResp types.ResponseEnvelope[CheckoutResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &successResp); err != nil {
		t.Fatalf("failed to parse success response: %v", err)
	}

	if !successResp.Success {
		t.Fatalf("expected success response true, got false")
	}

	// Verify database changes
	var ticketStatus string
	err = dbConn.QueryRowContext(ctx, "SELECT status FROM tickets WHERE id = $1", ticketID).Scan(&ticketStatus)
	if err != nil || ticketStatus != "Sold" {
		t.Errorf("expected ticket status Sold, got %q (error: %v)", ticketStatus, err)
	}

	var orderCount int
	err = dbConn.QueryRowContext(ctx, "SELECT COUNT(*) FROM orders WHERE ticket_id = $1 AND session_id = $2 AND status = 'Paid'", ticketID, testSessionID).Scan(&orderCount)
	if err != nil || orderCount != 1 {
		t.Errorf("expected 1 paid order in db, got %d (error: %v)", orderCount, err)
	}

	// Verify Redis shield updates
	existsHold := rdb.Exists(ctx, "hold:"+testSessionID).Val()
	if existsHold != 0 {
		t.Errorf("expected Redis hold key to be deleted, but it exists")
	}

	isPurchased := rdb.SIsMember(ctx, "purchased:sessions", testSessionID).Val()
	if !isPurchased {
		t.Errorf("expected session ID to be in Redis purchased:sessions set")
	}

	// 9. Scenario 2: Idempotent request retry with same key
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/payments/checkout", bytes.NewBuffer(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", idempotencyKey)

	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected idempotent response status 200, got %d", w2.Code)
	}

	var retryResp types.ResponseEnvelope[CheckoutResponse]
	_ = json.Unmarshal(w2.Body.Bytes(), &retryResp)
	if retryResp.Data.OrderID != successResp.Data.OrderID {
		t.Errorf("expected cached Order ID %q, got %q", successResp.Data.OrderID, retryResp.Data.OrderID)
	}

	// 10. Scenario 3: Failed simulated checkout
	// Create another holding ticket
	testSessionID2 := "sess_payment_test_fail"
	rdb.Del(ctx, "hold:"+testSessionID2)
	var ticketID2 int64
	err = dbConn.QueryRowContext(ctx, `
		INSERT INTO tickets (ticket_code, category, price, status, session_id, held_at, expires_at) 
		VALUES ('TKT-PMT-VIP-002', 'VIP', 150.0, 'Holding', $1, NOW(), NOW() + INTERVAL '5 minutes') 
		RETURNING id
	`, testSessionID2).Scan(&ticketID2)
	if err != nil {
		t.Fatalf("failed to insert second holding test ticket: %v", err)
	}

	rdb.Set(ctx, "hold:"+testSessionID2, fmt.Sprintf("%d:VIP", ticketID2), 5*time.Minute)

	rFail := gin.New()
	rFail.Use(middleware.ErrorHandlerMiddleware(false))
	rFail.Use(func(c *gin.Context) {
		c.Set("session_id", testSessionID2)
		c.Next()
	})
	rFail.Use(middleware.IdempotencyMiddleware(rdb))
	rFail.POST("/api/v1/payments/checkout", module.Controller.Checkout)

	reqBodyFail := CheckoutRequest{
		TicketID:       ticketID2,
		Email:          "test@example.com",
		CardHolderName: "Jane Doe",
		PaymentMethod:  "credit_card",
		SimulateStatus: "fail",
	}
	bodyBytesFail, _ := json.Marshal(reqBodyFail)
	idempotencyKeyFail := uuid.New().String()

	reqFail := httptest.NewRequest(http.MethodPost, "/api/v1/payments/checkout", bytes.NewBuffer(bodyBytesFail))
	reqFail.Header.Set("Content-Type", "application/json")
	reqFail.Header.Set("Idempotency-Key", idempotencyKeyFail)

	wFail := httptest.NewRecorder()
	rFail.ServeHTTP(wFail, reqFail)

	if wFail.Code != http.StatusPaymentRequired {
		t.Fatalf("expected payment fail status 402, got %d. Body: %s", wFail.Code, wFail.Body.String())
	}

	// Verify database changes: ticket is still in holding state, no order is created
	var ticketStatus2 string
	err = dbConn.QueryRowContext(ctx, "SELECT status FROM tickets WHERE id = $1", ticketID2).Scan(&ticketStatus2)
	if err != nil || ticketStatus2 != "Holding" {
		t.Errorf("expected ticket status Holding, got %q (error: %v)", ticketStatus2, err)
	}

	var orderCount2 int
	err = dbConn.QueryRowContext(ctx, "SELECT COUNT(*) FROM orders WHERE ticket_id = $1", ticketID2).Scan(&orderCount2)
	if err != nil || orderCount2 != 0 {
		t.Errorf("expected 0 orders in db for failed checkout, got %d (error: %v)", orderCount2, err)
	}

	// Verify Redis idempotency key is deleted
	existsKey := rdb.Exists(ctx, "idempotency:"+testSessionID2+":"+idempotencyKeyFail).Val()
	if existsKey != 0 {
		t.Errorf("expected Redis idempotency key to be deleted on failure, but it exists")
	}
}

// TestPaymentCheckout_ConcurrentIdempotentRequests fires many concurrent
// checkout requests, all reusing the same Idempotency-Key, ticket and
// session, simulating a client double-click or a network-retry storm. The
// IdempotencyMiddleware's own check-then-set (GET, then SET) is not atomic,
// so it cannot guarantee mutual exclusion by itself; the real safety net is
// the row lock taken in paymentService.Checkout (LockTicket ... FOR UPDATE)
// plus the idx_orders_ticket_id_paid unique constraint. This test asserts
// that invariant holds under real concurrency: no matter how many identical
// requests race in, exactly one 'Paid' order is ever created for the ticket.
func TestPaymentCheckout_ConcurrentIdempotentRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

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

	sse.GlobalBroker = sse.NewBroker()
	sse.GlobalBroker.Start()

	testSessionID := "sess_payment_race_test"
	rdb.Del(ctx, "hold:"+testSessionID, "purchased:sessions")
	_, _ = dbConn.ExecContext(ctx, "DELETE FROM orders WHERE session_id = $1", testSessionID)
	_, _ = dbConn.ExecContext(ctx, "DELETE FROM tickets WHERE ticket_code LIKE 'TKT-PMTRACE-%'")
	defer func() {
		_, _ = dbConn.ExecContext(ctx, "DELETE FROM orders WHERE session_id = $1", testSessionID)
		_, _ = dbConn.ExecContext(ctx, "DELETE FROM tickets WHERE ticket_code LIKE 'TKT-PMTRACE-%'")
		rdb.Del(ctx, "hold:"+testSessionID, "purchased:sessions")
	}()

	var ticketID int64
	err = dbConn.QueryRowContext(ctx, `
		INSERT INTO tickets (ticket_code, category, price, status, session_id, held_at, expires_at)
		VALUES ('TKT-PMTRACE-VIP-001', 'VIP', 150.0, 'Holding', $1, NOW(), NOW() + INTERVAL '5 minutes')
		RETURNING id
	`, testSessionID).Scan(&ticketID)
	if err != nil {
		t.Fatalf("failed to insert holding test ticket: %v", err)
	}
	rdb.Set(ctx, "hold:"+testSessionID, fmt.Sprintf("%d:VIP", ticketID), 5*time.Minute)

	module := NewModule(dbConn, rdb)

	r := gin.New()
	r.Use(middleware.ErrorHandlerMiddleware(false))
	r.Use(func(c *gin.Context) {
		c.Set("session_id", testSessionID)
		c.Next()
	})
	r.Use(middleware.IdempotencyMiddleware(rdb))
	r.POST("/api/v1/payments/checkout", module.Controller.Checkout)

	sharedIdempotencyKey := uuid.New().String()
	reqBody := CheckoutRequest{
		TicketID:       ticketID,
		Email:          "race@example.com",
		CardHolderName: "Race Condition",
		PaymentMethod:  "credit_card",
		SimulateStatus: "success",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	const numConcurrent = 15
	var wg sync.WaitGroup
	start := make(chan struct{})
	statusCodes := make([]int, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start

			req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/checkout", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", sharedIdempotencyKey)

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			statusCodes[idx] = w.Code
		}(i)
	}
	close(start)
	wg.Wait()

	// Every response must be one of: success, cached/duplicate-in-progress,
	// or hold-already-consumed. Anything else (e.g. a 500) means the race
	// wasn't handled safely.
	successCount := 0
	for i, code := range statusCodes {
		switch code {
		case http.StatusOK:
			successCount++
		case http.StatusConflict, http.StatusGone:
			// DUPLICATE_REQUEST (in-flight) or hold no longer 'Holding'
			// (another request already consumed it) - both expected.
		default:
			t.Errorf("request %d: unexpected status code %d", i, code)
		}
	}
	if successCount < 1 {
		t.Fatalf("expected at least 1 successful checkout out of %d concurrent identical requests, got %d", numConcurrent, successCount)
	}

	// The invariant that actually matters: no matter how many identical
	// requests raced in, exactly one 'Paid' order was ever created.
	var paidOrderCount int
	err = dbConn.QueryRowContext(ctx, "SELECT COUNT(*) FROM orders WHERE ticket_id = $1 AND status = 'Paid'", ticketID).Scan(&paidOrderCount)
	if err != nil {
		t.Fatalf("failed to count paid orders: %v", err)
	}
	if paidOrderCount != 1 {
		t.Errorf("expected exactly 1 paid order despite %d concurrent identical requests, got %d", numConcurrent, paidOrderCount)
	}

	var ticketStatus string
	err = dbConn.QueryRowContext(ctx, "SELECT status FROM tickets WHERE id = $1", ticketID).Scan(&ticketStatus)
	if err != nil || ticketStatus != "Sold" {
		t.Errorf("expected ticket status Sold, got %q (error: %v)", ticketStatus, err)
	}
}

// TestPreventDuplicatePaidOrders verifies the DB-level safety net (migration
// 000002, idx_orders_ticket_id_paid): even a caller that bypasses the
// application-level ticket-status lock in Checkout cannot persist two 'Paid'
// orders for the same ticket. Checkout itself can't be driven into this race
// directly (it already refuses a second attempt once the ticket is no longer
// 'Holding'), so this exercises the repository call the constraint guards.
func TestPreventDuplicatePaidOrders(t *testing.T) {
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

	// Uses a ticket_code prefix distinct from TKT-PMT-% and TKT-TEST-% (used by
	// other tests' cleanup queries) so this test's leftover order — which
	// references its ticket via FK — can never block an unrelated test's
	// same-transaction DELETE (Postgres DELETE is all-or-nothing).
	const testTicketCode = "TKT-DUPCHK-001"

	ctx := context.Background()
	_, _ = dbConn.ExecContext(ctx, "DELETE FROM orders WHERE ticket_id IN (SELECT id FROM tickets WHERE ticket_code = $1)", testTicketCode)
	_, _ = dbConn.ExecContext(ctx, "DELETE FROM tickets WHERE ticket_code = $1", testTicketCode)
	defer func() {
		_, _ = dbConn.ExecContext(ctx, "DELETE FROM orders WHERE ticket_id IN (SELECT id FROM tickets WHERE ticket_code = $1)", testTicketCode)
		_, _ = dbConn.ExecContext(ctx, "DELETE FROM tickets WHERE ticket_code = $1", testTicketCode)
	}()

	var ticketID int64
	err = dbConn.QueryRowContext(ctx, `
		INSERT INTO tickets (ticket_code, category, price, status, session_id, held_at, expires_at)
		VALUES ($1, 'VIP', 150.0, 'Sold', 'sess_dup_test_1', NOW(), NOW() + INTERVAL '5 minutes')
		RETURNING id
	`, testTicketCode).Scan(&ticketID)
	if err != nil {
		t.Fatalf("failed to insert test ticket: %v", err)
	}

	repo := NewPaymentRepository(dbConn)

	createPaidOrder := func(sessionID, paymentRef string) error {
		tx, txErr := dbConn.BeginTx(ctx, nil)
		if txErr != nil {
			t.Fatalf("failed to begin transaction: %v", txErr)
		}
		defer tx.Rollback()

		order := &Order{
			TicketID:         ticketID,
			SessionID:        sessionID,
			Amount:           150.0,
			Status:           StatusPaid,
			Email:            "dup-test@example.com",
			CardHolderName:   "Dup Test",
			PaymentReference: paymentRef,
		}
		if createErr := repo.CreateOrder(ctx, tx, order); createErr != nil {
			return createErr
		}
		return tx.Commit()
	}

	if err := createPaidOrder("sess_dup_test_1", "PAY-DUP-TEST-1"); err != nil {
		t.Fatalf("expected first paid order to succeed, got error: %v", err)
	}

	err = createPaidOrder("sess_dup_test_2", "PAY-DUP-TEST-2")
	if err == nil {
		t.Fatalf("expected second paid order for the same ticket to be rejected by the DB, but it succeeded")
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("expected a pgconn.PgError, got: %v", err)
	}
	// idx_orders_ticket_id_paid (added by migration 000002) is the constraint
	// this test targets. Some environments may also carry a pre-existing,
	// stricter orders_ticket_id_key unique constraint that intercepts first —
	// either way the DB must reject the duplicate paid order.
	if pgErr.Code != "23505" || (pgErr.ConstraintName != "idx_orders_ticket_id_paid" && pgErr.ConstraintName != "orders_ticket_id_key") {
		t.Fatalf("expected unique_violation on idx_orders_ticket_id_paid, got code=%q constraint=%q", pgErr.Code, pgErr.ConstraintName)
	}

	var paidOrderCount int
	if err := dbConn.QueryRowContext(ctx, "SELECT COUNT(*) FROM orders WHERE ticket_id = $1 AND status = 'Paid'", ticketID).Scan(&paidOrderCount); err != nil {
		t.Fatalf("failed to count paid orders: %v", err)
	}
	if paidOrderCount != 1 {
		t.Errorf("expected exactly 1 paid order to persist, got %d", paidOrderCount)
	}
}
