package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wms-platform/consolidation-service/internal/domain"
	"github.com/wms-platform/consolidation-service/internal/infrastructure/mongodb"
	sharedtesting "github.com/wms-platform/shared/pkg/testing"
)

// Test fixtures
func createTestConsolidationUnit(consolidationID, orderID, waveID string, status domain.ConsolidationStatus) (*domain.ConsolidationUnit, error) {
	items := []domain.ExpectedItem{
		{
			SKU:          "SKU-001",
			ProductName:  "Test Product 1",
			Quantity:     5,
			SourceToteID: "TOTE-001",
			Received:     0,
			Status:       "pending",
		},
		{
			SKU:          "SKU-002",
			ProductName:  "Test Product 2",
			Quantity:     3,
			SourceToteID: "TOTE-002",
			Received:     0,
			Status:       "pending",
		},
	}

	unit, err := domain.NewConsolidationUnit(
		consolidationID,
		orderID,
		waveID,
		domain.StrategyOrderBased,
		items,
	)
	if err != nil {
		return nil, err
	}

	// Set status if different from default
	if status != domain.ConsolidationStatusPending {
		unit.Status = status
	}

	return unit, nil
}

func setupTestRepository(t *testing.T) (*mongodb.ConsolidationRepository, *sharedtesting.MongoDBContainer, func()) {
	ctx := context.Background()

	// Start MongoDB container
	mongoContainer, err := sharedtesting.NewMongoDBContainer(ctx)
	require.NoError(t, err)

	// Get MongoDB client
	client, err := mongoContainer.GetClient(ctx)
	require.NoError(t, err)

	// Create repository
	db := client.Database("test_consolidation_db")
	repo := mongodb.NewConsolidationRepository(db)

	cleanup := func() {
		if err := client.Disconnect(ctx); err != nil {
			t.Logf("Failed to disconnect MongoDB client: %v", err)
		}
		if err := mongoContainer.Close(ctx); err != nil {
			t.Logf("Failed to close MongoDB container: %v", err)
		}
	}

	return repo, mongoContainer, cleanup
}

// TestConsolidationRepository_Save tests the Save operation
func TestConsolidationRepository_Save(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Save new consolidation unit", func(t *testing.T) {
		unit, err := createTestConsolidationUnit("CONS-001", "ORD-001", "WAVE-001", domain.ConsolidationStatusPending)
		require.NoError(t, err)

		err = repo.Save(ctx, unit)
		assert.NoError(t, err)

		// Verify it was saved
		found, err := repo.FindByID(ctx, "CONS-001")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "CONS-001", found.ConsolidationID)
		assert.Equal(t, "ORD-001", found.OrderID)
		assert.Equal(t, domain.ConsolidationStatusPending, found.Status)
	})

	t.Run("Update existing consolidation unit (upsert)", func(t *testing.T) {
		unit, err := createTestConsolidationUnit("CONS-002", "ORD-002", "WAVE-001", domain.ConsolidationStatusPending)
		require.NoError(t, err)

		// Save first time
		err = repo.Save(ctx, unit)
		require.NoError(t, err)

		// Update status and save again
		unit.Status = domain.ConsolidationStatusInProgress
		err = repo.Save(ctx, unit)
		assert.NoError(t, err)

		// Verify update
		found, err := repo.FindByID(ctx, "CONS-002")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, domain.ConsolidationStatusInProgress, found.Status)
	})
}

// TestConsolidationRepository_FindByID tests finding a consolidation unit by ID
func TestConsolidationRepository_FindByID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Find existing consolidation unit", func(t *testing.T) {
		unit, err := createTestConsolidationUnit("CONS-003", "ORD-003", "WAVE-001", domain.ConsolidationStatusPending)
		require.NoError(t, err)

		err = repo.Save(ctx, unit)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, "CONS-003")
		assert.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "CONS-003", found.ConsolidationID)
		assert.Equal(t, 2, len(found.ExpectedItems))
	})

	t.Run("Find non-existent consolidation unit", func(t *testing.T) {
		found, err := repo.FindByID(ctx, "CONS-NONEXISTENT")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

// TestConsolidationRepository_FindByOrderID tests finding a consolidation unit by order ID
func TestConsolidationRepository_FindByOrderID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Find consolidation unit by order ID", func(t *testing.T) {
		unit, err := createTestConsolidationUnit("CONS-004", "ORD-004", "WAVE-001", domain.ConsolidationStatusPending)
		require.NoError(t, err)

		err = repo.Save(ctx, unit)
		require.NoError(t, err)

		found, err := repo.FindByOrderID(ctx, "ORD-004")
		assert.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "ORD-004", found.OrderID)
		assert.Equal(t, "CONS-004", found.ConsolidationID)
	})

	t.Run("Find for non-existent order", func(t *testing.T) {
		found, err := repo.FindByOrderID(ctx, "ORD-NONEXISTENT")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

// TestConsolidationRepository_FindByWaveID tests finding consolidation units by wave ID
func TestConsolidationRepository_FindByWaveID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	waveID := "WAVE-002"

	// Create multiple units for the same wave
	for i := 1; i <= 4; i++ {
		unit, err := createTestConsolidationUnit(
			fmt.Sprintf("CONS-WAVE2-%d", i),
			fmt.Sprintf("ORD-%d", i),
			waveID,
			domain.ConsolidationStatusPending,
		)
		require.NoError(t, err)
		err = repo.Save(ctx, unit)
		require.NoError(t, err)
	}

	t.Run("Find all consolidation units for wave", func(t *testing.T) {
		units, err := repo.FindByWaveID(ctx, waveID)
		assert.NoError(t, err)
		assert.Len(t, units, 4)

		// Verify all units belong to the wave
		for _, unit := range units {
			assert.Equal(t, waveID, unit.WaveID)
		}
	})

	t.Run("Find for non-existent wave", func(t *testing.T) {
		units, err := repo.FindByWaveID(ctx, "WAVE-NONEXISTENT")
		assert.NoError(t, err)
		assert.Empty(t, units)
	})
}

// TestConsolidationRepository_FindByStatus tests finding consolidation units by status
func TestConsolidationRepository_FindByStatus(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create units with different statuses
	statuses := []domain.ConsolidationStatus{
		domain.ConsolidationStatusPending,
		domain.ConsolidationStatusInProgress,
		domain.ConsolidationStatusCompleted,
	}

	for i, status := range statuses {
		unit, err := createTestConsolidationUnit(
			fmt.Sprintf("CONS-STATUS-%d", i+1),
			fmt.Sprintf("ORD-%d", i+1),
			"WAVE-003",
			status,
		)
		require.NoError(t, err)
		err = repo.Save(ctx, unit)
		require.NoError(t, err)
	}

	t.Run("Find consolidation units by status", func(t *testing.T) {
		units, err := repo.FindByStatus(ctx, domain.ConsolidationStatusPending)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(units), 1)

		// Verify all units have the correct status
		for _, unit := range units {
			assert.Equal(t, domain.ConsolidationStatusPending, unit.Status)
		}
	})

	t.Run("Find with no matching status", func(t *testing.T) {
		units, err := repo.FindByStatus(ctx, domain.ConsolidationStatusCancelled)
		assert.NoError(t, err)
		// Could be empty or have some from other tests
		for _, unit := range units {
			assert.Equal(t, domain.ConsolidationStatusCancelled, unit.Status)
		}
	})
}

// TestConsolidationRepository_FindByStation tests finding consolidation units by station
func TestConsolidationRepository_FindByStation(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	station := "STATION-001"

	// Create units assigned to the same station
	for i := 1; i <= 3; i++ {
		unit, err := createTestConsolidationUnit(
			fmt.Sprintf("CONS-STATION-%d", i),
			fmt.Sprintf("ORD-%d", i),
			"WAVE-004",
			domain.ConsolidationStatusPending,
		)
		require.NoError(t, err)

		// Assign to station
		err = unit.AssignStation(station, "WORKER-001", "BIN-001")
		require.NoError(t, err)

		err = repo.Save(ctx, unit)
		require.NoError(t, err)
	}

	t.Run("Find all consolidation units at station", func(t *testing.T) {
		units, err := repo.FindByStation(ctx, station)
		assert.NoError(t, err)
		assert.Len(t, units, 3)

		// Verify all units belong to the station
		for _, unit := range units {
			assert.Equal(t, station, unit.Station)
		}
	})

	t.Run("Find for non-existent station", func(t *testing.T) {
		units, err := repo.FindByStation(ctx, "STATION-NONEXISTENT")
		assert.NoError(t, err)
		assert.Empty(t, units)
	})
}

// TestConsolidationRepository_FindPending tests finding pending consolidation units
func TestConsolidationRepository_FindPending(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create pending units
	for i := 1; i <= 5; i++ {
		unit, err := createTestConsolidationUnit(
			fmt.Sprintf("CONS-PENDING-%d", i),
			fmt.Sprintf("ORD-%d", i),
			"WAVE-005",
			domain.ConsolidationStatusPending,
		)
		require.NoError(t, err)
		err = repo.Save(ctx, unit)
		require.NoError(t, err)
	}

	t.Run("Find all pending units", func(t *testing.T) {
		units, err := repo.FindPending(ctx, 10)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(units), 5)

		// Verify all units are pending
		for _, unit := range units {
			assert.Equal(t, domain.ConsolidationStatusPending, unit.Status)
		}
	})

	t.Run("Find pending with limit", func(t *testing.T) {
		units, err := repo.FindPending(ctx, 3)
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(units), 3)
	})
}

// TestConsolidationRepository_Delete tests deleting a consolidation unit
func TestConsolidationRepository_Delete(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Delete existing consolidation unit", func(t *testing.T) {
		unit, err := createTestConsolidationUnit("CONS-DELETE-001", "ORD-020", "WAVE-007", domain.ConsolidationStatusPending)
		require.NoError(t, err)
		err = repo.Save(ctx, unit)
		require.NoError(t, err)

		// Delete unit
		err = repo.Delete(ctx, "CONS-DELETE-001")
		assert.NoError(t, err)

		// Verify it's deleted
		found, err := repo.FindByID(ctx, "CONS-DELETE-001")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("Delete non-existent consolidation unit", func(t *testing.T) {
		err := repo.Delete(ctx, "CONS-NONEXISTENT")
		// Should not error, just no-op
		assert.NoError(t, err)
	})
}
