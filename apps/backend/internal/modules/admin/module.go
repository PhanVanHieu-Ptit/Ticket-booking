package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Module coordinates the admin package setup.
type Module struct {
	Repo       Repository
	Service    Service
	Controller *AdminController
}

// NewModule initializes all layers of the admin module.
func NewModule(db *pgxpool.Pool) *Module {
	repo := NewAdminRepository(db)
	svc := NewAdminService(repo)
	ctrl := NewAdminController(svc)

	return &Module{
		Repo:       repo,
		Service:    svc,
		Controller: ctrl,
	}
}

// RegisterRoutes registers the admin routes under the given router group.
func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	v1 := rg.Group("/v1")
	{
		v1.GET("/admin/metrics", m.Controller.GetSalesMetrics)
		v1.GET("/admin/holds", m.Controller.GetActiveHolds)
	}
}
