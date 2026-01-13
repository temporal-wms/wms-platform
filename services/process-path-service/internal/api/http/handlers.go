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

// OptimizeRouting handles POST /api/v1/routing/optimize
func (h *Handlers) OptimizeRouting() gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, h.logger.Logger)

		var cmd application.OptimizeRoutingCommand
		if err := c.ShouldBindJSON(&cmd); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"order.id": cmd.OrderID,
			"priority": cmd.Priority,
		})

		result, err := h.service.OptimizeRouting(c.Request.Context(), cmd)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

// GetRoutingMetrics handles GET /api/v1/routing/metrics
func (h *Handlers) GetRoutingMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, h.logger.Logger)

		facilityID := c.Query("facilityId")
		zone := c.Query("zone")
		timeWindow := c.Query("timeWindow")
		if timeWindow == "" {
			timeWindow = "1h"
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"facility.id": facilityID,
			"zone":        zone,
			"timeWindow":  timeWindow,
		})

		result, err := h.service.GetRoutingMetrics(c.Request.Context(), facilityID, zone, timeWindow)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

// RerouteOrder handles POST /api/v1/routing/reroute
func (h *Handlers) RerouteOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, h.logger.Logger)

		var cmd application.RerouteOrderCommand
		if err := c.ShouldBindJSON(&cmd); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"order.id":     cmd.OrderID,
			"currentPath":  cmd.CurrentPath,
			"forceReroute": cmd.ForceReroute,
		})

		result, err := h.service.RerouteOrder(c.Request.Context(), cmd)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

// EscalateProcessPath handles POST /api/v1/process-paths/:pathId/escalate
func (h *Handlers) EscalateProcessPath() gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, h.logger.Logger)

		pathID := c.Param("pathId")

		var req struct {
			ToTier      string `json:"toTier" binding:"required"`
			Trigger     string `json:"trigger" binding:"required"`
			Reason      string `json:"reason" binding:"required"`
			EscalatedBy string `json:"escalatedBy"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"process_path.id": pathID,
			"toTier":          req.ToTier,
			"trigger":         req.Trigger,
		})

		cmd := application.EscalateProcessPathCommand{
			PathID:      pathID,
			ToTier:      req.ToTier,
			Trigger:     req.Trigger,
			Reason:      req.Reason,
			EscalatedBy: req.EscalatedBy,
		}

		result, err := h.service.EscalateProcessPath(c.Request.Context(), cmd)
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

// DowngradeProcessPath handles POST /api/v1/process-paths/:pathId/downgrade
func (h *Handlers) DowngradeProcessPath() gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, h.logger.Logger)

		pathID := c.Param("pathId")

		var req struct {
			ToTier       string `json:"toTier" binding:"required"`
			Reason       string `json:"reason" binding:"required"`
			DowngradedBy string `json:"downgradedBy"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"process_path.id": pathID,
			"toTier":          req.ToTier,
		})

		cmd := application.DowngradeProcessPathCommand{
			PathID:       pathID,
			ToTier:       req.ToTier,
			Reason:       req.Reason,
			DowngradedBy: req.DowngradedBy,
		}

		result, err := h.service.DowngradeProcessPath(c.Request.Context(), cmd)
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
