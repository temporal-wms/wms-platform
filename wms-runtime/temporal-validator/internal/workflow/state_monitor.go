package workflow

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/client"
)

// StateMonitor monitors workflow execution states
type StateMonitor struct {
	client client.Client
}

// NewStateMonitor creates a new workflow state monitor
func NewStateMonitor(client client.Client) *StateMonitor {
	return &StateMonitor{
		client: client,
	}
}

// DescribeWorkflow returns detailed information about a workflow execution
func (sm *StateMonitor) DescribeWorkflow(ctx context.Context, workflowID string, runID string) (*WorkflowDescription, error) {
	desc, err := sm.client.DescribeWorkflowExecution(ctx, workflowID, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to describe workflow: %w", err)
	}

	// Extract workflow info
	workflowInfo := desc.WorkflowExecutionInfo

	description := &WorkflowDescription{
		WorkflowID:   workflowID,
		RunID:        workflowInfo.Execution.RunId,
		WorkflowType: workflowInfo.Type.Name,
		Status:       workflowInfo.Status.String(),
		StartTime:    workflowInfo.StartTime,
		CloseTime:    workflowInfo.CloseTime,
		ExecutionTime: workflowInfo.ExecutionTime,
	}

	return description, nil
}

// GetWorkflowStatus returns the current status of a workflow
func (sm *StateMonitor) GetWorkflowStatus(ctx context.Context, workflowID string) (*WorkflowStatus, error) {
	desc, err := sm.client.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow status: %w", err)
	}

	workflowInfo := desc.WorkflowExecutionInfo

	status := &WorkflowStatus{
		WorkflowID: workflowID,
		RunID:      workflowInfo.Execution.RunId,
		Status:     workflowInfo.Status.String(),
		IsRunning:  workflowInfo.Status == 1, // Running = 1 in temporal
	}

	return status, nil
}

// QueryWorkflow executes a query on a workflow
func (sm *StateMonitor) QueryWorkflow(ctx context.Context, workflowID string, runID string, queryType string, args interface{}) (interface{}, error) {
	resp, err := sm.client.QueryWorkflow(ctx, workflowID, runID, queryType, args)
	if err != nil {
		return nil, fmt.Errorf("failed to query workflow: %w", err)
	}

	var result interface{}
	if err := resp.Get(&result); err != nil {
		return nil, fmt.Errorf("failed to decode query result: %w", err)
	}

	return result, nil
}

// WorkflowDescription represents detailed workflow information
type WorkflowDescription struct {
	WorkflowID    string      `json:"workflowId"`
	RunID         string      `json:"runId"`
	WorkflowType  string      `json:"workflowType"`
	Status        string      `json:"status"`
	StartTime     interface{} `json:"startTime"`
	CloseTime     interface{} `json:"closeTime,omitempty"`
	ExecutionTime interface{} `json:"executionTime,omitempty"`
}

// WorkflowStatus represents the current status of a workflow
type WorkflowStatus struct {
	WorkflowID string `json:"workflowId"`
	RunID      string `json:"runId"`
	Status     string `json:"status"`
	IsRunning  bool   `json:"isRunning"`
}
