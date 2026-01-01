package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// OrderFulfillmentInput represents the input for the order fulfillment workflow
type OrderFulfillmentInput struct {
	OrderID            string              `json:"orderId"`
	CustomerID         string              `json:"customerId"`
	Items              []Item              `json:"items"`
	Priority           string              `json:"priority"`
	PromisedDeliveryAt time.Time           `json:"promisedDeliveryAt"`
	IsMultiItem        bool                `json:"isMultiItem"`
	// Process path fields
	GiftWrap         bool                  `json:"giftWrap"`
	GiftWrapDetails  *GiftWrapDetailsInput `json:"giftWrapDetails,omitempty"`
	HazmatDetails    *HazmatDetailsInput   `json:"hazmatDetails,omitempty"`
	ColdChainDetails *ColdChainDetailsInput `json:"coldChainDetails,omitempty"`
	TotalValue       float64               `json:"totalValue"`
}

// Item represents an order item
type Item struct {
	SKU               string  `json:"sku"`
	Quantity          int     `json:"quantity"`
	Weight            float64 `json:"weight"`
	IsFragile         bool    `json:"isFragile"`
	IsHazmat          bool    `json:"isHazmat"`
	RequiresColdChain bool    `json:"requiresColdChain"`
}

// GiftWrapDetailsInput contains gift wrap configuration
type GiftWrapDetailsInput struct {
	WrapType    string `json:"wrapType"`
	GiftMessage string `json:"giftMessage"`
	HidePrice   bool   `json:"hidePrice"`
}

// HazmatDetailsInput contains hazmat details
type HazmatDetailsInput struct {
	Class              string `json:"class"`
	UNNumber           string `json:"unNumber"`
	PackingGroup       string `json:"packingGroup"`
	ProperShippingName string `json:"properShippingName"`
	LimitedQuantity    bool   `json:"limitedQuantity"`
}

// ColdChainDetailsInput contains cold chain requirements
type ColdChainDetailsInput struct {
	MinTempCelsius  float64 `json:"minTempCelsius"`
	MaxTempCelsius  float64 `json:"maxTempCelsius"`
	RequiresDryIce  bool    `json:"requiresDryIce"`
	RequiresGelPack bool    `json:"requiresGelPack"`
}

// ProcessPathResult represents the determined process path
type ProcessPathResult struct {
	PathID                string   `json:"pathId"`
	Requirements          []string `json:"requirements"`
	ConsolidationRequired bool     `json:"consolidationRequired"`
	GiftWrapRequired      bool     `json:"giftWrapRequired"`
	SpecialHandling       []string `json:"specialHandling"`
	TargetStation         string   `json:"targetStation,omitempty"`
}

// OrderFulfillmentResult represents the result of the order fulfillment workflow
type OrderFulfillmentResult struct {
	OrderID        string `json:"orderId"`
	Status         string `json:"status"`
	TrackingNumber string `json:"trackingNumber,omitempty"`
	WaveID         string `json:"waveId,omitempty"`
	Error          string `json:"error,omitempty"`
}

// WaveAssignment represents a wave assignment signal
type WaveAssignment struct {
	WaveID         string    `json:"waveId"`
	ScheduledStart time.Time `json:"scheduledStart"`
}

// PickResult represents the result of the picking workflow
type PickResult struct {
	TaskID        string       `json:"taskId"`
	PickedItems   []PickedItem `json:"pickedItems"`
	AllocationIDs []string     `json:"allocationIds,omitempty"` // Hard allocation IDs from staging
	Success       bool         `json:"success"`
}

// PickedItem represents a picked item
type PickedItem struct {
	SKU        string `json:"sku"`
	Quantity   int    `json:"quantity"`
	LocationID string `json:"locationId"`
	ToteID     string `json:"toteId"`
}

// PackResult represents the result of the packing workflow
type PackResult struct {
	PackageID      string `json:"packageId"`
	TrackingNumber string `json:"trackingNumber"`
	Carrier        string `json:"carrier"`
	Weight         float64 `json:"weight"`
}

// RouteResult represents the result of route calculation
type RouteResult struct {
	RouteID           string     `json:"routeId"`
	Stops             []RouteStop `json:"stops"`
	EstimatedDistance float64    `json:"estimatedDistance"`
	Strategy          string     `json:"strategy"`
}

// RouteStop represents a stop in a pick route
type RouteStop struct {
	LocationID string `json:"locationId"`
	SKU        string `json:"sku"`
	Quantity   int    `json:"quantity"`
}

// OrderFulfillmentWorkflow is the main saga that orchestrates the entire order fulfillment process
// This workflow coordinates across all bounded contexts: Order -> Waving -> Routing -> Picking -> Consolidation -> Packing -> Shipping
func OrderFulfillmentWorkflow(ctx workflow.Context, input OrderFulfillmentInput) (*OrderFulfillmentResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting order fulfillment workflow", "orderId", input.OrderID)

	result := &OrderFulfillmentResult{
		OrderID: input.OrderID,
		Status:  "in_progress",
	}

	// Activity options with retry policy
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: DefaultActivityTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    DefaultRetryInitialInterval,
			BackoffCoefficient: DefaultRetryBackoffCoefficient,
			MaximumInterval:    DefaultRetryMaxInterval,
			MaximumAttempts:    DefaultMaxRetryAttempts,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Child workflow options
	childOpts := workflow.ChildWorkflowOptions{
		WorkflowExecutionTimeout: DefaultChildWorkflowTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: DefaultMaxRetryAttempts,
		},
	}

	// ========================================
	// Step 1: Validate Order
	// ========================================
	logger.Info("Step 1: Validating order", "orderId", input.OrderID)

	var orderValidated bool
	err := workflow.ExecuteActivity(ctx, "ValidateOrder", input).Get(ctx, &orderValidated)
	if err != nil {
		result.Status = "validation_failed"
		result.Error = fmt.Sprintf("order validation failed: %v", err)
		return result, err
	}

	// ========================================
	// Step 2: Determine Process Path
	// ========================================
	logger.Info("Step 2: Determining process path", "orderId", input.OrderID)

	// Build process path items
	processPathItems := make([]map[string]interface{}, len(input.Items))
	for i, item := range input.Items {
		processPathItems[i] = map[string]interface{}{
			"sku":               item.SKU,
			"quantity":          item.Quantity,
			"weight":            item.Weight,
			"isFragile":         item.IsFragile,
			"isHazmat":          item.IsHazmat,
			"requiresColdChain": item.RequiresColdChain,
		}
	}

	processPathInput := map[string]interface{}{
		"orderId":    input.OrderID,
		"items":      processPathItems,
		"giftWrap":   input.GiftWrap,
		"totalValue": input.TotalValue,
	}
	if input.GiftWrapDetails != nil {
		processPathInput["giftWrapDetails"] = input.GiftWrapDetails
	}
	if input.HazmatDetails != nil {
		processPathInput["hazmatDetails"] = input.HazmatDetails
	}
	if input.ColdChainDetails != nil {
		processPathInput["coldChainDetails"] = input.ColdChainDetails
	}

	var processPath ProcessPathResult
	err = workflow.ExecuteActivity(ctx, "DetermineProcessPath", processPathInput).Get(ctx, &processPath)
	if err != nil {
		result.Status = "process_path_failed"
		result.Error = fmt.Sprintf("process path determination failed: %v", err)
		return result, err
	}

	logger.Info("Process path determined",
		"orderId", input.OrderID,
		"pathId", processPath.PathID,
		"requirements", processPath.Requirements,
		"consolidationRequired", processPath.ConsolidationRequired,
		"giftWrapRequired", processPath.GiftWrapRequired,
	)

	// ========================================
	// Step 3: Wait for Wave Assignment
	// ========================================
	logger.Info("Step 3: Waiting for wave assignment", "orderId", input.OrderID)

	// Set up signal channel for wave assignment
	var waveAssignment WaveAssignment
	waveSignal := workflow.GetSignalChannel(ctx, "waveAssigned")

	// Wait for wave assignment with timeout based on priority
	waveTimeout := getWaveTimeout(input.Priority)
	waveCtx, cancelWave := workflow.WithCancel(ctx)
	defer cancelWave()

	selector := workflow.NewSelector(waveCtx)

	var waveAssigned bool
	selector.AddReceive(waveSignal, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(waveCtx, &waveAssignment)
		waveAssigned = true
	})

	selector.AddFuture(workflow.NewTimer(waveCtx, waveTimeout), func(f workflow.Future) {
		// Timeout - order not assigned to wave in time
		logger.Warn("Wave assignment timeout", "orderId", input.OrderID)
	})

	selector.Select(waveCtx)

	if !waveAssigned {
		result.Status = "wave_timeout"
		result.Error = "order was not assigned to a wave within the expected time"
		return result, fmt.Errorf("wave assignment timeout for order %s", input.OrderID)
	}

	result.WaveID = waveAssignment.WaveID
	logger.Info("Order assigned to wave", "orderId", input.OrderID, "waveId", waveAssignment.WaveID)

	// Update order status to wave_assigned
	err = workflow.ExecuteActivity(ctx, "AssignToWave", input.OrderID, waveAssignment.WaveID).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to update order status to wave_assigned", "orderId", input.OrderID, "error", err)
		// Non-fatal: continue with workflow - order status can be reconciled later
	}

	// ========================================
	// Step 4: Calculate Route
	// ========================================
	logger.Info("Step 4: Calculating pick route", "orderId", input.OrderID, "waveId", waveAssignment.WaveID)

	var routeResult RouteResult
	err = workflow.ExecuteActivity(ctx, "CalculateRoute", map[string]interface{}{
		"orderId": input.OrderID,
		"waveId":  waveAssignment.WaveID,
		"items":   input.Items,
	}).Get(ctx, &routeResult)
	if err != nil {
		result.Status = "routing_failed"
		result.Error = fmt.Sprintf("route calculation failed: %v", err)
		return result, err
	}

	// ========================================
	// Step 5: Execute Picking (Child Workflow)
	// ========================================
	logger.Info("Step 5: Starting picking workflow", "orderId", input.OrderID, "routeId", routeResult.RouteID)

	// Update order status to "picking"
	err = workflow.ExecuteActivity(ctx, "StartPicking", input.OrderID).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to update order status to picking", "orderId", input.OrderID, "error", err)
		// Non-fatal: continue with picking workflow
	}

	pickingChildCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID:               fmt.Sprintf("picking-%s", input.OrderID),
		WorkflowExecutionTimeout: childOpts.WorkflowExecutionTimeout,
		RetryPolicy:              childOpts.RetryPolicy,
	})

	var pickResult PickResult
	err = workflow.ExecuteChildWorkflow(pickingChildCtx, "PickingWorkflow", map[string]interface{}{
		"orderId": input.OrderID,
		"waveId":  waveAssignment.WaveID,
		"route":   routeResult,
	}).Get(ctx, &pickResult)
	if err != nil {
		result.Status = "picking_failed"
		result.Error = fmt.Sprintf("picking workflow failed: %v", err)
		// Trigger compensation - release inventory reservations
		_ = workflow.ExecuteActivity(ctx, "ReleaseInventoryReservation", input.OrderID).Get(ctx, nil)
		return result, err
	}

	// ========================================
	// Step 6: Consolidation (based on process path)
	// ========================================
	if processPath.ConsolidationRequired {
		logger.Info("Step 6: Starting consolidation workflow", "orderId", input.OrderID)

		consolidationChildCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID:               fmt.Sprintf("consolidation-%s", input.OrderID),
			WorkflowExecutionTimeout: childOpts.WorkflowExecutionTimeout,
			RetryPolicy:              childOpts.RetryPolicy,
		})

		err = workflow.ExecuteChildWorkflow(consolidationChildCtx, "ConsolidationWorkflow", map[string]interface{}{
			"orderId":     input.OrderID,
			"waveId":      waveAssignment.WaveID,
			"pickedItems": pickResult.PickedItems,
		}).Get(ctx, nil)
		if err != nil {
			result.Status = "consolidation_failed"
			result.Error = fmt.Sprintf("consolidation workflow failed: %v", err)
			return result, err
		}

		// Update order status to "consolidated"
		err = workflow.ExecuteActivity(ctx, "MarkConsolidated", input.OrderID).Get(ctx, nil)
		if err != nil {
			logger.Warn("Failed to update order status to consolidated", "orderId", input.OrderID, "error", err)
			// Non-fatal: continue with workflow
		}
	} else {
		logger.Info("Step 6: Skipping consolidation (single item order)", "orderId", input.OrderID)
	}

	// ========================================
	// Step 7: Gift Wrap (if required by process path)
	// ========================================
	if processPath.GiftWrapRequired {
		logger.Info("Step 7: Starting gift wrap workflow", "orderId", input.OrderID)

		giftWrapChildCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID:               fmt.Sprintf("giftwrap-%s", input.OrderID),
			WorkflowExecutionTimeout: childOpts.WorkflowExecutionTimeout,
			RetryPolicy:              childOpts.RetryPolicy,
		})

		giftWrapInput := map[string]interface{}{
			"orderId": input.OrderID,
			"waveId":  waveAssignment.WaveID,
			"items":   input.Items,
		}
		if input.GiftWrapDetails != nil {
			giftWrapInput["wrapDetails"] = map[string]interface{}{
				"wrapType":    input.GiftWrapDetails.WrapType,
				"giftMessage": input.GiftWrapDetails.GiftMessage,
				"hidePrice":   input.GiftWrapDetails.HidePrice,
			}
		}

		var giftWrapResult map[string]interface{}
		err = workflow.ExecuteChildWorkflow(giftWrapChildCtx, "GiftWrapWorkflow", giftWrapInput).Get(ctx, &giftWrapResult)
		if err != nil {
			result.Status = "giftwrap_failed"
			result.Error = fmt.Sprintf("gift wrap workflow failed: %v", err)
			return result, err
		}
	} else {
		logger.Info("Step 7: Skipping gift wrap (not required)", "orderId", input.OrderID)
	}

	// ========================================
	// Step 8: Find Capable Station for Packing
	// ========================================
	var targetStationID string
	if len(processPath.Requirements) > 0 {
		logger.Info("Step 8: Finding capable packing station", "orderId", input.OrderID, "requirements", processPath.Requirements)

		var capableStation map[string]interface{}
		err = workflow.ExecuteActivity(ctx, "FindCapableStation", map[string]interface{}{
			"requirements": processPath.Requirements,
			"stationType":  "packing",
		}).Get(ctx, &capableStation)
		if err != nil {
			logger.Warn("Failed to find capable station, using default routing", "orderId", input.OrderID, "error", err)
			// Non-fatal: continue with default station routing
		} else if stationID, ok := capableStation["stationId"].(string); ok {
			targetStationID = stationID
			logger.Info("Capable station found", "orderId", input.OrderID, "stationId", targetStationID)
		}
	}

	// ========================================
	// Step 9: Packing (Child Workflow)
	// ========================================
	logger.Info("Step 9: Starting packing workflow", "orderId", input.OrderID, "stationId", targetStationID)

	packingChildCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID:               fmt.Sprintf("packing-%s", input.OrderID),
		WorkflowExecutionTimeout: childOpts.WorkflowExecutionTimeout,
		RetryPolicy:              childOpts.RetryPolicy,
	})

	packingInput := map[string]interface{}{
		"orderId":         input.OrderID,
		"waveId":          waveAssignment.WaveID,
		"requirements":    processPath.Requirements,
		"specialHandling": processPath.SpecialHandling,
	}
	if targetStationID != "" {
		packingInput["stationId"] = targetStationID
	}

	var packResult PackResult
	err = workflow.ExecuteChildWorkflow(packingChildCtx, "PackingWorkflow", packingInput).Get(ctx, &packResult)
	if err != nil {
		result.Status = "packing_failed"
		result.Error = fmt.Sprintf("packing workflow failed: %v", err)
		return result, err
	}

	// Update order status to "packed"
	err = workflow.ExecuteActivity(ctx, "MarkPacked", input.OrderID).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to update order status to packed", "orderId", input.OrderID, "error", err)
		// Non-fatal: continue with shipping workflow
	}

	result.TrackingNumber = packResult.TrackingNumber

	// ========================================
	// Step 10: Shipping/SLAM (Child Workflow)
	// ========================================
	logger.Info("Step 10: Starting shipping workflow", "orderId", input.OrderID, "trackingNumber", packResult.TrackingNumber)

	shippingChildCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID:               fmt.Sprintf("shipping-%s", input.OrderID),
		WorkflowExecutionTimeout: childOpts.WorkflowExecutionTimeout,
		RetryPolicy:              childOpts.RetryPolicy,
	})

	err = workflow.ExecuteChildWorkflow(shippingChildCtx, "ShippingWorkflow", map[string]interface{}{
		"orderId":        input.OrderID,
		"packageId":      packResult.PackageID,
		"trackingNumber": packResult.TrackingNumber,
		"carrier":        packResult.Carrier,
	}).Get(ctx, nil)
	if err != nil {
		result.Status = "shipping_failed"
		result.Error = fmt.Sprintf("shipping workflow failed: %v", err)
		return result, err
	}

	// ========================================
	// Workflow Complete
	// ========================================
	result.Status = "completed"
	logger.Info("Order fulfillment completed successfully",
		"orderId", input.OrderID,
		"waveId", result.WaveID,
		"trackingNumber", result.TrackingNumber,
	)

	return result, nil
}

// getWaveTimeout returns the wave assignment timeout based on order priority
func getWaveTimeout(priority string) time.Duration {
	switch priority {
	case "same_day":
		return WaveTimeoutSameDay
	case "next_day":
		return WaveTimeoutNextDay
	default:
		return WaveTimeoutDefault
	}
}

// OrderCancellationInput holds optional input for order cancellation
type OrderCancellationInput struct {
	OrderID       string                       `json:"orderId"`
	Reason        string                       `json:"reason"`
	AllocationIDs []string                     `json:"allocationIds,omitempty"` // Hard allocations to return to shelf
	PickedItems   []PickedItem                 `json:"pickedItems,omitempty"`   // Items that were picked (for return-to-shelf)
	IsHardAllocated bool                       `json:"isHardAllocated"`         // Whether inventory has been staged
}

// OrderCancellationWorkflow handles order cancellation with compensation
func OrderCancellationWorkflow(ctx workflow.Context, orderID string, reason string) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting order cancellation workflow", "orderId", orderID, "reason", reason)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: DefaultActivityTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: DefaultMaxRetryAttempts,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Cancel the order
	err := workflow.ExecuteActivity(ctx, "CancelOrder", orderID, reason).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	// Step 2: Release inventory - handle both soft reservations and hard allocations
	// First, try to release soft reservations (for orders not yet staged)
	err = workflow.ExecuteActivity(ctx, "ReleaseInventoryReservation", orderID).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to release inventory reservation", "orderId", orderID, "error", err)
		// Continue with other compensations
	}

	// Step 3: Notify customer
	err = workflow.ExecuteActivity(ctx, "NotifyCustomerCancellation", orderID, reason).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to notify customer of cancellation", "orderId", orderID, "error", err)
	}

	logger.Info("Order cancellation completed", "orderId", orderID)
	return nil
}

// OrderCancellationWorkflowWithAllocations handles order cancellation with hard allocation support
// Use this when cancelling orders that have been staged (hard allocated)
func OrderCancellationWorkflowWithAllocations(ctx workflow.Context, input OrderCancellationInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting order cancellation workflow with allocations",
		"orderId", input.OrderID,
		"reason", input.Reason,
		"isHardAllocated", input.IsHardAllocated,
		"allocationCount", len(input.AllocationIDs),
	)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: DefaultActivityTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: DefaultMaxRetryAttempts,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Cancel the order
	err := workflow.ExecuteActivity(ctx, "CancelOrder", input.OrderID, input.Reason).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	// Step 2: Handle inventory based on allocation status
	if input.IsHardAllocated && len(input.AllocationIDs) > 0 {
		// Order has been staged - need to return inventory to shelf
		logger.Info("Returning hard allocated inventory to shelf", "orderId", input.OrderID, "allocationCount", len(input.AllocationIDs))

		// Build return items from allocations
		returnItems := make([]map[string]interface{}, 0, len(input.AllocationIDs))
		for i, allocID := range input.AllocationIDs {
			sku := ""
			if i < len(input.PickedItems) {
				sku = input.PickedItems[i].SKU
			}
			returnItems = append(returnItems, map[string]interface{}{
				"sku":          sku,
				"allocationId": allocID,
			})
		}

		err = workflow.ExecuteActivity(ctx, "ReturnInventoryToShelf", map[string]interface{}{
			"orderId":    input.OrderID,
			"returnedBy": "cancellation-workflow",
			"reason":     input.Reason,
			"items":      returnItems,
		}).Get(ctx, nil)
		if err != nil {
			logger.Warn("Failed to return inventory to shelf",
				"orderId", input.OrderID,
				"error", err,
			)
			// Continue - this is a compensation that can be reconciled manually
		} else {
			logger.Info("Hard allocated inventory returned to shelf successfully",
				"orderId", input.OrderID,
			)
		}
	} else {
		// Order only has soft reservation - release normally
		logger.Info("Releasing soft inventory reservation", "orderId", input.OrderID)
		err = workflow.ExecuteActivity(ctx, "ReleaseInventoryReservation", input.OrderID).Get(ctx, nil)
		if err != nil {
			logger.Warn("Failed to release inventory reservation",
				"orderId", input.OrderID,
				"error", err,
			)
			// Continue with other compensations
		}
	}

	// Step 3: Notify customer
	err = workflow.ExecuteActivity(ctx, "NotifyCustomerCancellation", input.OrderID, input.Reason).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to notify customer of cancellation",
			"orderId", input.OrderID,
			"error", err,
		)
	}

	logger.Info("Order cancellation completed", "orderId", input.OrderID)
	return nil
}
