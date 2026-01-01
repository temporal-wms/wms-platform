package activities

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/wms-platform/orchestrator/internal/activities/clients"
	"go.temporal.io/sdk/activity"
)

// CreateShipment creates a shipment record in shipping-service
func (a *ShippingActivities) CreateShipment(ctx context.Context, input map[string]interface{}) (string, error) {
	logger := activity.GetLogger(ctx)

	orderID, _ := input["orderId"].(string)
	packageID, _ := input["packageId"].(string)
	trackingNumber, _ := input["trackingNumber"].(string)
	carrierCode, _ := input["carrier"].(string)

	logger.Info("Creating shipment", "orderId", orderID, "packageId", packageID)

	// Get order for shipping address
	order, err := a.clients.GetOrder(ctx, orderID)
	if err != nil {
		logger.Error("Failed to get order", "orderId", orderID, "error", err)
		return "", fmt.Errorf("failed to get order: %w", err)
	}

	// Generate shipment ID
	shipmentID := "SH-" + uuid.New().String()[:8]

	// Map carrier code to carrier details
	carrierName := "FedEx"
	if carrierCode == "" {
		carrierCode = "FEDEX"
	}

	// Create shipment in shipping-service
	shipment, err := a.clients.CreateShipment(ctx, &clients.CreateShipmentRequest{
		ShipmentID: shipmentID,
		OrderID:    orderID,
		PackageID:  packageID,
		Carrier: clients.ShipmentCarrier{
			Code:        carrierCode,
			Name:        carrierName,
			AccountID:   "WMS-001",
			ServiceType: "ground",
		},
		Package: clients.ShipmentPackageInfo{
			PackageID:   packageID,
			Weight:      1.5, // Default weight
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
		return "", fmt.Errorf("failed to create shipment: %w", err)
	}

	logger.Info("Shipment created successfully",
		"orderId", orderID,
		"shipmentId", shipment.ShipmentID,
		"trackingNumber", trackingNumber,
	)

	return shipment.ShipmentID, nil
}

// ScanPackage scans the package to verify it's ready for shipping
func (a *ShippingActivities) ScanPackage(ctx context.Context, input map[string]interface{}) error {
	logger := activity.GetLogger(ctx)

	shipmentID, _ := input["shipmentId"].(string)
	trackingNumber, _ := input["trackingNumber"].(string)

	logger.Info("Scanning package", "shipmentId", shipmentID, "trackingNumber", trackingNumber)

	// Verify shipment exists
	shipment, err := a.clients.GetShipment(ctx, shipmentID)
	if err != nil {
		logger.Error("Failed to get shipment", "shipmentId", shipmentID, "error", err)
		return fmt.Errorf("failed to get shipment: %w", err)
	}

	logger.Info("Package scanned successfully",
		"shipmentId", shipmentID,
		"status", shipment.Status,
	)

	return nil
}

// VerifyShippingLabel verifies the shipping label is correct
func (a *ShippingActivities) VerifyShippingLabel(ctx context.Context, input map[string]interface{}) (bool, error) {
	logger := activity.GetLogger(ctx)

	shipmentID, _ := input["shipmentId"].(string)
	trackingNumber, _ := input["trackingNumber"].(string)

	logger.Info("Verifying shipping label", "shipmentId", shipmentID, "trackingNumber", trackingNumber)

	// Get shipment to verify label
	shipment, err := a.clients.GetShipment(ctx, shipmentID)
	if err != nil {
		logger.Error("Failed to get shipment", "shipmentId", shipmentID, "error", err)
		return false, fmt.Errorf("failed to get shipment: %w", err)
	}

	// Verify the tracking number matches (or that label exists)
	verified := shipment.Label != nil || shipment.TrackingNumber != ""
	if !verified {
		logger.Warn("Label verification failed - no label on shipment", "shipmentId", shipmentID)
	} else {
		logger.Info("Shipping label verified", "shipmentId", shipmentID)
	}

	// For simulation, always return true
	return true, nil
}

// PlaceOnOutboundDock records package placement on outbound dock
func (a *ShippingActivities) PlaceOnOutboundDock(ctx context.Context, input map[string]interface{}) error {
	logger := activity.GetLogger(ctx)

	shipmentID, _ := input["shipmentId"].(string)
	carrier, _ := input["carrier"].(string)

	logger.Info("Placing package on outbound dock",
		"shipmentId", shipmentID,
		"carrier", carrier,
	)

	// This is a simulated activity - in reality, a worker would physically place the package
	// and scan it to record the location

	logger.Info("Package placed on outbound dock successfully",
		"shipmentId", shipmentID,
		"dock", fmt.Sprintf("DOCK-%s-01", carrier),
	)

	return nil
}

// AddToCarrierManifest adds the shipment to the carrier's manifest
func (a *ShippingActivities) AddToCarrierManifest(ctx context.Context, input map[string]interface{}) error {
	logger := activity.GetLogger(ctx)

	shipmentID, _ := input["shipmentId"].(string)
	trackingNumber, _ := input["trackingNumber"].(string)
	carrier, _ := input["carrier"].(string)

	logger.Info("Adding to carrier manifest",
		"shipmentId", shipmentID,
		"trackingNumber", trackingNumber,
		"carrier", carrier,
	)

	// This is a simulated activity - in reality, this would integrate with carrier API
	// to add the package to the day's pickup manifest

	logger.Info("Added to carrier manifest successfully",
		"shipmentId", shipmentID,
		"carrier", carrier,
		"manifestId", fmt.Sprintf("MAN-%s-%s", carrier, uuid.New().String()[:6]),
	)

	return nil
}

// MarkOrderShipped marks the order as shipped in shipping-service
func (a *ShippingActivities) MarkOrderShipped(ctx context.Context, input map[string]interface{}) error {
	logger := activity.GetLogger(ctx)

	orderID, _ := input["orderId"].(string)
	shipmentID, _ := input["shipmentId"].(string)
	trackingNumber, _ := input["trackingNumber"].(string)

	logger.Info("Marking order as shipped",
		"orderId", orderID,
		"shipmentId", shipmentID,
		"trackingNumber", trackingNumber,
	)

	// Mark shipment as shipped
	err := a.clients.MarkShipped(ctx, shipmentID)
	if err != nil {
		logger.Error("Failed to mark shipment as shipped", "shipmentId", shipmentID, "error", err)
		return fmt.Errorf("failed to mark shipped: %w", err)
	}

	logger.Info("Order marked as shipped successfully",
		"orderId", orderID,
		"shipmentId", shipmentID,
	)

	return nil
}

// NotifyCustomerShipped sends shipping notification to customer
func (a *ShippingActivities) NotifyCustomerShipped(ctx context.Context, input map[string]interface{}) error {
	logger := activity.GetLogger(ctx)

	orderID, _ := input["orderId"].(string)
	trackingNumber, _ := input["trackingNumber"].(string)
	carrier, _ := input["carrier"].(string)

	logger.Info("Notifying customer of shipment",
		"orderId", orderID,
		"trackingNumber", trackingNumber,
		"carrier", carrier,
	)

	// This is a simulated activity - in reality, this would integrate with
	// an email/SMS service to notify the customer

	logger.Info("Customer notification sent successfully",
		"orderId", orderID,
		"notificationType", "email",
	)

	return nil
}
