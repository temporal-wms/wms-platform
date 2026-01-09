package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wms-platform/picking-service/internal/domain"
	"github.com/wms-platform/picking-service/internal/infrastructure/mongodb"
	sharedtesting "github.com/wms-platform/shared/pkg/testing"
)

// Test fixtures
func createTestPickTask(taskID, orderID, waveID, routeID string, method domain.PickMethod, status domain.PickTaskStatus) *domain.PickTask {
	items := []domain.PickItem{
		{
			SKU:         "SKU-001",
			ProductName: "Product 001",
			Quantity:    5,
			PickedQty:   0,
			Location: domain.Location{
				LocationID: "A-12-3-B",
				Aisle:      "A",
				Rack:       12,
				Level:      3,
				Position:   "B",
				Zone:       "ZONE-A",
			},
			Status: "pending",
		},
		{
			SKU:         "SKU-002",
			ProductName: "Product 002",
			Quantity:    3,
			PickedQty:   0,
			Location: domain.Location{
				LocationID: "A-15-2-A",
				Aisle:      "A",
				Rack:       15,
				Level:      2,
				Position:   "A",
				Zone:       "ZONE-A",
			},
			Status: "pending",
		},
	}

	task, _ := domain.NewPickTask(taskID, orderID, waveID, routeID, method, items)

	// Set status if different from default
	if status != domain.PickTaskStatusPending {
		task.Status = status
	}

	return task
}

func setupTestRepository(t *testing.T) (*mongodb.PickTaskRepository, *sharedtesting.MongoDBContainer, func()) {
	ctx := context.Background()

	// Start MongoDB container
	mongoContainer, err := sharedtesting.NewMongoDBContainer(ctx)
	require.NoError(t, err)

	// Get MongoDB client
	client, err := mongoContainer.GetClient(ctx)
	require.NoError(t, err)

	// Create database and repository
	db := client.Database("pick_tasks_test")
	repo := mongodb.NewPickTaskRepository(db)

	// Cleanup function
	cleanup := func() {
		client.Disconnect(ctx)
		mongoContainer.Close(ctx)
	}

	return repo, mongoContainer, cleanup
}

// TestPickTaskRepository_Save tests pick task saving
func TestPickTaskRepository_Save(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Save new pick task", func(t *testing.T) {
		task := createTestPickTask("TASK-001", "ORD-001", "WAVE-001", "ROUTE-001", domain.PickMethodSingle, domain.PickTaskStatusPending)

		err := repo.Save(ctx, task)
		assert.NoError(t, err)

		// Verify task was saved
		found, err := repo.FindByID(ctx, "TASK-001")
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, "TASK-001", found.TaskID)
		assert.Equal(t, domain.PickMethodSingle, found.Method)
	})

	t.Run("Update existing pick task", func(t *testing.T) {
		task := createTestPickTask("TASK-002", "ORD-002", "WAVE-001", "ROUTE-002", domain.PickMethodBatch, domain.PickTaskStatusPending)
		err := repo.Save(ctx, task)
		assert.NoError(t, err)

		// Update task
		task.Status = domain.PickTaskStatusAssigned
		task.PickerID = "PICKER-001"
		err = repo.Save(ctx, task)
		assert.NoError(t, err)

		// Verify update
		found, err := repo.FindByID(ctx, "TASK-002")
		assert.NoError(t, err)
		assert.Equal(t, domain.PickTaskStatusAssigned, found.Status)
		assert.Equal(t, "PICKER-001", found.PickerID)
	})
}

// TestPickTaskRepository_FindByID tests finding pick task by ID
func TestPickTaskRepository_FindByID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Find existing pick task", func(t *testing.T) {
		task := createTestPickTask("TASK-001", "ORD-001", "WAVE-001", "ROUTE-001", domain.PickMethodSingle, domain.PickTaskStatusPending)
		err := repo.Save(ctx, task)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, "TASK-001")
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, "TASK-001", found.TaskID)
	})

	t.Run("Find non-existent pick task", func(t *testing.T) {
		found, err := repo.FindByID(ctx, "TASK-999")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

// TestPickTaskRepository_FindByOrderID tests finding pick tasks by order ID
func TestPickTaskRepository_FindByOrderID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create tasks for the same order
	task1 := createTestPickTask("TASK-001", "ORD-001", "WAVE-001", "ROUTE-001", domain.PickMethodSingle, domain.PickTaskStatusPending)
	task2 := createTestPickTask("TASK-002", "ORD-001", "WAVE-001", "ROUTE-002", domain.PickMethodSingle, domain.PickTaskStatusPending)
	task3 := createTestPickTask("TASK-003", "ORD-002", "WAVE-001", "ROUTE-003", domain.PickMethodSingle, domain.PickTaskStatusPending)

	require.NoError(t, repo.Save(ctx, task1))
	require.NoError(t, repo.Save(ctx, task2))
	require.NoError(t, repo.Save(ctx, task3))

	tasks, err := repo.FindByOrderID(ctx, "ORD-001")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(tasks), 2)
	for _, task := range tasks {
		assert.Equal(t, "ORD-001", task.OrderID)
	}
}

// TestPickTaskRepository_FindByWaveID tests finding pick tasks by wave ID
func TestPickTaskRepository_FindByWaveID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create tasks for the same wave
	task1 := createTestPickTask("TASK-001", "ORD-001", "WAVE-001", "ROUTE-001", domain.PickMethodWave, domain.PickTaskStatusPending)
	task2 := createTestPickTask("TASK-002", "ORD-002", "WAVE-001", "ROUTE-002", domain.PickMethodWave, domain.PickTaskStatusPending)
	task3 := createTestPickTask("TASK-003", "ORD-003", "WAVE-002", "ROUTE-003", domain.PickMethodWave, domain.PickTaskStatusPending)

	require.NoError(t, repo.Save(ctx, task1))
	require.NoError(t, repo.Save(ctx, task2))
	require.NoError(t, repo.Save(ctx, task3))

	tasks, err := repo.FindByWaveID(ctx, "WAVE-001")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(tasks), 2)
	for _, task := range tasks {
		assert.Equal(t, "WAVE-001", task.WaveID)
	}
}

// TestPickTaskRepository_FindByPickerID tests finding pick tasks by picker ID
func TestPickTaskRepository_FindByPickerID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create tasks with different pickers
	task1 := createTestPickTask("TASK-001", "ORD-001", "WAVE-001", "ROUTE-001", domain.PickMethodSingle, domain.PickTaskStatusAssigned)
	task1.PickerID = "PICKER-001"
	task2 := createTestPickTask("TASK-002", "ORD-002", "WAVE-001", "ROUTE-002", domain.PickMethodSingle, domain.PickTaskStatusAssigned)
	task2.PickerID = "PICKER-001"
	task3 := createTestPickTask("TASK-003", "ORD-003", "WAVE-001", "ROUTE-003", domain.PickMethodSingle, domain.PickTaskStatusAssigned)
	task3.PickerID = "PICKER-002"

	require.NoError(t, repo.Save(ctx, task1))
	require.NoError(t, repo.Save(ctx, task2))
	require.NoError(t, repo.Save(ctx, task3))

	tasks, err := repo.FindByPickerID(ctx, "PICKER-001")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(tasks), 2)
	for _, task := range tasks {
		assert.Equal(t, "PICKER-001", task.PickerID)
	}
}

// TestPickTaskRepository_FindByStatus tests finding pick tasks by status
func TestPickTaskRepository_FindByStatus(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create tasks with different statuses
	task1 := createTestPickTask("TASK-001", "ORD-001", "WAVE-001", "ROUTE-001", domain.PickMethodSingle, domain.PickTaskStatusPending)
	task2 := createTestPickTask("TASK-002", "ORD-002", "WAVE-001", "ROUTE-002", domain.PickMethodSingle, domain.PickTaskStatusPending)
	task3 := createTestPickTask("TASK-003", "ORD-003", "WAVE-001", "ROUTE-003", domain.PickMethodSingle, domain.PickTaskStatusInProgress)

	require.NoError(t, repo.Save(ctx, task1))
	require.NoError(t, repo.Save(ctx, task2))
	require.NoError(t, repo.Save(ctx, task3))

	t.Run("Find pending tasks", func(t *testing.T) {
		tasks, err := repo.FindByStatus(ctx, domain.PickTaskStatusPending)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(tasks), 2)
		for _, task := range tasks {
			assert.Equal(t, domain.PickTaskStatusPending, task.Status)
		}
	})

	t.Run("Find in-progress tasks", func(t *testing.T) {
		tasks, err := repo.FindByStatus(ctx, domain.PickTaskStatusInProgress)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(tasks), 1)
		for _, task := range tasks {
			assert.Equal(t, domain.PickTaskStatusInProgress, task.Status)
		}
	})
}

// TestPickTaskRepository_FindActiveByPicker tests finding active pick task for a picker
func TestPickTaskRepository_FindActiveByPicker(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create an active task
	task := createTestPickTask("TASK-001", "ORD-001", "WAVE-001", "ROUTE-001", domain.PickMethodSingle, domain.PickTaskStatusInProgress)
	task.PickerID = "PICKER-001"
	require.NoError(t, repo.Save(ctx, task))

	t.Run("Find active task for picker", func(t *testing.T) {
		found, err := repo.FindActiveByPicker(ctx, "PICKER-001")
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, "TASK-001", found.TaskID)
		assert.Equal(t, "PICKER-001", found.PickerID)
		assert.Equal(t, domain.PickTaskStatusInProgress, found.Status)
	})

	t.Run("Find active task for picker with no active tasks", func(t *testing.T) {
		found, err := repo.FindActiveByPicker(ctx, "PICKER-999")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

// TestPickTaskRepository_FindPendingByZone tests finding pending pick tasks by zone
func TestPickTaskRepository_FindPendingByZone(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create pending tasks in different zones
	task1 := createTestPickTask("TASK-001", "ORD-001", "WAVE-001", "ROUTE-001", domain.PickMethodSingle, domain.PickTaskStatusPending)
	task1.Zone = "ZONE-A"
	task2 := createTestPickTask("TASK-002", "ORD-002", "WAVE-001", "ROUTE-002", domain.PickMethodSingle, domain.PickTaskStatusPending)
	task2.Zone = "ZONE-A"
	task3 := createTestPickTask("TASK-003", "ORD-003", "WAVE-001", "ROUTE-003", domain.PickMethodSingle, domain.PickTaskStatusPending)
	task3.Zone = "ZONE-B"

	require.NoError(t, repo.Save(ctx, task1))
	require.NoError(t, repo.Save(ctx, task2))
	require.NoError(t, repo.Save(ctx, task3))

	t.Run("Find pending tasks for zone", func(t *testing.T) {
		tasks, err := repo.FindPendingByZone(ctx, "ZONE-A", 10)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(tasks), 2)
		for _, task := range tasks {
			assert.Equal(t, domain.PickTaskStatusPending, task.Status)
			assert.Equal(t, "ZONE-A", task.Zone)
		}
	})

	t.Run("Find pending tasks with limit", func(t *testing.T) {
		tasks, err := repo.FindPendingByZone(ctx, "", 2)
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(tasks), 2)
	})
}

// TestPickTaskRepository_Delete tests pick task deletion
func TestPickTaskRepository_Delete(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create task
	task := createTestPickTask("TASK-001", "ORD-001", "WAVE-001", "ROUTE-001", domain.PickMethodSingle, domain.PickTaskStatusPending)
	require.NoError(t, repo.Save(ctx, task))

	// Delete task
	err := repo.Delete(ctx, "TASK-001")
	assert.NoError(t, err)

	// Verify deletion
	found, err := repo.FindByID(ctx, "TASK-001")
	assert.NoError(t, err)
	assert.Nil(t, found)
}
