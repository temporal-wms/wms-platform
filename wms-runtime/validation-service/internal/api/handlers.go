package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// StartTracking starts event tracking for an order
func (s *Server) StartTracking(c *gin.Context) {
	orderID := c.Param("orderId")

	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "orderId is required"})
		return
	}

	s.eventStore.StartTracking(orderID)

	c.JSON(http.StatusOK, gin.H{
		"orderId":  orderID,
		"tracking": true,
		"message":  "Event tracking started",
	})
}

// GetEvents returns all captured events for an order
func (s *Server) GetEvents(c *gin.Context) {
	orderID := c.Param("orderId")

	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "orderId is required"})
		return
	}

	events := s.eventStore.GetEvents(orderID)

	c.JSON(http.StatusOK, gin.H{
		"orderId":    orderID,
		"eventCount": len(events),
		"events":     events,
	})
}

// AssertEventsRequest represents the request body for asserting events
type AssertEventsRequest struct {
	ExpectedTypes []string `json:"expectedTypes" binding:"required"`
}

// AssertEvents validates that expected events were received
func (s *Server) AssertEvents(c *gin.Context) {
	orderID := c.Param("orderId")

	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "orderId is required"})
		return
	}

	var req AssertEventsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	events := s.eventStore.GetEvents(orderID)
	eventTypeSet := make(map[string]bool)

	for _, event := range events {
		eventTypeSet[event.Type] = true
	}

	missingEvents := []string{}
	for _, expectedType := range req.ExpectedTypes {
		if !eventTypeSet[expectedType] {
			missingEvents = append(missingEvents, expectedType)
		}
	}

	success := len(missingEvents) == 0

	c.JSON(http.StatusOK, gin.H{
		"orderId":       orderID,
		"success":       success,
		"expectedCount": len(req.ExpectedTypes),
		"receivedCount": len(events),
		"missingEvents": missingEvents,
	})
}

// GetStatus returns the validation status for an order
func (s *Server) GetStatus(c *gin.Context) {
	orderID := c.Param("orderId")

	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "orderId is required"})
		return
	}

	isTracking := s.eventStore.IsTracking(orderID)
	eventCount := s.eventStore.GetEventCount(orderID)

	c.JSON(http.StatusOK, gin.H{
		"orderId":    orderID,
		"tracking":   isTracking,
		"eventCount": eventCount,
	})
}

// ClearTracking clears all tracking data for an order
func (s *Server) ClearTracking(c *gin.Context) {
	orderID := c.Param("orderId")

	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "orderId is required"})
		return
	}

	s.eventStore.Clear(orderID)

	c.JSON(http.StatusOK, gin.H{
		"orderId": orderID,
		"message": "Tracking data cleared",
	})
}

// ValidateSequenceRequest represents the request for sequence validation
type ValidateSequenceRequest struct {
	FlowType string `json:"flowType" binding:"required"`
}

// ValidateSequence validates the event sequence for an order
func (s *Server) ValidateSequence(c *gin.Context) {
	orderID := c.Param("orderId")

	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "orderId is required"})
		return
	}

	var req ValidateSequenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get actual events
	events := s.eventStore.GetEvents(orderID)
	eventTypes := make([]string, len(events))
	for i, event := range events {
		eventTypes[i] = event.Type
	}

	// Validate sequence
	result, err := s.sequenceValidator.ValidateSequence(req.FlowType, eventTypes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetReport returns a comprehensive validation report
func (s *Server) GetReport(c *gin.Context) {
	orderID := c.Param("orderId")

	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "orderId is required"})
		return
	}

	// Get event flow analysis
	flowAnalysis, err := s.correlationTracker.AnalyzeEventFlow(orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Get correlation validation
	correlationResult, _ := s.correlationTracker.ValidateEventCorrelation(orderID)

	c.JSON(http.StatusOK, gin.H{
		"orderId":      orderID,
		"flowAnalysis": flowAnalysis,
		"correlation":  correlationResult,
	})
}

// GetStatsSummary returns overall statistics
func (s *Server) GetStatsSummary(c *gin.Context) {
	stats := s.eventStore.GetStats()
	c.JSON(http.StatusOK, stats)
}

// GetEventsByType returns events grouped by type
func (s *Server) GetEventsByType(c *gin.Context) {
	orderIDs := s.eventStore.GetAllOrderIDs()

	eventsByType := make(map[string]int)

	for _, orderID := range orderIDs {
		events := s.eventStore.GetEvents(orderID)
		for _, event := range events {
			eventsByType[event.Type]++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"eventsByType": eventsByType,
		"totalTypes":   len(eventsByType),
	})
}
