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
	// ContinueAsNew support - accumulated results from previous runs
	AccumulatedResult *ReprocessingResult `json:"accumulatedResult,omitempty"`
	// Internal continuation tracking
	ContinuationCount int `json:"continuationCount,omitempty"`
}

// ReprocessingOrchestrationWorkflow orchestrates the reprocessing of failed workflows
func ReprocessingOrchestrationWorkflow(ctx workflow.Context, input ReprocessingOrchestrationInput) (*ReprocessingResult, error) {
	logger := workflow.GetLogger(ctx)

	// Workflow versioning for safe deployments
	// Version 1: Added ContinueAsNew support for handling large batches
	version := workflow.GetVersion(ctx, "ReprocessingOrchestrationWorkflow", workflow.DefaultVersion, ReprocessingOrchestrationWorkflowVersion)
	logger.Info("Workflow version", "version", version)

	logger.Info("Starting reprocessing orchestration",
		"failureStatuses", input.FailureStatuses,
		"maxRetries", input.MaxRetries,
		"batchSize", input.BatchSize,
		"continuationCount", input.ContinuationCount,
	)

	// Initialize or use accumulated results from previous continuation
	var result *ReprocessingResult
	if input.AccumulatedResult != nil {
		result = input.AccumulatedResult
		logger.Info("Continuing from previous run",
			"previousFoundCount", result.FoundCount,
			"previousRestartedCount", result.RestartedCount,
		)
	} else {
		result = &ReprocessingResult{
			ProcessedAt: workflow.Now(ctx),
		}
	}

	// ContinueAsNew safety limit: process max 1000 workflows per continuation
	// This prevents hitting the 50K event history limit
	const maxWorkflowsPerContinuation = 1000

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
	// Use maxWorkflowsPerContinuation to limit query size for ContinueAsNew safety
	queryLimit := input.BatchSize
	if queryLimit > maxWorkflowsPerContinuation {
		queryLimit = maxWorkflowsPerContinuation
	}

	var failedWorkflows []FailedWorkflowInfo
	err := workflow.ExecuteActivity(ctx, "QueryFailedWorkflows", QueryFailedWorkflowsInput{
		FailureStatuses: input.FailureStatuses,
		MaxRetries:      input.MaxRetries,
		Limit:           queryLimit,
	}).Get(ctx, &failedWorkflows)

	if err != nil {
		logger.Error("Failed to query failed workflows", "error", err)
		return result, fmt.Errorf("failed to query failed workflows: %w", err)
	}

	currentBatchCount := len(failedWorkflows)
	result.FoundCount += currentBatchCount

	if currentBatchCount == 0 {
		logger.Info("No more failed workflows found for reprocessing - completing",
			"totalFound", result.FoundCount,
			"totalRestarted", result.RestartedCount,
			"totalDLQ", result.DLQCount,
			"totalErrors", result.ErrorCount,
		)
		return result, nil
	}

	logger.Info("Found failed workflows for this batch",
		"batchCount", currentBatchCount,
		"continuationCount", input.ContinuationCount,
	)

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

	logger.Info("Batch processing completed",
		"batchFound", currentBatchCount,
		"totalFound", result.FoundCount,
		"totalRestarted", result.RestartedCount,
		"totalDLQ", result.DLQCount,
		"totalErrors", result.ErrorCount,
		"totalSkipped", result.SkippedCount,
	)

	// Check if we should continue processing more workflows
	// If we found exactly queryLimit workflows, there might be more to process
	if currentBatchCount >= queryLimit {
		logger.Info("Batch limit reached - using ContinueAsNew to process more workflows",
			"queryLimit", queryLimit,
			"continuationCount", input.ContinuationCount,
		)

		// Use ContinueAsNew to start a fresh workflow execution with accumulated results
		return result, workflow.NewContinueAsNewError(ctx, ReprocessingOrchestrationWorkflow, ReprocessingOrchestrationInput{
			FailureStatuses:   input.FailureStatuses,
			MaxRetries:        input.MaxRetries,
			BatchSize:         input.BatchSize,
			AccumulatedResult: result,
			ContinuationCount: input.ContinuationCount + 1,
		})
	}

	// No more workflows to process - return final results
	logger.Info("Reprocessing orchestration completed - no more workflows to process",
		"totalFound", result.FoundCount,
		"totalRestarted", result.RestartedCount,
		"totalDLQ", result.DLQCount,
		"totalErrors", result.ErrorCount,
		"totalSkipped", result.SkippedCount,
		"continuations", input.ContinuationCount,
	)

	return result, nil
}
