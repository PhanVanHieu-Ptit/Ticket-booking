package admin

import (
	"net/http"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/types"
	"github.com/gin-gonic/gin"
)

type AdminController struct {
	service Service
}

// NewAdminController creates a new instance of AdminController.
func NewAdminController(service Service) *AdminController {
	return &AdminController{
		service: service,
	}
}

// GetSalesMetrics handles GET /api/v1/admin/metrics
func (ctrl *AdminController) GetSalesMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, types.NewSuccessResponse(MetricsResponse{}))
}

// GetActiveHolds handles GET /api/v1/admin/holds
func (ctrl *AdminController) GetActiveHolds(c *gin.Context) {
	c.JSON(http.StatusOK, types.NewSuccessResponse(HoldsResponse{}))
}
