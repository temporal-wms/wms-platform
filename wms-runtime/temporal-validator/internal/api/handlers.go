package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// DescribeWorkflow returns detailed workflow information
func (s *Server) DescribeWorkflow(c *gin.Context) {
	workflowID := c.Param("workflowId")
	runID := c.Query("runId")

	if workflowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workflowId is required"})
		return
	}

	description, err := s.stateMonitor.DescribeWorkflow(c.Request.Context(), workflowID, runID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, description)
}

// GetWorkflowHistory returns the complete workflow history
func (s *Server) GetWorkflowHistory(c *gin.Context) {
	workflowID := c.Param("workflowId")
	runID := c.Query("runId")

	if workflowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workflowId is required"})
		return
	}

	history, err := s.signalTracker.GetWorkflowHistory(c.Request.Context(), workflowID, runID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, history)
}

// AssertSignalRequest represents the request to assert a signal
type AssertSignalRequest struct {
	SignalName string `json:"signalName" binding:"required"`
}

// AssertSignal validates that a signal was received
func (s *Server) AssertSignal(c *gin.Context) {
	workflowID := c.Param("workflowId")

	if workflowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workflowId is required"})
		return
	}

	var req AssertSignalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := s.signalTracker.ValidateSignalDelivery(c.Request.Context(), workflowID, req.SignalName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetWorkflowStatus returns the current workflow status
func (s *Server) GetWorkflowStatus(c *gin.Context) {
	workflowID := c.Param("workflowId")

	if workflowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workflowId is required"})
		return
	}

	status, err := s.stateMonitor.GetWorkflowStatus(c.Request.Context(), workflowID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// QueryWorkflowRequest represents a workflow query request
type QueryWorkflowRequest struct {
	QueryType string      `json:"queryType" binding:"required"`
	Args      interface{} `json:"args,omitempty"`
}

// QueryWorkflow executes a query on a workflow
func (s *Server) QueryWorkflow(c *gin.Context) {
	workflowID := c.Param("workflowId")
	runID := c.Query("runId")

	if workflowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workflowId is required"})
		return
	}

	var req QueryWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := s.stateMonitor.QueryWorkflow(c.Request.Context(), workflowID, runID, req.QueryType, req.Args)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"workflowId": workflowID,
		"queryType":  req.QueryType,
		"result":     result,
	})
}

// ListSignals returns all signals for a workflow
func (s *Server) ListSignals(c *gin.Context) {
	workflowID := c.Param("workflowId")

	if workflowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workflowId is required"})
		return
	}

	signals, err := s.signalTracker.GetSignalsForWorkflow(c.Request.Context(), workflowID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"workflowId":   workflowID,
		"signalCount":  len(signals),
		"signals":      signals,
	})
}

// ValidateSignalDeliveryRequest represents signal validation request
type ValidateSignalDeliveryRequest struct {
	SignalName string `json:"signalName" binding:"required"`
}

// ValidateSignalDelivery validates signal delivery
func (s *Server) ValidateSignalDelivery(c *gin.Context) {
	workflowID := c.Param("workflowId")

	if workflowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workflowId is required"})
		return
	}

	var req ValidateSignalDeliveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := s.signalTracker.ValidateSignalDelivery(c.Request.Context(), workflowID, req.SignalName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetStatsSummary returns summary statistics
func (s *Server) GetStatsSummary(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service": "temporal-validator",
		"status":  "healthy",
	})
}
