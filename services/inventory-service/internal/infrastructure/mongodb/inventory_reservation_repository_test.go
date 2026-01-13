package mongodb_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wms-platform/inventory-service/internal/domain"
	"github.com/wms-platform/inventory-service/internal/infrastructure/mongodb"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Test MongoDB connection string - use testcontainers or local instance
const testMongoURI = "mongodb://localhost:27017"
const testDatabase = "temporal_war_test"

func setupTestDB(t *testing.T) (*mongo.Database, func()) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(testMongoURI))
	require.NoError(t, err)

	db := client.Database(testDatabase)

	// Cleanup function
	cleanup := func() {
		_ = db.Drop(ctx)
		_ = client.Disconnect(ctx)
	}

	return db, cleanup
}

func Test_InventoryReservationRepository_Save(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := mongodb.NewInventoryReservationRepository(db)
	ctx := context.Background()

	// Create test reservation
	reservation := domain.NewInventoryReservation(
		"RES-001",
		"SKU-TEST-001",
		"ORDER-001",
		"LOC-A1",
		10,
		nil,
		"test-user",
		&domain.ReservationTenantInfo{
			TenantID:    "TENANT-001",
			FacilityID:  "FAC-001",
			WarehouseID: "WH-001",
		},
	)

	// Save reservation
	err := repo.Save(ctx, reservation)
	require.NoError(t, err)

	// Retrieve and verify
	retrieved, err := repo.FindByID(ctx, "RES-001")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, "RES-001", retrieved.ReservationID)
	assert.Equal(t, "SKU-TEST-001", retrieved.SKU)
	assert.Equal(t, "ORDER-001", retrieved.OrderID)
	assert.Equal(t, 10, retrieved.Quantity)
	assert.Equal(t, domain.ReservationStatusActive, retrieved.Status)
}

func Test_InventoryReservationRepository_FindBySKU(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := mongodb.NewInventoryReservationRepository(db)
	ctx := context.Background()

	// Create multiple reservations for same SKU
	sku := "SKU-TEST-001"
	tenant := &domain.ReservationTenantInfo{
		TenantID:    "TENANT-001",
		FacilityID:  "FAC-001",
		WarehouseID: "WH-001",
	}

	res1 := domain.NewInventoryReservation("RES-001", sku, "ORDER-001", "LOC-A1", 5, nil, "user1", tenant)
	res2 := domain.NewInventoryReservation("RES-002", sku, "ORDER-002", "LOC-A2", 10, nil, "user1", tenant)
	res3 := domain.NewInventoryReservation("RES-003", sku, "ORDER-003", "LOC-A3", 15, nil, "user1", tenant)

	// Cancel one reservation
	_ = res2.Cancel("user1", "test cancellation")

	require.NoError(t, repo.Save(ctx, res1))
	require.NoError(t, repo.Save(ctx, res2))
	require.NoError(t, repo.Save(ctx, res3))

	// Find active reservations only
	activeReservations, err := repo.FindBySKU(ctx, sku, domain.ReservationStatusActive)
	require.NoError(t, err)

	assert.Len(t, activeReservations, 2)
	assert.Equal(t, domain.ReservationStatusActive, activeReservations[0].Status)
}

func Test_InventoryReservationRepository_FindByOrderID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := mongodb.NewInventoryReservationRepository(db)
	ctx := context.Background()

	orderID := "ORDER-001"
	tenant := &domain.ReservationTenantInfo{
		TenantID:    "TENANT-001",
		FacilityID:  "FAC-001",
		WarehouseID: "WH-001",
	}

	// Create multiple reservations for same order (different SKUs)
	res1 := domain.NewInventoryReservation("RES-001", "SKU-001", orderID, "LOC-A1", 5, nil, "user1", tenant)
	res2 := domain.NewInventoryReservation("RES-002", "SKU-002", orderID, "LOC-A2", 10, nil, "user1", tenant)

	require.NoError(t, repo.Save(ctx, res1))
	require.NoError(t, repo.Save(ctx, res2))

	// Find all reservations for order
	reservations, err := repo.FindByOrderID(ctx, orderID)
	require.NoError(t, err)

	assert.Len(t, reservations, 2)
}

func Test_InventoryReservationRepository_FindExpired(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := mongodb.NewInventoryReservationRepository(db)
	ctx := context.Background()

	tenant := &domain.ReservationTenantInfo{
		TenantID:    "TENANT-001",
		FacilityID:  "FAC-001",
		WarehouseID: "WH-001",
	}

	// Create expired reservation
	expiredRes := domain.NewInventoryReservation("RES-001", "SKU-001", "ORDER-001", "LOC-A1", 5, nil, "user1", tenant)
	expiredRes.ExpiresAt = time.Now().Add(-1 * time.Hour) // Expired 1 hour ago

	// Create active reservation
	activeRes := domain.NewInventoryReservation("RES-002", "SKU-002", "ORDER-002", "LOC-A2", 10, nil, "user1", tenant)

	require.NoError(t, repo.Save(ctx, expiredRes))
	require.NoError(t, repo.Save(ctx, activeRes))

	// Find expired reservations
	expired, err := repo.FindExpired(ctx, 100)
	require.NoError(t, err)

	assert.Len(t, expired, 1)
	assert.Equal(t, "RES-001", expired[0].ReservationID)
	assert.True(t, expired[0].ExpiresAt.Before(time.Now()))
}

func Test_InventoryReservationRepository_UpdateStatus(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := mongodb.NewInventoryReservationRepository(db)
	ctx := context.Background()

	// Create reservation
	reservation := domain.NewInventoryReservation(
		"RES-001",
		"SKU-001",
		"ORDER-001",
		"LOC-A1",
		10,
		nil,
		"user1",
		&domain.ReservationTenantInfo{
			TenantID:    "TENANT-001",
			FacilityID:  "FAC-001",
			WarehouseID: "WH-001",
		},
	)

	require.NoError(t, repo.Save(ctx, reservation))

	// Update status
	err := repo.UpdateStatus(ctx, "RES-001", domain.ReservationStatusStaged)
	require.NoError(t, err)

	// Verify status changed
	updated, err := repo.FindByID(ctx, "RES-001")
	require.NoError(t, err)
	assert.Equal(t, domain.ReservationStatusStaged, updated.Status)
}

func Test_InventoryReservationRepository_GetActiveReservationCount(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := mongodb.NewInventoryReservationRepository(db)
	ctx := context.Background()

	sku := "SKU-TEST-001"
	tenant := &domain.ReservationTenantInfo{
		TenantID:    "TENANT-001",
		FacilityID:  "FAC-001",
		WarehouseID: "WH-001",
	}

	// Create 3 active reservations
	for i := 1; i <= 3; i++ {
		res := domain.NewInventoryReservation(
			fmt.Sprintf("RES-%03d", i),
			sku,
			fmt.Sprintf("ORDER-%03d", i),
			"LOC-A1",
			5,
			nil,
			"user1",
			tenant,
		)
		require.NoError(t, repo.Save(ctx, res))
	}

	// Create 1 cancelled reservation
	cancelledRes := domain.NewInventoryReservation("RES-004", sku, "ORDER-004", "LOC-A1", 5, nil, "user1", tenant)
	_ = cancelledRes.Cancel("user1", "test")
	require.NoError(t, repo.Save(ctx, cancelledRes))

	// Count active reservations
	count, err := repo.GetActiveReservationCountBySKU(ctx, sku)
	require.NoError(t, err)

	assert.Equal(t, int64(3), count)
}

func Test_ReservationAggregate_Lifecycle(t *testing.T) {
	// Test the reservation aggregate lifecycle
	reservation := domain.NewInventoryReservation(
		"RES-001",
		"SKU-001",
		"ORDER-001",
		"LOC-A1",
		10,
		nil,
		"user1",
		&domain.ReservationTenantInfo{
			TenantID:    "TENANT-001",
			FacilityID:  "FAC-001",
			WarehouseID: "WH-001",
		},
	)

	// Initial state
	assert.Equal(t, domain.ReservationStatusActive, reservation.Status)
	assert.True(t, reservation.IsActive())

	// Mark as staged
	err := reservation.MarkStaged("user1")
	require.NoError(t, err)
	assert.Equal(t, domain.ReservationStatusStaged, reservation.Status)
	assert.False(t, reservation.IsActive())

	// Mark as fulfilled
	err = reservation.MarkFulfilled("user1")
	require.NoError(t, err)
	assert.Equal(t, domain.ReservationStatusFulfilled, reservation.Status)

	// Cannot cancel fulfilled reservation
	err = reservation.Cancel("user1", "test")
	assert.Error(t, err)
}

func Test_ReservationAggregate_Expiration(t *testing.T) {
	// Create expired reservation
	reservation := domain.NewInventoryReservation(
		"RES-001",
		"SKU-001",
		"ORDER-001",
		"LOC-A1",
		10,
		nil,
		"user1",
		nil,
	)

	// Set expiration in the past
	reservation.ExpiresAt = time.Now().Add(-1 * time.Hour)

	assert.True(t, reservation.IsExpired())
	assert.False(t, reservation.IsActive())

	// Mark as expired
	reservation.MarkExpired()
	assert.Equal(t, domain.ReservationStatusExpired, reservation.Status)
}

func Test_ReservationAggregate_ExtendExpiration(t *testing.T) {
	reservation := domain.NewInventoryReservation(
		"RES-001",
		"SKU-001",
		"ORDER-001",
		"LOC-A1",
		10,
		nil,
		"user1",
		nil,
	)

	originalExpiration := reservation.ExpiresAt

	// Extend by 1 hour
	err := reservation.ExtendExpiration(1 * time.Hour)
	require.NoError(t, err)

	assert.True(t, reservation.ExpiresAt.After(originalExpiration))
	assert.Equal(t, 1*time.Hour, reservation.ExpiresAt.Sub(originalExpiration))
}
