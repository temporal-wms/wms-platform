package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wms-platform/walling-service/internal/application"
)

// Handlers holds the HTTP handlers for Walling service
type Handlers struct {
	service *application.WallingApplicationService
}

// NewHandlers creates a new Handlers instance
func NewHandlers(service *application.WallingApplicationService) *Handlers {
	return &Handlers{service: service}
}

// CreateTask handles POST /api/v1/tasks
func (h *Handlers) CreateTask(c *gin.Context) {
	var cmd application.CreateWallingTaskCommand
	if err := c.ShouldBindJSON(&cmd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.service.CreateWallingTask(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, task)
}

// GetTask handles GET /api/v1/tasks/:taskId
func (h *Handlers) GetTask(c *gin.Context) {
	taskID := c.Param("taskId")

	task, err := h.service.GetTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if task == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

// AssignRequest represents the request body for assigning a walliner
type AssignRequest struct {
	WallinerID string `json:"wallinerId" binding:"required"`
	Station    string `json:"station"`
}

// AssignWalliner handles POST /api/v1/tasks/:taskId/assign
func (h *Handlers) AssignWalliner(c *gin.Context) {
	taskID := c.Param("taskId")

	var req AssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd := application.AssignWallinerCommand{
		TaskID:     taskID,
		WallinerID: req.WallinerID,
		Station:    req.Station,
	}

	task, err := h.service.AssignWalliner(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

// SortRequest represents the request body for sorting an item
type SortRequest struct {
	SKU        string `json:"sku" binding:"required"`
	Quantity   int    `json:"quantity" binding:"required,min=1"`
	FromToteID string `json:"fromToteId" binding:"required"`
}

// SortItem handles POST /api/v1/tasks/:taskId/sort
func (h *Handlers) SortItem(c *gin.Context) {
	taskID := c.Param("taskId")

	var req SortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd := application.SortItemCommand{
		TaskID:     taskID,
		SKU:        req.SKU,
		Quantity:   req.Quantity,
		FromToteID: req.FromToteID,
	}

	task, err := h.service.SortItem(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

// CompleteTask handles POST /api/v1/tasks/:taskId/complete
func (h *Handlers) CompleteTask(c *gin.Context) {
	taskID := c.Param("taskId")

	cmd := application.CompleteTaskCommand{TaskID: taskID}
	task, err := h.service.CompleteTask(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

// GetActiveTaskByWalliner handles GET /api/v1/tasks/walliner/:wallinerId/active
func (h *Handlers) GetActiveTaskByWalliner(c *gin.Context) {
	wallinerID := c.Param("wallinerId")

	task, err := h.service.GetActiveTaskByWalliner(c.Request.Context(), wallinerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if task == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active task found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

// GetPendingTasksByPutWall handles GET /api/v1/tasks/pending
func (h *Handlers) GetPendingTasksByPutWall(c *gin.Context) {
	putWallID := c.Query("putWallId")
	if putWallID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "putWallId query parameter is required"})
		return
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	tasks, err := h.service.GetPendingTasksByPutWall(c.Request.Context(), putWallID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// HealthCheck handles GET /health
func (h *Handlers) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

// ReadyCheck handles GET /ready
func (h *Handlers) ReadyCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
