package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// LedgerEntryAggregate represents a persisted ledger entry with tenant context
// Stored in separate collection for unbounded entry history
type LedgerEntryAggregate struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	Entry      LedgerEntry        `bson:"entry"`

	// Multi-tenant fields for filtering
	TenantID    string `bson:"tenantId"`
	FacilityID  string `bson:"facilityId"`
	WarehouseID string `bson:"warehouseId"`
	SellerID    string `bson:"sellerId,omitempty"`

	DomainEvents []DomainEvent `bson:"-"`
}

// NewLedgerEntryAggregate creates a new ledger entry aggregate
func NewLedgerEntryAggregate(entry LedgerEntry, tenant *LedgerTenantInfo) *LedgerEntryAggregate {
	aggregate := &LedgerEntryAggregate{
		Entry:        entry,
		TenantID:     tenant.TenantID,
		FacilityID:   tenant.FacilityID,
		WarehouseID:  tenant.WarehouseID,
		SellerID:     tenant.SellerID,
		DomainEvents: make([]DomainEvent, 0),
	}

	// Emit LedgerEntryCreatedEvent
	aggregate.addDomainEvent(&LedgerEntryCreatedEvent{
		EntryID:        entry.EntryID.String(),
		TransactionID:  entry.TransactionID.String(),
		SKU:            entry.SKU,
		AccountType:    entry.AccountType.String(),
		DebitAmount:    entry.DebitAmount,
		CreditAmount:   entry.CreditAmount,
		DebitValue:     entry.DebitValue.ToCents(),
		CreditValue:    entry.CreditValue.ToCents(),
		RunningBalance: entry.RunningBalance,
		RunningValue:   entry.RunningValue.ToCents(),
		LocationID:     entry.LocationID,
		ReferenceID:    entry.ReferenceID,
		ReferenceType:  entry.ReferenceType,
		Description:    entry.Description,
		CreatedAt:      entry.CreatedAt,
		CreatedBy:      entry.CreatedBy,
	})

	return aggregate
}

// PullEvents returns and clears pending domain events
func (a *LedgerEntryAggregate) PullEvents() []DomainEvent {
	events := a.DomainEvents
	a.DomainEvents = nil
	return events
}

// addDomainEvent adds a domain event to the pending events
func (a *LedgerEntryAggregate) addDomainEvent(event DomainEvent) {
	a.DomainEvents = append(a.DomainEvents, event)
}

// ClearDomainEvents clears all pending domain events
func (a *LedgerEntryAggregate) ClearDomainEvents() {
	a.DomainEvents = nil
}

// GetSKU returns the SKU from the entry
func (a *LedgerEntryAggregate) GetSKU() string {
	return a.Entry.SKU
}

// GetCreatedAt returns the creation time from the entry
func (a *LedgerEntryAggregate) GetCreatedAt() time.Time {
	return a.Entry.CreatedAt
}

// GetTransactionID returns the transaction ID from the entry
func (a *LedgerEntryAggregate) GetTransactionID() string {
	return a.Entry.TransactionID.String()
}
