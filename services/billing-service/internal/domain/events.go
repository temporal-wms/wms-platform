package domain

import "time"

// DomainEvent is the base interface for all domain events
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// InvoiceCreatedEvent is emitted when a new invoice is created
type InvoiceCreatedEvent struct {
	InvoiceID   string    `json:"invoiceId"`
	SellerID    string    `json:"sellerId"`
	PeriodStart time.Time `json:"periodStart"`
	PeriodEnd   time.Time `json:"periodEnd"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (e *InvoiceCreatedEvent) EventType() string    { return "billing.invoice.created" }
func (e *InvoiceCreatedEvent) OccurredAt() time.Time { return e.CreatedAt }

// InvoiceFinalizedEvent is emitted when an invoice is finalized
type InvoiceFinalizedEvent struct {
	InvoiceID   string    `json:"invoiceId"`
	SellerID    string    `json:"sellerId"`
	Total       float64   `json:"total"`
	DueDate     time.Time `json:"dueDate"`
	FinalizedAt time.Time `json:"finalizedAt"`
}

func (e *InvoiceFinalizedEvent) EventType() string    { return "billing.invoice.finalized" }
func (e *InvoiceFinalizedEvent) OccurredAt() time.Time { return e.FinalizedAt }

// InvoicePaidEvent is emitted when an invoice is paid
type InvoicePaidEvent struct {
	InvoiceID     string    `json:"invoiceId"`
	SellerID      string    `json:"sellerId"`
	Amount        float64   `json:"amount"`
	PaymentMethod string    `json:"paymentMethod"`
	PaidAt        time.Time `json:"paidAt"`
}

func (e *InvoicePaidEvent) EventType() string    { return "billing.invoice.paid" }
func (e *InvoicePaidEvent) OccurredAt() time.Time { return e.PaidAt }

// InvoiceOverdueEvent is emitted when an invoice becomes overdue
type InvoiceOverdueEvent struct {
	InvoiceID string    `json:"invoiceId"`
	SellerID  string    `json:"sellerId"`
	Total     float64   `json:"total"`
	DueDate   time.Time `json:"dueDate"`
}

func (e *InvoiceOverdueEvent) EventType() string    { return "billing.invoice.overdue" }
func (e *InvoiceOverdueEvent) OccurredAt() time.Time { return time.Now() }

// ActivityRecordedEvent is emitted when a billable activity is recorded
type ActivityRecordedEvent struct {
	ActivityID    string       `json:"activityId"`
	SellerID      string       `json:"sellerId"`
	Type          ActivityType `json:"type"`
	Amount        float64      `json:"amount"`
	ReferenceType string       `json:"referenceType"`
	ReferenceID   string       `json:"referenceId"`
	RecordedAt    time.Time    `json:"recordedAt"`
}

func (e *ActivityRecordedEvent) EventType() string    { return "billing.activity.recorded" }
func (e *ActivityRecordedEvent) OccurredAt() time.Time { return e.RecordedAt }

// StorageCalculatedEvent is emitted when daily storage fees are calculated
type StorageCalculatedEvent struct {
	CalculationID  string    `json:"calculationId"`
	SellerID       string    `json:"sellerId"`
	FacilityID     string    `json:"facilityId"`
	TotalCubicFeet float64   `json:"totalCubicFeet"`
	TotalAmount    float64   `json:"totalAmount"`
	CalculatedAt   time.Time `json:"calculatedAt"`
}

func (e *StorageCalculatedEvent) EventType() string    { return "billing.storage.calculated" }
func (e *StorageCalculatedEvent) OccurredAt() time.Time { return e.CalculatedAt }
