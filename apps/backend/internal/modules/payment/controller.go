package payment

import (
	"net/http"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/types"
	"github.com/gin-gonic/gin"
)

type PaymentController struct {
	service Service
}

// NewPaymentController creates a new instance of PaymentController.
func NewPaymentController(service Service) *PaymentController {
	return &PaymentController{
		service: service,
	}
}

// Checkout handles POST /api/v1/payments/checkout
func (ctrl *PaymentController) Checkout(c *gin.Context) {
	c.JSON(http.StatusOK, types.NewSuccessResponse(CheckoutResponse{}))
}
