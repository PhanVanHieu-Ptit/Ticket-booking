# AI Context: Concert Ticket Booking System

This file serves as the central context and progress tracker for AI assistants working on the Concert Ticket Booking System. It outlines the project structure, current implementation progress, technical decisions, known issues, and next steps.

---

## 1. Project Overview & Architecture

The system is a high-concurrency ticket booking application designed to handle a sudden spike of **5,000 concurrent users** competing for **500 tickets** (100 VIP, 400 Standard). It uses a **dual-database design**:
1. **PostgreSQL**: The persistent relational database serving as the absolute source of truth for ticket ownership, financial audits, and system configuration.
2. **Redis**: The in-memory **Concurrency Shield** that intercepts write spikes, manages temporary 5-minute ticket holds, and enforces the 1-ticket-per-session limit.

### Project Structure
```
.
├── apps/
│   ├── backend/             # Go (Gin) API Server
│   │   ├── cmd/api/         # Entry point (main.go)
│   │   ├── db/              # Migrations and Seeds
│   │   └── internal/shared/ # Shared utilities (logger, validator, db, config)
│   └── frontend/            # React (Vite + TS + Tailwind) client application
├── docs/                    # Architecture, specs, and task definitions
└── docker/                  # Docker configuration (PgBouncer, etc.)
```

---

## 2. Technology Stack

- **Backend**: Go 1.21+ with the **Gin** web framework.
- **Database Driver**: `github.com/jackc/pgx/v5` (using `pgx/v5/stdlib` for `database/sql` compatibility).
- **Frontend**: React (Vite, TypeScript, TailwindCSS).
- **Infrastructure**: PostgreSQL, Redis, PgBouncer (in transaction mode).

---

## 3. Implementation Progress

### Completed Tasks

- [x] **TS-01: Project & Database Setup**
  - Initialized monorepo structure for Go backend and React frontend.
  - Set up containerized PostgreSQL and Redis.
  - Configured PgBouncer in transaction mode.
  - Created database schema migrations and seed scripts for 500 tickets (100 VIP, 400 Standard) and default admin passcode.
  - Implemented backend foundation layer (structured logging, custom error codes, request validation, environment configuration).
  - Integrated `pgx/v5` with simple protocol mode to ensure PgBouncer transaction mode compatibility.
  - Added performance index on `orders(email)`.
- [x] **TS-03: Ticket Availability Backend**
  - Redis integration for real-time ticket availability counts.
  - Server-Sent Events (SSE) endpoint to stream inventory updates.

### Pending Tasks

- [/] **TS-02: Session Management** (Prerequisite for all booking actions)
  - [x] Backend: Stateless JWT session cookie generation and verification middleware.
  - [ ] Frontend: Automatically request and store the session token on initial load.

- [ ] **TS-04: Ticket Availability Frontend**
  - React integration with the SSE endpoint to display live, real-time ticket counts.
- [ ] **TS-05: Ticket Reservation Backend**
  - Redis Lua script for atomic ticket popping and reservation.
  - PostgreSQL transaction to sync the reservation (`Holding` state, 5-minute TTL).
- [ ] **TS-06: Ticket Reservation Frontend**
  - Booking page UI with seat category selection.
  - Debounced reservation button and 5-minute countdown timer.
- [ ] **TS-07: Manual Cancellation**
  - API and UI to allow users to release their ticket hold before the 5-minute expiration.
- [ ] **TS-08: Hold Reclamation**
  - Background worker in Go to poll and reclaim expired ticket holds in PostgreSQL if Redis notifications fail.
- [ ] **TS-09: Payment Checkout Backend**
  - Mock payment gateway integration.
  - Atomic database transaction to mark ticket as `Sold` and create an order record.
- [ ] **TS-10: Payment Checkout Frontend**
  - Checkout form and payment simulation.
  - Success/failure feedback screens.
- [ ] **TS-11: Admin Analytics**
  - Admin login page.
  - Dashboard displaying real-time sales velocity, total revenue, and active locks.

---

## 4. Known Issues & Workarounds

- **Ticket Lockout on Failed Payments**: 
  - *Status*: **RESOLVED**
  - *Details*: The initial schema had a `UNIQUE` constraint on `orders.ticket_id`. If an order failed, the ticket could never be ordered again. This was resolved by removing the `UNIQUE` constraint on `orders.ticket_id`, allowing a ticket to have multiple order records (e.g., a `Failed` order followed by a `Paid` order).
- **PgBouncer Transaction Mode Incompatibility**:
  - *Status*: **RESOLVED**
  - *Details*: PgBouncer in transaction mode does not support prepared statements. Using standard Go SQL drivers resulted in prepared statement errors. This was resolved by migrating to the `pgx/v5` driver and configuring `cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol` in the database initialization.

---

## 5. Technical Debt

- **Global DB Variable**: Currently, `internal/shared/db/db.go` exposes a global `DB *sql.DB`. For better unit testing and modularity, we should transition to a dependency-injection pattern (passing database handles or repositories to service/handler structs) as we start implementing business logic.
- **Redis Integration**: Redis connection pooling and client setup have been initialized in `TS-03`. Remaining Lua scripts and transactional synchronizations will be handled in `TS-05`.

---

## 6. Next Recommended Task

### **`TS-04: Ticket Availability Frontend`**
- **Objective**: Connect the React frontend application to the SSE availability stream.
- **Why**: Enables reactive, real-time ticket availability count display on the event home page.
- **Steps**:
  1. Build `useTicketAvailability` custom hook with EventSource connection and visibility listener.
  2. Implement `TicketCategoryCard` component and home page layout.
  3. Validate real-time inventory counter updates.

