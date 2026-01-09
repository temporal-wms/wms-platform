package validation

import (
	"fmt"

	"github.com/wms-platform/wms-runtime/validation-service/internal/eventcapture"
)

// CorrelationTracker tracks event correlations across services
type CorrelationTracker struct {
	eventStore *eventcapture.EventStore
}

// NewCorrelationTracker creates a new correlation tracker
func NewCorrelationTracker(eventStore *eventcapture.EventStore) *CorrelationTracker {
	return &CorrelationTracker{
		eventStore: eventStore,
	}
}

// GetEventChain returns the chain of events for an order
func (ct *CorrelationTracker) GetEventChain(orderID string) []*eventcapture.CapturedEvent {
	return ct.eventStore.GetEvents(orderID)
}

// AnalyzeEventFlow analyzes the flow of events for an order
func (ct *CorrelationTracker) AnalyzeEventFlow(orderID string) (*EventFlowAnalysis, error) {
	events := ct.eventStore.GetEvents(orderID)

	if len(events) == 0 {
		return nil, fmt.Errorf("no events found for order %s", orderID)
	}

	analysis := &EventFlowAnalysis{
		OrderID:      orderID,
		TotalEvents:  len(events),
		EventsByType: make(map[string]int),
		EventChain:   make([]string, len(events)),
		Services:     make(map[string]bool),
		Topics:       make(map[string]int),
	}

	// Analyze events
	for i, event := range events {
		// Count by type
		analysis.EventsByType[event.Type]++

		// Build event chain
		analysis.EventChain[i] = event.Type

		// Track services
		if event.Source != "" {
			analysis.Services[event.Source] = true
		}

		// Track topics
		analysis.Topics[event.Topic]++

		// Track first and last event times
		if i == 0 {
			analysis.FirstEventTime = event.Timestamp
		}
		if i == len(events)-1 {
			analysis.LastEventTime = event.Timestamp
		}
	}

	// Calculate duration
	if len(events) > 1 {
		analysis.TotalDuration = analysis.LastEventTime.Sub(analysis.FirstEventTime)
	}

	// Extract service list
	analysis.ServiceList = make([]string, 0, len(analysis.Services))
	for service := range analysis.Services {
		analysis.ServiceList = append(analysis.ServiceList, service)
	}

	return analysis, nil
}

// ValidateEventCorrelation validates that events are properly correlated
func (ct *CorrelationTracker) ValidateEventCorrelation(orderID string) (*CorrelationValidationResult, error) {
	analysis, err := ct.AnalyzeEventFlow(orderID)
	if err != nil {
		return nil, err
	}

	result := &CorrelationValidationResult{
		OrderID:     orderID,
		IsValid:     true,
		Issues:      []string{},
		EventCount:  analysis.TotalEvents,
		ServiceList: analysis.ServiceList,
	}

	// Check for minimum expected events
	if analysis.TotalEvents < 3 {
		result.Issues = append(result.Issues, "insufficient events captured (minimum 3 expected)")
		result.IsValid = false
	}

	// Check for expected event types
	expectedTypes := []string{
		"wms.order.received",
		"wms.order.validated",
	}

	for _, expectedType := range expectedTypes {
		if analysis.EventsByType[expectedType] == 0 {
			result.Issues = append(result.Issues, fmt.Sprintf("missing expected event: %s", expectedType))
			result.IsValid = false
		}
	}

	return result, nil
}

// EventFlowAnalysis represents an analysis of event flow
type EventFlowAnalysis struct {
	OrderID        string            `json:"orderId"`
	TotalEvents    int               `json:"totalEvents"`
	EventsByType   map[string]int    `json:"eventsByType"`
	EventChain     []string          `json:"eventChain"`
	Services       map[string]bool   `json:"-"`
	ServiceList    []string          `json:"services"`
	Topics         map[string]int    `json:"topics"`
	FirstEventTime interface{}       `json:"firstEventTime"`
	LastEventTime  interface{}       `json:"lastEventTime"`
	TotalDuration  interface{}       `json:"totalDuration"`
}

// CorrelationValidationResult represents correlation validation result
type CorrelationValidationResult struct {
	OrderID     string   `json:"orderId"`
	IsValid     bool     `json:"isValid"`
	Issues      []string `json:"issues"`
	EventCount  int      `json:"eventCount"`
	ServiceList []string `json:"services"`
}
