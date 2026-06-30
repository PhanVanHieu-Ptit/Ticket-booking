# Task: Hold Reclamation (Background Worker)

## Task ID

`TS-08`

## Goal

Implement an internal, event-driven background worker to reclaim expired ticket holds. This ensures that tickets are not locked indefinitely if a user abandons their checkout. Implement two parallel mechanisms: a Redis keyspace notification listener for low-latency reclamation, and a periodic database polling cron job as a fail-safe.

## Files Expected to Change

- [NEW] `internal/worker/reclaimer.go` (Redis expired keyspace event listener)
- [NEW] `internal/worker/cron.go` (periodic database polling job)
- [MODIFY] `internal/redis/client.go` (configure Redis to enable keyspace notifications: `CONFIG SET notify-keyspace-events Ex`)
- [MODIFY] `main.go` (start the background worker and cron job on startup)

## Dependencies

- `TS-05` (Requires the reservation logic to create the Redis keys and PostgreSQL records)

## Acceptance Criteria

1. **Redis Keyspace Listener**:
   - The background worker subscribes to Redis keyspace events for key expirations (`__keyevent@0__:expired`).
   - When a `hold:{session_id}` key expires, the worker receives the event, extracts the `session_id`, and initiates a database transaction.
2. **Database Reclamation**: The transaction must:
   - Lock the ticket row associated with the `session_id` using `SELECT ... FOR UPDATE`.
   - Verify the ticket is still in the `Holding` state.
   - Update the status to `Available` and clear all session/hold timestamps.
   - If the database write succeeds, add the `ticket_id` back to the corresponding Redis set (`tickets:available:{category}`).
3. **Fail-Safe Polling**:
   - A background cron job runs every 10 seconds.
   - It queries PostgreSQL for any tickets in the `Holding` state where `expires_at < NOW()`.
   - For each expired ticket, it performs the same database release and Redis restoration, acting as a backup in case of Redis event delivery failure.
4. **Concurrency Safety**: If the keyspace listener and the cron job attempt to reclaim the same ticket simultaneously, the database row lock must prevent race conditions. The second worker to acquire the lock must see the status is already `Available` and exit gracefully without double-incrementing.
5. **Real-Time Broadcast**: Any successful reclamation must trigger the SSE broker to broadcast the updated inventory count.

## Estimated Complexity

High (5 - 6 hours)
