#!/usr/bin/env bash
# Collapses a ticket category down to exactly 1 Available seat via the
# test-only /api/v1/test/reset-inventory endpoint, so you can manually
# exercise the "last ticket" / oversell scenario in the browser during
# local dev (see apps/backend/internal/handlers/test_handler.go).
# Requires the backend running with APP_ENV != production.
#
# Usage: seed_low_stock.sh [CATEGORY]   (default: VIP)
set -euo pipefail

API_URL="${API_URL:-http://localhost:8080}"
CATEGORY="${1:-VIP}"

echo "==> Setting category '${CATEGORY}' down to 1 available ticket..."

body_file="$(mktemp)"
trap 'rm -f "${body_file}"' EXIT

status="$(curl -s -o "${body_file}" -w '%{http_code}' -X POST "${API_URL}/api/v1/test/reset-inventory" \
  -H "Content-Type: application/json" \
  -d "{\"category\": \"${CATEGORY}\", \"available\": 1}")" || {
  echo "Failed to reach ${API_URL}/api/v1/test/reset-inventory" >&2
  echo "Is the backend running with APP_ENV != production (npm run dev:backend)?" >&2
  exit 1
}

body="$(cat "${body_file}")"

if [[ "${status}" != "200" ]]; then
  echo "Request failed (HTTP ${status}): ${body}" >&2
  exit 1
fi

echo "==> ${body}"
echo "Category '${CATEGORY}' now has 1 available ticket."
