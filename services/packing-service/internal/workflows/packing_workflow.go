package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// PackingWorkflowInput represents the input for the packing workflow
type PackingWorkflowInput struct {
	OrderID string `json:"orderId"`
	// Multi-tenant context
	TenantID    string `json:"tenantId"`
	FacilityID  string `json:"facilityId"`
	WarehouseID string `json:"warehouseId"`
}

// PackingWorkflowResult represents the result of the packing workflow
type PackingWorkflowResult struct {
	PackageID      string  `json:"packageId"`
	TrackingNumber string  `json:"trackingNumber"`
	Carrier        string  `json:"carrier"`
	Weight         float64 `json:"weight"`
	Success        bool    `json:"success"`
	Error          string  `json:"error,omitempty"`
}

// PackingWorkflow orchestrates the packing process for an order
func PackingWorkflow(ctx workflow.Context, input map[string]interface{}) (*PackingWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)

	// Extract input
	orderID, _ := input["orderId"].(string)

	// Extract tenant context
	tenantID, _ := input["tenantId"].(string)
	facilityID, _ := input["facilityId"].(string)
	warehouseID, _ := input["warehouseId"].(string)

	logger.Info("Starting packing workflow", "orderId", orderID)

	result := &PackingWorkflowResult{
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

	// Step 1: Create pack task
	logger.Info("Creating pack task", "orderId", orderID)
	var taskID string
	err := workflow.ExecuteActivity(ctx, "CreatePackTask", orderID).Get(ctx, &taskID)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create pack task: %v", err)
		return result, err
	}

	// Step 2: Wait for packer assignment
	logger.Info("Waiting for packer assignment", "taskId", taskID)
	packerSignal := workflow.GetSignalChannel(ctx, "packerAssigned")

	var packerInfo struct {
		PackerID string `json:"packerId"`
		Station  string `json:"station"`
	}

	packerCtx, cancelPacker := workflow.WithCancel(ctx)
	defer cancelPacker()

	selector := workflow.NewSelector(packerCtx)

	var assigned bool
	selector.AddReceive(packerSignal, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(packerCtx, &packerInfo)
		assigned = true
	})

	// Timeout for packer assignment - 20 minutes
	selector.AddFuture(workflow.NewTimer(packerCtx, 20*time.Minute), func(f workflow.Future) {
		logger.Warn("Packer assignment timeout", "taskId", taskID)
	})

	selector.Select(packerCtx)

	if !assigned {
		result.Error = "packer assignment timeout"
		return result, fmt.Errorf("packer assignment timeout for task %s", taskID)
	}

	logger.Info("Packer assigned", "taskId", taskID, "packerId", packerInfo.PackerID)

	// Step 3: Assign packer to task
	err = workflow.ExecuteActivity(ctx, "AssignPacker", map[string]string{
		"taskId":   taskID,
		"packerId": packerInfo.PackerID,
		"station":  packerInfo.Station,
	}).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to assign packer", "taskId", taskID, "error", err)
	}

	// Step 4: Process packing with signal-based updates
	logger.Info("Processing packing", "taskId", taskID)

	verifySignal := workflow.GetSignalChannel(ctx, "itemVerified")
	sealSignal := workflow.GetSignalChannel(ctx, "packageSealed")
	labelSignal := workflow.GetSignalChannel(ctx, "labelApplied")
	completeSignal := workflow.GetSignalChannel(ctx, "packingComplete")

	packingComplete := false
	var packageInfo struct {
		PackageID      string  `json:"packageId"`
		TrackingNumber string  `json:"trackingNumber"`
		Carrier        string  `json:"carrier"`
		Weight         float64 `json:"weight"`
	}

	for !packingComplete {
		packingCtx, cancelPacking := workflow.WithCancel(ctx)

		packingSelector := workflow.NewSelector(packingCtx)

		// Handle item verified signal
		packingSelector.AddReceive(verifySignal, func(c workflow.ReceiveChannel, more bool) {
			var item struct {
				SKU      string `json:"sku"`
				Verified bool   `json:"verified"`
			}
			c.Receive(packingCtx, &item)
			logger.Info("Item verified", "taskId", taskID, "sku", item.SKU)
		})

		// Handle package sealed signal
		packingSelector.AddReceive(sealSignal, func(c workflow.ReceiveChannel, more bool) {
			var sealed struct {
				PackageID string  `json:"packageId"`
				Weight    float64 `json:"weight"`
			}
			c.Receive(packingCtx, &sealed)
			packageInfo.PackageID = sealed.PackageID
			packageInfo.Weight = sealed.Weight
			logger.Info("Package sealed", "taskId", taskID, "packageId", sealed.PackageID)
		})

		// Handle label applied signal
		packingSelector.AddReceive(labelSignal, func(c workflow.ReceiveChannel, more bool) {
			var label struct {
				TrackingNumber string `json:"trackingNumber"`
				Carrier        string `json:"carrier"`
			}
			c.Receive(packingCtx, &label)
			packageInfo.TrackingNumber = label.TrackingNumber
			packageInfo.Carrier = label.Carrier
			logger.Info("Label applied", "taskId", taskID, "tracking", label.TrackingNumber)
		})

		// Handle packing complete signal
		packingSelector.AddReceive(completeSignal, func(c workflow.ReceiveChannel, more bool) {
			var complete struct {
				Success bool `json:"success"`
			}
			c.Receive(packingCtx, &complete)
			packingComplete = true
			logger.Info("Packing completed", "taskId", taskID)
		})

		// Activity timeout - 1 hour for entire packing
		packingSelector.AddFuture(workflow.NewTimer(packingCtx, time.Hour), func(f workflow.Future) {
			packingComplete = true
			result.Error = "packing timeout"
			logger.Warn("Packing timeout", "taskId", taskID)
		})

		packingSelector.Select(packingCtx)
		cancelPacking()
	}

	// Step 5: Complete the pack task
	logger.Info("Completing pack task", "taskId", taskID)
	err = workflow.ExecuteActivity(ctx, "CompletePackTask", taskID).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to complete pack task", "taskId", taskID, "error", err)
	}

	result.PackageID = packageInfo.PackageID
	result.TrackingNumber = packageInfo.TrackingNumber
	result.Carrier = packageInfo.Carrier
	result.Weight = packageInfo.Weight
	result.Success = packageInfo.TrackingNumber != ""

	logger.Info("Packing workflow completed",
		"orderId", orderID,
		"taskId", taskID,
		"packageId", result.PackageID,
		"tracking", result.TrackingNumber,
		"success", result.Success,
	)

	return result, nil
}
