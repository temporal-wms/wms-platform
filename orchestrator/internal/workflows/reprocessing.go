package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ReprocessingBatchInput is the input for the batch workflow (can be empty for scheduled runs)
type ReprocessingBatchInput struct {
	// MaxOrders overrides the default batch size
	MaxOrders int `json:"maxOrders,omitempty"`
}

// ReprocessingResult contains the results of a reprocessing batch run
type ReprocessingResult struct {
	ProcessedAt     time.Time `json:"processedAt"`
	FoundCount      int       `json:"foundCount"`
	RestartedCount  int       `json:"restartedCount"`
	DLQCount        int       `json:"dlqCount"`
	ErrorCount      int       `json:"errorCount"`
	SkippedCount    int       `json:"skippedCount"`
}

// FailedWorkflowInfo contains information about a failed workflow
type FailedWorkflowInfo struct {
	OrderID       string    `json:"orderId"`
	WorkflowID    string    `json:"workflowId"`
	RunID         string    `json:"runId"`
	FailureStatus string    `json:"failureStatus"`
	FailureReason string    `json:"failureReason"`
	FailedAt      time.Time `json:"failedAt"`
	RetryCount    int       `json:"retryCount"`
	CustomerID    string    `json:"customerId"`
	Priority      string    `json:"priority"`
}

// ProcessWorkflowResult contains the result of processing a single failed workflow
type ProcessWorkflowResult struct {
	OrderID       string `json:"orderId"`
	Restarted     bool   `json:"restarted"`
	MovedToDLQ    bool   `json:"movedToDlq"`
	NewWorkflowID string `json:"newWorkflowId,omitempty"`
	Error         string `json:"error,omitempty"`
}

// QueryFailedWorkflowsInput is the input for querying failed workflows
type QueryFailedWorkflowsInput struct {
	FailureStatuses []string `json:"failureStatuses"`
	MaxRetries      int      `json:"maxRetries"`
	Limit           int      `json:"limit"`
}

// ReprocessingBatchWorkflow is the scheduled workflow that triggers reprocessing batches
// This workflow is meant to be triggered by a Temporal Schedule
func ReprocessingBatchWorkflow(ctx workflow.Context, input ReprocessingBatchInput) (*ReprocessingResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting reprocessing batch workflow")

	// Use child workflow for the actual processing so the scheduled workflow returns quickly
	childOpts := workflow.ChildWorkflowOptions{
		WorkflowID:               fmt.Sprintf("reprocessing-run-%d", workflow.Now(ctx).Unix()),
		WorkflowExecutionTimeout: 15 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 2,
		},
	}
	childCtx := workflow.WithChildOptions(ctx, childOpts)

	maxOrders := ReprocessingBatchSize
	if input.MaxOrders > 0 {
		maxOrders = input.MaxOrders
	}

	var result ReprocessingResult
	err := workflow.ExecuteChildWorkflow(childCtx, ReprocessingOrchestrationWorkflow, ReprocessingOrchestrationInput{
		FailureStatuses: ReprocessableStatuses,
		MaxRetries:      MaxReprocessingRetries,
		BatchSize:       maxOrders,
	}).Get(ctx, &result)

	if err != nil {
		logger.Error("Reprocessing orchestration failed", "error", err)
		return nil, err
	}

	logger.Info("Reprocessing batch completed",
		"found", result.FoundCount,
		"restarted", result.RestartedCount,
		"dlq", result.DLQCount,
		"errors", result.ErrorCount,
	)

	return &result, nil
}

// ReprocessingOrchestrationInput is the input for the orchestration workflow
type ReprocessingOrchestrationInput struct {
	FailureStatuses []string `json:"failureStatuses"`
	MaxRetries      int      `json:"maxRetries"`
	BatchSize       int      `json:"batchSize"`
}

// ReprocessingOrchestrationWorkflow orchestrates the reprocessing of failed workflows
func ReprocessingOrchestrationWorkflow(ctx workflow.Context, input ReprocessingOrchestrationInput) (*ReprocessingResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting reprocessing orchestration",
		"failureStatuses", input.FailureStatuses,
		"maxRetries", input.MaxRetries,
		"batchSize", input.BatchSize,
	)

	result := &ReprocessingResult{
		ProcessedAt: workflow.Now(ctx),
	}

	// Activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Query for failed workflows that need reprocessing
	var failedWorkflows []FailedWorkflowInfo
	err := workflow.ExecuteActivity(ctx, "QueryFailedWorkflows", QueryFailedWorkflowsInput{
		FailureStatuses: input.FailureStatuses,
		MaxRetries:      input.MaxRetries,
		Limit:           input.BatchSize,
	}).Get(ctx, &failedWorkflows)

	if err != nil {
		logger.Error("Failed to query failed workflows", "error", err)
		return result, fmt.Errorf("failed to query failed workflows: %w", err)
	}

	result.FoundCount = len(failedWorkflows)

	if len(failedWorkflows) == 0 {
		logger.Info("No failed workflows found for reprocessing")
		return result, nil
	}

	logger.Info("Found failed workflows for reprocessing", "count", len(failedWorkflows))

	// Step 2: Process each failed workflow
	for _, fw := range failedWorkflows {
		processCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 3 * time.Minute,
			RetryPolicy: &temporal.RetryPolicy{
				InitialInterval:    time.Second,
				BackoffCoefficient: 2.0,
				MaximumAttempts:    2,
			},
		})

		var processResult ProcessWorkflowResult
		err := workflow.ExecuteActivity(processCtx, "ProcessFailedWorkflow", fw).Get(ctx, &processResult)

		if err != nil {
			logger.Warn("Failed to process workflow",
				"orderId", fw.OrderID,
				"error", err,
			)
			result.ErrorCount++
			continue
		}

		if processResult.MovedToDLQ {
			result.DLQCount++
			logger.Info("Order moved to DLQ",
				"orderId", fw.OrderID,
				"retryCount", fw.RetryCount,
			)
		} else if processResult.Restarted {
			result.RestartedCount++
			logger.Info("Order workflow restarted",
				"orderId", fw.OrderID,
				"newWorkflowId", processResult.NewWorkflowID,
			)
		} else {
			result.SkippedCount++
		}
	}

	logger.Info("Reprocessing orchestration completed",
		"found", result.FoundCount,
		"restarted", result.RestartedCount,
		"dlq", result.DLQCount,
		"errors", result.ErrorCount,
		"skipped", result.SkippedCount,
	)

	return result, nil
}
