package workflow

import (
	"context"
	"fmt"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

// SignalTracker tracks workflow signals
type SignalTracker struct {
	client client.Client
}

// NewSignalTracker creates a new signal tracker
func NewSignalTracker(client client.Client) *SignalTracker {
	return &SignalTracker{
		client: client,
	}
}

// GetWorkflowHistory retrieves the complete workflow history including signals
func (st *SignalTracker) GetWorkflowHistory(ctx context.Context, workflowID string, runID string) (*WorkflowHistory, error) {
	iter := st.client.GetWorkflowHistory(ctx, workflowID, runID, false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)

	history := &WorkflowHistory{
		WorkflowID: workflowID,
		RunID:      runID,
		Events:     []HistoryEvent{},
		Signals:    []SignalEvent{},
	}

	for iter.HasNext() {
		event, err := iter.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to read history event: %w", err)
		}

		historyEvent := HistoryEvent{
			EventID:   event.EventId,
			EventType: event.EventType.String(),
			Timestamp: event.EventTime,
		}

		history.Events = append(history.Events, historyEvent)

		// Extract signal events
		if event.EventType == enums.EVENT_TYPE_WORKFLOW_EXECUTION_SIGNALED {
			attrs := event.GetWorkflowExecutionSignaledEventAttributes()
			signalEvent := SignalEvent{
				EventID:    event.EventId,
				SignalName: attrs.SignalName,
				Timestamp:  event.EventTime,
				Input:      attrs.Input,
			}
			history.Signals = append(history.Signals, signalEvent)
		}
	}

	return history, nil
}

// GetSignalsForWorkflow returns all signals received by a workflow
func (st *SignalTracker) GetSignalsForWorkflow(ctx context.Context, workflowID string) ([]SignalEvent, error) {
	history, err := st.GetWorkflowHistory(ctx, workflowID, "")
	if err != nil {
		return nil, err
	}

	return history.Signals, nil
}

// ValidateSignalDelivery validates that a specific signal was delivered
func (st *SignalTracker) ValidateSignalDelivery(ctx context.Context, workflowID string, signalName string) (*SignalValidationResult, error) {
	signals, err := st.GetSignalsForWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	result := &SignalValidationResult{
		WorkflowID: workflowID,
		SignalName: signalName,
		Delivered:  false,
		Count:      0,
	}

	for _, signal := range signals {
		if signal.SignalName == signalName {
			result.Delivered = true
			result.Count++
			if result.FirstDelivery == nil {
				result.FirstDelivery = signal.Timestamp
			}
			result.LastDelivery = signal.Timestamp
		}
	}

	return result, nil
}

// WorkflowHistory represents the complete workflow history
type WorkflowHistory struct {
	WorkflowID string         `json:"workflowId"`
	RunID      string         `json:"runId"`
	Events     []HistoryEvent `json:"events"`
	Signals    []SignalEvent  `json:"signals"`
}

// HistoryEvent represents a workflow history event
type HistoryEvent struct {
	EventID   int64       `json:"eventId"`
	EventType string      `json:"eventType"`
	Timestamp interface{} `json:"timestamp"`
}

// SignalEvent represents a workflow signal event
type SignalEvent struct {
	EventID    int64       `json:"eventId"`
	SignalName string      `json:"signalName"`
	Timestamp  interface{} `json:"timestamp"`
	Input      interface{} `json:"input,omitempty"`
}

// SignalValidationResult represents the result of signal validation
type SignalValidationResult struct {
	WorkflowID    string      `json:"workflowId"`
	SignalName    string      `json:"signalName"`
	Delivered     bool        `json:"delivered"`
	Count         int         `json:"count"`
	FirstDelivery interface{} `json:"firstDelivery,omitempty"`
	LastDelivery  interface{} `json:"lastDelivery,omitempty"`
}
