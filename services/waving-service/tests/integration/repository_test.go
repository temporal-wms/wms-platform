package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wms-platform/waving-service/internal/domain"
	"github.com/wms-platform/waving-service/internal/infrastructure/mongodb"
	sharedtesting "github.com/wms-platform/shared/pkg/testing"
)

// Test fixtures
func createTestWave(waveID string, waveType domain.WaveType, status domain.WaveStatus) *domain.Wave {
	config := domain.WaveConfiguration{
		MaxOrders:  50,
		MaxItems:   200,
		MaxWeight:  500.0,
		AutoRelease: false,
	}

	wave, _ := domain.NewWave(
		waveID,
		waveType,
		domain.FulfillmentModeWave,
		config,
	)

	// Set status if different from default
	if status != domain.WaveStatusPlanning {
		wave.Status = status
	}

	return wave
}

func setupTestRepository(t *testing.T) (*mongodb.WaveRepository, *sharedtesting.MongoDBContainer, func()) {
	ctx := context.Background()

	// Start MongoDB container
	mongoContainer, err := sharedtesting.NewMongoDBContainer(ctx)
	require.NoError(t, err)

	// Get MongoDB client
	client, err := mongoContainer.GetClient(ctx)
	require.NoError(t, err)

	// Create database and repository
	db := client.Database("waves_test")
	repo := mongodb.NewWaveRepository(db)

	// Cleanup function
	cleanup := func() {
		client.Disconnect(ctx)
		mongoContainer.Close(ctx)
	}

	return repo, mongoContainer, cleanup
}

// TestWaveRepository_Save tests wave saving
func TestWaveRepository_Save(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Save new wave", func(t *testing.T) {
		wave := createTestWave("WAVE-001", domain.WaveTypeDigital, domain.WaveStatusPlanning)

		err := repo.Save(ctx, wave)
		assert.NoError(t, err)

		// Verify wave was saved
		found, err := repo.FindByID(ctx, "WAVE-001")
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, "WAVE-001", found.WaveID)
		assert.Equal(t, domain.WaveTypeDigital, found.WaveType)
	})

	t.Run("Update existing wave", func(t *testing.T) {
		wave := createTestWave("WAVE-002", domain.WaveTypeDigital, domain.WaveStatusPlanning)
		err := repo.Save(ctx, wave)
		assert.NoError(t, err)

		// Update wave
		wave.Status = domain.WaveStatusScheduled
		err = repo.Save(ctx, wave)
		assert.NoError(t, err)

		// Verify update
		found, err := repo.FindByID(ctx, "WAVE-002")
		assert.NoError(t, err)
		assert.Equal(t, domain.WaveStatusScheduled, found.Status)
	})
}

// TestWaveRepository_FindByID tests finding wave by ID
func TestWaveRepository_FindByID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Find existing wave", func(t *testing.T) {
		wave := createTestWave("WAVE-001", domain.WaveTypeDigital, domain.WaveStatusPlanning)
		err := repo.Save(ctx, wave)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, "WAVE-001")
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, "WAVE-001", found.WaveID)
	})

	t.Run("Find non-existent wave", func(t *testing.T) {
		found, err := repo.FindByID(ctx, "WAVE-999")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

// TestWaveRepository_FindByStatus tests finding waves by status
func TestWaveRepository_FindByStatus(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create waves with different statuses
	wave1 := createTestWave("WAVE-001", domain.WaveTypeDigital, domain.WaveStatusPlanning)
	wave2 := createTestWave("WAVE-002", domain.WaveTypeDigital, domain.WaveStatusPlanning)
	wave3 := createTestWave("WAVE-003", domain.WaveTypeDigital, domain.WaveStatusScheduled)

	require.NoError(t, repo.Save(ctx, wave1))
	require.NoError(t, repo.Save(ctx, wave2))
	require.NoError(t, repo.Save(ctx, wave3))

	t.Run("Find waves by planning status", func(t *testing.T) {
		waves, err := repo.FindByStatus(ctx, domain.WaveStatusPlanning)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(waves), 2)
		for _, wave := range waves {
			assert.Equal(t, domain.WaveStatusPlanning, wave.Status)
		}
	})

	t.Run("Find waves by scheduled status", func(t *testing.T) {
		waves, err := repo.FindByStatus(ctx, domain.WaveStatusScheduled)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(waves), 1)
		for _, wave := range waves {
			assert.Equal(t, domain.WaveStatusScheduled, wave.Status)
		}
	})
}

// TestWaveRepository_FindByType tests finding waves by type
func TestWaveRepository_FindByType(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create waves with different types
	wave1 := createTestWave("WAVE-001", domain.WaveTypeDigital, domain.WaveStatusPlanning)
	wave2 := createTestWave("WAVE-002", domain.WaveTypeDigital, domain.WaveStatusPlanning)
	wave3 := createTestWave("WAVE-003", domain.WaveTypeWholesale, domain.WaveStatusPlanning)

	require.NoError(t, repo.Save(ctx, wave1))
	require.NoError(t, repo.Save(ctx, wave2))
	require.NoError(t, repo.Save(ctx, wave3))

	waves, err := repo.FindByType(ctx, domain.WaveTypeDigital)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(waves), 2)
	for _, wave := range waves {
		assert.Equal(t, domain.WaveTypeDigital, wave.WaveType)
	}
}

// TestWaveRepository_FindByZone tests finding waves by zone
func TestWaveRepository_FindByZone(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create waves with different zones
	wave1 := createTestWave("WAVE-001", domain.WaveTypeDigital, domain.WaveStatusPlanning)
	wave1.Zone = "ZONE-A"
	wave2 := createTestWave("WAVE-002", domain.WaveTypeDigital, domain.WaveStatusPlanning)
	wave2.Zone = "ZONE-A"
	wave3 := createTestWave("WAVE-003", domain.WaveTypeDigital, domain.WaveStatusPlanning)
	wave3.Zone = "ZONE-B"

	require.NoError(t, repo.Save(ctx, wave1))
	require.NoError(t, repo.Save(ctx, wave2))
	require.NoError(t, repo.Save(ctx, wave3))

	waves, err := repo.FindByZone(ctx, "ZONE-A")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(waves), 2)
	for _, wave := range waves {
		assert.Equal(t, "ZONE-A", wave.Zone)
	}
}

// TestWaveRepository_FindByOrderID tests finding wave by order ID
func TestWaveRepository_FindByOrderID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create wave and add order
	wave := createTestWave("WAVE-001", domain.WaveTypeDigital, domain.WaveStatusPlanning)
	order := domain.WaveOrder{
		OrderID:    "ORD-001",
		Priority:   "high",
		ItemCount:  5,
		TotalWeight: 10.5,
	}
	wave.AddOrder(order)
	require.NoError(t, repo.Save(ctx, wave))

	// Find wave by order ID
	found, err := repo.FindByOrderID(ctx, "ORD-001")
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "WAVE-001", found.WaveID)
}

// TestWaveRepository_Delete tests wave deletion
func TestWaveRepository_Delete(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create wave
	wave := createTestWave("WAVE-001", domain.WaveTypeDigital, domain.WaveStatusPlanning)
	require.NoError(t, repo.Save(ctx, wave))

	// Delete wave
	err := repo.Delete(ctx, "WAVE-001")
	assert.NoError(t, err)

	// Verify deletion
	found, err := repo.FindByID(ctx, "WAVE-001")
	assert.NoError(t, err)
	assert.Nil(t, found)
}

// TestWaveRepository_FindActive tests finding active waves
func TestWaveRepository_FindActive(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create waves with different statuses
	wave1 := createTestWave("WAVE-001", domain.WaveTypeDigital, domain.WaveStatusPlanning)
	wave2 := createTestWave("WAVE-002", domain.WaveTypeDigital, domain.WaveStatusScheduled)
	wave3 := createTestWave("WAVE-003", domain.WaveTypeDigital, domain.WaveStatusReleased)
	wave4 := createTestWave("WAVE-004", domain.WaveTypeDigital, domain.WaveStatusCompleted)

	require.NoError(t, repo.Save(ctx, wave1))
	require.NoError(t, repo.Save(ctx, wave2))
	require.NoError(t, repo.Save(ctx, wave3))
	require.NoError(t, repo.Save(ctx, wave4))

	waves, err := repo.FindActive(ctx)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(waves), 3) // planning, scheduled, released

	// Verify completed wave is not included
	for _, w := range waves {
		assert.NotEqual(t, domain.WaveStatusCompleted, w.Status)
		assert.NotEqual(t, domain.WaveStatusCancelled, w.Status)
	}
}

// TestWaveRepository_Count tests wave counting
func TestWaveRepository_Count(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create waves
	wave1 := createTestWave("WAVE-001", domain.WaveTypeDigital, domain.WaveStatusPlanning)
	wave2 := createTestWave("WAVE-002", domain.WaveTypeDigital, domain.WaveStatusPlanning)
	wave3 := createTestWave("WAVE-003", domain.WaveTypeDigital, domain.WaveStatusScheduled)

	require.NoError(t, repo.Save(ctx, wave1))
	require.NoError(t, repo.Save(ctx, wave2))
	require.NoError(t, repo.Save(ctx, wave3))

	count, err := repo.Count(ctx, domain.WaveStatusPlanning)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(2))
}
