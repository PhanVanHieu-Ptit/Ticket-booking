# Checkout & Payment Module

The **Checkout & Payment Module** is responsible for finalizing ticket purchases. It processes payments, creates order records, and permanently transitions ticket statuses from `Holding` to `Sold` in the database.

---

## 1. Purpose

To securely transition temporarily held tickets into permanently owned tickets upon successful payment. It enforces the lifetime purchase limit of one ticket per session and protects the payment process against duplicate submissions (double-charging and double-booking) using a Redis-backed idempotency mechanism.

---

## 2. Responsibilities

- **Payment Validation**: Ensure that the ticket being paid for is currently in the `Holding` state, belongs to the active session, and that the 5-minute hold window has not expired.
- **Simulated Payment Processing**: Integrate with a simulated payment gateway to authorize transactions.
- **Atomic Purchase Transition**: Execute a database transaction that:
  1. Locks the ticket row using pessimistic locking (`FOR UPDATE`).
  2. Transitions the ticket status from `Holding` to `Sold`.
  3. Inserts a new record into the `orders` table.
- **Purchase Registry**: Register the session in the Redis purchase registry (`purchased:sessions`) to enforce the **1-ticket-per-session** limit.
- **Idempotency Enforcement**: Intercept payment requests using an `Idempotency-Key` header and cache responses in Redis to prevent duplicate processing of the same transaction.
- **Inventory Update Notification**: Publish an event to Redis Pub/Sub indicating that a ticket has been sold, triggering a broadcast to all active clients.

---

## 3. Dependencies

- **[Session Management Module](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/docs/modules/session_management.md)**: Relies on this module to validate the user session.
- **[Inventory & Reservation Module](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/docs/modules/inventory_reservation.md)**: Relies on this module to verify the active hold state, lock the ticket row, and transition its status.

---

## 4. APIs

### Public APIs

- **`POST /api/v1/payments/checkout`**: Finalizes the purchase of a held ticket.
  - _Headers_: `Idempotency-Key: <UUIDv4>` (Required)
  - _Request Body_:
    ```json
    {
      "ticket_id": 105,
      "email": "buyer@example.com",
      "card_holder_name": "John Doe",
      "payment_method": "simulated",
      "simulate_status": "success"
    }
    ```
  - _Response_: Returns the details of the created order and transaction reference.

---

## 5. Data Ownership

### PostgreSQL

- **`orders`**: Owns the orders table.
  - Columns: `id` (UUID), `ticket_id` (foreign key), `session_id`, `amount`, `status` (`Paid`, `Refunded`, `Failed`), `email`, `card_holder_name`, `payment_reference` (unique), `created_at`, `updated_at`.
  - Constraints: Unique constraint on `session_id` (enforces 1 purchase per session) and unique constraint on `ticket_id` (prevents multiple orders for a single ticket).

### Redis

- **`purchased:sessions`**: A Set containing session IDs that have completed a purchase. Used for O(1) checks during the reservation phase.
- **`idempotency:{session_id}:{idempotency_key}`**: A String storing the status (`PENDING` or `RESOLVED`) and the cached response payload of a transaction, with a TTL (e.g., 2 hours).
