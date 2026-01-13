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
	TenantID        string             `bson:"tenantId"`
	FacilityID      string             `bson:"facilityId"`
	WarehouseID     string             `bson:"warehouseId"`
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

	// Safely get tracking number - label might be nil if generated separately
	trackingNumber := ""
	if s.Label != nil {
		trackingNumber = s.Label.TrackingNumber
	}

	s.AddDomainEvent(&ShipConfirmedEvent{
		ShipmentID:        s.ShipmentID,
		OrderID:           s.OrderID,
		TrackingNumber:    trackingNumber,
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

// ===== OutboundManifest Aggregate =====

// ManifestStatus represents the status of an outbound manifest
type ManifestStatus string

const (
	ManifestStatusOpen       ManifestStatus = "open"
	ManifestStatusClosed     ManifestStatus = "closed"
	ManifestStatusDispatched ManifestStatus = "dispatched"
	ManifestStatusCancelled  ManifestStatus = "cancelled"
)

// ManifestPackage represents a package in the manifest
type ManifestPackage struct {
	PackageID      string    `bson:"packageId" json:"packageId"`
	ShipmentID     string    `bson:"shipmentId" json:"shipmentId"`
	OrderID        string    `bson:"orderId" json:"orderId"`
	TrackingNumber string    `bson:"trackingNumber" json:"trackingNumber"`
	Weight         float64   `bson:"weight" json:"weight"`
	AddedAt        time.Time `bson:"addedAt" json:"addedAt"`
}

// OutboundManifest represents a carrier manifest for outbound shipments
type OutboundManifest struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ManifestID      string             `bson:"manifestId" json:"manifestId"`
	TenantID        string             `bson:"tenantId" json:"tenantId"`
	FacilityID      string             `bson:"facilityId" json:"facilityId"`
	WarehouseID     string             `bson:"warehouseId" json:"warehouseId"`
	CarrierID       string             `bson:"carrierId" json:"carrierId"`
	CarrierName     string             `bson:"carrierName" json:"carrierName"`
	ServiceType     string             `bson:"serviceType" json:"serviceType"`
	TrailerID       string             `bson:"trailerId,omitempty" json:"trailerId,omitempty"`
	DispatchDock    string             `bson:"dispatchDock,omitempty" json:"dispatchDock,omitempty"`
	Packages        []ManifestPackage  `bson:"packages" json:"packages"`
	TotalPackages   int                `bson:"totalPackages" json:"totalPackages"`
	TotalWeight     float64            `bson:"totalWeight" json:"totalWeight"`
	Status          ManifestStatus     `bson:"status" json:"status"`
	ScheduledPickup *time.Time         `bson:"scheduledPickup,omitempty" json:"scheduledPickup,omitempty"`
	ClosedAt        *time.Time         `bson:"closedAt,omitempty" json:"closedAt,omitempty"`
	DispatchedAt    *time.Time         `bson:"dispatchedAt,omitempty" json:"dispatchedAt,omitempty"`
	CreatedAt       time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time          `bson:"updatedAt" json:"updatedAt"`
	DomainEvents    []DomainEvent      `bson:"-" json:"-"`
}

// NewOutboundManifest creates a new OutboundManifest aggregate
func NewOutboundManifest(manifestID, carrierID, carrierName, serviceType string) *OutboundManifest {
	now := time.Now().UTC()
	return &OutboundManifest{
		ID:            primitive.NewObjectID(),
		ManifestID:    manifestID,
		CarrierID:     carrierID,
		CarrierName:   carrierName,
		ServiceType:   serviceType,
		Packages:      make([]ManifestPackage, 0),
		TotalPackages: 0,
		TotalWeight:   0,
		Status:        ManifestStatusOpen,
		CreatedAt:     now,
		UpdatedAt:     now,
		DomainEvents:  make([]DomainEvent, 0),
	}
}

// AddPackage adds a package to the manifest
func (m *OutboundManifest) AddPackage(pkg ManifestPackage) error {
	if m.Status != ManifestStatusOpen {
		return errors.New("cannot add package to closed manifest")
	}

	pkg.AddedAt = time.Now().UTC()
	m.Packages = append(m.Packages, pkg)
	m.TotalPackages++
	m.TotalWeight += pkg.Weight
	m.UpdatedAt = time.Now().UTC()

	return nil
}

// RemovePackage removes a package from the manifest
func (m *OutboundManifest) RemovePackage(packageID string) error {
	if m.Status != ManifestStatusOpen {
		return errors.New("cannot remove package from closed manifest")
	}

	for i, pkg := range m.Packages {
		if pkg.PackageID == packageID {
			m.TotalWeight -= pkg.Weight
			m.Packages = append(m.Packages[:i], m.Packages[i+1:]...)
			m.TotalPackages--
			m.UpdatedAt = time.Now().UTC()
			return nil
		}
	}

	return errors.New("package not found in manifest")
}

// Close closes the manifest for further additions
func (m *OutboundManifest) Close() error {
	if m.Status != ManifestStatusOpen {
		return errors.New("manifest is not open")
	}
	if m.TotalPackages == 0 {
		return errors.New("cannot close empty manifest")
	}

	now := time.Now().UTC()
	m.Status = ManifestStatusClosed
	m.ClosedAt = &now
	m.UpdatedAt = now

	m.addDomainEvent(&ManifestClosedEvent{
		ManifestID:    m.ManifestID,
		CarrierID:     m.CarrierID,
		PackageCount:  m.TotalPackages,
		TotalWeight:   m.TotalWeight,
		ClosedAt:      now,
	})

	return nil
}

// AssignTrailer assigns a trailer and dispatch dock
func (m *OutboundManifest) AssignTrailer(trailerID, dispatchDock string) error {
	if m.Status != ManifestStatusClosed {
		return errors.New("manifest must be closed before assigning trailer")
	}

	m.TrailerID = trailerID
	m.DispatchDock = dispatchDock
	m.UpdatedAt = time.Now().UTC()

	return nil
}

// Dispatch dispatches the manifest
func (m *OutboundManifest) Dispatch() error {
	if m.Status != ManifestStatusClosed {
		return errors.New("manifest must be closed before dispatch")
	}

	now := time.Now().UTC()
	m.Status = ManifestStatusDispatched
	m.DispatchedAt = &now
	m.UpdatedAt = now

	m.addDomainEvent(&ManifestDispatchedEvent{
		ManifestID:   m.ManifestID,
		CarrierID:    m.CarrierID,
		TrailerID:    m.TrailerID,
		DispatchDock: m.DispatchDock,
		PackageCount: m.TotalPackages,
		TotalWeight:  m.TotalWeight,
		DispatchedAt: now,
	})

	return nil
}

// Cancel cancels the manifest
func (m *OutboundManifest) Cancel(reason string) error {
	if m.Status == ManifestStatusDispatched {
		return errors.New("cannot cancel dispatched manifest")
	}

	m.Status = ManifestStatusCancelled
	m.UpdatedAt = time.Now().UTC()

	return nil
}

// GetPackageIDs returns all package IDs in the manifest
func (m *OutboundManifest) GetPackageIDs() []string {
	ids := make([]string, len(m.Packages))
	for i, pkg := range m.Packages {
		ids[i] = pkg.PackageID
	}
	return ids
}

// addDomainEvent adds a domain event
func (m *OutboundManifest) addDomainEvent(event DomainEvent) {
	m.DomainEvents = append(m.DomainEvents, event)
}

// GetManifestDomainEvents returns all domain events
func (m *OutboundManifest) GetManifestDomainEvents() []DomainEvent {
	return m.DomainEvents
}

// ClearManifestDomainEvents clears all domain events
func (m *OutboundManifest) ClearManifestDomainEvents() {
	m.DomainEvents = make([]DomainEvent, 0)
}
