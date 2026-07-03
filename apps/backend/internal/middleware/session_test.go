package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/session"
	"github.com/gin-gonic/gin"
)

func TestSessionMiddleware_NewSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	jwtSecret := "test-jwt-secret-key-2026"
	r.Use(SessionMiddleware(jwtSecret, false, false))

	r.GET("/test", func(c *gin.Context) {
		sessionID, exists := c.Get("session_id")
		if !exists {
			c.String(http.StatusInternalServerError, "session_id missing")
			return
		}
		c.String(http.StatusOK, sessionID.(string))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	sessionID := w.Body.String()
	if !strings.HasPrefix(sessionID, "sess_") {
		t.Errorf("expected session_id to have prefix 'sess_', got %q", sessionID)
	}

	cookieHeader := w.Header().Get("Set-Cookie")
	if !strings.Contains(cookieHeader, "session_token=") {
		t.Errorf("expected Set-Cookie header with session_token, got %q", cookieHeader)
	}
	if !strings.Contains(cookieHeader, "HttpOnly") {
		t.Errorf("expected HttpOnly cookie, got %q", cookieHeader)
	}
}

func TestSessionMiddleware_ExistingSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	jwtSecret := "test-jwt-secret-key-2026"
	r.Use(SessionMiddleware(jwtSecret, false, false))

	r.GET("/test", func(c *gin.Context) {
		sessionID, _ := c.Get("session_id")
		c.String(http.StatusOK, sessionID.(string))
	})

	// Generate a valid token
	sessionID := "sess_valid_123"
	tokenStr, _, err := session.SignSessionToken(sessionID, []byte(jwtSecret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: tokenStr,
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	returnedID := w.Body.String()
	if returnedID != sessionID {
		t.Errorf("expected returned session ID %q, got %q", sessionID, returnedID)
	}

	// Since the token is valid, it shouldn't set a new cookie
	cookieHeader := w.Header().Get("Set-Cookie")
	if cookieHeader != "" {
		t.Errorf("expected no Set-Cookie header for valid existing session, got %q", cookieHeader)
	}
}
