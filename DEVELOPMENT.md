# Local Development Setup Guide

This guide provides step-by-step instructions to set up, run, and verify the high-concurrency ticket booking system on your local machine.

---

## Prerequisites

Before starting, ensure you have the following installed:

1. **Node.js** (v18.x or later) & **npm** (v10.x or later)
2. **Go (Golang)** (v1.21 or later)
3. **Docker** & **Docker Compose**

---

## Step-by-Step Setup

### 1. Install Node.js Dependencies

From the root directory, run the following command to install the required packages for the root workspace and frontend:

```bash
npm install
```

### 2. Start the Local Infrastructure

Spin up the local database, connection pooler, and caching layers using Docker Compose:

```bash
npm run infra:up
```

This command starts:

- **PostgreSQL** on host port `5433` (direct access)
- **PgBouncer** on host port `5432` (transaction-pooled database access)
- **Redis** on host port `6379` (with keyspace expiration events enabled)

To verify the services are running and healthy:

```bash
docker compose ps
```

### 3. Configure Environment Variables

Copy the backend `.env.example` file to `.env`:

```bash
cp apps/backend/.env.example apps/backend/.env
```

The default values are pre-configured to connect to the PgBouncer and Redis instances running in Docker.

### 4. Start the Development Servers

Start both the Go API server and the Vite React frontend concurrently:

```bash
npm run dev
```

- The **Frontend** will be available at [http://localhost:3000](http://localhost:3000)
- The **Backend** will run at [http://localhost:8080](http://localhost:8080)
- Requests from the frontend to `/api/*` are automatically proxied to the backend.

---

## Folder Explanation

Here is a breakdown of the repository structure:

```
├── apps/
│   ├── frontend/             # React SPA (Vite + TS + Tailwind)
│   │   ├── src/
│   │   │   ├── assets/       # Images, SVG icons, and fonts
│   │   │   ├── components/   # UI components (buttons, dialogs, cards)
│   │   │   ├── hooks/        # React hooks (e.g., countdown timer, SSE listener)
│   │   │   ├── lib/          # Helper libraries (Axios client, utils)
│   │   │   ├── pages/        # Application views (Home, Booking, Admin)
│   │   │   ├── App.tsx       # Root React component (UI shell)
│   │   │   └── main.tsx      # Entrypoint script
│   │   ├── vite.config.ts    # Vite configuration (port 3000, API proxy)
│   │   └── tsconfig.json     # TypeScript configuration
│   └── backend/              # Go (Gin) API Server
│       ├── cmd/
│       │   └── api/
│       │       └── main.go   # Go entrypoint (port 8080, health check)
│       └── go.mod            # Go module dependencies
├── docker/
│   └── pgbouncer/
│       ├── pgbouncer.ini     # PgBouncer configuration (transaction pool)
│       └── userlist.txt      # Allowed database credentials
├── docker-compose.yml        # Orchestrates Postgres, PgBouncer, and Redis
├── package.json              # Root npm workspace configuration & script runner
├── eslint.config.js          # ESLint v9 Flat configuration
├── .prettierrc               # Prettier formatting rules
└── .editorconfig             # Editor settings standardization
```

---

## Database & Caching Details

### PgBouncer Connection Pooling

The application is configured to connect to PostgreSQL via **PgBouncer** on port `5432` rather than connecting directly to PostgreSQL on port `5433`.

- **Why?** Under high concurrency (5,000 concurrent users), direct connections to PostgreSQL would exhaust the database's connection limit and crash the server. PgBouncer multiplexes these connections.
- **Pool Mode**: `transaction`. Each database transaction gets a connection from the pool, which is returned immediately after the transaction commits or rolls back.
- **Note**: For schema migrations, you should bypass PgBouncer and connect directly to PostgreSQL on port `5433` (configured as `DIRECT_DATABASE_URL` in `.env`), because transaction pooling does not support certain DDL commands or session-level states.

### Redis Keyspace Notifications

Redis is configured with the command:

```bash
redis-server --notify-keyspace-events Kx
```

- **`K`**: Keyspace events (events targeted at keyspace, published with `__keyspace@<db>__` prefix).
- **`x`**: Expired events (events generated when a key expires).
- This is critical for the **Hold Reclamation** feature. When a ticket hold key (`hold:{session_id}`) expires after 5 minutes (300 seconds), Redis emits an event. The background worker listens to this event and automatically releases the corresponding ticket in PostgreSQL.

---

## Helpful Development Scripts

| Command                | Description                                        |
| :--------------------- | :------------------------------------------------- |
| `npm run dev`          | Starts both frontend and backend dev servers.      |
| `npm run dev:frontend` | Starts the Vite frontend server only.              |
| `npm run dev:backend`  | Starts the Go backend server only.                 |
| `npm run infra:up`     | Starts Postgres, PgBouncer, and Redis containers.  |
| `npm run infra:down`   | Stops and removes local infrastructure containers. |
| `npm run infra:logs`   | Tunnels container logs in real-time.               |
| `npm run lint`         | Runs ESLint checks across the frontend workspace.  |
| `npm run format`       | Formats all source files with Prettier.            |
