package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ShippingWorkflowInput represents the input for the shipping workflow
type ShippingWorkflowInput struct {
	OrderID        string `json:"orderId"`
	PackageID      string `json:"packageId"`
	TrackingNumber string `json:"trackingNumber"`
	Carrier        string `json:"carrier"`
}

// ShippingWorkflowResult represents the result of the shipping workflow
type ShippingWorkflowResult struct {
	ShipmentID     string     `json:"shipmentId"`
	TrackingNumber string     `json:"trackingNumber"`
	ManifestID     string     `json:"manifestId,omitempty"`
	ShippedAt      *time.Time `json:"shippedAt,omitempty"`
	Success        bool       `json:"success"`
	Error          string     `json:"error,omitempty"`
}

// ShippingWorkflow orchestrates the shipping/SLAM process for an order
func ShippingWorkflow(ctx workflow.Context, input map[string]interface{}) (*ShippingWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)

	// Extract input
	orderID, _ := input["orderId"].(string)
	packageID, _ := input["packageId"].(string)
	trackingNumber, _ := input["trackingNumber"].(string)
	carrier, _ := input["carrier"].(string)

	logger.Info("Starting shipping workflow", "orderId", orderID, "packageId", packageID)

	result := &ShippingWorkflowResult{
		TrackingNumber: trackingNumber,
		Success:        false,
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

	// Step 1: Create shipment
	logger.Info("Creating shipment", "orderId", orderID)
	var shipmentID string
	err := workflow.ExecuteActivity(ctx, "CreateShipment", map[string]string{
		"orderId":        orderID,
		"packageId":      packageID,
		"trackingNumber": trackingNumber,
		"carrier":        carrier,
	}).Get(ctx, &shipmentID)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create shipment: %v", err)
		return result, err
	}
	result.ShipmentID = shipmentID

	// Step 2: Generate shipping label (if not already done)
	if trackingNumber == "" {
		logger.Info("Generating shipping label", "shipmentId", shipmentID)
		var labelInfo struct {
			TrackingNumber string `json:"trackingNumber"`
			LabelURL       string `json:"labelUrl"`
		}
		err = workflow.ExecuteActivity(ctx, "GenerateShippingLabel", shipmentID).Get(ctx, &labelInfo)
		if err != nil {
			result.Error = fmt.Sprintf("failed to generate label: %v", err)
			return result, err
		}
		result.TrackingNumber = labelInfo.TrackingNumber
	} else {
		// Apply existing tracking number
		err = workflow.ExecuteActivity(ctx, "ApplyTrackingNumber", map[string]string{
			"shipmentId":     shipmentID,
			"trackingNumber": trackingNumber,
			"carrier":        carrier,
		}).Get(ctx, nil)
		if err != nil {
			logger.Warn("Failed to apply tracking number", "error", err)
		}
	}

	// Step 3: Add to manifest
	logger.Info("Adding to manifest", "shipmentId", shipmentID)
	var manifestID string
	err = workflow.ExecuteActivity(ctx, "AddToManifest", shipmentID).Get(ctx, &manifestID)
	if err != nil {
		logger.Warn("Failed to add to manifest", "shipmentId", shipmentID, "error", err)
		// Continue without manifest
	}
	result.ManifestID = manifestID

	// Step 4: Wait for carrier pickup or manual ship confirmation
	logger.Info("Waiting for ship confirmation", "shipmentId", shipmentID)
	shipSignal := workflow.GetSignalChannel(ctx, "shipConfirmed")
	scanSignal := workflow.GetSignalChannel(ctx, "packageScanned")

	shipCtx, cancelShip := workflow.WithCancel(ctx)
	defer cancelShip()

	selector := workflow.NewSelector(shipCtx)

	var shipped bool
	var shippedTime time.Time

	// Handle ship confirmation signal
	selector.AddReceive(shipSignal, func(c workflow.ReceiveChannel, more bool) {
		var confirmation struct {
			ShippedAt         time.Time  `json:"shippedAt"`
			EstimatedDelivery *time.Time `json:"estimatedDelivery,omitempty"`
		}
		c.Receive(shipCtx, &confirmation)
		shipped = true
		shippedTime = confirmation.ShippedAt
		logger.Info("Ship confirmed via signal", "shipmentId", shipmentID)
	})

	// Handle package scanned signal (auto-confirms ship)
	selector.AddReceive(scanSignal, func(c workflow.ReceiveChannel, more bool) {
		var scan struct {
			Location  string    `json:"location"`
			ScannedAt time.Time `json:"scannedAt"`
		}
		c.Receive(shipCtx, &scan)
		shipped = true
		shippedTime = scan.ScannedAt
		logger.Info("Package scanned, auto-confirming ship", "shipmentId", shipmentID, "location", scan.Location)
	})

	// Auto-confirm after manifest (typical SLAM workflow)
	// In real-world, this would wait for carrier pickup
	selector.AddFuture(workflow.NewTimer(shipCtx, 5*time.Second), func(f workflow.Future) {
		shipped = true
		shippedTime = workflow.Now(ctx)
		logger.Info("Auto-confirming ship after manifest", "shipmentId", shipmentID)
	})

	selector.Select(shipCtx)

	if !shipped {
		result.Error = "ship confirmation timeout"
		return result, fmt.Errorf("ship confirmation timeout for shipment %s", shipmentID)
	}

	// Step 5: Confirm shipment
	logger.Info("Confirming shipment", "shipmentId", shipmentID)
	err = workflow.ExecuteActivity(ctx, "ConfirmShipment", map[string]interface{}{
		"shipmentId": shipmentID,
		"shippedAt":  shippedTime,
	}).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to confirm shipment", "shipmentId", shipmentID, "error", err)
	}

	result.ShippedAt = &shippedTime
	result.Success = true

	logger.Info("Shipping workflow completed",
		"orderId", orderID,
		"shipmentId", shipmentID,
		"tracking", result.TrackingNumber,
		"success", result.Success,
	)

	return result, nil
}
