package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// InboundFulfillmentInput represents the input for the inbound fulfillment workflow
type InboundFulfillmentInput struct {
	ShipmentID      string                 `json:"shipmentId"`
	ASNID           string                 `json:"asnId"`
	SupplierID      string                 `json:"supplierId"`
	ExpectedItems   []InboundExpectedItem  `json:"expectedItems"`
	ExpectedArrival time.Time              `json:"expectedArrival"`
	DockID          string                 `json:"dockId,omitempty"`
	// Multi-tenant context
	TenantID    string `json:"tenantId"`
	FacilityID  string `json:"facilityId"`
	WarehouseID string `json:"warehouseId"`
}

// InboundExpectedItem represents an expected item in the inbound shipment
type InboundExpectedItem struct {
	SKU               string  `json:"sku"`
	ProductName       string  `json:"productName"`
	ExpectedQuantity  int     `json:"expectedQuantity"`
	UnitCost          float64 `json:"unitCost"`
	Weight            float64 `json:"weight"`
	IsHazmat          bool    `json:"isHazmat"`
	RequiresColdChain bool    `json:"requiresColdChain"`
}

// InboundFulfillmentResult represents the result of the inbound fulfillment workflow
type InboundFulfillmentResult struct {
	ShipmentID         string   `json:"shipmentId"`
	Status             string   `json:"status"`
	TotalExpected      int      `json:"totalExpected"`
	TotalReceived      int      `json:"totalReceived"`
	TotalDamaged       int      `json:"totalDamaged"`
	PutawayTaskIDs     []string `json:"putawayTaskIds,omitempty"`
	DiscrepancyCount   int      `json:"discrepancyCount"`
	Error              string   `json:"error,omitempty"`
}

// ReceivingResult represents the result of receiving activities
type ReceivingResult struct {
	ShipmentID       string `json:"shipmentId"`
	TotalReceived    int    `json:"totalReceived"`
	TotalDamaged     int    `json:"totalDamaged"`
	DiscrepancyCount int    `json:"discrepancyCount"`
	Success          bool   `json:"success"`
}

// QualityInspectionResult represents the result of quality inspection
type QualityInspectionResult struct {
	ShipmentID    string `json:"shipmentId"`
	InspectedCount int   `json:"inspectedCount"`
	PassedCount    int   `json:"passedCount"`
	FailedCount    int   `json:"failedCount"`
	Passed         bool  `json:"passed"`
}

// PutawayResult represents the result of putaway task creation
type PutawayResult struct {
	ShipmentID  string   `json:"shipmentId"`
	TaskIDs     []string `json:"taskIds"`
	TaskCount   int      `json:"taskCount"`
	TotalItems  int      `json:"totalItems"`
}

// StowResult represents the result of stow execution
type StowResult struct {
	ShipmentID     string `json:"shipmentId"`
	StowedCount    int    `json:"stowedCount"`
	FailedCount    int    `json:"failedCount"`
	Success        bool   `json:"success"`
}

// InboundFulfillmentWorkflow orchestrates the inbound fulfillment process
// This workflow coordinates: ASN → Receive → Quality Check → Stow
func InboundFulfillmentWorkflow(ctx workflow.Context, input InboundFulfillmentInput) (*InboundFulfillmentResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting inbound fulfillment workflow",
		"shipmentId", input.ShipmentID,
		"asnId", input.ASNID,
	)

	result := &InboundFulfillmentResult{
		ShipmentID: input.ShipmentID,
		Status:     "in_progress",
	}

	// Calculate total expected
	for _, item := range input.ExpectedItems {
		result.TotalExpected += item.ExpectedQuantity
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
	// Step 1: Validate ASN
	// ========================================
	logger.Info("Step 1: Validating ASN", "asnId", input.ASNID)

	var asnValid bool
	err := workflow.ExecuteActivity(ctx, "ValidateASN", map[string]interface{}{
		"shipmentId": input.ShipmentID,
		"asnId":      input.ASNID,
		"supplierId": input.SupplierID,
	}).Get(ctx, &asnValid)
	if err != nil {
		result.Status = "asn_validation_failed"
		result.Error = fmt.Sprintf("ASN validation failed: %v", err)
		return result, err
	}

	// ========================================
	// Step 2: Wait for Shipment Arrival
	// ========================================
	logger.Info("Step 2: Waiting for shipment arrival", "shipmentId", input.ShipmentID)

	// Set up signal channel for arrival notification
	arrivalSignal := workflow.GetSignalChannel(ctx, "shipmentArrived")

	// Wait for arrival with timeout
	arrivalTimeout := input.ExpectedArrival.Add(4 * time.Hour).Sub(workflow.Now(ctx))
	if arrivalTimeout < 0 {
		arrivalTimeout = 4 * time.Hour // Default if already past expected time
	}

	var arrivedDockID string
	arrivalCtx, cancelArrival := workflow.WithCancel(ctx)
	defer cancelArrival()

	selector := workflow.NewSelector(arrivalCtx)

	var arrived bool
	selector.AddReceive(arrivalSignal, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(arrivalCtx, &arrivedDockID)
		arrived = true
	})

	selector.AddFuture(workflow.NewTimer(arrivalCtx, arrivalTimeout), func(f workflow.Future) {
		logger.Warn("Shipment arrival timeout", "shipmentId", input.ShipmentID)
	})

	selector.Select(arrivalCtx)

	if !arrived {
		result.Status = "arrival_timeout"
		result.Error = "shipment did not arrive within expected time"
		return result, fmt.Errorf("arrival timeout for shipment %s", input.ShipmentID)
	}

	// Mark shipment as arrived
	err = workflow.ExecuteActivity(ctx, "MarkShipmentArrived", map[string]interface{}{
		"shipmentId": input.ShipmentID,
		"dockId":     arrivedDockID,
	}).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to mark shipment as arrived", "error", err)
		// Non-fatal, continue
	}

	// ========================================
	// Step 3: Execute Receiving (Child Workflow)
	// ========================================
	logger.Info("Step 3: Starting receiving process", "shipmentId", input.ShipmentID, "dockId", arrivedDockID)

	receivingChildCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID:               fmt.Sprintf("receiving-%s", input.ShipmentID),
		WorkflowExecutionTimeout: childOpts.WorkflowExecutionTimeout,
		RetryPolicy:              childOpts.RetryPolicy,
	})

	var receivingResult ReceivingResult
	err = workflow.ExecuteChildWorkflow(receivingChildCtx, "ReceivingWorkflow", map[string]interface{}{
		"shipmentId":    input.ShipmentID,
		"dockId":        arrivedDockID,
		"expectedItems": input.ExpectedItems,
	}).Get(ctx, &receivingResult)
	if err != nil {
		result.Status = "receiving_failed"
		result.Error = fmt.Sprintf("receiving workflow failed: %v", err)
		return result, err
	}

	result.TotalReceived = receivingResult.TotalReceived
	result.TotalDamaged = receivingResult.TotalDamaged
	result.DiscrepancyCount = receivingResult.DiscrepancyCount

	// ========================================
	// Step 4: Quality Inspection (Sampling)
	// ========================================
	logger.Info("Step 4: Starting quality inspection", "shipmentId", input.ShipmentID)

	var inspectionResult QualityInspectionResult
	err = workflow.ExecuteActivity(ctx, "PerformQualityInspection", map[string]interface{}{
		"shipmentId":     input.ShipmentID,
		"samplingRate":   0.1, // 10% sampling
		"expectedItems":  input.ExpectedItems,
	}).Get(ctx, &inspectionResult)
	if err != nil {
		logger.Warn("Quality inspection failed", "error", err)
		// Non-fatal for sampling inspection, continue
	}

	if !inspectionResult.Passed && inspectionResult.FailedCount > 0 {
		// Log quality issue but continue with stow
		logger.Warn("Quality inspection found issues",
			"shipmentId", input.ShipmentID,
			"failedCount", inspectionResult.FailedCount,
		)
	}

	// ========================================
	// Step 5: Create Putaway Tasks
	// ========================================
	logger.Info("Step 5: Creating putaway tasks", "shipmentId", input.ShipmentID)

	var putawayResult PutawayResult
	err = workflow.ExecuteActivity(ctx, "CreatePutawayTasks", map[string]interface{}{
		"shipmentId":      input.ShipmentID,
		"receivedItems":   input.ExpectedItems,
		"storageStrategy": "chaotic", // Default to chaotic storage
	}).Get(ctx, &putawayResult)
	if err != nil {
		result.Status = "putaway_creation_failed"
		result.Error = fmt.Sprintf("putaway task creation failed: %v", err)
		return result, err
	}

	result.PutawayTaskIDs = putawayResult.TaskIDs

	// ========================================
	// Step 6: Execute Stow (Child Workflow)
	// ========================================
	logger.Info("Step 6: Starting stow workflow",
		"shipmentId", input.ShipmentID,
		"taskCount", putawayResult.TaskCount,
	)

	stowChildCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID:               fmt.Sprintf("stow-%s", input.ShipmentID),
		WorkflowExecutionTimeout: childOpts.WorkflowExecutionTimeout,
		RetryPolicy:              childOpts.RetryPolicy,
	})

	var stowResult StowResult
	err = workflow.ExecuteChildWorkflow(stowChildCtx, "StowWorkflow", map[string]interface{}{
		"shipmentId": input.ShipmentID,
		"taskIds":    putawayResult.TaskIDs,
	}).Get(ctx, &stowResult)
	if err != nil {
		result.Status = "stow_failed"
		result.Error = fmt.Sprintf("stow workflow failed: %v", err)
		return result, err
	}

	// ========================================
	// Step 7: Confirm Inventory Receipt
	// ========================================
	logger.Info("Step 7: Confirming inventory receipt", "shipmentId", input.ShipmentID)

	err = workflow.ExecuteActivity(ctx, "ConfirmInventoryReceipt", map[string]interface{}{
		"shipmentId":    input.ShipmentID,
		"receivedItems": input.ExpectedItems,
		"stowedCount":   stowResult.StowedCount,
	}).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to confirm inventory receipt", "error", err)
		// Non-fatal, continue
	}

	// ========================================
	// Step 8: Complete Receiving
	// ========================================
	logger.Info("Step 8: Completing receiving process", "shipmentId", input.ShipmentID)

	err = workflow.ExecuteActivity(ctx, "CompleteReceiving", input.ShipmentID).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to complete receiving", "error", err)
		// Non-fatal, continue
	}

	// ========================================
	// Workflow Complete
	// ========================================
	result.Status = "completed"
	logger.Info("Inbound fulfillment completed successfully",
		"shipmentId", input.ShipmentID,
		"totalReceived", result.TotalReceived,
		"totalDamaged", result.TotalDamaged,
		"putawayTasks", len(result.PutawayTaskIDs),
	)

	return result, nil
}

// ShipmentArrivalSignal is the signal payload for shipment arrival
type ShipmentArrivalSignal struct {
	ShipmentID string `json:"shipmentId"`
	DockID     string `json:"dockId"`
	ArrivedAt  time.Time `json:"arrivedAt"`
}
