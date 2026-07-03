.PHONY: help infra-up infra-down migrate-up migrate-down seed db-reset reset-tickets seed-low-stock run-dev

DB_CONTAINER=ticket-booking-postgres
DB_NAME=ticket_booking
DB_USER=postgres

help:
	@echo "Available commands:"
	@echo "  make infra-up      - Start PostgreSQL and Redis containers"
	@echo "  make infra-down    - Stop and remove containers"
	@echo "  make migrate-up    - Run database migrations (up)"
	@echo "  make migrate-down  - Rollback database migrations (down)"
	@echo "  make seed          - Seed the database with tickets and admin config"
	@echo "  make db-reset      - Reset database (down, up, and seed)"
	@echo "  make reset-tickets - Reset ticket/order data only (keep schema) and resync Redis"
	@echo "  make seed-low-stock [CATEGORY=VIP] - Collapse a category down to 1 available ticket"
	@echo "  make run-dev       - Start frontend and backend development servers"

infra-up:
	docker compose up -d

infra-down:
	docker compose down

migrate-up:
	@echo "Applying migrations..."
	docker exec -i $(DB_CONTAINER) psql -U $(DB_USER) -d $(DB_NAME) < apps/backend/db/migrations/000001_init_schema.up.sql
	@echo "Migrations applied successfully."

migrate-down:
	@echo "Rolling back migrations..."
	docker exec -i $(DB_CONTAINER) psql -U $(DB_USER) -d $(DB_NAME) < apps/backend/db/migrations/000001_init_schema.down.sql
	@echo "Migrations rolled back successfully."

seed:
	@echo "Seeding database..."
	docker exec -i $(DB_CONTAINER) psql -U $(DB_USER) -d $(DB_NAME) < apps/backend/db/seeds/seed_tickets.sql
	@echo "Database seeded successfully."

db-reset: migrate-down migrate-up seed

reset-tickets:
	@echo "Resetting ticket-related data (tickets, orders) and resyncing Redis..."
	@bash apps/backend/db/reset_tickets.sh
	@echo "Ticket data reset complete."

seed-low-stock:
	@bash apps/backend/db/seed_low_stock.sh $(CATEGORY)

run-dev:
	npm run dev
