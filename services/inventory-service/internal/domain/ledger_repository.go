package domain

import (
	"context"
	"time"
)

// InventoryLedgerRepository defines the port for ledger persistence
type InventoryLedgerRepository interface {
	// Save persists the inventory ledger
	Save(ctx context.Context, ledger *InventoryLedger) error

	// FindBySKU retrieves a ledger by SKU and tenant
	FindBySKU(ctx context.Context, tenantID, facilityID, sku string) (*InventoryLedger, error)

	// FindByLocation retrieves all ledgers for a location
	FindByLocation(ctx context.Context, tenantID, facilityID, locationID string) ([]*InventoryLedger, error)

	// FindAll retrieves all ledgers with pagination
	FindAll(ctx context.Context, tenantID, facilityID string, limit, offset int) ([]*InventoryLedger, error)

	// Delete removes a ledger
	Delete(ctx context.Context, tenantID, facilityID, sku string) error
}

// LedgerEntryRepository defines the port for ledger entry persistence (for history)
type LedgerEntryRepository interface {
	// Save persists a single ledger entry
	Save(ctx context.Context, entry *LedgerEntryAggregate) error

	// SaveAll persists multiple ledger entries atomically
	SaveAll(ctx context.Context, entries []*LedgerEntryAggregate) error

	// FindBySKU retrieves entries for a SKU with pagination
	FindBySKU(ctx context.Context, tenantID, facilityID, sku string, limit int) ([]*LedgerEntryAggregate, error)

	// FindByTransactionID retrieves entries for a transaction (debit/credit pair)
	FindByTransactionID(ctx context.Context, tenantID, transactionID string) ([]*LedgerEntryAggregate, error)

	// FindByTimeRange retrieves entries within a time range
	FindByTimeRange(ctx context.Context, tenantID, facilityID, sku string, start, end time.Time) ([]*LedgerEntryAggregate, error)

	// FindByAccountType retrieves entries for a specific account type
	FindByAccountType(ctx context.Context, tenantID, facilityID, sku string, accountType AccountType, limit int) ([]*LedgerEntryAggregate, error)

	// GetBalanceAtTime calculates the running balance at a specific time
	GetBalanceAtTime(ctx context.Context, tenantID, facilityID, sku string, timestamp time.Time) (int, Money, error)
}
