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
}

// PickingWorkflow coordinates the picking process for an order
func PickingWorkflow(ctx workflow.Context, input map[string]interface{}) (PickResult, error) {
	logger := workflow.GetLogger(ctx)

	orderID, _ := input["orderId"].(string)
	waveID, _ := input["waveId"].(string)

	logger.Info("Starting picking workflow", "orderId", orderID, "waveId", waveID)

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
					pickedItems = append(pickedItems, PickedItem{
						SKU:        itemMap["sku"].(string),
						Quantity:   int(itemMap["quantity"].(float64)),
						LocationID: itemMap["locationId"].(string),
						ToteID:     itemMap["toteId"].(string),
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

	logger.Info("Picking completed successfully",
		"orderId", orderID,
		"taskId", taskID,
		"itemsCount", len(pickedItems),
	)

	return result, nil
}
