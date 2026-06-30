# Task: Manual Reservation Cancellation

## Task ID

`TS-07`

## Goal

Implement the manual cancellation flow. Create the backend endpoint to release the held ticket, clearing its status in PostgreSQL, deleting the Redis hold key, returning the ticket ID to the available pool, and broadcasting the update. Add a "Cancel Reservation" button on the frontend Checkout Page.

## Files Expected to Change

- [NEW] `internal/handlers/cancellation_handler.go` (implements the `POST /api/v1/tickets/hold/cancel` endpoint)
- [MODIFY] `internal/db/tickets.go` (PostgreSQL query to clear the ticket's session and reset status to `Available`)
- [MODIFY] `frontend/src/pages/CheckoutPage.tsx` (add a "Cancel Reservation" button and handle click event)
- [MODIFY] `main.go` (register the cancellation route)

## Dependencies

- `TS-05` (Requires backend reservation logic)
- `TS-06` (Requires the frontend Checkout Page)

## Acceptance Criteria

1. **Verification**: `POST /api/v1/tickets/hold/cancel` verifies that the active session owns a ticket currently in the `Holding` state. If not, return `400 Bad Request` with `NO_ACTIVE_HOLD`.
2. **Database Release**: The handler executes a database transaction to:
   - Lock the ticket row in PostgreSQL.
   - Update `status` to `Available`.
   - Set `session_id`, `held_at`, and `expires_at` to `NULL`.
3. **Redis Release**:
   - The handler deletes the Redis key `hold:{session_id}`.
   - The handler adds the `ticket_id` back to the corresponding Redis set (`tickets:available:VIP` or `tickets:available:Standard`).
4. **Real-time Broadcast**: The handler triggers the SSE broker to broadcast the incremented inventory count to all connected clients immediately.
5. **Frontend Flow**:
   - A "Cancel Reservation" or "Back to Home" button is displayed on the Checkout Page.
   - Clicking the button disables it, sends the request, and on success redirects the user back to the Event Home Page with an inline notification confirming the cancellation.
   - The user must be able to immediately initiate a new reservation on the Home Page.

## Estimated Complexity

Low (2 - 3 hours)
