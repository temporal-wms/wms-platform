package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/wms-platform/services/billing-service/internal/application"
	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/middleware"
)

// BillingHandler handles HTTP requests for billing
type BillingHandler struct {
	service *application.BillingService
	logger  *logging.Logger
}

// NewBillingHandler creates a new BillingHandler
func NewBillingHandler(service *application.BillingService, logger *logging.Logger) *BillingHandler {
	return &BillingHandler{
		service: service,
		logger:  logger,
	}
}

// RecordActivity handles POST /api/v1/activities
func (h *BillingHandler) RecordActivity(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	var cmd application.RecordActivityCommand
	if appErr := middleware.BindAndValidate(c, &cmd); appErr != nil {
		responder.RespondWithAppError(appErr)
		return
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id":     cmd.SellerID,
		"activity.type": cmd.Type,
	})

	result, err := h.service.RecordActivity(c.Request.Context(), cmd)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			responder.RespondWithAppError(appErr)
		} else {
			responder.RespondInternalError(err)
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": result})
}

// RecordActivities handles POST /api/v1/activities/batch
func (h *BillingHandler) RecordActivities(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	var cmd application.RecordActivitiesCommand
	if appErr := middleware.BindAndValidate(c, &cmd); appErr != nil {
		responder.RespondWithAppError(appErr)
		return
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"activities.count": len(cmd.Activities),
	})

	result, err := h.service.RecordActivities(c.Request.Context(), cmd)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			responder.RespondWithAppError(appErr)
		} else {
			responder.RespondInternalError(err)
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": result})
}

// GetActivity handles GET /api/v1/activities/:activityId
func (h *BillingHandler) GetActivity(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	activityID := c.Param("activityId")

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"activity.id": activityID,
	})

	result, err := h.service.GetActivity(c.Request.Context(), activityID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			responder.RespondWithAppError(appErr)
		} else {
			responder.RespondInternalError(err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// ListActivities handles GET /api/v1/sellers/:sellerId/activities
func (h *BillingHandler) ListActivities(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	sellerID := c.Param("sellerId")
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	pageSize, _ := strconv.ParseInt(c.DefaultQuery("pageSize", "20"), 10, 64)

	query := application.ListActivitiesQuery{
		SellerID: sellerID,
		Page:     page,
		PageSize: pageSize,
	}

	result, err := h.service.ListActivities(c.Request.Context(), query)
	if err != nil {
		responder.RespondInternalError(err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetActivitySummary handles GET /api/v1/sellers/:sellerId/activities/summary
func (h *BillingHandler) GetActivitySummary(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	sellerID := c.Param("sellerId")

	// Parse period
	periodStartStr := c.Query("periodStart")
	periodEndStr := c.Query("periodEnd")

	if periodStartStr == "" || periodEndStr == "" {
		responder.RespondWithAppError(errors.ErrValidation("periodStart and periodEnd are required"))
		return
	}

	periodStart, err := time.Parse(time.RFC3339, periodStartStr)
	if err != nil {
		responder.RespondWithAppError(errors.ErrValidation("invalid periodStart format"))
		return
	}

	periodEnd, err := time.Parse(time.RFC3339, periodEndStr)
	if err != nil {
		responder.RespondWithAppError(errors.ErrValidation("invalid periodEnd format"))
		return
	}

	result, err := h.service.GetActivitySummary(c.Request.Context(), sellerID, periodStart, periodEnd)
	if err != nil {
		responder.RespondInternalError(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// CreateInvoice handles POST /api/v1/invoices
func (h *BillingHandler) CreateInvoice(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	var cmd application.CreateInvoiceCommand
	if appErr := middleware.BindAndValidate(c, &cmd); appErr != nil {
		responder.RespondWithAppError(appErr)
		return
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id":    cmd.SellerID,
		"period.start": cmd.PeriodStart,
		"period.end":   cmd.PeriodEnd,
	})

	result, err := h.service.CreateInvoice(c.Request.Context(), cmd)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			responder.RespondWithAppError(appErr)
		} else {
			responder.RespondInternalError(err)
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": result})
}

// GetInvoice handles GET /api/v1/invoices/:invoiceId
func (h *BillingHandler) GetInvoice(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	invoiceID := c.Param("invoiceId")

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"invoice.id": invoiceID,
	})

	result, err := h.service.GetInvoice(c.Request.Context(), invoiceID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			responder.RespondWithAppError(appErr)
		} else {
			responder.RespondInternalError(err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// ListInvoices handles GET /api/v1/sellers/:sellerId/invoices
func (h *BillingHandler) ListInvoices(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	sellerID := c.Param("sellerId")
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	pageSize, _ := strconv.ParseInt(c.DefaultQuery("pageSize", "20"), 10, 64)

	query := application.ListInvoicesQuery{
		SellerID: sellerID,
		Page:     page,
		PageSize: pageSize,
	}

	if status := c.Query("status"); status != "" {
		query.Status = &status
	}

	result, err := h.service.ListInvoices(c.Request.Context(), query)
	if err != nil {
		responder.RespondInternalError(err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// FinalizeInvoice handles PUT /api/v1/invoices/:invoiceId/finalize
func (h *BillingHandler) FinalizeInvoice(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	invoiceID := c.Param("invoiceId")

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"invoice.id": invoiceID,
	})

	result, err := h.service.FinalizeInvoice(c.Request.Context(), invoiceID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			responder.RespondWithAppError(appErr)
		} else {
			responder.RespondInternalError(err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// MarkInvoicePaid handles PUT /api/v1/invoices/:invoiceId/pay
func (h *BillingHandler) MarkInvoicePaid(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	var req struct {
		PaymentMethod string `json:"paymentMethod" binding:"required"`
		PaymentRef    string `json:"paymentRef"`
	}
	if appErr := middleware.BindAndValidate(c, &req); appErr != nil {
		responder.RespondWithAppError(appErr)
		return
	}

	cmd := application.MarkPaidCommand{
		InvoiceID:     c.Param("invoiceId"),
		PaymentMethod: req.PaymentMethod,
		PaymentRef:    req.PaymentRef,
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"invoice.id":     cmd.InvoiceID,
		"payment.method": cmd.PaymentMethod,
	})

	result, err := h.service.MarkInvoicePaid(c.Request.Context(), cmd)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			responder.RespondWithAppError(appErr)
		} else {
			responder.RespondInternalError(err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// VoidInvoice handles PUT /api/v1/invoices/:invoiceId/void
func (h *BillingHandler) VoidInvoice(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if appErr := middleware.BindAndValidate(c, &req); appErr != nil {
		responder.RespondWithAppError(appErr)
		return
	}

	invoiceID := c.Param("invoiceId")

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"invoice.id": invoiceID,
		"reason":     req.Reason,
	})

	result, err := h.service.VoidInvoice(c.Request.Context(), invoiceID, req.Reason)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			responder.RespondWithAppError(appErr)
		} else {
			responder.RespondInternalError(err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// CalculateFees handles POST /api/v1/fees/calculate
func (h *BillingHandler) CalculateFees(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	var cmd application.CalculateFeesCommand
	if appErr := middleware.BindAndValidate(c, &cmd); appErr != nil {
		responder.RespondWithAppError(appErr)
		return
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id": cmd.SellerID,
	})

	result, err := h.service.CalculateFees(c.Request.Context(), cmd)
	if err != nil {
		responder.RespondInternalError(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// RecordStorage handles POST /api/v1/storage/calculate
func (h *BillingHandler) RecordStorage(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	var cmd application.RecordStorageCommand
	if appErr := middleware.BindAndValidate(c, &cmd); appErr != nil {
		responder.RespondWithAppError(appErr)
		return
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id":   cmd.SellerID,
		"cubic.feet":  cmd.TotalCubicFeet,
	})

	if err := h.service.RecordStorageCalculation(c.Request.Context(), cmd); err != nil {
		responder.RespondInternalError(err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Storage calculation recorded",
	})
}
