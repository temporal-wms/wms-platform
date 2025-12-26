package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wms-platform/packing-service/internal/domain"
	"github.com/wms-platform/packing-service/internal/infrastructure/mongodb"
	"github.com/wms-platform/shared/pkg/cloudevents"
	sharedtesting "github.com/wms-platform/shared/pkg/testing"
)

// Test fixtures
func createTestPackTask(taskID, orderID, waveID string, status domain.PackTaskStatus) (*domain.PackTask, error) {
	items := []domain.PackItem{
		{
			SKU:         "SKU-001",
			ProductName: "Test Product 1",
			Quantity:    2,
			Weight:      0.5,
			Fragile:     false,
			Verified:    false,
		},
		{
			SKU:         "SKU-002",
			ProductName: "Test Product 2",
			Quantity:    1,
			Weight:      1.2,
			Fragile:     true,
			Verified:    false,
		},
	}

	task, err := domain.NewPackTask(taskID, orderID, waveID, items)
	if err != nil {
		return nil, err
	}

	// Set status if different from default
	if status != domain.PackTaskStatusPending {
		task.Status = status
	}

	return task, nil
}

func setupTestRepository(t *testing.T) (*mongodb.PackTaskRepository, *sharedtesting.MongoDBContainer, func()) {
	ctx := context.Background()

	// Start MongoDB container
	mongoContainer, err := sharedtesting.NewMongoDBContainer(ctx)
	require.NoError(t, err)

	// Get MongoDB client
	client, err := mongoContainer.GetClient(ctx)
	require.NoError(t, err)

	// Initialize CloudEvents factory
	eventFactory := cloudevents.NewEventFactory("/packing-service")

	// Create repository
	db := client.Database("test_packing_db")
	repo := mongodb.NewPackTaskRepository(db, eventFactory)

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

// TestPackTaskRepository_Save tests the Save operation
func TestPackTaskRepository_Save(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Save new pack task", func(t *testing.T) {
		task, err := createTestPackTask("PACK-001", "ORD-001", "WAVE-001", domain.PackTaskStatusPending)
		require.NoError(t, err)

		err = repo.Save(ctx, task)
		assert.NoError(t, err)

		// Verify it was saved
		found, err := repo.FindByID(ctx, "PACK-001")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "PACK-001", found.TaskID)
		assert.Equal(t, "ORD-001", found.OrderID)
		assert.Equal(t, domain.PackTaskStatusPending, found.Status)
	})

	t.Run("Update existing pack task (upsert)", func(t *testing.T) {
		task, err := createTestPackTask("PACK-002", "ORD-002", "WAVE-001", domain.PackTaskStatusPending)
		require.NoError(t, err)

		// Save first time
		err = repo.Save(ctx, task)
		require.NoError(t, err)

		// Update status and save again
		task.Status = domain.PackTaskStatusInProgress
		err = repo.Save(ctx, task)
		assert.NoError(t, err)

		// Verify update
		found, err := repo.FindByID(ctx, "PACK-002")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, domain.PackTaskStatusInProgress, found.Status)
	})
}

// TestPackTaskRepository_FindByID tests finding a pack task by ID
func TestPackTaskRepository_FindByID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Find existing pack task", func(t *testing.T) {
		task, err := createTestPackTask("PACK-003", "ORD-003", "WAVE-001", domain.PackTaskStatusPending)
		require.NoError(t, err)

		err = repo.Save(ctx, task)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, "PACK-003")
		assert.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "PACK-003", found.TaskID)
		assert.Equal(t, 2, len(found.Items))
	})

	t.Run("Find non-existent pack task", func(t *testing.T) {
		found, err := repo.FindByID(ctx, "PACK-NONEXISTENT")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

// TestPackTaskRepository_FindByOrderID tests finding a pack task by order ID
func TestPackTaskRepository_FindByOrderID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Find pack task by order ID", func(t *testing.T) {
		task, err := createTestPackTask("PACK-004", "ORD-004", "WAVE-001", domain.PackTaskStatusPending)
		require.NoError(t, err)

		err = repo.Save(ctx, task)
		require.NoError(t, err)

		found, err := repo.FindByOrderID(ctx, "ORD-004")
		assert.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "ORD-004", found.OrderID)
		assert.Equal(t, "PACK-004", found.TaskID)
	})

	t.Run("Find for non-existent order", func(t *testing.T) {
		found, err := repo.FindByOrderID(ctx, "ORD-NONEXISTENT")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

// TestPackTaskRepository_FindByWaveID tests finding pack tasks by wave ID
func TestPackTaskRepository_FindByWaveID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	waveID := "WAVE-002"

	// Create multiple tasks for the same wave
	for i := 1; i <= 4; i++ {
		task, err := createTestPackTask(
			fmt.Sprintf("PACK-WAVE2-%d", i),
			fmt.Sprintf("ORD-%d", i),
			waveID,
			domain.PackTaskStatusPending,
		)
		require.NoError(t, err)
		err = repo.Save(ctx, task)
		require.NoError(t, err)
	}

	t.Run("Find all pack tasks for wave", func(t *testing.T) {
		tasks, err := repo.FindByWaveID(ctx, waveID)
		assert.NoError(t, err)
		assert.Len(t, tasks, 4)

		// Verify all tasks belong to the wave
		for _, task := range tasks {
			assert.Equal(t, waveID, task.WaveID)
		}
	})

	t.Run("Find for non-existent wave", func(t *testing.T) {
		tasks, err := repo.FindByWaveID(ctx, "WAVE-NONEXISTENT")
		assert.NoError(t, err)
		assert.Empty(t, tasks)
	})
}

// TestPackTaskRepository_FindByPackerID tests finding pack tasks by packer ID
func TestPackTaskRepository_FindByPackerID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	packerID := "PACKER-001"

	// Create tasks and assign to packer
	for i := 1; i <= 3; i++ {
		task, err := createTestPackTask(
			fmt.Sprintf("PACK-PACKER1-%d", i),
			fmt.Sprintf("ORD-%d", i),
			"WAVE-003",
			domain.PackTaskStatusPending,
		)
		require.NoError(t, err)

		// Assign to packer
		err = task.Assign(packerID, "STATION-001")
		require.NoError(t, err)

		err = repo.Save(ctx, task)
		require.NoError(t, err)
	}

	t.Run("Find all pack tasks for packer", func(t *testing.T) {
		tasks, err := repo.FindByPackerID(ctx, packerID)
		assert.NoError(t, err)
		assert.Len(t, tasks, 3)

		// Verify all tasks belong to the packer
		for _, task := range tasks {
			assert.Equal(t, packerID, task.PackerID)
		}
	})

	t.Run("Find for non-existent packer", func(t *testing.T) {
		tasks, err := repo.FindByPackerID(ctx, "PACKER-NONEXISTENT")
		assert.NoError(t, err)
		assert.Empty(t, tasks)
	})
}

// TestPackTaskRepository_FindByStatus tests finding pack tasks by status
func TestPackTaskRepository_FindByStatus(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create tasks with different statuses
	statuses := []domain.PackTaskStatus{
		domain.PackTaskStatusPending,
		domain.PackTaskStatusInProgress,
		domain.PackTaskStatusPacked,
		domain.PackTaskStatusCompleted,
	}

	for i, status := range statuses {
		task, err := createTestPackTask(
			fmt.Sprintf("PACK-STATUS-%d", i+1),
			fmt.Sprintf("ORD-%d", i+1),
			"WAVE-004",
			status,
		)
		require.NoError(t, err)
		err = repo.Save(ctx, task)
		require.NoError(t, err)
	}

	t.Run("Find pack tasks by status", func(t *testing.T) {
		tasks, err := repo.FindByStatus(ctx, domain.PackTaskStatusPending)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(tasks), 1)

		// Verify all tasks have the correct status
		for _, task := range tasks {
			assert.Equal(t, domain.PackTaskStatusPending, task.Status)
		}
	})

	t.Run("Find with no matching status", func(t *testing.T) {
		tasks, err := repo.FindByStatus(ctx, domain.PackTaskStatusCancelled)
		assert.NoError(t, err)
		// Could be empty or have some from other tests
		for _, task := range tasks {
			assert.Equal(t, domain.PackTaskStatusCancelled, task.Status)
		}
	})
}

// TestPackTaskRepository_FindByStation tests finding pack tasks by station
func TestPackTaskRepository_FindByStation(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	station := "STATION-002"

	// Create tasks assigned to the same station
	for i := 1; i <= 3; i++ {
		task, err := createTestPackTask(
			fmt.Sprintf("PACK-STATION-%d", i),
			fmt.Sprintf("ORD-%d", i),
			"WAVE-005",
			domain.PackTaskStatusPending,
		)
		require.NoError(t, err)

		// Assign to station
		err = task.Assign("PACKER-002", station)
		require.NoError(t, err)

		err = repo.Save(ctx, task)
		require.NoError(t, err)
	}

	t.Run("Find all pack tasks at station", func(t *testing.T) {
		tasks, err := repo.FindByStation(ctx, station)
		assert.NoError(t, err)
		assert.Len(t, tasks, 3)

		// Verify all tasks belong to the station
		for _, task := range tasks {
			assert.Equal(t, station, task.Station)
		}
	})

	t.Run("Find for non-existent station", func(t *testing.T) {
		tasks, err := repo.FindByStation(ctx, "STATION-NONEXISTENT")
		assert.NoError(t, err)
		assert.Empty(t, tasks)
	})
}

// TestPackTaskRepository_FindPending tests finding pending pack tasks
func TestPackTaskRepository_FindPending(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create pending tasks with different priorities
	for i := 1; i <= 5; i++ {
		task, err := createTestPackTask(
			fmt.Sprintf("PACK-PENDING-%d", i),
			fmt.Sprintf("ORD-%d", i),
			"WAVE-006",
			domain.PackTaskStatusPending,
		)
		require.NoError(t, err)
		task.Priority = i
		err = repo.Save(ctx, task)
		require.NoError(t, err)
	}

	t.Run("Find all pending tasks", func(t *testing.T) {
		tasks, err := repo.FindPending(ctx, 10)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(tasks), 5)

		// Verify all tasks are pending
		for _, task := range tasks {
			assert.Equal(t, domain.PackTaskStatusPending, task.Status)
		}
	})

	t.Run("Find pending with limit", func(t *testing.T) {
		tasks, err := repo.FindPending(ctx, 3)
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(tasks), 3)
	})
}

// TestPackTaskRepository_FindByTrackingNumber tests finding a pack task by tracking number
func TestPackTaskRepository_FindByTrackingNumber(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Find pack task by tracking number", func(t *testing.T) {
		task, err := createTestPackTask("PACK-TRACK-001", "ORD-010", "WAVE-007", domain.PackTaskStatusPending)
		require.NoError(t, err)

		// Add shipping label
		label := domain.ShippingLabel{
			TrackingNumber: "TRACK-12345",
			Carrier:        "UPS",
			ServiceType:    "Ground",
			GeneratedAt:    time.Now(),
		}
		task.ShippingLabel = &label

		err = repo.Save(ctx, task)
		require.NoError(t, err)

		found, err := repo.FindByTrackingNumber(ctx, "TRACK-12345")
		assert.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "TRACK-12345", found.ShippingLabel.TrackingNumber)
		assert.Equal(t, "PACK-TRACK-001", found.TaskID)
	})

	t.Run("Find for non-existent tracking number", func(t *testing.T) {
		found, err := repo.FindByTrackingNumber(ctx, "TRACK-NONEXISTENT")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

// TestPackTaskRepository_Delete tests deleting a pack task
func TestPackTaskRepository_Delete(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Delete existing pack task", func(t *testing.T) {
		task, err := createTestPackTask("PACK-DELETE-001", "ORD-020", "WAVE-008", domain.PackTaskStatusPending)
		require.NoError(t, err)
		err = repo.Save(ctx, task)
		require.NoError(t, err)

		// Delete task
		err = repo.Delete(ctx, "PACK-DELETE-001")
		assert.NoError(t, err)

		// Verify it's deleted
		found, err := repo.FindByID(ctx, "PACK-DELETE-001")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("Delete non-existent pack task", func(t *testing.T) {
		err := repo.Delete(ctx, "PACK-NONEXISTENT")
		// Should not error, just no-op
		assert.NoError(t, err)
	})
}
