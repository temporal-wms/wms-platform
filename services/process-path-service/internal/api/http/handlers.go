package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/wms-platform/process-path-service/internal/application"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/middleware"
)

// Handlers contains HTTP handlers for process path endpoints
type Handlers struct {
	service *application.ProcessPathService
	logger  *logging.Logger
}

// NewHandlers creates new HTTP handlers
func NewHandlers(service *application.ProcessPathService, logger *logging.Logger) *Handlers {
	return &Handlers{
		service: service,
		logger:  logger,
	}
}

// DetermineProcessPath handles POST /api/v1/process-paths/determine
func (h *Handlers) DetermineProcessPath() gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, h.logger.Logger)

		var cmd application.DetermineProcessPathCommand
		if err := c.ShouldBindJSON(&cmd); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"order.id": cmd.OrderID,
		})

		result, err := h.service.DetermineProcessPath(c.Request.Context(), cmd)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusCreated, result)
	}
}

// GetProcessPath handles GET /api/v1/process-paths/:pathId
func (h *Handlers) GetProcessPath() gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, h.logger.Logger)

		pathID := c.Param("pathId")

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"process_path.id": pathID,
		})

		result, err := h.service.GetProcessPath(c.Request.Context(), pathID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

// GetProcessPathByOrder handles GET /api/v1/process-paths/order/:orderId
func (h *Handlers) GetProcessPathByOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, h.logger.Logger)

		orderID := c.Param("orderId")

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"order.id": orderID,
		})

		result, err := h.service.GetProcessPathByOrderID(c.Request.Context(), orderID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

// AssignStation handles PUT /api/v1/process-paths/:pathId/station
func (h *Handlers) AssignStation() gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, h.logger.Logger)

		pathID := c.Param("pathId")

		var req struct {
			StationID string `json:"stationId" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"process_path.id": pathID,
			"station.id":      req.StationID,
		})

		cmd := application.AssignStationCommand{
			PathID:    pathID,
			StationID: req.StationID,
		}

		result, err := h.service.AssignStation(c.Request.Context(), cmd)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, result)
	}
}
