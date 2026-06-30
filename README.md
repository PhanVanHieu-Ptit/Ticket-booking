# High-Concurrency Concert Ticket Booking System

A production-ready, high-performance system designed to handle sudden traffic spikes (5,000 concurrent users) competing for a limited ticket inventory (500 tickets), ensuring **zero overselling**, strict **1-ticket-per-user limits**, and **real-time responsiveness**.

---

## Architectural Highlights

To protect the database and ensure sub-second response times under extreme load, this project implements a **Concurrency Shield** pattern:

```
                  ┌───────────────┐
                  │ User Browsers │
                  └───────┬───────┘
                          │ (HTTPS / SSE)
                          ▼
               ┌─────────────────────┐
               │ Ingress Load Balancer│
               └──────────┬──────────┘
                          │
               ┌──────────▼──────────┐
               │ Stateless API Nodes │
               └────┬───────────┬────┘
                    │           │
   (Atomic Lua pop) │           │ (Transaction Pool)
                    ▼           ▼
             ┌───────────┐ ┌───────────┐
             │   Redis   │ │ PgBouncer │
             │ (Shield)  │ └─────┬─────┘
             └───────────┘       │
                                 ▼
                           ┌───────────┐
                           │PostgreSQL │
                           │   (DB)    │
                           └───────────┘
```

1. **In-Memory Concurrency Shield (Redis)**: All reservation requests are validated and claimed atomically in Redis via Lua scripts before touching the relational database. This shields PostgreSQL from high lock contention and prevents double-booking.
2. **Connection Pooling (PgBouncer)**: Multiplexes database connections using transaction-level pooling, ensuring that the database does not exhaust its connection limit during peak traffic.
3. **Active Hold Expiration**: Implements a 5-minute ticket reservation TTL. If payment is not completed, the ticket is automatically reclaimed via Redis Keyspace Notifications (`expired` events) and restored to the available pool.
4. **Real-time Live Inventory (SSE)**: Streams real-time remaining ticket counts to the home page using Server-Sent Events (SSE).

---

## Quick Start

### 1. Start Local Infrastructure

Ensure you have Docker running, then start the database, PgBouncer, and Redis:

```bash
npm run infra:up
```

### 2. Install Dependencies

Install all Node.js workspace dependencies:

```bash
npm install
```

### 3. Run Development Servers

Start both the Go API server and the React frontend concurrently:

```bash
npm run dev
```

- **Frontend**: [http://localhost:3000](http://localhost:3000)
- **Backend API**: [http://localhost:8080](http://localhost:8080)

---

## Documentation Directory

For deep dives into the requirements, specifications, and architecture, refer to the following documents:

- [DEVELOPMENT.md](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/DEVELOPMENT.md) — Step-by-step local setup, folder explanation, and database configurations.
- [docs/00_REQUIREMENTS.md](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/docs/00_REQUIREMENTS.md) — Product requirements and scope.
- [docs/01_PROJECT_CONTEXT.md](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/docs/01_PROJECT_CONTEXT.md) — High-level project context and goals.
- [docs/02_PRODUCT_SPEC.md](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/docs/02_PRODUCT_SPEC.md) — Feature specifications and business rules.
- [docs/03_SYSTEM_ARCHITECTURE.md](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/docs/03_SYSTEM_ARCHITECTURE.md) — Detailed system design and flow diagrams.
- [docs/04_TECH_STACK.md](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/docs/04_TECH_STACK.md) — Technology stack analysis and trade-offs.
- [docs/05_DATABASE.md](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/docs/05_DATABASE.md) — Relational database schema and transaction strategy.
- [docs/06_API_SPEC.md](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/docs/06_API_SPEC.md) — Public API endpoint specifications.
