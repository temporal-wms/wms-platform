package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wms-platform/services/order-service/internal/domain"
	"github.com/wms-platform/services/order-service/internal/infrastructure/mongodb"
	"github.com/wms-platform/shared/pkg/cloudevents"
	sharedtesting "github.com/wms-platform/shared/pkg/testing"
)

// Test fixtures
func createTestOrder(orderID, customerID string, status domain.Status, priority domain.Priority) (*domain.Order, error) {
	items := []domain.OrderItem{
		{
			SKU:      "SKU-001",
			Name:     "Test Product",
			Quantity: 2,
			Weight:   1.5,
			Dimensions: domain.Dims{
				Length: 10,
				Width:  5,
				Height: 3,
			},
			UnitPrice: 29.99,
		},
	}

	address := domain.Address{
		Street:        "123 Main St",
		City:          "San Francisco",
		State:         "CA",
		ZipCode:       "94105",
		Country:       "USA",
		Phone:         "+1-555-0123",
		RecipientName: "John Doe",
	}

	order, err := domain.NewOrder(
		orderID,
		customerID,
		items,
		address,
		priority,
		time.Now().Add(24*time.Hour),
	)
	if err != nil {
		return nil, err
	}

	// Set the status if different from default
	if status != domain.StatusReceived {
		order.Status = status
	}

	return order, nil
}

func setupTestRepository(t *testing.T) (*mongodb.OrderRepository, *sharedtesting.MongoDBContainer, func()) {
	ctx := context.Background()

	// Start MongoDB container
	mongoContainer, err := sharedtesting.NewMongoDBContainer(ctx)
	require.NoError(t, err)

	// Get MongoDB client
	client, err := mongoContainer.GetClient(ctx)
	require.NoError(t, err)

	// Create repository with event factory
	db := client.Database("test_orders_db")
	eventFactory := cloudevents.NewEventFactory("/order-service")
	repo := mongodb.NewOrderRepository(db, eventFactory)

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

// TestOrderRepository_Save tests the Save operation
func TestOrderRepository_Save(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Save new order", func(t *testing.T) {
		order, err := createTestOrder("ORD-001", "CUST-001", domain.StatusReceived, domain.PriorityStandard)
		require.NoError(t, err)

		err = repo.Save(ctx, order)
		assert.NoError(t, err)

		// Verify it was saved
		found, err := repo.FindByID(ctx, "ORD-001")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "ORD-001", found.OrderID)
		assert.Equal(t, "CUST-001", found.CustomerID)
		assert.Equal(t, domain.StatusReceived, found.Status)
	})

	t.Run("Update existing order (upsert)", func(t *testing.T) {
		order, err := createTestOrder("ORD-002", "CUST-001", domain.StatusReceived, domain.PriorityStandard)
		require.NoError(t, err)

		// Save first time
		err = repo.Save(ctx, order)
		require.NoError(t, err)

		// Update status and save again
		order.Status = domain.StatusValidated
		err = repo.Save(ctx, order)
		assert.NoError(t, err)

		// Verify update
		found, err := repo.FindByID(ctx, "ORD-002")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, domain.StatusValidated, found.Status)
	})
}

// TestOrderRepository_FindByID tests finding an order by ID
func TestOrderRepository_FindByID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Find existing order", func(t *testing.T) {
		order, err := createTestOrder("ORD-003", "CUST-001", domain.StatusReceived, domain.PrioritySameDay)
		require.NoError(t, err)

		err = repo.Save(ctx, order)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, "ORD-003")
		assert.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "ORD-003", found.OrderID)
		assert.Equal(t, domain.PrioritySameDay, found.Priority)
	})

	t.Run("Find non-existent order", func(t *testing.T) {
		found, err := repo.FindByID(ctx, "ORD-NONEXISTENT")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

// TestOrderRepository_FindByCustomerID tests finding orders by customer ID
func TestOrderRepository_FindByCustomerID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create multiple orders for the same customer
	customerID := "CUST-002"
	for i := 1; i <= 5; i++ {
		order, err := createTestOrder(
			fmt.Sprintf("ORD-CUST2-%d", i),
			customerID,
			domain.StatusReceived,
			domain.PriorityStandard,
		)
		require.NoError(t, err)
		err = repo.Save(ctx, order)
		require.NoError(t, err)
	}

	t.Run("Find all orders for customer", func(t *testing.T) {
		pagination := domain.DefaultPagination()
		orders, err := repo.FindByCustomerID(ctx, customerID, pagination)
		assert.NoError(t, err)
		assert.Len(t, orders, 5)

		// Verify all orders belong to the customer
		for _, order := range orders {
			assert.Equal(t, customerID, order.CustomerID)
		}
	})

	t.Run("Find with pagination", func(t *testing.T) {
		pagination := domain.Pagination{Page: 1, PageSize: 2}
		orders, err := repo.FindByCustomerID(ctx, customerID, pagination)
		assert.NoError(t, err)
		assert.Len(t, orders, 2)

		// Get second page
		pagination.Page = 2
		orders, err = repo.FindByCustomerID(ctx, customerID, pagination)
		assert.NoError(t, err)
		assert.Len(t, orders, 2)
	})

	t.Run("Find for non-existent customer", func(t *testing.T) {
		pagination := domain.DefaultPagination()
		orders, err := repo.FindByCustomerID(ctx, "CUST-NONEXISTENT", pagination)
		assert.NoError(t, err)
		assert.Empty(t, orders)
	})
}

// TestOrderRepository_FindByStatus tests finding orders by status
func TestOrderRepository_FindByStatus(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create orders with different statuses
	statuses := []domain.Status{
		domain.StatusReceived,
		domain.StatusValidated,
		domain.StatusValidated,
		domain.StatusWaveAssigned,
	}

	for i, status := range statuses {
		order, err := createTestOrder(
			fmt.Sprintf("ORD-STATUS-%d", i+1),
			"CUST-003",
			status,
			domain.PriorityStandard,
		)
		require.NoError(t, err)
		err = repo.Save(ctx, order)
		require.NoError(t, err)
	}

	t.Run("Find orders by status", func(t *testing.T) {
		pagination := domain.DefaultPagination()
		orders, err := repo.FindByStatus(ctx, domain.StatusValidated, pagination)
		assert.NoError(t, err)
		assert.Len(t, orders, 2)

		// Verify all orders have the correct status
		for _, order := range orders {
			assert.Equal(t, domain.StatusValidated, order.Status)
		}
	})

	t.Run("Find with no matching status", func(t *testing.T) {
		pagination := domain.DefaultPagination()
		orders, err := repo.FindByStatus(ctx, domain.StatusShipped, pagination)
		assert.NoError(t, err)
		assert.Empty(t, orders)
	})
}

// TestOrderRepository_FindByWaveID tests finding orders by wave ID
func TestOrderRepository_FindByWaveID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	waveID := "WAVE-001"

	// Create orders and assign them to a wave
	for i := 1; i <= 3; i++ {
		order, err := createTestOrder(
			fmt.Sprintf("ORD-WAVE-%d", i),
			"CUST-004",
			domain.StatusValidated,
			domain.PriorityStandard,
		)
		require.NoError(t, err)
		err = repo.Save(ctx, order)
		require.NoError(t, err)

		// Assign to wave
		err = repo.AssignToWave(ctx, order.OrderID, waveID)
		require.NoError(t, err)
	}

	t.Run("Find orders by wave ID", func(t *testing.T) {
		orders, err := repo.FindByWaveID(ctx, waveID)
		assert.NoError(t, err)
		assert.Len(t, orders, 3)

		// Verify all orders belong to the wave
		for _, order := range orders {
			assert.Equal(t, waveID, order.WaveID)
		}
	})

	t.Run("Find with non-existent wave ID", func(t *testing.T) {
		orders, err := repo.FindByWaveID(ctx, "WAVE-NONEXISTENT")
		assert.NoError(t, err)
		assert.Empty(t, orders)
	})
}

// TestOrderRepository_FindValidatedOrders tests finding validated orders ready for wave assignment
func TestOrderRepository_FindValidatedOrders(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create orders with different priorities
	priorities := []domain.Priority{
		domain.PrioritySameDay,
		domain.PriorityNextDay,
		domain.PriorityStandard,
	}

	for i, priority := range priorities {
		order, err := createTestOrder(
			fmt.Sprintf("ORD-VALIDATED-%d", i+1),
			"CUST-005",
			domain.StatusValidated,
			priority,
		)
		require.NoError(t, err)
		err = repo.Save(ctx, order)
		require.NoError(t, err)
	}

	// Create one order already assigned to a wave (should be excluded)
	order, err := createTestOrder("ORD-ALREADY-WAVED", "CUST-005", domain.StatusValidated, domain.PrioritySameDay)
	require.NoError(t, err)
	order.WaveID = "WAVE-EXISTS"
	err = repo.Save(ctx, order)
	require.NoError(t, err)

	t.Run("Find all validated orders", func(t *testing.T) {
		orders, err := repo.FindValidatedOrders(ctx, "", 10)
		assert.NoError(t, err)
		assert.Len(t, orders, 3) // Should not include already waved order

		// Verify all orders are validated and not assigned to a wave
		for _, order := range orders {
			assert.Equal(t, domain.StatusValidated, order.Status)
			assert.Empty(t, order.WaveID)
		}

		// Verify we have all three priority types
		priorities := make(map[domain.Priority]bool)
		for _, order := range orders {
			priorities[order.Priority] = true
		}
		assert.Len(t, priorities, 3)
	})

	t.Run("Find validated orders by priority", func(t *testing.T) {
		orders, err := repo.FindValidatedOrders(ctx, domain.PrioritySameDay, 10)
		assert.NoError(t, err)
		assert.Len(t, orders, 1)
		assert.Equal(t, domain.PrioritySameDay, orders[0].Priority)
	})

	t.Run("Find with limit", func(t *testing.T) {
		orders, err := repo.FindValidatedOrders(ctx, "", 2)
		assert.NoError(t, err)
		assert.Len(t, orders, 2)
	})
}

// TestOrderRepository_UpdateStatus tests updating order status
func TestOrderRepository_UpdateStatus(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Update existing order status", func(t *testing.T) {
		order, err := createTestOrder("ORD-UPDATE-1", "CUST-006", domain.StatusReceived, domain.PriorityStandard)
		require.NoError(t, err)
		err = repo.Save(ctx, order)
		require.NoError(t, err)

		// Update status
		err = repo.UpdateStatus(ctx, "ORD-UPDATE-1", domain.StatusValidated)
		assert.NoError(t, err)

		// Verify update
		found, err := repo.FindByID(ctx, "ORD-UPDATE-1")
		require.NoError(t, err)
		assert.Equal(t, domain.StatusValidated, found.Status)
	})

	t.Run("Update non-existent order", func(t *testing.T) {
		err := repo.UpdateStatus(ctx, "ORD-NONEXISTENT", domain.StatusValidated)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestOrderRepository_AssignToWave tests assigning orders to waves
func TestOrderRepository_AssignToWave(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Assign validated order to wave", func(t *testing.T) {
		order, err := createTestOrder("ORD-ASSIGN-1", "CUST-007", domain.StatusValidated, domain.PriorityStandard)
		require.NoError(t, err)
		err = repo.Save(ctx, order)
		require.NoError(t, err)

		// Assign to wave
		err = repo.AssignToWave(ctx, "ORD-ASSIGN-1", "WAVE-002")
		assert.NoError(t, err)

		// Verify assignment
		found, err := repo.FindByID(ctx, "ORD-ASSIGN-1")
		require.NoError(t, err)
		assert.Equal(t, "WAVE-002", found.WaveID)
		assert.Equal(t, domain.StatusWaveAssigned, found.Status)
	})

	t.Run("Cannot assign non-validated order", func(t *testing.T) {
		order, err := createTestOrder("ORD-ASSIGN-2", "CUST-007", domain.StatusReceived, domain.PriorityStandard)
		require.NoError(t, err)
		err = repo.Save(ctx, order)
		require.NoError(t, err)

		// Try to assign (should fail)
		err = repo.AssignToWave(ctx, "ORD-ASSIGN-2", "WAVE-003")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not in validated status")
	})

	t.Run("Cannot assign non-existent order", func(t *testing.T) {
		err := repo.AssignToWave(ctx, "ORD-NONEXISTENT", "WAVE-004")
		assert.Error(t, err)
	})
}

// TestOrderRepository_Delete tests soft delete functionality
func TestOrderRepository_Delete(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Delete existing order (soft delete)", func(t *testing.T) {
		order, err := createTestOrder("ORD-DELETE-1", "CUST-008", domain.StatusReceived, domain.PriorityStandard)
		require.NoError(t, err)
		err = repo.Save(ctx, order)
		require.NoError(t, err)

		// Delete order
		err = repo.Delete(ctx, "ORD-DELETE-1")
		assert.NoError(t, err)

		// Verify it's marked as cancelled (soft delete)
		found, err := repo.FindByID(ctx, "ORD-DELETE-1")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, domain.StatusCancelled, found.Status)
	})

	t.Run("Delete non-existent order", func(t *testing.T) {
		err := repo.Delete(ctx, "ORD-NONEXISTENT")
		// Should not error, just no-op
		assert.NoError(t, err)
	})
}

// TestOrderRepository_Count tests counting orders with filters
func TestOrderRepository_Count(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create test data
	customerID := "CUST-009"
	status := domain.StatusValidated
	priority := domain.PrioritySameDay

	for i := 1; i <= 5; i++ {
		order, err := createTestOrder(
			fmt.Sprintf("ORD-COUNT-%d", i),
			customerID,
			status,
			priority,
		)
		require.NoError(t, err)
		err = repo.Save(ctx, order)
		require.NoError(t, err)
	}

	t.Run("Count all orders for customer", func(t *testing.T) {
		filter := domain.OrderFilter{CustomerID: &customerID}
		count, err := repo.Count(ctx, filter)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})

	t.Run("Count orders by status", func(t *testing.T) {
		filter := domain.OrderFilter{Status: &status}
		count, err := repo.Count(ctx, filter)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(5))
	})

	t.Run("Count orders by priority", func(t *testing.T) {
		filter := domain.OrderFilter{Priority: &priority}
		count, err := repo.Count(ctx, filter)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(5))
	})

	t.Run("Count with multiple filters", func(t *testing.T) {
		filter := domain.OrderFilter{
			CustomerID: &customerID,
			Status:     &status,
			Priority:   &priority,
		}
		count, err := repo.Count(ctx, filter)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})

	t.Run("Count with no matching filter", func(t *testing.T) {
		nonExistentCustomer := "CUST-NONEXISTENT"
		filter := domain.OrderFilter{CustomerID: &nonExistentCustomer}
		count, err := repo.Count(ctx, filter)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}
