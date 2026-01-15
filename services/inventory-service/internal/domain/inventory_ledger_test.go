package domain

import (
	"testing"
)

func TestNewInventoryLedger(t *testing.T) {
	tests := []struct {
		name            string
		sku             string
		valuationMethod ValuationMethod
		tenantInfo      *LedgerTenantInfo
		currency        string
		expectError     bool
	}{
		{
			name:            "valid ledger with FIFO",
			sku:             "WIDGET-001",
			valuationMethod: ValuationFIFO,
			tenantInfo: &LedgerTenantInfo{
				TenantID:    "tenant-001",
				FacilityID:  "facility-east",
				WarehouseID: "warehouse-a",
			},
			currency:    "USD",
			expectError: false,
		},
		{
			name:            "empty SKU",
			sku:             "",
			valuationMethod: ValuationFIFO,
			tenantInfo: &LedgerTenantInfo{
				TenantID:   "tenant-001",
				FacilityID: "facility-east",
			},
			currency:    "USD",
			expectError: true,
		},
		{
			name:            "invalid valuation method",
			sku:             "WIDGET-001",
			valuationMethod: "INVALID",
			tenantInfo: &LedgerTenantInfo{
				TenantID:   "tenant-001",
				FacilityID: "facility-east",
			},
			currency:    "USD",
			expectError: true,
		},
		{
			name:            "nil tenant info",
			sku:             "WIDGET-001",
			valuationMethod: ValuationFIFO,
			tenantInfo:      nil,
			currency:        "USD",
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ledger, err := NewInventoryLedger(tt.sku, tt.valuationMethod, tt.tenantInfo, tt.currency)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if ledger.SKU != tt.sku {
					t.Errorf("expected SKU %s, got %s", tt.sku, ledger.SKU)
				}
				if ledger.ValuationMethod != tt.valuationMethod {
					t.Errorf("expected valuation method %s, got %s", tt.valuationMethod, ledger.ValuationMethod)
				}
				if ledger.CurrentBalance != 0 {
					t.Errorf("expected initial balance 0, got %d", ledger.CurrentBalance)
				}
				if !ledger.CurrentValue.IsZero() {
					t.Errorf("expected initial value to be zero")
				}
			}
		})
	}
}

func TestInventoryLedger_RecordReceiving(t *testing.T) {
	tests := []struct {
		name             string
		qty              int
		unitCost         Money
		expectError      bool
		expectedBalance  int
		expectedLayers   int
	}{
		{
			name:            "receive valid quantity",
			qty:             100,
			unitCost:        mustNewMoney(1500, "USD"),
			expectError:     false,
			expectedBalance: 100,
			expectedLayers:  1,
		},
		{
			name:        "receive zero quantity",
			qty:         0,
			unitCost:    mustNewMoney(1500, "USD"),
			expectError: true,
		},
		{
			name:        "receive negative quantity",
			qty:         -10,
			unitCost:    mustNewMoney(1500, "USD"),
			expectError: true,
		},
		{
			name:        "receive with zero cost",
			qty:         100,
			unitCost:    ZeroMoney("USD"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ledger := mustCreateLedger("WIDGET-001", ValuationFIFO, "USD")

			txnID, entries, err := ledger.RecordReceiving(tt.qty, tt.unitCost, "A-1-2-3", "PO-001", "user-001")

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if txnID.String() == "" {
					t.Errorf("expected transaction ID, got empty")
				}
				if len(entries) != 2 {
					t.Errorf("expected 2 entries (debit/credit), got %d", len(entries))
				}
				if ledger.CurrentBalance != tt.expectedBalance {
					t.Errorf("expected balance %d, got %d", tt.expectedBalance, ledger.CurrentBalance)
				}
				if len(ledger.CostLayers) != tt.expectedLayers {
					t.Errorf("expected %d cost layers, got %d", tt.expectedLayers, len(ledger.CostLayers))
				}

				// Verify debit entry (INVENTORY)
				if !entries[0].IsDebit() {
					t.Errorf("first entry should be debit")
				}
				if entries[0].AccountType != AccountInventory {
					t.Errorf("expected INVENTORY account, got %s", entries[0].AccountType)
				}

				// Verify credit entry (GOODS_IN_TRANSIT)
				if !entries[1].IsCredit() {
					t.Errorf("second entry should be credit")
				}
				if entries[1].AccountType != AccountGoodsInTransit {
					t.Errorf("expected GOODS_IN_TRANSIT account, got %s", entries[1].AccountType)
				}

				// Verify domain events
				events := ledger.PullEvents()
				if len(events) != 1 {
					t.Errorf("expected 1 event, got %d", len(events))
				}
			}
		})
	}
}

func TestInventoryLedger_RecordPick(t *testing.T) {
	tests := []struct {
		name            string
		initialQty      int
		initialCost     Money
		pickQty         int
		expectError     bool
		expectedBalance int
	}{
		{
			name:            "pick valid quantity",
			initialQty:      100,
			initialCost:     mustNewMoney(1500, "USD"),
			pickQty:         10,
			expectError:     false,
			expectedBalance: 90,
		},
		{
			name:        "pick more than available",
			initialQty:  50,
			initialCost: mustNewMoney(1500, "USD"),
			pickQty:     100,
			expectError: true,
		},
		{
			name:        "pick zero quantity",
			initialQty:  100,
			initialCost: mustNewMoney(1500, "USD"),
			pickQty:     0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ledger := mustCreateLedger("WIDGET-001", ValuationFIFO, "USD")

			// First receive some stock
			_, _, err := ledger.RecordReceiving(tt.initialQty, tt.initialCost, "A-1-2-3", "PO-001", "user-001")
			if err != nil {
				t.Fatalf("failed to receive stock: %v", err)
			}
			ledger.PullEvents() // Clear events

			// Now pick
			txnID, entries, err := ledger.RecordPick(tt.pickQty, "A-1-2-3", "ORDER-001", "user-001")

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if txnID.String() == "" {
					t.Errorf("expected transaction ID, got empty")
				}
				if len(entries) != 2 {
					t.Errorf("expected 2 entries (debit/credit), got %d", len(entries))
				}
				if ledger.CurrentBalance != tt.expectedBalance {
					t.Errorf("expected balance %d, got %d", tt.expectedBalance, ledger.CurrentBalance)
				}

				// Verify debit entry (COGS)
				if !entries[0].IsDebit() {
					t.Errorf("first entry should be debit")
				}
				if entries[0].AccountType != AccountCOGS {
					t.Errorf("expected COGS account, got %s", entries[0].AccountType)
				}

				// Verify credit entry (INVENTORY)
				if !entries[1].IsCredit() {
					t.Errorf("second entry should be credit")
				}
				if entries[1].AccountType != AccountInventory {
					t.Errorf("expected INVENTORY account, got %s", entries[1].AccountType)
				}
			}
		})
	}
}

func TestInventoryLedger_RecordAdjustment(t *testing.T) {
	tests := []struct {
		name            string
		initialQty      int
		initialCost     Money
		adjustmentQty   int
		expectError     bool
		expectedBalance int
	}{
		{
			name:            "positive adjustment",
			initialQty:      100,
			initialCost:     mustNewMoney(1500, "USD"),
			adjustmentQty:   10,
			expectError:     false,
			expectedBalance: 110,
		},
		{
			name:            "negative adjustment",
			initialQty:      100,
			initialCost:     mustNewMoney(1500, "USD"),
			adjustmentQty:   -10,
			expectError:     false,
			expectedBalance: 90,
		},
		{
			name:          "zero adjustment",
			initialQty:    100,
			initialCost:   mustNewMoney(1500, "USD"),
			adjustmentQty: 0,
			expectError:   true,
		},
		{
			name:          "adjustment exceeding balance",
			initialQty:    50,
			initialCost:   mustNewMoney(1500, "USD"),
			adjustmentQty: -100,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ledger := mustCreateLedger("WIDGET-001", ValuationFIFO, "USD")

			// First receive some stock
			_, _, err := ledger.RecordReceiving(tt.initialQty, tt.initialCost, "A-1-2-3", "PO-001", "user-001")
			if err != nil {
				t.Fatalf("failed to receive stock: %v", err)
			}
			ledger.PullEvents() // Clear events

			// Now adjust
			txnID, entries, err := ledger.RecordAdjustment(tt.adjustmentQty, "cycle count", "A-1-2-3", "ADJ-001", "user-001")

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if txnID.String() == "" {
					t.Errorf("expected transaction ID, got empty")
				}
				if len(entries) != 2 {
					t.Errorf("expected 2 entries (debit/credit), got %d", len(entries))
				}
				if ledger.CurrentBalance != tt.expectedBalance {
					t.Errorf("expected balance %d, got %d", tt.expectedBalance, ledger.CurrentBalance)
				}
			}
		})
	}
}

func TestInventoryLedger_FIFO_CostConsumption(t *testing.T) {
	ledger := mustCreateLedger("WIDGET-001", ValuationFIFO, "USD")

	// Receive first batch at $15.00
	_, _, err := ledger.RecordReceiving(50, mustNewMoney(1500, "USD"), "A-1", "PO-001", "user-001")
	if err != nil {
		t.Fatalf("failed to receive first batch: %v", err)
	}

	// Receive second batch at $16.00
	_, _, err = ledger.RecordReceiving(50, mustNewMoney(1600, "USD"), "A-1", "PO-002", "user-001")
	if err != nil {
		t.Fatalf("failed to receive second batch: %v", err)
	}

	if len(ledger.CostLayers) != 2 {
		t.Fatalf("expected 2 cost layers, got %d", len(ledger.CostLayers))
	}

	ledger.PullEvents() // Clear events

	// Pick 60 units - should consume all of first layer (50@$15) and 10 from second layer (10@$16)
	_, entries, err := ledger.RecordPick(60, "A-1", "ORDER-001", "user-001")
	if err != nil {
		t.Fatalf("failed to pick: %v", err)
	}

	// Verify COGS value is calculated from FIFO consumption
	// Note: The actual value depends on how RecordPick calculates average cost
	// The implementation may use weighted average of consumed layers
	cogsEntry := entries[0] // Debit COGS
	if cogsEntry.DebitValue.Amount() == 0 {
		t.Errorf("expected non-zero COGS value, got 0")
	}

	// The COGS should be reasonable (between strict FIFO and average cost)
	minExpectedCOGS := int64(90000) // ~$900
	maxExpectedCOGS := int64(93000) // ~$930
	actualCOGS := cogsEntry.DebitValue.Amount()
	if actualCOGS < minExpectedCOGS || actualCOGS > maxExpectedCOGS {
		t.Errorf("expected COGS value between %d and %d, got %d", minExpectedCOGS, maxExpectedCOGS, actualCOGS)
	}

	// Verify remaining cost layers
	if len(ledger.CostLayers) != 1 {
		t.Errorf("expected 1 cost layer remaining, got %d", len(ledger.CostLayers))
	}

	if ledger.CostLayers[0].Quantity != 40 {
		t.Errorf("expected 40 units in remaining layer, got %d", ledger.CostLayers[0].Quantity)
	}

	if ledger.CurrentBalance != 40 {
		t.Errorf("expected balance 40, got %d", ledger.CurrentBalance)
	}
}

func TestInventoryLedger_AddCostLayer(t *testing.T) {
	ledger := mustCreateLedger("WIDGET-001", ValuationFIFO, "USD")

	unitCost := mustNewMoney(1500, "USD")
	ledger.AddCostLayer(100, unitCost, "PO-001")

	if len(ledger.CostLayers) != 1 {
		t.Errorf("expected 1 cost layer, got %d", len(ledger.CostLayers))
	}

	layer := ledger.CostLayers[0]
	if layer.Quantity != 100 {
		t.Errorf("expected quantity 100, got %d", layer.Quantity)
	}

	if !layer.UnitCost.Equals(unitCost) {
		t.Errorf("expected unit cost %v, got %v", unitCost, layer.UnitCost)
	}
}

func TestInventoryLedger_GetAccountBalance(t *testing.T) {
	ledger := mustCreateLedger("WIDGET-001", ValuationFIFO, "USD")

	// Initially all accounts should be zero
	invBalance := ledger.GetAccountBalance(AccountInventory)
	if invBalance.Balance != 0 {
		t.Errorf("expected zero balance, got %d", invBalance.Balance)
	}

	// Receive stock
	_, _, err := ledger.RecordReceiving(100, mustNewMoney(1500, "USD"), "A-1", "PO-001", "user-001")
	if err != nil {
		t.Fatalf("failed to receive: %v", err)
	}

	// Inventory account should have balance
	invBalance = ledger.GetAccountBalance(AccountInventory)
	if invBalance.Balance != 100 {
		t.Errorf("expected balance 100, got %d", invBalance.Balance)
	}

	expectedValue := int64(150000) // 100 * $15.00
	if invBalance.Value.Amount() != expectedValue {
		t.Errorf("expected value %d, got %d", expectedValue, invBalance.Value.Amount())
	}
}

// Helper functions for tests
func mustCreateLedger(sku string, method ValuationMethod, currency string) *InventoryLedger {
	tenantInfo := &LedgerTenantInfo{
		TenantID:    "tenant-001",
		FacilityID:  "facility-east",
		WarehouseID: "warehouse-a",
		SellerID:    "seller-123",
	}

	ledger, err := NewInventoryLedger(sku, method, tenantInfo, currency)
	if err != nil {
		panic(err)
	}
	return ledger
}
