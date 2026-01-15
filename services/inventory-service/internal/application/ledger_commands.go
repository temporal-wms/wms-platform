package application

// RecordReceivingCommand represents the command to record stock receiving in ledger
type RecordReceivingCommand struct {
	SKU         string
	Quantity    int
	UnitCost    int64  // In cents
	Currency    string // ISO 4217 code
	LocationID  string
	ReferenceID string // PO ID
	CreatedBy   string
	TenantID    string
	FacilityID  string
	WarehouseID string
	SellerID    string
}

// RecordPickCommand represents the command to record picking in ledger
type RecordPickCommand struct {
	SKU         string
	Quantity    int
	LocationID  string
	OrderID     string
	CreatedBy   string
	TenantID    string
	FacilityID  string
	WarehouseID string
	SellerID    string
}

// RecordAdjustmentCommand represents the command to record inventory adjustment in ledger
type RecordAdjustmentCommand struct {
	SKU         string
	Quantity    int    // Can be positive or negative
	Reason      string
	LocationID  string
	ReferenceID string
	CreatedBy   string
	TenantID    string
	FacilityID  string
	WarehouseID string
	SellerID    string
}

// CreateLedgerCommand represents the command to create a new inventory ledger
type CreateLedgerCommand struct {
	SKU             string
	ValuationMethod string // FIFO, LIFO, WEIGHTED_AVERAGE
	Currency        string // ISO 4217 code
	TenantID        string
	FacilityID      string
	WarehouseID     string
	SellerID        string
}

// GetLedgerQuery represents the query to get a ledger by SKU
type GetLedgerQuery struct {
	SKU        string
	TenantID   string
	FacilityID string
}

// GetLedgerEntriesQuery represents the query to get ledger entries
type GetLedgerEntriesQuery struct {
	SKU        string
	TenantID   string
	FacilityID string
	Limit      int
}

// GetLedgerByTransactionQuery represents the query to get entries by transaction ID
type GetLedgerByTransactionQuery struct {
	TransactionID string
	TenantID      string
}
