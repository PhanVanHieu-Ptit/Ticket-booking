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

// InitializeSession returns the current session ID, its raw signed token, and expiration time.
// Note: Session middleware automatically intercepts the request, generates a session, and sets the cookie.
//
// The raw token is included in the body (not just the session_token cookie)
// so clients can authenticate the ticket availability SSE stream via a
// ?session_token= query param when the cookie itself never reaches the
// browser's jar — e.g. private/incognito tabs block third-party cookies
// outright when the frontend (Vercel) and backend (Render) are on different origins.
func (h *SessionHandler) InitializeSession(c *gin.Context) {
	sessionID, ok := c.Get("session_id")
	expiresAt, okExp := c.Get("session_expires_at")
	sessionToken, okToken := c.Get("session_token")

	if !ok || !okExp || !okToken {
		c.Error(errors.New(http.StatusInternalServerError, errors.ErrCodeInternal, "Session not initialized in request context"))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"session_id":    sessionID.(string),
		"session_token": sessionToken.(string),
		"expires_at":    expiresAt.(time.Time).Format(time.RFC3339),
	}))
}
