package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Sentinel errors returned by RedisService.HoldTicket, mapped 1:1 to the
// existing business error codes the reservation API already returns.
var (
	ErrPurchaseLimitExceeded = errors.New("PURCHASE_LIMIT_EXCEEDED")
	ErrActiveHoldExists      = errors.New("ACTIVE_HOLD_EXISTS")
	ErrTicketUnavailable     = errors.New("TICKET_UNAVAILABLE")
)

// holdTicketScript atomically validates session eligibility (purchase limit,
// existing hold) and, if eligible, decrements the category's available pool
// by popping one ticket ID and writing a TTL-bound hold key. The whole check
// runs as a single Redis Lua script, which Redis executes atomically, so two
// concurrent callers can never pop the same ticket ID.
const holdTicketScript = `
local session_id = KEYS[1]
local category = ARGV[1]
local ttl = tonumber(ARGV[2])

local purchased_set = "purchased:sessions"
local hold_key = "hold:" .. session_id
local available_set = "tickets:available:" .. category

-- 1. Verify session does not already exist in purchased:sessions
if redis.call("SISMEMBER", purchased_set, session_id) == 1 then
    return { "ERR", "PURCHASE_LIMIT_EXCEEDED" }
end

-- 2. Verify session does not already have an active hold
if redis.call("EXISTS", hold_key) == 1 then
    return { "ERR", "ACTIVE_HOLD_EXISTS" }
end

-- 3. Verify there is at least one ticket ID in the set
local count = redis.call("SCARD", available_set)
if count == 0 then
    return { "ERR", "TICKET_UNAVAILABLE" }
end

-- 4. Pop ticket ID and set hold key
local ticket_id = redis.call("SPOP", available_set)
if not ticket_id then
    return { "ERR", "TICKET_UNAVAILABLE" }
end

redis.call("SET", hold_key, ticket_id .. ":" .. category, "EX", ttl)
return { "OK", ticket_id }
`

// RedisService encapsulates all atomic, Redis-backed ticket inventory
// operations used by the reservation flow. It is the sole owner of the
// "tickets:available:{category}", "hold:{sessionID}" and "purchased:sessions"
// key shapes.
type RedisService struct {
	rdb *redis.Client
}

// NewRedisService creates a new RedisService bound to the given Redis client.
func NewRedisService(rdb *redis.Client) *RedisService {
	return &RedisService{rdb: rdb}
}

// HoldTicket atomically checks purchase-limit/active-hold eligibility for the
// session and, if eligible, decrements the category's available pool by
// popping one ticket ID. Returns the popped ticket ID on success, or one of
// ErrPurchaseLimitExceeded / ErrActiveHoldExists / ErrTicketUnavailable if the
// hold could not be acquired. No database access happens before this call
// returns, so callers can fail fast without touching the DB.
func (s *RedisService) HoldTicket(ctx context.Context, sessionID, category string, ttl time.Duration) (int64, error) {
	res, err := s.rdb.Eval(ctx, holdTicketScript, []string{sessionID}, category, int(ttl.Seconds())).Slice()
	if err != nil {
		return 0, fmt.Errorf("redis hold script failed: %w", err)
	}
	if len(res) < 2 {
		return 0, errors.New("invalid response from redis hold script")
	}

	status, _ := res[0].(string)
	payload, _ := res[1].(string)

	if status == "ERR" {
		switch payload {
		case "PURCHASE_LIMIT_EXCEEDED":
			return 0, ErrPurchaseLimitExceeded
		case "ACTIVE_HOLD_EXISTS":
			return 0, ErrActiveHoldExists
		case "TICKET_UNAVAILABLE":
			return 0, ErrTicketUnavailable
		default:
			return 0, fmt.Errorf("redis hold script error: %s", payload)
		}
	}

	ticketID, err := strconv.ParseInt(payload, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse ticket id from redis: %w", err)
	}
	return ticketID, nil
}

// ReleaseHold reverses a previously acquired hold: it deletes the session's
// hold key and returns the ticket ID back to the category's available pool.
// Used both for explicit cancellation and for rolling back the Redis-side
// hold when the subsequent DB write fails.
func (s *RedisService) ReleaseHold(ctx context.Context, sessionID, category string, ticketID int64) error {
	pipe := s.rdb.Pipeline()
	pipe.Del(ctx, "hold:"+sessionID)
	pipe.SAdd(ctx, "tickets:available:"+category, ticketID)
	_, err := pipe.Exec(ctx)
	return err
}

// GetHoldTTL returns the remaining time-to-live of a session's active hold key.
func (s *RedisService) GetHoldTTL(ctx context.Context, sessionID string) (time.Duration, error) {
	return s.rdb.TTL(ctx, "hold:"+sessionID).Result()
}

// AvailableCount returns the number of tickets currently available in a category.
func (s *RedisService) AvailableCount(ctx context.Context, category string) (int64, error) {
	return s.rdb.SCard(ctx, "tickets:available:"+category).Result()
}
