package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/logger"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/types"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// IdempotencyRecord is stored in Redis to cache request state and response.
type IdempotencyRecord struct {
	Status     string `json:"status"` // "PENDING" or "RESOLVED"
	StatusCode int    `json:"status_code,omitempty"`
	Body       string `json:"body,omitempty"`
}

type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// IdempotencyMiddleware ensures endpoints are idempotent using Redis caching.
func IdempotencyMiddleware(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(
				"INVALID_INPUT",
				"Idempotency-Key header is required",
				nil,
			))
			c.Abort()
			return
		}

		parsedUUID, err := uuid.Parse(key)
		if err != nil || parsedUUID.Version() != 4 {
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(
				"INVALID_INPUT",
				"Idempotency-Key must be a valid UUIDv4",
				nil,
			))
			c.Abort()
			return
		}

		sessionIDVal, exists := c.Get("session_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, types.NewErrorResponse(
				"SESSION_REQUIRED",
				"Session token is required",
				nil,
			))
			c.Abort()
			return
		}
		sessionID := sessionIDVal.(string)

		redisKey := fmt.Sprintf("idempotency:%s:%s", sessionID, key)

		// 1. Check Redis for existing record
		val, err := rdb.Get(c.Request.Context(), redisKey).Result()
		if err == nil {
			var record IdempotencyRecord
			if err := json.Unmarshal([]byte(val), &record); err == nil {
				if record.Status == "PENDING" {
					c.JSON(http.StatusConflict, types.NewErrorResponse(
						"DUPLICATE_REQUEST",
						"A request with this idempotency key is already in progress.",
						nil,
					))
					c.Abort()
					return
				} else if record.Status == "RESOLVED" {
					c.Header("Content-Type", "application/json")
					c.Status(record.StatusCode)
					_, _ = c.Writer.Write([]byte(record.Body))
					c.Abort()
					return
				}
			}
		}

		// 2. Set to PENDING with 120-second TTL
		pendingRecord := IdempotencyRecord{Status: "PENDING"}
		pendingData, _ := json.Marshal(pendingRecord)
		err = rdb.Set(c.Request.Context(), redisKey, pendingData, 120*time.Second).Err()
		if err != nil {
			logger.Error("Failed to set idempotency key in Redis", "error", err)
			c.JSON(http.StatusInternalServerError, types.NewErrorResponse(
				"INTERNAL_ERROR",
				"Internal server error",
				nil,
			))
			c.Abort()
			return
		}

		// 3. Wrap response writer to capture output
		w := &responseBodyWriter{body: &bytes.Buffer{}, ResponseWriter: c.Writer}
		c.Writer = w

		c.Next()

		// 4. Cache response or delete key based on outcome
		statusCode := c.Writer.Status()
		if statusCode >= 200 && statusCode < 300 {
			resolvedRecord := IdempotencyRecord{
				Status:     "RESOLVED",
				StatusCode: statusCode,
				Body:       w.body.String(),
			}
			resolvedData, _ := json.Marshal(resolvedRecord)
			// Cache successful response for 1 hour
			rdb.Set(context.Background(), redisKey, resolvedData, 1*time.Hour)
		} else {
			// Delete key on failure/error to allow retries
			rdb.Del(context.Background(), redisKey)
		}
	}
}
