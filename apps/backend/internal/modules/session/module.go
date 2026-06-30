package session

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Module coordinates the session package setup.
type Module struct {
	Repo       Repository
	Service    Service
	Controller *SessionController
}

// NewModule initializes all layers of the session module.
func NewModule(db *pgxpool.Pool) *Module {
	repo := NewSessionRepository(db)
	svc := NewSessionService(repo)
	ctrl := NewSessionController(svc)

	return &Module{
		Repo:       repo,
		Service:    svc,
		Controller: ctrl,
	}
}

// RegisterRoutes registers the session routes under the given router group.
func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	v1 := rg.Group("/v1")
	{
		v1.POST("/sessions", m.Controller.CreateSession)
		v1.POST("/admin/login", m.Controller.AdminLogin)
	}
}
