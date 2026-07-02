package handlers

import (
	"database/sql"
	"net/http"

	appredis "github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/redis"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/errors"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/logger"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/types"
	"github.com/gin-gonic/gin"
)

// TestHandler exposes test-only endpoints that let e2e/integration tests
// deterministically manipulate ticket inventory (e.g. force a category down
// to a single remaining seat to exercise oversell/race-condition scenarios).
// Routes are only mounted by main.go when the server is not running in
// production (see cfg.IsProduction()); never expose these outside that guard.
type TestHandler struct {
	db *sql.DB
}

// NewTestHandler creates a new instance of TestHandler.
func NewTestHandler(db *sql.DB) *TestHandler {
	return &TestHandler{db: db}
}

type resetInventoryRequest struct {
	Category  string `json:"category" binding:"required"`
	Available int    `json:"available" binding:"required,min=1"`
}

// ResetInventory handles POST /api/v1/test/reset-inventory.
// It collapses the given category down to exactly `available` Available
// tickets (the lowest-ID rows in that category), marking every other ticket
// in the category Sold and clearing any stale hold/session state, purges any
// orders recorded against the category's tickets, then resyncs the Redis
// available-ticket sets so the atomic hold script agrees with Postgres. This
// is destructive to the category's inventory/order history and must never
// run against production data.
func (h *TestHandler) ResetInventory(c *gin.Context) {
	ctx := c.Request.Context()

	var req resetInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.New(http.StatusBadRequest, errors.ErrCodeInvalidInput, "category and available (>=1) are required"))
		return
	}
	if req.Category != "VIP" && req.Category != "Standard" {
		c.Error(errors.New(http.StatusBadRequest, errors.ErrCodeInvalidCategory, "category must be VIP or Standard"))
		return
	}

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("Failed to start transaction for inventory reset", "error", err)
		c.Error(errors.NewInternal(err, "Failed to start reset transaction"))
		return
	}
	defer tx.Rollback() // Safe to call: no-op if committed

	// Purge any orders recorded against this category's tickets first. Without
	// this, a ticket ID that was ever successfully purchased in an earlier test
	// run keeps its row in `orders`, and idx_orders_ticket_id_paid (one Paid
	// order per ticket_id) then rejects every future purchase of that same
	// low-ID ticket once this reset recycles it back to Available.
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM orders
		WHERE ticket_id IN (SELECT id FROM tickets WHERE category = $1)
	`, req.Category); err != nil {
		logger.Error("Failed to purge category orders during inventory reset", "error", err, "category", req.Category)
		c.Error(errors.NewInternal(err, "Failed to purge category orders"))
		return
	}

	// chk_hold_dates requires a non-NULL session_id whenever status = 'Sold',
	// and idx_tickets_session_id_unique requires it to be unique across all
	// tickets, so every wiped row gets its own placeholder session_id
	// (derived from its own id) rather than a shared constant or NULL.
	if _, err := tx.ExecContext(ctx, `
		UPDATE tickets
		SET status = 'Sold', session_id = 'sess_test-reset-' || id, held_at = NULL, expires_at = NULL, updated_at = NOW()
		WHERE category = $1
	`, req.Category); err != nil {
		logger.Error("Failed to clear category tickets during inventory reset", "error", err, "category", req.Category)
		c.Error(errors.NewInternal(err, "Failed to clear category tickets"))
		return
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE tickets
		SET status = 'Available', session_id = NULL, held_at = NULL, expires_at = NULL, updated_at = NOW()
		WHERE id IN (
			SELECT id FROM tickets WHERE category = $1 ORDER BY id LIMIT $2
		)
	`, req.Category, req.Available)
	if err != nil {
		logger.Error("Failed to reopen tickets during inventory reset", "error", err, "category", req.Category)
		c.Error(errors.NewInternal(err, "Failed to reopen tickets"))
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logger.Error("Failed to read rows affected during inventory reset", "error", err)
		c.Error(errors.NewInternal(err, "Failed to verify reset"))
		return
	}
	if int(rowsAffected) != req.Available {
		c.Error(errors.New(http.StatusBadRequest, errors.ErrCodeInvalidInput,
			"category does not have enough tickets to satisfy the requested availability"))
		return
	}

	if err := tx.Commit(); err != nil {
		logger.Error("Failed to commit inventory reset", "error", err)
		c.Error(errors.NewInternal(err, "Failed to commit reset"))
		return
	}

	// Rebuild the Redis available-ticket sets from Postgres so the atomic
	// hold script (which is the source of truth during concurrent reserve
	// requests) agrees with the state we just wrote.
	if err := appredis.SyncInventory(h.db); err != nil {
		logger.Error("Failed to resync Redis inventory after reset", "error", err)
		c.Error(errors.NewInternal(err, "Failed to resync redis inventory"))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"category":  req.Category,
		"available": req.Available,
	}))
}
