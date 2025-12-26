package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewInventoryItem tests inventory item creation
func TestNewInventoryItem(t *testing.T) {
	item := NewInventoryItem("SKU-001", "Test Product", 10, 50)

	require.NotNil(t, item)
	assert.Equal(t, "SKU-001", item.SKU)
	assert.Equal(t, "Test Product", item.ProductName)
	assert.Equal(t, 10, item.ReorderPoint)
	assert.Equal(t, 50, item.ReorderQuantity)
	assert.Equal(t, 0, item.TotalQuantity)
	assert.Equal(t, 0, item.AvailableQuantity)
	assert.NotZero(t, item.CreatedAt)
}

// TestInventoryReceiveStock tests receiving stock
func TestInventoryReceiveStock(t *testing.T) {
	tests := []struct {
		name        string
		setupItem   func() *InventoryItem
		locationID  string
		zone        string
		quantity    int
		expectError error
	}{
		{
			name: "Receive stock at new location",
			setupItem: func() *InventoryItem {
				return NewInventoryItem("SKU-001", "Test Product", 10, 50)
			},
			locationID:  "LOC-A1",
			zone:        "ZONE-A",
			quantity:    100,
			expectError: nil,
		},
		{
			name: "Receive stock at existing location",
			setupItem: func() *InventoryItem {
				item := NewInventoryItem("SKU-001", "Test Product", 10, 50)
				item.ReceiveStock("LOC-A1", "ZONE-A", 100, "PO-001", "user1")
				return item
			},
			locationID:  "LOC-A1",
			zone:        "ZONE-A",
			quantity:    50,
			expectError: nil,
		},
		{
			name: "Invalid quantity",
			setupItem: func() *InventoryItem {
				return NewInventoryItem("SKU-001", "Test Product", 10, 50)
			},
			locationID:  "LOC-A1",
			zone:        "ZONE-A",
			quantity:    -10,
			expectError: ErrInvalidQuantity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := tt.setupItem()
			initialTotal := item.TotalQuantity
			err := item.ReceiveStock(tt.locationID, tt.zone, tt.quantity, "PO-001", "user1")

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, initialTotal+tt.quantity, item.TotalQuantity)
				assert.Equal(t, initialTotal+tt.quantity, item.AvailableQuantity)
			}
		})
	}
}

// TestInventoryReserve tests reserving stock
func TestInventoryReserve(t *testing.T) {
	tests := []struct {
		name        string
		setupItem   func() *InventoryItem
		orderID     string
		quantity    int
		expectError error
	}{
		{
			name: "Reserve available stock",
			setupItem: func() *InventoryItem {
				item := NewInventoryItem("SKU-001", "Test Product", 10, 50)
				item.ReceiveStock("LOC-A1", "ZONE-A", 100, "PO-001", "user1")
				return item
			},
			orderID:     "ORD-001",
			quantity:    20,
			expectError: nil,
		},
		{
			name: "Insufficient stock",
			setupItem: func() *InventoryItem {
				item := NewInventoryItem("SKU-001", "Test Product", 10, 50)
				item.ReceiveStock("LOC-A1", "ZONE-A", 10, "PO-001", "user1")
				return item
			},
			orderID:     "ORD-002",
			quantity:    20,
			expectError: ErrInsufficientStock,
		},
		{
			name: "Invalid quantity",
			setupItem: func() *InventoryItem {
				item := NewInventoryItem("SKU-001", "Test Product", 10, 50)
				item.ReceiveStock("LOC-A1", "ZONE-A", 100, "PO-001", "user1")
				return item
			},
			orderID:     "ORD-003",
			quantity:    0,
			expectError: ErrInvalidQuantity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := tt.setupItem()
			initialReserved := item.ReservedQuantity
			initialAvailable := item.AvailableQuantity
			err := item.Reserve(tt.orderID, "LOC-A1", tt.quantity)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, initialReserved+tt.quantity, item.ReservedQuantity)
				assert.Equal(t, initialAvailable-tt.quantity, item.AvailableQuantity)
				assert.Len(t, item.Reservations, 1)
				assert.Equal(t, tt.orderID, item.Reservations[0].OrderID)
			}
		})
	}
}

// TestInventoryReleaseReservation tests releasing reservations
func TestInventoryReleaseReservation(t *testing.T) {
	tests := []struct {
		name        string
		setupItem   func() *InventoryItem
		orderID     string
		expectError error
	}{
		{
			name: "Release existing reservation",
			setupItem: func() *InventoryItem {
				item := NewInventoryItem("SKU-001", "Test Product", 10, 50)
				item.ReceiveStock("LOC-A1", "ZONE-A", 100, "PO-001", "user1")
				item.Reserve("ORD-001", "LOC-A1", 20)
				return item
			},
			orderID:     "ORD-001",
			expectError: nil,
		},
		{
			name: "Reservation not found",
			setupItem: func() *InventoryItem {
				item := NewInventoryItem("SKU-001", "Test Product", 10, 50)
				item.ReceiveStock("LOC-A1", "ZONE-A", 100, "PO-001", "user1")
				return item
			},
			orderID:     "ORD-999",
			expectError: ErrReservationNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := tt.setupItem()
			initialReserved := item.ReservedQuantity
			initialAvailable := item.AvailableQuantity
			err := item.ReleaseReservation(tt.orderID)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.Less(t, item.ReservedQuantity, initialReserved)
				assert.Greater(t, item.AvailableQuantity, initialAvailable)
			}
		})
	}
}

// TestInventoryPick tests picking/fulfilling reservations
func TestInventoryPick(t *testing.T) {
	tests := []struct {
		name        string
		setupItem   func() *InventoryItem
		orderID     string
		locationID  string
		quantity    int
		expectError error
	}{
		{
			name: "Pick reserved stock",
			setupItem: func() *InventoryItem {
				item := NewInventoryItem("SKU-001", "Test Product", 10, 50)
				item.ReceiveStock("LOC-A1", "ZONE-A", 100, "PO-001", "user1")
				item.Reserve("ORD-001", "LOC-A1", 20)
				return item
			},
			orderID:     "ORD-001",
			locationID:  "LOC-A1",
			quantity:    20,
			expectError: nil,
		},
		{
			name: "Pick more than reserved",
			setupItem: func() *InventoryItem {
				item := NewInventoryItem("SKU-001", "Test Product", 10, 50)
				item.ReceiveStock("LOC-A1", "ZONE-A", 100, "PO-001", "user1")
				item.Reserve("ORD-002", "LOC-A1", 10)
				return item
			},
			orderID:     "ORD-002",
			locationID:  "LOC-A1",
			quantity:    20,
			expectError: ErrInsufficientStock,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := tt.setupItem()
			initialTotal := item.TotalQuantity
			err := item.Pick(tt.orderID, tt.locationID, tt.quantity, "user1")

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, initialTotal-tt.quantity, item.TotalQuantity)
			}
		})
	}
}

// TestInventoryAdjust tests inventory adjustments
func TestInventoryAdjust(t *testing.T) {
	tests := []struct {
		name       string
		setupItem  func() *InventoryItem
		locationID string
		quantity   int
		reason     string
		expectError error
	}{
		{
			name: "Positive adjustment",
			setupItem: func() *InventoryItem {
				item := NewInventoryItem("SKU-001", "Test Product", 10, 50)
				item.ReceiveStock("LOC-A1", "ZONE-A", 100, "PO-001", "user1")
				return item
			},
			locationID: "LOC-A1",
			quantity:   5,
			reason:     "Cycle count correction",
			expectError: nil,
		},
		{
			name: "Negative adjustment",
			setupItem: func() *InventoryItem {
				item := NewInventoryItem("SKU-001", "Test Product", 10, 50)
				item.ReceiveStock("LOC-A1", "ZONE-A", 100, "PO-001", "user1")
				return item
			},
			locationID: "LOC-A1",
			quantity:   -10,
			reason:     "Damaged goods",
			expectError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := tt.setupItem()
			initialTotal := item.TotalQuantity
			err := item.Adjust(tt.locationID, tt.quantity, tt.reason, "user1")

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, initialTotal+tt.quantity, item.TotalQuantity)
			}
		})
	}
}

// TestInventoryNeedsReorder tests reorder point logic
// Note: NeedsReorder method not implemented in aggregate yet
/*
func TestInventoryNeedsReorder(t *testing.T) {
	item := NewInventoryItem("SKU-001", "Test Product", 10, 50)
	item.ReceiveStock("LOC-A1", "ZONE-A", 5, "PO-001", "user1")
	// Would test: needsReorder := item.NeedsReorder()
	// assert.True(t, needsReorder)
}
*/

// TestInventoryDomainEvents tests domain event handling
func TestInventoryDomainEvents(t *testing.T) {
	item := NewInventoryItem("SKU-001", "Test Product", 10, 50)

	// Receive stock
	item.ReceiveStock("LOC-A1", "ZONE-A", 100, "PO-001", "user1")
	events := item.GetDomainEvents()
	assert.GreaterOrEqual(t, len(events), 1)

	// Reserve
	item.Reserve("ORD-001", "LOC-A1", 20)
	events = item.GetDomainEvents()
	assert.GreaterOrEqual(t, len(events), 2)

	// Clear events
	item.ClearDomainEvents()
	events = item.GetDomainEvents()
	assert.Len(t, events, 0)
}

// BenchmarkReceiveStock benchmarks receiving stock
func BenchmarkReceiveStock(b *testing.B) {
	item := NewInventoryItem("SKU-001", "Test Product", 10, 50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item.ReceiveStock("LOC-A1", "ZONE-A", 10, "PO-001", "user1")
	}
}

// BenchmarkReserve benchmarks stock reservation
func BenchmarkReserve(b *testing.B) {
	item := NewInventoryItem("SKU-001", "Test Product", 10, 50)
	item.ReceiveStock("LOC-A1", "ZONE-A", 1000000, "PO-001", "user1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		orderID := "ORD-" + string(rune(i))
		item.Reserve(orderID, "LOC-A1", 1)
	}
}
