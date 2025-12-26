package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wms-platform/inventory-service/internal/domain"
	"github.com/wms-platform/inventory-service/internal/infrastructure/mongodb"
	sharedtesting "github.com/wms-platform/shared/pkg/testing"
)

// Test fixtures
func createTestInventoryItem(sku, productName string, locations []domain.StockLocation) *domain.InventoryItem {
	item := domain.NewInventoryItem(sku, productName, 10, 50)

	// Add locations
	item.Locations = locations

	// Calculate totals
	totalQty := 0
	availableQty := 0
	for _, loc := range locations {
		totalQty += loc.Quantity
		availableQty += loc.Available
	}
	item.TotalQuantity = totalQty
	item.AvailableQuantity = availableQty

	return item
}

func setupTestRepository(t *testing.T) (*mongodb.InventoryRepository, *sharedtesting.MongoDBContainer, func()) {
	ctx := context.Background()

	// Start MongoDB container
	mongoContainer, err := sharedtesting.NewMongoDBContainer(ctx)
	require.NoError(t, err)

	// Get MongoDB client
	client, err := mongoContainer.GetClient(ctx)
	require.NoError(t, err)

	// Create repository
	db := client.Database("test_inventory_db")
	repo := mongodb.NewInventoryRepository(db)

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

// TestInventoryRepository_Save tests the Save operation
func TestInventoryRepository_Save(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Save new inventory item", func(t *testing.T) {
		locations := []domain.StockLocation{
			{
				LocationID: "A-10-1-A",
				Zone:       "ZONE-A",
				Aisle:      "A",
				Rack:       10,
				Level:      1,
				Quantity:   100,
				Reserved:   0,
				Available:  100,
			},
		}
		item := createTestInventoryItem("SKU-001", "Test Product 1", locations)

		err := repo.Save(ctx, item)
		assert.NoError(t, err)

		// Verify it was saved
		found, err := repo.FindBySKU(ctx, "SKU-001")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "SKU-001", found.SKU)
		assert.Equal(t, "Test Product 1", found.ProductName)
		assert.Equal(t, 100, found.TotalQuantity)
	})

	t.Run("Update existing inventory item (upsert)", func(t *testing.T) {
		locations := []domain.StockLocation{
			{
				LocationID: "B-05-2-B",
				Zone:       "ZONE-B",
				Aisle:      "B",
				Rack:       5,
				Level:      2,
				Quantity:   50,
				Reserved:   0,
				Available:  50,
			},
		}
		item := createTestInventoryItem("SKU-002", "Test Product 2", locations)

		// Save first time
		err := repo.Save(ctx, item)
		require.NoError(t, err)

		// Update quantity and save again
		item.ReceiveStock("B-05-2-B", "ZONE-B", 25, "PO-001", "user1")
		err = repo.Save(ctx, item)
		assert.NoError(t, err)

		// Verify update
		found, err := repo.FindBySKU(ctx, "SKU-002")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, 75, found.TotalQuantity)
	})
}

// TestInventoryRepository_FindBySKU tests finding an inventory item by SKU
func TestInventoryRepository_FindBySKU(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Find existing inventory item", func(t *testing.T) {
		locations := []domain.StockLocation{
			{
				LocationID: "C-12-3-A",
				Zone:       "ZONE-C",
				Aisle:      "C",
				Rack:       12,
				Level:      3,
				Quantity:   200,
				Reserved:   0,
				Available:  200,
			},
		}
		item := createTestInventoryItem("SKU-003", "Test Product 3", locations)

		err := repo.Save(ctx, item)
		require.NoError(t, err)

		found, err := repo.FindBySKU(ctx, "SKU-003")
		assert.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "SKU-003", found.SKU)
		assert.Equal(t, 1, len(found.Locations))
	})

	t.Run("Find non-existent inventory item", func(t *testing.T) {
		found, err := repo.FindBySKU(ctx, "SKU-NONEXISTENT")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

// TestInventoryRepository_FindByLocation tests finding inventory items by location
func TestInventoryRepository_FindByLocation(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	locationID := "D-08-1-A"

	// Create items at the same location
	for i := 1; i <= 3; i++ {
		locations := []domain.StockLocation{
			{
				LocationID: locationID,
				Zone:       "ZONE-D",
				Aisle:      "D",
				Rack:       8,
				Level:      1,
				Quantity:   50,
				Reserved:   0,
				Available:  50,
			},
		}
		item := createTestInventoryItem("SKU-LOC-00"+string(rune('0'+i)), "Test Product "+string(rune('0'+i)), locations)
		err := repo.Save(ctx, item)
		require.NoError(t, err)
	}

	t.Run("Find all inventory items at location", func(t *testing.T) {
		items, err := repo.FindByLocation(ctx, locationID)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(items), 3)

		// Verify all items have the location
		for _, item := range items {
			hasLocation := false
			for _, loc := range item.Locations {
				if loc.LocationID == locationID {
					hasLocation = true
					break
				}
			}
			assert.True(t, hasLocation)
		}
	})

	t.Run("Find for non-existent location", func(t *testing.T) {
		items, err := repo.FindByLocation(ctx, "LOC-NONEXISTENT")
		assert.NoError(t, err)
		assert.Empty(t, items)
	})
}

// TestInventoryRepository_FindByZone tests finding inventory items by zone
func TestInventoryRepository_FindByZone(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	zone := "ZONE-E"

	// Create items in the same zone
	for i := 1; i <= 4; i++ {
		locations := []domain.StockLocation{
			{
				LocationID: "E-10-1-A",
				Zone:       zone,
				Aisle:      "E",
				Rack:       10,
				Level:      1,
				Quantity:   75,
				Reserved:   0,
				Available:  75,
			},
		}
		item := createTestInventoryItem("SKU-ZONE-00"+string(rune('0'+i)), "Test Product Zone "+string(rune('0'+i)), locations)
		err := repo.Save(ctx, item)
		require.NoError(t, err)
	}

	t.Run("Find all inventory items in zone", func(t *testing.T) {
		items, err := repo.FindByZone(ctx, zone)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(items), 4)

		// Verify all items have location in the zone
		for _, item := range items {
			hasZone := false
			for _, loc := range item.Locations {
				if loc.Zone == zone {
					hasZone = true
					break
				}
			}
			assert.True(t, hasZone)
		}
	})

	t.Run("Find for non-existent zone", func(t *testing.T) {
		items, err := repo.FindByZone(ctx, "ZONE-NONEXISTENT")
		assert.NoError(t, err)
		assert.Empty(t, items)
	})
}

// TestInventoryRepository_FindLowStock tests finding low stock items
func TestInventoryRepository_FindLowStock(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create items with low stock (available <= reorderPoint)
	locations1 := []domain.StockLocation{
		{
			LocationID: "F-10-1-A",
			Zone:       "ZONE-F",
			Aisle:      "F",
			Rack:       10,
			Level:      1,
			Quantity:   8,
			Reserved:   0,
			Available:  8,
		},
	}
	item1 := createTestInventoryItem("SKU-LOW-001", "Low Stock Product 1", locations1)
	item1.ReorderPoint = 10
	err := repo.Save(ctx, item1)
	require.NoError(t, err)

	locations2 := []domain.StockLocation{
		{
			LocationID: "F-11-1-A",
			Zone:       "ZONE-F",
			Aisle:      "F",
			Rack:       11,
			Level:      1,
			Quantity:   5,
			Reserved:   0,
			Available:  5,
		},
	}
	item2 := createTestInventoryItem("SKU-LOW-002", "Low Stock Product 2", locations2)
	item2.ReorderPoint = 20
	err = repo.Save(ctx, item2)
	require.NoError(t, err)

	// Create item with sufficient stock
	locations3 := []domain.StockLocation{
		{
			LocationID: "F-12-1-A",
			Zone:       "ZONE-F",
			Aisle:      "F",
			Rack:       12,
			Level:      1,
			Quantity:   100,
			Reserved:   0,
			Available:  100,
		},
	}
	item3 := createTestInventoryItem("SKU-GOOD-001", "Good Stock Product", locations3)
	item3.ReorderPoint = 10
	err = repo.Save(ctx, item3)
	require.NoError(t, err)

	t.Run("Find low stock items", func(t *testing.T) {
		items, err := repo.FindLowStock(ctx)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(items), 2)

		// Verify all items have low stock
		for _, item := range items {
			assert.LessOrEqual(t, item.AvailableQuantity, item.ReorderPoint)
		}
	})
}

// TestInventoryRepository_FindAll tests finding all inventory items with pagination
func TestInventoryRepository_FindAll(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create multiple items
	for i := 1; i <= 10; i++ {
		locations := []domain.StockLocation{
			{
				LocationID: "G-10-1-A",
				Zone:       "ZONE-G",
				Aisle:      "G",
				Rack:       10,
				Level:      1,
				Quantity:   50,
				Reserved:   0,
				Available:  50,
			},
		}
		item := createTestInventoryItem("SKU-ALL-0"+string(rune('0'+i%10)), "Test Product All "+string(rune('0'+i%10)), locations)
		err := repo.Save(ctx, item)
		require.NoError(t, err)
	}

	t.Run("Find all with pagination", func(t *testing.T) {
		items, err := repo.FindAll(ctx, 5, 0)
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(items), 5)
	})

	t.Run("Find all with offset", func(t *testing.T) {
		items, err := repo.FindAll(ctx, 5, 5)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(items), 0)
	})
}

// TestInventoryRepository_Delete tests deleting an inventory item
func TestInventoryRepository_Delete(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Delete existing inventory item", func(t *testing.T) {
		locations := []domain.StockLocation{
			{
				LocationID: "H-10-1-A",
				Zone:       "ZONE-H",
				Aisle:      "H",
				Rack:       10,
				Level:      1,
				Quantity:   50,
				Reserved:   0,
				Available:  50,
			},
		}
		item := createTestInventoryItem("SKU-DELETE-001", "Delete Test Product", locations)
		err := repo.Save(ctx, item)
		require.NoError(t, err)

		// Delete item
		err = repo.Delete(ctx, "SKU-DELETE-001")
		assert.NoError(t, err)

		// Verify it's deleted
		found, err := repo.FindBySKU(ctx, "SKU-DELETE-001")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("Delete non-existent inventory item", func(t *testing.T) {
		err := repo.Delete(ctx, "SKU-NONEXISTENT")
		// Should not error, just no-op
		assert.NoError(t, err)
	})
}
