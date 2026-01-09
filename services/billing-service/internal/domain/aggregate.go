package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Errors for Billing domain
var (
	ErrInvalidActivityType   = errors.New("invalid activity type")
	ErrInvalidAmount         = errors.New("amount must be positive")
	ErrInvoiceAlreadyFinalized = errors.New("invoice is already finalized")
	ErrInvoiceNotFinalized   = errors.New("invoice must be finalized before payment")
	ErrInvalidPayment        = errors.New("payment amount does not match invoice total")
)

// ActivityType represents the type of billable activity
type ActivityType string

const (
	ActivityTypeStorage          ActivityType = "storage"
	ActivityTypePick             ActivityType = "pick"
	ActivityTypePack             ActivityType = "pack"
	ActivityTypeReceiving        ActivityType = "receiving"
	ActivityTypeShipping         ActivityType = "shipping"
	ActivityTypeReturnProcessing ActivityType = "return_processing"
	ActivityTypeGiftWrap         ActivityType = "gift_wrap"
	ActivityTypeHazmat           ActivityType = "hazmat"
	ActivityTypeOversized        ActivityType = "oversized"
	ActivityTypeColdChain        ActivityType = "cold_chain"
	ActivityTypeFragile          ActivityType = "fragile"
	ActivityTypeSpecialHandling  ActivityType = "special_handling"
)

// IsValid checks if the activity type is valid
func (a ActivityType) IsValid() bool {
	switch a {
	case ActivityTypeStorage, ActivityTypePick, ActivityTypePack, ActivityTypeReceiving,
		ActivityTypeShipping, ActivityTypeReturnProcessing, ActivityTypeGiftWrap,
		ActivityTypeHazmat, ActivityTypeOversized, ActivityTypeColdChain,
		ActivityTypeFragile, ActivityTypeSpecialHandling:
		return true
	}
	return false
}

// InvoiceStatus represents the status of an invoice
type InvoiceStatus string

const (
	InvoiceStatusDraft     InvoiceStatus = "draft"
	InvoiceStatusFinalized InvoiceStatus = "finalized"
	InvoiceStatusPaid      InvoiceStatus = "paid"
	InvoiceStatusOverdue   InvoiceStatus = "overdue"
	InvoiceStatusVoided    InvoiceStatus = "voided"
)

// BillableActivity represents a single billable event
type BillableActivity struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ActivityID  string             `bson:"activityId" json:"activityId"`
	TenantID    string             `bson:"tenantId" json:"tenantId"`
	SellerID    string             `bson:"sellerId" json:"sellerId"`
	FacilityID  string             `bson:"facilityId" json:"facilityId"`

	// Activity details
	Type        ActivityType `bson:"type" json:"type"`
	Description string       `bson:"description" json:"description"`
	Quantity    float64      `bson:"quantity" json:"quantity"`
	UnitPrice   float64      `bson:"unitPrice" json:"unitPrice"`
	Amount      float64      `bson:"amount" json:"amount"`
	Currency    string       `bson:"currency" json:"currency"`

	// Reference to the source entity
	ReferenceType string `bson:"referenceType" json:"referenceType"` // order, inventory, shipment
	ReferenceID   string `bson:"referenceId" json:"referenceId"`

	// Billing period
	ActivityDate time.Time `bson:"activityDate" json:"activityDate"`
	BillingDate  time.Time `bson:"billingDate" json:"billingDate"`

	// Invoice association
	InvoiceID *string `bson:"invoiceId,omitempty" json:"invoiceId,omitempty"`
	Invoiced  bool    `bson:"invoiced" json:"invoiced"`

	// Metadata
	Metadata  map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
	CreatedAt time.Time              `bson:"createdAt" json:"createdAt"`
}

// NewBillableActivity creates a new billable activity
func NewBillableActivity(
	tenantID, sellerID, facilityID string,
	activityType ActivityType,
	description string,
	quantity, unitPrice float64,
	referenceType, referenceID string,
) (*BillableActivity, error) {
	if !activityType.IsValid() {
		return nil, ErrInvalidActivityType
	}

	amount := quantity * unitPrice
	if amount < 0 {
		return nil, ErrInvalidAmount
	}

	now := time.Now().UTC()
	activityID := fmt.Sprintf("ACT-%s", uuid.New().String()[:8])

	return &BillableActivity{
		ID:            primitive.NewObjectID(),
		ActivityID:    activityID,
		TenantID:      tenantID,
		SellerID:      sellerID,
		FacilityID:    facilityID,
		Type:          activityType,
		Description:   description,
		Quantity:      quantity,
		UnitPrice:     unitPrice,
		Amount:        amount,
		Currency:      "USD",
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
		ActivityDate:  now,
		BillingDate:   now,
		Invoiced:      false,
		CreatedAt:     now,
	}, nil
}

// Invoice represents an aggregated billing document
type Invoice struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	InvoiceID  string             `bson:"invoiceId" json:"invoiceId"`
	TenantID   string             `bson:"tenantId" json:"tenantId"`
	SellerID   string             `bson:"sellerId" json:"sellerId"`
	FacilityID string             `bson:"facilityId,omitempty" json:"facilityId,omitempty"`

	// Invoice details
	Status       InvoiceStatus `bson:"status" json:"status"`
	InvoiceNumber string       `bson:"invoiceNumber" json:"invoiceNumber"`

	// Billing period
	PeriodStart time.Time `bson:"periodStart" json:"periodStart"`
	PeriodEnd   time.Time `bson:"periodEnd" json:"periodEnd"`

	// Line items (summary by activity type)
	LineItems []InvoiceLineItem `bson:"lineItems" json:"lineItems"`

	// Totals
	Subtotal    float64 `bson:"subtotal" json:"subtotal"`
	TaxRate     float64 `bson:"taxRate" json:"taxRate"`
	TaxAmount   float64 `bson:"taxAmount" json:"taxAmount"`
	Discount    float64 `bson:"discount" json:"discount"`
	Total       float64 `bson:"total" json:"total"`
	Currency    string  `bson:"currency" json:"currency"`

	// Payment info
	DueDate       time.Time  `bson:"dueDate" json:"dueDate"`
	PaidAt        *time.Time `bson:"paidAt,omitempty" json:"paidAt,omitempty"`
	PaymentMethod string     `bson:"paymentMethod,omitempty" json:"paymentMethod,omitempty"`
	PaymentRef    string     `bson:"paymentRef,omitempty" json:"paymentRef,omitempty"`

	// Seller info snapshot (for historical records)
	SellerName    string `bson:"sellerName" json:"sellerName"`
	SellerEmail   string `bson:"sellerEmail" json:"sellerEmail"`
	SellerAddress string `bson:"sellerAddress,omitempty" json:"sellerAddress,omitempty"`

	// Notes
	Notes string `bson:"notes,omitempty" json:"notes,omitempty"`

	// Timestamps
	CreatedAt   time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time  `bson:"updatedAt" json:"updatedAt"`
	FinalizedAt *time.Time `bson:"finalizedAt,omitempty" json:"finalizedAt,omitempty"`

	// Domain events
	domainEvents []DomainEvent `bson:"-" json:"-"`
}

// InvoiceLineItem represents a line item on an invoice
type InvoiceLineItem struct {
	ActivityType ActivityType `bson:"activityType" json:"activityType"`
	Description  string       `bson:"description" json:"description"`
	Quantity     float64      `bson:"quantity" json:"quantity"`
	UnitPrice    float64      `bson:"unitPrice" json:"unitPrice"`
	Amount       float64      `bson:"amount" json:"amount"`
	ActivityIDs  []string     `bson:"activityIds" json:"activityIds"` // References to BillableActivity
}

// NewInvoice creates a new invoice for a billing period
func NewInvoice(
	tenantID, sellerID string,
	periodStart, periodEnd time.Time,
	sellerName, sellerEmail string,
) *Invoice {
	now := time.Now().UTC()
	invoiceID := fmt.Sprintf("INV-%s", uuid.New().String()[:8])
	invoiceNumber := fmt.Sprintf("INV-%s-%s", time.Now().Format("200601"), uuid.New().String()[:6])

	// Due date is 30 days from now by default
	dueDate := now.AddDate(0, 0, 30)

	invoice := &Invoice{
		ID:            primitive.NewObjectID(),
		InvoiceID:     invoiceID,
		TenantID:      tenantID,
		SellerID:      sellerID,
		Status:        InvoiceStatusDraft,
		InvoiceNumber: invoiceNumber,
		PeriodStart:   periodStart,
		PeriodEnd:     periodEnd,
		LineItems:     make([]InvoiceLineItem, 0),
		Subtotal:      0,
		TaxRate:       0,
		TaxAmount:     0,
		Discount:      0,
		Total:         0,
		Currency:      "USD",
		DueDate:       dueDate,
		SellerName:    sellerName,
		SellerEmail:   sellerEmail,
		CreatedAt:     now,
		UpdatedAt:     now,
		domainEvents:  make([]DomainEvent, 0),
	}

	invoice.addDomainEvent(&InvoiceCreatedEvent{
		InvoiceID:   invoiceID,
		SellerID:    sellerID,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		CreatedAt:   now,
	})

	return invoice
}

// AddLineItem adds a line item to the invoice
func (i *Invoice) AddLineItem(activityType ActivityType, description string, quantity, unitPrice float64, activityIDs []string) error {
	if i.Status != InvoiceStatusDraft {
		return ErrInvoiceAlreadyFinalized
	}

	amount := quantity * unitPrice

	// Check if we already have a line item for this activity type
	for idx, item := range i.LineItems {
		if item.ActivityType == activityType {
			// Aggregate into existing line item
			i.LineItems[idx].Quantity += quantity
			i.LineItems[idx].Amount += amount
			i.LineItems[idx].ActivityIDs = append(i.LineItems[idx].ActivityIDs, activityIDs...)
			i.recalculateTotals()
			return nil
		}
	}

	// Add new line item
	i.LineItems = append(i.LineItems, InvoiceLineItem{
		ActivityType: activityType,
		Description:  description,
		Quantity:     quantity,
		UnitPrice:    unitPrice,
		Amount:       amount,
		ActivityIDs:  activityIDs,
	})

	i.recalculateTotals()
	return nil
}

// recalculateTotals recalculates subtotal, tax, and total
func (i *Invoice) recalculateTotals() {
	i.Subtotal = 0
	for _, item := range i.LineItems {
		i.Subtotal += item.Amount
	}

	i.TaxAmount = i.Subtotal * i.TaxRate
	i.Total = i.Subtotal + i.TaxAmount - i.Discount
	i.UpdatedAt = time.Now().UTC()
}

// SetTaxRate sets the tax rate and recalculates totals
func (i *Invoice) SetTaxRate(rate float64) error {
	if i.Status != InvoiceStatusDraft {
		return ErrInvoiceAlreadyFinalized
	}
	i.TaxRate = rate
	i.recalculateTotals()
	return nil
}

// ApplyDiscount applies a discount and recalculates totals
func (i *Invoice) ApplyDiscount(discount float64) error {
	if i.Status != InvoiceStatusDraft {
		return ErrInvoiceAlreadyFinalized
	}
	i.Discount = discount
	i.recalculateTotals()
	return nil
}

// Finalize marks the invoice as finalized and ready for payment
func (i *Invoice) Finalize() error {
	if i.Status != InvoiceStatusDraft {
		return ErrInvoiceAlreadyFinalized
	}

	now := time.Now().UTC()
	i.Status = InvoiceStatusFinalized
	i.FinalizedAt = &now
	i.UpdatedAt = now

	i.addDomainEvent(&InvoiceFinalizedEvent{
		InvoiceID:   i.InvoiceID,
		SellerID:    i.SellerID,
		Total:       i.Total,
		DueDate:     i.DueDate,
		FinalizedAt: now,
	})

	return nil
}

// MarkPaid marks the invoice as paid
func (i *Invoice) MarkPaid(paymentMethod, paymentRef string) error {
	if i.Status != InvoiceStatusFinalized && i.Status != InvoiceStatusOverdue {
		return ErrInvoiceNotFinalized
	}

	now := time.Now().UTC()
	i.Status = InvoiceStatusPaid
	i.PaidAt = &now
	i.PaymentMethod = paymentMethod
	i.PaymentRef = paymentRef
	i.UpdatedAt = now

	i.addDomainEvent(&InvoicePaidEvent{
		InvoiceID:     i.InvoiceID,
		SellerID:      i.SellerID,
		Amount:        i.Total,
		PaymentMethod: paymentMethod,
		PaidAt:        now,
	})

	return nil
}

// MarkOverdue marks the invoice as overdue
func (i *Invoice) MarkOverdue() {
	if i.Status == InvoiceStatusFinalized && time.Now().After(i.DueDate) {
		i.Status = InvoiceStatusOverdue
		i.UpdatedAt = time.Now().UTC()

		i.addDomainEvent(&InvoiceOverdueEvent{
			InvoiceID: i.InvoiceID,
			SellerID:  i.SellerID,
			Total:     i.Total,
			DueDate:   i.DueDate,
		})
	}
}

// Void voids the invoice
func (i *Invoice) Void(reason string) error {
	if i.Status == InvoiceStatusPaid {
		return errors.New("cannot void a paid invoice")
	}

	i.Status = InvoiceStatusVoided
	i.Notes = reason
	i.UpdatedAt = time.Now().UTC()
	return nil
}

// Domain event helpers
func (i *Invoice) addDomainEvent(event DomainEvent) {
	i.domainEvents = append(i.domainEvents, event)
}

// DomainEvents returns all pending domain events
func (i *Invoice) DomainEvents() []DomainEvent {
	return i.domainEvents
}

// ClearDomainEvents clears all pending domain events
func (i *Invoice) ClearDomainEvents() {
	i.domainEvents = make([]DomainEvent, 0)
}

// StorageCalculation represents daily storage fee calculation
type StorageCalculation struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CalculationID string            `bson:"calculationId" json:"calculationId"`
	TenantID     string             `bson:"tenantId" json:"tenantId"`
	SellerID     string             `bson:"sellerId" json:"sellerId"`
	FacilityID   string             `bson:"facilityId" json:"facilityId"`

	// Calculation details
	CalculationDate time.Time `bson:"calculationDate" json:"calculationDate"`
	TotalCubicFeet  float64   `bson:"totalCubicFeet" json:"totalCubicFeet"`
	RatePerCubicFt  float64   `bson:"ratePerCubicFt" json:"ratePerCubicFt"`
	TotalAmount     float64   `bson:"totalAmount" json:"totalAmount"`

	// Breakdown by storage type
	StandardStorage   float64 `bson:"standardStorage" json:"standardStorage"`
	OversizedStorage  float64 `bson:"oversizedStorage" json:"oversizedStorage"`
	HazmatStorage     float64 `bson:"hazmatStorage" json:"hazmatStorage"`
	ColdChainStorage  float64 `bson:"coldChainStorage" json:"coldChainStorage"`

	// Activity reference
	ActivityID string `bson:"activityId,omitempty" json:"activityId,omitempty"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}

// NewStorageCalculation creates a new storage calculation
func NewStorageCalculation(
	tenantID, sellerID, facilityID string,
	calculationDate time.Time,
	totalCubicFeet, ratePerCubicFt float64,
) *StorageCalculation {
	return &StorageCalculation{
		ID:              primitive.NewObjectID(),
		CalculationID:   fmt.Sprintf("STO-%s", uuid.New().String()[:8]),
		TenantID:        tenantID,
		SellerID:        sellerID,
		FacilityID:      facilityID,
		CalculationDate: calculationDate,
		TotalCubicFeet:  totalCubicFeet,
		RatePerCubicFt:  ratePerCubicFt,
		TotalAmount:     totalCubicFeet * ratePerCubicFt,
		CreatedAt:       time.Now().UTC(),
	}
}
