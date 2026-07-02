package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/middleware"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/session"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/sse"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func TestGetAvailability(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set up Redis client on db 1 for test isolation
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

	// Clear previous keys
	rdb.Del(ctx, "tickets:available:VIP", "tickets:available:Standard")

	// Set up mock inventory
	rdb.SAdd(ctx, "tickets:available:VIP", 1, 2, 3)
	rdb.SAdd(ctx, "tickets:available:Standard", 101, 102)

	broker := sse.NewBroker()
	broker.Start()

	handler := NewAvailabilityHandler(nil, rdb, broker, []byte("test-secret"))

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

	if len(resp.Data.Categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(resp.Data.Categories))
	}

	// Verify VIP category
	vip := resp.Data.Categories[0]
	if vip.Name != "VIP" || vip.Available != 3 || vip.Status != "Available" {
		t.Errorf("unexpected VIP details: %+v", vip)
	}

	// Verify Standard category
	std := resp.Data.Categories[1]
	if std.Name != "Standard" || std.Available != 2 || std.Status != "Available" {
		t.Errorf("unexpected Standard details: %+v", std)
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

	rdb.Del(ctx, "tickets:available:VIP", "tickets:available:Standard")
	rdb.SAdd(ctx, "tickets:available:VIP", 1, 2)
	rdb.SAdd(ctx, "tickets:available:Standard", 101)

	broker := sse.NewBroker()
	broker.Start()
	sse.GlobalBroker = broker

	jwtSecret := []byte("test-jwt-secret-key-2026")
	handler := NewAvailabilityHandler(nil, rdb, broker, jwtSecret)

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

	if !strings.Contains(body, `{"Standard":{"available":1,"status":"Available"},"VIP":{"available":2,"status":"Available"}}`) {
		t.Errorf("expected stream body to contain initial ticket counts, got: %s", body)
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

