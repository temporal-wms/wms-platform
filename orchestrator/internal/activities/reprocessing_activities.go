package activities

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"

	"github.com/wms-platform/orchestrator/internal/activities/clients"
	"github.com/wms-platform/orchestrator/internal/workflows"
)

// QueryFailedWorkflows queries for failed workflows that are eligible for reprocessing
func (a *ReprocessingActivities) QueryFailedWorkflows(ctx context.Context, input workflows.QueryFailedWorkflowsInput) ([]workflows.FailedWorkflowInfo, error) {
	a.logger.Info("Querying for failed workflows",
		"failureStatuses", input.FailureStatuses,
		"maxRetries", input.MaxRetries,
		"limit", input.Limit,
	)

	// Query order service for orders with retry metadata
	response, err := a.clients.GetEligibleOrders(ctx, input.FailureStatuses, input.MaxRetries, input.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query failed workflows: %w", err)
	}

	// Convert to workflow types
	result := make([]workflows.FailedWorkflowInfo, 0, len(response.Data))
	for _, order := range response.Data {
		result = append(result, workflows.FailedWorkflowInfo{
			OrderID:       order.OrderID,
			WorkflowID:    order.WorkflowID,
			RunID:         order.RunID,
			FailureStatus: order.FailureStatus,
			FailureReason: order.FailureReason,
			FailedAt:      order.FailedAt,
			RetryCount:    order.RetryCount,
			CustomerID:    order.CustomerID,
			Priority:      order.Priority,
		})
	}

	a.logger.Info("Found failed workflows", "count", len(result))
	return result, nil
}

// ProcessFailedWorkflow processes a single failed workflow - either restarts it or moves to DLQ
func (a *ReprocessingActivities) ProcessFailedWorkflow(ctx context.Context, fw workflows.FailedWorkflowInfo) (*workflows.ProcessWorkflowResult, error) {
	a.logger.Info("Processing failed workflow",
		"orderId", fw.OrderID,
		"retryCount", fw.RetryCount,
		"failureStatus", fw.FailureStatus,
	)

	result := &workflows.ProcessWorkflowResult{
		OrderID: fw.OrderID,
	}

	// Check if should move to DLQ (max retries exceeded)
	if fw.RetryCount >= workflows.MaxReprocessingRetries {
		a.logger.Info("Order exceeded max retries, moving to DLQ",
			"orderId", fw.OrderID,
			"retryCount", fw.RetryCount,
		)

		err := a.moveToDeadLetterQueue(ctx, fw)
		if err != nil {
			return nil, fmt.Errorf("failed to move to DLQ: %w", err)
		}

		// Record DLQ metric
		if a.failureMetrics != nil {
			a.failureMetrics.RecordMovedToDLQ(fw.FailureStatus)
		}

		result.MovedToDLQ = true
		return result, nil
	}

	// Increment retry count
	err := a.incrementRetryCount(ctx, fw)
	if err != nil {
		// Record retry failure metric
		if a.failureMetrics != nil {
			a.failureMetrics.RecordRetryFailure()
		}
		return nil, fmt.Errorf("failed to increment retry count: %w", err)
	}

	// Reset order status for retry
	err = a.resetOrderForRetry(ctx, fw.OrderID)
	if err != nil {
		// Record retry failure metric
		if a.failureMetrics != nil {
			a.failureMetrics.RecordRetryFailure()
		}
		return nil, fmt.Errorf("failed to reset order: %w", err)
	}

	// Restart the workflow
	newWorkflowID, err := a.restartWorkflow(ctx, fw)
	if err != nil {
		// Record retry failure metric
		if a.failureMetrics != nil {
			a.failureMetrics.RecordRetryFailure()
		}
		return nil, fmt.Errorf("failed to restart workflow: %w", err)
	}

	// Record retry success and workflow retry metrics
	if a.failureMetrics != nil {
		a.failureMetrics.RecordRetrySuccess()
		a.failureMetrics.RecordWorkflowRetry("OrderFulfillmentWorkflow")
	}

	result.Restarted = true
	result.NewWorkflowID = newWorkflowID

	a.logger.Info("Workflow restarted successfully",
		"orderId", fw.OrderID,
		"newWorkflowId", newWorkflowID,
	)

	return result, nil
}

// moveToDeadLetterQueue moves an order to the dead letter queue
func (a *ReprocessingActivities) moveToDeadLetterQueue(ctx context.Context, fw workflows.FailedWorkflowInfo) error {
	req := &clients.MoveToDeadLetterRequest{
		FailureStatus: fw.FailureStatus,
		FailureReason: fw.FailureReason,
		RetryCount:    fw.RetryCount,
		WorkflowID:    fw.WorkflowID,
		RunID:         fw.RunID,
	}

	return a.clients.MoveToDeadLetter(ctx, fw.OrderID, req)
}

// incrementRetryCount increments the retry count for an order
func (a *ReprocessingActivities) incrementRetryCount(ctx context.Context, fw workflows.FailedWorkflowInfo) error {
	req := &clients.IncrementRetryCountRequest{
		FailureStatus: fw.FailureStatus,
		FailureReason: fw.FailureReason,
		WorkflowID:    fw.WorkflowID,
		RunID:         fw.RunID,
	}

	return a.clients.IncrementRetryCount(ctx, fw.OrderID, req)
}

// resetOrderForRetry resets an order to allow reprocessing
func (a *ReprocessingActivities) resetOrderForRetry(ctx context.Context, orderID string) error {
	return a.clients.ResetOrderForRetry(ctx, orderID)
}

// restartWorkflow starts a new workflow execution for the order
func (a *ReprocessingActivities) restartWorkflow(ctx context.Context, fw workflows.FailedWorkflowInfo) (string, error) {
	temporalClient, ok := a.temporalClient.(client.Client)
	if !ok {
		return "", fmt.Errorf("temporal client not configured")
	}

	// Get order details from order service
	orderDetails, err := a.clients.GetOrder(ctx, fw.OrderID)
	if err != nil {
		return "", fmt.Errorf("failed to get order details: %w", err)
	}

	// Create new workflow ID for the retry
	newWorkflowID := fmt.Sprintf("order-fulfillment-%s-retry-%d", fw.OrderID, time.Now().Unix())

	// Build workflow input
	input := workflows.OrderFulfillmentInput{
		OrderID:            fw.OrderID,
		CustomerID:         orderDetails.CustomerID,
		Items:              convertOrderItems(orderDetails.Items),
		Priority:           orderDetails.Priority,
		PromisedDeliveryAt: orderDetails.PromisedDeliveryAt,
		IsMultiItem:        len(orderDetails.Items) > 1,
	}

	// Start the workflow
	workflowOptions := client.StartWorkflowOptions{
		ID:        newWorkflowID,
		TaskQueue: "orchestrator",
	}

	we, err := temporalClient.ExecuteWorkflow(ctx, workflowOptions, "OrderFulfillmentWorkflow", input)
	if err != nil {
		return "", fmt.Errorf("failed to start workflow: %w", err)
	}

	a.logger.Info("Started new workflow",
		"workflowId", we.GetID(),
		"runId", we.GetRunID(),
		"orderId", fw.OrderID,
	)

	return we.GetID(), nil
}

// convertOrderItems converts order items from client type to workflow type
func convertOrderItems(items []clients.OrderItem) []workflows.Item {
	result := make([]workflows.Item, 0, len(items))
	for _, item := range items {
		result = append(result, workflows.Item{
			SKU:      item.SKU,
			Quantity: item.Quantity,
			Weight:   item.Weight,
		})
	}
	return result
}
