package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/middleware"

	"github.com/wms-platform/facility-service/internal/application"
)

// StationService defines the application service behavior needed by handlers.
type StationService interface {
	CreateStation(ctx context.Context, cmd application.CreateStationCommand) (*application.StationDTO, error)
	GetStation(ctx context.Context, query application.GetStationQuery) (*application.StationDTO, error)
	UpdateStation(ctx context.Context, cmd application.UpdateStationCommand) (*application.StationDTO, error)
	DeleteStation(ctx context.Context, cmd application.DeleteStationCommand) error
	SetCapabilities(ctx context.Context, cmd application.SetCapabilitiesCommand) (*application.StationDTO, error)
	AddCapability(ctx context.Context, cmd application.AddCapabilityCommand) (*application.StationDTO, error)
	RemoveCapability(ctx context.Context, cmd application.RemoveCapabilityCommand) (*application.StationDTO, error)
	SetStatus(ctx context.Context, cmd application.SetStationStatusCommand) (*application.StationDTO, error)
	FindCapableStations(ctx context.Context, query application.FindCapableStationsQuery) ([]application.StationDTO, error)
	ListStations(ctx context.Context, query application.ListStationsQuery) ([]application.StationDTO, error)
	GetByZone(ctx context.Context, query application.GetStationsByZoneQuery) ([]application.StationDTO, error)
	GetByType(ctx context.Context, query application.GetStationsByTypeQuery) ([]application.StationDTO, error)
	GetByStatus(ctx context.Context, query application.GetStationsByStatusQuery) ([]application.StationDTO, error)
}

// StationHandlers contains handlers for station operations
type StationHandlers struct {
	service         StationService
	logger          *logging.Logger
	businessMetrics *middleware.BusinessMetrics
}

// NewStationHandlers creates a new StationHandlers
func NewStationHandlers(
	service StationService,
	logger *logging.Logger,
	businessMetrics *middleware.BusinessMetrics,
) *StationHandlers {
	return &StationHandlers{
		service:         service,
		logger:          logger,
		businessMetrics: businessMetrics,
	}
}

// RegisterRoutes registers station routes on the router
func (h *StationHandlers) RegisterRoutes(router *gin.RouterGroup) {
	stations := router.Group("/stations")
	{
		stations.POST("", h.CreateStation)
		stations.GET("", h.ListStations)
		stations.GET("/:stationId", h.GetStation)
		stations.PUT("/:stationId", h.UpdateStation)
		stations.DELETE("/:stationId", h.DeleteStation)
		stations.PUT("/:stationId/capabilities", h.SetCapabilities)
		stations.POST("/:stationId/capabilities/:capability", h.AddCapability)
		stations.DELETE("/:stationId/capabilities/:capability", h.RemoveCapability)
		stations.PUT("/:stationId/status", h.SetStatus)
		stations.POST("/find-capable", h.FindCapableStations)
		stations.GET("/zone/:zone", h.GetByZone)
		stations.GET("/type/:type", h.GetByType)
		stations.GET("/status/:status", h.GetByStatus)
		stations.POST("/:stationId/capacity/reserve", h.ReserveStationCapacity)
		stations.POST("/:stationId/capacity/release", h.ReleaseStationCapacity)
	}
}

// CreateStation handles station creation
func (h *StationHandlers) CreateStation(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	var req struct {
		StationID          string   `json:"stationId" binding:"required"`
		Name               string   `json:"name" binding:"required"`
		Zone               string   `json:"zone" binding:"required"`
		StationType        string   `json:"stationType" binding:"required"`
		Capabilities       []string `json:"capabilities"`
		MaxConcurrentTasks int      `json:"maxConcurrentTasks"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Enhanced span attributes
	middleware.AddSpanAttributes(c, map[string]interface{}{
		"operation":            "create",
		"station.id":           req.StationID,
		"station.zone":         req.Zone,
		"station.type":         req.StationType,
		"station.capabilities": len(req.Capabilities),
	})

	cmd := application.CreateStationCommand{
		StationID:          req.StationID,
		Name:               req.Name,
		Zone:               req.Zone,
		StationType:        req.StationType,
		Capabilities:       req.Capabilities,
		MaxConcurrentTasks: req.MaxConcurrentTasks,
	}

	station, err := h.service.CreateStation(c.Request.Context(), cmd)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			responder.RespondWithAppError(appErr)
		} else {
			responder.RespondInternalError(err)
		}
		return
	}

	// Add span event for successful creation
	middleware.AddSpanEvent(c, "station_created", map[string]interface{}{
		"station_id": station.StationID,
		"zone":       station.Zone,
	})

	c.JSON(http.StatusCreated, station)
}

// GetStation handles getting a station by ID
func (h *StationHandlers) GetStation(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	stationID := c.Param("stationId")
	middleware.AddSpanAttributes(c, map[string]interface{}{
		"operation":  "read",
		"station.id": stationID,
	})

	query := application.GetStationQuery{StationID: stationID}

	station, err := h.service.GetStation(c.Request.Context(), query)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			responder.RespondWithAppError(appErr)
		} else {
			responder.RespondInternalError(err)
		}
		return
	}

	c.JSON(http.StatusOK, station)
}

// UpdateStation handles updating a station
func (h *StationHandlers) UpdateStation(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	stationID := c.Param("stationId")
	middleware.AddSpanAttributes(c, map[string]interface{}{
		"operation":  "update",
		"station.id": stationID,
	})

	var req struct {
		Name               string `json:"name"`
		Zone               string `json:"zone"`
		MaxConcurrentTasks int    `json:"maxConcurrentTasks"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd := application.UpdateStationCommand{
		StationID:          stationID,
		Name:               req.Name,
		Zone:               req.Zone,
		MaxConcurrentTasks: req.MaxConcurrentTasks,
	}

	station, err := h.service.UpdateStation(c.Request.Context(), cmd)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			responder.RespondWithAppError(appErr)
		} else {
			responder.RespondInternalError(err)
		}
		return
	}

	// Add span event for successful update
	middleware.AddSpanEvent(c, "station_updated", map[string]interface{}{
		"station_id": stationID,
	})

	c.JSON(http.StatusOK, station)
}

// DeleteStation handles deleting a station
func (h *StationHandlers) DeleteStation(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	stationID := c.Param("stationId")
	middleware.AddSpanAttributes(c, map[string]interface{}{
		"operation":  "delete",
		"station.id": stationID,
	})

	cmd := application.DeleteStationCommand{StationID: stationID}

	if err := h.service.DeleteStation(c.Request.Context(), cmd); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			responder.RespondWithAppError(appErr)
		} else {
			responder.RespondInternalError(err)
		}
		return
	}

	// Add span event for successful deletion
	middleware.AddSpanEvent(c, "station_deleted", map[string]interface{}{
		"station_id": stationID,
	})

	c.Status(http.StatusNoContent)
}

// SetCapabilities handles setting all capabilities for a station
func (h *StationHandlers) SetCapabilities(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	stationID := c.Param("stationId")
	middleware.AddSpanAttributes(c, map[string]interface{}{
		"operation":            "update",
		"station.id":           stationID,
		"capabilities.count":   0, // will be updated after parsing
	})

	var req struct {
		Capabilities []string `json:"capabilities" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update span with actual capabilities count
	middleware.AddSpanAttributes(c, map[string]interface{}{
		"capabilities.count": len(req.Capabilities),
	})

	cmd := application.SetCapabilitiesCommand{
		StationID:    stationID,
		Capabilities: req.Capabilities,
	}

	station, err := h.service.SetCapabilities(c.Request.Context(), cmd)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			responder.RespondWithAppError(appErr)
		} else {
			responder.RespondInternalError(err)
		}
		return
	}

	// Add span event for successful capabilities update
	middleware.AddSpanEvent(c, "capabilities_updated", map[string]interface{}{
		"station_id":         stationID,
		"capabilities_count": len(req.Capabilities),
	})

	c.JSON(http.StatusOK, station)
}

// AddCapability handles adding a capability to a station
func (h *StationHandlers) AddCapability(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	stationID := c.Param("stationId")
	capability := c.Param("capability")
	middleware.AddSpanAttributes(c, map[string]interface{}{
		"operation":  "update",
		"station.id": stationID,
		"capability": capability,
	})

	cmd := application.AddCapabilityCommand{
		StationID:  stationID,
		Capability: capability,
	}

	station, err := h.service.AddCapability(c.Request.Context(), cmd)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			responder.RespondWithAppError(appErr)
		} else {
			responder.RespondInternalError(err)
		}
		return
	}

	// Add span event for capability added
	middleware.AddSpanEvent(c, "capability_added", map[string]interface{}{
		"station_id": stationID,
		"capability": capability,
	})

	c.JSON(http.StatusOK, station)
}

// RemoveCapability handles removing a capability from a station
func (h *StationHandlers) RemoveCapability(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	stationID := c.Param("stationId")
	capability := c.Param("capability")
	middleware.AddSpanAttributes(c, map[string]interface{}{
		"operation":  "update",
		"station.id": stationID,
		"capability": capability,
	})

	cmd := application.RemoveCapabilityCommand{
		StationID:  stationID,
		Capability: capability,
	}

	station, err := h.service.RemoveCapability(c.Request.Context(), cmd)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			responder.RespondWithAppError(appErr)
		} else {
			responder.RespondInternalError(err)
		}
		return
	}

	// Add span event for capability removed
	middleware.AddSpanEvent(c, "capability_removed", map[string]interface{}{
		"station_id": stationID,
		"capability": capability,
	})

	c.JSON(http.StatusOK, station)
}

// SetStatus handles setting the station status
func (h *StationHandlers) SetStatus(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	stationID := c.Param("stationId")
	middleware.AddSpanAttributes(c, map[string]interface{}{
		"operation":  "update",
		"station.id": stationID,
	})

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd := application.SetStationStatusCommand{
		StationID: stationID,
		Status:    req.Status,
	}

	station, err := h.service.SetStatus(c.Request.Context(), cmd)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			responder.RespondWithAppError(appErr)
		} else {
			responder.RespondInternalError(err)
		}
		return
	}

	// Add span event for status change
	middleware.AddSpanEvent(c, "station_status_changed", map[string]interface{}{
		"station_id": stationID,
		"new_status": req.Status,
	})

	c.JSON(http.StatusOK, station)
}

// FindCapableStations handles finding stations with required capabilities
func (h *StationHandlers) FindCapableStations(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	var req struct {
		Requirements []string `json:"requirements" binding:"required"`
		StationType  string   `json:"stationType"`
		Zone         string   `json:"zone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"operation":              "read",
		"query.requirements":     req.Requirements,
		"query.requirements_count": len(req.Requirements),
		"query.has_type_filter":  req.StationType != "",
		"query.has_zone_filter":  req.Zone != "",
	})

	query := application.FindCapableStationsQuery{
		Requirements: req.Requirements,
		StationType:  req.StationType,
		Zone:         req.Zone,
	}

	stations, err := h.service.FindCapableStations(c.Request.Context(), query)
	if err != nil {
		responder.RespondInternalError(err)
		return
	}

	// Add result count to span
	middleware.AddSpanAttributes(c, map[string]interface{}{
		"result.stations_count": len(stations),
	})

	c.JSON(http.StatusOK, stations)
}

// ListStations handles listing all stations
func (h *StationHandlers) ListStations(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"operation":    "read",
		"query.limit":  limit,
		"query.offset": offset,
	})

	query := application.ListStationsQuery{
		Limit:  limit,
		Offset: offset,
	}

	stations, err := h.service.ListStations(c.Request.Context(), query)
	if err != nil {
		responder.RespondInternalError(err)
		return
	}

	// Add result count to span
	middleware.AddSpanAttributes(c, map[string]interface{}{
		"result.stations_count": len(stations),
	})

	c.JSON(http.StatusOK, stations)
}

// GetByZone handles getting stations by zone
func (h *StationHandlers) GetByZone(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	zone := c.Param("zone")
	middleware.AddSpanAttributes(c, map[string]interface{}{
		"operation": "read",
		"zone":      zone,
	})

	query := application.GetStationsByZoneQuery{Zone: zone}

	stations, err := h.service.GetByZone(c.Request.Context(), query)
	if err != nil {
		responder.RespondInternalError(err)
		return
	}

	// Add result count to span
	middleware.AddSpanAttributes(c, map[string]interface{}{
		"result.stations_count": len(stations),
	})

	c.JSON(http.StatusOK, stations)
}

// GetByType handles getting stations by type
func (h *StationHandlers) GetByType(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	stationType := c.Param("type")
	middleware.AddSpanAttributes(c, map[string]interface{}{
		"operation":   "read",
		"stationType": stationType,
	})

	query := application.GetStationsByTypeQuery{StationType: stationType}

	stations, err := h.service.GetByType(c.Request.Context(), query)
	if err != nil {
		responder.RespondInternalError(err)
		return
	}

	// Add result count to span
	middleware.AddSpanAttributes(c, map[string]interface{}{
		"result.stations_count": len(stations),
	})

	c.JSON(http.StatusOK, stations)
}

// GetByStatus handles getting stations by status
func (h *StationHandlers) GetByStatus(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	status := c.Param("status")
	middleware.AddSpanAttributes(c, map[string]interface{}{
		"operation": "read",
		"status":    status,
	})

	query := application.GetStationsByStatusQuery{Status: status}

	stations, err := h.service.GetByStatus(c.Request.Context(), query)
	if err != nil {
		responder.RespondInternalError(err)
		return
	}

	// Add result count to span
	middleware.AddSpanAttributes(c, map[string]interface{}{
		"result.stations_count": len(stations),
	})

	c.JSON(http.StatusOK, stations)
}

// ReserveStationCapacity handles reserving capacity on a station
func (h *StationHandlers) ReserveStationCapacity(c *gin.Context) {
	stationID := c.Param("stationId")

	var req struct {
		StationID     string `json:"stationId" binding:"required"`
		OrderID       string `json:"orderId" binding:"required"`
		RequiredSlots int    `json:"requiredSlots" binding:"required"`
		ReservationID string `json:"reservationId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"operation":      "reserve_capacity",
		"station.id":     stationID,
		"order.id":       req.OrderID,
		"required_slots": req.RequiredSlots,
	})

	// For now, return a simple success response
	// TODO: Implement actual station capacity management domain logic
	h.logger.Info("Station capacity reserved",
		"stationId", stationID,
		"orderId", req.OrderID,
		"requiredSlots", req.RequiredSlots,
		"reservationId", req.ReservationID,
	)

	middleware.AddSpanEvent(c, "capacity_reserved", map[string]interface{}{
		"station_id":     stationID,
		"reservation_id": req.ReservationID,
	})

	c.JSON(http.StatusOK, gin.H{
		"reservationId":     req.ReservationID,
		"stationId":         stationID,
		"reservedSlots":     req.RequiredSlots,
		"remainingCapacity": 10, // Mock value - TODO: implement actual capacity tracking
	})
}

// ReleaseStationCapacity handles releasing previously reserved capacity
func (h *StationHandlers) ReleaseStationCapacity(c *gin.Context) {
	stationID := c.Param("stationId")

	var req struct {
		StationID     string `json:"stationId" binding:"required"`
		OrderID       string `json:"orderId" binding:"required"`
		ReservationID string `json:"reservationId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"operation":  "release_capacity",
		"station.id": stationID,
		"order.id":   req.OrderID,
	})

	// For now, return a simple success response
	// TODO: Implement actual station capacity management domain logic
	h.logger.Info("Station capacity released",
		"stationId", stationID,
		"orderId", req.OrderID,
		"reservationId", req.ReservationID,
	)

	middleware.AddSpanEvent(c, "capacity_released", map[string]interface{}{
		"station_id":     stationID,
		"reservation_id": req.ReservationID,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":   "capacity released successfully",
		"stationId": stationID,
	})
}
