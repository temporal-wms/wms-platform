package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/middleware"

	"github.com/wms-platform/facility-service/internal/application"
)

// StationHandlers contains handlers for station operations
type StationHandlers struct {
	service *application.StationApplicationService
	logger  *logging.Logger
}

// NewStationHandlers creates a new StationHandlers
func NewStationHandlers(service *application.StationApplicationService, logger *logging.Logger) *StationHandlers {
	return &StationHandlers{
		service: service,
		logger:  logger,
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

	middleware.AddSpanAttributes(c, map[string]interface{}{
		"station.id": req.StationID,
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

	c.JSON(http.StatusCreated, station)
}

// GetStation handles getting a station by ID
func (h *StationHandlers) GetStation(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	stationID := c.Param("stationId")
	middleware.AddSpanAttributes(c, map[string]interface{}{
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

	c.JSON(http.StatusOK, station)
}

// DeleteStation handles deleting a station
func (h *StationHandlers) DeleteStation(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	stationID := c.Param("stationId")
	middleware.AddSpanAttributes(c, map[string]interface{}{
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

	c.Status(http.StatusNoContent)
}

// SetCapabilities handles setting all capabilities for a station
func (h *StationHandlers) SetCapabilities(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	stationID := c.Param("stationId")
	middleware.AddSpanAttributes(c, map[string]interface{}{
		"station.id": stationID,
	})

	var req struct {
		Capabilities []string `json:"capabilities" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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

	c.JSON(http.StatusOK, station)
}

// AddCapability handles adding a capability to a station
func (h *StationHandlers) AddCapability(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	stationID := c.Param("stationId")
	capability := c.Param("capability")
	middleware.AddSpanAttributes(c, map[string]interface{}{
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

	c.JSON(http.StatusOK, station)
}

// RemoveCapability handles removing a capability from a station
func (h *StationHandlers) RemoveCapability(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	stationID := c.Param("stationId")
	capability := c.Param("capability")
	middleware.AddSpanAttributes(c, map[string]interface{}{
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

	c.JSON(http.StatusOK, station)
}

// SetStatus handles setting the station status
func (h *StationHandlers) SetStatus(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	stationID := c.Param("stationId")
	middleware.AddSpanAttributes(c, map[string]interface{}{
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
		"requirements": req.Requirements,
		"stationType":  req.StationType,
		"zone":         req.Zone,
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

	c.JSON(http.StatusOK, stations)
}

// ListStations handles listing all stations
func (h *StationHandlers) ListStations(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	query := application.ListStationsQuery{
		Limit:  limit,
		Offset: offset,
	}

	stations, err := h.service.ListStations(c.Request.Context(), query)
	if err != nil {
		responder.RespondInternalError(err)
		return
	}

	c.JSON(http.StatusOK, stations)
}

// GetByZone handles getting stations by zone
func (h *StationHandlers) GetByZone(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	zone := c.Param("zone")
	middleware.AddSpanAttributes(c, map[string]interface{}{
		"zone": zone,
	})

	query := application.GetStationsByZoneQuery{Zone: zone}

	stations, err := h.service.GetByZone(c.Request.Context(), query)
	if err != nil {
		responder.RespondInternalError(err)
		return
	}

	c.JSON(http.StatusOK, stations)
}

// GetByType handles getting stations by type
func (h *StationHandlers) GetByType(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	stationType := c.Param("type")
	middleware.AddSpanAttributes(c, map[string]interface{}{
		"stationType": stationType,
	})

	query := application.GetStationsByTypeQuery{StationType: stationType}

	stations, err := h.service.GetByType(c.Request.Context(), query)
	if err != nil {
		responder.RespondInternalError(err)
		return
	}

	c.JSON(http.StatusOK, stations)
}

// GetByStatus handles getting stations by status
func (h *StationHandlers) GetByStatus(c *gin.Context) {
	responder := middleware.NewErrorResponder(c, h.logger.Logger)

	status := c.Param("status")
	middleware.AddSpanAttributes(c, map[string]interface{}{
		"status": status,
	})

	query := application.GetStationsByStatusQuery{Status: status}

	stations, err := h.service.GetByStatus(c.Request.Context(), query)
	if err != nil {
		responder.RespondInternalError(err)
		return
	}

	c.JSON(http.StatusOK, stations)
}
