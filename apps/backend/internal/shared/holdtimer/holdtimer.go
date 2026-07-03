// Package holdtimer tracks ticket-hold expiry using the process's monotonic
// clock instead of wall-clock arithmetic against a database timestamp.
//
// A time.Time obtained directly from time.Now() carries an internal
// monotonic reading. Sub()-ing two such values uses only that reading,
// making the result immune to the OS wall clock being changed afterward.
// A time.Time round-tripped through Postgres (like tickets.expires_at) has
// already lost that reading, so comparing it against a fresh time.Now()
// falls back to wall-clock-only arithmetic -- which does drift if the
// machine's system clock is changed, as it is in local dev where the
// backend runs on the same OS whose clock a developer might adjust.
package holdtimer

import (
	"sync"
	"time"
)

var (
	mu      sync.RWMutex
	anchors = map[int64]time.Time{}
)

// Set records the monotonic-capable time.Now() value marking when a hold
// for ticketID was created or renewed.
func Set(ticketID int64, at time.Time) {
	mu.Lock()
	anchors[ticketID] = at
	mu.Unlock()
}

// Clear removes the anchor for ticketID once its hold ends (cancelled,
// reclaimed as expired, or the ticket is sold).
func Clear(ticketID int64) {
	mu.Lock()
	delete(anchors, ticketID)
	mu.Unlock()
}

// SecondsRemaining returns how many seconds are left on ticketID's hold.
// Prefers the monotonic anchor (immune to wall-clock changes); falls back
// to wall-clock arithmetic against expiresAt when no anchor is present,
// e.g. right after a server restart wiped the in-memory map.
func SecondsRemaining(ticketID int64, expiresAt time.Time, ttl time.Duration) int64 {
	mu.RLock()
	anchor, ok := anchors[ticketID]
	mu.RUnlock()

	var remaining time.Duration
	if ok {
		remaining = ttl - time.Since(anchor)
	} else {
		remaining = time.Until(expiresAt)
	}
	if remaining < 0 {
		remaining = 0
	}
	return int64(remaining.Seconds())
}
