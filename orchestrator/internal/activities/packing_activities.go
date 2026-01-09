package activities

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/wms-platform/orchestrator/internal/activities/clients"
	"go.temporal.io/sdk/activity"
)

// CreatePackTask creates a packing task for an order
func (a *PackingActivities) CreatePackTask(ctx context.Context, input map[string]interface{}) (string, error) {
	logger := activity.GetLogger(ctx)

	orderID, _ := input["orderId"].(string)
	waveID, _ := input["waveId"].(string)

	logger.Info("Creating pack task", "orderId", orderID, "waveId", waveID)

	// Get order details to get items
	order, err := a.clients.GetOrder(ctx, orderID)
	if err != nil {
		logger.Error("Failed to get order", "orderId", orderID, "error", err)
		return "", fmt.Errorf("failed to get order: %w", err)
	}

	// Convert order items to pack items
	items := make([]clients.PackItem, len(order.Items))
	for i, item := range order.Items {
		items[i] = clients.PackItem{
			SKU:      item.SKU,
			Quantity: item.Quantity,
		}
	}

	// Generate task ID
	taskID := "PK-" + uuid.New().String()[:8]

	// Call packing-service to create task
	task, err := a.clients.CreatePackTask(ctx, &clients.CreatePackTaskRequest{
		TaskID:  taskID,
		OrderID: orderID,
		WaveID:  waveID,
		Items:   items,
	})
	if err != nil {
		logger.Error("Failed to create pack task", "orderId", orderID, "error", err)
		return "", fmt.Errorf("failed to create pack task: %w", err)
	}

	logger.Info("Pack task created successfully",
		"orderId", orderID,
		"taskId", task.TaskID,
		"itemCount", len(items),
	)

	return task.TaskID, nil
}

// StartPackTask starts a packing task (sets startedAt timestamp)
func (a *PackingActivities) StartPackTask(ctx context.Context, taskID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting pack task", "taskId", taskID)

	_, err := a.clients.StartPackTask(ctx, taskID)
	if err != nil {
		logger.Error("Failed to start pack task", "taskId", taskID, "error", err)
		return fmt.Errorf("failed to start pack task: %w", err)
	}

	logger.Info("Pack task started", "taskId", taskID)
	return nil
}

// CompletePackTask completes a packing task (sets completedAt timestamp)
func (a *PackingActivities) CompletePackTask(ctx context.Context, taskID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Completing pack task", "taskId", taskID)

	_, err := a.clients.CompletePackTask(ctx, taskID)
	if err != nil {
		logger.Error("Failed to complete pack task", "taskId", taskID, "error", err)
		return fmt.Errorf("failed to complete pack task: %w", err)
	}

	logger.Info("Pack task completed", "taskId", taskID)
	return nil
}

// SelectPackagingMaterials selects appropriate packaging for items
func (a *PackingActivities) SelectPackagingMaterials(ctx context.Context, input map[string]interface{}) (string, error) {
	logger := activity.GetLogger(ctx)

	taskID, _ := input["taskId"].(string)
	orderID, _ := input["orderId"].(string)

	logger.Info("Selecting packaging materials", "taskId", taskID, "orderId", orderID)

	// Get order details to determine package size
	order, err := a.clients.GetOrder(ctx, orderID)
	if err != nil {
		logger.Warn("Failed to get order for packaging selection, using default", "orderId", orderID, "error", err)
	}

	// Calculate total weight and determine package type
	var totalWeight float64
	if order != nil {
		for _, item := range order.Items {
			totalWeight += item.Weight * float64(item.Quantity)
		}
	}

	// Select package type based on weight
	var packageType string
	switch {
	case totalWeight <= 0.5:
		packageType = "small_box"
	case totalWeight <= 2.0:
		packageType = "medium_box"
	case totalWeight <= 10.0:
		packageType = "large_box"
	default:
		packageType = "extra_large_box"
	}

	// Generate package ID
	packageID := "PKG-" + uuid.New().String()[:8]

	logger.Info("Packaging materials selected",
		"taskId", taskID,
		"packageId", packageID,
		"packageType", packageType,
		"estimatedWeight", totalWeight,
	)

	return packageID, nil
}

// PackItems packs items into the package
func (a *PackingActivities) PackItems(ctx context.Context, input map[string]interface{}) error {
	logger := activity.GetLogger(ctx)

	taskID, _ := input["taskId"].(string)
	packageID, _ := input["packageId"].(string)

	logger.Info("Packing items", "taskId", taskID, "packageId", packageID)

	// This is a simulated activity - in reality, a worker would physically pack items
	// and the system would track completion via a signal or status update

	logger.Info("Items packed successfully", "taskId", taskID, "packageId", packageID)
	return nil
}

// WeighPackage weighs the packed package
func (a *PackingActivities) WeighPackage(ctx context.Context, packageID string) (float64, error) {
	logger := activity.GetLogger(ctx)

	logger.Info("Weighing package", "packageId", packageID)

	// In a real system, this would integrate with a scale
	// For simulation, return an estimated weight
	weight := 1.5 // kg

	logger.Info("Package weighed", "packageId", packageID, "weight", weight)
	return weight, nil
}

// GenerateShippingLabel creates a shipping label via shipping-service
func (a *PackingActivities) GenerateShippingLabel(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)

	orderID, _ := input["orderId"].(string)
	packageID, _ := input["packageId"].(string)
	weight, _ := input["weight"].(float64)

	logger.Info("Generating shipping label", "orderId", orderID, "packageId", packageID)

	// Get order for shipping address
	order, err := a.clients.GetOrder(ctx, orderID)
	if err != nil {
		logger.Error("Failed to get order for shipping label", "orderId", orderID, "error", err)
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// Generate shipment ID
	shipmentID := "SH-" + uuid.New().String()[:8]

	// Create shipment in shipping-service
	shipment, err := a.clients.CreateShipment(ctx, &clients.CreateShipmentRequest{
		ShipmentID: shipmentID,
		OrderID:    orderID,
		PackageID:  packageID,
		Carrier: clients.ShipmentCarrier{
			Code:        "FEDEX",
			Name:        "FedEx",
			AccountID:   "WMS-001",
			ServiceType: "ground",
		},
		Package: clients.ShipmentPackageInfo{
			PackageID:   packageID,
			Weight:      weight,
			Dimensions: clients.Dimensions{
				Length: 30.0,
				Width:  20.0,
				Height: 15.0,
			},
			PackageType: "box",
		},
		Recipient: clients.ShipmentAddress{
			Name:       order.CustomerID,
			Street1:    order.ShippingAddress.Street,
			City:       order.ShippingAddress.City,
			State:      order.ShippingAddress.State,
			PostalCode: order.ShippingAddress.PostalCode,
			Country:    order.ShippingAddress.Country,
		},
		Shipper: clients.ShipmentAddress{
			Name:       "WMS Platform Warehouse",
			Company:    "WMS Platform",
			Street1:    "100 Warehouse Way",
			City:       "San Francisco",
			State:      "CA",
			PostalCode: "94105",
			Country:    "US",
		},
	})
	if err != nil {
		logger.Error("Failed to create shipment", "orderId", orderID, "error", err)
		return nil, fmt.Errorf("failed to create shipment: %w", err)
	}

	// Generate label
	label, err := a.clients.GenerateLabel(ctx, shipment.ShipmentID)
	if err != nil {
		logger.Error("Failed to generate label", "shipmentId", shipment.ShipmentID, "error", err)
		return nil, fmt.Errorf("failed to generate label: %w", err)
	}

	result := map[string]interface{}{
		"shipmentId":     shipment.ShipmentID,
		"trackingNumber": label.TrackingNumber,
		"carrier":        label.Carrier.Code,
		"labelUrl":       label.LabelURL,
	}

	logger.Info("Shipping label generated",
		"orderId", orderID,
		"shipmentId", shipment.ShipmentID,
		"trackingNumber", label.TrackingNumber,
	)

	return result, nil
}

// ApplyLabelToPackage records that the label has been applied
func (a *PackingActivities) ApplyLabelToPackage(ctx context.Context, input map[string]interface{}) error {
	logger := activity.GetLogger(ctx)

	packageID, _ := input["packageId"].(string)
	trackingNumber, _ := input["trackingNumber"].(string)

	logger.Info("Applying label to package", "packageId", packageID, "trackingNumber", trackingNumber)

	// In a real system, this would be a physical action tracked via scanning
	// For simulation, just log the action

	logger.Info("Label applied successfully", "packageId", packageID, "trackingNumber", trackingNumber)
	return nil
}

// SealPackage seals the package
func (a *PackingActivities) SealPackage(ctx context.Context, packageID string) error {
	logger := activity.GetLogger(ctx)

	logger.Info("Sealing package", "packageId", packageID)

	// In a real system, this would be a physical action
	// For simulation, just log the action

	logger.Info("Package sealed successfully", "packageId", packageID)
	return nil
}
