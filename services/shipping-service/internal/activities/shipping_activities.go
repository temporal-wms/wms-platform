package activities

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/wms-platform/shipping-service/internal/domain"
	"go.temporal.io/sdk/activity"
)

// ShippingActivities contains activities for the shipping workflow
type ShippingActivities struct {
	repo   domain.ShipmentRepository
	logger *slog.Logger
}

// NewShippingActivities creates a new ShippingActivities instance
func NewShippingActivities(repo domain.ShipmentRepository, logger *slog.Logger) *ShippingActivities {
	return &ShippingActivities{
		repo:   repo,
		logger: logger,
	}
}

// CreateShipment creates a new shipment
func (a *ShippingActivities) CreateShipment(ctx context.Context, input map[string]string) (string, error) {
	logger := activity.GetLogger(ctx)

	orderID := input["orderId"]
	packageID := input["packageId"]
	carrierCode := input["carrier"]

	logger.Info("Creating shipment", "orderId", orderID, "packageId", packageID)

	// Generate shipment ID
	shipmentID := "SHP-" + uuid.New().String()[:8]

	// Create carrier info
	carrier := domain.Carrier{
		Code:        carrierCode,
		Name:        getCarrierName(carrierCode),
		ServiceType: "ground",
	}

	// Create package info (in real impl, would fetch from packing service)
	pkg := domain.PackageInfo{
		PackageID: packageID,
		Weight:    1.0,
		Dimensions: domain.Dimensions{
			Length: 30,
			Width:  20,
			Height: 10,
		},
		PackageType: "box",
	}

	// Create addresses (in real impl, would fetch from order service)
	recipient := domain.Address{
		Name:       "Customer",
		Street1:    "123 Main St",
		City:       "Anytown",
		State:      "CA",
		PostalCode: "90210",
		Country:    "US",
	}

	shipper := domain.Address{
		Name:       "WMS Warehouse",
		Street1:    "456 Warehouse Blvd",
		City:       "Logistics City",
		State:      "TX",
		PostalCode: "75001",
		Country:    "US",
	}

	waveID := "WAVE-" + uuid.New().String()[:8]

	// Create the shipment
	shipment := domain.NewShipment(shipmentID, orderID, packageID, waveID, carrier, pkg, recipient, shipper)

	// Save to repository
	if err := a.repo.Save(ctx, shipment); err != nil {
		logger.Error("Failed to save shipment", "error", err)
		return "", fmt.Errorf("failed to save shipment: %w", err)
	}

	logger.Info("Shipment created", "shipmentId", shipmentID)
	return shipmentID, nil
}

// GenerateShippingLabel generates a shipping label
func (a *ShippingActivities) GenerateShippingLabel(ctx context.Context, shipmentID string) (*LabelInfo, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Generating shipping label", "shipmentId", shipmentID)

	shipment, err := a.repo.FindByID(ctx, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to find shipment: %w", err)
	}

	if shipment == nil {
		return nil, fmt.Errorf("shipment not found: %s", shipmentID)
	}

	// Generate tracking number (in real impl, would call carrier API)
	trackingNumber := generateTrackingNumber(shipment.Carrier.Code)

	label := domain.ShippingLabel{
		TrackingNumber: trackingNumber,
		LabelFormat:    "PDF",
		LabelData:      "", // Base64 encoded label data
		LabelURL:       fmt.Sprintf("https://labels.example.com/%s.pdf", trackingNumber),
		GeneratedAt:    time.Now(),
	}

	if err := shipment.GenerateLabel(label); err != nil {
		return nil, fmt.Errorf("failed to generate label: %w", err)
	}

	if err := a.repo.Save(ctx, shipment); err != nil {
		return nil, fmt.Errorf("failed to save shipment: %w", err)
	}

	logger.Info("Shipping label generated", "shipmentId", shipmentID, "tracking", trackingNumber)
	return &LabelInfo{
		TrackingNumber: trackingNumber,
		LabelURL:       label.LabelURL,
	}, nil
}

// LabelInfo contains label information
type LabelInfo struct {
	TrackingNumber string `json:"trackingNumber"`
	LabelURL       string `json:"labelUrl"`
}

// ApplyTrackingNumber applies an existing tracking number to a shipment
func (a *ShippingActivities) ApplyTrackingNumber(ctx context.Context, input map[string]string) error {
	logger := activity.GetLogger(ctx)

	shipmentID := input["shipmentId"]
	trackingNumber := input["trackingNumber"]
	carrierCode := input["carrier"]

	logger.Info("Applying tracking number", "shipmentId", shipmentID, "tracking", trackingNumber)

	shipment, err := a.repo.FindByID(ctx, shipmentID)
	if err != nil {
		return fmt.Errorf("failed to find shipment: %w", err)
	}

	if shipment == nil {
		return fmt.Errorf("shipment not found: %s", shipmentID)
	}

	label := domain.ShippingLabel{
		TrackingNumber: trackingNumber,
		LabelFormat:    "PDF",
		LabelURL:       fmt.Sprintf("https://labels.example.com/%s.pdf", trackingNumber),
		GeneratedAt:    time.Now(),
	}

	// Update carrier if different
	if carrierCode != "" && carrierCode != shipment.Carrier.Code {
		shipment.Carrier.Code = carrierCode
		shipment.Carrier.Name = getCarrierName(carrierCode)
	}

	if err := shipment.GenerateLabel(label); err != nil {
		return fmt.Errorf("failed to apply label: %w", err)
	}

	if err := a.repo.Save(ctx, shipment); err != nil {
		return fmt.Errorf("failed to save shipment: %w", err)
	}

	logger.Info("Tracking number applied", "shipmentId", shipmentID, "tracking", trackingNumber)
	return nil
}

// AddToManifest adds a shipment to a manifest
func (a *ShippingActivities) AddToManifest(ctx context.Context, shipmentID string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Adding to manifest", "shipmentId", shipmentID)

	shipment, err := a.repo.FindByID(ctx, shipmentID)
	if err != nil {
		return "", fmt.Errorf("failed to find shipment: %w", err)
	}

	if shipment == nil {
		return "", fmt.Errorf("shipment not found: %s", shipmentID)
	}

	// Generate manifest ID
	manifestID := "MAN-" + time.Now().Format("20060102") + "-" + uuid.New().String()[:4]

	manifest := domain.Manifest{
		ManifestID:    manifestID,
		CarrierCode:   shipment.Carrier.Code,
		ShipmentCount: 1, // In real impl, would batch multiple shipments
		GeneratedAt:   time.Now(),
	}

	if err := shipment.AddToManifest(manifest); err != nil {
		return "", fmt.Errorf("failed to add to manifest: %w", err)
	}

	if err := a.repo.Save(ctx, shipment); err != nil {
		return "", fmt.Errorf("failed to save shipment: %w", err)
	}

	logger.Info("Added to manifest", "shipmentId", shipmentID, "manifestId", manifestID)
	return manifestID, nil
}

// ConfirmShipment confirms that a shipment has been picked up
func (a *ShippingActivities) ConfirmShipment(ctx context.Context, input map[string]interface{}) error {
	logger := activity.GetLogger(ctx)

	shipmentID, _ := input["shipmentId"].(string)

	logger.Info("Confirming shipment", "shipmentId", shipmentID)

	shipment, err := a.repo.FindByID(ctx, shipmentID)
	if err != nil {
		return fmt.Errorf("failed to find shipment: %w", err)
	}

	if shipment == nil {
		return fmt.Errorf("shipment not found: %s", shipmentID)
	}

	// Calculate estimated delivery (3-5 business days)
	estimatedDelivery := time.Now().AddDate(0, 0, 5)

	if err := shipment.ConfirmShipment(&estimatedDelivery); err != nil {
		return fmt.Errorf("failed to confirm shipment: %w", err)
	}

	if err := a.repo.Save(ctx, shipment); err != nil {
		return fmt.Errorf("failed to save shipment: %w", err)
	}

	logger.Info("Shipment confirmed", "shipmentId", shipmentID)
	return nil
}

// CancelShipment cancels a shipment
func (a *ShippingActivities) CancelShipment(ctx context.Context, shipmentID, reason string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Cancelling shipment", "shipmentId", shipmentID, "reason", reason)

	shipment, err := a.repo.FindByID(ctx, shipmentID)
	if err != nil {
		return fmt.Errorf("failed to find shipment: %w", err)
	}

	if shipment == nil {
		return fmt.Errorf("shipment not found: %s", shipmentID)
	}

	if err := shipment.Cancel(reason); err != nil {
		return fmt.Errorf("failed to cancel shipment: %w", err)
	}

	if err := a.repo.Save(ctx, shipment); err != nil {
		return fmt.Errorf("failed to save shipment: %w", err)
	}

	logger.Info("Shipment cancelled", "shipmentId", shipmentID)
	return nil
}

// Helper functions

func getCarrierName(code string) string {
	switch code {
	case "UPS":
		return "United Parcel Service"
	case "FEDEX":
		return "FedEx"
	case "USPS":
		return "United States Postal Service"
	case "DHL":
		return "DHL Express"
	default:
		return code
	}
}

func generateTrackingNumber(carrierCode string) string {
	switch carrierCode {
	case "UPS":
		return "1Z" + uuid.New().String()[:16]
	case "FEDEX":
		return uuid.New().String()[:12]
	case "USPS":
		return "9400" + uuid.New().String()[:18]
	default:
		return "TRK" + uuid.New().String()[:12]
	}
}
