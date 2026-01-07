package domain

import (
	"context"
)

// SellerRepository defines the interface for seller persistence
type SellerRepository interface {
	// Save persists a seller (upsert)
	Save(ctx context.Context, seller *Seller) error

	// FindByID retrieves a seller by its SellerID
	FindByID(ctx context.Context, sellerID string) (*Seller, error)

	// FindByTenantID retrieves all sellers for a tenant
	FindByTenantID(ctx context.Context, tenantID string, pagination Pagination) ([]*Seller, error)

	// FindByStatus retrieves sellers by status
	FindByStatus(ctx context.Context, status SellerStatus, pagination Pagination) ([]*Seller, error)

	// FindByAPIKey finds a seller by API key (hashed)
	FindByAPIKey(ctx context.Context, hashedKey string) (*Seller, error)

	// FindByEmail finds a seller by contact email
	FindByEmail(ctx context.Context, email string) (*Seller, error)

	// UpdateStatus updates the seller status
	UpdateStatus(ctx context.Context, sellerID string, status SellerStatus) error

	// Delete deletes a seller (soft delete in practice)
	Delete(ctx context.Context, sellerID string) error

	// Count returns the total number of sellers matching the filter
	Count(ctx context.Context, filter SellerFilter) (int64, error)

	// Search searches sellers by company name or email
	Search(ctx context.Context, query string, pagination Pagination) ([]*Seller, error)
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

// SellerFilter represents filter options for querying sellers
type SellerFilter struct {
	TenantID   *string
	Status     *SellerStatus
	FacilityID *string
	HasChannel *string // Filter by channel type (shopify, amazon, etc.)
}

// EventPublisher defines the interface for publishing domain events
type EventPublisher interface {
	// Publish publishes a domain event
	Publish(ctx context.Context, event DomainEvent) error

	// PublishAll publishes multiple domain events
	PublishAll(ctx context.Context, events []DomainEvent) error
}
