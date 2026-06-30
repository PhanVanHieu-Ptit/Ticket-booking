# Inventory & Reservation Module

The **Inventory & Reservation Module** manages the ticket inventory (VIP and Standard categories) and coordinates the temporary hold (reservation) process. It acts as the primary gatekeeper of the system's inventory, utilizing an in-memory **Concurrency Shield** to handle high-concurrency spikes without bottlenecking the database.

---

## 1. Purpose

To manage ticket availability, display real-time counts, and process atomic ticket reservations (locks). It ensures that concurrent reservation attempts do not result in overselling or duplicate holds, and that a single user session can hold at most one ticket.

---

## 2. Responsibilities

- **Inventory Tracking**: Maintain real-time counts of available tickets in VIP (100 total) and Standard (400 total) categories.
- **Atomic Reservation (Locking)**: Execute an atomic "check-and-hold" operation using Redis Lua scripting to pop a ticket from the available pool and bind it to the user's session with a 5-minute (300 seconds) Time-To-Live (TTL).
- **Database Synchronization**: Persist the ticket hold in the PostgreSQL database once the in-memory hold is successfully established.
- **Hold Limit Enforcement**: Prevent a user session from initiating a reservation if they already have an active hold or a completed purchase.
- **Real-time Status Streaming**: Stream ticket availability updates (counts and sold-out status) to all connected clients in real-time via Server-Sent Events (SSE).
- **Manual Hold Cancellation**: Allow users to explicitly release their hold, returning the ticket to the available pool immediately.

---

## 3. Dependencies

- **[Session Management Module](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/docs/modules/session_management.md)**: Relies on this module to validate the user's `session_token` and extract the unique session ID.
- **[Hold Reclamation Module](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/docs/modules/hold_reclamation.md)**: Relies on this module to automatically release holds that exceed the 5-minute window.

---

## 4. APIs

### Public APIs

- **`GET /api/v1/tickets/availability`**: Retrieves the current counts of available tickets and prices for each category.
- **`GET /api/v1/tickets/availability/stream`**: Establishes a persistent Server-Sent Events (SSE) connection to receive real-time inventory updates.
- **`POST /api/v1/tickets/reserve`**: Temporarily locks one ticket of the requested category (`VIP` or `Standard`) for 5 minutes.
- **`GET /api/v1/tickets/hold`**: Retrieves details (including remaining seconds) of the active hold associated with the session.
- **`POST /api/v1/tickets/hold/cancel`**: Manually releases the active hold, returning the ticket to the available pool.

---

## 5. Data Ownership

### PostgreSQL

- **`tickets`**: Owns the ticket table.
  - Columns: `id`, `ticket_code`, `category`, `price`, `status` (`Available`, `Holding`, `Sold`), `session_id` (nullable), `held_at` (nullable), `expires_at` (nullable).
  - Constraints: State machine transitions, price validation, and the partial unique index `idx_tickets_session_id_unique` (ensuring a session is associated with at most one active hold or purchase).

### Redis

- **`tickets:available:VIP`**: A Set containing available VIP ticket IDs.
- **`tickets:available:Standard`**: A Set containing available Standard ticket IDs.
- **`hold:{session_id}`**: A String storing `{ticket_id}:{category}` with a 300-second TTL.
