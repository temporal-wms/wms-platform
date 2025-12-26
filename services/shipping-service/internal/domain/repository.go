package domain

import "context"

// ShipmentRepository defines the interface for shipment persistence
type ShipmentRepository interface {
	Save(ctx context.Context, shipment *Shipment) error
	FindByID(ctx context.Context, shipmentID string) (*Shipment, error)
	FindByOrderID(ctx context.Context, orderID string) (*Shipment, error)
	FindByTrackingNumber(ctx context.Context, trackingNumber string) (*Shipment, error)
	FindByStatus(ctx context.Context, status ShipmentStatus) ([]*Shipment, error)
	FindByCarrier(ctx context.Context, carrierCode string) ([]*Shipment, error)
	FindByManifestID(ctx context.Context, manifestID string) ([]*Shipment, error)
	FindPendingForManifest(ctx context.Context, carrierCode string) ([]*Shipment, error)
	Delete(ctx context.Context, shipmentID string) error
}

// EventPublisher defines the interface for publishing domain events
type EventPublisher interface {
	Publish(ctx context.Context, event DomainEvent) error
	PublishAll(ctx context.Context, events []DomainEvent) error
}
