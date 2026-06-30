# Task: Project & Database Setup

## Task ID

`TS-01`

## Goal

Initialize the project structure for both the Go backend and the React frontend. Set up the PostgreSQL database schema (with tables `tickets`, `orders`, and `admin_configs`), configure the database constraints, and seed the database with the initial inventory of 500 tickets.

## Files Expected to Change

- [NEW] `docker-compose.yml` (local PostgreSQL and Redis setup)
- [NEW] `Makefile` (commands for running migrations, seeds, and local servers)
- [NEW] `go.mod`
- [NEW] `go.sum`
- [NEW] `main.go` (basic server entry point)
- [NEW] `db/migrations/000001_init_schema.up.sql` (table definitions, constraints, and indexes)
- [NEW] `db/migrations/000001_init_schema.down.sql` (migration rollback)
- [NEW] `db/seeds/seed_tickets.sql` (seed script for 100 VIP and 400 Standard tickets)

## Dependencies

None

## Acceptance Criteria

1. **Containerized Environment**: Running `docker-compose up -d` successfully starts a PostgreSQL instance and a Redis instance locally.
2. **Database Schema**:
   - Table `tickets` is created with columns `id`, `ticket_code`, `category`, `price`, `status`, `session_id`, `held_at`, `expires_at`, `created_at`, `updated_at`.
   - Table `orders` is created with columns `id`, `ticket_id`, `session_id`, `amount`, `status`, `email`, `card_holder_name`, `payment_reference`, `created_at`, `updated_at`.
   - Table `admin_configs` is created with columns `key`, `value`, `description`, `updated_at`.
3. **Database Constraints**:
   - Ticket price must be non-negative (`price >= 0`).
   - Ticket status must be one of `Available`, `Holding`, `Sold`.
   - Ticket state consistency check is enforced (`chk_hold_dates`).
   - Order status must be one of `Paid`, `Refunded`, `Failed`.
4. **Database Indexes**:
   - Partial unique index `idx_tickets_session_id_unique` is created on `tickets(session_id) WHERE session_id IS NOT NULL`.
   - Partial index `idx_tickets_expires_at_holding` is created on `tickets(expires_at) WHERE status = 'Holding'`.
   - Composite index `idx_tickets_status_category` is created on `tickets(status, category)`.
5. **Seeding**:
   - Running the seed script populates the `tickets` table with exactly 500 rows:
     - 100 VIP tickets: category `VIP`, price `$100.00`, status `Available`, all session/timestamp fields `NULL`.
     - 400 Standard tickets: category `Standard`, price `$50.00`, status `Available`, all session/timestamp fields `NULL`.
   - The `admin_configs` table is seeded with a default BCrypt-hashed admin passcode (e.g., `admin123`).

## Estimated Complexity

Low (2 - 3 hours)
