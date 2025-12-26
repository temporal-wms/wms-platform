package application

import (
	"time"

	"github.com/wms-platform/shipping-service/internal/domain"
)

// CreateShipmentCommand represents the command to create a new shipment
type CreateShipmentCommand struct {
	ShipmentID string
	OrderID    string
	PackageID  string
	WaveID     string
	Carrier    domain.Carrier
	Package    domain.PackageInfo
	Recipient  domain.Address
	Shipper    domain.Address
}

// GenerateLabelCommand represents the command to generate a shipping label
type GenerateLabelCommand struct {
	ShipmentID string
	Label      domain.ShippingLabel
}

// AddToManifestCommand represents the command to add shipment to manifest
type AddToManifestCommand struct {
	ShipmentID string
	Manifest   domain.Manifest
}

// ConfirmShipmentCommand represents the command to confirm shipment
type ConfirmShipmentCommand struct {
	ShipmentID        string
	EstimatedDelivery *time.Time
}

// GetShipmentQuery represents the query to get a shipment by ID
type GetShipmentQuery struct {
	ShipmentID string
}

// GetByOrderQuery represents the query to get shipment by order ID
type GetByOrderQuery struct {
	OrderID string
}

// GetByTrackingQuery represents the query to get shipment by tracking number
type GetByTrackingQuery struct {
	TrackingNumber string
}

// GetByStatusQuery represents the query to get shipments by status
type GetByStatusQuery struct {
	Status string
}

// GetByCarrierQuery represents the query to get shipments by carrier
type GetByCarrierQuery struct {
	CarrierCode string
}

// GetPendingForManifestQuery represents the query to get pending shipments for manifest
type GetPendingForManifestQuery struct {
	CarrierCode string
}
