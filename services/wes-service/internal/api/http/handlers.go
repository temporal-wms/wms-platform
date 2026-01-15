package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/middleware"
	"github.com/wms-platform/wes-service/internal/application"
)

// Handlers holds the HTTP handlers for WES service
type Handlers struct {
	service         *application.WESApplicationService
	logger          *logging.Logger
	businessMetrics *middleware.BusinessMetrics
}

// NewHandlers creates a new Handlers instance
func NewHandlers(
	service *application.WESApplicationService,
	logger *logging.Logger,
	businessMetrics *middleware.BusinessMetrics,
) *Handlers {
	return &Handlers{
		service:         service,
		logger:          logger,
		businessMetrics: businessMetrics,
	}
}

// ResolveExecutionPlan handles POST /api/v1/execution-plans/resolve
func (h *Handlers) ResolveExecutionPlan(c *gin.Context) {
	var cmd application.ResolveExecutionPlanCommand
	if err := c.ShouldBindJSON(&cmd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	plan, err := h.service.ResolveExecutionPlan(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, plan)
}

// CreateTaskRoute handles POST /api/v1/routes
func (h *Handlers) CreateTaskRoute(c *gin.Context) {
	var cmd application.CreateTaskRouteCommand
	if err := c.ShouldBindJSON(&cmd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	route, err := h.service.CreateTaskRoute(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, route)
}

// GetTaskRoute handles GET /api/v1/routes/:routeId
func (h *Handlers) GetTaskRoute(c *gin.Context) {
	routeID := c.Param("routeId")

	route, err := h.service.GetTaskRoute(c.Request.Context(), application.GetTaskRouteQuery{RouteID: routeID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if route == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
		return
	}

	c.JSON(http.StatusOK, route)
}

// GetTaskRouteByOrder handles GET /api/v1/routes/order/:orderId
func (h *Handlers) GetTaskRouteByOrder(c *gin.Context) {
	orderID := c.Param("orderId")

	route, err := h.service.GetTaskRouteByOrder(c.Request.Context(), application.GetRouteByOrderQuery{OrderID: orderID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if route == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
		return
	}

	c.JSON(http.StatusOK, route)
}

// AssignWorkerRequest represents the request body for assigning a worker
type AssignWorkerRequest struct {
	WorkerID string `json:"workerId" binding:"required"`
	TaskID   string `json:"taskId" binding:"required"`
}

// AssignWorkerToStage handles POST /api/v1/routes/:routeId/stages/current/assign
func (h *Handlers) AssignWorkerToStage(c *gin.Context) {
	routeID := c.Param("routeId")

	var req AssignWorkerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd := application.AssignWorkerCommand{
		RouteID:  routeID,
		WorkerID: req.WorkerID,
		TaskID:   req.TaskID,
	}

	route, err := h.service.AssignWorkerToStage(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, route)
}

// StartStage handles POST /api/v1/routes/:routeId/stages/current/start
func (h *Handlers) StartStage(c *gin.Context) {
	routeID := c.Param("routeId")

	route, err := h.service.StartStage(c.Request.Context(), application.StartStageCommand{RouteID: routeID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, route)
}

// CompleteStage handles POST /api/v1/routes/:routeId/stages/current/complete
func (h *Handlers) CompleteStage(c *gin.Context) {
	routeID := c.Param("routeId")

	route, err := h.service.CompleteStage(c.Request.Context(), application.CompleteStageCommand{RouteID: routeID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, route)
}

// FailStageRequest represents the request body for failing a stage
type FailStageRequest struct {
	Error string `json:"error" binding:"required"`
}

// FailStage handles POST /api/v1/routes/:routeId/stages/current/fail
func (h *Handlers) FailStage(c *gin.Context) {
	routeID := c.Param("routeId")

	var req FailStageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd := application.FailStageCommand{
		RouteID: routeID,
		Error:   req.Error,
	}

	route, err := h.service.FailStage(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, route)
}

// ListTemplates handles GET /api/v1/templates
func (h *Handlers) ListTemplates(c *gin.Context) {
	activeOnly := c.Query("activeOnly") == "true"

	templates, err := h.service.ListTemplates(c.Request.Context(), application.ListTemplatesQuery{ActiveOnly: activeOnly})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, templates)
}

// GetTemplate handles GET /api/v1/templates/:templateId
func (h *Handlers) GetTemplate(c *gin.Context) {
	templateID := c.Param("templateId")

	template, err := h.service.GetTemplate(c.Request.Context(), application.GetTemplateQuery{TemplateID: templateID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if template == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}

	c.JSON(http.StatusOK, template)
}

// HealthCheck handles GET /health
func (h *Handlers) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

// ReadyCheck handles GET /ready
func (h *Handlers) ReadyCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// ReserveStationCapacityRequest represents the request body for reserving station capacity
type ReserveStationCapacityRequest struct {
	StationID     string `json:"stationId" binding:"required"`
	OrderID       string `json:"orderId" binding:"required"`
	RequiredSlots int    `json:"requiredSlots" binding:"required"`
	ReservationID string `json:"reservationId" binding:"required"`
}

// ReserveStationCapacity handles POST /api/v1/stations/:stationId/capacity/reserve
func (h *Handlers) ReserveStationCapacity(c *gin.Context) {
	stationID := c.Param("stationId")

	var req ReserveStationCapacityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// For now, return a simple success response
	// TODO: Implement actual station capacity management
	h.logger.Info("Station capacity reserved",
		"stationId", stationID,
		"orderId", req.OrderID,
		"requiredSlots", req.RequiredSlots,
		"reservationId", req.ReservationID,
	)

	c.JSON(http.StatusOK, gin.H{
		"reservationId":     req.ReservationID,
		"stationId":         stationID,
		"reservedSlots":     req.RequiredSlots,
		"remainingCapacity": 10, // Mock value
	})
}

// ReleaseStationCapacityRequest represents the request body for releasing station capacity
type ReleaseStationCapacityRequest struct {
	StationID     string `json:"stationId" binding:"required"`
	OrderID       string `json:"orderId" binding:"required"`
	ReservationID string `json:"reservationId" binding:"required"`
}

// ReleaseStationCapacity handles POST /api/v1/stations/:stationId/capacity/release
func (h *Handlers) ReleaseStationCapacity(c *gin.Context) {
	stationID := c.Param("stationId")

	var req ReleaseStationCapacityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// For now, return a simple success response
	// TODO: Implement actual station capacity management
	h.logger.Info("Station capacity released",
		"stationId", stationID,
		"orderId", req.OrderID,
		"reservationId", req.ReservationID,
	)

	c.JSON(http.StatusOK, gin.H{
		"message":   "capacity released successfully",
		"stationId": stationID,
	})
}
