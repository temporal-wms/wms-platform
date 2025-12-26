package domain

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Errors
var (
	ErrShipmentAlreadyManifested = errors.New("shipment is already manifested")
	ErrShipmentAlreadyShipped    = errors.New("shipment is already shipped")
	ErrNoLabel                   = errors.New("shipment has no label")
)

// ShipmentStatus represents the status of a shipment
type ShipmentStatus string

const (
	ShipmentStatusPending    ShipmentStatus = "pending"
	ShipmentStatusLabeled    ShipmentStatus = "labeled"
	ShipmentStatusManifested ShipmentStatus = "manifested"
	ShipmentStatusShipped    ShipmentStatus = "shipped"
	ShipmentStatusDelivered  ShipmentStatus = "delivered"
	ShipmentStatusCancelled  ShipmentStatus = "cancelled"
)

// Shipment is the aggregate root for the Shipping bounded context (SLAM)
type Shipment struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	ShipmentID      string             `bson:"shipmentId"`
	OrderID         string             `bson:"orderId"`
	PackageID       string             `bson:"packageId"`
	WaveID          string             `bson:"waveId"`
	Status          ShipmentStatus     `bson:"status"`
	Carrier         Carrier            `bson:"carrier"`
	Label           *ShippingLabel     `bson:"label,omitempty"`
	Manifest        *Manifest          `bson:"manifest,omitempty"`
	Package         PackageInfo        `bson:"package"`
	Recipient       Address            `bson:"recipient"`
	Shipper         Address            `bson:"shipper"`
	ServiceType     string             `bson:"serviceType"`
	EstimatedDelivery *time.Time       `bson:"estimatedDelivery,omitempty"`
	ActualDelivery  *time.Time         `bson:"actualDelivery,omitempty"`
	CreatedAt       time.Time          `bson:"createdAt"`
	UpdatedAt       time.Time          `bson:"updatedAt"`
	LabeledAt       *time.Time         `bson:"labeledAt,omitempty"`
	ManifestedAt    *time.Time         `bson:"manifestedAt,omitempty"`
	ShippedAt       *time.Time         `bson:"shippedAt,omitempty"`
	DomainEvents    []DomainEvent      `bson:"-"`
}

// Carrier represents a shipping carrier
type Carrier struct {
	Code        string `bson:"code"` // UPS, FEDEX, USPS
	Name        string `bson:"name"`
	AccountID   string `bson:"accountId"`
	ServiceType string `bson:"serviceType"`
}

// ShippingLabel represents the shipping label
type ShippingLabel struct {
	TrackingNumber string    `bson:"trackingNumber"`
	LabelFormat    string    `bson:"labelFormat"` // PDF, ZPL, PNG
	LabelData      string    `bson:"labelData"`   // Base64 encoded
	LabelURL       string    `bson:"labelUrl,omitempty"`
	GeneratedAt    time.Time `bson:"generatedAt"`
}

// Manifest represents a shipping manifest
type Manifest struct {
	ManifestID    string    `bson:"manifestId"`
	CarrierCode   string    `bson:"carrierCode"`
	ShipmentCount int       `bson:"shipmentCount"`
	GeneratedAt   time.Time `bson:"generatedAt"`
}

// PackageInfo represents package information
type PackageInfo struct {
	PackageID   string     `bson:"packageId"`
	Weight      float64    `bson:"weight"`      // in kg
	Dimensions  Dimensions `bson:"dimensions"`
	PackageType string     `bson:"packageType"`
}

// Dimensions represents package dimensions
type Dimensions struct {
	Length float64 `bson:"length"` // in cm
	Width  float64 `bson:"width"`
	Height float64 `bson:"height"`
}

// Address represents a shipping address
type Address struct {
	Name       string `bson:"name"`
	Company    string `bson:"company,omitempty"`
	Street1    string `bson:"street1"`
	Street2    string `bson:"street2,omitempty"`
	City       string `bson:"city"`
	State      string `bson:"state"`
	PostalCode string `bson:"postalCode"`
	Country    string `bson:"country"`
	Phone      string `bson:"phone,omitempty"`
	Email      string `bson:"email,omitempty"`
}

// NewShipment creates a new Shipment aggregate
func NewShipment(shipmentID, orderID, packageID, waveID string, carrier Carrier, pkg PackageInfo, recipient, shipper Address) *Shipment {
	now := time.Now()
	s := &Shipment{
		ShipmentID:   shipmentID,
		OrderID:      orderID,
		PackageID:    packageID,
		WaveID:       waveID,
		Status:       ShipmentStatusPending,
		Carrier:      carrier,
		Package:      pkg,
		Recipient:    recipient,
		Shipper:      shipper,
		ServiceType:  carrier.ServiceType,
		CreatedAt:    now,
		UpdatedAt:    now,
		DomainEvents: make([]DomainEvent, 0),
	}

	s.AddDomainEvent(&ShipmentCreatedEvent{
		ShipmentID: shipmentID,
		OrderID:    orderID,
		Carrier:    carrier.Code,
		CreatedAt:  now,
	})

	return s
}

// GenerateLabel generates and applies a shipping label
func (s *Shipment) GenerateLabel(label ShippingLabel) error {
	if s.Status == ShipmentStatusShipped {
		return ErrShipmentAlreadyShipped
	}

	now := time.Now()
	s.Label = &label
	s.Status = ShipmentStatusLabeled
	s.LabeledAt = &now
	s.UpdatedAt = now

	s.AddDomainEvent(&LabelGeneratedEvent{
		ShipmentID:     s.ShipmentID,
		TrackingNumber: label.TrackingNumber,
		Carrier:        s.Carrier.Code,
		GeneratedAt:    now,
	})

	return nil
}

// AddToManifest adds the shipment to a manifest
func (s *Shipment) AddToManifest(manifest Manifest) error {
	if s.Label == nil {
		return ErrNoLabel
	}
	if s.Status == ShipmentStatusManifested {
		return ErrShipmentAlreadyManifested
	}

	now := time.Now()
	s.Manifest = &manifest
	s.Status = ShipmentStatusManifested
	s.ManifestedAt = &now
	s.UpdatedAt = now

	s.AddDomainEvent(&ShipmentManifestedEvent{
		ShipmentID:     s.ShipmentID,
		ManifestID:     manifest.ManifestID,
		TrackingNumber: s.Label.TrackingNumber,
		ManifestedAt:   now,
	})

	return nil
}

// ConfirmShipment confirms the shipment has left the warehouse
func (s *Shipment) ConfirmShipment(estimatedDelivery *time.Time) error {
	if s.Status == ShipmentStatusShipped {
		return ErrShipmentAlreadyShipped
	}

	now := time.Now()
	s.Status = ShipmentStatusShipped
	s.ShippedAt = &now
	s.EstimatedDelivery = estimatedDelivery
	s.UpdatedAt = now

	s.AddDomainEvent(&ShipConfirmedEvent{
		ShipmentID:        s.ShipmentID,
		OrderID:           s.OrderID,
		TrackingNumber:    s.Label.TrackingNumber,
		Carrier:           s.Carrier.Code,
		EstimatedDelivery: estimatedDelivery,
		ShippedAt:         now,
	})

	return nil
}

// ConfirmDelivery confirms the shipment has been delivered
func (s *Shipment) ConfirmDelivery(deliveredAt time.Time) error {
	s.Status = ShipmentStatusDelivered
	s.ActualDelivery = &deliveredAt
	s.UpdatedAt = time.Now()
	return nil
}

// Cancel cancels the shipment
func (s *Shipment) Cancel(reason string) error {
	if s.Status == ShipmentStatusShipped || s.Status == ShipmentStatusDelivered {
		return errors.New("cannot cancel shipped or delivered shipment")
	}

	s.Status = ShipmentStatusCancelled
	s.UpdatedAt = time.Now()
	return nil
}

// AddDomainEvent adds a domain event
func (s *Shipment) AddDomainEvent(event DomainEvent) {
	s.DomainEvents = append(s.DomainEvents, event)
}

// ClearDomainEvents clears all domain events
func (s *Shipment) ClearDomainEvents() {
	s.DomainEvents = make([]DomainEvent, 0)
}

// GetDomainEvents returns all domain events
func (s *Shipment) GetDomainEvents() []DomainEvent {
	return s.DomainEvents
}
