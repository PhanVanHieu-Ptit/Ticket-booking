.PHONY: help infra-up infra-down migrate-up migrate-down seed db-reset reset-tickets seed-low-stock test run-dev

DB_CONTAINER=ticket-booking-postgres
DB_NAME=ticket_booking
DB_USER=postgres

help:
	@echo "Available commands:"
	@echo "  make infra-up      - Start PostgreSQL and Redis containers"
	@echo "  make infra-down    - Stop and remove containers"
	@echo "  make migrate-up    - Run database migrations (up). Set DIRECT_DATABASE_URL to target a remote DB instead of local Docker."
	@echo "  make migrate-down  - Rollback database migrations (down). Set DIRECT_DATABASE_URL to target a remote DB instead of local Docker."
	@echo "  make seed          - Seed the database with tickets and admin config. Set DIRECT_DATABASE_URL to target a remote DB instead of local Docker."
	@echo "  make db-reset      - Reset database (down, up, and seed)"
	@echo "  make reset-tickets - Reset ticket/order data only (keep schema) and resync Redis"
	@echo "  make seed-low-stock [CATEGORY=VIP] - Collapse a category down to 1 available ticket"
	@echo "  make test          - Run the backend Go test suite (Postgres/Redis integration tests skip automatically if infra isn't running)"
	@echo "  make run-dev       - Start frontend and backend development servers"

infra-up:
	docker compose up -d

infra-down:
	docker compose down

migrate-up:
	@echo "Applying migrations..."
ifdef DIRECT_DATABASE_URL
	@for f in apps/backend/db/migrations/*.up.sql; do \
		echo "  applying $$f"; \
		psql "$(DIRECT_DATABASE_URL)" -v ON_ERROR_STOP=1 -f $$f || exit 1; \
	done
else
	@for f in apps/backend/db/migrations/*.up.sql; do \
		echo "  applying $$f"; \
		docker exec -i $(DB_CONTAINER) psql -U $(DB_USER) -d $(DB_NAME) -v ON_ERROR_STOP=1 < $$f || exit 1; \
	done
endif
	@echo "Migrations applied successfully."

migrate-down:
	@echo "Rolling back migrations..."
ifdef DIRECT_DATABASE_URL
	@for f in $$(ls -r apps/backend/db/migrations/*.down.sql); do \
		echo "  applying $$f"; \
		psql "$(DIRECT_DATABASE_URL)" -v ON_ERROR_STOP=1 -f $$f || exit 1; \
	done
else
	@for f in $$(ls -r apps/backend/db/migrations/*.down.sql); do \
		echo "  applying $$f"; \
		docker exec -i $(DB_CONTAINER) psql -U $(DB_USER) -d $(DB_NAME) -v ON_ERROR_STOP=1 < $$f || exit 1; \
	done
endif
	@echo "Migrations rolled back successfully."

seed:
	@echo "Seeding database..."
ifdef DIRECT_DATABASE_URL
	psql "$(DIRECT_DATABASE_URL)" -v ON_ERROR_STOP=1 -f apps/backend/db/seeds/seed_tickets.sql
else
	docker exec -i $(DB_CONTAINER) psql -U $(DB_USER) -d $(DB_NAME) -v ON_ERROR_STOP=1 < apps/backend/db/seeds/seed_tickets.sql
endif
	@echo "Database seeded successfully."

db-reset: migrate-down migrate-up seed

reset-tickets:
	@echo "Resetting ticket-related data (tickets, orders) and resyncing Redis..."
	@bash apps/backend/db/reset_tickets.sh
	@echo "Ticket data reset complete."

seed-low-stock:
	@bash apps/backend/db/seed_low_stock.sh $(CATEGORY)

test:
	npm run test:backend

run-dev:
	npm run dev
