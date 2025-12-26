package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ShippingWorkflowInput represents input for the shipping workflow
type ShippingWorkflowInput struct {
	OrderID        string `json:"orderId"`
	PackageID      string `json:"packageId"`
	TrackingNumber string `json:"trackingNumber"`
	Carrier        string `json:"carrier"`
}

// ShippingWorkflow coordinates the SLAM (Scan, Label, Apply, Manifest) and shipping process
func ShippingWorkflow(ctx workflow.Context, input map[string]interface{}) error {
	logger := workflow.GetLogger(ctx)

	orderID, _ := input["orderId"].(string)
	packageID, _ := input["packageId"].(string)
	trackingNumber, _ := input["trackingNumber"].(string)
	carrier, _ := input["carrier"].(string)

	logger.Info("Starting shipping workflow (SLAM)",
		"orderId", orderID,
		"packageId", packageID,
		"trackingNumber", trackingNumber,
		"carrier", carrier,
	)

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

	// Step 1: Create shipment record
	logger.Info("Creating shipment record", "orderId", orderID)
	var shipmentID string
	err := workflow.ExecuteActivity(ctx, "CreateShipment", map[string]interface{}{
		"orderId":        orderID,
		"packageId":      packageID,
		"trackingNumber": trackingNumber,
		"carrier":        carrier,
	}).Get(ctx, &shipmentID)
	if err != nil {
		return fmt.Errorf("failed to create shipment: %w", err)
	}

	// Step 2: Scan package (SLAM - Scan)
	logger.Info("Scanning package", "orderId", orderID, "shipmentId", shipmentID)
	err = workflow.ExecuteActivity(ctx, "ScanPackage", map[string]interface{}{
		"shipmentId":     shipmentID,
		"trackingNumber": trackingNumber,
	}).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to scan package: %w", err)
	}

	// Step 3: Verify label (SLAM - Label verification)
	logger.Info("Verifying shipping label", "orderId", orderID, "shipmentId", shipmentID)
	var labelVerified bool
	err = workflow.ExecuteActivity(ctx, "VerifyShippingLabel", map[string]interface{}{
		"shipmentId":     shipmentID,
		"trackingNumber": trackingNumber,
	}).Get(ctx, &labelVerified)
	if err != nil {
		return fmt.Errorf("failed to verify label: %w", err)
	}

	if !labelVerified {
		return fmt.Errorf("label verification failed for shipment %s", shipmentID)
	}

	// Step 4: Place package on outbound dock (SLAM - Apply to manifest)
	logger.Info("Placing package on outbound dock", "orderId", orderID, "shipmentId", shipmentID)
	err = workflow.ExecuteActivity(ctx, "PlaceOnOutboundDock", map[string]interface{}{
		"shipmentId": shipmentID,
		"carrier":    carrier,
	}).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to place on dock: %w", err)
	}

	// Step 5: Add to carrier manifest (SLAM - Manifest)
	logger.Info("Adding to carrier manifest", "orderId", orderID, "shipmentId", shipmentID, "carrier", carrier)
	err = workflow.ExecuteActivity(ctx, "AddToCarrierManifest", map[string]interface{}{
		"shipmentId":     shipmentID,
		"trackingNumber": trackingNumber,
		"carrier":        carrier,
	}).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to add to manifest: %w", err)
	}

	// Step 6: Mark order as shipped
	logger.Info("Marking order as shipped", "orderId", orderID, "shipmentId", shipmentID)
	err = workflow.ExecuteActivity(ctx, "MarkOrderShipped", map[string]interface{}{
		"orderId":        orderID,
		"shipmentId":     shipmentID,
		"trackingNumber": trackingNumber,
	}).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to mark order shipped: %w", err)
	}

	// Step 7: Notify customer with tracking info
	logger.Info("Notifying customer", "orderId", orderID, "trackingNumber", trackingNumber)
	err = workflow.ExecuteActivity(ctx, "NotifyCustomerShipped", map[string]interface{}{
		"orderId":        orderID,
		"trackingNumber": trackingNumber,
		"carrier":        carrier,
	}).Get(ctx, nil)
	if err != nil {
		// Don't fail workflow if notification fails - it's best effort
		logger.Warn("Failed to notify customer", "orderId", orderID, "error", err)
	}

	logger.Info("Shipping workflow completed successfully",
		"orderId", orderID,
		"shipmentId", shipmentID,
		"trackingNumber", trackingNumber,
	)

	return nil
}
