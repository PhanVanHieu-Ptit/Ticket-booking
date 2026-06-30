package payment

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Module coordinates the payment package setup.
type Module struct {
	Repo       Repository
	Service    Service
	Controller *PaymentController
}

// NewModule initializes all layers of the payment module.
func NewModule(db *pgxpool.Pool) *Module {
	repo := NewPaymentRepository(db)
	svc := NewPaymentService(repo)
	ctrl := NewPaymentController(svc)

	return &Module{
		Repo:       repo,
		Service:    svc,
		Controller: ctrl,
	}
}

// RegisterRoutes registers the payment routes under the given router group.
func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	v1 := rg.Group("/v1")
	{
		v1.POST("/payments/checkout", m.Controller.Checkout)
	}
}
