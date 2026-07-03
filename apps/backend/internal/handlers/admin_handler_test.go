package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/middleware"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/session"
	appDB "github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/db"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"
)

// withTestAdminPasscode upserts the 'admin_passcode' row in admin_configs to a
// known bcrypt hash for the duration of the test, restoring (or removing) the
// prior value afterward so this test doesn't permanently change local/dev
// admin credentials.
func withTestAdminPasscode(t *testing.T, dbConn *sql.DB, plaintext string) {
	t.Helper()
	ctx := context.Background()

	var previousValue sql.NullString
	hadPrevious := true
	err := dbConn.QueryRowContext(ctx, "SELECT value FROM admin_configs WHERE key = 'admin_passcode'").Scan(&previousValue)
	if err == sql.ErrNoRows {
		hadPrevious = false
	} else if err != nil {
		t.Fatalf("failed to read existing admin_passcode: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash test passcode: %v", err)
	}

	_, err = dbConn.ExecContext(ctx, `
		INSERT INTO admin_configs (key, value)
		VALUES ('admin_passcode', $1)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`, string(hash))
	if err != nil {
		t.Fatalf("failed to set test admin_passcode: %v", err)
	}

	t.Cleanup(func() {
		if hadPrevious {
			_, _ = dbConn.ExecContext(context.Background(), `
				UPDATE admin_configs SET value = $1 WHERE key = 'admin_passcode'
			`, previousValue.String)
		} else {
			_, _ = dbConn.ExecContext(context.Background(), `DELETE FROM admin_configs WHERE key = 'admin_passcode'`)
		}
	})
}

// withoutAdminPasscode removes the 'admin_passcode' row entirely for the
// duration of the test, restoring it afterward.
func withoutAdminPasscode(t *testing.T, dbConn *sql.DB) {
	t.Helper()
	ctx := context.Background()

	var previousValue sql.NullString
	hadPrevious := true
	err := dbConn.QueryRowContext(ctx, "SELECT value FROM admin_configs WHERE key = 'admin_passcode'").Scan(&previousValue)
	if err == sql.ErrNoRows {
		hadPrevious = false
	} else if err != nil {
		t.Fatalf("failed to read existing admin_passcode: %v", err)
	}

	_, err = dbConn.ExecContext(ctx, `DELETE FROM admin_configs WHERE key = 'admin_passcode'`)
	if err != nil {
		t.Fatalf("failed to delete admin_passcode: %v", err)
	}

	t.Cleanup(func() {
		if hadPrevious {
			_, _ = dbConn.ExecContext(context.Background(), `
				INSERT INTO admin_configs (key, value) VALUES ('admin_passcode', $1)
				ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
			`, previousValue.String)
		}
	})
}

func setupAdminHandlerTest(t *testing.T) (*sql.DB, *AdminHandler) {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:postgres@localhost:5432/ticket_booking?sslmode=disable"
	}
	dbConn, err := appDB.Init(dbURL)
	if err != nil {
		t.Skipf("PostgreSQL connection failed, skipping integration test: %v", err)
	}
	t.Cleanup(func() { appDB.Close() })

	jwtSecret := []byte("test-jwt-secret-key-2026")
	return dbConn, NewAdminHandler(dbConn, jwtSecret)
}

type apiEnvelope[T any] struct {
	Success bool `json:"success"`
	Data    T    `json:"data"`
	Error   *any `json:"error"`
}

func TestAdminHandler_Login(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dbConn, handler := setupAdminHandlerTest(t)

	r := gin.New()
	r.Use(middleware.ErrorHandlerMiddleware(false))
	r.POST("/login", handler.Login)

	t.Run("correct passcode returns valid admin token", func(t *testing.T) {
		withTestAdminPasscode(t, dbConn, "correct-horse-battery-staple")

		body := `{"passcode": "correct-horse-battery-staple"}`
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp apiEnvelope[struct {
			Token     string `json:"token"`
			ExpiresAt string `json:"expires_at"`
		}]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if resp.Data.Token == "" {
			t.Fatal("expected non-empty token")
		}

		valid, err := session.VerifyAdminToken(resp.Data.Token, []byte("test-jwt-secret-key-2026"))
		if err != nil || !valid {
			t.Errorf("expected issued token to verify as a valid admin token, got valid=%v err=%v", valid, err)
		}
	})

	t.Run("wrong passcode returns 401", func(t *testing.T) {
		withTestAdminPasscode(t, dbConn, "correct-horse-battery-staple")

		body := `{"passcode": "wrong-passcode"}`
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("missing admin_passcode config returns 401 not 500", func(t *testing.T) {
		withoutAdminPasscode(t, dbConn)

		body := `{"passcode": "anything"}`
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 when admin_passcode is unset, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("missing passcode field returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for missing passcode, got %d", w.Code)
		}
	})
}

func TestAdminHandler_GetMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dbConn, handler := setupAdminHandlerTest(t)
	ctx := context.Background()

	r := gin.New()
	r.Use(middleware.ErrorHandlerMiddleware(false))
	r.GET("/metrics", handler.GetMetrics)

	const testCategory = "AdminMetricsTestCat"
	const soldPrice = 120.0

	cleanup := func() {
		_, _ = dbConn.ExecContext(ctx, "DELETE FROM tickets WHERE ticket_code LIKE 'TKT-ADMTEST-%'")
	}
	cleanup()
	t.Cleanup(cleanup)

	// Baseline snapshot before inserting test rows, since GetMetrics aggregates
	// globally over the tickets table (not scoped to our test category).
	baseline := fetchMetrics(t, r)

	// 1 Sold, 1 Holding, 1 Available ticket, all in a category unique to this test.
	mustInsertTicket(t, dbConn, "TKT-ADMTEST-SOLD-1", testCategory, soldPrice, "Sold", sql.NullString{String: "sess_admtest_sold", Valid: true}, sql.NullTime{}, sql.NullTime{})

	heldAt := time.Now()
	expiresAt := heldAt.Add(5 * time.Minute)
	mustInsertTicket(t, dbConn, "TKT-ADMTEST-HOLD-1", testCategory, 80.0, "Holding", sql.NullString{String: "sess_admtest_hold", Valid: true}, sql.NullTime{Time: heldAt, Valid: true}, sql.NullTime{Time: expiresAt, Valid: true})

	mustInsertTicket(t, dbConn, "TKT-ADMTEST-AVAIL-1", testCategory, 50.0, "Available", sql.NullString{}, sql.NullTime{}, sql.NullTime{})

	after := fetchMetrics(t, r)

	if after.Data.TotalTicketsSold-baseline.Data.TotalTicketsSold != 1 {
		t.Errorf("expected total_tickets_sold to increase by 1, went from %d to %d",
			baseline.Data.TotalTicketsSold, after.Data.TotalTicketsSold)
	}
	if diff := after.Data.TotalRevenue - baseline.Data.TotalRevenue; diff < soldPrice-0.001 || diff > soldPrice+0.001 {
		t.Errorf("expected total_revenue to increase by %.2f, got delta %.2f", soldPrice, diff)
	}
	if after.Data.AvailableInventory[testCategory] != 1 {
		t.Errorf("expected 1 available ticket in %s, got %d", testCategory, after.Data.AvailableInventory[testCategory])
	}
	if after.Data.HeldInventory[testCategory] != 1 {
		t.Errorf("expected 1 held ticket in %s, got %d", testCategory, after.Data.HeldInventory[testCategory])
	}
	if after.Data.RemainingInventory[testCategory] != 2 {
		t.Errorf("expected remaining inventory (available+held) of 2 in %s, got %d", testCategory, after.Data.RemainingInventory[testCategory])
	}
}

func TestAdminHandler_GetHolds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dbConn, handler := setupAdminHandlerTest(t)
	ctx := context.Background()

	r := gin.New()
	r.Use(middleware.ErrorHandlerMiddleware(false))
	r.GET("/holds", handler.GetHolds)

	cleanup := func() {
		_, _ = dbConn.ExecContext(ctx, "DELETE FROM tickets WHERE ticket_code LIKE 'TKT-ADMHOLD-%'")
	}
	cleanup()
	t.Cleanup(cleanup)

	// A hold that already expired in the past (seconds_remaining should clamp to 0)
	// and a hold expiring far in the future, so we can check relative ordering.
	pastHeldAt := time.Now().Add(-10 * time.Minute)
	pastExpiresAt := time.Now().Add(-5 * time.Minute)
	mustInsertTicket(t, dbConn, "TKT-ADMHOLD-EXPIRED", "AdminHoldTestCat", 60.0, "Holding",
		sql.NullString{String: "sess_admhold_expired", Valid: true},
		sql.NullTime{Time: pastHeldAt, Valid: true}, sql.NullTime{Time: pastExpiresAt, Valid: true})

	futureHeldAt := time.Now()
	futureExpiresAt := time.Now().Add(10 * time.Minute)
	mustInsertTicket(t, dbConn, "TKT-ADMHOLD-FUTURE", "AdminHoldTestCat", 60.0, "Holding",
		sql.NullString{String: "sess_admhold_future", Valid: true},
		sql.NullTime{Time: futureHeldAt, Valid: true}, sql.NullTime{Time: futureExpiresAt, Valid: true})

	req := httptest.NewRequest(http.MethodGet, "/holds", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp apiEnvelope[[]HoldDetail]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	var expiredIdx, futureIdx = -1, -1
	for i, h := range resp.Data {
		switch h.TicketCode {
		case "TKT-ADMHOLD-EXPIRED":
			expiredIdx = i
			if h.SecondsRemaining != 0 {
				t.Errorf("expected expired hold to have seconds_remaining clamped to 0, got %d", h.SecondsRemaining)
			}
		case "TKT-ADMHOLD-FUTURE":
			futureIdx = i
			if h.SecondsRemaining <= 0 {
				t.Errorf("expected future hold to have positive seconds_remaining, got %d", h.SecondsRemaining)
			}
		}
	}

	if expiredIdx == -1 || futureIdx == -1 {
		t.Fatalf("expected both test holds present in response, expiredIdx=%d futureIdx=%d", expiredIdx, futureIdx)
	}
	if expiredIdx > futureIdx {
		t.Errorf("expected holds ordered by expires_at ASC (expired hold before future hold), got expiredIdx=%d futureIdx=%d", expiredIdx, futureIdx)
	}
}

func fetchMetrics(t *testing.T, r *gin.Engine) apiEnvelope[MetricsResponse] {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from GetMetrics, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp apiEnvelope[MetricsResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse metrics response: %v", err)
	}
	return resp
}

func mustInsertTicket(t *testing.T, dbConn *sql.DB, ticketCode, category string, price float64, status string, sessionID sql.NullString, heldAt, expiresAt sql.NullTime) {
	t.Helper()
	_, err := dbConn.ExecContext(context.Background(), `
		INSERT INTO tickets (ticket_code, category, price, status, session_id, held_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, ticketCode, category, price, status, sessionID, heldAt, expiresAt)
	if err != nil {
		t.Fatalf("failed to insert test ticket %s: %v", ticketCode, err)
	}
}
