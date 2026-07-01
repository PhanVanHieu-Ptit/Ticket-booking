package reclamation

import (
	"context"
	"strings"
	"time"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/logger"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/sse"
	"github.com/redis/go-redis/v9"
)

type reclamationService struct {
	repo  Repository
	redis *redis.Client
}

// NewReclamationService creates a new instance of reclamation Service.
func NewReclamationService(repo Repository, redis *redis.Client) Service {
	return &reclamationService{
		repo:  repo,
		redis: redis,
	}
}

func (s *reclamationService) StartReclamationWorker(ctx context.Context) error {
	pubsub := s.redis.Subscribe(ctx, "__keyevent@0__:expired")
	defer pubsub.Close()

	ch := pubsub.Channel()
	logger.Info("Redis expired keyspace event listener started successfully")

	for {
		select {
		case <-ctx.Done():
			logger.Info("Redis keyspace event listener stopping...")
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				logger.Warn("Redis keyspace event listener channel closed")
				return nil
			}

			key := msg.Payload
			if strings.HasPrefix(key, "hold:") {
				sessionID := strings.TrimPrefix(key, "hold:")
				logger.Info("Received Redis expiration hold event", "session_id", sessionID)

				category, err := s.repo.ReclaimSessionHold(ctx, sessionID)
				if err != nil {
					logger.Error("Failed to reclaim expired session hold", "session_id", sessionID, "error", err)
					continue
				}

				if category != "" {
					logger.Info("Successfully reclaimed expired ticket hold from Redis keyspace event", "session_id", sessionID, "category", category)
					// Broadcast new available count
					count, err := s.redis.SCard(ctx, "tickets:available:"+category).Result()
					if err == nil {
						sse.BroadcastInventoryUpdate(category, count)
					} else {
						logger.Error("Failed to fetch ticket count for SSE broadcast", "category", category, "error", err)
					}
				}
			}
		}
	}
}

func (s *reclamationService) StartReconciliationCron(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	logger.Info("Reconciliation cron sweeper started successfully")

	for {
		select {
		case <-ctx.Done():
			logger.Info("Reconciliation cron sweeper stopping...")
			return ctx.Err()
		case <-ticker.C:
			count, err := s.repo.ReclaimExpiredHolds(ctx)
			if err != nil {
				logger.Error("Reconciliation sweep failed", "error", err)
				continue
			}

			if count > 0 {
				logger.Info("Cron sweeper reclaimed expired holds", "count", count)

				// Broadcast inventory updates for both categories
				for _, category := range []string{"VIP", "Standard"} {
					available, err := s.redis.SCard(ctx, "tickets:available:"+category).Result()
					if err == nil {
						sse.BroadcastInventoryUpdate(category, available)
					} else {
						logger.Error("Failed to fetch ticket count for cron SSE broadcast", "category", category, "error", err)
					}
				}
			}
		}
	}
}
