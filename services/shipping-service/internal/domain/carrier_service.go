package domain

import (
	"context"
	"time"
)

// CarrierService is the domain interface (port) for carrier integration
// Implementations (adapters) will translate domain models to carrier-specific APIs
type CarrierService interface {
	// GenerateLabel generates a shipping label for a shipment
	GenerateLabel(ctx context.Context, request LabelRequest) (*ShippingLabel, error)

	// CreateManifest creates an end-of-day manifest for multiple shipments
	CreateManifest(ctx context.Context, shipments []Shipment) (*Manifest, error)

	// TrackShipment retrieves tracking information for a shipment
	TrackShipment(ctx context.Context, trackingNumber string) (*TrackingInfo, error)

	// ValidateAddress validates a shipping address with the carrier
	ValidateAddress(ctx context.Context, address Address) (*AddressValidationResult, error)

	// GetRates retrieves shipping rates for a package
	GetRates(ctx context.Context, request RateRequest) ([]ShippingRate, error)

	// CancelShipment cancels a shipment with the carrier
	CancelShipment(ctx context.Context, trackingNumber string) error

	// GetCarrierCode returns the carrier code this service handles (UPS, FEDEX, etc.)
	GetCarrierCode() string
}

// LabelRequest represents a request to generate a shipping label
type LabelRequest struct {
	ShipmentID  string
	PackageInfo PackageInfo
	Shipper     Address
	Recipient   Address
	ServiceType string
	LabelFormat string // PDF, ZPL, PNG
	Reference1  string
	Reference2  string
}

// RateRequest represents a request to get shipping rates
type RateRequest struct {
	Shipper     Address
	Recipient   Address
	PackageInfo PackageInfo
	ServiceType string
}

// ShippingRate represents a shipping rate from a carrier
type ShippingRate struct {
	ServiceType       string
	ServiceName       string
	TotalCost         float64
	Currency          string
	EstimatedDelivery time.Time
	IsGuaranteed      bool
}

// TrackingInfo represents tracking information for a shipment
type TrackingInfo struct {
	TrackingNumber string
	Status         string
	StatusDetail   string
	CurrentLocation string
	EstimatedDelivery *time.Time
	ActualDelivery  *time.Time
	Events         []TrackingEvent
}

// TrackingEvent represents a single tracking event
type TrackingEvent struct {
	Timestamp   time.Time
	Location    string
	Status      string
	Description string
}

// AddressValidationResult represents the result of address validation
type AddressValidationResult struct {
	IsValid         bool
	SuggestedAddress *Address
	ValidationErrors []string
}

// CarrierError represents errors from carrier APIs
type CarrierError struct {
	Code       string
	Message    string
	Severity   string // ERROR, WARNING, INFO
	Retryable  bool
	OriginalErr error
}

func (e *CarrierError) Error() string {
	if e.OriginalErr != nil {
		return e.Message + ": " + e.OriginalErr.Error()
	}
	return e.Message
}

func (e *CarrierError) Unwrap() error {
	return e.OriginalErr
}

// NewCarrierError creates a new CarrierError
func NewCarrierError(code, message, severity string, retryable bool, originalErr error) *CarrierError {
	return &CarrierError{
		Code:       code,
		Message:    message,
		Severity:   severity,
		Retryable:  retryable,
		OriginalErr: originalErr,
	}
}
