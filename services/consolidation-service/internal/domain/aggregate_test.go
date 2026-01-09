package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test fixtures
func createTestExpectedItems() []ExpectedItem {
	return []ExpectedItem{
		{
			SKU:          "SKU-001",
			ProductName:  "Test Product 1",
			Quantity:     5,
			SourceToteID: "TOTE-001",
		},
		{
			SKU:          "SKU-002",
			ProductName:  "Test Product 2",
			Quantity:     3,
			SourceToteID: "TOTE-002",
		},
		{
			SKU:          "SKU-003",
			ProductName:  "Test Product 3",
			Quantity:     2,
			SourceToteID: "TOTE-001",
		},
	}
}

// TestNewConsolidationUnit tests consolidation unit creation
func TestNewConsolidationUnit(t *testing.T) {
	tests := []struct {
		name            string
		consolidationID string
		orderID         string
		waveID          string
		strategy        ConsolidationStrategy
		items           []ExpectedItem
		expectError     bool
	}{
		{
			name:            "Valid consolidation unit creation",
			consolidationID: "CONS-001",
			orderID:         "ORD-001",
			waveID:          "WAVE-001",
			strategy:        StrategyOrderBased,
			items:           createTestExpectedItems(),
			expectError:     false,
		},
		{
			name:            "Cannot create with no items",
			consolidationID: "CONS-002",
			orderID:         "ORD-002",
			waveID:          "WAVE-001",
			strategy:        StrategyOrderBased,
			items:           []ExpectedItem{},
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unit, err := NewConsolidationUnit(tt.consolidationID, tt.orderID, tt.waveID, tt.strategy, tt.items)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, unit)
			} else {
				require.NoError(t, err)
				require.NotNil(t, unit)
				assert.Equal(t, tt.consolidationID, unit.ConsolidationID)
				assert.Equal(t, tt.orderID, unit.OrderID)
				assert.Equal(t, tt.waveID, unit.WaveID)
				assert.Equal(t, ConsolidationStatusPending, unit.Status)
				assert.Equal(t, tt.strategy, unit.Strategy)
				assert.Equal(t, 10, unit.TotalExpected) // 5 + 3 + 2
				assert.Equal(t, 0, unit.TotalConsolidated)
				assert.False(t, unit.ReadyForPacking)
				assert.NotZero(t, unit.CreatedAt)

				// Check domain event
				events := unit.GetDomainEvents()
				assert.Len(t, events, 1)
				event, ok := events[0].(*ConsolidationStartedEvent)
				assert.True(t, ok)
				assert.Equal(t, tt.consolidationID, event.ConsolidationID)
			}
		})
	}
}

// TestConsolidationUnitAssignStation tests station assignment
func TestConsolidationUnitAssignStation(t *testing.T) {
	tests := []struct {
		name           string
		setupUnit      func() *ConsolidationUnit
		station        string
		workerID       string
		destinationBin string
		expectError    error
	}{
		{
			name: "Assign station to pending unit",
			setupUnit: func() *ConsolidationUnit {
				unit, _ := NewConsolidationUnit("CONS-001", "ORD-001", "WAVE-001", StrategyOrderBased, createTestExpectedItems())
				return unit
			},
			station:        "STATION-A1",
			workerID:       "WORKER-123",
			destinationBin: "BIN-001",
			expectError:    nil,
		},
		{
			name: "Cannot assign station to completed unit",
			setupUnit: func() *ConsolidationUnit {
				unit, _ := NewConsolidationUnit("CONS-002", "ORD-002", "WAVE-001", StrategyOrderBased, createTestExpectedItems())
				unit.Status = ConsolidationStatusCompleted
				return unit
			},
			station:        "STATION-A2",
			workerID:       "WORKER-456",
			destinationBin: "BIN-002",
			expectError:    ErrConsolidationComplete,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unit := tt.setupUnit()
			err := unit.AssignStation(tt.station, tt.workerID, tt.destinationBin)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.station, unit.Station)
				assert.Equal(t, tt.workerID, unit.WorkerID)
				assert.Equal(t, tt.destinationBin, unit.DestinationBin)
			}
		})
	}
}

// TestConsolidationUnitStart tests starting consolidation
func TestConsolidationUnitStart(t *testing.T) {
	tests := []struct {
		name        string
		setupUnit   func() *ConsolidationUnit
		expectError bool
	}{
		{
			name: "Start pending consolidation",
			setupUnit: func() *ConsolidationUnit {
				unit, _ := NewConsolidationUnit("CONS-001", "ORD-001", "WAVE-001", StrategyOrderBased, createTestExpectedItems())
				return unit
			},
			expectError: false,
		},
		{
			name: "Cannot start already started consolidation",
			setupUnit: func() *ConsolidationUnit {
				unit, _ := NewConsolidationUnit("CONS-002", "ORD-002", "WAVE-001", StrategyOrderBased, createTestExpectedItems())
				unit.Start()
				return unit
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unit := tt.setupUnit()
			err := unit.Start()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, ConsolidationStatusInProgress, unit.Status)
				assert.NotNil(t, unit.StartedAt)
			}
		})
	}
}

// TestConsolidationUnitConsolidateItem tests item consolidation
func TestConsolidationUnitConsolidateItem(t *testing.T) {
	tests := []struct {
		name         string
		setupUnit    func() *ConsolidationUnit
		sku          string
		quantity     int
		sourceToteID string
		verifiedBy   string
		expectError  error
	}{
		{
			name: "Consolidate expected item",
			setupUnit: func() *ConsolidationUnit {
				unit, _ := NewConsolidationUnit("CONS-001", "ORD-001", "WAVE-001", StrategyOrderBased, createTestExpectedItems())
				unit.AssignStation("STATION-A1", "WORKER-123", "BIN-001")
				return unit
			},
			sku:          "SKU-001",
			quantity:     5,
			sourceToteID: "TOTE-001",
			verifiedBy:   "WORKER-123",
			expectError:  nil,
		},
		{
			name: "Consolidate partial quantity",
			setupUnit: func() *ConsolidationUnit {
				unit, _ := NewConsolidationUnit("CONS-002", "ORD-002", "WAVE-001", StrategyOrderBased, createTestExpectedItems())
				unit.AssignStation("STATION-A1", "WORKER-123", "BIN-001")
				return unit
			},
			sku:          "SKU-001",
			quantity:     3,
			sourceToteID: "TOTE-001",
			verifiedBy:   "WORKER-123",
			expectError:  nil,
		},
		{
			name: "Cannot consolidate unexpected item",
			setupUnit: func() *ConsolidationUnit {
				unit, _ := NewConsolidationUnit("CONS-003", "ORD-003", "WAVE-001", StrategyOrderBased, createTestExpectedItems())
				unit.AssignStation("STATION-A1", "WORKER-123", "BIN-001")
				return unit
			},
			sku:          "SKU-999",
			quantity:     5,
			sourceToteID: "TOTE-999",
			verifiedBy:   "WORKER-123",
			expectError:  ErrItemNotExpected,
		},
		{
			name: "Cannot consolidate item from wrong tote",
			setupUnit: func() *ConsolidationUnit {
				unit, _ := NewConsolidationUnit("CONS-004", "ORD-004", "WAVE-001", StrategyOrderBased, createTestExpectedItems())
				unit.AssignStation("STATION-A1", "WORKER-123", "BIN-001")
				return unit
			},
			sku:          "SKU-001",
			quantity:     5,
			sourceToteID: "TOTE-999", // Wrong tote
			verifiedBy:   "WORKER-123",
			expectError:  ErrItemNotExpected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unit := tt.setupUnit()
			initialConsolidated := unit.TotalConsolidated
			err := unit.ConsolidateItem(tt.sku, tt.quantity, tt.sourceToteID, tt.verifiedBy)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, ConsolidationStatusInProgress, unit.Status)
				assert.Equal(t, initialConsolidated+tt.quantity, unit.TotalConsolidated)
				assert.NotEmpty(t, unit.ConsolidatedItems)

				// Verify item status updated
				for _, item := range unit.ExpectedItems {
					if item.SKU == tt.sku && item.SourceToteID == tt.sourceToteID {
						assert.GreaterOrEqual(t, item.Received, tt.quantity)
						if item.Received >= item.Quantity {
							assert.Equal(t, "received", item.Status)
						} else if item.Received > 0 {
							assert.Equal(t, "partial", item.Status)
						}
					}
				}
			}
		})
	}
}

// TestConsolidationUnitAutoComplete tests automatic completion
func TestConsolidationUnitAutoComplete(t *testing.T) {
	unit, _ := NewConsolidationUnit("CONS-001", "ORD-001", "WAVE-001", StrategyOrderBased, createTestExpectedItems())
	unit.AssignStation("STATION-A1", "WORKER-123", "BIN-001")

	// Consolidate all items
	err := unit.ConsolidateItem("SKU-001", 5, "TOTE-001", "WORKER-123")
	assert.NoError(t, err)
	assert.Equal(t, ConsolidationStatusInProgress, unit.Status)

	err = unit.ConsolidateItem("SKU-002", 3, "TOTE-002", "WORKER-123")
	assert.NoError(t, err)
	assert.Equal(t, ConsolidationStatusInProgress, unit.Status)

	// Last item should auto-complete
	err = unit.ConsolidateItem("SKU-003", 2, "TOTE-001", "WORKER-123")
	assert.NoError(t, err)
	assert.Equal(t, ConsolidationStatusCompleted, unit.Status)
	assert.True(t, unit.ReadyForPacking)
	assert.NotNil(t, unit.CompletedAt)
	assert.Equal(t, 10, unit.TotalConsolidated)
}

// TestConsolidationUnitComplete tests manual completion
func TestConsolidationUnitComplete(t *testing.T) {
	tests := []struct {
		name        string
		setupUnit   func() *ConsolidationUnit
		expectError error
	}{
		{
			name: "Complete in-progress consolidation",
			setupUnit: func() *ConsolidationUnit {
				unit, _ := NewConsolidationUnit("CONS-001", "ORD-001", "WAVE-001", StrategyOrderBased, createTestExpectedItems())
				unit.Start()
				return unit
			},
			expectError: nil,
		},
		{
			name: "Cannot complete already completed consolidation",
			setupUnit: func() *ConsolidationUnit {
				unit, _ := NewConsolidationUnit("CONS-002", "ORD-002", "WAVE-001", StrategyOrderBased, createTestExpectedItems())
				unit.Start()
				unit.Complete()
				return unit
			},
			expectError: ErrConsolidationComplete,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unit := tt.setupUnit()
			err := unit.Complete()

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, ConsolidationStatusCompleted, unit.Status)
				assert.True(t, unit.ReadyForPacking)
				assert.NotNil(t, unit.CompletedAt)
			}
		})
	}
}

// TestConsolidationUnitMarkShort tests marking items as short
func TestConsolidationUnitMarkShort(t *testing.T) {
	tests := []struct {
		name         string
		setupUnit    func() *ConsolidationUnit
		sku          string
		sourceToteID string
		shortQty     int
		reason       string
		expectError  error
	}{
		{
			name: "Mark expected item as short",
			setupUnit: func() *ConsolidationUnit {
				unit, _ := NewConsolidationUnit("CONS-001", "ORD-001", "WAVE-001", StrategyOrderBased, createTestExpectedItems())
				return unit
			},
			sku:          "SKU-001",
			sourceToteID: "TOTE-001",
			shortQty:     2,
			reason:       "Item damaged",
			expectError:  nil,
		},
		{
			name: "Cannot mark unexpected item as short",
			setupUnit: func() *ConsolidationUnit {
				unit, _ := NewConsolidationUnit("CONS-002", "ORD-002", "WAVE-001", StrategyOrderBased, createTestExpectedItems())
				return unit
			},
			sku:          "SKU-999",
			sourceToteID: "TOTE-999",
			shortQty:     1,
			reason:       "Not found",
			expectError:  ErrItemNotExpected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unit := tt.setupUnit()
			err := unit.MarkShort(tt.sku, tt.sourceToteID, tt.shortQty, tt.reason)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)

				// Verify item status updated
				for _, item := range unit.ExpectedItems {
					if item.SKU == tt.sku && item.SourceToteID == tt.sourceToteID {
						assert.Equal(t, "short", item.Status)
					}
				}
			}
		})
	}
}

// TestConsolidationUnitCancel tests consolidation cancellation
func TestConsolidationUnitCancel(t *testing.T) {
	tests := []struct {
		name        string
		setupUnit   func() *ConsolidationUnit
		reason      string
		expectError error
	}{
		{
			name: "Cancel pending consolidation",
			setupUnit: func() *ConsolidationUnit {
				unit, _ := NewConsolidationUnit("CONS-001", "ORD-001", "WAVE-001", StrategyOrderBased, createTestExpectedItems())
				return unit
			},
			reason:      "Order cancelled",
			expectError: nil,
		},
		{
			name: "Cancel in-progress consolidation",
			setupUnit: func() *ConsolidationUnit {
				unit, _ := NewConsolidationUnit("CONS-002", "ORD-002", "WAVE-001", StrategyOrderBased, createTestExpectedItems())
				unit.Start()
				return unit
			},
			reason:      "Order cancelled",
			expectError: nil,
		},
		{
			name: "Cannot cancel completed consolidation",
			setupUnit: func() *ConsolidationUnit {
				unit, _ := NewConsolidationUnit("CONS-003", "ORD-003", "WAVE-001", StrategyOrderBased, createTestExpectedItems())
				unit.Start()
				unit.Complete()
				return unit
			},
			reason:      "Too late",
			expectError: ErrConsolidationComplete,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unit := tt.setupUnit()
			err := unit.Cancel(tt.reason)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, ConsolidationStatusCancelled, unit.Status)
			}
		})
	}
}

// TestConsolidationUnitGetProgress tests progress calculation
func TestConsolidationUnitGetProgress(t *testing.T) {
	unit, _ := NewConsolidationUnit("CONS-001", "ORD-001", "WAVE-001", StrategyOrderBased, createTestExpectedItems())
	unit.AssignStation("STATION-A1", "WORKER-123", "BIN-001")

	// Initially 0%
	progress := unit.GetProgress()
	assert.Equal(t, 0.0, progress)

	// Consolidate partial
	unit.ConsolidateItem("SKU-001", 5, "TOTE-001", "WORKER-123")
	progress = unit.GetProgress()
	assert.InDelta(t, 50.0, progress, 0.1) // 5 out of 10 items

	// Consolidate more
	unit.ConsolidateItem("SKU-002", 3, "TOTE-002", "WORKER-123")
	progress = unit.GetProgress()
	assert.InDelta(t, 80.0, progress, 0.1) // 8 out of 10 items

	// Complete
	unit.ConsolidateItem("SKU-003", 2, "TOTE-001", "WORKER-123")
	progress = unit.GetProgress()
	assert.Equal(t, 100.0, progress)
}

// TestConsolidationUnitGetPendingItems tests getting pending items
func TestConsolidationUnitGetPendingItems(t *testing.T) {
	unit, _ := NewConsolidationUnit("CONS-001", "ORD-001", "WAVE-001", StrategyOrderBased, createTestExpectedItems())
	unit.AssignStation("STATION-A1", "WORKER-123", "BIN-001")

	// Initially all pending
	pending := unit.GetPendingItems()
	assert.Len(t, pending, 3)

	// Consolidate one item
	unit.ConsolidateItem("SKU-001", 5, "TOTE-001", "WORKER-123")
	pending = unit.GetPendingItems()
	assert.Len(t, pending, 2)

	// Consolidate all items
	unit.ConsolidateItem("SKU-002", 3, "TOTE-002", "WORKER-123")
	unit.ConsolidateItem("SKU-003", 2, "TOTE-001", "WORKER-123")
	pending = unit.GetPendingItems()
	assert.Len(t, pending, 0)
}

// TestConsolidationUnitStrategies tests different consolidation strategies
func TestConsolidationUnitStrategies(t *testing.T) {
	strategies := []ConsolidationStrategy{
		StrategyOrderBased,
		StrategyCarrierBased,
		StrategyRouteBased,
		StrategyTimeBased,
	}

	for _, strategy := range strategies {
		t.Run(string(strategy), func(t *testing.T) {
			unit, err := NewConsolidationUnit("CONS-001", "ORD-001", "WAVE-001", strategy, createTestExpectedItems())
			assert.NoError(t, err)
			assert.Equal(t, strategy, unit.Strategy)
		})
	}
}

// TestConsolidationUnitDomainEvents tests domain event handling
func TestConsolidationUnitDomainEvents(t *testing.T) {
	unit, _ := NewConsolidationUnit("CONS-001", "ORD-001", "WAVE-001", StrategyOrderBased, createTestExpectedItems())

	// Check initial event
	events := unit.GetDomainEvents()
	assert.Len(t, events, 1)
	_, ok := events[0].(*ConsolidationStartedEvent)
	assert.True(t, ok)

	// Consolidate item
	unit.AssignStation("STATION-A1", "WORKER-123", "BIN-001")
	unit.ConsolidateItem("SKU-001", 5, "TOTE-001", "WORKER-123")
	events = unit.GetDomainEvents()
	assert.Len(t, events, 2)

	// Complete
	unit.ConsolidateItem("SKU-002", 3, "TOTE-002", "WORKER-123")
	unit.ConsolidateItem("SKU-003", 2, "TOTE-001", "WORKER-123")
	events = unit.GetDomainEvents()
	assert.Len(t, events, 5) // Started + 3 Items + Completed

	// Clear events
	unit.ClearDomainEvents()
	events = unit.GetDomainEvents()
	assert.Len(t, events, 0)
}

// BenchmarkNewConsolidationUnit benchmarks consolidation unit creation
func BenchmarkNewConsolidationUnit(b *testing.B) {
	items := createTestExpectedItems()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewConsolidationUnit("CONS-001", "ORD-001", "WAVE-001", StrategyOrderBased, items)
	}
}

// BenchmarkConsolidateItem benchmarks item consolidation
func BenchmarkConsolidateItem(b *testing.B) {
	unit, _ := NewConsolidationUnit("CONS-001", "ORD-001", "WAVE-001", StrategyOrderBased, createTestExpectedItems())
	unit.AssignStation("STATION-A1", "WORKER-123", "BIN-001")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset for benchmark (this will complete after first iteration)
		unit.ConsolidateItem("SKU-001", 1, "TOTE-001", "WORKER-123")
	}
}
