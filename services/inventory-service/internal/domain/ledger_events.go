package domain

import "time"

// LedgerEntryCreatedEvent is published when a ledger entry is created
type LedgerEntryCreatedEvent struct {
	EntryID        string    `json:"entryId"`
	TransactionID  string    `json:"transactionId"`
	SKU            string    `json:"sku"`
	AccountType    string    `json:"accountType"`
	DebitAmount    int       `json:"debitAmount"`
	CreditAmount   int       `json:"creditAmount"`
	DebitValue     int64     `json:"debitValue"`     // In cents
	CreditValue    int64     `json:"creditValue"`    // In cents
	RunningBalance int       `json:"runningBalance"` // Balance after this entry
	RunningValue   int64     `json:"runningValue"`   // Value after this entry (cents)
	LocationID     string    `json:"locationId"`
	ReferenceID    string    `json:"referenceId"`
	ReferenceType  string    `json:"referenceType"`
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"createdAt"`
	CreatedBy      string    `json:"createdBy"`
}

func (e *LedgerEntryCreatedEvent) EventType() string {
	return "wms.inventory.ledger-entry-created"
}

func (e *LedgerEntryCreatedEvent) OccurredAt() time.Time {
	return e.CreatedAt
}

// InventoryValuedEvent is published when inventory value changes
type InventoryValuedEvent struct {
	SKU             string    `json:"sku"`
	ValuationMethod string    `json:"valuationMethod"`
	TotalQuantity   int       `json:"totalQuantity"`
	TotalValue      int64     `json:"totalValue"`      // In cents
	AverageUnitCost int64     `json:"averageUnitCost"` // In cents
	Currency        string    `json:"currency"`
	CostLayerCount  int       `json:"costLayerCount"`
	ValuedAt        time.Time `json:"valuedAt"`
}

func (e *InventoryValuedEvent) EventType() string {
	return "wms.inventory.valued"
}

func (e *InventoryValuedEvent) OccurredAt() time.Time {
	return e.ValuedAt
}

// ReconciliationCompletedEvent is published after book-to-physical reconciliation
type ReconciliationCompletedEvent struct {
	ReconciliationID string    `json:"reconciliationId"`
	SKU              string    `json:"sku"`
	LocationID       string    `json:"locationId"`
	BookBalance      int       `json:"bookBalance"`
	PhysicalCount    int       `json:"physicalCount"`
	Variance         int       `json:"variance"`
	VarianceValue    int64     `json:"varianceValue"` // In cents
	Status           string    `json:"status"`        // "matched", "variance_recorded", "adjustment_required"
	CompletedAt      time.Time `json:"completedAt"`
	CompletedBy      string    `json:"completedBy"`
}

func (e *ReconciliationCompletedEvent) EventType() string {
	return "wms.inventory.reconciliation-completed"
}

func (e *ReconciliationCompletedEvent) OccurredAt() time.Time {
	return e.CompletedAt
}
