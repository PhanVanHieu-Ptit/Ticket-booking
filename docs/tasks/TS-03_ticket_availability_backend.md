# Task: Real-Time Ticket Availability - Backend

## Task ID

`TS-03`

## Goal

Set up the Redis Concurrency Shield on backend startup. Implement the REST endpoint to retrieve current available counts and prices, and implement the Server-Sent Events (SSE) streaming endpoint to push real-time availability updates to connected clients.

## Files Expected to Change

- [NEW] `internal/redis/client.go` (Redis client initialization and startup inventory loading logic)
- [NEW] `internal/sse/broker.go` (in-memory SSE broker that manages client connections, heartbeats, and broadcasts updates)
- [NEW] `internal/handlers/availability_handler.go` (handlers for `GET /api/v1/tickets/availability` and `GET /api/v1/tickets/availability/stream`)
- [MODIFY] `main.go` (initialize Redis client, load inventory, start SSE broker, and register availability routes)

## Dependencies

- `TS-02` (Requires session validation middleware to ensure the SSE connection request contains a valid session token)

## Acceptance Criteria

1. **Startup Synchronization**: On application startup, the backend must query PostgreSQL for all tickets in the `Available` state and populate the corresponding Redis sets:
   - VIP ticket IDs are added to `tickets:available:VIP`.
   - Standard ticket IDs are added to `tickets:available:Standard`.
2. **REST Endpoint**: `GET /api/v1/tickets/availability` returns a `200 OK` response containing:
   - Ticket categories with their prices ($100 for VIP, $50 for Standard).
   - Accurate available counts retrieved directly from the Redis sets (e.g., via `SCARD`).
3. **SSE Connection**: `GET /api/v1/tickets/availability/stream` successfully establishes a persistent HTTP connection with headers:
   - `Content-Type: text/event-stream`
   - `Cache-Control: no-cache`
   - `Connection: keep-alive`
4. **SSE Broadcasting**: The SSE broker must be able to receive inventory update events and broadcast them as JSON payloads (e.g., `{"VIP": 100, "Standard": 400}`) to all connected clients within 500ms.
5. **Robustness**:
   - A heartbeat message (e.g., `: keep-alive`) is sent every 15 seconds to prevent connection timeouts.
   - The SSE connection requires a valid session token (passed via query parameter or cookie). If missing, return `400 Bad Request` with `INVALID_SESSION_TOKEN`.
   - Rate limiting is enforced on both endpoints.

## Estimated Complexity

Medium (4 - 5 hours)
