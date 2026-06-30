package session

import (
	"net/http"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/types"
	"github.com/gin-gonic/gin"
)

type SessionController struct {
	service Service
}

// NewSessionController creates a new instance of SessionController.
func NewSessionController(service Service) *SessionController {
	return &SessionController{
		service: service,
	}
}

// CreateSession handles POST /api/v1/sessions
func (ctrl *SessionController) CreateSession(c *gin.Context) {
	c.JSON(http.StatusOK, types.NewSuccessResponse(CreateSessionResponse{}))
}

// AdminLogin handles POST /api/v1/admin/login
func (ctrl *SessionController) AdminLogin(c *gin.Context) {
	c.JSON(http.StatusOK, types.NewSuccessResponse(AdminLoginResponse{}))
}
