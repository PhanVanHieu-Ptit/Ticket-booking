package payment

import (
	"net/http"

	appErrors "github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/errors"
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
	var req CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(
			appErrors.ErrCodeInvalidInput,
			"Invalid request payload",
			nil,
		))
		return
	}

	sessionIDVal, exists := c.Get("session_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(
			appErrors.ErrCodeSessionRequired,
			"Session token is required",
			nil,
		))
		return
	}
	sessionID := sessionIDVal.(string)

	order, err := ctrl.service.Checkout(
		c.Request.Context(),
		sessionID,
		req.TicketID,
		req.Email,
		req.CardHolderName,
		req.SimulateStatus,
	)

	if err != nil {
		appErr := appErrors.FromError(err)
		c.JSON(appErr.HTTPStatus, types.NewErrorResponse(
			appErr.Code,
			appErr.Message,
			appErr.Details,
		))
		return
	}

	resp := CheckoutResponse{
		OrderID:          order.ID,
		TicketID:         order.TicketID,
		Amount:           order.Amount,
		PaymentReference: order.PaymentReference,
		PaidAt:           order.CreatedAt,
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse(resp))
}
