package domain

import (
	"context"
)

// OrderRepository defines the interface for order persistence
type OrderRepository interface {
	// Save persists an order (upsert)
	Save(ctx context.Context, order *Order) error

	// FindByID retrieves an order by its OrderID
	FindByID(ctx context.Context, orderID string) (*Order, error)

	// FindByCustomerID retrieves all orders for a customer
	FindByCustomerID(ctx context.Context, customerID string, pagination Pagination) ([]*Order, error)

	// FindByStatus retrieves orders by status
	FindByStatus(ctx context.Context, status Status, pagination Pagination) ([]*Order, error)

	// FindByWaveID retrieves all orders in a wave
	FindByWaveID(ctx context.Context, waveID string) ([]*Order, error)

	// FindValidatedOrders retrieves orders ready for wave assignment
	FindValidatedOrders(ctx context.Context, priority Priority, limit int) ([]*Order, error)

	// UpdateStatus updates the order status
	UpdateStatus(ctx context.Context, orderID string, status Status) error

	// AssignToWave assigns an order to a wave
	AssignToWave(ctx context.Context, orderID string, waveID string) error

	// Delete deletes an order (soft delete in practice)
	Delete(ctx context.Context, orderID string) error

	// Count returns the total number of orders matching the filter
	Count(ctx context.Context, filter OrderFilter) (int64, error)
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

// OrderFilter represents filter options for querying orders
type OrderFilter struct {
	CustomerID *string
	Status     *Status
	Priority   *Priority
	WaveID     *string
	FromDate   *string
	ToDate     *string
	// Multi-tenant filters
	SellerID  *string // Filter by seller (for 3PL queries)
	ChannelID *string // Filter by sales channel
}

// EventPublisher defines the interface for publishing domain events
type EventPublisher interface {
	// Publish publishes a domain event
	Publish(ctx context.Context, event DomainEvent) error

	// PublishAll publishes multiple domain events
	PublishAll(ctx context.Context, events []DomainEvent) error
}
