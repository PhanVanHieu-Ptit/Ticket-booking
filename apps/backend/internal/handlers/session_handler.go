package handlers

import (
	"net/http"
	"time"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/errors"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/types"
	"github.com/gin-gonic/gin"
)

type SessionHandler struct{}

// NewSessionHandler creates a new instance of SessionHandler.
func NewSessionHandler() *SessionHandler {
	return &SessionHandler{}
}

// InitializeSession returns the current session ID and expiration time.
// Note: Session middleware automatically intercepts the request, generates a session, and sets the cookie.
func (h *SessionHandler) InitializeSession(c *gin.Context) {
	sessionID, ok := c.Get("session_id")
	expiresAt, okExp := c.Get("session_expires_at")

	if !ok || !okExp {
		c.Error(errors.New(http.StatusInternalServerError, errors.ErrCodeInternal, "Session not initialized in request context"))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"session_id": sessionID.(string),
		"expires_at": expiresAt.(time.Time).Format(time.RFC3339),
	}))
}
