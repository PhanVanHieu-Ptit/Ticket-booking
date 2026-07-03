package handlers

import (
	"database/sql"
	"net/http"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/sse"
	appErrors "github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/errors"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/holdtimer"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/logger"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/types"
	"github.com/gin-gonic/gin"
)

// CancelHold handles POST /api/v1/tickets/hold/cancel
func (h *ReservationHandler) CancelHold(c *gin.Context) {
	ctx := c.Request.Context()

	// 1. Fetch session ID from request context (set by SessionMiddleware)
	sessionIDVal, exists := c.Get("session_id")
	if !exists {
		c.Error(appErrors.New(http.StatusUnauthorized, appErrors.ErrCodeSessionRequired, "Session token is required"))
		return
	}
	sessionID := sessionIDVal.(string)

	// 2. Start PostgreSQL Transaction
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("Failed to start database transaction for cancelling hold", "error", err, "session_id", sessionID)
		c.Error(appErrors.NewInternal(err, "Internal server error"))
		return
	}
	defer tx.Rollback()

	// 3. Query and lock the active hold row for the session
	var ticketID int64
	var category string
	err = tx.QueryRowContext(ctx, `
		SELECT id, category
		FROM tickets
		WHERE session_id = $1 AND status = 'Holding' AND expires_at > NOW()
		FOR UPDATE
	`, sessionID).Scan(&ticketID, &category)

	if err != nil {
		if err == sql.ErrNoRows {
			c.Error(appErrors.New(http.StatusBadRequest, appErrors.ErrCodeNoActiveHold, "No active reservation was found for this session."))
			return
		}
		logger.Error("Database query failed while fetching hold to cancel", "error", err, "session_id", sessionID)
		c.Error(appErrors.NewInternal(err, "Internal server error"))
		return
	}

	// 4. Update PostgreSQL ticket row to Available
	_, err = tx.ExecContext(ctx, `
		UPDATE tickets
		SET status = 'Available', session_id = NULL, held_at = NULL, expires_at = NULL, updated_at = NOW()
		WHERE id = $1
	`, ticketID)
	if err != nil {
		logger.Error("Database update failed while cancelling hold", "error", err, "ticket_id", ticketID, "session_id", sessionID)
		c.Error(appErrors.NewInternal(err, "Internal server error"))
		return
	}

	// 5. Commit PostgreSQL transaction
	if err := tx.Commit(); err != nil {
		logger.Error("Database commit failed while cancelling hold", "error", err, "ticket_id", ticketID, "session_id", sessionID)
		c.Error(appErrors.NewInternal(err, "Internal server error"))
		return
	}
	holdtimer.Clear(ticketID)

	// 6. Redis updates - Delete hold key and add ticket back to available set
	if err := h.redisSvc.ReleaseHold(ctx, sessionID, category, ticketID); err != nil {
		logger.Error("Failed to execute Redis cancellation release", "error", err, "session_id", sessionID, "ticket_id", ticketID)
	}

	// 7. Get new Redis count and broadcast to SSE broker
	newCount, err := h.redisSvc.AvailableCount(ctx, category)
	if err == nil {
		sse.BroadcastInventoryUpdate(category, newCount)
	} else {
		logger.Error("Failed to get updated ticket count from Redis", "error", err, "category", category)
	}

	// 8. Return success response
	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"message": "Reservation cancelled successfully",
	}))
}
