package ticket

import (
	"net/http"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/types"
	"github.com/gin-gonic/gin"
)

type TicketController struct {
	service Service
}

// NewTicketController creates a new instance of TicketController.
func NewTicketController(service Service) *TicketController {
	return &TicketController{
		service: service,
	}
}

// GetAvailability handles GET /api/v1/tickets/availability
func (ctrl *TicketController) GetAvailability(c *gin.Context) {
	c.JSON(http.StatusOK, types.NewSuccessResponse(AvailabilityResponse{}))
}

// StreamAvailability handles GET /api/v1/tickets/availability/stream
func (ctrl *TicketController) StreamAvailability(c *gin.Context) {
	c.Status(http.StatusOK)
}

// ReserveTicket handles POST /api/v1/tickets/reserve
func (ctrl *TicketController) ReserveTicket(c *gin.Context) {
	c.JSON(http.StatusOK, types.NewSuccessResponse(ReserveResponse{}))
}

// GetActiveHold handles GET /api/v1/tickets/hold
func (ctrl *TicketController) GetActiveHold(c *gin.Context) {
	c.JSON(http.StatusOK, types.NewSuccessResponse(HoldResponse{}))
}

// CancelHold handles POST /api/v1/tickets/hold/cancel
func (ctrl *TicketController) CancelHold(c *gin.Context) {
	// Placeholder: extract session, call service.CancelHold
	c.JSON(http.StatusOK, types.NewSuccessResponse[any](nil))
}
