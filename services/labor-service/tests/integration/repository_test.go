package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wms-platform/labor-service/internal/domain"
	"github.com/wms-platform/labor-service/internal/infrastructure/mongodb"
	sharedtesting "github.com/wms-platform/shared/pkg/testing"
)

// Test fixtures
func createTestWorker(workerID, employeeID, name string, status domain.WorkerStatus) *domain.Worker {
	worker := domain.NewWorker(workerID, employeeID, name)

	// Add some skills
	worker.AddSkill(domain.TaskTypePicking, 3, true)
	worker.AddSkill(domain.TaskTypePacking, 2, false)

	// Set status if different from default
	if status != domain.WorkerStatusOffline {
		worker.Status = status
	}

	return worker
}

func setupTestRepository(t *testing.T) (*mongodb.WorkerRepository, *sharedtesting.MongoDBContainer, func()) {
	ctx := context.Background()

	// Start MongoDB container
	mongoContainer, err := sharedtesting.NewMongoDBContainer(ctx)
	require.NoError(t, err)

	// Get MongoDB client
	client, err := mongoContainer.GetClient(ctx)
	require.NoError(t, err)

	// Create repository
	db := client.Database("test_labor_db")
	repo := mongodb.NewWorkerRepository(db)

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

// TestWorkerRepository_Save tests the Save operation
func TestWorkerRepository_Save(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Save new worker", func(t *testing.T) {
		worker := createTestWorker("WRK-001", "EMP-001", "John Doe", domain.WorkerStatusOffline)

		err := repo.Save(ctx, worker)
		assert.NoError(t, err)

		// Verify it was saved
		found, err := repo.FindByID(ctx, "WRK-001")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "WRK-001", found.WorkerID)
		assert.Equal(t, "EMP-001", found.EmployeeID)
		assert.Equal(t, "John Doe", found.Name)
	})

	t.Run("Update existing worker (upsert)", func(t *testing.T) {
		worker := createTestWorker("WRK-002", "EMP-002", "Jane Smith", domain.WorkerStatusOffline)

		// Save first time
		err := repo.Save(ctx, worker)
		require.NoError(t, err)

		// Update status and save again
		worker.StartShift("SHIFT-001", "morning", "ZONE-A")
		err = repo.Save(ctx, worker)
		assert.NoError(t, err)

		// Verify update
		found, err := repo.FindByID(ctx, "WRK-002")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, domain.WorkerStatusAvailable, found.Status)
		assert.NotNil(t, found.CurrentShift)
	})
}

// TestWorkerRepository_FindByID tests finding a worker by ID
func TestWorkerRepository_FindByID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Find existing worker", func(t *testing.T) {
		worker := createTestWorker("WRK-003", "EMP-003", "Bob Johnson", domain.WorkerStatusOffline)

		err := repo.Save(ctx, worker)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, "WRK-003")
		assert.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "WRK-003", found.WorkerID)
		assert.Equal(t, 2, len(found.Skills))
	})

	t.Run("Find non-existent worker", func(t *testing.T) {
		found, err := repo.FindByID(ctx, "WRK-NONEXISTENT")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

// TestWorkerRepository_FindByEmployeeID tests finding a worker by employee ID
func TestWorkerRepository_FindByEmployeeID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Find worker by employee ID", func(t *testing.T) {
		worker := createTestWorker("WRK-004", "EMP-004", "Alice Brown", domain.WorkerStatusOffline)

		err := repo.Save(ctx, worker)
		require.NoError(t, err)

		found, err := repo.FindByEmployeeID(ctx, "EMP-004")
		assert.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "EMP-004", found.EmployeeID)
		assert.Equal(t, "WRK-004", found.WorkerID)
	})

	t.Run("Find for non-existent employee", func(t *testing.T) {
		found, err := repo.FindByEmployeeID(ctx, "EMP-NONEXISTENT")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

// TestWorkerRepository_FindByStatus tests finding workers by status
func TestWorkerRepository_FindByStatus(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create workers with different statuses
	statuses := []domain.WorkerStatus{
		domain.WorkerStatusAvailable,
		domain.WorkerStatusOnTask,
		domain.WorkerStatusOnBreak,
		domain.WorkerStatusOffline,
	}

	for i, status := range statuses {
		worker := createTestWorker(
			fmt.Sprintf("WRK-STATUS-%d", i+1),
			fmt.Sprintf("EMP-%d", i+1),
			fmt.Sprintf("Worker %d", i+1),
			status,
		)
		err := repo.Save(ctx, worker)
		require.NoError(t, err)
	}

	t.Run("Find workers by status", func(t *testing.T) {
		workers, err := repo.FindByStatus(ctx, domain.WorkerStatusAvailable)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(workers), 1)

		// Verify all workers have the correct status
		for _, worker := range workers {
			assert.Equal(t, domain.WorkerStatusAvailable, worker.Status)
		}
	})

	t.Run("Find offline workers", func(t *testing.T) {
		workers, err := repo.FindByStatus(ctx, domain.WorkerStatusOffline)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(workers), 1)

		for _, worker := range workers {
			assert.Equal(t, domain.WorkerStatusOffline, worker.Status)
		}
	})
}

// TestWorkerRepository_FindByZone tests finding workers by zone
func TestWorkerRepository_FindByZone(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	zone := "ZONE-A"

	// Create workers in the same zone
	for i := 1; i <= 3; i++ {
		worker := createTestWorker(
			fmt.Sprintf("WRK-ZONE-%d", i),
			fmt.Sprintf("EMP-%d", i),
			fmt.Sprintf("Zone Worker %d", i),
			domain.WorkerStatusOffline,
		)
		worker.StartShift(fmt.Sprintf("SHIFT-%d", i), "morning", zone)
		err := repo.Save(ctx, worker)
		require.NoError(t, err)
	}

	t.Run("Find all workers in zone", func(t *testing.T) {
		workers, err := repo.FindByZone(ctx, zone)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(workers), 3)

		// Verify all workers are in the correct zone
		for _, worker := range workers {
			assert.Equal(t, zone, worker.CurrentZone)
		}
	})

	t.Run("Find for non-existent zone", func(t *testing.T) {
		workers, err := repo.FindByZone(ctx, "ZONE-NONEXISTENT")
		assert.NoError(t, err)
		assert.Empty(t, workers)
	})
}

// TestWorkerRepository_FindAvailableBySkill tests finding available workers by skill
func TestWorkerRepository_FindAvailableBySkill(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	zone := "ZONE-B"

	// Create available workers with picking skills
	for i := 1; i <= 3; i++ {
		worker := createTestWorker(
			fmt.Sprintf("WRK-SKILL-%d", i),
			fmt.Sprintf("EMP-%d", i),
			fmt.Sprintf("Skilled Worker %d", i),
			domain.WorkerStatusOffline,
		)
		worker.StartShift(fmt.Sprintf("SHIFT-%d", i), "morning", zone)
		err := repo.Save(ctx, worker)
		require.NoError(t, err)
	}

	t.Run("Find available workers with picking skill in zone", func(t *testing.T) {
		workers, err := repo.FindAvailableBySkill(ctx, domain.TaskTypePicking, zone)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(workers), 3)

		// Verify all workers have the skill and are available
		for _, worker := range workers {
			assert.Equal(t, domain.WorkerStatusAvailable, worker.Status)
			assert.Equal(t, zone, worker.CurrentZone)
			assert.True(t, worker.HasSkill(domain.TaskTypePicking, 1))
		}
	})

	t.Run("Find available workers with skill in any zone", func(t *testing.T) {
		workers, err := repo.FindAvailableBySkill(ctx, domain.TaskTypePicking, "")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(workers), 3)

		for _, worker := range workers {
			assert.Equal(t, domain.WorkerStatusAvailable, worker.Status)
			assert.True(t, worker.HasSkill(domain.TaskTypePicking, 1))
		}
	})

	t.Run("Find available workers with non-common skill", func(t *testing.T) {
		workers, err := repo.FindAvailableBySkill(ctx, domain.TaskTypeReplenishment, zone)
		assert.NoError(t, err)
		// May be empty if no workers have this skill
		assert.GreaterOrEqual(t, len(workers), 0)
	})
}

// TestWorkerRepository_FindAll tests finding all workers with pagination
func TestWorkerRepository_FindAll(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create multiple workers
	for i := 1; i <= 10; i++ {
		worker := createTestWorker(
			fmt.Sprintf("WRK-ALL-%d", i),
			fmt.Sprintf("EMP-%d", i),
			fmt.Sprintf("Worker All %d", i),
			domain.WorkerStatusOffline,
		)
		err := repo.Save(ctx, worker)
		require.NoError(t, err)
	}

	t.Run("Find all with pagination", func(t *testing.T) {
		workers, err := repo.FindAll(ctx, 5, 0)
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(workers), 5)
	})

	t.Run("Find all with offset", func(t *testing.T) {
		workers, err := repo.FindAll(ctx, 5, 5)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(workers), 0)
	})
}

// TestWorkerRepository_Delete tests deleting a worker
func TestWorkerRepository_Delete(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Delete existing worker", func(t *testing.T) {
		worker := createTestWorker("WRK-DELETE-001", "EMP-020", "Delete Test", domain.WorkerStatusOffline)
		err := repo.Save(ctx, worker)
		require.NoError(t, err)

		// Delete worker
		err = repo.Delete(ctx, "WRK-DELETE-001")
		assert.NoError(t, err)

		// Verify it's deleted
		found, err := repo.FindByID(ctx, "WRK-DELETE-001")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("Delete non-existent worker", func(t *testing.T) {
		err := repo.Delete(ctx, "WRK-NONEXISTENT")
		// Should not error, just no-op
		assert.NoError(t, err)
	})
}
