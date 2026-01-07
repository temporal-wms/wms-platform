package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/wms-platform/services/seller-service/internal/application"
	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/middleware"
)

// SellerHandler handles HTTP requests for seller management
type SellerHandler struct {
	service *application.SellerApplicationService
	logger  *logging.Logger
}

// NewSellerHandler creates a new SellerHandler
func NewSellerHandler(service *application.SellerApplicationService, logger *logging.Logger) *SellerHandler {
	return &SellerHandler{
		service: service,
		logger:  logger,
	}
}

// CreateSeller handles POST /api/v1/sellers
func (h *SellerHandler) CreateSeller(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	var cmd application.CreateSellerCommand
	if appErr := middleware.BindAndValidate(c, &cmd); appErr != nil {
		responder.RespondWithAppError(appErr)
		return
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"tenant.id":     cmd.TenantID,
		"company.name":  cmd.CompanyName,
		"billing.cycle": cmd.BillingCycle,
	})

	result, err := h.service.CreateSeller(c.Request.Context(), cmd)
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

// GetSeller handles GET /api/v1/sellers/:sellerId
func (h *SellerHandler) GetSeller(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	query := application.GetSellerQuery{
		SellerID: c.Param("sellerId"),
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id": query.SellerID,
	})

	result, err := h.service.GetSeller(c.Request.Context(), query)
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

// ListSellers handles GET /api/v1/sellers
func (h *SellerHandler) ListSellers(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	pageSize, _ := strconv.ParseInt(c.DefaultQuery("pageSize", "20"), 10, 64)

	query := application.ListSellersQuery{
		Page:     page,
		PageSize: pageSize,
	}

	if tenantID := c.Query("tenantId"); tenantID != "" {
		query.TenantID = &tenantID
	}
	if status := c.Query("status"); status != "" {
		query.Status = &status
	}
	if facilityID := c.Query("facilityId"); facilityID != "" {
		query.FacilityID = &facilityID
	}
	if hasChannel := c.Query("hasChannel"); hasChannel != "" {
		query.HasChannel = &hasChannel
	}

	result, err := h.service.ListSellers(c.Request.Context(), query)
	if err != nil {
		responder.RespondInternalError(err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// ActivateSeller handles PUT /api/v1/sellers/:sellerId/activate
func (h *SellerHandler) ActivateSeller(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	cmd := application.ActivateSellerCommand{
		SellerID: c.Param("sellerId"),
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id": cmd.SellerID,
	})

	result, err := h.service.ActivateSeller(c.Request.Context(), cmd)
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

// SuspendSeller handles PUT /api/v1/sellers/:sellerId/suspend
func (h *SellerHandler) SuspendSeller(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if appErr := middleware.BindAndValidate(c, &req); appErr != nil {
		responder.RespondWithAppError(appErr)
		return
	}

	cmd := application.SuspendSellerCommand{
		SellerID: c.Param("sellerId"),
		Reason:   req.Reason,
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id":      cmd.SellerID,
		"suspend.reason": cmd.Reason,
	})

	result, err := h.service.SuspendSeller(c.Request.Context(), cmd)
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

// CloseSeller handles PUT /api/v1/sellers/:sellerId/close
func (h *SellerHandler) CloseSeller(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if appErr := middleware.BindAndValidate(c, &req); appErr != nil {
		responder.RespondWithAppError(appErr)
		return
	}

	cmd := application.CloseSellerCommand{
		SellerID: c.Param("sellerId"),
		Reason:   req.Reason,
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id":    cmd.SellerID,
		"close.reason": cmd.Reason,
	})

	result, err := h.service.CloseSeller(c.Request.Context(), cmd)
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

// AssignFacility handles POST /api/v1/sellers/:sellerId/facilities
func (h *SellerHandler) AssignFacility(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	var req struct {
		FacilityID     string   `json:"facilityId" binding:"required"`
		FacilityName   string   `json:"facilityName" binding:"required"`
		WarehouseIDs   []string `json:"warehouseIds"`
		AllocatedSpace float64  `json:"allocatedSpace"`
		IsDefault      bool     `json:"isDefault"`
	}
	if appErr := middleware.BindAndValidate(c, &req); appErr != nil {
		responder.RespondWithAppError(appErr)
		return
	}

	cmd := application.AssignFacilityCommand{
		SellerID:       c.Param("sellerId"),
		FacilityID:     req.FacilityID,
		FacilityName:   req.FacilityName,
		WarehouseIDs:   req.WarehouseIDs,
		AllocatedSpace: req.AllocatedSpace,
		IsDefault:      req.IsDefault,
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id":   cmd.SellerID,
		"facility.id": cmd.FacilityID,
	})

	result, err := h.service.AssignFacility(c.Request.Context(), cmd)
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

// RemoveFacility handles DELETE /api/v1/sellers/:sellerId/facilities/:facilityId
func (h *SellerHandler) RemoveFacility(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	cmd := application.RemoveFacilityCommand{
		SellerID:   c.Param("sellerId"),
		FacilityID: c.Param("facilityId"),
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id":   cmd.SellerID,
		"facility.id": cmd.FacilityID,
	})

	result, err := h.service.RemoveFacility(c.Request.Context(), cmd)
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

// UpdateFeeSchedule handles PUT /api/v1/sellers/:sellerId/fee-schedule
func (h *SellerHandler) UpdateFeeSchedule(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	var cmd application.UpdateFeeScheduleCommand
	if appErr := middleware.BindAndValidate(c, &cmd); appErr != nil {
		responder.RespondWithAppError(appErr)
		return
	}
	cmd.SellerID = c.Param("sellerId")

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id": cmd.SellerID,
	})

	result, err := h.service.UpdateFeeSchedule(c.Request.Context(), cmd)
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

// ConnectChannel handles POST /api/v1/sellers/:sellerId/integrations
func (h *SellerHandler) ConnectChannel(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	var cmd application.ConnectChannelCommand
	if appErr := middleware.BindAndValidate(c, &cmd); appErr != nil {
		responder.RespondWithAppError(appErr)
		return
	}
	cmd.SellerID = c.Param("sellerId")

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id":    cmd.SellerID,
		"channel.type": cmd.ChannelType,
		"store.name":   cmd.StoreName,
	})

	result, err := h.service.ConnectChannel(c.Request.Context(), cmd)
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

// DisconnectChannel handles DELETE /api/v1/sellers/:sellerId/integrations/:channelId
func (h *SellerHandler) DisconnectChannel(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	cmd := application.DisconnectChannelCommand{
		SellerID:  c.Param("sellerId"),
		ChannelID: c.Param("channelId"),
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id":  cmd.SellerID,
		"channel.id": cmd.ChannelID,
	})

	result, err := h.service.DisconnectChannel(c.Request.Context(), cmd)
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

// GenerateAPIKey handles POST /api/v1/sellers/:sellerId/api-keys
func (h *SellerHandler) GenerateAPIKey(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	var cmd application.GenerateAPIKeyCommand
	if appErr := middleware.BindAndValidate(c, &cmd); appErr != nil {
		responder.RespondWithAppError(appErr)
		return
	}
	cmd.SellerID = c.Param("sellerId")

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id": cmd.SellerID,
		"key.name":  cmd.Name,
	})

	result, err := h.service.GenerateAPIKey(c.Request.Context(), cmd)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			responder.RespondWithAppError(appErr)
		} else {
			responder.RespondInternalError(err)
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data":    result,
		"message": "Store this API key securely. It will not be shown again.",
	})
}

// ListAPIKeys handles GET /api/v1/sellers/:sellerId/api-keys
func (h *SellerHandler) ListAPIKeys(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	sellerID := c.Param("sellerId")

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id": sellerID,
	})

	result, err := h.service.GetAPIKeys(c.Request.Context(), sellerID)
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

// RevokeAPIKey handles DELETE /api/v1/sellers/:sellerId/api-keys/:keyId
func (h *SellerHandler) RevokeAPIKey(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	cmd := application.RevokeAPIKeyCommand{
		SellerID: c.Param("sellerId"),
		KeyID:    c.Param("keyId"),
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id": cmd.SellerID,
		"key.id":    cmd.KeyID,
	})

	if err := h.service.RevokeAPIKey(c.Request.Context(), cmd); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			responder.RespondWithAppError(appErr)
		} else {
			responder.RespondInternalError(err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "API key revoked successfully",
	})
}

// SearchSellers handles GET /api/v1/sellers/search
func (h *SellerHandler) SearchSellers(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	query := c.Query("q")
	if query == "" {
		responder.RespondWithAppError(errors.ErrValidation("search query 'q' is required"))
		return
	}

	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	pageSize, _ := strconv.ParseInt(c.DefaultQuery("pageSize", "20"), 10, 64)

	result, err := h.service.SearchSellers(c.Request.Context(), application.SearchSellersQuery{
		Query:    query,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		responder.RespondInternalError(err)
		return
	}

	c.JSON(http.StatusOK, result)
}
