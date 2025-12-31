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
	WaveID  string `json:"waveId"`
}

// PackingWorkflow coordinates the packing process for an order
func PackingWorkflow(ctx workflow.Context, input map[string]interface{}) (PackResult, error) {
	logger := workflow.GetLogger(ctx)

	orderID, _ := input["orderId"].(string)
	waveID, _ := input["waveId"].(string)
	logger.Info("Starting packing workflow", "orderId", orderID, "waveId", waveID)

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
		"waveId":  waveID,
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
	// carrier is returned as a nested object (ShipmentCarrier), extract the code
	if carrierMap, ok := labelData["carrier"].(map[string]interface{}); ok {
		if code, ok := carrierMap["code"].(string); ok {
			result.Carrier = code
		} else if name, ok := carrierMap["name"].(string); ok {
			result.Carrier = name
		}
	} else if carrierStr, ok := labelData["carrier"].(string); ok {
		result.Carrier = carrierStr
	}

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

	// Step 8: Mark inventory as packed (hard allocation status update)
	// Extract allocation info from input if available
	if allocationIDs, ok := input["allocationIds"].([]interface{}); ok && len(allocationIDs) > 0 {
		logger.Info("Marking inventory as packed", "orderId", orderID, "allocationCount", len(allocationIDs))

		// Build pack items from allocations and picked items
		packItems := make([]map[string]interface{}, 0)
		pickedItems, _ := input["pickedItems"].([]interface{})

		for i, allocID := range allocationIDs {
			if strAllocID, ok := allocID.(string); ok {
				sku := ""
				// Try to get SKU from picked items
				if i < len(pickedItems) {
					if itemMap, ok := pickedItems[i].(map[string]interface{}); ok {
						if skuVal, ok := itemMap["sku"].(string); ok {
							sku = skuVal
						}
					}
				}
				packItems = append(packItems, map[string]interface{}{
					"sku":          sku,
					"allocationId": strAllocID,
				})
			}
		}

		if len(packItems) > 0 {
			err = workflow.ExecuteActivity(ctx, "PackInventory", map[string]interface{}{
				"orderId":  orderID,
				"packedBy": "packing-station",
				"items":    packItems,
			}).Get(ctx, nil)
			if err != nil {
				// Log but don't fail - inventory status can be reconciled
				logger.Warn("Failed to mark inventory as packed, continuing workflow",
					"orderId", orderID,
					"error", err,
				)
			} else {
				logger.Info("Inventory marked as packed successfully", "orderId", orderID)
			}
		}
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
