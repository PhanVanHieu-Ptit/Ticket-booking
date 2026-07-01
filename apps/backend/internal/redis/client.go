package redis

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/logger"
	"github.com/redis/go-redis/v9"
)

// RDB is the global Redis client instance.
var RDB *redis.Client

// Init initializes the Redis connection pool using the provided URL.
func Init(redisURL string) (*redis.Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// Verify connection to Redis using a Ping request
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	// Enable Redis keyspace notifications for expired events dynamically.
	// This generates __keyevent@0__:expired events on key timeouts.
	if err := client.ConfigSet(ctx, "notify-keyspace-events", "Ex").Err(); err != nil {
		logger.Warn("Failed to configure Redis keyspace notifications dynamically. "+
			"Ensure notify-keyspace-events is set to 'Ex' manually in Redis config.", "error", err)
	}

	RDB = client
	return client, nil
}

// SyncInventory queries all Available tickets in PostgreSQL and builds the available ticket sets in Redis.
func SyncInventory(db *sql.DB) error {
	ctx := context.Background()

	// Query PostgreSQL for all Available tickets
	rows, err := db.QueryContext(ctx, "SELECT id, category FROM tickets WHERE status = 'Available'")
	if err != nil {
		return fmt.Errorf("failed to query available tickets: %w", err)
	}
	defer rows.Close()

	var vipIDs []interface{}
	var stdIDs []interface{}

	for rows.Next() {
		var id int
		var category string
		if err := rows.Scan(&id, &category); err != nil {
			return fmt.Errorf("failed to scan ticket row: %w", err)
		}

		if category == "VIP" {
			vipIDs = append(vipIDs, id)
		} else if category == "Standard" {
			stdIDs = append(stdIDs, id)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("ticket rows reading error: %w", err)
	}

	// Rebuild sets in Redis using a Pipeline for atomic execution and network optimization
	pipe := RDB.Pipeline()

	vipKey := "tickets:available:VIP"
	stdKey := "tickets:available:Standard"

	// Delete existing keys to clean stale caches
	pipe.Del(ctx, vipKey, stdKey)

	if len(vipIDs) > 0 {
		pipe.SAdd(ctx, vipKey, vipIDs...)
	}
	if len(stdIDs) > 0 {
		pipe.SAdd(ctx, stdKey, stdIDs...)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute Redis pipeline sync: %w", err)
	}

	logger.Info("Ticket availability synchronized to Redis sets", "vip_available", len(vipIDs), "standard_available", len(stdIDs))
	return nil
}
