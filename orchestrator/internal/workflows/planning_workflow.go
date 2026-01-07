package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// PlanningWorkflowInput represents the input for the planning workflow
type PlanningWorkflowInput struct {
	OrderID            string                 `json:"orderId"`
	CustomerID         string                 `json:"customerId"`
	Items              []Item                 `json:"items"`
	Priority           string                 `json:"priority"`
	PromisedDeliveryAt time.Time              `json:"promisedDeliveryAt"`
	IsMultiItem        bool                   `json:"isMultiItem"`
	GiftWrap           bool                   `json:"giftWrap"`
	GiftWrapDetails    *GiftWrapDetailsInput  `json:"giftWrapDetails,omitempty"`
	HazmatDetails      *HazmatDetailsInput    `json:"hazmatDetails,omitempty"`
	ColdChainDetails   *ColdChainDetailsInput `json:"coldChainDetails,omitempty"`
	TotalValue         float64                `json:"totalValue"`
	UnitIDs            []string               `json:"unitIds,omitempty"` // Unit tracking now always enabled
}

// PlanningWorkflowResult represents the consolidated planning output
type PlanningWorkflowResult struct {
	ProcessPath        ProcessPathResult `json:"processPath"`
	PathID             string            `json:"pathId,omitempty"`
	WaveID             string            `json:"waveId"`
	WaveScheduledStart time.Time         `json:"waveScheduledStart"`
	ReservedUnitIDs    []string          `json:"reservedUnitIds,omitempty"`
	Success            bool              `json:"success"`
	Error              string            `json:"error,omitempty"`
}

// PlanningWorkflow coordinates process path determination and wave assignment
// This workflow is executed as a child workflow of OrderFulfillmentWorkflow
func PlanningWorkflow(ctx workflow.Context, input PlanningWorkflowInput) (*PlanningWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting planning workflow", "orderId", input.OrderID)

	result := &PlanningWorkflowResult{
		Success: false,
	}

	// Activity options with retry policy
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: PlanningActivityTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    DefaultRetryInitialInterval,
			BackoffCoefficient: DefaultRetryBackoffCoefficient,
			MaximumInterval:    DefaultRetryMaxInterval,
			MaximumAttempts:    DefaultMaxRetryAttempts,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// ========================================
	// Step 1: Determine Process Path
	// ========================================
	logger.Info("Planning Step 1: Determining process path", "orderId", input.OrderID)

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
	err := workflow.ExecuteActivity(ctx, "DetermineProcessPath", processPathInput).Get(ctx, &processPath)
	if err != nil {
		result.Error = fmt.Sprintf("process path determination failed: %v", err)
		return result, err
	}

	result.ProcessPath = processPath
	logger.Info("Process path determined",
		"orderId", input.OrderID,
		"pathId", processPath.PathID,
		"requirements", processPath.Requirements,
		"consolidationRequired", processPath.ConsolidationRequired,
		"giftWrapRequired", processPath.GiftWrapRequired,
	)

	// ========================================
	// Step 2: Persist Process Path (always enabled)
	// ========================================
	logger.Info("Planning Step 2: Persisting process path", "orderId", input.OrderID)

	var persistPathResult map[string]string
	err = workflow.ExecuteActivity(ctx, "PersistProcessPath", map[string]interface{}{
		"orderId":    input.OrderID,
		"items":      processPathItems,
		"giftWrap":   input.GiftWrap,
		"totalValue": input.TotalValue,
	}).Get(ctx, &persistPathResult)
	if err != nil {
		logger.Warn("Failed to persist process path", "orderId", input.OrderID, "error", err)
		// Non-fatal: generate default path ID for unit tracking
		result.PathID = fmt.Sprintf("path-%s", input.OrderID)
		processPath.PathID = result.PathID
	} else if pathID, ok := persistPathResult["pathId"]; ok {
		result.PathID = pathID
		processPath.PathID = pathID
	} else {
		// PersistProcessPath succeeded but didn't return pathId, generate default
		result.PathID = fmt.Sprintf("path-%s", input.OrderID)
		processPath.PathID = result.PathID
	}

	// ========================================
	// Step 3: Reserve Units (always enabled)
	// ========================================
	logger.Info("Planning Step 3: Reserving units", "orderId", input.OrderID)

	if len(input.UnitIDs) > 0 {
		// Use pre-existing unit IDs (units already created and passed in)
		result.ReservedUnitIDs = input.UnitIDs
		logger.Info("Using pre-existing units", "orderId", input.OrderID, "unitCount", len(input.UnitIDs))
	} else {
		// Reserve units from available inventory (units should already exist from receiving)
		reserveItems := make([]map[string]interface{}, len(input.Items))
		for i, item := range input.Items {
			reserveItems[i] = map[string]interface{}{
				"sku":      item.SKU,
				"quantity": item.Quantity,
			}
		}

		var reserveResult map[string]interface{}
		err = workflow.ExecuteActivity(ctx, "ReserveUnits", map[string]interface{}{
			"orderId":   input.OrderID,
			"pathId":    result.PathID,
			"items":     reserveItems,
			"handlerId": "planning-workflow",
		}).Get(ctx, &reserveResult)
		if err != nil {
			result.Error = fmt.Sprintf("unit reservation failed: %v", err)
			return result, err
		}

		// Extract reserved unit IDs
		if reserved, ok := reserveResult["reservedUnits"].([]interface{}); ok {
			for _, u := range reserved {
				if unit, ok := u.(map[string]interface{}); ok {
					if id, ok := unit["unitId"].(string); ok {
						result.ReservedUnitIDs = append(result.ReservedUnitIDs, id)
					}
				}
			}
		}

		// Check for failed reservations
		if failed, ok := reserveResult["failedItems"].([]interface{}); ok && len(failed) > 0 {
			logger.Warn("Some units could not be reserved", "orderId", input.OrderID, "failedCount", len(failed))
			// Continue with partial reservation - workflow will handle partial completion
		}
	}

	logger.Info("Units reserved for order", "orderId", input.OrderID, "unitCount", len(result.ReservedUnitIDs))

	// ========================================
	// Step 4: Wait for Wave Assignment
	// ========================================
	logger.Info("Planning Step 4: Waiting for wave assignment", "orderId", input.OrderID)

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
		logger.Warn("Wave assignment timeout", "orderId", input.OrderID, "timeout", waveTimeout)
	})

	selector.Select(waveCtx)

	if !waveAssigned {
		result.Error = "wave assignment timeout"
		return result, fmt.Errorf("wave assignment timeout for order %s", input.OrderID)
	}

	result.WaveID = waveAssignment.WaveID
	result.WaveScheduledStart = waveAssignment.ScheduledStart
	logger.Info("Order assigned to wave", "orderId", input.OrderID, "waveId", waveAssignment.WaveID)

	// ========================================
	// Step 5: Update Order Status
	// ========================================
	logger.Info("Planning Step 5: Updating order status to wave_assigned", "orderId", input.OrderID)

	err = workflow.ExecuteActivity(ctx, "AssignToWave", input.OrderID, waveAssignment.WaveID).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to update order status to wave_assigned", "orderId", input.OrderID, "error", err)
		// Non-fatal: continue
	}

	result.Success = true

	logger.Info("Planning workflow completed",
		"orderId", input.OrderID,
		"waveId", result.WaveID,
		"pathId", processPath.PathID,
	)

	return result, nil
}
