package middleware

import (
	"net/http"
	"time"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/session"
	apperrors "github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/errors"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SessionMiddleware intercepts requests, verifies/issues the session_token cookie, and injects session ID into request context.
func SessionMiddleware(jwtSecret string, isProd bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var sessionID string
		var expiresAt time.Time
		var needsNewCookie bool

		// 1. Try to read the cookie
		cookieVal, err := c.Cookie("session_token")
		if err == nil && cookieVal != "" {
			// 2. Parse and verify
			sessionID, expiresAt, err = session.VerifySessionToken(cookieVal, []byte(jwtSecret))
			if err != nil {
				logger.Warn("Invalid or expired session token cookie", "error", err)
				needsNewCookie = true
			}
		} else {
			needsNewCookie = true
		}

		// 3. Generate new session if needed
		if needsNewCookie {
			sessionID = "sess_" + uuid.New().String()
			tokenStr, exp, err := session.SignSessionToken(sessionID, []byte(jwtSecret))
			if err != nil {
				logger.Error("Failed to generate signed session token", "error", err)
				c.Error(apperrors.NewInternal(err, "An unexpected error occurred during session initialization"))
				c.Abort()
				return
			}
			expiresAt = exp

			// Set the cookie
			c.SetSameSite(http.SameSiteStrictMode)
			c.SetCookie("session_token", tokenStr, 1800, "/", "", isProd, true)
		}

		// 4. Inject into context
		c.Set("session_id", sessionID)
		c.Set("session_expires_at", expiresAt)

		c.Next()
	}
}
