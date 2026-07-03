package holdtimer

import (
	"sync"
	"testing"
	"time"
)

func TestSecondsRemaining_UsesMonotonicAnchorOverExpiresAt(t *testing.T) {
	ticketID := int64(1001)
	defer Clear(ticketID)

	ttl := 5 * time.Minute
	Set(ticketID, time.Now())

	// expiresAt deliberately wrong/stale; SecondsRemaining should ignore it
	// and rely on the anchor instead when one is present.
	staleExpiresAt := time.Now().Add(-1 * time.Hour)

	remaining := SecondsRemaining(ticketID, staleExpiresAt, ttl)
	if remaining <= 290 || remaining > 300 {
		t.Errorf("expected remaining close to 300s (ttl), got %d", remaining)
	}
}

func TestSecondsRemaining_FallsBackToExpiresAtWithoutAnchor(t *testing.T) {
	ticketID := int64(1002)
	// No Set() call, so no anchor exists for this ticketID.

	expiresAt := time.Now().Add(2 * time.Minute)
	remaining := SecondsRemaining(ticketID, expiresAt, 5*time.Minute)

	if remaining <= 110 || remaining > 120 {
		t.Errorf("expected remaining close to 120s (expiresAt fallback), got %d", remaining)
	}
}

func TestSecondsRemaining_ClampsNegativeToZero(t *testing.T) {
	ticketID := int64(1003)
	defer Clear(ticketID)

	// Anchor set far enough in the past that ttl has already elapsed.
	Set(ticketID, time.Now().Add(-10*time.Minute))
	remaining := SecondsRemaining(ticketID, time.Now(), 5*time.Minute)
	if remaining != 0 {
		t.Errorf("expected remaining to clamp to 0 when anchor-based hold expired, got %d", remaining)
	}

	// Fallback path: expiresAt already in the past, no anchor.
	ticketID2 := int64(1004)
	remaining2 := SecondsRemaining(ticketID2, time.Now().Add(-1*time.Minute), 5*time.Minute)
	if remaining2 != 0 {
		t.Errorf("expected remaining to clamp to 0 when expiresAt fallback is in the past, got %d", remaining2)
	}
}

func TestClear_RemovesAnchorAndFallsBackToExpiresAt(t *testing.T) {
	ticketID := int64(1005)
	Set(ticketID, time.Now())
	Clear(ticketID)

	expiresAt := time.Now().Add(1 * time.Minute)
	remaining := SecondsRemaining(ticketID, expiresAt, 5*time.Minute)
	if remaining <= 50 || remaining > 60 {
		t.Errorf("expected remaining close to 60s (expiresAt fallback after Clear), got %d", remaining)
	}
}

func TestHoldTimer_ConcurrentAccess(t *testing.T) {
	const numTickets = 50
	const numGoroutinesPerTicket = 10

	var wg sync.WaitGroup
	for i := 0; i < numTickets; i++ {
		ticketID := int64(2000 + i)
		for j := 0; j < numGoroutinesPerTicket; j++ {
			wg.Add(1)
			go func(id int64) {
				defer wg.Done()
				Set(id, time.Now())
				_ = SecondsRemaining(id, time.Now().Add(time.Minute), 5*time.Minute)
				Clear(id)
			}(ticketID)
		}
	}
	wg.Wait()
}
