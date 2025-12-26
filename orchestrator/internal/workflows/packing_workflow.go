package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// PackingWorkflowInput represents input for the packing workflow
type PackingWorkflowInput struct {
	OrderID string `json:"orderId"`
}

// PackingWorkflow coordinates the packing process for an order
func PackingWorkflow(ctx workflow.Context, input map[string]interface{}) (PackResult, error) {
	logger := workflow.GetLogger(ctx)

	orderID, _ := input["orderId"].(string)
	logger.Info("Starting packing workflow", "orderId", orderID)

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

	result := PackResult{}

	// Step 1: Create pack task
	logger.Info("Creating pack task", "orderId", orderID)
	var taskID string
	err := workflow.ExecuteActivity(ctx, "CreatePackTask", map[string]interface{}{
		"orderId": orderID,
	}).Get(ctx, &taskID)
	if err != nil {
		return result, fmt.Errorf("failed to create pack task: %w", err)
	}

	// Step 2: Select packaging materials
	logger.Info("Selecting packaging materials", "orderId", orderID, "taskId", taskID)
	var packageID string
	err = workflow.ExecuteActivity(ctx, "SelectPackagingMaterials", map[string]interface{}{
		"taskId":  taskID,
		"orderId": orderID,
	}).Get(ctx, &packageID)
	if err != nil {
		return result, fmt.Errorf("failed to select packaging: %w", err)
	}
	result.PackageID = packageID

	// Step 3: Pack items
	logger.Info("Packing items", "orderId", orderID, "packageId", packageID)
	err = workflow.ExecuteActivity(ctx, "PackItems", map[string]interface{}{
		"taskId":    taskID,
		"packageId": packageID,
	}).Get(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("failed to pack items: %w", err)
	}

	// Step 4: Weigh package
	logger.Info("Weighing package", "orderId", orderID, "packageId", packageID)
	var weight float64
	err = workflow.ExecuteActivity(ctx, "WeighPackage", packageID).Get(ctx, &weight)
	if err != nil {
		return result, fmt.Errorf("failed to weigh package: %w", err)
	}
	result.Weight = weight

	// Step 5: Generate shipping label
	logger.Info("Generating shipping label", "orderId", orderID, "packageId", packageID)
	var labelData map[string]interface{}
	err = workflow.ExecuteActivity(ctx, "GenerateShippingLabel", map[string]interface{}{
		"orderId":   orderID,
		"packageId": packageID,
		"weight":    weight,
	}).Get(ctx, &labelData)
	if err != nil {
		return result, fmt.Errorf("failed to generate shipping label: %w", err)
	}

	result.TrackingNumber = labelData["trackingNumber"].(string)
	result.Carrier = labelData["carrier"].(string)

	// Step 6: Apply label to package
	logger.Info("Applying label to package", "orderId", orderID, "packageId", packageID, "trackingNumber", result.TrackingNumber)
	err = workflow.ExecuteActivity(ctx, "ApplyLabelToPackage", map[string]interface{}{
		"packageId":      packageID,
		"trackingNumber": result.TrackingNumber,
	}).Get(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("failed to apply label: %w", err)
	}

	// Step 7: Seal package
	logger.Info("Sealing package", "orderId", orderID, "packageId", packageID)
	err = workflow.ExecuteActivity(ctx, "SealPackage", packageID).Get(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("failed to seal package: %w", err)
	}

	logger.Info("Packing completed successfully",
		"orderId", orderID,
		"packageId", packageID,
		"trackingNumber", result.TrackingNumber,
		"carrier", result.Carrier,
		"weight", result.Weight,
	)

	return result, nil
}
