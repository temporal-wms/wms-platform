package domain

import "time"

// DomainEvent is the interface for all domain events
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// ShipmentCreatedEvent is published when a shipment is created
type ShipmentCreatedEvent struct {
	ShipmentID string    `json:"shipmentId"`
	OrderID    string    `json:"orderId"`
	Carrier    string    `json:"carrier"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (e *ShipmentCreatedEvent) EventType() string    { return "wms.shipping.shipment-created" }
func (e *ShipmentCreatedEvent) OccurredAt() time.Time { return e.CreatedAt }

// LabelGeneratedEvent is published when a shipping label is generated
type LabelGeneratedEvent struct {
	ShipmentID     string    `json:"shipmentId"`
	TrackingNumber string    `json:"trackingNumber"`
	Carrier        string    `json:"carrier"`
	GeneratedAt    time.Time `json:"generatedAt"`
}

func (e *LabelGeneratedEvent) EventType() string    { return "wms.shipping.label-generated" }
func (e *LabelGeneratedEvent) OccurredAt() time.Time { return e.GeneratedAt }

// ShipmentManifestedEvent is published when a shipment is added to a manifest
type ShipmentManifestedEvent struct {
	ShipmentID     string    `json:"shipmentId"`
	ManifestID     string    `json:"manifestId"`
	TrackingNumber string    `json:"trackingNumber"`
	ManifestedAt   time.Time `json:"manifestedAt"`
}

func (e *ShipmentManifestedEvent) EventType() string    { return "wms.shipping.manifested" }
func (e *ShipmentManifestedEvent) OccurredAt() time.Time { return e.ManifestedAt }

// ShipConfirmedEvent is published when a shipment is confirmed shipped
type ShipConfirmedEvent struct {
	ShipmentID        string     `json:"shipmentId"`
	OrderID           string     `json:"orderId"`
	TrackingNumber    string     `json:"trackingNumber"`
	Carrier           string     `json:"carrier"`
	EstimatedDelivery *time.Time `json:"estimatedDelivery,omitempty"`
	ShippedAt         time.Time  `json:"shippedAt"`
}

func (e *ShipConfirmedEvent) EventType() string    { return "wms.shipping.confirmed" }
func (e *ShipConfirmedEvent) OccurredAt() time.Time { return e.ShippedAt }
