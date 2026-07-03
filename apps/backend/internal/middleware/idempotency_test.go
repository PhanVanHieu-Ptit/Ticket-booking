package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func setupIdempotencyTest(t *testing.T) (*redis.Client, context.Context) {
	t.Helper()

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
		t.Skip("Redis is not running, skipping integration test")
	}
	t.Cleanup(func() { rdb.Close() })

	return rdb, ctx
}

func newIdempotencyRouter(rdb *redis.Client, sessionID string, callCount *int32) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandlerMiddleware(false))
	r.Use(func(c *gin.Context) {
		if sessionID != "" {
			c.Set("session_id", sessionID)
		}
		c.Next()
	})
	r.Use(IdempotencyMiddleware(rdb))
	r.POST("/checkout", func(c *gin.Context) {
		if callCount != nil {
			atomic.AddInt32(callCount, 1)
		}
		c.JSON(http.StatusCreated, gin.H{"order_id": "test-order-123"})
	})
	return r
}

func TestIdempotencyMiddleware_MissingHeader(t *testing.T) {
	rdb, _ := setupIdempotencyTest(t)
	r := newIdempotencyRouter(rdb, "sess_idem_1", nil)

	req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing Idempotency-Key, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestIdempotencyMiddleware_InvalidUUID(t *testing.T) {
	rdb, _ := setupIdempotencyTest(t)
	r := newIdempotencyRouter(rdb, "sess_idem_2", nil)

	req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
	req.Header.Set("Idempotency-Key", "not-a-uuid")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid UUID, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestIdempotencyMiddleware_MissingSession(t *testing.T) {
	rdb, _ := setupIdempotencyTest(t)
	r := newIdempotencyRouter(rdb, "", nil) // no session_id set in context

	req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
	req.Header.Set("Idempotency-Key", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing session, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestIdempotencyMiddleware_ReplaysResolvedResponseWithoutRerunningHandler(t *testing.T) {
	rdb, ctx := setupIdempotencyTest(t)
	sessionID := "sess_idem_replay"
	key := uuid.New().String()

	t.Cleanup(func() { rdb.Del(ctx, "idempotency:"+sessionID+":"+key) })

	var callCount int32
	r := newIdempotencyRouter(rdb, sessionID, &callCount)

	// First request: handler should run and response gets cached.
	req1 := httptest.NewRequest(http.MethodPost, "/checkout", nil)
	req1.Header.Set("Idempotency-Key", key)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("expected 201 on first request, got %d. Body: %s", w1.Code, w1.Body.String())
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected handler to run exactly once, ran %d times", callCount)
	}

	// Second request with the same key: should replay the cached response and
	// NOT invoke the handler again.
	req2 := httptest.NewRequest(http.MethodPost, "/checkout", nil)
	req2.Header.Set("Idempotency-Key", key)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusCreated {
		t.Fatalf("expected replayed 201, got %d. Body: %s", w2.Code, w2.Body.String())
	}
	if w2.Body.String() != w1.Body.String() {
		t.Errorf("expected replayed body to match original, got %q vs %q", w2.Body.String(), w1.Body.String())
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected handler NOT to run again on replay, ran %d times total", callCount)
	}
}

func TestIdempotencyMiddleware_PendingRequestReturnsConflict(t *testing.T) {
	rdb, ctx := setupIdempotencyTest(t)
	sessionID := "sess_idem_pending"
	key := uuid.New().String()
	redisKey := "idempotency:" + sessionID + ":" + key

	rdb.Set(ctx, redisKey, `{"status":"PENDING"}`, 0)
	t.Cleanup(func() { rdb.Del(ctx, redisKey) })

	var callCount int32
	r := newIdempotencyRouter(rdb, sessionID, &callCount)

	req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
	req.Header.Set("Idempotency-Key", key)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for in-progress request, got %d. Body: %s", w.Code, w.Body.String())
	}
	if atomic.LoadInt32(&callCount) != 0 {
		t.Errorf("expected handler not to run while a request is PENDING, ran %d times", callCount)
	}
}

func TestIdempotencyMiddleware_FailedHandlerDoesNotCacheAndAllowsRetry(t *testing.T) {
	rdb, ctx := setupIdempotencyTest(t)
	sessionID := "sess_idem_fail"
	key := uuid.New().String()
	redisKey := "idempotency:" + sessionID + ":" + key
	t.Cleanup(func() { rdb.Del(ctx, redisKey) })

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandlerMiddleware(false))
	r.Use(func(c *gin.Context) {
		c.Set("session_id", sessionID)
		c.Next()
	})
	r.Use(IdempotencyMiddleware(rdb))
	var callCount int32
	r.POST("/checkout", func(c *gin.Context) {
		atomic.AddInt32(&callCount, 1)
		c.Error(&testAppError{})
	})

	req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
	req.Header.Set("Idempotency-Key", key)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 from failing handler, got %d. Body: %s", w.Code, w.Body.String())
	}

	exists, err := rdb.Exists(ctx, redisKey).Result()
	if err != nil {
		t.Fatalf("failed to check idempotency key existence: %v", err)
	}
	if exists != 0 {
		t.Error("expected idempotency key to be deleted after a failed request, allowing retry")
	}
}

// testAppError is a minimal error implementing the error interface, used to
// simulate a handler failure path via c.Error() without depending on the
// shared errors package's specific HTTP status mapping.
type testAppError struct{}

func (e *testAppError) Error() string { return "simulated handler failure" }
