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
// crossSite indicates the frontend is deployed on a different origin than this backend
// (e.g. Vercel + Render), which requires SameSite=None and Secure for the cookie to be sent at all.
func SessionMiddleware(jwtSecret string, isProd bool, crossSite bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var sessionID string
		var expiresAt time.Time
		var tokenStr string
		var needsNewCookie bool

		// 1. Try to read the cookie
		cookieVal, err := c.Cookie("session_token")
		if err == nil && cookieVal != "" {
			// 2. Parse and verify
			sessionID, expiresAt, err = session.VerifySessionToken(cookieVal, []byte(jwtSecret))
			if err != nil {
				logger.Warn("Invalid or expired session token cookie", "error", err)
				needsNewCookie = true
			} else {
				tokenStr = cookieVal
			}
		} else {
			needsNewCookie = true
		}

		// 3. Generate new session if needed
		if needsNewCookie {
			sessionID = "sess_" + uuid.New().String()
			var exp time.Time
			tokenStr, exp, err = session.SignSessionToken(sessionID, []byte(jwtSecret))
			if err != nil {
				logger.Error("Failed to generate signed session token", "error", err)
				c.Error(apperrors.NewInternal(err, "An unexpected error occurred during session initialization"))
				c.Abort()
				return
			}
			expiresAt = exp

			// Set the cookie. SameSite=None requires Secure regardless of isProd,
			// since browsers reject non-Secure SameSite=None cookies outright.
			if crossSite {
				c.SetSameSite(http.SameSiteNoneMode)
				c.SetCookie("session_token", tokenStr, 1800, "/", "", true, true)
			} else {
				c.SetSameSite(http.SameSiteStrictMode)
				c.SetCookie("session_token", tokenStr, 1800, "/", "", isProd, true)
			}
		}

		// 4. Inject into context. session_token is exposed (not just set as a
		// cookie) so callers like /api/v1/sessions can hand it back in the
		// response body: browsers that block third-party cookies (e.g.
		// private/incognito tabs, when frontend and backend are on different
		// origins) never store the cookie, so EventSource connections must be
		// able to authenticate via an explicit ?session_token= query param instead.
		c.Set("session_id", sessionID)
		c.Set("session_expires_at", expiresAt)
		c.Set("session_token", tokenStr)

		c.Next()
	}
}
