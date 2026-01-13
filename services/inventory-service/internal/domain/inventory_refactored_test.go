package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInventoryItemRefactored_ReceiveStock(t *testing.T) {
	item := NewInventoryItemRefactored("SKU-1", "Widget", 10, 50, nil)

	require.NoError(t, item.ReceiveStock("LOC-1", "ZONE-A", 10))
	assert.Equal(t, 10, item.TotalQuantity)
	assert.Equal(t, 10, item.AvailableQuantity)
	require.Len(t, item.DomainEvents, 1)
	_, ok := item.DomainEvents[0].(*InventoryReceivedEvent)
	assert.True(t, ok)

	require.NoError(t, item.ReceiveStock("LOC-1", "ZONE-A", 5))
	assert.Equal(t, 15, item.TotalQuantity)
	assert.Equal(t, 15, item.AvailableQuantity)

	assert.ErrorIs(t, item.ReceiveStock("LOC-1", "ZONE-A", 0), ErrInvalidQuantity)
}

func TestInventoryItemRefactored_ReserveAndRelease(t *testing.T) {
	item := NewInventoryItemRefactored("SKU-1", "Widget", 10, 50, nil)
	require.NoError(t, item.ReceiveStock("LOC-1", "ZONE-A", 10))

	require.NoError(t, item.ReserveStock("LOC-1", 4))
	assert.Equal(t, 4, item.ReservedQuantity)
	assert.Equal(t, 6, item.AvailableQuantity)

	assert.ErrorIs(t, item.ReserveStock("LOC-2", 1), ErrLocationNotFound)
	assert.ErrorIs(t, item.ReserveStock("LOC-1", 20), ErrInsufficientStock)
	assert.ErrorIs(t, item.ReserveStock("LOC-1", 0), ErrInvalidQuantity)

	require.NoError(t, item.ReleaseReservation("LOC-1", 2))
	assert.Equal(t, 2, item.ReservedQuantity)
	assert.Equal(t, 8, item.AvailableQuantity)
	assert.ErrorIs(t, item.ReleaseReservation("LOC-1", 0), ErrInvalidQuantity)
}

func TestInventoryItemRefactored_HardAllocateAndRelease(t *testing.T) {
	item := NewInventoryItemRefactored("SKU-1", "Widget", 10, 50, nil)
	require.NoError(t, item.ReceiveStock("LOC-1", "ZONE-A", 10))
	require.NoError(t, item.ReserveStock("LOC-1", 4))

	require.NoError(t, item.HardAllocateStock("LOC-1", 3))
	assert.Equal(t, 1, item.ReservedQuantity)
	assert.Equal(t, 3, item.HardAllocatedQuantity)

	assert.Error(t, item.HardAllocateStock("LOC-1", 10))

	require.NoError(t, item.ReleaseHardAllocation("LOC-1", 2))
	assert.Equal(t, 1, item.HardAllocatedQuantity)
	assert.Equal(t, 8, item.AvailableQuantity)

	assert.Error(t, item.ReleaseHardAllocation("LOC-1", 10))
}

func TestInventoryItemRefactored_ShipStock(t *testing.T) {
	item := NewInventoryItemRefactored("SKU-1", "Widget", 10, 50, nil)
	require.NoError(t, item.ReceiveStock("LOC-1", "ZONE-A", 5))
	require.NoError(t, item.ReserveStock("LOC-1", 5))
	require.NoError(t, item.HardAllocateStock("LOC-1", 5))

	require.NoError(t, item.ShipStock("LOC-1", 5))
	assert.Equal(t, 0, item.HardAllocatedQuantity)
	assert.Equal(t, 0, item.TotalQuantity)
	assert.GreaterOrEqual(t, len(item.DomainEvents), 1)

	assert.ErrorIs(t, item.ShipStock("LOC-1", 0), ErrInvalidQuantity)
}

func TestInventoryItemRefactored_AdjustAndShortage(t *testing.T) {
	item := NewInventoryItemRefactored("SKU-1", "Widget", 10, 50, nil)
	require.NoError(t, item.ReceiveStock("LOC-1", "ZONE-A", 10))

	require.NoError(t, item.AdjustStock("LOC-1", 7))
	assert.Equal(t, 7, item.TotalQuantity)
	assert.Equal(t, 7, item.AvailableQuantity)
	assert.ErrorIs(t, item.AdjustStock("LOC-2", 1), ErrLocationNotFound)

	item2 := NewInventoryItemRefactored("SKU-2", "Widget", 10, 50, nil)
	require.NoError(t, item2.ReceiveStock("LOC-1", "ZONE-A", 5))
	require.NoError(t, item2.ReserveStock("LOC-1", 4))
	require.NoError(t, item2.RecordShortage("LOC-1", 3))
	assert.Equal(t, 2, item2.TotalQuantity)
	assert.Equal(t, 0, item2.AvailableQuantity)
	assert.Equal(t, 2, item2.ReservedQuantity)

	assert.ErrorIs(t, item2.RecordShortage("LOC-1", 0), ErrNoShortageToRecord)
}

func TestInventoryItemRefactored_VelocityAndStorage(t *testing.T) {
	item := NewInventoryItemRefactored("SKU-1", "Widget", 10, 50, nil)

	item.UpdatePickFrequency(5)
	assert.Equal(t, VelocityC, item.VelocityClass)
	assert.Len(t, item.DomainEvents, 1)

	item.UpdatePickFrequency(5)
	assert.Len(t, item.DomainEvents, 1)

	item.UpdatePickFrequency(20)
	assert.Equal(t, VelocityB, item.VelocityClass)
	assert.Len(t, item.DomainEvents, 2)

	item.SetStorageStrategy(StorageVelocity)
	assert.Equal(t, StorageVelocity, item.StorageStrategy)

	item.SetStorageStrategy(StorageStrategy("invalid"))
	assert.Equal(t, StorageVelocity, item.StorageStrategy)
}

func TestInventoryItemRefactored_Utilities(t *testing.T) {
	item := NewInventoryItemRefactored("SKU-1", "Widget", 10, 50, nil)
	require.NoError(t, item.ReceiveStock("LOC-1", "ZONE-A", 10))
	require.NoError(t, item.ReserveStock("LOC-1", 4))

	loc := item.GetLocationStock("LOC-1")
	require.NotNil(t, loc)
	assert.Equal(t, "LOC-1", loc.LocationID)

	available := item.GetAvailableLocations()
	require.Len(t, available, 1)

	item.RecordCycleCount()
	require.NotNil(t, item.LastCycleCount)

	item.RecordStow()
	require.NotNil(t, item.LastStowedAt)

	item.UpdatePickFrequency(1)
	assert.True(t, item.IsLowVelocity())
	item.UpdatePickFrequency(100)
	assert.True(t, item.IsHighVelocity())
}

func TestInventoryItemRefactored_EventsLifecycle(t *testing.T) {
	item := NewInventoryItemRefactored("SKU-1", "Widget", 10, 50, nil)
	item.AddDomainEvent(&InventoryAdjustedEvent{SKU: "SKU-1"})
	require.Len(t, item.DomainEvents, 1)

	item.ClearDomainEvents()
	assert.Len(t, item.DomainEvents, 0)
}
