package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/metrics"
	"github.com/wms-platform/shared/pkg/middleware"

	"github.com/wms-platform/services/seller-portal/internal/application"
	"github.com/wms-platform/services/seller-portal/internal/domain"
)

// DashboardHandler handles dashboard HTTP requests
type DashboardHandler struct {
	service *application.DashboardService
	logger  *logging.Logger
	metrics *metrics.Metrics
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(service *application.DashboardService, logger *logging.Logger, m *metrics.Metrics) *DashboardHandler {
	return &DashboardHandler{
		service: service,
		logger:  logger,
		metrics: m,
	}
}

// RegisterRoutes registers the dashboard routes
func (h *DashboardHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Dashboard
	r.GET("/dashboard/summary", h.GetDashboardSummary)

	// Orders
	r.GET("/orders", h.GetOrders)
	r.GET("/orders/:id", h.GetOrder)

	// Inventory
	r.GET("/inventory", h.GetInventory)

	// Billing
	r.GET("/billing/invoices", h.GetInvoices)
	r.GET("/billing/invoices/:id", h.GetInvoice)

	// Channels/Integrations
	r.GET("/integrations", h.GetChannels)
	r.POST("/integrations", h.ConnectChannel)
	r.DELETE("/integrations/:id", h.DisconnectChannel)
	r.POST("/integrations/:id/sync", h.SyncChannel)

	// API Keys
	r.GET("/api-keys", h.GetAPIKeys)
	r.POST("/api-keys", h.GenerateAPIKey)
	r.DELETE("/api-keys/:id", h.RevokeAPIKey)
}

// GetDashboardSummary handles GET /dashboard/summary
func (h *DashboardHandler) GetDashboardSummary(c *gin.Context) {
	start := time.Now()
	sellerID := c.GetHeader("X-WMS-Seller-ID")
	if sellerID == "" {
		h.logger.Warn("Missing seller ID header")
		c.JSON(http.StatusBadRequest, gin.H{"error": "seller ID required"})
		return
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id": sellerID,
		"operation": "get_dashboard_summary",
	})

	// Parse period
	periodType := c.DefaultQuery("period", "today")
	period := h.parsePeriod(periodType, c.Query("startDate"), c.Query("endDate"))

	summary, err := h.service.GetDashboardSummary(c.Request.Context(), sellerID, period)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get dashboard summary", "seller_id", sellerID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Record dashboard assembly duration using HTTP request duration metric
	h.metrics.HTTPRequestDuration.WithLabelValues(
		"seller-portal",
		"GET",
		"/dashboard/summary",
	).Observe(time.Since(start).Seconds())

	c.JSON(http.StatusOK, summary)
}

func (h *DashboardHandler) parsePeriod(periodType, startDateStr, endDateStr string) domain.Period {
	now := time.Now()
	period := domain.Period{Type: periodType}

	switch periodType {
	case "today":
		period.Start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		period.End = period.Start.Add(24 * time.Hour)
	case "week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		period.Start = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		period.End = period.Start.Add(7 * 24 * time.Hour)
	case "month":
		period.Start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		period.End = period.Start.AddDate(0, 1, 0)
	case "custom":
		if startDate, err := time.Parse("2006-01-02", startDateStr); err == nil {
			period.Start = startDate
		} else {
			period.Start = now.AddDate(0, 0, -30)
		}
		if endDate, err := time.Parse("2006-01-02", endDateStr); err == nil {
			period.End = endDate.Add(24 * time.Hour)
		} else {
			period.End = now.Add(24 * time.Hour)
		}
	default:
		period.Start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		period.End = period.Start.Add(24 * time.Hour)
	}

	return period
}

// GetOrders handles GET /orders
func (h *DashboardHandler) GetOrders(c *gin.Context) {
	sellerID := c.GetHeader("X-WMS-Seller-ID")
	if sellerID == "" {
		h.logger.Warn("Missing seller ID header")
		c.JSON(http.StatusBadRequest, gin.H{"error": "seller ID required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id": sellerID,
		"page":      page,
		"page_size": pageSize,
		"operation": "get_orders",
	})

	filter := domain.OrderFilter{
		SellerID:  sellerID,
		ChannelID: c.Query("channelId"),
		Search:    c.Query("search"),
		Page:      page,
		PageSize:  pageSize,
		SortBy:    c.DefaultQuery("sortBy", "createdAt"),
		SortOrder: c.DefaultQuery("sortOrder", "desc"),
	}

	if status := c.Query("status"); status != "" {
		filter.Status = []string{status}
	}

	orders, total, err := h.service.GetOrders(c.Request.Context(), filter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get orders", "seller_id", sellerID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"total":  total,
		"page":   page,
		"size":   pageSize,
	})
}

// GetOrder handles GET /orders/:id
func (h *DashboardHandler) GetOrder(c *gin.Context) {
	sellerID := c.GetHeader("X-WMS-Seller-ID")
	if sellerID == "" {
		h.logger.Warn("Missing seller ID header")
		c.JSON(http.StatusBadRequest, gin.H{"error": "seller ID required"})
		return
	}

	orderID := c.Param("id")
	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id": sellerID,
		"order.id":  orderID,
		"operation": "get_order",
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Order details endpoint",
	})
}

// GetInventory handles GET /inventory
func (h *DashboardHandler) GetInventory(c *gin.Context) {
	sellerID := c.GetHeader("X-WMS-Seller-ID")
	if sellerID == "" {
		h.logger.Warn("Missing seller ID header")
		c.JSON(http.StatusBadRequest, gin.H{"error": "seller ID required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id": sellerID,
		"page":      page,
		"page_size": pageSize,
		"operation": "get_inventory",
	})

	filter := domain.InventoryFilter{
		SellerID:    sellerID,
		WarehouseID: c.Query("warehouseId"),
		Search:      c.Query("search"),
		Page:        page,
		PageSize:    pageSize,
		SortBy:      c.DefaultQuery("sortBy", "sku"),
		SortOrder:   c.DefaultQuery("sortOrder", "asc"),
	}

	if status := c.Query("status"); status != "" {
		filter.Status = []string{status}
	}

	inventory, total, err := h.service.GetInventory(c.Request.Context(), filter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get inventory", "seller_id", sellerID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"inventory": inventory,
		"total":     total,
		"page":      page,
		"size":      pageSize,
	})
}

// GetInvoices handles GET /billing/invoices
func (h *DashboardHandler) GetInvoices(c *gin.Context) {
	sellerID := c.GetHeader("X-WMS-Seller-ID")
	if sellerID == "" {
		h.logger.Warn("Missing seller ID header")
		c.JSON(http.StatusBadRequest, gin.H{"error": "seller ID required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id": sellerID,
		"page":      page,
		"page_size": pageSize,
		"operation": "get_invoices",
	})

	filter := domain.InvoiceFilter{
		SellerID: sellerID,
		Page:     page,
		PageSize: pageSize,
	}

	if status := c.Query("status"); status != "" {
		filter.Status = []string{status}
	}

	invoices, total, err := h.service.GetInvoices(c.Request.Context(), filter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get invoices", "seller_id", sellerID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"invoices": invoices,
		"total":    total,
		"page":     page,
		"size":     pageSize,
	})
}

// GetInvoice handles GET /billing/invoices/:id
func (h *DashboardHandler) GetInvoice(c *gin.Context) {
	sellerID := c.GetHeader("X-WMS-Seller-ID")
	if sellerID == "" {
		h.logger.Warn("Missing seller ID header")
		c.JSON(http.StatusBadRequest, gin.H{"error": "seller ID required"})
		return
	}

	invoiceID := c.Param("id")
	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id":  sellerID,
		"invoice.id": invoiceID,
		"operation":  "get_invoice",
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Invoice details endpoint",
	})
}

// GetChannels handles GET /integrations
func (h *DashboardHandler) GetChannels(c *gin.Context) {
	sellerID := c.GetHeader("X-WMS-Seller-ID")
	if sellerID == "" {
		h.logger.Warn("Missing seller ID header")
		c.JSON(http.StatusBadRequest, gin.H{"error": "seller ID required"})
		return
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id": sellerID,
		"operation": "get_channels",
	})

	channels, err := h.service.GetChannels(c.Request.Context(), sellerID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get channels", "seller_id", sellerID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"channels": channels,
		"total":    len(channels),
	})
}

// ConnectChannelRequest represents a channel connection request
type ConnectChannelRequest struct {
	Type        string                 `json:"type" binding:"required"`
	Name        string                 `json:"name" binding:"required"`
	Credentials map[string]interface{} `json:"credentials" binding:"required"`
	WebhookURL  string                 `json:"webhookUrl"`
}

// ConnectChannel handles POST /integrations
func (h *DashboardHandler) ConnectChannel(c *gin.Context) {
	sellerID := c.GetHeader("X-WMS-Seller-ID")
	if sellerID == "" {
		h.logger.Warn("Missing seller ID header")
		c.JSON(http.StatusBadRequest, gin.H{"error": "seller ID required"})
		return
	}

	var req ConnectChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid connect channel request", "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id":    sellerID,
		"channel.type": req.Type,
		"channel.name": req.Name,
		"operation":    "connect_channel",
	})

	channelReq := map[string]interface{}{
		"type":        req.Type,
		"name":        req.Name,
		"credentials": req.Credentials,
		"webhookUrl":  req.WebhookURL,
	}

	result, err := h.service.ConnectChannel(c.Request.Context(), sellerID, channelReq)
	if err != nil {
		h.logger.WithError(err).Error("Failed to connect channel",
			"seller_id", sellerID,
			"channel_type", req.Type,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Channel connected",
		"seller_id", sellerID,
		"channel_type", req.Type,
	)
	c.JSON(http.StatusCreated, result)
}

// DisconnectChannel handles DELETE /integrations/:id
func (h *DashboardHandler) DisconnectChannel(c *gin.Context) {
	sellerID := c.GetHeader("X-WMS-Seller-ID")
	if sellerID == "" {
		h.logger.Warn("Missing seller ID header")
		c.JSON(http.StatusBadRequest, gin.H{"error": "seller ID required"})
		return
	}

	channelID := c.Param("id")

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id":  sellerID,
		"channel.id": channelID,
		"operation":  "disconnect_channel",
	})

	if err := h.service.DisconnectChannel(c.Request.Context(), channelID); err != nil {
		h.logger.WithError(err).Error("Failed to disconnect channel",
			"seller_id", sellerID,
			"channel_id", channelID,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Channel disconnected",
		"seller_id", sellerID,
		"channel_id", channelID,
	)
	c.JSON(http.StatusOK, gin.H{"message": "Channel disconnected"})
}

// SyncChannel handles POST /integrations/:id/sync
func (h *DashboardHandler) SyncChannel(c *gin.Context) {
	sellerID := c.GetHeader("X-WMS-Seller-ID")
	if sellerID == "" {
		h.logger.Warn("Missing seller ID header")
		c.JSON(http.StatusBadRequest, gin.H{"error": "seller ID required"})
		return
	}

	channelID := c.Param("id")
	syncType := c.DefaultQuery("type", "orders")

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id":  sellerID,
		"channel.id": channelID,
		"sync.type":  syncType,
		"operation":  "sync_channel",
	})

	result, err := h.service.SyncChannel(c.Request.Context(), channelID, syncType)
	if err != nil {
		h.logger.WithError(err).Error("Failed to sync channel",
			"seller_id", sellerID,
			"channel_id", channelID,
			"sync_type", syncType,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Channel sync started",
		"seller_id", sellerID,
		"channel_id", channelID,
		"sync_type", syncType,
	)
	c.JSON(http.StatusOK, result)
}

// GetAPIKeys handles GET /api-keys
func (h *DashboardHandler) GetAPIKeys(c *gin.Context) {
	sellerID := c.GetHeader("X-WMS-Seller-ID")
	if sellerID == "" {
		h.logger.Warn("Missing seller ID header")
		c.JSON(http.StatusBadRequest, gin.H{"error": "seller ID required"})
		return
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id": sellerID,
		"operation": "get_api_keys",
	})

	keys, err := h.service.GetAPIKeys(c.Request.Context(), sellerID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get API keys", "seller_id", sellerID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"apiKeys": keys,
		"total":   len(keys),
	})
}

// GenerateAPIKeyRequest represents an API key generation request
type GenerateAPIKeyRequest struct {
	Name        string   `json:"name" binding:"required"`
	Permissions []string `json:"permissions"`
}

// GenerateAPIKey handles POST /api-keys
func (h *DashboardHandler) GenerateAPIKey(c *gin.Context) {
	sellerID := c.GetHeader("X-WMS-Seller-ID")
	if sellerID == "" {
		h.logger.Warn("Missing seller ID header")
		c.JSON(http.StatusBadRequest, gin.H{"error": "seller ID required"})
		return
	}

	var req GenerateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid generate API key request", "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id":   sellerID,
		"api_key.name": req.Name,
		"operation":   "generate_api_key",
	})

	result, err := h.service.GenerateAPIKey(c.Request.Context(), sellerID, req.Name, req.Permissions)
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate API key", "seller_id", sellerID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("API key generated",
		"seller_id", sellerID,
		"key_name", req.Name,
	)
	c.JSON(http.StatusCreated, result)
}

// RevokeAPIKey handles DELETE /api-keys/:id
func (h *DashboardHandler) RevokeAPIKey(c *gin.Context) {
	sellerID := c.GetHeader("X-WMS-Seller-ID")
	if sellerID == "" {
		h.logger.Warn("Missing seller ID header")
		c.JSON(http.StatusBadRequest, gin.H{"error": "seller ID required"})
		return
	}

	keyID := c.Param("id")

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id":  sellerID,
		"api_key.id": keyID,
		"operation":  "revoke_api_key",
	})

	if err := h.service.RevokeAPIKey(c.Request.Context(), sellerID, keyID); err != nil {
		h.logger.WithError(err).Error("Failed to revoke API key",
			"seller_id", sellerID,
			"key_id", keyID,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("API key revoked",
		"seller_id", sellerID,
		"key_id", keyID,
	)
	c.JSON(http.StatusOK, gin.H{"message": "API key revoked"})
}
