package domain

import (
	"context"
	"time"
)

// InboundShipmentRepository defines the interface for shipment persistence
type InboundShipmentRepository interface {
	// Save persists an inbound shipment (upsert)
	Save(ctx context.Context, shipment *InboundShipment) error

	// FindByID retrieves a shipment by its ShipmentID
	FindByID(ctx context.Context, shipmentID string) (*InboundShipment, error)

	// FindByASNID retrieves a shipment by ASN ID
	FindByASNID(ctx context.Context, asnID string) (*InboundShipment, error)

	// FindByStatus retrieves shipments by status
	FindByStatus(ctx context.Context, status ShipmentStatus, pagination Pagination) ([]*InboundShipment, error)

	// FindBySupplierID retrieves shipments by supplier
	FindBySupplierID(ctx context.Context, supplierID string, pagination Pagination) ([]*InboundShipment, error)

	// FindByDockID retrieves shipments at a specific dock
	FindByDockID(ctx context.Context, dockID string) ([]*InboundShipment, error)

	// FindExpectedToday retrieves shipments expected to arrive today
	FindExpectedToday(ctx context.Context) ([]*InboundShipment, error)

	// FindPendingReceiving retrieves shipments pending receiving
	FindPendingReceiving(ctx context.Context, limit int) ([]*InboundShipment, error)

	// FindAll retrieves all shipments up to the specified limit
	FindAll(ctx context.Context, limit int) ([]*InboundShipment, error)

	// FindExpectedArrivals retrieves shipments expected within a time range
	FindExpectedArrivals(ctx context.Context, from, to time.Time) ([]*InboundShipment, error)

	// UpdateStatus updates the shipment status
	UpdateStatus(ctx context.Context, shipmentID string, status ShipmentStatus) error

	// Delete deletes a shipment
	Delete(ctx context.Context, shipmentID string) error

	// Count returns the total number of shipments matching the filter
	Count(ctx context.Context, filter ShipmentFilter) (int64, error)
}

// Pagination represents pagination options
type Pagination struct {
	Page     int64
	PageSize int64
}

// DefaultPagination returns default pagination options
func DefaultPagination() Pagination {
	return Pagination{
		Page:     1,
		PageSize: 20,
	}
}

// Skip returns the number of documents to skip
func (p Pagination) Skip() int64 {
	return (p.Page - 1) * p.PageSize
}

// Limit returns the maximum number of documents to return
func (p Pagination) Limit() int64 {
	return p.PageSize
}

// ShipmentFilter represents filter options for querying shipments
type ShipmentFilter struct {
	SupplierID      *string
	Status          *ShipmentStatus
	DockID          *string
	PurchaseOrderID *string
	FromDate        *string
	ToDate          *string
}

// EventPublisher defines the interface for publishing domain events
type EventPublisher interface {
	// Publish publishes a domain event
	Publish(ctx context.Context, event DomainEvent) error

	// PublishAll publishes multiple domain events
	PublishAll(ctx context.Context, events []DomainEvent) error
}
