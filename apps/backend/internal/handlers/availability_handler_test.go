package handlers

import (
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
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/session"
	appDB "github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/db"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/sse"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
)

func TestGetAvailability(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("Redis is not running at localhost:6379, skipping integration test")
		return
	}
	defer rdb.Close()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:postgres@localhost:5432/ticket_booking?sslmode=disable"
	}
	dbConn, err := appDB.Init(dbURL)
	if err != nil {
		t.Skipf("PostgreSQL connection failed, skipping integration test: %v", err)
		return
	}
	defer appDB.Close()

	// Use a category name that cannot collide with any pre-existing seed data
	// (e.g. VIP/Standard) so this test proves categories are derived
	// dynamically from the tickets table, not hardcoded in Go.
	const testCategory = "AvailTestCat"
	const testPrice = 75.0
	redisKey := "tickets:available:" + testCategory

	cleanup := func() {
		rdb.Del(ctx, redisKey)
		_, _ = dbConn.ExecContext(ctx, "DELETE FROM tickets WHERE ticket_code LIKE 'TKT-AVAIL-TEST-%'")
	}
	cleanup()
	defer cleanup()

	// Insert 3 tickets in the new category; only 2 will be marked available
	// in Redis so we can assert available (2) independently from total (3).
	var ticketIDs [3]int64
	for i := 0; i < 3; i++ {
		err := dbConn.QueryRowContext(ctx, `
			INSERT INTO tickets (ticket_code, category, price, status)
			VALUES ($1, $2, $3, 'Available')
			RETURNING id
		`, fmt.Sprintf("TKT-AVAIL-TEST-%03d", i+1), testCategory, testPrice).Scan(&ticketIDs[i])
		if err != nil {
			t.Fatalf("failed to insert test ticket %d: %v", i, err)
		}
	}
	rdb.SAdd(ctx, redisKey, ticketIDs[0], ticketIDs[1])

	broker := sse.NewBroker()
	broker.Start()

	handler := NewAvailabilityHandler(dbConn, rdb, broker, []byte("test-secret"))

	r := gin.New()
	r.Use(middleware.ErrorHandlerMiddleware(false))
	r.GET("/api/v1/tickets/availability", handler.GetAvailability)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/availability", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			EventName     string `json:"event_name"`
			TotalCapacity int    `json:"total_capacity"`
			Categories    []struct {
				Name      string  `json:"name"`
				Price     float64 `json:"price"`
				Available int     `json:"available"`
				Total     int     `json:"total"`
				Status    string  `json:"status"`
			} `json:"categories"`
		} `json:"data"`
	}

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !resp.Success {
		t.Error("expected response.success to be true")
	}

	var found *struct {
		Name      string  `json:"name"`
		Price     float64 `json:"price"`
		Available int     `json:"available"`
		Total     int     `json:"total"`
		Status    string  `json:"status"`
	}
	for i := range resp.Data.Categories {
		if resp.Data.Categories[i].Name == testCategory {
			found = &resp.Data.Categories[i]
			break
		}
	}

	if found == nil {
		t.Fatalf("expected category %q to be present in dynamically-derived response, got: %+v", testCategory, resp.Data.Categories)
	}
	if found.Price != testPrice {
		t.Errorf("expected price %v, got %v", testPrice, found.Price)
	}
	if found.Total != 3 {
		t.Errorf("expected total 3, got %d", found.Total)
	}
	if found.Available != 2 {
		t.Errorf("expected available 2, got %d", found.Available)
	}
	if found.Status != "Available" {
		t.Errorf("expected status Available, got %s", found.Status)
	}
}

func TestStreamAvailability_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewAvailabilityHandler(nil, nil, nil, []byte("test-secret"))

	r := gin.New()
	r.Use(middleware.ErrorHandlerMiddleware(false))
	r.GET("/api/v1/tickets/availability/stream", handler.StreamAvailability)

	// Try without token
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/availability/stream", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	var resp struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Success {
		t.Error("expected response.success to be false")
	}

	if resp.Error.Code != "INVALID_SESSION_TOKEN" {
		t.Errorf("expected error code INVALID_SESSION_TOKEN, got %s", resp.Error.Code)
	}
}

func TestStreamAvailability_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("Redis is not running at localhost:6379, skipping integration test")
		return
	}
	defer rdb.Close()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:postgres@localhost:5432/ticket_booking?sslmode=disable"
	}
	dbConn, err := appDB.Init(dbURL)
	if err != nil {
		t.Skipf("PostgreSQL connection failed, skipping integration test: %v", err)
		return
	}
	defer appDB.Close()

	// Same rationale as TestGetAvailability: use a category name that can't
	// collide with pre-existing seed data, proving the SSE initial_state
	// payload is built dynamically rather than hardcoded to VIP/Standard.
	const testCategory = "StreamTestCat"
	redisKey := "tickets:available:" + testCategory

	cleanup := func() {
		rdb.Del(ctx, redisKey)
		_, _ = dbConn.ExecContext(ctx, "DELETE FROM tickets WHERE ticket_code LIKE 'TKT-STREAM-TEST-%'")
	}
	cleanup()
	defer cleanup()

	var ticketID int64
	err = dbConn.QueryRowContext(ctx, `
		INSERT INTO tickets (ticket_code, category, price, status)
		VALUES ('TKT-STREAM-TEST-001', $1, 60.0, 'Available')
		RETURNING id
	`, testCategory).Scan(&ticketID)
	if err != nil {
		t.Fatalf("failed to insert test ticket: %v", err)
	}
	rdb.SAdd(ctx, redisKey, ticketID)

	broker := sse.NewBroker()
	broker.Start()
	sse.GlobalBroker = broker

	jwtSecret := []byte("test-jwt-secret-key-2026")
	handler := NewAvailabilityHandler(dbConn, rdb, broker, jwtSecret)

	r := gin.New()
	r.GET("/api/v1/tickets/availability/stream", handler.StreamAvailability)

	token, _, err := session.SignSessionToken("sess_test_123", jwtSecret)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	// Request with valid session token as query parameter
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/availability/stream?session_token="+token, nil)
	ctxCancel, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctxCancel)

	w := newCloseNotifyingRecorder()

	// Run handler in separate thread or short duration to read initial response
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Channel to signal execution completion
	done := make(chan bool)

	go func() {
		handler.StreamAvailability(c)
		done <- true
	}()

	// Wait briefly for initial_state event to be flushed
	time.Sleep(100 * time.Millisecond)

	// Cancel context and notify close to force connection close
	cancel()
	w.closeNotifyChan <- true

	// Wait for handler to exit
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not exit after context cancelled")
	}

	body := w.Body.String()
	if !strings.Contains(body, "event: initial_state") {
		t.Error("expected stream body to contain initial_state event")
	}

	expectedFragment := fmt.Sprintf(`"%s":{"available":1,"status":"Available"}`, testCategory)
	if !strings.Contains(body, expectedFragment) {
		t.Errorf("expected stream body to contain dynamically-derived category %q, got: %s", testCategory, body)
	}
}

type closeNotifyingRecorder struct {
	*httptest.ResponseRecorder
	closeNotifyChan chan bool
}

func newCloseNotifyingRecorder() *closeNotifyingRecorder {
	return &closeNotifyingRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		closeNotifyChan:  make(chan bool, 1),
	}
}

func (c *closeNotifyingRecorder) CloseNotify() <-chan bool {
	return c.closeNotifyChan
}
