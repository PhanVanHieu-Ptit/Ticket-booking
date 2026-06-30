# Task: Atomic Ticket Reservation - Backend

## Task ID

`TS-05`

## Goal

Implement the atomic check-and-hold reservation logic to handle high-concurrency spikes. Create a Redis Lua script to perform the reservation check and pop operations atomically in memory, and create the `POST /api/v1/tickets/reserve` endpoint to execute the script and synchronize the reservation to PostgreSQL, with rollback logic on database failure.

## Files Expected to Change

- [NEW] `internal/redis/lua/reserve.lua` (atomic check-and-hold Lua script)
- [NEW] `internal/handlers/reservation_handler.go` (implements the `POST /api/v1/tickets/reserve` endpoint and `GET /api/v1/tickets/hold` endpoint)
- [NEW] `internal/db/tickets.go` (PostgreSQL query to update ticket status to `Holding` and set session/timestamps)
- [MODIFY] `main.go` (register the reservation routes)

## Dependencies

- `TS-03` (Requires Redis client setup and populated availability sets)

## Acceptance Criteria

1. **Atomic Check-and-Hold (Redis)**: The Redis Lua script must execute atomically and perform the following checks:
   - Verify that the session does not already exist in the `purchased:sessions` set (limit 1 ticket purchase).
   - Verify that the session does not already have a key `hold:{session_id}` (limit 1 active hold).
   - Verify that there is at least one ticket ID in the `tickets:available:{category}` set.
   - If all checks pass, it pops a ticket ID, sets `hold:{session_id}` to `{ticket_id}:{category}` with a 300-second TTL, and returns success with the ticket ID.
2. **Database Synchronization**: On receiving success from Redis, the handler starts a PostgreSQL transaction to:
   - Lock the ticket row using `SELECT ... FOR UPDATE` (or update directly with status check).
   - Update the ticket status to `Holding`, set `session_id`, `held_at = NOW()`, and `expires_at = NOW() + 5 minutes`.
   - Commit the transaction.
3. **Database Write Failure Rollback**: If the database update fails or times out, the handler must catch the exception, immediately execute a rollback in Redis (delete `hold:{session_id}` and add the ticket ID back to `tickets:available:{category}`), and return `500 Internal Server Error` with `RESERVATION_FAILED`.
4. **Validation and Error Codes**:
   - Validate that `category` is exactly `VIP` or `Standard` (return `400 Bad Request` with `INVALID_CATEGORY`).
   - If the session already has an active hold, return `400 Bad Request` with `ACTIVE_HOLD_EXISTS`.
   - If the session has already purchased a ticket, return `400 Bad Request` with `PURCHASE_LIMIT_EXCEEDED`.
   - If the category is sold out, return `409 Conflict` with `TICKET_UNAVAILABLE`.
5. **Real-time Event**: On successful reservation, publish an update event to the SSE broker to broadcast the new available counts.
6. **Active Hold Query**: `GET /api/v1/tickets/hold` returns details of the active hold (ticket ID, category, remaining seconds) for the current session.

## Estimated Complexity

High (5 - 6 hours)
