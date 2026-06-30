# Infrastructure Layer Design: High-Concurrency Ticket Booking System

This document outlines the architectural design for the **Infrastructure Layer** of the Go backend. In accordance with Clean Architecture / Hexagonal Architecture principles, the infrastructure layer provides concrete implementations for the interfaces defined in the Domain and Application layers. 

To support **5,000 concurrent users** competing for **500 tickets** under strict ACID requirements, this design guarantees sub-millisecond latencies, atomic reservation controls, and robust fault tolerance.

---

## 1. Core Infrastructure Components

### 1.1. Database
*   **Technology**: PostgreSQL 16+ (Amazon RDS) with `PgBouncer` for connection pooling.
*   **Client Driver**: `jackc/pgx/v5` (high-performance, PostgreSQL-specific connection pool and driver).
*   **Query Generation**: `sqlc` (compiles raw SQL to type-safe Go code, avoiding runtime ORM reflection overhead).
*   **Transaction Management**:
    *   **Unit of Work / Transaction Manager**: A database transaction manager (`TxManager`) will be designed to handle multi-repository operations atomically.
    *   It exposes a `WithTx(ctx, fn)` method that automatically begins, commits, or rolls back a transaction based on the error returned by the execution function `fn`.
*   **Database Migrations**: Managed via `golang-migrate/migrate/v4` or run out-of-band via CI/CD.

### 1.2. Cache
*   **Technology**: Redis 7.2+ (Amazon ElastiCache).
*   **Client Driver**: `go-redis/redis/v9`.
*   **Concurrency Shield Coordination**:
    *   Encapsulates the execution of the atomic **Lua Reservation Script** (detailed in [05_DATABASE.md](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/docs/05_DATABASE.md#L214-L251)).
    *   Handles connection pool tuning (e.g., setting optimal `PoolSize`, `MinIdleConns`, and timeouts).
    *   Implements the `LockRepository` interface for temporary hold management.

### 1.3. Queue
*   **Technology**: Redis Pub/Sub & Redis Streams.
*   **Event Distribution (Pub/Sub)**:
    *   Used as the message backplane to broadcast inventory changes across multiple stateless API nodes.
    *   API nodes publish events to the `inventory:change` channel.
*   **Asynchronous Task Queue (Streams)**:
    *   For background tasks that require guaranteed delivery (e.g., sending ticket confirmation emails, processing refunds).
    *   Managed via a lightweight worker pool or `hibiken/asynq` (a Go library for queueing jobs backed by Redis).

### 1.4. Authentication
*   **Mechanism**: Cryptographic Session Tokens and Passcode Verification.
*   **Session Tokens**:
    *   Issued upon initial landing page load.
    *   Signed JWT (JSON Web Tokens) containing `session_id` and `exp` (expiration).
    *   Stored in an `HttpOnly`, `Secure`, `SameSite=Strict` cookie.
*   **Admin Authentication**:
    *   Passcode-based authentication using `golang.org/x/crypto/bcrypt`.
    *   Admin API requests must include an `Authorization: Bearer <Admin_Token>` header.
*   **Implementation**: An `Authenticator` component in the infrastructure layer handles token generation, signing, and verification.

### 1.5. Authorization
*   **Mechanism**: Role-Based Access Control (RBAC) Middleware.
*   **Roles**:
    *   `SessionUser`: Authorized to reserve and purchase a single ticket. Must possess a valid session token.
    *   `Admin`: Authorized to access `/api/admin/*` endpoints. Must possess a valid admin token.
*   **Implementation**: Gin HTTP middleware interceptors that decode the authentication context and enforce role validation.

### 1.6. Realtime
*   **Technology**: Server-Sent Events (SSE) over HTTP/2.
*   **Mechanism**:
    *   An `SSEBroker` manages client connections (`text/event-stream`).
    *   It maintains a registry of active client channels.
    *   A background goroutine subscribes to the Redis Pub/Sub `inventory:change` channel.
    *   When an event is received, the broker writes the payload to all registered client channels.
    *   Automatically handles client disconnects and cleans up channel memory.

### 1.7. File Storage
*   **Technology**: Abstract Object Storage.
*   **Implementations**:
    *   `S3Storage`: Integrates with AWS S3 / Google Cloud Storage using `aws-sdk-go-v2`.
    *   `LocalStorage`: A local directory-based implementation for development and testing.
*   **Interface**:
    ```go
    type Storage interface {
        Upload(ctx context.Context, key string, data []byte) error
        GetSignedURL(ctx context.Context, key string, ttl time.Duration) (string, error)
        Delete(ctx context.Context, key string) error
    }
    ```

### 1.8. Background Jobs
*   **Workers**:
    1.  **Keyspace Notification Listener**: A long-running goroutine that subscribes to Redis `__keyevent@0__:expired` events. When a `hold:{session_id}` key expires, it triggers the ticket release process in PostgreSQL.
    2.  **Reconciliation Cron**: A scheduled task running every 10 seconds (using `robfig/cron/v3`) to sweep PostgreSQL for expired holds (`expires_at < NOW()`) that were missed due to network partitions, ensuring eventual consistency.
*   **Graceful Shutdown**: All workers listen to a context cancellation signal to finish processing active tasks before the application exits.

### 1.9. Monitoring & Telemetry
*   **Technology**: OpenTelemetry (OTel) + Prometheus.
*   **Metrics**:
    *   Exposes a `/metrics` endpoint for Prometheus scraping.
    *   Tracks HTTP request latencies, active SSE connections, Redis pool saturation, and DB connection states.
*   **Distributed Tracing**:
    *   Instruments the `pgx` driver and `go-redis` client to trace database queries and Redis commands.
    *   Injects and extracts trace contexts across HTTP boundaries.

### 1.10. Logging
*   **Technology**: Structured JSON Logging.
*   **Implementation**: Integrates Go's standard `charles/slog` (leveraging the existing [logger.go](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/apps/backend/internal/shared/logger/logger.go)) with all infrastructure components.

### 1.11. Configuration & Secrets
*   **Configuration**: Extends [config.go](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/apps/backend/internal/shared/config/config.go).
*   **Secret Management**:
    *   **Local**: Loaded from `.env` via `godotenv`.
    *   **Production**: Secrets are injected as environment variables by the container orchestrator (e.g., AWS ECS retrieving from AWS Secrets Manager).

---

## 2. Infrastructure Folder Structure

The infrastructure layer resides in `apps/backend/internal/infrastructure`. It is strictly separated by concern:

```
apps/backend/internal/infrastructure/
├── auth/
│   ├── jwt.go                 # JWT token generation and validation
│   ├── hash.go                # Bcrypt passcode hashing utilities
│   └── middleware.go          # Gin authentication & authorization middlewares
├── cache/
│   ├── redis.go               # Redis client setup and pool configuration
│   ├── scripts.go             # Lua script loader and execution wrapper
│   └── lock_repository.go     # Redis-backed implementation of domain LockRepository
├── database/
│   ├── postgres.go            # Pgxpool connection setup
│   ├── tx_manager.go          # Transaction manager (Unit of Work)
│   ├── migrations/            # SQL migration files (.sql)
│   │   ├── 0001_init.up.sql
│   │   └── 0001_init.down.sql
│   └── repositories/          # Sqlc-generated code and repository implementations
│       ├── models.go          # Sqlc-generated models
│       ├── db.go              # Sqlc-generated DB interface
│       ├── tickets.sql.go     # Sqlc-generated ticket queries
│       ├── orders.sql.go      # Sqlc-generated order queries
│       ├── ticket_repository.go # Postgres implementation of domain TicketRepository
│       └── order_repository.go  # Postgres implementation of domain OrderRepository
├── jobs/
│   ├── scheduler.go           # Cron job manager (robfig/cron)
│   ├── expiration_worker.go   # Redis Keyspace Notification subscriber
│   └── reconciliation.go      # Fail-safe database hold sweeper
├── realtime/
│   ├── sse_broker.go          # Server-Sent Events connection broker
│   └── pubsub_listener.go     # Listens to Redis Pub/Sub for inventory broadcasts
├── storage/
│   ├── storage.go             # Storage interface definition
│   ├── s3.go                  # AWS S3 storage implementation
│   └── local.go               # Local filesystem storage implementation
└── telemetry/
    ├── metrics.go             # Prometheus metrics registry and instruments
    └── tracer.go              # OpenTelemetry tracer provider configuration
```

---

## 3. Package Responsibilities & Boundaries

| Package | Primary Responsibility | Outer Dependencies | Inner Interfaces Implemented |
| :--- | :--- | :--- | :--- |
| `auth` | Token creation, validation, and route guard middleware. | `jwt-go`, `bcrypt`, `gin` | None (provides middleware helper functions) |
| `cache` | Interacts with Redis; manages atomic Lua scripts. | `go-redis` | `domain/repository.LockRepository` |
| `database` | Manages PostgreSQL connection pools and SQL transactions. | `pgx/v5`, `sqlc` | `domain/repository.TicketRepository`, `domain/repository.OrderRepository` |
| `jobs` | Coordinates background tasks and cron schedules. | `robfig/cron`, `go-redis` | None (runs as background workers) |
| `realtime` | Manages active SSE client streams and distributes events. | `gin`, `go-redis` | `domain/service.NotificationService` |
| `storage` | Uploads/downloads files (invoices, tickets). | `aws-sdk-go-v2` | `infrastructure/storage.Storage` |
| `telemetry` | Collects Prometheus metrics and manages OTel spans. | `prometheus/client_golang`, `otel` | None (provides interceptors/instruments) |

---

## 4. Initialization Order & Bootstrapping

To ensure that components are initialized only when their dependencies are ready, the application must follow a strict **topological bootstrapping order** in `main.go`.

### 4.1. Initialization Sequence

```mermaid
sequenceDiagram
    autonumber
    participant Main as main.go
    participant Config as Config Loader
    participant Telemetry as Telemetry (OTel/Prom)
    participant DB as Postgres (pgxpool)
    participant Cache as Redis Client
    participant Repos as Repositories & TxManager
    participant SSE as SSE Broker
    participant Jobs as Background Jobs
    participant Server as HTTP Server (Gin)

    Main->>Config: 1. Load & Validate Env Variables
    Config-->>Main: Config struct

    Main->>Telemetry: 2. Initialize OTel & Prometheus
    Telemetry-->>Main: TracerProvider, MetricsRegistry

    Main->>DB: 3. Connect to PostgreSQL (pgxpool)
    DB-->>Main: *pgxpool.Pool

    Main->>Cache: 4. Connect to Redis & Load Lua Scripts
    Cache-->>Main: *redis.Client

    Main->>Repos: 5. Initialize Repositories & TxManager
    Repos-->>Main: TicketRepo, OrderRepo, TxManager

    Main->>SSE: 6. Start SSE Broker & Pub/Sub Listener
    SSE-->>Main: *SSEBroker (starts goroutine)

    Main->>Jobs: 7. Start Expiration Worker & Cron Scheduler
    Jobs-->>Main: Background goroutines started

    Main->>Server: 8. Setup Routes & Start HTTP Engine
    Server-->>Main: Listening on :PORT
```

### 4.2. Detailed Initialization Steps

1.  **Step 1: Configuration**: Load environment variables. If any critical variable (`DATABASE_URL`, `REDIS_URL`) is missing, the application panics immediately.
2.  **Step 2: Telemetry**: Configure the global OpenTelemetry Tracer Provider and register Prometheus metrics. This ensures all subsequent database and cache connection attempts are traced.
3.  **Step 3: PostgreSQL (`database`)**: Establish the `pgxpool.Pool`. Execute a ping to verify connectivity. If the database is unreachable, retry 3 times before failing.
4.  **Step 4: Redis (`cache`)**: Establish the `redis.Client`. Run a ping command. Load the Lua reservation script into the Redis script cache using `SCRIPT LOAD` to obtain its SHA-1 hash.
5.  **Step 5: Repositories & Services**: Inject the database pool and Redis client into the respective repositories. Initialize the `TxManager`.
6.  **Step 6: Realtime SSE**: Initialize `SSEBroker`. Spin up its event loop in a background goroutine. Initialize the Pub/Sub listener which subscribes to the Redis channel and forwards events to the broker.
7.  **Step 7: Background Workers**:
    *   Start the `ExpirationWorker` to subscribe to Redis keyspace events.
    *   Initialize and start the `robfig/cron` scheduler with the 10-second reconciliation task.
8.  **Step 8: HTTP Server**: Initialize the Gin engine, register middleware (Logger, Recovery, CORS, Telemetry, Auth), attach routing groups (public, user, admin), and start the server.

---

## 5. Environment Variables & Secret Management

The following variables must be defined in the environment. Secrets are marked as **[SECRET]** and must be managed securely in production environments.

```env
# --- Application Configuration ---
PORT=8080
APP_ENV=production                  # development | staging | production
GIN_MODE=release                    # debug | release

# --- Database (PostgreSQL) ---
DATABASE_URL=postgres://user:password@pgbouncer-host:6432/booking?sslmode=require  # [SECRET] (PgBouncer Pool)
DIRECT_DATABASE_URL=postgres://user:password@postgres-host:5432/booking?sslmode=require  # [SECRET] (Direct for Migrations)

# --- Cache & Pub/Sub (Redis) ---
REDIS_URL=rediss://:password@redis-host:6379/0  # [SECRET]

# --- Security ---
ADMIN_TOKEN=sha256_hash_of_admin_passcode      # [SECRET] (Verifies admin access)
JWT_SECRET=super_secret_signing_key            # [SECRET] (Signs user session tokens)

# --- Object Storage (AWS S3) ---
AWS_REGION=ap-southeast-1
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE         # [SECRET]
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY  # [SECRET]
S3_BUCKET_NAME=ticket-booking-invoices

# --- Telemetry & Monitoring ---
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
OTEL_SERVICE_NAME=ticket-booking-backend
```

---

## 6. Infrastructure Error Handling Strategy

All infrastructure components must map raw library errors (e.g., `pgx.ErrNoRows`, `redis.Nil`) to the unified domain error codes defined in [codes.go](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/apps/backend/internal/shared/errors/codes.go). 

*   **PostgreSQL**: If a query returns `pgx.ErrNoRows`, the repository must wrap it as a `shared/errors.NotFound` error.
*   **Redis**: If a lock check returns `redis.Nil`, it must be translated to `shared/errors.NotFound` or handled gracefully depending on the context.
*   **Database Constraints**: Violations of the partial unique index `idx_tickets_session_id_unique` must be caught via Postgres error code `23505` (unique_violation) and mapped to `shared/errors.Conflict` (e.g., "Active reservation already exists for this session").
