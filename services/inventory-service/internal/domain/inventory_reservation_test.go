package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInventoryReservationAggregate_Lifecycle(t *testing.T) {
	reservation := NewInventoryReservation(
		"RES-1",
		"SKU-1",
		"ORD-1",
		"LOC-1",
		5,
		nil,
		"user1",
		&ReservationTenantInfo{
			TenantID:    "T-1",
			FacilityID:  "F-1",
			WarehouseID: "W-1",
			SellerID:    "S-1",
		},
	)

	assert.Equal(t, ReservationStatusActive, reservation.Status)
	assert.Equal(t, "T-1", reservation.TenantID)

	require.NoError(t, reservation.MarkStaged("user2"))
	assert.Equal(t, ReservationStatusStaged, reservation.Status)

	require.NoError(t, reservation.MarkFulfilled("user3"))
	assert.Equal(t, ReservationStatusFulfilled, reservation.Status)

	assert.Error(t, reservation.Cancel("user4", "too late"))
}

func TestInventoryReservationAggregate_ErrorsAndExpiration(t *testing.T) {
	reservation := NewInventoryReservation("RES-2", "SKU-2", "ORD-2", "LOC-2", 2, nil, "user1", nil)
	reservation.ExpiresAt = time.Now().Add(-1 * time.Minute)
	assert.ErrorIs(t, reservation.MarkStaged("user2"), ErrReservationExpired)

	reservation.Status = ReservationStatusCancelled
	assert.ErrorIs(t, reservation.MarkStaged("user2"), ErrReservationNotActive)

	reservation.Status = ReservationStatusCancelled
	assert.ErrorIs(t, reservation.MarkFulfilled("user3"), ErrReservationAlreadyUsed)
}

func TestInventoryReservationAggregate_ExpirationAndHelpers(t *testing.T) {
	reservation := NewInventoryReservation("RES-3", "SKU-3", "ORD-3", "LOC-3", 1, nil, "user1", nil)
	assert.True(t, reservation.IsActive())
	assert.False(t, reservation.IsExpired())

	reservation.ExpiresAt = time.Now().Add(-1 * time.Minute)
	reservation.MarkExpired()
	assert.Equal(t, ReservationStatusExpired, reservation.Status)
	assert.True(t, reservation.IsExpired())

	reservation.Status = ReservationStatusActive
	reservation.ExpiresAt = time.Now().Add(1 * time.Hour)
	oldExpiry := reservation.ExpiresAt
	require.NoError(t, reservation.ExtendExpiration(30*time.Minute))
	assert.True(t, reservation.ExpiresAt.After(oldExpiry))

	reservation.Status = ReservationStatusCancelled
	assert.ErrorIs(t, reservation.ExtendExpiration(10*time.Minute), ErrReservationNotActive)
}

func TestInventoryReservationAggregate_EventsLifecycle(t *testing.T) {
	reservation := NewInventoryReservation("RES-4", "SKU-4", "ORD-4", "LOC-4", 1, nil, "user1", nil)
	reservation.AddDomainEvent(&InventoryReceivedEvent{SKU: "SKU-4"})
	assert.Len(t, reservation.DomainEvents, 1)
	reservation.ClearDomainEvents()
	assert.Len(t, reservation.DomainEvents, 0)
}
