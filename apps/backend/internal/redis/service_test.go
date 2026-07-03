package redis

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func setupRedisServiceTest(t *testing.T) (*redis.Client, context.Context) {
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

func TestRedisService_HoldTicket_Success(t *testing.T) {
	rdb, ctx := setupRedisServiceTest(t)
	svc := NewRedisService(rdb)

	category := "SvcTestVIP"
	sessionID := "sess_redis_svc_success"
	availableKey := "tickets:available:" + category
	holdKey := "hold:" + sessionID

	rdb.Del(ctx, availableKey, holdKey)
	t.Cleanup(func() { rdb.Del(ctx, availableKey, holdKey) })

	rdb.SAdd(ctx, availableKey, 4242)

	ticketID, err := svc.HoldTicket(ctx, sessionID, category, 5*time.Minute)
	if err != nil {
		t.Fatalf("expected HoldTicket to succeed, got error: %v", err)
	}
	if ticketID != 4242 {
		t.Errorf("expected popped ticket ID 4242, got %d", ticketID)
	}

	card, err := rdb.SCard(ctx, availableKey).Result()
	if err != nil {
		t.Fatalf("failed to check available set: %v", err)
	}
	if card != 0 {
		t.Errorf("expected ticket to be popped from available set, %d remain", card)
	}

	ttl, err := rdb.TTL(ctx, holdKey).Result()
	if err != nil {
		t.Fatalf("failed to check hold key TTL: %v", err)
	}
	if ttl <= 0 || ttl > 5*time.Minute {
		t.Errorf("expected hold key TTL between 0 and 5m, got %v", ttl)
	}
}

func TestRedisService_HoldTicket_PurchaseLimitExceeded(t *testing.T) {
	rdb, ctx := setupRedisServiceTest(t)
	svc := NewRedisService(rdb)

	sessionID := "sess_redis_svc_purchased"
	category := "SvcTestStandard"
	availableKey := "tickets:available:" + category

	rdb.Del(ctx, availableKey, "hold:"+sessionID)
	rdb.SAdd(ctx, "purchased:sessions", sessionID)
	t.Cleanup(func() { rdb.SRem(ctx, "purchased:sessions", sessionID); rdb.Del(ctx, availableKey) })

	rdb.SAdd(ctx, availableKey, 1)

	_, err := svc.HoldTicket(ctx, sessionID, category, 5*time.Minute)
	if err != ErrPurchaseLimitExceeded {
		t.Errorf("expected ErrPurchaseLimitExceeded, got %v", err)
	}
}

func TestRedisService_HoldTicket_ActiveHoldExists(t *testing.T) {
	rdb, ctx := setupRedisServiceTest(t)
	svc := NewRedisService(rdb)

	sessionID := "sess_redis_svc_activehold"
	category := "SvcTestStandard2"
	availableKey := "tickets:available:" + category
	holdKey := "hold:" + sessionID

	rdb.Del(ctx, availableKey, holdKey)
	rdb.Set(ctx, holdKey, "999:SomeCategory", 5*time.Minute)
	t.Cleanup(func() { rdb.Del(ctx, availableKey, holdKey) })

	rdb.SAdd(ctx, availableKey, 1)

	_, err := svc.HoldTicket(ctx, sessionID, category, 5*time.Minute)
	if err != ErrActiveHoldExists {
		t.Errorf("expected ErrActiveHoldExists, got %v", err)
	}
}

func TestRedisService_HoldTicket_TicketUnavailable(t *testing.T) {
	rdb, ctx := setupRedisServiceTest(t)
	svc := NewRedisService(rdb)

	sessionID := "sess_redis_svc_unavailable"
	category := "SvcTestEmptyCat"
	availableKey := "tickets:available:" + category

	rdb.Del(ctx, availableKey, "hold:"+sessionID)
	t.Cleanup(func() { rdb.Del(ctx, availableKey, "hold:"+sessionID) })
	// Deliberately leave availableKey empty/nonexistent.

	_, err := svc.HoldTicket(ctx, sessionID, category, 5*time.Minute)
	if err != ErrTicketUnavailable {
		t.Errorf("expected ErrTicketUnavailable, got %v", err)
	}
}

func TestRedisService_ReleaseHold(t *testing.T) {
	rdb, ctx := setupRedisServiceTest(t)
	svc := NewRedisService(rdb)

	sessionID := "sess_redis_svc_release"
	category := "SvcTestRelease"
	availableKey := "tickets:available:" + category
	holdKey := "hold:" + sessionID
	ticketID := int64(777)

	rdb.Del(ctx, availableKey, holdKey)
	rdb.Set(ctx, holdKey, fmt.Sprintf("%d:%s", ticketID, category), 5*time.Minute)
	t.Cleanup(func() { rdb.Del(ctx, availableKey, holdKey) })

	if err := svc.ReleaseHold(ctx, sessionID, category, ticketID); err != nil {
		t.Fatalf("ReleaseHold failed: %v", err)
	}

	exists, err := rdb.Exists(ctx, holdKey).Result()
	if err != nil {
		t.Fatalf("failed to check hold key: %v", err)
	}
	if exists != 0 {
		t.Error("expected hold key to be deleted after ReleaseHold")
	}

	isMember, err := rdb.SIsMember(ctx, availableKey, ticketID).Result()
	if err != nil {
		t.Fatalf("failed to check available set membership: %v", err)
	}
	if !isMember {
		t.Error("expected ticket to be returned to the available set after ReleaseHold")
	}
}

func TestRedisService_GetHoldTTL(t *testing.T) {
	rdb, ctx := setupRedisServiceTest(t)
	svc := NewRedisService(rdb)

	sessionID := "sess_redis_svc_ttl"
	holdKey := "hold:" + sessionID

	t.Run("existing hold returns positive TTL", func(t *testing.T) {
		rdb.Set(ctx, holdKey, "1:VIP", 5*time.Minute)
		t.Cleanup(func() { rdb.Del(ctx, holdKey) })

		ttl, err := svc.GetHoldTTL(ctx, sessionID)
		if err != nil {
			t.Fatalf("GetHoldTTL failed: %v", err)
		}
		if ttl <= 0 || ttl > 5*time.Minute {
			t.Errorf("expected TTL between 0 and 5m, got %v", ttl)
		}
	})

	t.Run("nonexistent hold returns negative TTL sentinel", func(t *testing.T) {
		rdb.Del(ctx, "hold:sess_redis_svc_ttl_missing")
		ttl, err := svc.GetHoldTTL(ctx, "sess_redis_svc_ttl_missing")
		if err != nil {
			t.Fatalf("expected no error for missing key, got: %v", err)
		}
		if ttl >= 0 {
			t.Errorf("expected negative TTL sentinel for nonexistent key, got %v", ttl)
		}
	})
}

func TestRedisService_AvailableCount(t *testing.T) {
	rdb, ctx := setupRedisServiceTest(t)
	svc := NewRedisService(rdb)

	category := "SvcTestCount"
	availableKey := "tickets:available:" + category

	t.Run("counts existing members", func(t *testing.T) {
		rdb.Del(ctx, availableKey)
		rdb.SAdd(ctx, availableKey, 1, 2, 3)
		t.Cleanup(func() { rdb.Del(ctx, availableKey) })

		count, err := svc.AvailableCount(ctx, category)
		if err != nil {
			t.Fatalf("AvailableCount failed: %v", err)
		}
		if count != 3 {
			t.Errorf("expected count 3, got %d", count)
		}
	})

	t.Run("unknown category returns zero without error", func(t *testing.T) {
		unknownCategory := "SvcTestNeverSeededCategory"
		rdb.Del(ctx, "tickets:available:"+unknownCategory)

		count, err := svc.AvailableCount(ctx, unknownCategory)
		if err != nil {
			t.Fatalf("expected no error for unknown category, got: %v", err)
		}
		if count != 0 {
			t.Errorf("expected count 0 for unknown category, got %d", count)
		}
	})
}

func TestRedisService_HoldTicket_ConcurrentPopsAreDisjoint(t *testing.T) {
	rdb, ctx := setupRedisServiceTest(t)
	svc := NewRedisService(rdb)

	category := "SvcTestConcurrent"
	availableKey := "tickets:available:" + category
	const numTickets = 20

	rdb.Del(ctx, availableKey)
	t.Cleanup(func() {
		rdb.Del(ctx, availableKey)
	})

	ticketIDs := make([]interface{}, numTickets)
	for i := 0; i < numTickets; i++ {
		ticketIDs[i] = int64(90000 + i)
	}
	rdb.SAdd(ctx, availableKey, ticketIDs...)

	var (
		mu       sync.Mutex
		popped   = map[int64]int{}
		wg       sync.WaitGroup
		holdKeys []string
	)

	for i := 0; i < numTickets; i++ {
		wg.Add(1)
		sessionID := fmt.Sprintf("sess_redis_svc_concurrent_%d", i)
		mu.Lock()
		holdKeys = append(holdKeys, "hold:"+sessionID)
		mu.Unlock()

		go func(sessID string) {
			defer wg.Done()
			ticketID, err := svc.HoldTicket(ctx, sessID, category, 5*time.Minute)
			if err != nil {
				return
			}
			mu.Lock()
			popped[ticketID]++
			mu.Unlock()
		}(sessionID)
	}
	wg.Wait()
	t.Cleanup(func() {
		if len(holdKeys) > 0 {
			rdb.Del(ctx, holdKeys...)
		}
	})

	if len(popped) != numTickets {
		t.Errorf("expected %d distinct tickets popped, got %d", numTickets, len(popped))
	}
	for ticketID, count := range popped {
		if count != 1 {
			t.Errorf("ticket %d was popped %d times, expected exactly once", ticketID, count)
		}
	}
}
