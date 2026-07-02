// Command seedredis seeds Redis's "tickets:available:{category}" sets from
// the current PostgreSQL ticket inventory. It is meant to be run once as a
// deploy-time step (e.g. after a fresh Redis instance is provisioned, or
// after a Redis data loss/flush), independently of the API server boot
// sequence which also runs this sync on every startup.
//
// This schema tracks inventory as individual ticket rows (tickets.status),
// not as total_quantity/sold_quantity counters, so the "available count" per
// category is COUNT(*) WHERE status = 'Available' AND category = ? — the
// equivalent of total_quantity - sold_quantity - held_quantity for this data
// model. See internal/redis.SyncInventory for the query.
package main

import (
	"log"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/redis"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/config"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/db"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	logger.Init(cfg.AppEnv)

	database, err := db.Init(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	if _, err := redis.Init(cfg.RedisURL); err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}

	if err := redis.SyncInventory(database); err != nil {
		log.Fatalf("Failed to seed Redis inventory from database: %v", err)
	}

	logger.Info("Redis inventory seed completed successfully")
}
