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

	// Extract unit-level tracking fields
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
	useUnitTracking := len(unitIDs) > 0

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

	// Step 4: Confirm inventory pick (decrement stock)
	if len(pickedItems) > 0 {
		logger.Info("Confirming inventory pick", "orderId", orderID, "itemCount", len(pickedItems))

		// Convert picked items to inventory pick format
		inventoryPickItems := make([]map[string]interface{}, len(pickedItems))
		for i, item := range pickedItems {
			inventoryPickItems[i] = map[string]interface{}{
				"sku":        item.SKU,
				"quantity":   item.Quantity,
				"locationId": item.LocationID,
			}
		}

		err = workflow.ExecuteActivity(ctx, "ConfirmInventoryPick", map[string]interface{}{
			"orderId":     orderID,
			"pickedItems": inventoryPickItems,
		}).Get(ctx, nil)
		if err != nil {
			// Log but don't fail the workflow - inventory can be reconciled later
			logger.Warn("Failed to confirm inventory pick, continuing workflow",
				"orderId", orderID,
				"error", err,
			)
		} else {
			logger.Info("Inventory pick confirmed successfully", "orderId", orderID)
		}

		// Step 5: Stage inventory (convert soft reservation to hard allocation)
		// This creates a physical claim on the inventory that cannot be released without return-to-shelf
		logger.Info("Staging inventory (hard allocation)", "orderId", orderID, "itemCount", len(pickedItems))

		// Get the first tote ID for staging location (items should be in same tote)
		stagingLocationID := "STAGING-DEFAULT"
		if len(pickedItems) > 0 && pickedItems[0].ToteID != "" {
			stagingLocationID = pickedItems[0].ToteID
		}

		// Build stage inventory items - each picked item needs its reservation staged
		stageItems := make([]map[string]interface{}, len(pickedItems))
		for i, item := range pickedItems {
			// ReservationID is typically orderID-sku format from the reservation system
			stageItems[i] = map[string]interface{}{
				"sku":           item.SKU,
				"reservationId": fmt.Sprintf("%s-%s", orderID, item.SKU),
			}
		}

		var stageResult map[string]interface{}
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
	}

	// Step 6: Unit-level pick confirmation (if unit tracking enabled)
	if useUnitTracking && len(unitIDs) > 0 {
		logger.Info("Confirming unit-level picks", "orderId", orderID, "unitCount", len(unitIDs))

		var pickedUnitIDs []string
		var failedUnitIDs []string
		var exceptionIDs []string

		// Get the first tote ID from picked items
		toteID := "TOTE-DEFAULT"
		if len(pickedItems) > 0 && pickedItems[0].ToteID != "" {
			toteID = pickedItems[0].ToteID
		}

		// Get a station ID (could come from route or task)
		stationID := "PICK-STATION-DEFAULT"

		for _, unitID := range unitIDs {
			// Attempt to confirm pick for each unit
			err := workflow.ExecuteActivity(ctx, "ConfirmUnitPick", map[string]interface{}{
				"unitId":    unitID,
				"toteId":    toteID,
				"pickerId":  workerID,
				"stationId": stationID,
			}).Get(ctx, nil)

			if err != nil {
				logger.Warn("Failed to confirm unit pick, creating exception",
					"orderId", orderID,
					"unitId", unitID,
					"error", err,
				)
				failedUnitIDs = append(failedUnitIDs, unitID)

				// Create exception for failed unit
				var exceptionResult map[string]interface{}
				exErr := workflow.ExecuteActivity(ctx, "CreateUnitException", map[string]interface{}{
					"unitId":        unitID,
					"exceptionType": "picking_failure",
					"stage":         "picking",
					"description":   fmt.Sprintf("Failed to confirm pick: %v", err),
					"stationId":     stationID,
					"reportedBy":    workerID,
				}).Get(ctx, &exceptionResult)
				if exErr == nil {
					if exID, ok := exceptionResult["exceptionId"].(string); ok {
						exceptionIDs = append(exceptionIDs, exID)
					}
				}
			} else {
				pickedUnitIDs = append(pickedUnitIDs, unitID)
			}
		}

		result.PickedUnitIDs = pickedUnitIDs
		result.FailedUnitIDs = failedUnitIDs
		result.ExceptionIDs = exceptionIDs

		logger.Info("Unit-level pick confirmation completed",
			"orderId", orderID,
			"pickedUnits", len(pickedUnitIDs),
			"failedUnits", len(failedUnitIDs),
		)

		// If all units failed, consider the pick failed
		if len(pickedUnitIDs) == 0 && len(failedUnitIDs) > 0 {
			result.Success = false
			return result, fmt.Errorf("all units failed picking for order %s", orderID)
		}
	}

	// Suppress unused variable warning for pathID (will be used in future enhancements)
	_ = pathID

	logger.Info("Picking completed successfully",
		"orderId", orderID,
		"taskId", taskID,
		"itemsCount", len(pickedItems),
	)

	return result, nil
}
