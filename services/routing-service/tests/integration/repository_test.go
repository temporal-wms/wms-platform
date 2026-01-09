package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wms-platform/routing-service/internal/domain"
	"github.com/wms-platform/routing-service/internal/infrastructure/mongodb"
	sharedtesting "github.com/wms-platform/shared/pkg/testing"
)

// Test fixtures
func createTestPickRoute(routeID, orderID, waveID string, status domain.RouteStatus) (*domain.PickRoute, error) {
	items := []domain.RouteItem{
		{
			SKU:      "SKU-001",
			Quantity: 5,
			Location: domain.Location{
				LocationID: "A-10-2-A",
				Aisle:      "A",
				Rack:       10,
				Level:      2,
				Position:   "A",
				Zone:       "ZONE-A",
				X:          10.0,
				Y:          20.0,
			},
		},
		{
			SKU:      "SKU-002",
			Quantity: 3,
			Location: domain.Location{
				LocationID: "A-15-3-B",
				Aisle:      "A",
				Rack:       15,
				Level:      3,
				Position:   "B",
				Zone:       "ZONE-A",
				X:          15.0,
				Y:          30.0,
			},
		},
		{
			SKU:      "SKU-003",
			Quantity: 2,
			Location: domain.Location{
				LocationID: "B-05-1-A",
				Aisle:      "B",
				Rack:       5,
				Level:      1,
				Position:   "A",
				Zone:       "ZONE-A",
				X:          5.0,
				Y:          10.0,
			},
		},
	}

	route, err := domain.NewPickRoute(routeID, orderID, waveID, domain.StrategyReturn, items)
	if err != nil {
		return nil, err
	}

	// Set status if different from default
	if status != domain.RouteStatusPending {
		route.Status = status
	}

	return route, nil
}

func setupTestRepository(t *testing.T) (*mongodb.RouteRepository, *sharedtesting.MongoDBContainer, func()) {
	ctx := context.Background()

	// Start MongoDB container
	mongoContainer, err := sharedtesting.NewMongoDBContainer(ctx)
	require.NoError(t, err)

	// Get MongoDB client
	client, err := mongoContainer.GetClient(ctx)
	require.NoError(t, err)

	// Create repository
	db := client.Database("test_routing_db")
	repo := mongodb.NewRouteRepository(db)

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

// TestRouteRepository_Save tests the Save operation
func TestRouteRepository_Save(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Save new pick route", func(t *testing.T) {
		route, err := createTestPickRoute("ROUTE-001", "ORD-001", "WAVE-001", domain.RouteStatusPending)
		require.NoError(t, err)

		err = repo.Save(ctx, route)
		assert.NoError(t, err)

		// Verify it was saved
		found, err := repo.FindByID(ctx, "ROUTE-001")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "ROUTE-001", found.RouteID)
		assert.Equal(t, "ORD-001", found.OrderID)
		assert.Equal(t, domain.RouteStatusPending, found.Status)
	})

	t.Run("Update existing pick route (upsert)", func(t *testing.T) {
		route, err := createTestPickRoute("ROUTE-002", "ORD-002", "WAVE-001", domain.RouteStatusPending)
		require.NoError(t, err)

		// Save first time
		err = repo.Save(ctx, route)
		require.NoError(t, err)

		// Update status and save again
		route.Status = domain.RouteStatusInProgress
		err = repo.Save(ctx, route)
		assert.NoError(t, err)

		// Verify update
		found, err := repo.FindByID(ctx, "ROUTE-002")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, domain.RouteStatusInProgress, found.Status)
	})
}

// TestRouteRepository_FindByID tests finding a pick route by ID
func TestRouteRepository_FindByID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Find existing pick route", func(t *testing.T) {
		route, err := createTestPickRoute("ROUTE-003", "ORD-003", "WAVE-001", domain.RouteStatusPending)
		require.NoError(t, err)

		err = repo.Save(ctx, route)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, "ROUTE-003")
		assert.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "ROUTE-003", found.RouteID)
		assert.Equal(t, 3, len(found.Stops))
	})

	t.Run("Find non-existent pick route", func(t *testing.T) {
		found, err := repo.FindByID(ctx, "ROUTE-NONEXISTENT")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

// TestRouteRepository_FindByOrderID tests finding pick routes by order ID
func TestRouteRepository_FindByOrderID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	orderID := "ORD-004"

	// Create multiple routes for the same order
	for i := 1; i <= 2; i++ {
		route, err := createTestPickRoute(
			fmt.Sprintf("ROUTE-ORD4-%d", i),
			orderID,
			"WAVE-001",
			domain.RouteStatusPending,
		)
		require.NoError(t, err)
		err = repo.Save(ctx, route)
		require.NoError(t, err)
	}

	t.Run("Find all pick routes for order", func(t *testing.T) {
		routes, err := repo.FindByOrderID(ctx, orderID)
		assert.NoError(t, err)
		assert.Len(t, routes, 2)

		// Verify all routes belong to the order
		for _, route := range routes {
			assert.Equal(t, orderID, route.OrderID)
		}
	})

	t.Run("Find for non-existent order", func(t *testing.T) {
		routes, err := repo.FindByOrderID(ctx, "ORD-NONEXISTENT")
		assert.NoError(t, err)
		assert.Empty(t, routes)
	})
}

// TestRouteRepository_FindByWaveID tests finding pick routes by wave ID
func TestRouteRepository_FindByWaveID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	waveID := "WAVE-002"

	// Create multiple routes for the same wave
	for i := 1; i <= 4; i++ {
		route, err := createTestPickRoute(
			fmt.Sprintf("ROUTE-WAVE2-%d", i),
			fmt.Sprintf("ORD-%d", i),
			waveID,
			domain.RouteStatusPending,
		)
		require.NoError(t, err)
		err = repo.Save(ctx, route)
		require.NoError(t, err)
	}

	t.Run("Find all pick routes for wave", func(t *testing.T) {
		routes, err := repo.FindByWaveID(ctx, waveID)
		assert.NoError(t, err)
		assert.Len(t, routes, 4)

		// Verify all routes belong to the wave
		for _, route := range routes {
			assert.Equal(t, waveID, route.WaveID)
		}
	})

	t.Run("Find for non-existent wave", func(t *testing.T) {
		routes, err := repo.FindByWaveID(ctx, "WAVE-NONEXISTENT")
		assert.NoError(t, err)
		assert.Empty(t, routes)
	})
}

// TestRouteRepository_FindByPickerID tests finding pick routes by picker ID
func TestRouteRepository_FindByPickerID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pickerID := "PICKER-001"

	// Create routes and assign to picker
	for i := 1; i <= 3; i++ {
		route, err := createTestPickRoute(
			fmt.Sprintf("ROUTE-PICKER1-%d", i),
			fmt.Sprintf("ORD-%d", i),
			"WAVE-003",
			domain.RouteStatusPending,
		)
		require.NoError(t, err)

		// Assign to picker and start
		err = route.Start(pickerID)
		require.NoError(t, err)

		err = repo.Save(ctx, route)
		require.NoError(t, err)
	}

	t.Run("Find all pick routes for picker", func(t *testing.T) {
		routes, err := repo.FindByPickerID(ctx, pickerID)
		assert.NoError(t, err)
		assert.Len(t, routes, 3)

		// Verify all routes belong to the picker
		for _, route := range routes {
			assert.Equal(t, pickerID, route.PickerID)
		}
	})

	t.Run("Find for non-existent picker", func(t *testing.T) {
		routes, err := repo.FindByPickerID(ctx, "PICKER-NONEXISTENT")
		assert.NoError(t, err)
		assert.Empty(t, routes)
	})
}

// TestRouteRepository_FindByStatus tests finding pick routes by status
func TestRouteRepository_FindByStatus(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create routes with different statuses
	statuses := []domain.RouteStatus{
		domain.RouteStatusPending,
		domain.RouteStatusInProgress,
		domain.RouteStatusCompleted,
	}

	for i, status := range statuses {
		route, err := createTestPickRoute(
			fmt.Sprintf("ROUTE-STATUS-%d", i+1),
			fmt.Sprintf("ORD-%d", i+1),
			"WAVE-004",
			status,
		)
		require.NoError(t, err)
		err = repo.Save(ctx, route)
		require.NoError(t, err)
	}

	t.Run("Find pick routes by status", func(t *testing.T) {
		routes, err := repo.FindByStatus(ctx, domain.RouteStatusPending)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(routes), 1)

		// Verify all routes have the correct status
		for _, route := range routes {
			assert.Equal(t, domain.RouteStatusPending, route.Status)
		}
	})

	t.Run("Find with no matching status", func(t *testing.T) {
		routes, err := repo.FindByStatus(ctx, domain.RouteStatusCancelled)
		assert.NoError(t, err)
		// Could be empty or have some from other tests
		for _, route := range routes {
			assert.Equal(t, domain.RouteStatusCancelled, route.Status)
		}
	})
}

// TestRouteRepository_FindByZone tests finding pick routes by zone
func TestRouteRepository_FindByZone(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	zone := "ZONE-B"

	// Create routes in the same zone
	for i := 1; i <= 3; i++ {
		route, err := createTestPickRoute(
			fmt.Sprintf("ROUTE-ZONE-%d", i),
			fmt.Sprintf("ORD-%d", i),
			"WAVE-005",
			domain.RouteStatusPending,
		)
		require.NoError(t, err)
		route.Zone = zone
		err = repo.Save(ctx, route)
		require.NoError(t, err)
	}

	t.Run("Find all pick routes in zone", func(t *testing.T) {
		routes, err := repo.FindByZone(ctx, zone)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(routes), 3)

		// Verify all routes are in the correct zone
		for _, route := range routes {
			assert.Equal(t, zone, route.Zone)
		}
	})

	t.Run("Find for non-existent zone", func(t *testing.T) {
		routes, err := repo.FindByZone(ctx, "ZONE-NONEXISTENT")
		assert.NoError(t, err)
		assert.Empty(t, routes)
	})
}

// TestRouteRepository_FindActiveByPicker tests finding active pick route for a picker
func TestRouteRepository_FindActiveByPicker(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pickerID := "PICKER-002"

	// Create a route in progress for the picker
	route, err := createTestPickRoute("ROUTE-ACTIVE-001", "ORD-010", "WAVE-006", domain.RouteStatusPending)
	require.NoError(t, err)

	err = route.Start(pickerID)
	require.NoError(t, err)

	err = repo.Save(ctx, route)
	require.NoError(t, err)

	t.Run("Find active pick route for picker", func(t *testing.T) {
		found, err := repo.FindActiveByPicker(ctx, pickerID)
		assert.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, pickerID, found.PickerID)
		assert.Equal(t, domain.RouteStatusInProgress, found.Status)
	})

	t.Run("Find active for picker with no active route", func(t *testing.T) {
		found, err := repo.FindActiveByPicker(ctx, "PICKER-NONEXISTENT")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

// TestRouteRepository_FindPendingRoutes tests finding pending pick routes
func TestRouteRepository_FindPendingRoutes(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	zone := "ZONE-C"

	// Create pending routes in the zone
	for i := 1; i <= 5; i++ {
		route, err := createTestPickRoute(
			fmt.Sprintf("ROUTE-PENDING-%d", i),
			fmt.Sprintf("ORD-%d", i),
			"WAVE-007",
			domain.RouteStatusPending,
		)
		require.NoError(t, err)
		route.Zone = zone
		err = repo.Save(ctx, route)
		require.NoError(t, err)
	}

	t.Run("Find pending routes in zone", func(t *testing.T) {
		routes, err := repo.FindPendingRoutes(ctx, zone, 10)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(routes), 5)

		// Verify all routes are pending and in correct zone
		for _, route := range routes {
			assert.Equal(t, domain.RouteStatusPending, route.Status)
			assert.Equal(t, zone, route.Zone)
		}
	})

	t.Run("Find pending routes with limit", func(t *testing.T) {
		routes, err := repo.FindPendingRoutes(ctx, zone, 3)
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(routes), 3)
	})

	t.Run("Find pending routes in all zones", func(t *testing.T) {
		routes, err := repo.FindPendingRoutes(ctx, "", 10)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(routes), 5)
	})
}

// TestRouteRepository_CountByStatus tests counting routes by status
func TestRouteRepository_CountByStatus(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create routes with different statuses
	for i := 1; i <= 5; i++ {
		route, err := createTestPickRoute(
			fmt.Sprintf("ROUTE-COUNT-%d", i),
			fmt.Sprintf("ORD-%d", i),
			"WAVE-008",
			domain.RouteStatusPending,
		)
		require.NoError(t, err)
		err = repo.Save(ctx, route)
		require.NoError(t, err)
	}

	t.Run("Count routes by status", func(t *testing.T) {
		count, err := repo.CountByStatus(ctx, domain.RouteStatusPending)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(5))
	})

	t.Run("Count routes with no matching status", func(t *testing.T) {
		count, err := repo.CountByStatus(ctx, domain.RouteStatusCancelled)
		assert.NoError(t, err)
		// Could be 0 or have some from other tests
		assert.GreaterOrEqual(t, count, int64(0))
	})
}

// TestRouteRepository_Delete tests deleting a pick route
func TestRouteRepository_Delete(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Delete existing pick route", func(t *testing.T) {
		route, err := createTestPickRoute("ROUTE-DELETE-001", "ORD-020", "WAVE-009", domain.RouteStatusPending)
		require.NoError(t, err)
		err = repo.Save(ctx, route)
		require.NoError(t, err)

		// Delete route
		err = repo.Delete(ctx, "ROUTE-DELETE-001")
		assert.NoError(t, err)

		// Verify it's deleted
		found, err := repo.FindByID(ctx, "ROUTE-DELETE-001")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("Delete non-existent pick route", func(t *testing.T) {
		err := repo.Delete(ctx, "ROUTE-NONEXISTENT")
		// Should not error, just no-op
		assert.NoError(t, err)
	})
}
