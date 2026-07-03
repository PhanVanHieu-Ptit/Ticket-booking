#!/usr/bin/env bash
# Resets all ticket-related data (tickets, orders) back to the seeded state
# and resyncs Redis so the atomic hold script agrees with Postgres again.
# Does NOT touch admin_configs. Safe to re-run.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

DB_CONTAINER="${DB_CONTAINER:-ticket-booking-postgres}"
DB_NAME="${DB_NAME:-ticket_booking}"
DB_USER="${DB_USER:-postgres}"
REDIS_CONTAINER="${REDIS_CONTAINER:-ticket-booking-redis}"

echo "==> Truncating tickets and orders..."
docker exec -i "${DB_CONTAINER}" psql -U "${DB_USER}" -d "${DB_NAME}" \
  -c "TRUNCATE TABLE orders, tickets RESTART IDENTITY CASCADE;"

echo "==> Reseeding tickets..."
docker exec -i "${DB_CONTAINER}" psql -U "${DB_USER}" -d "${DB_NAME}" \
  < "${SCRIPT_DIR}/seeds/seed_tickets.sql"

echo "==> Resyncing Redis inventory from Postgres..."
(cd "${BACKEND_DIR}" && go run ./cmd/seedredis)

echo "==> Clearing stale hold/purchase state in Redis..."
docker exec "${REDIS_CONTAINER}" redis-cli --scan --pattern 'hold:*' \
  | xargs -r -I{} docker exec "${REDIS_CONTAINER}" redis-cli DEL {} > /dev/null
docker exec "${REDIS_CONTAINER}" redis-cli DEL purchased:sessions > /dev/null

echo "==> Summary:"
docker exec -i "${DB_CONTAINER}" psql -U "${DB_USER}" -d "${DB_NAME}" \
  -c "SELECT status, category, COUNT(*) FROM tickets GROUP BY status, category ORDER BY category, status;" \
  -c "SELECT COUNT(*) AS orders_remaining FROM orders;"

echo "Ticket data reset complete."
