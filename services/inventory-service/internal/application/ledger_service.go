package application

import (
	"context"
	"fmt"

	"github.com/wms-platform/inventory-service/internal/domain"
)

// LedgerApplicationService handles inventory ledger use cases
type LedgerApplicationService struct {
	ledgerRepo      domain.InventoryLedgerRepository
	entryRepo       domain.LedgerEntryRepository
}

// NewLedgerApplicationService creates a new ledger application service
func NewLedgerApplicationService(
	ledgerRepo domain.InventoryLedgerRepository,
	entryRepo domain.LedgerEntryRepository,
) *LedgerApplicationService {
	return &LedgerApplicationService{
		ledgerRepo:      ledgerRepo,
		entryRepo:       entryRepo,
	}
}

// CreateLedger creates a new inventory ledger
func (s *LedgerApplicationService) CreateLedger(ctx context.Context, cmd CreateLedgerCommand) (*LedgerDTO, error) {
	// Parse valuation method
	var valuationMethod domain.ValuationMethod
	switch cmd.ValuationMethod {
	case "FIFO":
		valuationMethod = domain.ValuationFIFO
	case "LIFO":
		valuationMethod = domain.ValuationLIFO
	case "WEIGHTED_AVERAGE":
		valuationMethod = domain.ValuationWeightedAverage
	default:
		valuationMethod = domain.DefaultValuationMethod
	}

	// Create tenant info
	tenantInfo := &domain.LedgerTenantInfo{
		TenantID:    cmd.TenantID,
		FacilityID:  cmd.FacilityID,
		WarehouseID: cmd.WarehouseID,
		SellerID:    cmd.SellerID,
	}

	// Create ledger
	ledger, err := domain.NewInventoryLedger(cmd.SKU, valuationMethod, tenantInfo, cmd.Currency)
	if err != nil {
		return nil, fmt.Errorf("failed to create ledger: %w", err)
	}

	// Save ledger
	if err := s.ledgerRepo.Save(ctx, ledger); err != nil {
		return nil, fmt.Errorf("failed to save ledger: %w", err)
	}

	return toLedgerDTO(ledger), nil
}

// RecordReceiving records stock receiving in ledger
func (s *LedgerApplicationService) RecordReceiving(ctx context.Context, cmd RecordReceivingCommand) (string, error) {
	// Get or create ledger
	ledger, err := s.getOrCreateLedger(ctx, cmd.SKU, cmd.TenantID, cmd.FacilityID, cmd.WarehouseID, cmd.SellerID, cmd.Currency)
	if err != nil {
		return "", fmt.Errorf("failed to get ledger: %w", err)
	}

	// Create money for unit cost
	unitCost, err := domain.NewMoney(cmd.UnitCost, cmd.Currency)
	if err != nil {
		return "", fmt.Errorf("invalid unit cost: %w", err)
	}

	// Record receiving
	transactionID, entries, err := ledger.RecordReceiving(cmd.Quantity, unitCost, cmd.LocationID, cmd.ReferenceID, cmd.CreatedBy)
	if err != nil {
		return "", fmt.Errorf("failed to record receiving: %w", err)
	}

	// Save ledger
	if err := s.ledgerRepo.Save(ctx, ledger); err != nil {
		return "", fmt.Errorf("failed to save ledger: %w", err)
	}

	// Save entries
	if err := s.saveEntries(ctx, entries, ledger.TenantID, ledger.FacilityID, ledger.WarehouseID, ledger.SellerID); err != nil {
		return "", fmt.Errorf("failed to save entries: %w", err)
	}

	return transactionID.String(), nil
}

// RecordPick records picking in ledger
func (s *LedgerApplicationService) RecordPick(ctx context.Context, cmd RecordPickCommand) (string, error) {
	// Get ledger
	ledger, err := s.ledgerRepo.FindBySKU(ctx, cmd.TenantID, cmd.FacilityID, cmd.SKU)
	if err != nil {
		return "", fmt.Errorf("failed to get ledger: %w", err)
	}

	// Record pick
	transactionID, entries, err := ledger.RecordPick(cmd.Quantity, cmd.LocationID, cmd.OrderID, cmd.CreatedBy)
	if err != nil {
		return "", fmt.Errorf("failed to record pick: %w", err)
	}

	// Save ledger
	if err := s.ledgerRepo.Save(ctx, ledger); err != nil {
		return "", fmt.Errorf("failed to save ledger: %w", err)
	}

	// Save entries
	if err := s.saveEntries(ctx, entries, ledger.TenantID, ledger.FacilityID, ledger.WarehouseID, ledger.SellerID); err != nil {
		return "", fmt.Errorf("failed to save entries: %w", err)
	}

	return transactionID.String(), nil
}

// RecordAdjustment records inventory adjustment in ledger
func (s *LedgerApplicationService) RecordAdjustment(ctx context.Context, cmd RecordAdjustmentCommand) (string, error) {
	// Get ledger
	ledger, err := s.ledgerRepo.FindBySKU(ctx, cmd.TenantID, cmd.FacilityID, cmd.SKU)
	if err != nil {
		return "", fmt.Errorf("failed to get ledger: %w", err)
	}

	// Record adjustment
	transactionID, entries, err := ledger.RecordAdjustment(cmd.Quantity, cmd.Reason, cmd.LocationID, cmd.ReferenceID, cmd.CreatedBy)
	if err != nil {
		return "", fmt.Errorf("failed to record adjustment: %w", err)
	}

	// Save ledger
	if err := s.ledgerRepo.Save(ctx, ledger); err != nil {
		return "", fmt.Errorf("failed to save ledger: %w", err)
	}

	// Save entries
	if err := s.saveEntries(ctx, entries, ledger.TenantID, ledger.FacilityID, ledger.WarehouseID, ledger.SellerID); err != nil {
		return "", fmt.Errorf("failed to save entries: %w", err)
	}

	return transactionID.String(), nil
}

// GetLedger retrieves a ledger by SKU
func (s *LedgerApplicationService) GetLedger(ctx context.Context, query GetLedgerQuery) (*LedgerDTO, error) {
	ledger, err := s.ledgerRepo.FindBySKU(ctx, query.TenantID, query.FacilityID, query.SKU)
	if err != nil {
		return nil, fmt.Errorf("failed to find ledger: %w", err)
	}

	return toLedgerDTO(ledger), nil
}

// GetLedgerEntries retrieves ledger entries for a SKU
func (s *LedgerApplicationService) GetLedgerEntries(ctx context.Context, query GetLedgerEntriesQuery) ([]LedgerEntryDTO, error) {
	entries, err := s.entryRepo.FindBySKU(ctx, query.TenantID, query.FacilityID, query.SKU, query.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to find entries: %w", err)
	}

	return toLedgerEntryDTOs(entries), nil
}

// GetTransactionEntries retrieves entries for a transaction (debit/credit pair)
func (s *LedgerApplicationService) GetTransactionEntries(ctx context.Context, query GetLedgerByTransactionQuery) (*LedgerTransactionDTO, error) {
	entries, err := s.entryRepo.FindByTransactionID(ctx, query.TenantID, query.TransactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find entries: %w", err)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("transaction not found")
	}

	return toLedgerTransactionDTO(entries), nil
}

// getOrCreateLedger gets existing ledger or creates a new one
func (s *LedgerApplicationService) getOrCreateLedger(ctx context.Context, sku, tenantID, facilityID, warehouseID, sellerID, currency string) (*domain.InventoryLedger, error) {
	ledger, err := s.ledgerRepo.FindBySKU(ctx, tenantID, facilityID, sku)
	if err == nil {
		return ledger, nil
	}

	if err != domain.ErrLedgerNotFound {
		return nil, err
	}

	// Create new ledger with default FIFO valuation
	tenantInfo := &domain.LedgerTenantInfo{
		TenantID:    tenantID,
		FacilityID:  facilityID,
		WarehouseID: warehouseID,
		SellerID:    sellerID,
	}

	ledger, err = domain.NewInventoryLedger(sku, domain.DefaultValuationMethod, tenantInfo, currency)
	if err != nil {
		return nil, err
	}

	if err := s.ledgerRepo.Save(ctx, ledger); err != nil {
		return nil, err
	}

	return ledger, nil
}

// saveEntries saves ledger entries to repository
func (s *LedgerApplicationService) saveEntries(ctx context.Context, entries []domain.LedgerEntry, tenantID, facilityID, warehouseID, sellerID string) error {
	if len(entries) == 0 {
		return nil
	}

	tenantInfo := &domain.LedgerTenantInfo{
		TenantID:    tenantID,
		FacilityID:  facilityID,
		WarehouseID: warehouseID,
		SellerID:    sellerID,
	}

	aggregates := make([]*domain.LedgerEntryAggregate, len(entries))
	for i, entry := range entries {
		aggregates[i] = domain.NewLedgerEntryAggregate(entry, tenantInfo)
	}

	return s.entryRepo.SaveAll(ctx, aggregates)
}

// Mapper functions
func toLedgerDTO(ledger *domain.InventoryLedger) *LedgerDTO {
	accountBalances := make(map[string]BalanceDTO)
	for accountType, balance := range ledger.AccountBalances {
		accountBalances[accountType.String()] = BalanceDTO{
			Balance: balance.Balance,
			Value: MoneyDTO{
				Amount:   balance.Value.ToCents(),
				Currency: balance.Value.Currency(),
			},
		}
	}

	return &LedgerDTO{
		SKU:             ledger.SKU,
		TenantID:        ledger.TenantID,
		FacilityID:      ledger.FacilityID,
		WarehouseID:     ledger.WarehouseID,
		SellerID:        ledger.SellerID,
		ValuationMethod: ledger.ValuationMethod.String(),
		CurrentBalance:  ledger.CurrentBalance,
		CurrentValue: MoneyDTO{
			Amount:   ledger.CurrentValue.ToCents(),
			Currency: ledger.CurrentValue.Currency(),
		},
		AverageUnitCost: MoneyDTO{
			Amount:   ledger.AverageUnitCost.ToCents(),
			Currency: ledger.AverageUnitCost.Currency(),
		},
		AccountBalances: accountBalances,
		CostLayerCount:  len(ledger.CostLayers),
		CreatedAt:       ledger.CreatedAt,
		UpdatedAt:       ledger.UpdatedAt,
	}
}

func toLedgerEntryDTOs(aggregates []*domain.LedgerEntryAggregate) []LedgerEntryDTO {
	dtos := make([]LedgerEntryDTO, len(aggregates))
	for i, agg := range aggregates {
		dtos[i] = toLedgerEntryDTO(&agg.Entry)
	}
	return dtos
}

func toLedgerEntryDTO(entry *domain.LedgerEntry) LedgerEntryDTO {
	return LedgerEntryDTO{
		EntryID:       entry.EntryID.String(),
		TransactionID: entry.TransactionID.String(),
		SKU:           entry.SKU,
		AccountType:   entry.AccountType.String(),
		DebitAmount:   entry.DebitAmount,
		CreditAmount:  entry.CreditAmount,
		DebitValue: MoneyDTO{
			Amount:   entry.DebitValue.ToCents(),
			Currency: entry.DebitValue.Currency(),
		},
		CreditValue: MoneyDTO{
			Amount:   entry.CreditValue.ToCents(),
			Currency: entry.CreditValue.Currency(),
		},
		RunningBalance: entry.RunningBalance,
		RunningValue: MoneyDTO{
			Amount:   entry.RunningValue.ToCents(),
			Currency: entry.RunningValue.Currency(),
		},
		LocationID:    entry.LocationID,
		UnitCost: MoneyDTO{
			Amount:   entry.UnitCost.ToCents(),
			Currency: entry.UnitCost.Currency(),
		},
		ReferenceID:   entry.ReferenceID,
		ReferenceType: entry.ReferenceType,
		Description:   entry.Description,
		CreatedAt:     entry.CreatedAt,
		CreatedBy:     entry.CreatedBy,
	}
}

func toLedgerTransactionDTO(aggregates []*domain.LedgerEntryAggregate) *LedgerTransactionDTO {
	if len(aggregates) == 0 {
		return nil
	}

	entries := make([]LedgerEntryDTO, len(aggregates))
	for i, agg := range aggregates {
		entries[i] = toLedgerEntryDTO(&agg.Entry)
	}

	return &LedgerTransactionDTO{
		TransactionID: aggregates[0].Entry.TransactionID.String(),
		Entries:       entries,
		CreatedAt:     aggregates[0].Entry.CreatedAt,
	}
}
