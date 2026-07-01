package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"strconv"

	appDB "github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/db"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/sse"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestReservationFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 1. Connect to Redis (isolation on DB 1)
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/1"
	} else {
		// Use database 1 for test isolation if possible
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
	rdb.Del(ctx, "tickets:available:VIP", "tickets:available:Standard", "hold:sess_test_success", "hold:sess_test_double", "purchased:sessions")
	_, _ = dbConn.ExecContext(ctx, "DELETE FROM tickets WHERE ticket_code LIKE 'TKT-TEST-%'")

	// 5. Insert test tickets in PostgreSQL
	var vipTicketID int64
	err = dbConn.QueryRowContext(ctx, `
		INSERT INTO tickets (ticket_code, category, price, status) 
		VALUES ('TKT-TEST-VIP-001', 'VIP', 100.0, 'Available') 
		RETURNING id
	`).Scan(&vipTicketID)
	if err != nil {
		t.Fatalf("failed to insert VIP test ticket: %v", err)
	}

	var stdTicketID int64
	err = dbConn.QueryRowContext(ctx, `
		INSERT INTO tickets (ticket_code, category, price, status) 
		VALUES ('TKT-TEST-STD-001', 'Standard', 50.0, 'Available') 
		RETURNING id
	`).Scan(&stdTicketID)
	if err != nil {
		t.Fatalf("failed to insert Standard test ticket: %v", err)
	}

	// 6. Populate Redis available ticket sets
	rdb.SAdd(ctx, "tickets:available:VIP", vipTicketID)
	// We deliberately DO NOT populate Standard category in Redis to simulate "sold out/unavailable" behavior

	jwtSecret := []byte("test-jwt-secret-key-2026")
	handler := NewReservationHandler(dbConn, rdb, jwtSecret)

	r := gin.New()
	// Mock middleware to inject session_id
	r.Use(func(c *gin.Context) {
		sessID := c.GetHeader("X-Test-Session-ID")
		if sessID != "" {
			c.Set("session_id", sessID)
			c.Set("session_expires_at", time.Now().Add(30 * time.Minute))
		}
		c.Next()
	})

	r.POST("/api/v1/tickets/reserve", handler.ReserveTicket)
	r.GET("/api/v1/tickets/hold", handler.GetActiveHold)
	r.POST("/api/v1/tickets/hold/cancel", handler.CancelHold)

	// Cleanup on exit
	defer func() {
		rdb.Del(ctx, "tickets:available:VIP", "tickets:available:Standard", "hold:sess_test_success", "hold:sess_test_double", "purchased:sessions")
		_, _ = dbConn.ExecContext(ctx, "DELETE FROM tickets WHERE ticket_code LIKE 'TKT-TEST-%'")
	}()

	// CASE 1: Successful Ticket Reservation
	t.Run("Reserve Ticket Success", func(t *testing.T) {
		body := `{"category": "VIP"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tickets/reserve", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-Session-ID", "sess_test_success")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201 Created, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				TicketID         int64   `json:"ticket_id"`
				TicketCode       string  `json:"ticket_code"`
				Category         string  `json:"category"`
				Price            float64 `json:"price"`
				Status           string  `json:"status"`
				SecondsRemaining int64   `json:"seconds_remaining"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if !resp.Success {
			t.Error("expected response.success to be true")
		}

		if resp.Data.TicketID != vipTicketID {
			t.Errorf("expected ticket ID %d, got %d", vipTicketID, resp.Data.TicketID)
		}

		if resp.Data.Status != "Holding" {
			t.Errorf("expected ticket status Holding, got %s", resp.Data.Status)
		}

		// Verify database state
		var dbStatus string
		var dbSessionID sql.NullString
		err := dbConn.QueryRowContext(ctx, "SELECT status, session_id FROM tickets WHERE id = $1", vipTicketID).Scan(&dbStatus, &dbSessionID)
		if err != nil {
			t.Fatalf("failed to query ticket state from PostgreSQL: %v", err)
		}
		if dbStatus != "Holding" || dbSessionID.String != "sess_test_success" {
			t.Errorf("PostgreSQL update was not applied correctly: status=%s, session_id=%s", dbStatus, dbSessionID.String)
		}

		// Verify Redis hold key state
		holdVal, err := rdb.Get(ctx, "hold:sess_test_success").Result()
		if err != nil {
			t.Fatalf("Redis hold key not set: %v", err)
		}
		expectedHoldVal := intToStr(vipTicketID) + ":VIP"
		if holdVal != expectedHoldVal {
			t.Errorf("expected Redis hold key %q, got %q", expectedHoldVal, holdVal)
		}
	})

	// CASE 2: Active Hold Exists
	t.Run("Active Hold Exists Block", func(t *testing.T) {
		body := `{"category": "VIP"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tickets/reserve", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-Session-ID", "sess_test_success") // Same session ID as Case 1

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 Bad Request, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Success bool `json:"success"`
			Error   struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse error response: %v", err)
		}

		if resp.Error.Code != "ACTIVE_HOLD_EXISTS" {
			t.Errorf("expected error code ACTIVE_HOLD_EXISTS, got %s", resp.Error.Code)
		}
	})

	// CASE 3: Category Sold Out / Unavailable
	t.Run("Category Sold Out Block", func(t *testing.T) {
		body := `{"category": "Standard"}` // Empty in Redis available set
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tickets/reserve", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-Session-ID", "sess_test_double")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected status 409 Conflict, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Success bool `json:"success"`
			Error   struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.Error.Code != "TICKET_UNAVAILABLE" {
			t.Errorf("expected error code TICKET_UNAVAILABLE, got %s", resp.Error.Code)
		}
	})

	// CASE 4: Get Active Hold Details
	t.Run("Get Active Hold Details Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/hold", nil)
		req.Header.Set("X-Test-Session-ID", "sess_test_success")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				TicketID         int64  `json:"ticket_id"`
				TicketCode       string `json:"ticket_code"`
				Category         string `json:"category"`
				SecondsRemaining int64  `json:"seconds_remaining"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.Data.TicketID != vipTicketID {
			t.Errorf("expected ticket ID %d, got %d", vipTicketID, resp.Data.TicketID)
		}

		if resp.Data.SecondsRemaining <= 0 || resp.Data.SecondsRemaining > 300 {
			t.Errorf("unexpected seconds remaining: %d", resp.Data.SecondsRemaining)
		}
	})

	// CASE 5: Get Active Hold Details Not Found
	t.Run("Get Active Hold Details Not Found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/hold", nil)
		req.Header.Set("X-Test-Session-ID", "sess_test_nohold")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404 Not Found, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Success bool `json:"success"`
			Error   struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.Error.Code != "NO_ACTIVE_HOLD" {
			t.Errorf("expected error code NO_ACTIVE_HOLD, got %s", resp.Error.Code)
		}
	})

	// CASE 6: Purchase Limit Exceeded Check
	t.Run("Purchase Limit Exceeded Block", func(t *testing.T) {
		// Populate purchase set in Redis
		rdb.SAdd(ctx, "purchased:sessions", "sess_test_purchased")

		body := `{"category": "VIP"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tickets/reserve", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-Session-ID", "sess_test_purchased")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 Bad Request, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Success bool `json:"success"`
			Error   struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.Error.Code != "LIMIT_EXCEEDED" && resp.Error.Code != "PURCHASE_LIMIT_EXCEEDED" {
			t.Errorf("expected error code PURCHASE_LIMIT_EXCEEDED, got %s", resp.Error.Code)
		}
	})

	// CASE 7: Cancel Reservation Success
	t.Run("Cancel Reservation Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tickets/hold/cancel", nil)
		req.Header.Set("X-Test-Session-ID", "sess_test_success")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Success bool `json:"success"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if !resp.Success {
			t.Error("expected response.success to be true")
		}

		// Verify database state: status should be 'Available', session_id, held_at, expires_at should be NULL
		var dbStatus string
		var dbSessionID sql.NullString
		err := dbConn.QueryRowContext(ctx, "SELECT status, session_id FROM tickets WHERE id = $1", vipTicketID).Scan(&dbStatus, &dbSessionID)
		if err != nil {
			t.Fatalf("failed to query ticket state from PostgreSQL: %v", err)
		}
		if dbStatus != "Available" || dbSessionID.Valid {
			t.Errorf("PostgreSQL ticket was not released correctly: status=%s, session_id=%v", dbStatus, dbSessionID)
		}

		// Verify Redis hold key is deleted
		exists, err := rdb.Exists(ctx, "hold:sess_test_success").Result()
		if err != nil {
			t.Fatalf("failed to query Redis hold key: %v", err)
		}
		if exists > 0 {
			t.Error("expected Redis hold key to be deleted")
		}

		// Verify Redis available set contains the ticket ID again
		isMember, err := rdb.SIsMember(ctx, "tickets:available:VIP", vipTicketID).Result()
		if err != nil {
			t.Fatalf("failed to query Redis available set: %v", err)
		}
		if !isMember {
			t.Error("expected VIP ticket ID to be returned to tickets:available:VIP set")
		}
	})

	// CASE 8: Cancel Reservation Not Found / No Active Hold
	t.Run("Cancel Reservation Not Found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tickets/hold/cancel", nil)
		req.Header.Set("X-Test-Session-ID", "sess_test_nohold")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 Bad Request, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Success bool `json:"success"`
			Error   struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.Error.Code != "NO_ACTIVE_HOLD" {
			t.Errorf("expected error code NO_ACTIVE_HOLD, got %s", resp.Error.Code)
		}
	})
}

func intToStr(i int64) string {
	return strconv.FormatInt(i, 10)
}
