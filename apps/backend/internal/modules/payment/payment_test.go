package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/middleware"
	appDB "github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/db"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/types"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/sse"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
