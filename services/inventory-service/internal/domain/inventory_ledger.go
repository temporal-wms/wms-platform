package domain

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// InventoryLedger is the aggregate root for double-entry inventory ledger
type InventoryLedger struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	SKU        string             `bson:"sku"`
	TenantID   string             `bson:"tenantId"`
	FacilityID string             `bson:"facilityId"`
	WarehouseID string            `bson:"warehouseId"`
	SellerID   string             `bson:"sellerId,omitempty"`

	ValuationMethod ValuationMethod        `bson:"valuationMethod"`
	CostLayers      []CostLayer            `bson:"costLayers"`

	// Denormalized current state for fast reads
	CurrentBalance  int                        `bson:"currentBalance"`
	CurrentValue    Money                      `bson:"currentValue"`
	AverageUnitCost Money                      `bson:"averageUnitCost"`
	AccountBalances map[AccountType]AccountBalance `bson:"accountBalances"`

	CreatedAt    time.Time     `bson:"createdAt"`
	UpdatedAt    time.Time     `bson:"updatedAt"`
	DomainEvents []DomainEvent `bson:"-"`
}

// LedgerTenantInfo holds tenant context for ledger creation
type LedgerTenantInfo struct {
	TenantID    string
	FacilityID  string
	WarehouseID string
	SellerID    string
}

// NewInventoryLedger creates a new inventory ledger
func NewInventoryLedger(sku string, valuationMethod ValuationMethod, tenant *LedgerTenantInfo, currency string) (*InventoryLedger, error) {
	if sku == "" {
		return nil, fmt.Errorf("SKU is required")
	}

	if !valuationMethod.IsValid() {
		return nil, ErrInvalidValuationMethod
	}

	if tenant == nil {
		return nil, fmt.Errorf("tenant info is required")
	}

	now := time.Now().UTC()

	ledger := &InventoryLedger{
		SKU:             sku,
		TenantID:        tenant.TenantID,
		FacilityID:      tenant.FacilityID,
		WarehouseID:     tenant.WarehouseID,
		SellerID:        tenant.SellerID,
		ValuationMethod: valuationMethod,
		CostLayers:      make([]CostLayer, 0),
		CurrentBalance:  0,
		CurrentValue:    ZeroMoney(currency),
		AverageUnitCost: ZeroMoney(currency),
		AccountBalances: make(map[AccountType]AccountBalance),
		CreatedAt:       now,
		UpdatedAt:       now,
		DomainEvents:    make([]DomainEvent, 0),
	}

	// Initialize all account balances to zero
	ledger.AccountBalances[AccountInventory] = ZeroAccountBalance(currency)
	ledger.AccountBalances[AccountCOGS] = ZeroAccountBalance(currency)
	ledger.AccountBalances[AccountGoodsInTransit] = ZeroAccountBalance(currency)
	ledger.AccountBalances[AccountAdjustments] = ZeroAccountBalance(currency)
	ledger.AccountBalances[AccountReturns] = ZeroAccountBalance(currency)

	return ledger, nil
}

// RecordReceiving records stock received
// Debit INVENTORY, Credit GOODS_IN_TRANSIT
func (l *InventoryLedger) RecordReceiving(qty int, unitCost Money, locationID, referenceID, createdBy string) (LedgerTransactionID, []LedgerEntry, error) {
	if qty <= 0 {
		return LedgerTransactionID{}, nil, ErrInvalidQuantity
	}

	if unitCost.IsZero() {
		return LedgerTransactionID{}, nil, ErrZeroUnitCost
	}

	transactionID := NewLedgerTransactionID()

	// Add cost layer for FIFO/LIFO
	if l.ValuationMethod.UsesLayers() {
		l.AddCostLayer(qty, unitCost, referenceID)
	}

	// Calculate new balance and value
	newBalance := l.CurrentBalance + qty
	costValue, err := unitCost.Multiply(qty)
	if err != nil {
		return LedgerTransactionID{}, nil, err
	}

	newValue, err := l.CurrentValue.Add(costValue)
	if err != nil {
		return LedgerTransactionID{}, nil, err
	}

	// Update average unit cost
	if newBalance > 0 {
		l.AverageUnitCost, err = newValue.Divide(newBalance)
		if err != nil {
			return LedgerTransactionID{}, nil, err
		}
	}

	// Debit INVENTORY
	debitEntry, err := NewDebitEntry(
		transactionID,
		AccountInventory,
		qty,
		unitCost,
		newBalance,
		newValue,
		l.SKU,
		locationID,
		referenceID,
		"po",
		fmt.Sprintf("Received %d units from %s", qty, referenceID),
		createdBy,
	)
	if err != nil {
		return LedgerTransactionID{}, nil, err
	}

	// Credit GOODS_IN_TRANSIT
	creditEntry, err := NewCreditEntry(
		transactionID,
		AccountGoodsInTransit,
		qty,
		unitCost,
		0, // GOODS_IN_TRANSIT balance not tracked in this aggregate
		ZeroMoney(unitCost.Currency()),
		l.SKU,
		locationID,
		referenceID,
		"po",
		fmt.Sprintf("Goods in transit cleared for %s", referenceID),
		createdBy,
	)
	if err != nil {
		return LedgerTransactionID{}, nil, err
	}

	// Update account balances
	l.updateAccountBalance(AccountInventory, qty, costValue)
	l.CurrentBalance = newBalance
	l.CurrentValue = newValue
	l.UpdatedAt = time.Now().UTC()

	// Emit valuation event
	l.addDomainEvent(&InventoryValuedEvent{
		SKU:             l.SKU,
		ValuationMethod: l.ValuationMethod.String(),
		TotalQuantity:   l.CurrentBalance,
		TotalValue:      l.CurrentValue.ToCents(),
		AverageUnitCost: l.AverageUnitCost.ToCents(),
		Currency:        l.CurrentValue.Currency(),
		CostLayerCount:  len(l.CostLayers),
		ValuedAt:        time.Now().UTC(),
	})

	return transactionID, []LedgerEntry{debitEntry, creditEntry}, nil
}

// RecordPick records inventory picked for order
// Debit COGS, Credit INVENTORY
func (l *InventoryLedger) RecordPick(qty int, locationID, orderID, createdBy string) (LedgerTransactionID, []LedgerEntry, error) {
	if qty <= 0 {
		return LedgerTransactionID{}, nil, ErrInvalidQuantity
	}

	if l.CurrentBalance < qty {
		return LedgerTransactionID{}, nil, ErrInsufficientStock
	}

	transactionID := NewLedgerTransactionID()

	// Consume cost layers based on valuation method
	var costConsumed Money
	var err error

	if l.ValuationMethod.UsesLayers() {
		costConsumed, err = l.consumeCostLayers(qty)
		if err != nil {
			return LedgerTransactionID{}, nil, err
		}
	} else {
		// Weighted average
		costConsumed, err = l.AverageUnitCost.Multiply(qty)
		if err != nil {
			return LedgerTransactionID{}, nil, err
		}
	}

	avgCost, err := costConsumed.Divide(qty)
	if err != nil {
		return LedgerTransactionID{}, nil, err
	}

	// Calculate new balance and value
	newBalance := l.CurrentBalance - qty
	newValue, err := l.CurrentValue.Subtract(costConsumed)
	if err != nil {
		return LedgerTransactionID{}, nil, err
	}

	// Debit COGS
	debitEntry, err := NewDebitEntry(
		transactionID,
		AccountCOGS,
		qty,
		avgCost,
		0, // COGS balance is cumulative, not tracked per SKU
		ZeroMoney(costConsumed.Currency()),
		l.SKU,
		locationID,
		orderID,
		"order",
		fmt.Sprintf("Picked %d units for order %s", qty, orderID),
		createdBy,
	)
	if err != nil {
		return LedgerTransactionID{}, nil, err
	}

	// Credit INVENTORY
	creditEntry, err := NewCreditEntry(
		transactionID,
		AccountInventory,
		qty,
		avgCost,
		newBalance,
		newValue,
		l.SKU,
		locationID,
		orderID,
		"order",
		fmt.Sprintf("Inventory reduced for order %s", orderID),
		createdBy,
	)
	if err != nil {
		return LedgerTransactionID{}, nil, err
	}

	// Update account balances
	l.updateAccountBalance(AccountInventory, -qty, Money{amount: -costConsumed.Amount(), currency: costConsumed.Currency()})
	l.updateAccountBalance(AccountCOGS, qty, costConsumed)
	l.CurrentBalance = newBalance
	l.CurrentValue = newValue
	l.UpdatedAt = time.Now().UTC()

	// Recalculate average unit cost
	if l.CurrentBalance > 0 {
		l.AverageUnitCost, _ = l.CurrentValue.Divide(l.CurrentBalance)
	}

	// Emit valuation event
	l.addDomainEvent(&InventoryValuedEvent{
		SKU:             l.SKU,
		ValuationMethod: l.ValuationMethod.String(),
		TotalQuantity:   l.CurrentBalance,
		TotalValue:      l.CurrentValue.ToCents(),
		AverageUnitCost: l.AverageUnitCost.ToCents(),
		Currency:        l.CurrentValue.Currency(),
		CostLayerCount:  len(l.CostLayers),
		ValuedAt:        time.Now().UTC(),
	})

	return transactionID, []LedgerEntry{debitEntry, creditEntry}, nil
}

// RecordAdjustment records inventory adjustment (positive or negative)
// Positive: Debit INVENTORY, Credit ADJUSTMENTS
// Negative: Debit ADJUSTMENTS, Credit INVENTORY
func (l *InventoryLedger) RecordAdjustment(qty int, reason, locationID, referenceID, createdBy string) (LedgerTransactionID, []LedgerEntry, error) {
	if qty == 0 {
		return LedgerTransactionID{}, nil, fmt.Errorf("adjustment quantity cannot be zero")
	}

	transactionID := NewLedgerTransactionID()
	var entries []LedgerEntry

	if qty > 0 {
		// Positive adjustment - add inventory
		unitCost := l.AverageUnitCost
		if unitCost.IsZero() && len(l.CostLayers) > 0 {
			unitCost = l.CostLayers[len(l.CostLayers)-1].UnitCost
		}

		// Add cost layer
		if l.ValuationMethod.UsesLayers() && !unitCost.IsZero() {
			l.AddCostLayer(qty, unitCost, referenceID)
		}

		newBalance := l.CurrentBalance + qty
		adjustValue, _ := unitCost.Multiply(qty)
		newValue, _ := l.CurrentValue.Add(adjustValue)

		// Debit INVENTORY
		debitEntry, _ := NewDebitEntry(transactionID, AccountInventory, qty, unitCost, newBalance, newValue, l.SKU, locationID, referenceID, "adjustment", fmt.Sprintf("Adjustment: %s", reason), createdBy)
		// Credit ADJUSTMENTS
		creditEntry, _ := NewCreditEntry(transactionID, AccountAdjustments, qty, unitCost, 0, ZeroMoney(unitCost.Currency()), l.SKU, locationID, referenceID, "adjustment", fmt.Sprintf("Adjustment: %s", reason), createdBy)

		entries = []LedgerEntry{debitEntry, creditEntry}

		l.updateAccountBalance(AccountInventory, qty, adjustValue)
		l.CurrentBalance = newBalance
		l.CurrentValue = newValue
	} else {
		// Negative adjustment - reduce inventory
		absQty := -qty

		if l.CurrentBalance < absQty {
			return LedgerTransactionID{}, nil, ErrInsufficientStock
		}

		var costConsumed Money
		var err error

		if l.ValuationMethod.UsesLayers() {
			costConsumed, err = l.consumeCostLayers(absQty)
			if err != nil {
				return LedgerTransactionID{}, nil, err
			}
		} else {
			costConsumed, _ = l.AverageUnitCost.Multiply(absQty)
		}

		avgCost, _ := costConsumed.Divide(absQty)
		newBalance := l.CurrentBalance - absQty
		newValue, _ := l.CurrentValue.Subtract(costConsumed)

		// Debit ADJUSTMENTS
		debitEntry, _ := NewDebitEntry(transactionID, AccountAdjustments, absQty, avgCost, 0, ZeroMoney(costConsumed.Currency()), l.SKU, locationID, referenceID, "adjustment", fmt.Sprintf("Adjustment: %s", reason), createdBy)
		// Credit INVENTORY
		creditEntry, _ := NewCreditEntry(transactionID, AccountInventory, absQty, avgCost, newBalance, newValue, l.SKU, locationID, referenceID, "adjustment", fmt.Sprintf("Adjustment: %s", reason), createdBy)

		entries = []LedgerEntry{debitEntry, creditEntry}

		l.updateAccountBalance(AccountInventory, -absQty, Money{amount: -costConsumed.Amount(), currency: costConsumed.Currency()})
		l.updateAccountBalance(AccountAdjustments, absQty, costConsumed)
		l.CurrentBalance = newBalance
		l.CurrentValue = newValue
	}

	l.UpdatedAt = time.Now().UTC()

	// Recalculate average
	if l.CurrentBalance > 0 {
		l.AverageUnitCost, _ = l.CurrentValue.Divide(l.CurrentBalance)
	}

	l.addDomainEvent(&InventoryValuedEvent{
		SKU:             l.SKU,
		ValuationMethod: l.ValuationMethod.String(),
		TotalQuantity:   l.CurrentBalance,
		TotalValue:      l.CurrentValue.ToCents(),
		AverageUnitCost: l.AverageUnitCost.ToCents(),
		Currency:        l.CurrentValue.Currency(),
		CostLayerCount:  len(l.CostLayers),
		ValuedAt:        time.Now().UTC(),
	})

	return transactionID, entries, nil
}

// AddCostLayer adds a new cost layer (for FIFO/LIFO)
func (l *InventoryLedger) AddCostLayer(qty int, unitCost Money, referenceID string) {
	layer := NewCostLayer(qty, unitCost, referenceID)
	l.CostLayers = append(l.CostLayers, layer)
}

// consumeCostLayers consumes cost layers based on valuation method
func (l *InventoryLedger) consumeCostLayers(qty int) (Money, error) {
	if len(l.CostLayers) == 0 {
		return Money{}, ErrNoCostLayers
	}

	remaining := qty
	totalCost := ZeroMoney(l.CostLayers[0].UnitCost.Currency())

	// FIFO: consume oldest first (index 0)
	// LIFO: consume newest first (index len-1)
	layers := make([]CostLayer, len(l.CostLayers))
	copy(layers, l.CostLayers)

	if l.ValuationMethod == ValuationLIFO {
		// Reverse for LIFO
		for i, j := 0, len(layers)-1; i < j; i, j = i+1, j-1 {
			layers[i], layers[j] = layers[j], layers[i]
		}
	}

	newLayers := make([]CostLayer, 0)

	for _, layer := range layers {
		if remaining == 0 {
			newLayers = append(newLayers, layer)
			continue
		}

		if layer.Quantity <= remaining {
			// Consume entire layer
			cost, _ := layer.TotalCost()
			totalCost, _ = totalCost.Add(cost)
			remaining -= layer.Quantity
		} else {
			// Consume partial layer
			cost, _ := layer.UnitCost.Multiply(remaining)
			totalCost, _ = totalCost.Add(cost)
			layer.Quantity -= remaining
			remaining = 0
			newLayers = append(newLayers, layer)
		}
	}

	if remaining > 0 {
		return Money{}, ErrInsufficientCostLayers
	}

	// Reverse back if LIFO
	if l.ValuationMethod == ValuationLIFO {
		for i, j := 0, len(newLayers)-1; i < j; i, j = i+1, j-1 {
			newLayers[i], newLayers[j] = newLayers[j], newLayers[i]
		}
	}

	l.CostLayers = newLayers

	return totalCost, nil
}

// updateAccountBalance updates the balance for a specific account
func (l *InventoryLedger) updateAccountBalance(accountType AccountType, qtyDelta int, valueDelta Money) {
	balance, exists := l.AccountBalances[accountType]
	if !exists {
		balance = ZeroAccountBalance(valueDelta.Currency())
	}

	balance.Balance += qtyDelta
	balance.Value, _ = balance.Value.Add(valueDelta)

	l.AccountBalances[accountType] = balance
}

// GetAccountBalance returns the balance for a specific account
func (l *InventoryLedger) GetAccountBalance(accountType AccountType) AccountBalance {
	if balance, exists := l.AccountBalances[accountType]; exists {
		return balance
	}
	return ZeroAccountBalance(l.CurrentValue.Currency())
}

// PullEvents returns and clears pending domain events
func (l *InventoryLedger) PullEvents() []DomainEvent {
	events := l.DomainEvents
	l.DomainEvents = nil
	return events
}

// addDomainEvent adds a domain event to the pending events
func (l *InventoryLedger) addDomainEvent(event DomainEvent) {
	l.DomainEvents = append(l.DomainEvents, event)
}

// ClearDomainEvents clears all pending domain events
func (l *InventoryLedger) ClearDomainEvents() {
	l.DomainEvents = nil
}
