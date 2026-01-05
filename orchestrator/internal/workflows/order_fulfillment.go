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

// WESExecutionInput represents the input for the WES execution workflow
type WESExecutionInput struct {
	OrderID         string           `json:"orderId"`
	WaveID          string           `json:"waveId"`
	Items           []WESItemInfo    `json:"items"`
	MultiZone       bool             `json:"multiZone"`
	ProcessPathID   string           `json:"processPathId,omitempty"`
	SpecialHandling []string         `json:"specialHandling,omitempty"`
}

// WESItemInfo represents item information for WES
type WESItemInfo struct {
	SKU        string `json:"sku"`
	Quantity   int    `json:"quantity"`
	LocationID string `json:"locationId,omitempty"`
	Zone       string `json:"zone,omitempty"`
}

// WESExecutionResult represents the result of the WES execution workflow
type WESExecutionResult struct {
	RouteID         string          `json:"routeId"`
	OrderID         string          `json:"orderId"`
	Status          string          `json:"status"`
	PathType        string          `json:"pathType"`
	StagesCompleted int             `json:"stagesCompleted"`
	TotalStages     int             `json:"totalStages"`
	PickResult      *WESStageResult `json:"pickResult,omitempty"`
	WallingResult   *WESStageResult `json:"wallingResult,omitempty"`
	PackingResult   *WESStageResult `json:"packingResult,omitempty"`
	CompletedAt     int64           `json:"completedAt,omitempty"`
	Error           string          `json:"error,omitempty"`
}

// WESStageResult represents the result of a stage in WES
type WESStageResult struct {
	StageType   string `json:"stageType"`
	TaskID      string `json:"taskId"`
	WorkerID    string `json:"workerId"`
	Success     bool   `json:"success"`
	CompletedAt int64  `json:"completedAt,omitempty"`
	Error       string `json:"error,omitempty"`
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

// OrderFulfillmentQueryStatus represents the current status of the order fulfillment workflow
type OrderFulfillmentQueryStatus struct {
	OrderID          string  `json:"orderId"`
	CurrentStage     string  `json:"currentStage"`
	Status           string  `json:"status"`
	CompletionPercent int    `json:"completionPercent"`
	TotalStages      int     `json:"totalStages"`
	CompletedStages  int     `json:"completedStages"`
	Error            string  `json:"error,omitempty"`
}

// OrderFulfillmentWorkflow is the main saga that orchestrates the entire order fulfillment process
// This workflow coordinates across all bounded contexts: Order -> Waving -> Routing -> Picking -> Consolidation -> Packing -> Shipping
func OrderFulfillmentWorkflow(ctx workflow.Context, input OrderFulfillmentInput) (*OrderFulfillmentResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting order fulfillment workflow", "orderId", input.OrderID)

	// Workflow versioning for safe deployments - establishes version tracking
	// Future breaking changes should increment OrderFulfillmentWorkflowVersion and add version checks
	version := workflow.GetVersion(ctx, "OrderFulfillmentWorkflow", workflow.DefaultVersion, OrderFulfillmentWorkflowVersion)
	logger.Info("Workflow version", "version", version)

	result := &OrderFulfillmentResult{
		OrderID: input.OrderID,
		Status:  "in_progress",
	}

	// Query handler for workflow status - allows external systems to inspect current state
	queryStatus := OrderFulfillmentQueryStatus{
		OrderID:         input.OrderID,
		CurrentStage:    "validation",
		Status:          "in_progress",
		TotalStages:     5, // validation, planning, picking, consolidation/packing, shipping
		CompletedStages: 0,
	}
	err := workflow.SetQueryHandler(ctx, "getStatus", func() (OrderFulfillmentQueryStatus, error) {
		return queryStatus, nil
	})
	if err != nil {
		logger.Error("Failed to set query handler", "error", err)
	}

	// Activity options with retry policy
	// ScheduleToCloseTimeout: Total time including all retries
	// StartToCloseTimeout: Time for a single attempt
	// HeartbeatTimeout: Detect stuck/crashed workers for long-running activities
	ao := workflow.ActivityOptions{
		ScheduleToCloseTimeout: 30 * time.Minute, // Total time including retries
		StartToCloseTimeout:    DefaultActivityTimeout,
		HeartbeatTimeout:       30 * time.Second, // Detect stuck workers quickly
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    DefaultRetryInitialInterval,
			BackoffCoefficient: DefaultRetryBackoffCoefficient,
			MaximumInterval:    DefaultRetryMaxInterval,
			MaximumAttempts:    DefaultMaxRetryAttempts,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Child workflow options
	// Note: Retry policies are intentionally omitted for child workflows
	// Workflows should be deterministic; retrying them would repeat the same logic
	// Instead, let activities within child workflows handle their own retries
	childOpts := workflow.ChildWorkflowOptions{
		WorkflowExecutionTimeout: DefaultChildWorkflowTimeout,
	}

	// ========================================
	// Step 1: Validate Order
	// ========================================
	queryStatus.CurrentStage = "validation"
	queryStatus.CompletedStages = 0
	queryStatus.CompletionPercent = 0
	logger.Info("Step 1: Validating order", "orderId", input.OrderID)

	var orderValidated bool
	err = workflow.ExecuteActivity(ctx, "ValidateOrder", input).Get(ctx, &orderValidated)
	if err != nil {
		queryStatus.Status = "failed"
		queryStatus.Error = fmt.Sprintf("validation failed: %v", err)
		result.Status = "validation_failed"
		result.Error = fmt.Sprintf("order validation failed: %v", err)
		return result, err
	}
	queryStatus.CompletedStages = 1
	queryStatus.CompletionPercent = 20

	// ========================================
	// Step 2: Execute Planning Workflow (Child)
	// ========================================
	queryStatus.CurrentStage = "planning"
	queryStatus.CompletionPercent = 20
	logger.Info("Step 2: Executing planning workflow", "orderId", input.OrderID)

	planningChildCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID:               fmt.Sprintf("planning-%s", input.OrderID),
		WorkflowExecutionTimeout: PlanningWorkflowTimeout,
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
		queryStatus.Status = "failed"
		queryStatus.Error = fmt.Sprintf("planning failed: %v", err)
		result.Status = "planning_failed"
		result.Error = fmt.Sprintf("planning workflow failed: %v", err)
		return result, err
	}
	queryStatus.CompletedStages = 2
	queryStatus.CompletionPercent = 40

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
	// Step 3: WES Execution
	// ========================================
	// Delegate picking, walling, and packing to WES (Warehouse Execution System)
	queryStatus.CurrentStage = "wes_execution"
	queryStatus.CompletionPercent = 40
	logger.Info("Step 3: Delegating to WES for execution", "orderId", input.OrderID)

	// Convert items to WES format
	wesItems := make([]WESItemInfo, len(input.Items))
	for i, item := range input.Items {
		wesItems[i] = WESItemInfo{
			SKU:      item.SKU,
			Quantity: item.Quantity,
		}
	}

	// Determine if multi-zone picking is needed
	multiZone := processPath.ConsolidationRequired

	wesInput := WESExecutionInput{
		OrderID:         input.OrderID,
		WaveID:          waveAssignment.WaveID,
		Items:           wesItems,
		MultiZone:       multiZone,
		ProcessPathID:   processPath.PathID,
		SpecialHandling: processPath.SpecialHandling,
	}

	wesChildCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID:               fmt.Sprintf("wes-%s", input.OrderID),
		WorkflowExecutionTimeout: WESExecutionWorkflowTimeout,
		TaskQueue:                WESTaskQueue,
	})

	var wesResult WESExecutionResult
	err = workflow.ExecuteChildWorkflow(wesChildCtx, "WESExecutionWorkflow", wesInput).Get(ctx, &wesResult)
	if err != nil {
		result.Status = "wes_execution_failed"
		result.Error = fmt.Sprintf("WES execution failed: %v", err)
		// Release inventory on failure
		_ = workflow.ExecuteActivity(ctx, "ReleaseInventoryReservation", input.OrderID).Get(ctx, nil)
		return result, err
	}

	logger.Info("WES execution completed",
		"orderId", input.OrderID,
		"routeId", wesResult.RouteID,
		"pathType", wesResult.PathType,
		"stagesCompleted", wesResult.StagesCompleted,
	)

	// Extract pack result from WES for downstream steps
	var packResult PackResult
	if wesResult.PackingResult != nil {
		packResult.PackageID = wesResult.PackingResult.TaskID
	}

	// Update order status to "packed" after WES completes
	err = workflow.ExecuteActivity(ctx, "MarkPacked", input.OrderID).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to update order status to packed", "orderId", input.OrderID, "error", err)
	}

	// ========================================
	// Step 4: SLAM Process (Scan, Label, Apply, Manifest)
	// ========================================
	logger.Info("Step 4: Starting SLAM process", "orderId", input.OrderID, "packageId", packResult.PackageID)

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
	// Step 5: Sortation (Route to Destination Chute)
	// ========================================
	logger.Info("Step 5: Starting sortation workflow", "orderId", input.OrderID, "packageId", packResult.PackageID)

	sortationChildCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID:               fmt.Sprintf("sortation-%s", input.OrderID),
		WorkflowExecutionTimeout: childOpts.WorkflowExecutionTimeout,
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
	// Step 6: Shipping (Carrier Handoff)
	// ========================================
	logger.Info("Step 6: Starting shipping workflow", "orderId", input.OrderID, "trackingNumber", result.TrackingNumber)

	shippingChildCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID:               fmt.Sprintf("shipping-%s", input.OrderID),
		WorkflowExecutionTimeout: childOpts.WorkflowExecutionTimeout,
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

	// Update final query status
	queryStatus.Status = "completed"
	queryStatus.CurrentStage = "completed"
	queryStatus.CompletedStages = 5
	queryStatus.CompletionPercent = 100

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
		ScheduleToCloseTimeout: 15 * time.Minute,
		StartToCloseTimeout:    DefaultActivityTimeout,
		HeartbeatTimeout:       30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    DefaultRetryInitialInterval,
			BackoffCoefficient: DefaultRetryBackoffCoefficient,
			MaximumInterval:    DefaultRetryMaxInterval,
			MaximumAttempts:    DefaultMaxRetryAttempts,
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
		ScheduleToCloseTimeout: 15 * time.Minute,
		StartToCloseTimeout:    DefaultActivityTimeout,
		HeartbeatTimeout:       30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    DefaultRetryInitialInterval,
			BackoffCoefficient: DefaultRetryBackoffCoefficient,
			MaximumInterval:    DefaultRetryMaxInterval,
			MaximumAttempts:    DefaultMaxRetryAttempts,
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
