package ticket

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Module coordinates the ticket package setup.
type Module struct {
	Repo       Repository
	Service    Service
	Controller *TicketController
}

// NewModule initializes all layers of the ticket module.
func NewModule(db *pgxpool.Pool, redis *redis.Client) *Module {
	repo := NewTicketRepository(db, redis)
	svc := NewTicketService(repo)
	ctrl := NewTicketController(svc)

	return &Module{
		Repo:       repo,
		Service:    svc,
		Controller: ctrl,
	}
}

// RegisterRoutes registers the ticket routes under the given router group.
func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	v1 := rg.Group("/v1")
	{
		v1.GET("/tickets/availability", m.Controller.GetAvailability)
		v1.GET("/tickets/availability/stream", m.Controller.StreamAvailability)
		v1.POST("/tickets/reserve", m.Controller.ReserveTicket)
		v1.GET("/tickets/hold", m.Controller.GetActiveHold)
		v1.POST("/tickets/hold/cancel", m.Controller.CancelHold)
	}
}
