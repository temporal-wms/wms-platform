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
	// Unit-level tracking fields
	UnitIDs []string `json:"unitIds,omitempty"` // Specific units being shipped
	PathID  string   `json:"pathId,omitempty"`  // Process path ID for consistency
	// Multi-tenant context
	TenantID    string `json:"tenantId"`
	FacilityID  string `json:"facilityId"`
	WarehouseID string `json:"warehouseId"`
}

// ShippingWorkflow coordinates the SLAM (Scan, Label, Apply, Manifest) and shipping process
func ShippingWorkflow(ctx workflow.Context, input map[string]interface{}) error {
	logger := workflow.GetLogger(ctx)

	orderID, _ := input["orderId"].(string)
	packageID, _ := input["packageId"].(string)
	trackingNumber, _ := input["trackingNumber"].(string)
	carrier, _ := input["carrier"].(string)

	// Extract tenant context
	tenantID, _ := input["tenantId"].(string)
	facilityID, _ := input["facilityId"].(string)
	warehouseID, _ := input["warehouseId"].(string)

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

	// Extract unit-level tracking fields (now always enabled)
	var unitIDs []string
	var pathID string
	if ids, ok := input["unitIds"].([]interface{}); ok {
		for _, id := range ids {
			if strID, ok := id.(string); ok {
				unitIDs = append(unitIDs, strID)
			}
		}
	}
	if pid, ok := input["pathId"].(string); ok {
		pathID = pid
	}

	logger.Info("Starting shipping workflow (SLAM)",
		"orderId", orderID,
		"packageId", packageID,
		"trackingNumber", trackingNumber,
		"carrier", carrier,
		"unitCount", len(unitIDs),
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

	// Step 7: Ship inventory (finalize hard allocation - remove from inventory system)
	// Extract allocation info from input if available
	if allocationIDs, ok := input["allocationIds"].([]interface{}); ok && len(allocationIDs) > 0 {
		logger.Info("Finalizing inventory shipment (removing from system)", "orderId", orderID, "allocationCount", len(allocationIDs))

		// Build ship items from allocations and picked items
		shipItems := make([]map[string]interface{}, 0)
		pickedItems, _ := input["pickedItems"].([]interface{})

		for i, allocID := range allocationIDs {
			if strAllocID, ok := allocID.(string); ok {
				sku := ""
				// Try to get SKU from picked items
				if i < len(pickedItems) {
					if itemMap, ok := pickedItems[i].(map[string]interface{}); ok {
						if skuVal, ok := itemMap["sku"].(string); ok {
							sku = skuVal
						}
					}
				}
				shipItems = append(shipItems, map[string]interface{}{
					"sku":          sku,
					"allocationId": strAllocID,
				})
			}
		}

		if len(shipItems) > 0 {
			err = workflow.ExecuteActivity(ctx, "ShipInventory", map[string]interface{}{
				"orderId": orderID,
				"items":   shipItems,
			}).Get(ctx, nil)
			if err != nil {
				// Log but don't fail - inventory removal can be reconciled
				logger.Warn("Failed to finalize inventory shipment, continuing workflow",
					"orderId", orderID,
					"error", err,
				)
			} else {
				logger.Info("Inventory shipped successfully (removed from system)", "orderId", orderID)
			}
		}
	}

	// Step 8: Notify customer with tracking info
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

	// Step 9: Unit-level shipping confirmation (always enabled)
	if len(unitIDs) > 0 {
		logger.Info("Confirming unit-level shipping", "orderId", orderID, "unitCount", len(unitIDs))

		handlerID := "shipping-workflow"

		for _, unitID := range unitIDs {
			err := workflow.ExecuteActivity(ctx, "ConfirmUnitShipped", map[string]interface{}{
				"unitId":         unitID,
				"shipmentId":     shipmentID,
				"trackingNumber": trackingNumber,
				"handlerId":      handlerID,
			}).Get(ctx, nil)

			if err != nil {
				logger.Warn("Failed to confirm unit shipped",
					"orderId", orderID,
					"unitId", unitID,
					"error", err,
				)
				// Continue with other units - partial failure handled at parent workflow
			}
		}

		logger.Info("Unit-level shipping confirmation completed", "orderId", orderID)
	}

	// Suppress unused variable warning
	_ = pathID

	logger.Info("Shipping workflow completed successfully",
		"orderId", orderID,
		"shipmentId", shipmentID,
		"trackingNumber", trackingNumber,
	)

	return nil
}
