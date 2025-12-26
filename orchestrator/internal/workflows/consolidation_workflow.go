package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ConsolidationWorkflowInput represents input for the consolidation workflow
type ConsolidationWorkflowInput struct {
	OrderID     string       `json:"orderId"`
	PickedItems []PickedItem `json:"pickedItems"`
}

// ConsolidationWorkflow coordinates the consolidation of multi-item orders
func ConsolidationWorkflow(ctx workflow.Context, input map[string]interface{}) error {
	logger := workflow.GetLogger(ctx)

	orderID, _ := input["orderId"].(string)
	logger.Info("Starting consolidation workflow", "orderId", orderID)

	// Activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Create consolidation unit
	logger.Info("Creating consolidation unit", "orderId", orderID)
	var consolidationID string
	err := workflow.ExecuteActivity(ctx, "CreateConsolidationUnit", map[string]interface{}{
		"orderId":     orderID,
		"pickedItems": input["pickedItems"],
	}).Get(ctx, &consolidationID)
	if err != nil {
		return fmt.Errorf("failed to create consolidation unit: %w", err)
	}

	// Step 2: Consolidate items from different totes
	logger.Info("Consolidating items", "orderId", orderID, "consolidationId", consolidationID)
	err = workflow.ExecuteActivity(ctx, "ConsolidateItems", map[string]interface{}{
		"consolidationId": consolidationID,
		"pickedItems":     input["pickedItems"],
	}).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to consolidate items: %w", err)
	}

	// Step 3: Verify all items consolidated
	logger.Info("Verifying consolidation", "orderId", orderID, "consolidationId", consolidationID)
	var verified bool
	err = workflow.ExecuteActivity(ctx, "VerifyConsolidation", consolidationID).Get(ctx, &verified)
	if err != nil {
		return fmt.Errorf("failed to verify consolidation: %w", err)
	}

	if !verified {
		return fmt.Errorf("consolidation verification failed for order %s", orderID)
	}

	// Step 4: Complete consolidation
	logger.Info("Completing consolidation", "orderId", orderID, "consolidationId", consolidationID)
	err = workflow.ExecuteActivity(ctx, "CompleteConsolidation", consolidationID).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to complete consolidation: %w", err)
	}

	logger.Info("Consolidation completed successfully", "orderId", orderID, "consolidationId", consolidationID)
	return nil
}
