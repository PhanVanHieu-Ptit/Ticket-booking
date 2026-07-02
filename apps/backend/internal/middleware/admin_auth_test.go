package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/session"
	"github.com/gin-gonic/gin"
)

func TestAdminAuthMiddleware_MissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	jwtSecret := "test-jwt-secret-key-2026"
	r.Use(ErrorHandlerMiddleware(false))
	r.Use(AdminAuthMiddleware(jwtSecret))

	r.GET("/admin/test", func(c *gin.Context) {
		c.String(http.StatusOK, "admin access granted")
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "ADMIN_UNAUTHORIZED") {
		t.Errorf("expected ADMIN_UNAUTHORIZED error code, got %q", w.Body.String())
	}
}

func TestAdminAuthMiddleware_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	jwtSecret := "test-jwt-secret-key-2026"
	r.Use(ErrorHandlerMiddleware(false))
	r.Use(AdminAuthMiddleware(jwtSecret))

	r.GET("/admin/test", func(c *gin.Context) {
		c.String(http.StatusOK, "admin access granted")
	})

	tokenStr, _, err := session.SignAdminToken([]byte(jwtSecret))
	if err != nil {
		t.Fatalf("failed to sign admin token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "admin access granted" {
		t.Errorf("expected access message, got %q", w.Body.String())
	}
}

func TestAdminAuthMiddleware_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	jwtSecret := "test-jwt-secret-key-2026"
	r.Use(ErrorHandlerMiddleware(false))
	r.Use(AdminAuthMiddleware(jwtSecret))

	r.GET("/admin/test", func(c *gin.Context) {
		c.String(http.StatusOK, "admin access granted")
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-value")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}
