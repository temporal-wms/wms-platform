package handlers

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/middleware"

	"github.com/wms-platform/services/channel-service/internal/application"
)

// ChannelMetrics interface for channel-specific metrics
type ChannelMetrics interface {
	RecordSyncOperation(channel, syncType, status string, duration time.Duration)
	RecordOrdersImported(channel string, count int)
	RecordAPILatency(channel, operation, status string, duration time.Duration)
	RecordWebhookReceived(channel, topic, status string)
}

// ChannelHandler handles channel HTTP requests
type ChannelHandler struct {
	service *application.ChannelService
	logger  *logging.Logger
	metrics ChannelMetrics
}

// NewChannelHandler creates a new channel handler
func NewChannelHandler(service *application.ChannelService, logger *logging.Logger, metrics ChannelMetrics) *ChannelHandler {
	return &ChannelHandler{
		service: service,
		logger:  logger,
		metrics: metrics,
	}
}

// RegisterRoutes registers the channel routes
func (h *ChannelHandler) RegisterRoutes(r *gin.RouterGroup) {
	channels := r.Group("/channels")
	{
		channels.POST("", h.ConnectChannel)
		channels.GET("/:id", h.GetChannel)
		channels.PUT("/:id", h.UpdateChannel)
		channels.DELETE("/:id", h.DisconnectChannel)
		channels.GET("/:id/orders", h.GetChannelOrders)
		channels.GET("/:id/orders/unimported", h.GetUnimportedOrders)
		channels.GET("/:id/sync-jobs", h.GetSyncJobs)
		channels.POST("/:id/sync/orders", h.SyncOrders)
		channels.POST("/:id/sync/inventory", h.SyncInventory)
		channels.POST("/:id/tracking", h.PushTracking)
		channels.POST("/:id/fulfillment", h.CreateFulfillment)
		channels.POST("/:id/orders/import", h.ImportOrder)
		channels.GET("/:id/inventory", h.GetInventoryLevels)
	}

	// Seller channels
	r.GET("/sellers/:sellerId/channels", h.GetChannelsBySeller)

	// Webhooks
	r.POST("/webhooks/:channelId", h.HandleWebhook)
	r.POST("/webhooks/:channelId/:topic", h.HandleWebhook)
}

// ConnectChannel handles POST /channels
func (h *ChannelHandler) ConnectChannel(c *gin.Context) {
	var cmd application.ConnectChannelCommand
	if err := c.ShouldBindJSON(&cmd); err != nil {
		h.logger.Warn("Invalid connect channel request", "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"channel.type":  cmd.Type,
		"channel.name":  cmd.Name,
		"seller.id":     cmd.SellerID,
		"operation":     "connect_channel",
	})

	channel, err := h.service.ConnectChannel(c.Request.Context(), cmd)
	if err != nil {
		h.logger.WithError(err).Error("Failed to connect channel",
			"channel_type", cmd.Type,
			"seller_id", cmd.SellerID,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Channel connected successfully",
		"channel_id", channel.ID,
		"channel_type", cmd.Type,
		"seller_id", cmd.SellerID,
	)
	c.JSON(http.StatusCreated, channel)
}

// GetChannel handles GET /channels/:id
func (h *ChannelHandler) GetChannel(c *gin.Context) {
	channelID := c.Param("id")

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"channel.id": channelID,
		"operation":  "get_channel",
	})

	channel, err := h.service.GetChannel(c.Request.Context(), channelID)
	if err != nil {
		h.logger.Warn("Channel not found", "channel_id", channelID)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, channel)
}

// GetChannelsBySeller handles GET /sellers/:sellerId/channels
func (h *ChannelHandler) GetChannelsBySeller(c *gin.Context) {
	sellerID := c.Param("sellerId")

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"seller.id": sellerID,
		"operation": "get_channels_by_seller",
	})

	channels, err := h.service.GetChannelsBySeller(c.Request.Context(), sellerID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get channels by seller", "seller_id", sellerID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"channels": channels,
		"total":    len(channels),
	})
}

// UpdateChannel handles PUT /channels/:id
func (h *ChannelHandler) UpdateChannel(c *gin.Context) {
	channelID := c.Param("id")

	var cmd application.UpdateChannelCommand
	if err := c.ShouldBindJSON(&cmd); err != nil {
		h.logger.Warn("Invalid update channel request", "channel_id", channelID, "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"channel.id": channelID,
		"operation":  "update_channel",
	})

	channel, err := h.service.UpdateChannel(c.Request.Context(), channelID, cmd)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update channel", "channel_id", channelID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Channel updated", "channel_id", channelID)
	c.JSON(http.StatusOK, channel)
}

// DisconnectChannel handles DELETE /channels/:id
func (h *ChannelHandler) DisconnectChannel(c *gin.Context) {
	channelID := c.Param("id")

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"channel.id": channelID,
		"operation":  "disconnect_channel",
	})

	if err := h.service.DisconnectChannel(c.Request.Context(), channelID); err != nil {
		h.logger.WithError(err).Error("Failed to disconnect channel", "channel_id", channelID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Channel disconnected", "channel_id", channelID)
	c.JSON(http.StatusOK, gin.H{"message": "Channel disconnected"})
}

// GetChannelOrders handles GET /channels/:id/orders
func (h *ChannelHandler) GetChannelOrders(c *gin.Context) {
	channelID := c.Param("id")
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	pageSize, _ := strconv.ParseInt(c.DefaultQuery("pageSize", "20"), 10, 64)

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"channel.id": channelID,
		"page":       page,
		"page_size":  pageSize,
		"operation":  "get_channel_orders",
	})

	orders, err := h.service.GetChannelOrders(c.Request.Context(), channelID, page, pageSize)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get channel orders", "channel_id", channelID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"page":   page,
		"size":   pageSize,
	})
}

// GetUnimportedOrders handles GET /channels/:id/orders/unimported
func (h *ChannelHandler) GetUnimportedOrders(c *gin.Context) {
	channelID := c.Param("id")

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"channel.id": channelID,
		"operation":  "get_unimported_orders",
	})

	orders, err := h.service.GetUnimportedOrders(c.Request.Context(), channelID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get unimported orders", "channel_id", channelID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"total":  len(orders),
	})
}

// GetSyncJobs handles GET /channels/:id/sync-jobs
func (h *ChannelHandler) GetSyncJobs(c *gin.Context) {
	channelID := c.Param("id")
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	pageSize, _ := strconv.ParseInt(c.DefaultQuery("pageSize", "20"), 10, 64)

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"channel.id": channelID,
		"page":       page,
		"page_size":  pageSize,
		"operation":  "get_sync_jobs",
	})

	jobs, err := h.service.GetSyncJobs(c.Request.Context(), channelID, page, pageSize)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get sync jobs", "channel_id", channelID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs": jobs,
		"page": page,
		"size": pageSize,
	})
}

// SyncOrders handles POST /channels/:id/sync/orders
func (h *ChannelHandler) SyncOrders(c *gin.Context) {
	channelID := c.Param("id")
	start := time.Now()

	var cmd application.SyncOrdersCommand
	if err := c.ShouldBindJSON(&cmd); err != nil {
		// Allow empty body for simple sync
		cmd = application.SyncOrdersCommand{}
	}
	cmd.ChannelID = channelID

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"channel.id": channelID,
		"operation":  "sync_orders",
	})

	job, err := h.service.SyncOrders(c.Request.Context(), cmd)
	if err != nil {
		h.logger.WithError(err).Error("Failed to sync orders", "channel_id", channelID)
		h.metrics.RecordSyncOperation("unknown", "orders", "error", time.Since(start))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Order sync started",
		"channel_id", channelID,
		"job_id", job.ID,
	)
	h.metrics.RecordSyncOperation("unknown", "orders", "success", time.Since(start))
	c.JSON(http.StatusOK, job)
}

// SyncInventory handles POST /channels/:id/sync/inventory
func (h *ChannelHandler) SyncInventory(c *gin.Context) {
	channelID := c.Param("id")
	start := time.Now()

	var cmd application.SyncInventoryCommand
	if err := c.ShouldBindJSON(&cmd); err != nil {
		h.logger.Warn("Invalid sync inventory request", "channel_id", channelID, "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cmd.ChannelID = channelID

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"channel.id":     channelID,
		"item_count":     len(cmd.Items),
		"operation":      "sync_inventory",
	})

	job, err := h.service.SyncInventory(c.Request.Context(), cmd)
	if err != nil {
		h.logger.WithError(err).Error("Failed to sync inventory", "channel_id", channelID)
		h.metrics.RecordSyncOperation("unknown", "inventory", "error", time.Since(start))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Inventory sync started",
		"channel_id", channelID,
		"job_id", job.ID,
		"item_count", len(cmd.Items),
	)
	h.metrics.RecordSyncOperation("unknown", "inventory", "success", time.Since(start))
	c.JSON(http.StatusOK, job)
}

// PushTracking handles POST /channels/:id/tracking
func (h *ChannelHandler) PushTracking(c *gin.Context) {
	channelID := c.Param("id")

	var cmd application.PushTrackingCommand
	if err := c.ShouldBindJSON(&cmd); err != nil {
		h.logger.Warn("Invalid push tracking request", "channel_id", channelID, "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cmd.ChannelID = channelID

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"channel.id":      channelID,
		"order.id":        cmd.ExternalOrderID,
		"tracking.number": cmd.TrackingNumber,
		"operation":       "push_tracking",
	})

	if err := h.service.PushTracking(c.Request.Context(), cmd); err != nil {
		h.logger.WithError(err).Error("Failed to push tracking",
			"channel_id", channelID,
			"order_id", cmd.ExternalOrderID,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Tracking pushed successfully",
		"channel_id", channelID,
		"order_id", cmd.ExternalOrderID,
		"tracking_number", cmd.TrackingNumber,
	)
	c.JSON(http.StatusOK, gin.H{"message": "Tracking pushed successfully"})
}

// CreateFulfillment handles POST /channels/:id/fulfillment
func (h *ChannelHandler) CreateFulfillment(c *gin.Context) {
	channelID := c.Param("id")

	var cmd application.CreateFulfillmentCommand
	if err := c.ShouldBindJSON(&cmd); err != nil {
		h.logger.Warn("Invalid create fulfillment request", "channel_id", channelID, "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cmd.ChannelID = channelID

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"channel.id": channelID,
		"order.id":   cmd.ExternalOrderID,
		"operation":  "create_fulfillment",
	})

	if err := h.service.CreateFulfillment(c.Request.Context(), cmd); err != nil {
		h.logger.WithError(err).Error("Failed to create fulfillment",
			"channel_id", channelID,
			"order_id", cmd.ExternalOrderID,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Fulfillment created successfully",
		"channel_id", channelID,
		"order_id", cmd.ExternalOrderID,
	)
	c.JSON(http.StatusOK, gin.H{"message": "Fulfillment created successfully"})
}

// ImportOrder handles POST /channels/:id/orders/import
func (h *ChannelHandler) ImportOrder(c *gin.Context) {
	channelID := c.Param("id")

	var cmd application.ImportOrderCommand
	if err := c.ShouldBindJSON(&cmd); err != nil {
		h.logger.Warn("Invalid import order request", "channel_id", channelID, "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cmd.ChannelID = channelID

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"channel.id":       channelID,
		"channel.order_id": cmd.ExternalOrderID,
		"wms.order_id":     cmd.WMSOrderID,
		"operation":        "import_order",
	})

	if err := h.service.ImportOrder(c.Request.Context(), cmd); err != nil {
		h.logger.WithError(err).Error("Failed to import order",
			"channel_id", channelID,
			"channel_order_id", cmd.ExternalOrderID,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Order imported successfully",
		"channel_id", channelID,
		"channel_order_id", cmd.ExternalOrderID,
		"wms_order_id", cmd.WMSOrderID,
	)
	h.metrics.RecordOrdersImported("unknown", 1)
	c.JSON(http.StatusOK, gin.H{"message": "Order marked as imported"})
}

// GetInventoryLevels handles GET /channels/:id/inventory
func (h *ChannelHandler) GetInventoryLevels(c *gin.Context) {
	channelID := c.Param("id")
	skus := c.QueryArray("sku")

	if len(skus) == 0 {
		h.logger.Warn("No SKUs provided for inventory levels", "channel_id", channelID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one SKU required"})
		return
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"channel.id": channelID,
		"sku_count":  len(skus),
		"operation":  "get_inventory_levels",
	})

	levels, err := h.service.GetInventoryLevels(c.Request.Context(), channelID, skus)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get inventory levels",
			"channel_id", channelID,
			"sku_count", len(skus),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"levels": levels,
		"total":  len(levels),
	})
}

// HandleWebhook handles POST /webhooks/:channelId
func (h *ChannelHandler) HandleWebhook(c *gin.Context) {
	channelID := c.Param("channelId")
	topic := c.Param("topic")

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"channel.id":    channelID,
		"webhook.topic": topic,
		"operation":     "handle_webhook",
	})

	// Read body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.WithError(err).Error("Failed to read webhook body", "channel_id", channelID)
		h.metrics.RecordWebhookReceived("unknown", topic, "error")
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	// Get signature from headers (varies by channel)
	signature := c.GetHeader("X-Shopify-Hmac-Sha256")
	if signature == "" {
		signature = c.GetHeader("X-Hub-Signature-256")
	}

	cmd := application.WebhookCommand{
		ChannelID: channelID,
		Topic:     topic,
		Signature: signature,
		Body:      body,
	}

	if err := h.service.HandleWebhook(c.Request.Context(), cmd); err != nil {
		h.logger.WithError(err).Error("Failed to handle webhook",
			"channel_id", channelID,
			"topic", topic,
		)
		h.metrics.RecordWebhookReceived("unknown", topic, "error")
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Webhook processed successfully",
		"channel_id", channelID,
		"topic", topic,
	)
	h.metrics.RecordWebhookReceived("unknown", topic, "success")
	c.JSON(http.StatusOK, gin.H{"message": "Webhook processed"})
}
