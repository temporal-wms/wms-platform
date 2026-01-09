package domain

import (
	"context"
	"time"
)

// BillableActivityRepository defines the interface for activity persistence
type BillableActivityRepository interface {
	// Save persists a billable activity
	Save(ctx context.Context, activity *BillableActivity) error

	// SaveAll persists multiple activities
	SaveAll(ctx context.Context, activities []*BillableActivity) error

	// FindByID retrieves an activity by ID
	FindByID(ctx context.Context, activityID string) (*BillableActivity, error)

	// FindBySellerID retrieves activities for a seller
	FindBySellerID(ctx context.Context, sellerID string, pagination Pagination) ([]*BillableActivity, error)

	// FindUninvoiced retrieves activities not yet invoiced for a seller
	FindUninvoiced(ctx context.Context, sellerID string, periodStart, periodEnd time.Time) ([]*BillableActivity, error)

	// FindByInvoiceID retrieves activities for an invoice
	FindByInvoiceID(ctx context.Context, invoiceID string) ([]*BillableActivity, error)

	// MarkAsInvoiced marks activities as invoiced
	MarkAsInvoiced(ctx context.Context, activityIDs []string, invoiceID string) error

	// SumBySellerAndType returns sum of amounts by activity type for a seller
	SumBySellerAndType(ctx context.Context, sellerID string, periodStart, periodEnd time.Time) (map[ActivityType]float64, error)

	// Count returns total count matching filter
	Count(ctx context.Context, filter ActivityFilter) (int64, error)
}

// InvoiceRepository defines the interface for invoice persistence
type InvoiceRepository interface {
	// Save persists an invoice
	Save(ctx context.Context, invoice *Invoice) error

	// FindByID retrieves an invoice by ID
	FindByID(ctx context.Context, invoiceID string) (*Invoice, error)

	// FindBySellerID retrieves invoices for a seller
	FindBySellerID(ctx context.Context, sellerID string, pagination Pagination) ([]*Invoice, error)

	// FindByStatus retrieves invoices by status
	FindByStatus(ctx context.Context, status InvoiceStatus, pagination Pagination) ([]*Invoice, error)

	// FindOverdue retrieves overdue invoices
	FindOverdue(ctx context.Context) ([]*Invoice, error)

	// FindByPeriod retrieves invoices for a billing period
	FindByPeriod(ctx context.Context, sellerID string, periodStart, periodEnd time.Time) (*Invoice, error)

	// UpdateStatus updates invoice status
	UpdateStatus(ctx context.Context, invoiceID string, status InvoiceStatus) error

	// Count returns total count matching filter
	Count(ctx context.Context, filter InvoiceFilter) (int64, error)
}

// StorageCalculationRepository defines the interface for storage calculation persistence
type StorageCalculationRepository interface {
	// Save persists a storage calculation
	Save(ctx context.Context, calc *StorageCalculation) error

	// FindBySellerAndDate retrieves calculation for a seller on a date
	FindBySellerAndDate(ctx context.Context, sellerID string, date time.Time) (*StorageCalculation, error)

	// FindBySellerAndPeriod retrieves calculations for a period
	FindBySellerAndPeriod(ctx context.Context, sellerID string, start, end time.Time) ([]*StorageCalculation, error)

	// SumByPeriod returns total storage fees for a period
	SumByPeriod(ctx context.Context, sellerID string, start, end time.Time) (float64, error)
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

// ActivityFilter represents filter options for querying activities
type ActivityFilter struct {
	TenantID      *string
	SellerID      *string
	FacilityID    *string
	Type          *ActivityType
	Invoiced      *bool
	ReferenceType *string
	ReferenceID   *string
	FromDate      *time.Time
	ToDate        *time.Time
}

// InvoiceFilter represents filter options for querying invoices
type InvoiceFilter struct {
	TenantID   *string
	SellerID   *string
	FacilityID *string
	Status     *InvoiceStatus
	FromDate   *time.Time
	ToDate     *time.Time
	Overdue    *bool
}
