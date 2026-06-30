# Database Design Specification: High-Concurrency Ticket Booking System

This document details the database design for the Concert Ticket Booking System. To support a high-concurrency spike of **5,000 concurrent users** competing for **500 tickets** (100 VIP, 400 Standard), the data architecture utilizes a **dual-database design**:

1. **PostgreSQL (RDS)**: The persistent, ACID-compliant relational database serving as the absolute source of truth for ticket ownership, financial audits, and system configuration.
2. **Redis (ElastiCache)**: The high-performance, in-memory **Concurrency Shield** that intercepting high-write spikes, manages temporary 5-minute ticket holds, and enforces the 1-ticket-per-session limit before requests touch PostgreSQL.

---

## 1. Entity-Relationship (ER) Diagram

The following diagram illustrates the PostgreSQL relational schema and its logical connection to the Redis in-memory Concurrency Shield.

```mermaid
erDiagram
    %% PostgreSQL Relational Schema
    TICKETS {
        int id PK "Auto-incrementing Identifier"
        varchar ticket_code UK "Cryptographically secure ticket code"
        varchar category "VIP | Standard"
        numeric price "100.00 | 50.00"
        varchar status "Available | Holding | Sold"
        varchar session_id UK "Nullable; Unique active session identifier"
        timestamp held_at "Timestamp of reservation"
        timestamp expires_at "Timestamp of hold expiration"
        timestamp created_at
        timestamp updated_at
    }

    ORDERS {
        uuid id PK "UUIDv4 Order Identifier"
        int ticket_id FK, UK "Unique reference to the ticket"
        varchar session_id UK "Session that completed the purchase"
        numeric amount "Total paid amount"
        varchar status "Paid | Refunded | Failed"
        varchar email "Buyer's email address"
        varchar card_holder_name "Name on payment card"
        varchar payment_reference UK "External gateway transaction ID"
        timestamp created_at
        timestamp updated_at
    }

    ADMIN_CONFIGS {
        varchar key PK "Config key (e.g., admin_passcode)"
        text value "Config value"
        text description "Configuration description"
        timestamp updated_at
    }

    %% Relationships
    TICKETS ||--o| ORDERS : "purchased_via (1:0..1)"

    %% Conceptual Redis Concurrency Shield mapping
    subgraph Redis Shield [In-Memory Concurrency Shield]
        redis_avail_vip["Set: tickets:available:VIP"]
        redis_avail_std["Set: tickets:available:Standard"]
        redis_hold_session["String: hold:{session_id}"]
        redis_purchased_sessions["Set: purchased:sessions"]
    end
```

---

## 2. PostgreSQL Schema (Relational Source of Truth)

### 2.1. Table: `tickets`

This table represents the individual tickets available for the concert. There are exactly 500 pre-generated rows (100 VIP, 400 Standard).

| Column Name   | Data Type       | Nullable | Default       | Constraints / Indexes | Description                                                                              |
| :------------ | :-------------- | :------- | :------------ | :-------------------- | :--------------------------------------------------------------------------------------- |
| `id`          | `INT`           | No       | _Nextval_     | `PRIMARY KEY`         | Auto-incrementing unique identifier.                                                     |
| `ticket_code` | `VARCHAR(64)`   | No       | None          | `UNIQUE`              | A unique, cryptographically secure string (e.g., `TKT-VIP-8f2a9d...`) shown to the user. |
| `category`    | `VARCHAR(20)`   | No       | None          | `CHECK`               | Category of the ticket: `VIP` or `Standard`.                                             |
| `price`       | `NUMERIC(10,2)` | No       | None          | `CHECK (>= 0)`        | Ticket price (VIP = $100.00, Standard = $50.00).                                         |
| `status`      | `VARCHAR(20)`   | No       | `'Available'` | `CHECK`               | State machine status: `Available`, `Holding`, `Sold`.                                    |
| `session_id`  | `VARCHAR(255)`  | Yes      | `NULL`        | `PARTIAL UNIQUE`      | The session ID holding or purchasing this ticket.                                        |
| `held_at`     | `TIMESTAMPTZ`   | Yes      | `NULL`        | None                  | The timestamp when the 5-minute hold started.                                            |
| `expires_at`  | `TIMESTAMPTZ`   | Yes      | `NULL`        | `INDEX`               | The timestamp when the 5-minute hold expires.                                            |
| `created_at`  | `TIMESTAMPTZ`   | No       | `NOW()`       | None                  | Record creation timestamp.                                                               |
| `updated_at`  | `TIMESTAMPTZ`   | No       | `NOW()`       | None                  | Record last-modification timestamp.                                                      |

### 2.2. Table: `orders`

This table records completed purchases. A successful payment creates a record here and permanently updates the corresponding ticket to `Sold`.

| Column Name         | Data Type       | Nullable | Default             | Constraints / Indexes   | Description                                             |
| :------------------ | :-------------- | :------- | :------------------ | :---------------------- | :------------------------------------------------------ |
| `id`                | `UUID`          | No       | `gen_random_uuid()` | `PRIMARY KEY`           | Unique transaction ID.                                  |
| `ticket_id`         | `INT`           | No       | None                | `FOREIGN KEY`, `UNIQUE` | Reference to the purchased ticket.                      |
| `session_id`        | `VARCHAR(255)`  | No       | None                | `UNIQUE`                | Enforces the "1 purchase per session" rule.             |
| `amount`            | `NUMERIC(10,2)` | No       | None                | `CHECK (>= 0)`          | Amount paid for the ticket.                             |
| `status`            | `VARCHAR(20)`   | No       | `'Paid'`            | `CHECK`                 | Order status: `Paid`, `Refunded`, `Failed`.             |
| `email`             | `VARCHAR(255)`  | No       | None                | None                    | Customer's email address for receipt delivery.          |
| `card_holder_name`  | `VARCHAR(255)`  | No       | None                | None                    | Cardholder name from checkout.                          |
| `payment_reference` | `VARCHAR(255)`  | No       | None                | `UNIQUE`                | Reference ID returned by the simulated payment gateway. |
| `created_at`        | `TIMESTAMPTZ`   | No       | `NOW()`             | None                    | Order creation timestamp.                               |
| `updated_at`        | `TIMESTAMPTZ`   | No       | `NOW()`             | None                    | Order last-update timestamp.                            |

### 2.3. Table: `admin_configs`

Stores system-wide administrative settings, such as the passcode required to access the `/admin` dashboard.

| Column Name   | Data Type     | Nullable | Default | Constraints   | Description                                   |
| :------------ | :------------ | :------- | :------ | :------------ | :-------------------------------------------- |
| `key`         | `VARCHAR(50)` | No       | None    | `PRIMARY KEY` | Configuration key (e.g., `'admin_passcode'`). |
| `value`       | `TEXT`        | No       | None    | None          | Configuration value (hashed passcode).        |
| `description` | `TEXT`        | Yes      | `NULL`  | None          | Description of the configuration's purpose.   |
| `updated_at`  | `TIMESTAMPTZ` | No       | `NOW()` | None          | Timestamp of last configuration change.       |

---

## 3. Database Constraints

Data integrity is enforced at the database layer using strict SQL constraints:

### 3.1. Domain & State Constraints (`tickets`)

- **Price Validation**: Enforces that ticket prices cannot be negative.
  ```sql
  CONSTRAINT chk_ticket_price CHECK (price >= 0)
  ```
- **State Machine Guard**: Restricts the status column to the three valid states.
  ```sql
  CONSTRAINT chk_ticket_status CHECK (status IN ('Available', 'Holding', 'Sold'))
  ```
- **State Consistency Check**: Enforces that column values must remain consistent with the ticket's current status:
  - An `Available` ticket must have no session, hold time, or expiration time.
  - A `Holding` ticket must have a session, a hold time, and an expiration time.
  - A `Sold` ticket must have a session.
  ```sql
  CONSTRAINT chk_hold_dates CHECK (
      (status = 'Available' AND session_id IS NULL AND held_at IS NULL AND expires_at IS NULL) OR
      (status = 'Holding' AND session_id IS NOT NULL AND held_at IS NOT NULL AND expires_at IS NOT NULL) OR
      (status = 'Sold' AND session_id IS NOT NULL)
  )
  ```
- **Timeline Sanity**: Ensures expiration occurs after the hold is initiated.
  ```sql
  CONSTRAINT chk_expiration_after_hold CHECK (expires_at > held_at)
  ```

### 3.2. Referential & Domain Constraints (`orders`)

- **Foreign Key**: Connects the order to the ticket, preventing orphan orders. Deletion of a ticket is blocked (`RESTRICT`) if it has an associated order.
  ```sql
  FOREIGN KEY (ticket_id) REFERENCES tickets(id) ON DELETE RESTRICT
  ```
- **Order Status Guard**: Enforces valid transaction states.
  ```sql
  CONSTRAINT chk_order_status CHECK (status IN ('Paid', 'Refunded', 'Failed'))
  ```

---

## 4. Indexes & Query Optimization

To achieve sub-millisecond query execution times under a 5,000-user load, the following indexes are defined:

### 4.1. Enforcing the One-Ticket-Per-Session Limit

While a session is active, it must not be associated with more than one ticket. Since multiple tickets in the `Available` state will have `session_id = NULL`, a standard `UNIQUE` constraint cannot be used (as SQL allows multiple `NULL` values, but we want to prevent multiple non-NULL identical sessions).

- **Optimization**: A **Partial Unique Index** is created. It ignores `NULL` values but guarantees that any given `session_id` can appear at most once across all tickets currently in the `Holding` or `Sold` states.
  ```sql
  CREATE UNIQUE INDEX idx_tickets_session_id_unique
  ON tickets(session_id)
  WHERE session_id IS NOT NULL;
  ```

### 4.2. Accelerating Hold Expiration Sweeps

The background worker service polls the database every 10 seconds as a fail-safe to clean up expired holds.

- **Optimization**: A **Partial Index** on `expires_at` restricted to tickets in the `Holding` state. This prevents a sequential scan of the table and allows the database to instantly locate expired holds.
  ```sql
  CREATE INDEX idx_tickets_expires_at_holding
  ON tickets(expires_at)
  WHERE status = 'Holding';
  ```

### 4.3. High-Speed Inventory Counting

The home page requires real-time counts of available tickets by category.

- **Optimization**: A composite index on `status` and `category` enables index-only scans for inventory count queries.
  ```sql
  CREATE INDEX idx_tickets_status_category
  ON tickets(status, category);
  ```

---

## 5. Redis Schema (In-Memory Concurrency Shield)

To shield PostgreSQL from the initial 5,000 concurrent write requests, Redis acts as the gatekeeper.

### 5.1. Redis Data Structures

1. **Available Ticket Pools (Sets)**:
   - **Keys**: `tickets:available:VIP` and `tickets:available:Standard`
   - **Data Type**: `Set` (contains integer ticket IDs, e.g., `1` to `100` for VIP).
   - **Usage**: `SPOP` is used to atomically pop a random available ticket ID.
2. **Active Session Holds (Strings)**:
   - **Key**: `hold:{session_id}` (e.g., `hold:sess_9f8372...`)
   - **Data Type**: `String` (value: `{ticket_id}:{category}`, e.g., `42:VIP`).
   - **TTL**: `300` (5 minutes).
   - **Usage**: Enforces the active hold. If the key exists, the session cannot reserve another ticket. The TTL handles automatic expiration.
3. **Purchase Registry (Set)**:
   - **Key**: `purchased:sessions`
   - **Data Type**: `Set` (contains `session_id` strings).
   - **Usage**: Enforces the lifetime limit of 1 purchase per session. Checked before allowing a reservation.

### 5.2. Atomic Reservation: Redis Lua Script

When a user clicks "Reserve", the API server executes the following Lua script on Redis. Because Redis is single-threaded, the script executes atomically, eliminating race conditions.

```lua
-- KEYS[1]: session_id
-- ARGV[1]: category ("VIP" or "Standard")
-- ARGV[2]: ttl_seconds (300)

local session_id = KEYS[1]
local category = ARGV[1]
local ttl = tonumber(ARGV[2])

local hold_key = "hold:" .. session_id
local purchase_registry = "purchased:sessions"
local ticket_pool = "tickets:available:" .. category

-- 1. Check if the session has already purchased a ticket
if redis.call("SISMEMBER", purchase_registry, session_id) == 1 then
    return -1 -- Error: Purchase limit exceeded
end

-- 2. Check if the session already has an active hold
if redis.call("EXISTS", hold_key) == 1 then
    return -2 -- Error: Active hold already exists
end

-- 3. Attempt to allocate a ticket from the pool
local ticket_id = redis.call("SPOP", ticket_pool)
if not ticket_id then
    return 0 -- Error: Sold out
end

-- 4. Create the hold with TTL
redis.call("SET", hold_key, ticket_id .. ":" .. category, "EX", ttl)

return tonumber(ticket_id) -- Success: Return the allocated ticket ID
```

---

## 6. Transactions & Concurrency Control

When committing state transitions to PostgreSQL, transactions must be handled carefully to prevent race conditions and ensure data consistency.

### 6.1. Transaction 1: Reserving a Ticket (PostgreSQL Sync)

Once the Redis Lua script succeeds, the API server must immediately reflect the hold in PostgreSQL. If the database update fails (e.g., network partition or database lag), the API server rolls back the Redis state.

```sql
-- Executed immediately after Redis SPOP succeeds
UPDATE tickets
SET status = 'Holding',
    session_id = :session_id,
    held_at = NOW(),
    expires_at = NOW() + INTERVAL '5 minutes'
WHERE id = :ticket_id
  AND status = 'Available';

-- The application checks the number of affected rows:
-- If affected_rows == 1: Success. Commit.
-- If affected_rows == 0: Fail. Roll back Redis (SADD ticket_pool ticket_id, DEL hold_key).
```

### 6.2. Transaction 2: Ticket Purchase (Payment Confirmation)

When the user submits payment, the API server processes the payment and executes the following SQL transaction. We use **pessimistic locking** (`FOR UPDATE`) on the ticket row to prevent concurrent updates (such as a late-running expiration sweep).

```sql
BEGIN;

-- 1. Lock the ticket row and verify it is still held by the active session
SELECT id, status, session_id, expires_at, price
FROM tickets
WHERE id = :ticket_id
FOR UPDATE;

-- 2. Application Logic Verification:
--    - Verify status is 'Holding'
--    - Verify session_id matches the paying session_id
--    - Verify NOW() <= expires_at
-- If any verification fails, ROLLBACK the transaction and return an error.

-- 3. Transition ticket status to 'Sold' and clear expiration
UPDATE tickets
SET status = 'Sold',
    expires_at = NULL
WHERE id = :ticket_id;

-- 4. Create the purchase order record
INSERT INTO orders (id, ticket_id, session_id, amount, status, email, card_holder_name, payment_reference)
VALUES (gen_random_uuid(), :ticket_id, :session_id, :amount, 'Paid', :email, :card_holder_name, :payment_reference);

COMMIT;

-- After successful SQL commit, update the Redis Concurrency Shield:
-- 1. DEL hold:{session_id}
-- 2. SADD purchased:sessions {session_id}
-- 3. Publish real-time event to SSE/WebSocket clients
```

### 6.3. Transaction 3: Hold Expiration & Release

If a hold expires, it is released back to the pool. This is triggered by a Redis `expired` keyspace notification or by the 10-second background cron job.

```sql
BEGIN;

-- 1. Lock the expired ticket row to prevent race conditions with a last-second payment
SELECT id, category, session_id
FROM tickets
WHERE session_id = :session_id
  AND status = 'Holding'
FOR UPDATE;

-- 2. Release the ticket back to 'Available'
UPDATE tickets
SET status = 'Available',
    session_id = NULL,
    held_at = NULL,
    expires_at = NULL
WHERE session_id = :session_id
  AND status = 'Holding';

COMMIT;

-- After successful SQL commit, restore the ticket in Redis:
-- 1. SADD tickets:available:{category} {ticket_id}
-- 2. Publish updated inventory counts to SSE/WebSockets
```

---

## 7. Migration & Seeding Strategy

### 7.1. Database Migration Setup

We recommend using a lightweight migration tool like `golang-migrate` (for Go) or `Drizzle Kit` (for Node.js) to manage schema versions.

#### Migration 1: Up (`0001_init_db.up.sql`)

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE tickets (
    id SERIAL PRIMARY KEY,
    ticket_code VARCHAR(64) UNIQUE NOT NULL,
    category VARCHAR(20) NOT NULL,
    price NUMERIC(10, 2) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'Available',
    session_id VARCHAR(255) NULL,
    held_at TIMESTAMP WITH TIME ZONE NULL,
    expires_at TIMESTAMP WITH TIME ZONE NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT chk_ticket_price CHECK (price >= 0),
    CONSTRAINT chk_ticket_status CHECK (status IN ('Available', 'Holding', 'Sold')),
    CONSTRAINT chk_hold_dates CHECK (
        (status = 'Available' AND session_id IS NULL AND held_at IS NULL AND expires_at IS NULL) OR
        (status = 'Holding' AND session_id IS NOT NULL AND held_at IS NOT NULL AND expires_at IS NOT NULL) OR
        (status = 'Sold' AND session_id IS NOT NULL)
    ),
    CONSTRAINT chk_expiration_after_hold CHECK (expires_at > held_at)
);

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id INT UNIQUE NOT NULL,
    session_id VARCHAR(255) UNIQUE NOT NULL,
    amount NUMERIC(10, 2) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'Paid',
    email VARCHAR(255) NOT NULL,
    card_holder_name VARCHAR(255) NOT NULL,
    payment_reference VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (ticket_id) REFERENCES tickets(id) ON DELETE RESTRICT,
    CONSTRAINT chk_order_status CHECK (status IN ('Paid', 'Refunded', 'Failed')),
    CONSTRAINT chk_order_amount CHECK (amount >= 0)
);

CREATE TABLE admin_configs (
    key VARCHAR(50) PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE UNIQUE INDEX idx_tickets_session_id_unique ON tickets(session_id) WHERE session_id IS NOT NULL;
CREATE INDEX idx_tickets_expires_at_holding ON tickets(expires_at) WHERE status = 'Holding';
CREATE INDEX idx_tickets_status_category ON tickets(status, category);
```

#### Migration 1: Down (`0001_init_db.down.sql`)

```sql
DROP TABLE IF EXISTS admin_configs;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS tickets;
```

### 7.2. Seed Strategy

To initialize the system, the database must be seeded with exactly 500 tickets (100 VIP, 400 Standard) and the default admin configuration.

#### Seed Script (`seed.sql`)

```sql
-- 1. Seed VIP Tickets (100 tickets, IDs 1-100, price $100.00)
INSERT INTO tickets (id, ticket_code, category, price, status)
SELECT
    i,
    'TKT-VIP-' || UPPER(SUBSTRING(MD5(RANDOM()::TEXT) FROM 1 FOR 8)),
    'VIP',
    100.00,
    'Available'
FROM generate_series(1, 100) AS i;

-- 2. Seed Standard Tickets (400 tickets, IDs 101-500, price $50.00)
INSERT INTO tickets (id, ticket_code, category, price, status)
SELECT
    i,
    'TKT-STD-' || UPPER(SUBSTRING(MD5(RANDOM()::TEXT) FROM 1 FOR 8)),
    'Standard',
    50.00,
    'Available'
FROM generate_series(101, 500) AS i;

-- Adjust the sequence to prevent ID collision in future inserts
SELECT setval('tickets_id_seq', 500);

-- 3. Seed Default Admin Config (passcode: 'admin123' bcrypt-hashed)
INSERT INTO admin_configs (key, value, description)
VALUES ('admin_passcode', '$2a$12$Z.jM4wH14v6.z37N7Gj1IeaW6tCq1G.tI9vN0X184k4O3h1u5yO2O', 'Bcrypt hash of the admin dashboard access passcode');
```

---

## 8. Multi-Event Extensibility Plan

If the business expands to support multiple concerts and events, the database can be scaled cleanly without breaking the core transaction logic.

### 8.1. Relational Schema Evolution

The following schema changes would be introduced:

1. **`events` Table**: Represents different concerts or shows.
   - `id` `UUID PRIMARY KEY`
   - `title`, `description`, `venue`, `event_date`, `created_at`, `updated_at`
2. **`ticket_categories` Table**: Replaces hardcoded categories.
   - `id` `SERIAL PRIMARY KEY`
   - `event_id` `UUID REFERENCES events(id)`
   - `name` (e.g., 'VIP', 'Standard', 'Balcony')
   - `price` `NUMERIC(10,2)`
   - `total_capacity` `INT`
3. **`tickets` Schema Modification**:
   - Add `event_id` `UUID REFERENCES events(id)`
   - Replace `category` and `price` with `category_id` `INT REFERENCES ticket_categories(id)`
4. **Constraint Modification**:
   - The partial unique index is updated to allow a session to hold or purchase **1 ticket per event**:
     ```sql
     DROP INDEX idx_tickets_session_id_unique;

     CREATE UNIQUE INDEX idx_tickets_session_event_unique
     ON tickets(event_id, session_id)
     WHERE session_id IS NOT NULL;
     ```

### 8.2. Redis Key Structure Evolution

Redis keys would include the `event_id` to partition the concurrency shield:

- Available Ticket Pool: `tickets:{event_id}:available:{category_id}`
- Active Session Hold: `hold:{event_id}:{session_id}`
- Purchase Registry: `purchased:{event_id}:sessions`
