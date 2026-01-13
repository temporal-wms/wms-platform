package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// PickingWorkflowInput represents input for the picking workflow
type PickingWorkflowInput struct {
	OrderID string      `json:"orderId"`
	WaveID  string      `json:"waveId"`
	Route   RouteResult `json:"route"`
	// Unit-level tracking fields
	UnitIDs []string `json:"unitIds,omitempty"` // Specific units to pick
	PathID  string   `json:"pathId,omitempty"`  // Process path ID for consistency
	// Multi-tenant context
	TenantID    string `json:"tenantId"`
	FacilityID  string `json:"facilityId"`
	WarehouseID string `json:"warehouseId"`
}

// OrchestratedPickingWorkflow coordinates the picking process for an order with
// enhanced features like unit-level tracking and inventory staging.
// Note: For simple picking, use PickingWorkflow from picking-service on picking-queue.
func OrchestratedPickingWorkflow(ctx workflow.Context, input map[string]interface{}) (PickResult, error) {
	logger := workflow.GetLogger(ctx)

	// Workflow versioning for safe deployments
	version := workflow.GetVersion(ctx, "OrchestratedPickingWorkflow", workflow.DefaultVersion, OrchestratedPickingWorkflowVersion)
	logger.Info("Workflow version", "version", version)

	orderID, _ := input["orderId"].(string)
	waveID, _ := input["waveId"].(string)

	// Extract tenant context
	tenantID, _ := input["tenantId"].(string)
	facilityID, _ := input["facilityId"].(string)
	warehouseID, _ := input["warehouseId"].(string)

	// Set tenant context for activities
	if tenantID != "" {
		ctx = workflow.WithValue(ctx, "tenantId", tenantID)
	}
	if facilityID != "" {
		ctx = workflow.WithValue(ctx, "facilityId", facilityID)
	}
	if warehouseID != "" {
		ctx = workflow.WithValue(ctx, "warehouseId", warehouseID)
	}

	// Extract unit-level tracking fields (now always enabled)
	var unitIDs []string
	var pathID string
	if ids, ok := input["unitIds"].([]interface{}); ok {
		for _, id := range ids {
			if strID, ok := id.(string); ok {
				unitIDs = append(unitIDs, strID)
			}
		}
	}
	if pid, ok := input["pathId"].(string); ok {
		pathID = pid
	}

	logger.Info("Starting picking workflow", "orderId", orderID, "waveId", waveID, "unitCount", len(unitIDs))

	// Activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	result := PickResult{
		Success: false,
	}

	// Step 1: Create pick task
	logger.Info("Creating pick task", "orderId", orderID)
	var taskID string
	err := workflow.ExecuteActivity(ctx, "CreatePickTask", map[string]interface{}{
		"orderId": orderID,
		"waveId":  waveID,
		"route":   input["route"],
		"items":   input["items"],
	}).Get(ctx, &taskID)
	if err != nil {
		return result, fmt.Errorf("failed to create pick task: %w", err)
	}
	result.TaskID = taskID

	// Step 2: Assign worker to task
	logger.Info("Assigning worker to pick task", "orderId", orderID, "taskId", taskID)
	var workerID string
	err = workflow.ExecuteActivity(ctx, "AssignPickerToTask", map[string]interface{}{
		"taskId": taskID,
		"waveId": waveID,
	}).Get(ctx, &workerID)
	if err != nil {
		return result, fmt.Errorf("failed to assign worker: %w", err)
	}

	// Step 3: Wait for picking completion (via signal or timeout)
	logger.Info("Waiting for pick completion", "orderId", orderID, "taskId", taskID, "workerId", workerID)

	pickCompletionSignal := workflow.GetSignalChannel(ctx, "pickCompleted")
	pickTimeout := 30 * time.Minute

	selector := workflow.NewSelector(ctx)
	var pickCompleted bool
	var pickedItems []PickedItem

	selector.AddReceive(pickCompletionSignal, func(c workflow.ReceiveChannel, more bool) {
		var completion map[string]interface{}
		c.Receive(ctx, &completion)
		pickCompleted = true

		// Extract picked items from signal
		if items, ok := completion["pickedItems"].([]interface{}); ok {
			for _, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					// Safe type assertions with defaults for nil values
					sku, _ := itemMap["sku"].(string)
					quantity, _ := itemMap["quantity"].(float64)
					locationID, _ := itemMap["locationId"].(string)
					toteID, _ := itemMap["toteId"].(string)

					pickedItems = append(pickedItems, PickedItem{
						SKU:        sku,
						Quantity:   int(quantity),
						LocationID: locationID,
						ToteID:     toteID,
					})
				}
			}
		}
	})

	selector.AddFuture(workflow.NewTimer(ctx, pickTimeout), func(f workflow.Future) {
		logger.Warn("Pick timeout", "orderId", orderID, "taskId", taskID)
	})

	selector.Select(ctx)

	if !pickCompleted {
		return result, fmt.Errorf("picking timeout for order %s", orderID)
	}

	result.PickedItems = pickedItems
	result.Success = true

	// Step 4: Stage inventory (convert soft reservation to hard allocation)
	// This creates a physical claim on the inventory that cannot be released without return-to-shelf
	// NOTE: Staging must happen BEFORE inventory pick to preserve active reservations
	logger.Info("Staging inventory (hard allocation)", "orderId", orderID, "itemCount", len(pickedItems))

	// Declare variables before any goto statements
	var stageResult map[string]interface{}
	var stagingLocationID string
	var getReservationItems []map[string]interface{}
	var reservationResult map[string]interface{}
	var reservations map[string]interface{}
	var stageItems []map[string]interface{}

	// Get the first tote ID for staging location (items should be in same tote)
	stagingLocationID = "STAGING-DEFAULT"
	if len(pickedItems) > 0 && pickedItems[0].ToteID != "" {
		stagingLocationID = pickedItems[0].ToteID
	}

	// First, get reservation IDs for the order's items
	getReservationItems = make([]map[string]interface{}, len(pickedItems))
	for i, item := range pickedItems {
		getReservationItems[i] = map[string]interface{}{
			"sku": item.SKU,
		}
	}

	err = workflow.ExecuteActivity(ctx, "GetReservationIDs", map[string]interface{}{
		"orderId": orderID,
		"items":   getReservationItems,
	}).Get(ctx, &reservationResult)
	if err != nil {
		logger.Warn("Failed to get reservation IDs, continuing workflow",
			"orderId", orderID,
			"error", err,
		)
		// Skip staging if we can't get reservation IDs
		goto skipStaging
	}

	// Extract reservations map (SKU -> ReservationID)
	if resMap, ok := reservationResult["reservations"].(map[string]interface{}); ok {
		reservations = resMap
	} else {
		logger.Warn("No reservations found in response, skipping staging", "orderId", orderID)
		goto skipStaging
	}

	// Build stage inventory items with actual reservation IDs
	stageItems = make([]map[string]interface{}, 0, len(pickedItems))
	for _, item := range pickedItems {
		if resID, ok := reservations[item.SKU].(string); ok {
			stageItems = append(stageItems, map[string]interface{}{
				"sku":           item.SKU,
				"reservationId": resID,
			})
		} else {
			logger.Warn("No reservation ID found for SKU, skipping staging",
				"orderId", orderID,
				"sku", item.SKU)
		}
	}

	if len(stageItems) == 0 {
		logger.Warn("No items to stage (no reservation IDs found), skipping staging", "orderId", orderID)
		goto skipStaging
	}

	err = workflow.ExecuteActivity(ctx, "StageInventory", map[string]interface{}{
		"orderId":           orderID,
		"stagingLocationId": stagingLocationID,
		"stagedBy":          workerID,
		"items":             stageItems,
	}).Get(ctx, &stageResult)
	if err != nil {
		// Log but don't fail - staging can be reconciled later
		logger.Warn("Failed to stage inventory, continuing workflow",
			"orderId", orderID,
			"error", err,
		)
	} else {
		// Store allocation IDs for downstream workflows (packing, shipping)
		if allocationIDs, ok := stageResult["allocationIds"].([]interface{}); ok {
			result.AllocationIDs = make([]string, len(allocationIDs))
			for i, id := range allocationIDs {
				if strID, ok := id.(string); ok {
					result.AllocationIDs[i] = strID
				}
			}
		}
		logger.Info("Inventory staged successfully (hard allocation created)",
			"orderId", orderID,
			"allocationCount", len(result.AllocationIDs),
		)
	}

skipStaging:
	// Continue with workflow even if staging was skipped

	// Note: Unit-level pick confirmation is handled during physical picking (Step 3).
	// Units are marked as picked when workers scan them into totes, so no additional
	// confirmation is needed here. This prevents duplicate pick attempts that would
	// fail with "cannot pick unit in status picked" errors.

	// Suppress unused variable warnings (will be used in future enhancements)
	_ = pathID
	_ = unitIDs

	logger.Info("Picking completed successfully",
		"orderId", orderID,
		"taskId", taskID,
		"itemsCount", len(pickedItems),
	)

	return result, nil
}
