# 🎫 High-Concurrency Concert Ticket Booking System

A production-grade booking platform that lets **5,000 concurrent users** compete for **500 tickets** with **zero overselling**, a strict **1-ticket-per-session limit**, and real-time inventory updates.

This README is the **single entry point** for the project — for humans and AI coding agents alike. Deep dives live under [`docs/`](docs/); this file tells you what exists and where to find it.

---

## 1. Project Overview

| | |
| --- | --- |
| **What it does** | Sells a fixed inventory of event tickets (VIP / Standard) under extreme traffic spikes without double-selling or losing sales to race conditions. |
| **Target users** | End customers booking a ticket during a flash on-sale; event-organizer admins monitoring sales velocity and revenue in real time. |
| **Business value** | Prevents lost revenue from overselling/refund disputes, keeps checkout responsive under load, and gives organizers live visibility into sell-through. |

**Key features**

- ⚡ **Real-time live inventory** — remaining ticket counts streamed to the browser via Server-Sent Events (SSE), no polling.
- 🔒 **Atomic reservation** — ticket claims are popped atomically in Redis via a Lua script before touching Postgres, eliminating double-booking under concurrent load.
- ⏱️ **5-minute hold + auto-reclamation** — unpaid holds expire and are automatically returned to the pool via Redis keyspace notifications, with a periodic sweeper as a fail-safe.
- 💳 **Idempotent checkout** — mock payment gateway with Redis-backed request idempotency and an atomic DB transaction to mark a ticket `Sold`.
- 📊 **Admin analytics dashboard** — live sales velocity, revenue, and active-lock counts, protected by admin-only JWT auth.
- 🔑 **Dual JWT auth** — short-lived user session cookies and a separate admin token/JWT flow.

---

## 2. Screenshots

> Screenshots are not yet checked into the repo. Placeholders below map to the real pages — replace with actual captures under `docs/screenshots/` when available.

| Page | Preview |
| --- | --- |
| Home — Live Ticket Availability | <img width="1916" height="932" alt="Screenshot 2026-07-03 at 21 49 16" src="https://github.com/user-attachments/assets/bf6cdca4-b397-4da2-a951-6e6da9ceeb19" /> |
| Booking / Reservation Hold | <img width="1916" height="934" alt="Screenshot 2026-07-03 at 21 49 44" src="https://github.com/user-attachments/assets/78b6482d-e9d4-43f5-ad1f-b9304abf4a7e" />|
| Checkout / Payment |<img width="1917" height="925" alt="Screenshot 2026-07-03 at 21 51 35" src="https://github.com/user-attachments/assets/121f01f1-5309-49ca-84c7-1b19d0adbe32" /> |
| Admin Analytics Dashboard |<img width="1918" height="928" alt="Screenshot 2026-07-03 at 21 53 36" src="https://github.com/user-attachments/assets/53ab2220-bb50-4733-a0fc-db42a67c343d" />|

---

## 3. Tech Stack

| Layer | Technology | Notes |
| --- | --- | --- |
| Backend framework | Go 1.25 + [Gin](https://github.com/gin-gonic/gin) | REST API, SSE endpoints, background workers, all in one process |
| Frontend framework | React 18 + [Vite](https://vitejs.dev) 5 + TypeScript | SPA, TailwindCSS for styling, `react-router-dom` v7 for routing |
| Database | PostgreSQL (behind PgBouncer, transaction pooling) | Source of truth for tickets/orders/config |
| Data access | [`pgx/v5`](https://github.com/jackc/pgx) with plain SQL migrations | **No ORM** — versioned `.sql` files in `apps/backend/db/migrations/` |
| Caching / concurrency shield | Redis (Lua scripts + keyspace notifications) | Atomic reservation, hold TTL, hold reclamation |
| Authentication | Custom JWT (`golang-jwt/jwt/v5`) | Separate short-lived user-session and admin tokens — no third-party auth provider |
| AI | Not used | This project has no AI/LLM integration |
| State management | React hooks (`useState`), per feature module | No Redux/Zustand/Query library — state is local to each `*.state.ts` |
| Analytics / Monitoring | Not integrated | No Sentry/PostHog/Vercel Analytics currently wired in |
| Real-time transport | Server-Sent Events (SSE) | Custom broker (`apps/backend/internal/sse`), consumed via React hooks |

Full rationale and trade-offs: [`docs/04_TECH_STACK.md`](docs/04_TECH_STACK.md).

---

## 4. Architecture Overview

The system implements a **Concurrency Shield** pattern so that write spikes never hit PostgreSQL directly:

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

1. **Redis Concurrency Shield** — reservation requests are validated and claimed atomically via Lua scripts before any Postgres write.
2. **PgBouncer (transaction pooling)** — multiplexes DB connections so the connection limit isn't exhausted under load.
3. **Active hold expiration** — a 5-minute hold TTL; expired holds are reclaimed via Redis keyspace notifications (with a cron sweeper fail-safe).
4. **SSE inventory streaming** — remaining ticket counts pushed live to clients.

For the full design and request flows, see:
- [`docs/03_SYSTEM_ARCHITECTURE.md`](docs/03_SYSTEM_ARCHITECTURE.md) — system design and flow diagrams (closest equivalent to an `ARCHITECTURE.md`).
- [`docs/01_PROJECT_CONTEXT.md`](docs/01_PROJECT_CONTEXT.md) — high-level project context and goals (closest equivalent to a `PROJECT_CONTEXT.md`).

---

## 5. Project Structure

This is an npm-workspaces monorepo — there is no top-level `src/`/`app/` (Next.js-style); the two apps each have their own source tree.

```
.
├── apps/
│   ├── backend/            # Go (Gin) API server
│   └── frontend/           # React (Vite) SPA
├── docker/                 # PgBouncer config for local infra
├── docs/                   # Architecture, specs, feature/module/task docs
├── e2e/                    # Playwright end-to-end tests
├── docker-compose.yml      # Postgres + PgBouncer + Redis for local dev
├── Makefile                # DB migration/seed shortcuts
├── render.yaml             # Backend deployment (Render)
└── vercel.json             # Frontend deployment (Vercel)
```

| Folder | Responsibility |
| --- | --- |
| `apps/backend/cmd/` | Entrypoints (`api/main.go` — server; `seedredis/main.go` — Redis seeding) |
| `apps/backend/db/` | SQL migrations and seed scripts |
| `apps/backend/internal/handlers/` | HTTP request handlers per domain (admin, availability, cancellation, reservation, session) |
| `apps/backend/internal/middleware/` | Cross-cutting concerns: admin auth, error handling, idempotency, session |
| `apps/backend/internal/modules/` | Business logic per domain (controller/dto/entity/repository/service layers) |
| `apps/backend/internal/redis/` | Redis client, service, and Lua scripts |
| `apps/backend/internal/session/` | JWT signing/verification |
| `apps/backend/internal/shared/` | Config, DB connection, logger, validator, shared types/utils |
| `apps/backend/internal/sse/` | Server-Sent Events broker |
| `apps/frontend/src/components/` | Reusable UI components |
| `apps/frontend/src/hooks/` | Custom hooks (SSE listeners: `useTicketAvailability`, `useAdminRealtime`) |
| `apps/frontend/src/lib/` | Frontend helper libraries (API base config, utils) |
| `apps/frontend/src/modules/` | Feature modules — `admin/`, `booking/`, `checkout/`, each with its own `.page.tsx`, `.api.ts`, `.state.ts`, `.routes.tsx` |
| `docs/` | Numbered spec docs (`00`–`07`), plus `features/`, `modules/`, `tasks/` subfolders — see [Documentation](#9-documentation) below |

---

## 6. Local Development

### Prerequisites

- Node.js v18+ and npm v10+
- Go 1.25+
- Docker & Docker Compose

### Installation

```bash
npm install
```

### Environment Variables

```bash
cp apps/backend/.env.example apps/backend/.env
cp apps/frontend/.env.example apps/frontend/.env
```

Defaults are pre-wired for the local Docker infra. Full variable reference: [section 8](#8-environment-variables).

### Database Setup

```bash
npm run infra:up        # starts Postgres, PgBouncer, Redis
make migrate-up         # applies SQL migrations
make seed               # seeds 500 tickets + default admin config
```

### Running Locally

```bash
npm run dev
```

- Frontend: http://localhost:3000
- Backend API: http://localhost:8080

Full walkthrough (including PgBouncer/Redis details and Lighthouse testing notes): [`DEVELOPMENT.md`](DEVELOPMENT.md).

---

## 7. Available Commands

| Command | Description |
| --- | --- |
| `npm run dev` | Starts both the Go backend and Vite frontend concurrently |
| `npm run dev:frontend` | Starts the Vite frontend dev server only |
| `npm run dev:backend` | Starts the Go backend server only |
| `npm run build:frontend` | Builds the frontend production bundle |
| `npm run preview:frontend` | Builds and serves the production frontend bundle locally |
| `npm run infra:up` | Starts Postgres, PgBouncer, and Redis containers |
| `npm run infra:down` | Stops and removes local infrastructure containers |
| `npm run infra:logs` | Tails local infrastructure container logs |
| `npm run db:reset-tickets` | Resets ticket inventory to its seeded state |
| `npm run db:seed-low-stock` | Seeds a low-stock scenario for a ticket category |
| `npm run lint` | Runs ESLint across the workspace |
| `npm run format` | Formats all source files with Prettier |
| `npm run format:check` | Checks formatting without writing changes |
| `npm run test:e2e` | Runs Playwright end-to-end tests |
| `npm run test:e2e:ci` | Runs Playwright tests with CI reporters |

Database migrations/seeding also have `Makefile` shortcuts: `make migrate-up`, `make migrate-down`, `make seed`, `make db-reset`.

---

## 8. Environment Variables

| Variable | Used by | Purpose |
| --- | --- | --- |
| `VITE_API_BASE_URL` | Frontend | Backend API base URL (empty locally — uses Vite proxy) |
| `PORT` | Backend | HTTP port for the Go server |
| `APP_ENV` | Backend | Environment name (`local`, `production`, ...) |
| `GIN_MODE` | Backend | Gin framework mode (`debug`/`release`) |
| `DATABASE_URL` | Backend | Postgres connection string, via **PgBouncer** |
| `DIRECT_DATABASE_URL` | Backend | Direct Postgres connection, used for migrations (bypasses PgBouncer) |
| `REDIS_URL` | Backend | Redis connection string |
| `ADMIN_TOKEN` | Backend | Passcode used to obtain an admin JWT |
| `JWT_SECRET` | Backend | Signing secret for user/admin session JWTs |
| `CORS_ALLOWED_ORIGINS` | Backend | Allowed cross-origin frontend URL (also toggles cookie `SameSite` mode) |

Full context: [`docs/01_PROJECT_CONTEXT.md`](docs/01_PROJECT_CONTEXT.md) and the `.env.example` files in `apps/backend/` and `apps/frontend/`.

---

## 9. Documentation

| File | Purpose |
| --- | --- |
| [`DEVELOPMENT.md`](DEVELOPMENT.md) | Full local setup guide, folder walkthrough, PgBouncer/Redis details, deployment steps |
| [`AI_CONTEXT.md`](AI_CONTEXT.md) | AI-assistant progress tracker: architecture summary, completed tasks, known issues, technical debt, next steps |
| [`docs/00_REQUIREMENTS.md`](docs/00_REQUIREMENTS.md) | Product requirements and scope |
| [`docs/01_PROJECT_CONTEXT.md`](docs/01_PROJECT_CONTEXT.md) | High-level project context and goals |
| [`docs/02_PRODUCT_SPEC.md`](docs/02_PRODUCT_SPEC.md) | Feature specs and business rules |
| [`docs/03_SYSTEM_ARCHITECTURE.md`](docs/03_SYSTEM_ARCHITECTURE.md) | System design and flow diagrams |
| [`docs/04_TECH_STACK.md`](docs/04_TECH_STACK.md) | Technology stack analysis and trade-offs |
| [`docs/05_DATABASE.md`](docs/05_DATABASE.md) | Database schema and transaction strategy |
| [`docs/06_API_SPEC.md`](docs/06_API_SPEC.md) | Public API endpoint specifications |
| [`docs/07_INFRASTRUCTURE_DESIGN.md`](docs/07_INFRASTRUCTURE_DESIGN.md) | Infra design (Redis, PgBouncer, deployment topology) |
| [`docs/features/atomic_ticket_reservation.md`](docs/features/atomic_ticket_reservation.md) | Atomic reservation feature deep dive |
| [`docs/features/manual_reservation_cancellation.md`](docs/features/manual_reservation_cancellation.md) | Manual cancellation feature deep dive |
| [`docs/features/ticket_availability_streaming.md`](docs/features/ticket_availability_streaming.md) | SSE inventory streaming feature deep dive |
| [`docs/modules/administration_analytics.md`](docs/modules/administration_analytics.md) | Admin analytics module |
| [`docs/modules/checkout_payment.md`](docs/modules/checkout_payment.md) | Checkout/payment module |
| [`docs/modules/hold_reclamation.md`](docs/modules/hold_reclamation.md) | Hold reclamation module |
| [`docs/modules/inventory_reservation.md`](docs/modules/inventory_reservation.md) | Inventory/reservation module |
| [`docs/modules/session_management.md`](docs/modules/session_management.md) | Session management module |
| [`docs/tasks/TS-01`](docs/tasks/TS-01_database_setup.md) … [`TS-11`](docs/tasks/TS-11_admin_analytics.md) | One file per implementation milestone, DB setup through admin analytics |
| [`e2e/README.md`](e2e/README.md) | Playwright end-to-end test suite guide |

> This repo does not yet split `AI_CONTEXT.md` into separate `DECISIONS.md` / `SESSION_SUMMARY.md` / `KNOWN_ISSUES.md` files — that content currently lives in `AI_CONTEXT.md`'s "Known Issues", "Technical Debt", and "Next Recommended Task" sections.

---

## 10. AI Agent Workflow

**Before coding:**

1. Read this `README.md`.
2. Read [`docs/01_PROJECT_CONTEXT.md`](docs/01_PROJECT_CONTEXT.md).
3. Read [`docs/03_SYSTEM_ARCHITECTURE.md`](docs/03_SYSTEM_ARCHITECTURE.md).
4. Read [`AI_CONTEXT.md`](AI_CONTEXT.md) (progress, known issues, technical debt).
5. Read the relevant `docs/modules/*.md` or `docs/features/*.md` for the area you're touching.

**After coding:**

1. Update `AI_CONTEXT.md`'s progress section with what changed.
2. Update `AI_CONTEXT.md`'s "Known Issues & Workarounds" if architecture changed or a new issue surfaced.
3. Add a new `docs/tasks/TS-NN_*.md` file if the change constitutes a new milestone.

---

## 11. Deployment

The backend (Postgres + Redis + long-lived SSE connections + cron workers, one Go process) cannot run on serverless/edge platforms — it deploys separately from the frontend.

| Component | Platform | Config |
| --- | --- | --- |
| Database | [Neon](https://neon.tech) | Pooled connection string → `DATABASE_URL`; direct/unpooled → `DIRECT_DATABASE_URL` (`sslmode=require`) |
| Redis | [Upstash](https://upstash.com) | TLS connection string (`rediss://`) → `REDIS_URL` |
| Backend | [Render](https://render.com) | [`render.yaml`](render.yaml) (Blueprint: Docker web service via [`apps/backend/Dockerfile`](apps/backend/Dockerfile); DB/Redis are external) |
| Frontend | [Vercel](https://vercel.com) | [`vercel.json`](vercel.json) (`npm run build:frontend` → `apps/frontend/dist`) |
| DB Migrations | Manual via `make migrate-up` | Run against Neon's `DIRECT_DATABASE_URL` (bypasses pooler — required for DDL) |

Set `DATABASE_URL`, `DIRECT_DATABASE_URL`, `REDIS_URL`, `JWT_SECRET`, and `ADMIN_TOKEN` on the Render service (all marked `sync: false`, manual entry). Set `VITE_API_BASE_URL` on Vercel to the Render backend URL, and `CORS_ALLOWED_ORIGINS` on Render to the Vercel frontend URL so credentialed cookies work cross-origin.

Full step-by-step: [`DEVELOPMENT.md` § Deployment](DEVELOPMENT.md#deployment).

---

## 12. Contributing

1. Create a feature branch off `main`.
2. Follow existing module patterns: backend changes go through `internal/modules/<domain>/{controller,dto,entity,repository,service}`; frontend features live under `src/modules/<feature>/`.
3. Before opening a PR, run:
   ```bash
   npm run lint
   npm run format:check
   npm run test:e2e
   ```
4. Update `AI_CONTEXT.md` if you touch architecture or discover a new known issue (see [AI Agent Workflow](#10-ai-agent-workflow)).
5. Keep PRs scoped to one module/feature where possible.

---

## 13. Author

**Phan Văn Hiểu**

---

## 14. License

MIT — see `LICENSE` *(placeholder — no LICENSE file currently checked into this repository)*.
