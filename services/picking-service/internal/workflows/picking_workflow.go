package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// PickingWorkflowInput represents the input for the picking workflow
type PickingWorkflowInput struct {
	OrderID string      `json:"orderId"`
	WaveID  string      `json:"waveId"`
	Route   RouteResult `json:"route"`
	// Multi-tenant context
	TenantID    string `json:"tenantId"`
	FacilityID  string `json:"facilityId"`
	WarehouseID string `json:"warehouseId"`
}

// RouteResult represents the route from the orchestrator
type RouteResult struct {
	RouteID           string      `json:"routeId"`
	Stops             []RouteStop `json:"stops"`
	EstimatedDistance float64     `json:"estimatedDistance"`
	Strategy          string      `json:"strategy"`
}

// RouteStop represents a stop in the pick route
type RouteStop struct {
	LocationID string `json:"locationId"`
	SKU        string `json:"sku"`
	Quantity   int    `json:"quantity"`
}

// PickingWorkflowResult represents the result of the picking workflow
type PickingWorkflowResult struct {
	TaskID      string       `json:"taskId"`
	PickedItems []PickedItem `json:"pickedItems"`
	Success     bool         `json:"success"`
	Error       string       `json:"error,omitempty"`
}

// PickedItem represents a picked item
type PickedItem struct {
	SKU        string `json:"sku"`
	Quantity   int    `json:"quantity"`
	LocationID string `json:"locationId"`
	ToteID     string `json:"toteId"`
}

// PickingWorkflow orchestrates the picking process for an order
// Using typed struct input to ensure determinism and type safety
func PickingWorkflow(ctx workflow.Context, input PickingWorkflowInput) (*PickingWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)

	// Extract input from typed struct
	orderID := input.OrderID
	waveID := input.WaveID
	routeID := input.Route.RouteID
	stops := input.Route.Stops

	logger.Info("Starting picking workflow", "orderId", orderID, "waveId", waveID, "routeId", routeID)

	result := &PickingWorkflowResult{
		Success: false,
	}

	// Set tenant context for activities
	if input.TenantID != "" {
		ctx = workflow.WithValue(ctx, "tenantId", input.TenantID)
	}
	if input.FacilityID != "" {
		ctx = workflow.WithValue(ctx, "facilityId", input.FacilityID)
	}
	if input.WarehouseID != "" {
		ctx = workflow.WithValue(ctx, "warehouseId", input.WarehouseID)
	}

	// Activity options with proper timeouts for picking operations
	// ScheduleToCloseTimeout: Total time including retries
	// StartToCloseTimeout: Time for a single attempt
	// HeartbeatTimeout: Detect stuck workers for long-running pick tasks
	ao := workflow.ActivityOptions{
		ScheduleToCloseTimeout: 30 * time.Minute, // Total time including retries
		StartToCloseTimeout:    10 * time.Minute,
		HeartbeatTimeout:       30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Create pick task
	logger.Info("Creating pick task", "orderId", orderID)
	var taskID string
	err := workflow.ExecuteActivity(ctx, "CreatePickTask", map[string]interface{}{
		"orderId": orderID,
		"waveId":  waveID,
		"routeId": routeID,
		"stops":   stops,
	}).Get(ctx, &taskID)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create pick task: %v", err)
		return result, err
	}
	result.TaskID = taskID

	// Step 2: Wait for worker assignment
	logger.Info("Waiting for worker assignment", "taskId", taskID)
	assignmentSignal := workflow.GetSignalChannel(ctx, "workerAssigned")

	var workerID string
	assignmentCtx, cancelAssignment := workflow.WithCancel(ctx)
	defer cancelAssignment()

	selector := workflow.NewSelector(assignmentCtx)

	var assigned bool
	selector.AddReceive(assignmentSignal, func(c workflow.ReceiveChannel, more bool) {
		var assignment struct {
			WorkerID string `json:"workerId"`
			ToteID   string `json:"toteId"`
		}
		c.Receive(assignmentCtx, &assignment)
		workerID = assignment.WorkerID
		assigned = true
	})

	// Timeout for assignment - 30 minutes
	selector.AddFuture(workflow.NewTimer(assignmentCtx, 30*time.Minute), func(f workflow.Future) {
		logger.Warn("Worker assignment timeout", "taskId", taskID)
	})

	selector.Select(assignmentCtx)

	if !assigned {
		result.Error = "worker assignment timeout"
		return result, fmt.Errorf("worker assignment timeout for task %s", taskID)
	}

	logger.Info("Worker assigned", "taskId", taskID, "workerId", workerID)

	// Step 3: Process picks with signal-based updates
	logger.Info("Processing picks", "taskId", taskID)

	pickSignal := workflow.GetSignalChannel(ctx, "itemPicked")
	completeSignal := workflow.GetSignalChannel(ctx, "pickingComplete")
	exceptionSignal := workflow.GetSignalChannel(ctx, "pickException")

	pickedItems := make([]PickedItem, 0)
	pendingItems := len(stops)
	pickingComplete := false

	for !pickingComplete && pendingItems > 0 {
		pickCtx, cancelPick := workflow.WithCancel(ctx)

		pickSelector := workflow.NewSelector(pickCtx)

		// Handle item picked signal
		pickSelector.AddReceive(pickSignal, func(c workflow.ReceiveChannel, more bool) {
			var picked PickedItem
			c.Receive(pickCtx, &picked)
			pickedItems = append(pickedItems, picked)
			pendingItems--
			logger.Info("Item picked", "taskId", taskID, "sku", picked.SKU, "remaining", pendingItems)
		})

		// Handle picking complete signal
		pickSelector.AddReceive(completeSignal, func(c workflow.ReceiveChannel, more bool) {
			var complete struct {
				Success bool `json:"success"`
			}
			c.Receive(pickCtx, &complete)
			pickingComplete = true
			logger.Info("Picking completed", "taskId", taskID, "totalPicked", len(pickedItems))
		})

		// Handle exception signal
		pickSelector.AddReceive(exceptionSignal, func(c workflow.ReceiveChannel, more bool) {
			var exception struct {
				SKU       string `json:"sku"`
				Reason    string `json:"reason"`
				Available int    `json:"available"`
			}
			c.Receive(pickCtx, &exception)
			pendingItems--
			logger.Warn("Pick exception", "taskId", taskID, "sku", exception.SKU, "reason", exception.Reason)
		})

		// Activity timeout - 2 hours for entire picking
		pickSelector.AddFuture(workflow.NewTimer(pickCtx, 2*time.Hour), func(f workflow.Future) {
			pickingComplete = true
			result.Error = "picking timeout"
			logger.Warn("Picking timeout", "taskId", taskID)
		})

		pickSelector.Select(pickCtx)
		cancelPick()
	}

	// Step 4: Complete the pick task
	logger.Info("Completing pick task", "taskId", taskID)
	err = workflow.ExecuteActivity(ctx, "CompletePickTask", taskID).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to complete pick task", "taskId", taskID, "error", err)
	}

	result.PickedItems = pickedItems
	result.Success = len(pickedItems) > 0

	logger.Info("Picking workflow completed",
		"orderId", orderID,
		"taskId", taskID,
		"pickedItems", len(pickedItems),
		"success", result.Success,
	)

	return result, nil
}
