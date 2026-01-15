package application

import "time"

// LedgerDTO represents a ledger for API responses
type LedgerDTO struct {
	SKU             string              `json:"sku"`
	TenantID        string              `json:"tenantId"`
	FacilityID      string              `json:"facilityId"`
	WarehouseID     string              `json:"warehouseId"`
	SellerID        string              `json:"sellerId,omitempty"`
	ValuationMethod string              `json:"valuationMethod"`
	CurrentBalance  int                 `json:"currentBalance"`
	CurrentValue    MoneyDTO            `json:"currentValue"`
	AverageUnitCost MoneyDTO            `json:"averageUnitCost"`
	AccountBalances map[string]BalanceDTO `json:"accountBalances"`
	CostLayerCount  int                 `json:"costLayerCount"`
	CreatedAt       time.Time           `json:"createdAt"`
	UpdatedAt       time.Time           `json:"updatedAt"`
}

// MoneyDTO represents money in API responses
type MoneyDTO struct {
	Amount   int64  `json:"amount"`   // In cents
	Currency string `json:"currency"`
}

// BalanceDTO represents an account balance
type BalanceDTO struct {
	Balance int      `json:"balance"`
	Value   MoneyDTO `json:"value"`
}

// LedgerEntryDTO represents a ledger entry for API responses
type LedgerEntryDTO struct {
	EntryID        string    `json:"entryId"`
	TransactionID  string    `json:"transactionId"`
	SKU            string    `json:"sku"`
	AccountType    string    `json:"accountType"`
	DebitAmount    int       `json:"debitAmount"`
	CreditAmount   int       `json:"creditAmount"`
	DebitValue     MoneyDTO  `json:"debitValue"`
	CreditValue    MoneyDTO  `json:"creditValue"`
	RunningBalance int       `json:"runningBalance"`
	RunningValue   MoneyDTO  `json:"runningValue"`
	LocationID     string    `json:"locationId"`
	UnitCost       MoneyDTO  `json:"unitCost"`
	ReferenceID    string    `json:"referenceId"`
	ReferenceType  string    `json:"referenceType"`
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"createdAt"`
	CreatedBy      string    `json:"createdBy"`
}

// LedgerTransactionDTO represents a transaction (debit/credit pair)
type LedgerTransactionDTO struct {
	TransactionID string           `json:"transactionId"`
	Entries       []LedgerEntryDTO `json:"entries"`
	CreatedAt     time.Time        `json:"createdAt"`
}

// ReconciliationResultDTO represents reconciliation result
type ReconciliationResultDTO struct {
	ReconciliationID string    `json:"reconciliationId"`
	SKU              string    `json:"sku"`
	LocationID       string    `json:"locationId"`
	BookBalance      int       `json:"bookBalance"`
	PhysicalCount    int       `json:"physicalCount"`
	Variance         int       `json:"variance"`
	VarianceValue    MoneyDTO  `json:"varianceValue"`
	Status           string    `json:"status"`
	CompletedAt      time.Time `json:"completedAt"`
}
