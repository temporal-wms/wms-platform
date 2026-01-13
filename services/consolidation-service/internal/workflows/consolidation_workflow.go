package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ConsolidationWorkflowInput represents the input for the consolidation workflow
type ConsolidationWorkflowInput struct {
	OrderID     string       `json:"orderId"`
	PickedItems []PickedItem `json:"pickedItems"`
	// Multi-tenant context
	TenantID    string `json:"tenantId"`
	FacilityID  string `json:"facilityId"`
	WarehouseID string `json:"warehouseId"`
}

// PickedItem represents a picked item from the picking workflow
type PickedItem struct {
	SKU        string `json:"sku"`
	Quantity   int    `json:"quantity"`
	LocationID string `json:"locationId"`
	ToteID     string `json:"toteId"`
}

// ConsolidationWorkflowResult represents the result of the consolidation workflow
type ConsolidationWorkflowResult struct {
	ConsolidationID   string `json:"consolidationId"`
	DestinationBin    string `json:"destinationBin"`
	TotalConsolidated int    `json:"totalConsolidated"`
	Success           bool   `json:"success"`
	Error             string `json:"error,omitempty"`
}

// ConsolidationWorkflow orchestrates the consolidation process for an order
func ConsolidationWorkflow(ctx workflow.Context, input map[string]interface{}) (*ConsolidationWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)

	// Extract input
	orderID, _ := input["orderId"].(string)
	pickedItemsRaw, _ := input["pickedItems"].([]interface{})

	// Extract tenant context
	tenantID, _ := input["tenantId"].(string)
	facilityID, _ := input["facilityId"].(string)
	warehouseID, _ := input["warehouseId"].(string)

	logger.Info("Starting consolidation workflow", "orderId", orderID, "itemCount", len(pickedItemsRaw))

	result := &ConsolidationWorkflowResult{
		Success: false,
	}

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

	// Activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
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
		"pickedItems": pickedItemsRaw,
	}).Get(ctx, &consolidationID)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create consolidation unit: %v", err)
		return result, err
	}
	result.ConsolidationID = consolidationID

	// Step 2: Wait for station assignment
	logger.Info("Waiting for station assignment", "consolidationId", consolidationID)
	stationSignal := workflow.GetSignalChannel(ctx, "stationAssigned")

	var stationInfo struct {
		Station        string `json:"station"`
		WorkerID       string `json:"workerId"`
		DestinationBin string `json:"destinationBin"`
	}

	stationCtx, cancelStation := workflow.WithCancel(ctx)
	defer cancelStation()

	selector := workflow.NewSelector(stationCtx)

	var assigned bool
	selector.AddReceive(stationSignal, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(stationCtx, &stationInfo)
		assigned = true
	})

	// Timeout for station assignment - 15 minutes
	selector.AddFuture(workflow.NewTimer(stationCtx, 15*time.Minute), func(f workflow.Future) {
		logger.Warn("Station assignment timeout", "consolidationId", consolidationID)
	})

	selector.Select(stationCtx)

	if !assigned {
		result.Error = "station assignment timeout"
		return result, fmt.Errorf("station assignment timeout for consolidation %s", consolidationID)
	}

	result.DestinationBin = stationInfo.DestinationBin
	logger.Info("Station assigned", "consolidationId", consolidationID, "station", stationInfo.Station)

	// Step 3: Assign station to consolidation unit
	err = workflow.ExecuteActivity(ctx, "AssignStation", map[string]interface{}{
		"consolidationId": consolidationID,
		"station":         stationInfo.Station,
		"workerId":        stationInfo.WorkerID,
		"destinationBin":  stationInfo.DestinationBin,
	}).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to assign station", "consolidationId", consolidationID, "error", err)
	}

	// Step 4: Process consolidation with signal-based updates
	logger.Info("Processing consolidation", "consolidationId", consolidationID)

	itemSignal := workflow.GetSignalChannel(ctx, "itemConsolidated")
	completeSignal := workflow.GetSignalChannel(ctx, "consolidationComplete")

	pendingItems := len(pickedItemsRaw)
	totalConsolidated := 0
	consolidationComplete := false

	for !consolidationComplete && pendingItems > 0 {
		consolidationCtx, cancelConsolidation := workflow.WithCancel(ctx)

		consolidationSelector := workflow.NewSelector(consolidationCtx)

		// Handle item consolidated signal
		consolidationSelector.AddReceive(itemSignal, func(c workflow.ReceiveChannel, more bool) {
			var item struct {
				SKU      string `json:"sku"`
				Quantity int    `json:"quantity"`
				ToteID   string `json:"toteId"`
			}
			c.Receive(consolidationCtx, &item)
			pendingItems--
			totalConsolidated += item.Quantity
			logger.Info("Item consolidated", "consolidationId", consolidationID, "sku", item.SKU, "remaining", pendingItems)
		})

		// Handle consolidation complete signal
		consolidationSelector.AddReceive(completeSignal, func(c workflow.ReceiveChannel, more bool) {
			var complete struct {
				Success           bool `json:"success"`
				TotalConsolidated int  `json:"totalConsolidated"`
			}
			c.Receive(consolidationCtx, &complete)
			consolidationComplete = true
			totalConsolidated = complete.TotalConsolidated
			logger.Info("Consolidation completed", "consolidationId", consolidationID, "total", totalConsolidated)
		})

		// Activity timeout - 1 hour for entire consolidation
		consolidationSelector.AddFuture(workflow.NewTimer(consolidationCtx, time.Hour), func(f workflow.Future) {
			consolidationComplete = true
			result.Error = "consolidation timeout"
			logger.Warn("Consolidation timeout", "consolidationId", consolidationID)
		})

		consolidationSelector.Select(consolidationCtx)
		cancelConsolidation()
	}

	// Step 5: Complete the consolidation unit
	logger.Info("Completing consolidation unit", "consolidationId", consolidationID)
	err = workflow.ExecuteActivity(ctx, "CompleteConsolidation", consolidationID).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to complete consolidation", "consolidationId", consolidationID, "error", err)
	}

	result.TotalConsolidated = totalConsolidated
	result.Success = totalConsolidated > 0

	logger.Info("Consolidation workflow completed",
		"orderId", orderID,
		"consolidationId", consolidationID,
		"totalConsolidated", totalConsolidated,
		"success", result.Success,
	)

	return result, nil
}
