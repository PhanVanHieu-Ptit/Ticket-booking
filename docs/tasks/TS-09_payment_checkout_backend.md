# Task: Checkout & Payment - Backend

## Task ID

`TS-09`

## Goal

Implement the payment checkout endpoint `POST /api/v1/payments/checkout` to finalize ticket purchases. Use a database transaction with pessimistic locking to ensure the hold is still valid, integrate with a simulated payment gateway, record the order, register the session in the Redis purchase registry, and enforce Redis-backed idempotency.

## Files Expected to Change

- [NEW] `internal/handlers/checkout_handler.go` (implements the `POST /api/v1/payments/checkout` endpoint)
- [NEW] `internal/middleware/idempotency.go` (middleware to handle `Idempotency-Key` headers using Redis)
- [NEW] `internal/db/orders.go` (SQL operations to insert order records)
- [MODIFY] `main.go` (register the checkout route with the idempotency middleware)

## Dependencies

- `TS-05` (Requires an active hold)
- `TS-02` (Requires user session middleware)

## Acceptance Criteria

1. **Idempotency Enforcement**:
   - The endpoint requires the `Idempotency-Key` header (UUIDv4).
   - If the key exists in Redis with status `RESOLVED`, return the cached response immediately.
   - If the key exists with status `PENDING`, return `409 Conflict` (`DUPLICATE_REQUEST`).
   - If the key does not exist, set it to `PENDING` with a 120-second TTL and proceed.
2. **Hold Validation**: The handler starts a database transaction:
   - Locks the ticket row using `SELECT * FROM tickets WHERE id = ? FOR UPDATE`.
   - Verifies the ticket is in the `Holding` state, belongs to the active session, and that the current time is before `expires_at`.
   - If invalid or expired, abort the transaction and return `410 Gone` (`RESERVATION_EXPIRED`) or `403 Forbidden` (`SESSION_MISMATCH`).
3. **Simulated Payment**:
   - The handler simulates payment processing. If the request payload contains `simulate_status: "success"`, the payment succeeds. If `"fail"`, it fails.
   - If the payment fails, the transaction is rolled back, the Redis idempotency key is deleted, and the endpoint returns `402 Payment Required` (`PAYMENT_FAILED`). The ticket remains in the `Holding` state.
4. **Order Recording**: On successful payment:
   - Update the ticket status in PostgreSQL to `Sold`.
   - Insert a new row in the `orders` table (generating a unique order UUID and a cryptographically secure payment reference).
   - Commit the transaction.
5. **Concurrency Shield Update**:
   - Delete the Redis hold key `hold:{session_id}`.
   - Add the `session_id` to the Redis `purchased:sessions` set to enforce the 1-ticket-per-session limit.
   - Save the successful response in Redis under `idempotency:{session_id}:{key}` with status `RESOLVED` and a 1-hour TTL.
6. **Real-Time Broadcast**: Publish an event to the SSE broker to broadcast the updated sales metrics and sold-out status.

## Estimated Complexity

High (5 - 6 hours)
