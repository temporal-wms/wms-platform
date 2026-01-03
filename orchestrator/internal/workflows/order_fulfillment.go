package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// OrderFulfillmentInput represents the input for the order fulfillment workflow
type OrderFulfillmentInput struct {
	OrderID            string    `json:"orderId"`
	CustomerID         string    `json:"customerId"`
	Items              []Item    `json:"items"`
	Priority           string    `json:"priority"`
	PromisedDeliveryAt time.Time `json:"promisedDeliveryAt"`
	IsMultiItem        bool      `json:"isMultiItem"`
	// Process path fields
	GiftWrap         bool                   `json:"giftWrap"`
	GiftWrapDetails  *GiftWrapDetailsInput  `json:"giftWrapDetails,omitempty"`
	HazmatDetails    *HazmatDetailsInput    `json:"hazmatDetails,omitempty"`
	ColdChainDetails *ColdChainDetailsInput `json:"coldChainDetails,omitempty"`
	TotalValue       float64                `json:"totalValue"`
	// Unit-level tracking fields
	UnitIDs         []string `json:"unitIds,omitempty"`         // Pre-reserved unit IDs if any
	UseUnitTracking bool     `json:"useUnitTracking,omitempty"` // Feature flag for unit-level tracking
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
	// Unit-level tracking results
	PathID         string   `json:"pathId,omitempty"`         // Persisted process path ID
	CompletedUnits []string `json:"completedUnits,omitempty"` // Units successfully processed
	FailedUnits    []string `json:"failedUnits,omitempty"`    // Units that failed processing
	ExceptionIDs   []string `json:"exceptionIds,omitempty"`   // Exception IDs for failed units
	PartialSuccess bool     `json:"partialSuccess,omitempty"` // True if some units succeeded but not all
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
	// Unit-level tracking fields
	PickedUnitIDs []string `json:"pickedUnitIds,omitempty"` // Units successfully picked
	FailedUnitIDs []string `json:"failedUnitIds,omitempty"` // Units that failed picking
	ExceptionIDs  []string `json:"exceptionIds,omitempty"`  // Exception IDs for failed units
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
	PackageID      string  `json:"packageId"`
	TrackingNumber string  `json:"trackingNumber"`
	Carrier        string  `json:"carrier"`
	Weight         float64 `json:"weight"`
}

// SLAMResult represents the result of the SLAM process
type SLAMResult struct {
	TaskID                string  `json:"taskId"`
	TrackingNumber        string  `json:"trackingNumber"`
	ManifestID            string  `json:"manifestId"`
	ActualWeight          float64 `json:"actualWeight"`
	WeightVariancePercent float64 `json:"weightVariancePercent"`
	BarcodeVerified       bool    `json:"barcodeVerified"`
	LabelApplied          bool    `json:"labelApplied"`
	Success               bool    `json:"success"`
	CarrierID             string  `json:"carrierId"`
	Destination           string  `json:"destination"` // Zip code for sortation
}

// SortationStepResult represents the result of sortation in the fulfillment workflow
type SortationStepResult struct {
	BatchID          string `json:"batchId"`
	ChuteID          string `json:"chuteId"`
	ChuteNumber      int    `json:"chuteNumber"`
	Zone             string `json:"zone"`
	DestinationGroup string `json:"destinationGroup"`
	Success          bool   `json:"success"`
}

// RouteResult represents the result of route calculation
type RouteResult struct {
	RouteID           string      `json:"routeId"`
	Stops             []RouteStop `json:"stops"`
	EstimatedDistance float64     `json:"estimatedDistance"`
	Strategy          string      `json:"strategy"`
}

// RouteStop represents a stop in a pick route
type RouteStop struct {
	LocationID string `json:"locationId"`
	SKU        string `json:"sku"`
	Quantity   int    `json:"quantity"`
}

// MultiRouteResult contains the result of multi-route calculation
type MultiRouteResult struct {
	OrderID       string         `json:"orderId"`
	Routes        []RouteResult  `json:"routes"`
	TotalRoutes   int            `json:"totalRoutes"`
	SplitReason   string         `json:"splitReason"`   // none, zone, capacity, both
	ZoneBreakdown map[string]int `json:"zoneBreakdown"` // Zone -> item count
	TotalItems    int            `json:"totalItems"`
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
	// Step 2: Execute Planning Workflow (Child)
	// ========================================
	logger.Info("Step 2: Executing planning workflow", "orderId", input.OrderID)

	planningChildCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID:               fmt.Sprintf("planning-%s", input.OrderID),
		WorkflowExecutionTimeout: PlanningWorkflowTimeout,
		RetryPolicy:              childOpts.RetryPolicy,
	})

	planningInput := PlanningWorkflowInput{
		OrderID:            input.OrderID,
		CustomerID:         input.CustomerID,
		Items:              input.Items,
		Priority:           input.Priority,
		PromisedDeliveryAt: input.PromisedDeliveryAt,
		IsMultiItem:        input.IsMultiItem,
		GiftWrap:           input.GiftWrap,
		GiftWrapDetails:    input.GiftWrapDetails,
		HazmatDetails:      input.HazmatDetails,
		ColdChainDetails:   input.ColdChainDetails,
		TotalValue:         input.TotalValue,
		UseUnitTracking:    input.UseUnitTracking,
		UnitIDs:            input.UnitIDs,
	}

	var planningResult *PlanningWorkflowResult
	err = workflow.ExecuteChildWorkflow(planningChildCtx, PlanningWorkflow, planningInput).Get(ctx, &planningResult)
	if err != nil {
		result.Status = "planning_failed"
		result.Error = fmt.Sprintf("planning workflow failed: %v", err)
		return result, err
	}

	// Extract planning results
	processPath := planningResult.ProcessPath
	result.WaveID = planningResult.WaveID
	result.PathID = planningResult.PathID

	// Track unit IDs for downstream workflows
	var unitIDs []string
	if input.UseUnitTracking {
		unitIDs = planningResult.ReservedUnitIDs
	}

	waveAssignment := WaveAssignment{
		WaveID:         planningResult.WaveID,
		ScheduledStart: planningResult.WaveScheduledStart,
	}

	logger.Info("Planning completed",
		"orderId", input.OrderID,
		"waveId", planningResult.WaveID,
		"pathId", processPath.PathID,
		"unitCount", len(unitIDs),
	)

	// ========================================
	// Step 4: Calculate Route (supports multi-route splitting)
	// ========================================
	logger.Info("Step 4: Calculating pick routes", "orderId", input.OrderID, "waveId", waveAssignment.WaveID)

	var multiRouteResult MultiRouteResult
	err = workflow.ExecuteActivity(ctx, "CalculateMultiRoute", map[string]interface{}{
		"orderId": input.OrderID,
		"waveId":  waveAssignment.WaveID,
		"items":   input.Items,
	}).Get(ctx, &multiRouteResult)
	if err != nil {
		result.Status = "routing_failed"
		result.Error = fmt.Sprintf("route calculation failed: %v", err)
		return result, err
	}

	logger.Info("Routes calculated",
		"orderId", input.OrderID,
		"totalRoutes", multiRouteResult.TotalRoutes,
		"splitReason", multiRouteResult.SplitReason,
	)

	// ========================================
	// Step 5: Execute Picking (supports parallel multi-route picking)
	// ========================================
	logger.Info("Step 5: Starting picking workflow(s)",
		"orderId", input.OrderID,
		"routeCount", multiRouteResult.TotalRoutes,
	)

	// Update order status to "picking"
	err = workflow.ExecuteActivity(ctx, "StartPicking", input.OrderID).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to update order status to picking", "orderId", input.OrderID, "error", err)
		// Non-fatal: continue with picking workflow
	}

	var pickResult PickResult

	if multiRouteResult.TotalRoutes <= 1 {
		// Single route - use existing flow
		routeResult := multiRouteResult.Routes[0]

		pickingChildCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID:               fmt.Sprintf("picking-%s", input.OrderID),
			WorkflowExecutionTimeout: childOpts.WorkflowExecutionTimeout,
			RetryPolicy:              childOpts.RetryPolicy,
		})

		pickingInput := map[string]interface{}{
			"orderId": input.OrderID,
			"waveId":  waveAssignment.WaveID,
			"route":   routeResult,
		}
		// Include unit-level tracking if enabled
		if input.UseUnitTracking && len(unitIDs) > 0 {
			pickingInput["unitIds"] = unitIDs
			pickingInput["pathId"] = result.PathID
		}

		err = workflow.ExecuteChildWorkflow(pickingChildCtx, "PickingWorkflow", pickingInput).Get(ctx, &pickResult)
		if err != nil {
			result.Status = "picking_failed"
			result.Error = fmt.Sprintf("picking workflow failed: %v", err)
			// Trigger compensation - release inventory reservations
			_ = workflow.ExecuteActivity(ctx, "ReleaseInventoryReservation", input.OrderID).Get(ctx, nil)
			return result, err
		}
	} else {
		// Multi-route - execute parallel picking workflows
		logger.Info("Executing parallel picking for multi-route order",
			"orderId", input.OrderID,
			"routeCount", multiRouteResult.TotalRoutes,
		)

		// Create futures for all picking workflows
		pickingFutures := make([]workflow.ChildWorkflowFuture, multiRouteResult.TotalRoutes)
		for i, route := range multiRouteResult.Routes {
			pickingChildCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
				WorkflowID:               fmt.Sprintf("picking-%s-route-%d", input.OrderID, i),
				WorkflowExecutionTimeout: childOpts.WorkflowExecutionTimeout,
				RetryPolicy:              childOpts.RetryPolicy,
			})

			pickingInput := map[string]interface{}{
				"orderId":    input.OrderID,
				"waveId":     waveAssignment.WaveID,
				"route":      route,
				"routeIndex": i,
				"totalRoutes": multiRouteResult.TotalRoutes,
			}
			// Include unit-level tracking if enabled
			if input.UseUnitTracking && len(unitIDs) > 0 {
				pickingInput["unitIds"] = unitIDs
				pickingInput["pathId"] = result.PathID
			}

			pickingFutures[i] = workflow.ExecuteChildWorkflow(pickingChildCtx, "PickingWorkflow", pickingInput)
		}

		// Wait for all picking workflows and aggregate results
		pickResult = PickResult{
			PickedItems:    make([]PickedItem, 0),
			PickedUnitIDs:  make([]string, 0),
			FailedUnitIDs:  make([]string, 0),
			ExceptionIDs:   make([]string, 0),
		}
		allRoutesSucceeded := true
		var failedRoutes []int

		for i, future := range pickingFutures {
			var routePickResult PickResult
			err := future.Get(ctx, &routePickResult)
			if err != nil {
				logger.Warn("Picking failed for route",
					"orderId", input.OrderID,
					"routeIndex", i,
					"error", err,
				)
				failedRoutes = append(failedRoutes, i)
				allRoutesSucceeded = false
				continue
			}

			// Aggregate picked items
			pickResult.PickedItems = append(pickResult.PickedItems, routePickResult.PickedItems...)
			pickResult.PickedUnitIDs = append(pickResult.PickedUnitIDs, routePickResult.PickedUnitIDs...)
			pickResult.FailedUnitIDs = append(pickResult.FailedUnitIDs, routePickResult.FailedUnitIDs...)
			pickResult.ExceptionIDs = append(pickResult.ExceptionIDs, routePickResult.ExceptionIDs...)
		}

		// Handle partial or complete failure
		if !allRoutesSucceeded {
			if len(pickResult.PickedItems) == 0 {
				// All routes failed
				result.Status = "picking_failed"
				result.Error = fmt.Sprintf("all %d picking routes failed", len(failedRoutes))
				_ = workflow.ExecuteActivity(ctx, "ReleaseInventoryReservation", input.OrderID).Get(ctx, nil)
				return result, fmt.Errorf("all picking routes failed for order %s", input.OrderID)
			}
			// Partial success - continue with picked items
			logger.Warn("Partial picking success",
				"orderId", input.OrderID,
				"failedRoutes", failedRoutes,
				"pickedItems", len(pickResult.PickedItems),
			)
			result.PartialSuccess = true
		}

		logger.Info("Parallel picking completed",
			"orderId", input.OrderID,
			"pickedItems", len(pickResult.PickedItems),
			"allRoutesSucceeded", allRoutesSucceeded,
		)
	}

	// Track unit-level picking results
	if input.UseUnitTracking {
		if len(pickResult.PickedUnitIDs) > 0 {
			result.CompletedUnits = append(result.CompletedUnits, pickResult.PickedUnitIDs...)
		}
		if len(pickResult.FailedUnitIDs) > 0 {
			result.FailedUnits = append(result.FailedUnits, pickResult.FailedUnitIDs...)
			result.PartialSuccess = len(pickResult.PickedUnitIDs) > 0
		}
		if len(pickResult.ExceptionIDs) > 0 {
			result.ExceptionIDs = append(result.ExceptionIDs, pickResult.ExceptionIDs...)
		}
		// Update unitIDs to continue with only successfully picked units
		if len(pickResult.PickedUnitIDs) > 0 {
			unitIDs = pickResult.PickedUnitIDs
		}
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

		consolidationInput := map[string]interface{}{
			"orderId":     input.OrderID,
			"waveId":      waveAssignment.WaveID,
			"pickedItems": pickResult.PickedItems,
		}
		// Include unit-level tracking if enabled
		if input.UseUnitTracking && len(unitIDs) > 0 {
			consolidationInput["unitIds"] = unitIDs
			consolidationInput["pathId"] = result.PathID
		}

		err = workflow.ExecuteChildWorkflow(consolidationChildCtx, "ConsolidationWorkflow", consolidationInput).Get(ctx, nil)
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
	// Include unit-level tracking if enabled
	if input.UseUnitTracking && len(unitIDs) > 0 {
		packingInput["unitIds"] = unitIDs
		packingInput["pathId"] = result.PathID
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
	// Step 10: SLAM Process (Scan, Label, Apply, Manifest)
	// ========================================
	logger.Info("Step 10: Starting SLAM process", "orderId", input.OrderID, "packageId", packResult.PackageID)

	var slamResult SLAMResult
	err = workflow.ExecuteActivity(ctx, "ExecuteSLAM", map[string]interface{}{
		"orderId":        input.OrderID,
		"packageId":      packResult.PackageID,
		"expectedWeight": packResult.Weight,
		"carrier":        packResult.Carrier,
	}).Get(ctx, &slamResult)
	if err != nil {
		result.Status = "slam_failed"
		result.Error = fmt.Sprintf("SLAM process failed: %v", err)
		return result, err
	}

	// Check weight verification - if out of tolerance, log warning but continue
	if slamResult.WeightVariancePercent > WeightToleranceThreshold {
		logger.Warn("Weight verification out of tolerance",
			"orderId", input.OrderID,
			"expectedWeight", packResult.Weight,
			"actualWeight", slamResult.ActualWeight,
			"variancePercent", slamResult.WeightVariancePercent,
		)
		// Could trigger investigation workflow here if needed
	}

	// Update tracking number from SLAM if different
	if slamResult.TrackingNumber != "" {
		result.TrackingNumber = slamResult.TrackingNumber
	}

	logger.Info("SLAM process completed",
		"orderId", input.OrderID,
		"trackingNumber", slamResult.TrackingNumber,
		"manifestId", slamResult.ManifestID,
	)

	// ========================================
	// Step 11: Sortation (Route to Destination Chute)
	// ========================================
	logger.Info("Step 11: Starting sortation workflow", "orderId", input.OrderID, "packageId", packResult.PackageID)

	sortationChildCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID:               fmt.Sprintf("sortation-%s", input.OrderID),
		WorkflowExecutionTimeout: childOpts.WorkflowExecutionTimeout,
		RetryPolicy:              childOpts.RetryPolicy,
	})

	// Get destination from SLAM result or use a default
	destination := slamResult.Destination
	if destination == "" {
		destination = "00000" // Default destination if not provided
	}

	sortationInput := SortationWorkflowInput{
		OrderID:        input.OrderID,
		PackageID:      packResult.PackageID,
		TrackingNumber: result.TrackingNumber,
		ManifestID:     slamResult.ManifestID,
		CarrierID:      slamResult.CarrierID,
		Destination:    destination,
		Weight:         slamResult.ActualWeight,
	}

	var sortationResult *SortationWorkflowResult
	err = workflow.ExecuteChildWorkflow(sortationChildCtx, SortationWorkflow, sortationInput).Get(ctx, &sortationResult)
	if err != nil {
		result.Status = "sortation_failed"
		result.Error = fmt.Sprintf("sortation workflow failed: %v", err)
		return result, err
	}

	logger.Info("Sortation completed",
		"orderId", input.OrderID,
		"batchId", sortationResult.BatchID,
		"chuteId", sortationResult.ChuteID,
		"zone", sortationResult.Zone,
	)

	// ========================================
	// Step 12: Shipping (Carrier Handoff)
	// ========================================
	logger.Info("Step 12: Starting shipping workflow", "orderId", input.OrderID, "trackingNumber", result.TrackingNumber)

	shippingChildCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID:               fmt.Sprintf("shipping-%s", input.OrderID),
		WorkflowExecutionTimeout: childOpts.WorkflowExecutionTimeout,
		RetryPolicy:              childOpts.RetryPolicy,
	})

	shippingInput := map[string]interface{}{
		"orderId":        input.OrderID,
		"packageId":      packResult.PackageID,
		"trackingNumber": result.TrackingNumber,
		"carrier":        packResult.Carrier,
		"manifestId":     slamResult.ManifestID,
		"batchId":        sortationResult.BatchID,
		"chuteId":        sortationResult.ChuteID,
	}
	// Include unit-level tracking if enabled
	if input.UseUnitTracking && len(unitIDs) > 0 {
		shippingInput["unitIds"] = unitIDs
		shippingInput["pathId"] = result.PathID
	}

	err = workflow.ExecuteChildWorkflow(shippingChildCtx, "ShippingWorkflow", shippingInput).Get(ctx, nil)
	if err != nil {
		result.Status = "shipping_failed"
		result.Error = fmt.Sprintf("shipping workflow failed: %v", err)
		return result, err
	}

	// ========================================
	// Workflow Complete
	// ========================================
	// Determine final status based on unit tracking
	if input.UseUnitTracking {
		if len(result.FailedUnits) > 0 && len(result.CompletedUnits) > 0 {
			result.Status = "partial_success"
			result.PartialSuccess = true
			logger.Info("Order fulfillment completed with partial success",
				"orderId", input.OrderID,
				"completedUnits", len(result.CompletedUnits),
				"failedUnits", len(result.FailedUnits),
			)
		} else if len(result.FailedUnits) > 0 {
			result.Status = "failed"
		} else {
			result.Status = "completed"
		}
	} else {
		result.Status = "completed"
	}

	logger.Info("Order fulfillment completed",
		"orderId", input.OrderID,
		"status", result.Status,
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
	OrderID         string       `json:"orderId"`
	Reason          string       `json:"reason"`
	AllocationIDs   []string     `json:"allocationIds,omitempty"` // Hard allocations to return to shelf
	PickedItems     []PickedItem `json:"pickedItems,omitempty"`   // Items that were picked (for return-to-shelf)
	IsHardAllocated bool         `json:"isHardAllocated"`         // Whether inventory has been staged
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
