package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInventoryAllocationAggregate_Lifecycle(t *testing.T) {
	allocation := NewInventoryAllocation(
		"ALLOC-1",
		"SKU-1",
		"RES-1",
		"ORD-1",
		5,
		"LOC-1",
		"STAGE-1",
		nil,
		"user1",
		&AllocationTenantInfo{
			TenantID:    "T-1",
			FacilityID:  "F-1",
			WarehouseID: "W-1",
			SellerID:    "S-1",
		},
	)

	assert.Equal(t, AllocationStatusStaged, allocation.Status)
	assert.Equal(t, "T-1", allocation.TenantID)
	require.Len(t, allocation.DomainEvents, 1)

	assert.ErrorIs(t, allocation.MarkShipped("user1"), ErrAllocationNotPacked)
	require.NoError(t, allocation.MarkPacked("user2"))
	assert.Equal(t, AllocationStatusPacked, allocation.Status)
	assert.NotNil(t, allocation.PackedAt)

	require.NoError(t, allocation.MarkShipped("user3"))
	assert.Equal(t, AllocationStatusShipped, allocation.Status)
	assert.NotNil(t, allocation.ShippedAt)

	assert.ErrorIs(t, allocation.ReturnToShelf("user4", "reason"), ErrAllocationAlreadyShipped)
	assert.True(t, allocation.IsShipped())
}

func TestInventoryAllocationAggregate_ReturnToShelf(t *testing.T) {
	allocation := NewInventoryAllocation(
		"ALLOC-2",
		"SKU-2",
		"RES-2",
		"ORD-2",
		3,
		"LOC-2",
		"STAGE-2",
		nil,
		"user1",
		nil,
	)

	require.NoError(t, allocation.ReturnToShelf("user2", "damaged"))
	assert.Equal(t, AllocationStatusReturned, allocation.Status)
	assert.True(t, allocation.IsReturned())
	assert.False(t, allocation.IsActive())
}

func TestInventoryAllocationAggregate_EventsLifecycle(t *testing.T) {
	allocation := NewInventoryAllocation(
		"ALLOC-3",
		"SKU-3",
		"RES-3",
		"ORD-3",
		1,
		"LOC-3",
		"STAGE-3",
		nil,
		"user1",
		nil,
	)

	allocation.ClearDomainEvents()
	assert.Len(t, allocation.DomainEvents, 0)
	allocation.AddDomainEvent(&InventoryPackedEvent{SKU: "SKU-3"})
	assert.Len(t, allocation.DomainEvents, 1)
}
