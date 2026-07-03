package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/errors"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/types"
	"github.com/gin-gonic/gin"
)

func decodeErrorEnvelope(t *testing.T, body []byte) types.ResponseEnvelope[any] {
	t.Helper()
	var resp types.ResponseEnvelope[any]
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to decode response body: %v (body: %s)", err, body)
	}
	return resp
}

func TestErrorHandlerMiddleware_FourXXNeverRedacted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandlerMiddleware(true)) // isProd = true

	r.GET("/test", func(c *gin.Context) {
		c.Error(apperrors.New(http.StatusBadRequest, apperrors.ErrCodeInvalidInput, "field X is required"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	resp := decodeErrorEnvelope(t, w.Body.Bytes())
	if resp.Success {
		t.Error("expected success=false")
	}
	if resp.Error == nil {
		t.Fatal("expected error field to be populated")
	}
	if resp.Error.Code != apperrors.ErrCodeInvalidInput {
		t.Errorf("expected code %s, got %s", apperrors.ErrCodeInvalidInput, resp.Error.Code)
	}
	if resp.Error.Message != "field X is required" {
		t.Errorf("expected 4xx message to pass through unredacted, got %q", resp.Error.Message)
	}
}

func TestErrorHandlerMiddleware_FiveXXRedactedInProd(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandlerMiddleware(true)) // isProd = true

	r.GET("/test", func(c *gin.Context) {
		c.Error(apperrors.NewInternal(errors.New("raw db connection refused on 10.0.0.5"), "Internal server error"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}

	resp := decodeErrorEnvelope(t, w.Body.Bytes())
	if resp.Error.Message == "raw db connection refused on 10.0.0.5" {
		t.Error("internal error details leaked to client despite isProd=true")
	}
	if resp.Error.Message != "An unexpected error occurred" {
		t.Errorf("expected generic redacted message, got %q", resp.Error.Message)
	}
}

func TestErrorHandlerMiddleware_FiveXXNotRedactedOutsideProd(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandlerMiddleware(false)) // isProd = false

	r.GET("/test", func(c *gin.Context) {
		c.Error(apperrors.NewInternal(errors.New("boom"), "Internal server error detail"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := decodeErrorEnvelope(t, w.Body.Bytes())
	if resp.Error.Message != "Internal server error detail" {
		t.Errorf("expected original message preserved outside prod, got %q", resp.Error.Message)
	}
}

func TestErrorHandlerMiddleware_NoOpWhenNoErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandlerMiddleware(false))

	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != `{"ok":true}` {
		t.Errorf("expected untouched handler body, got %q", w.Body.String())
	}
}

func TestErrorHandlerMiddleware_NoOpWhenResponseAlreadyWritten(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandlerMiddleware(false))

	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusTeapot, "already written")
		c.Error(apperrors.New(http.StatusBadRequest, apperrors.ErrCodeInvalidInput, "should not overwrite"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTeapot {
		t.Fatalf("expected original status 418 to be preserved, got %d", w.Code)
	}
	if w.Body.String() != "already written" {
		t.Errorf("expected original body to be preserved, got %q", w.Body.String())
	}
}
