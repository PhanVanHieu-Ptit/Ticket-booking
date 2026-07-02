package payment

import (
	"database/sql"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Module coordinates the payment package setup.
type Module struct {
	Repo       Repository
	Service    Service
	Controller *PaymentController
	rdb        *redis.Client
}

// NewModule initializes all layers of the payment module.
func NewModule(db *sql.DB, rdb *redis.Client) *Module {
	repo := NewPaymentRepository(db)
	svc := NewPaymentService(repo, rdb)
	ctrl := NewPaymentController(svc)

	return &Module{
		Repo:       repo,
		Service:    svc,
		Controller: ctrl,
		rdb:        rdb,
	}
}

// RegisterRoutes registers the payment routes under the given router group.
func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	v1 := rg.Group("/v1")
	{
		// IdempotencyMiddleware guards against duplicate processing when a
		// client retries the exact same checkout request (network lag,
		// double click): retries with the same Idempotency-Key return the
		// cached response instead of re-running the payment logic.
		v1.POST("/payments/checkout", middleware.IdempotencyMiddleware(m.rdb), m.Controller.Checkout)
	}
}
