package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shipping-service/internal/domain"
	"github.com/wms-platform/shipping-service/internal/infrastructure/mongodb"
	sharedtesting "github.com/wms-platform/shared/pkg/testing"
)

// Test fixtures
func createTestShipment(shipmentID, orderID, waveID string, carrierCode string, status domain.ShipmentStatus) *domain.Shipment {
	carrier := domain.Carrier{
		Code:        carrierCode,
		Name:        "Test Carrier",
		AccountID:   "ACC-001",
		ServiceType: "Ground",
	}

	pkg := domain.PackageInfo{
		PackageID:   "PKG-001",
		Weight:      2.5,
		Dimensions: domain.Dimensions{
			Length: 30,
			Width:  20,
			Height: 15,
		},
		PackageType: "box",
	}

	recipient := domain.Address{
		Name:       "John Doe",
		Street1:    "123 Main St",
		City:       "San Francisco",
		State:      "CA",
		PostalCode: "94105",
		Country:    "USA",
		Phone:      "+1-555-0123",
	}

	shipper := domain.Address{
		Name:       "Warehouse",
		Street1:    "456 Warehouse Blvd",
		City:       "Oakland",
		State:      "CA",
		PostalCode: "94601",
		Country:    "USA",
	}

	shipment := domain.NewShipment(shipmentID, orderID, pkg.PackageID, waveID, carrier, pkg, recipient, shipper)

	// Set status if different from default
	if status != domain.ShipmentStatusPending {
		shipment.Status = status
	}

	return shipment
}

func setupTestRepository(t *testing.T) (*mongodb.ShipmentRepository, *sharedtesting.MongoDBContainer, func()) {
	ctx := context.Background()

	// Start MongoDB container
	mongoContainer, err := sharedtesting.NewMongoDBContainer(ctx)
	require.NoError(t, err)

	// Get MongoDB client
	client, err := mongoContainer.GetClient(ctx)
	require.NoError(t, err)

	// Create event factory
	eventFactory := cloudevents.NewEventFactory("/shipping-service")

	// Create repository
	db := client.Database("test_shipping_db")
	repo := mongodb.NewShipmentRepository(db, eventFactory)

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

// TestShipmentRepository_Save tests the Save operation
func TestShipmentRepository_Save(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Save new shipment", func(t *testing.T) {
		shipment := createTestShipment("SHIP-001", "ORD-001", "WAVE-001", "UPS", domain.ShipmentStatusPending)

		err := repo.Save(ctx, shipment)
		assert.NoError(t, err)

		// Verify it was saved
		found, err := repo.FindByID(ctx, "SHIP-001")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "SHIP-001", found.ShipmentID)
		assert.Equal(t, "ORD-001", found.OrderID)
		assert.Equal(t, domain.ShipmentStatusPending, found.Status)
	})

	t.Run("Update existing shipment (upsert)", func(t *testing.T) {
		shipment := createTestShipment("SHIP-002", "ORD-002", "WAVE-001", "UPS", domain.ShipmentStatusPending)

		// Save first time
		err := repo.Save(ctx, shipment)
		require.NoError(t, err)

		// Update status and save again
		label := domain.ShippingLabel{
			TrackingNumber: "TRACK-001",
			LabelFormat:    "PDF",
			LabelData:      "base64data",
			GeneratedAt:    time.Now(),
		}
		shipment.GenerateLabel(label)
		err = repo.Save(ctx, shipment)
		assert.NoError(t, err)

		// Verify update
		found, err := repo.FindByID(ctx, "SHIP-002")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, domain.ShipmentStatusLabeled, found.Status)
		assert.NotNil(t, found.Label)
	})
}

// TestShipmentRepository_FindByID tests finding a shipment by ID
func TestShipmentRepository_FindByID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Find existing shipment", func(t *testing.T) {
		shipment := createTestShipment("SHIP-003", "ORD-003", "WAVE-001", "FEDEX", domain.ShipmentStatusPending)

		err := repo.Save(ctx, shipment)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, "SHIP-003")
		assert.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "SHIP-003", found.ShipmentID)
		assert.Equal(t, "FEDEX", found.Carrier.Code)
	})

	t.Run("Find non-existent shipment", func(t *testing.T) {
		found, err := repo.FindByID(ctx, "SHIP-NONEXISTENT")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

// TestShipmentRepository_FindByOrderID tests finding a shipment by order ID
func TestShipmentRepository_FindByOrderID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Find shipment by order ID", func(t *testing.T) {
		shipment := createTestShipment("SHIP-004", "ORD-004", "WAVE-001", "UPS", domain.ShipmentStatusPending)

		err := repo.Save(ctx, shipment)
		require.NoError(t, err)

		found, err := repo.FindByOrderID(ctx, "ORD-004")
		assert.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "ORD-004", found.OrderID)
		assert.Equal(t, "SHIP-004", found.ShipmentID)
	})

	t.Run("Find for non-existent order", func(t *testing.T) {
		found, err := repo.FindByOrderID(ctx, "ORD-NONEXISTENT")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

// TestShipmentRepository_FindByTrackingNumber tests finding a shipment by tracking number
func TestShipmentRepository_FindByTrackingNumber(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Find shipment by tracking number", func(t *testing.T) {
		shipment := createTestShipment("SHIP-005", "ORD-005", "WAVE-001", "UPS", domain.ShipmentStatusPending)

		label := domain.ShippingLabel{
			TrackingNumber: "TRACK-12345",
			LabelFormat:    "PDF",
			LabelData:      "base64data",
			GeneratedAt:    time.Now(),
		}
		shipment.GenerateLabel(label)

		err := repo.Save(ctx, shipment)
		require.NoError(t, err)

		found, err := repo.FindByTrackingNumber(ctx, "TRACK-12345")
		assert.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "TRACK-12345", found.Label.TrackingNumber)
		assert.Equal(t, "SHIP-005", found.ShipmentID)
	})

	t.Run("Find for non-existent tracking number", func(t *testing.T) {
		found, err := repo.FindByTrackingNumber(ctx, "TRACK-NONEXISTENT")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

// TestShipmentRepository_FindByStatus tests finding shipments by status
func TestShipmentRepository_FindByStatus(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create shipments with different statuses
	statuses := []domain.ShipmentStatus{
		domain.ShipmentStatusPending,
		domain.ShipmentStatusLabeled,
		domain.ShipmentStatusManifested,
		domain.ShipmentStatusShipped,
	}

	for i, status := range statuses {
		shipment := createTestShipment(
			fmt.Sprintf("SHIP-STATUS-%d", i+1),
			fmt.Sprintf("ORD-%d", i+1),
			"WAVE-002",
			"UPS",
			status,
		)
		err := repo.Save(ctx, shipment)
		require.NoError(t, err)
	}

	t.Run("Find shipments by status", func(t *testing.T) {
		shipments, err := repo.FindByStatus(ctx, domain.ShipmentStatusPending)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(shipments), 1)

		// Verify all shipments have the correct status
		for _, shipment := range shipments {
			assert.Equal(t, domain.ShipmentStatusPending, shipment.Status)
		}
	})

	t.Run("Find with no matching status", func(t *testing.T) {
		shipments, err := repo.FindByStatus(ctx, domain.ShipmentStatusCancelled)
		assert.NoError(t, err)
		// Could be empty or have some from other tests
		for _, shipment := range shipments {
			assert.Equal(t, domain.ShipmentStatusCancelled, shipment.Status)
		}
	})
}

// TestShipmentRepository_FindByCarrier tests finding shipments by carrier
func TestShipmentRepository_FindByCarrier(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	carrierCode := "FEDEX"

	// Create shipments with the same carrier
	for i := 1; i <= 3; i++ {
		shipment := createTestShipment(
			fmt.Sprintf("SHIP-CARRIER-%d", i),
			fmt.Sprintf("ORD-%d", i),
			"WAVE-003",
			carrierCode,
			domain.ShipmentStatusPending,
		)
		err := repo.Save(ctx, shipment)
		require.NoError(t, err)
	}

	t.Run("Find all shipments for carrier", func(t *testing.T) {
		shipments, err := repo.FindByCarrier(ctx, carrierCode)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(shipments), 3)

		// Verify all shipments belong to the carrier
		for _, shipment := range shipments {
			assert.Equal(t, carrierCode, shipment.Carrier.Code)
		}
	})

	t.Run("Find for non-existent carrier", func(t *testing.T) {
		shipments, err := repo.FindByCarrier(ctx, "CARRIER-NONEXISTENT")
		assert.NoError(t, err)
		assert.Empty(t, shipments)
	})
}

// TestShipmentRepository_FindByManifestID tests finding shipments by manifest ID
func TestShipmentRepository_FindByManifestID(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	manifestID := "MAN-001"

	// Create shipments and add to manifest
	for i := 1; i <= 3; i++ {
		shipment := createTestShipment(
			fmt.Sprintf("SHIP-MAN-%d", i),
			fmt.Sprintf("ORD-%d", i),
			"WAVE-004",
			"UPS",
			domain.ShipmentStatusLabeled,
		)

		// Add label first
		label := domain.ShippingLabel{
			TrackingNumber: fmt.Sprintf("TRACK-%d", i),
			LabelFormat:    "PDF",
			LabelData:      "base64data",
			GeneratedAt:    time.Now(),
		}
		shipment.Label = &label

		// Add to manifest
		manifest := domain.Manifest{
			ManifestID:    manifestID,
			CarrierCode:   "UPS",
			ShipmentCount: 10,
			GeneratedAt:   time.Now(),
		}
		shipment.AddToManifest(manifest)

		err := repo.Save(ctx, shipment)
		require.NoError(t, err)
	}

	t.Run("Find all shipments in manifest", func(t *testing.T) {
		shipments, err := repo.FindByManifestID(ctx, manifestID)
		assert.NoError(t, err)
		assert.Len(t, shipments, 3)

		// Verify all shipments belong to the manifest
		for _, shipment := range shipments {
			assert.NotNil(t, shipment.Manifest)
			assert.Equal(t, manifestID, shipment.Manifest.ManifestID)
		}
	})

	t.Run("Find for non-existent manifest", func(t *testing.T) {
		shipments, err := repo.FindByManifestID(ctx, "MAN-NONEXISTENT")
		assert.NoError(t, err)
		assert.Empty(t, shipments)
	})
}

// TestShipmentRepository_FindPendingForManifest tests finding shipments pending for manifest
func TestShipmentRepository_FindPendingForManifest(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	carrierCode := "USPS"

	// Create labeled shipments for the carrier
	for i := 1; i <= 4; i++ {
		shipment := createTestShipment(
			fmt.Sprintf("SHIP-PENDING-%d", i),
			fmt.Sprintf("ORD-%d", i),
			"WAVE-005",
			carrierCode,
			domain.ShipmentStatusLabeled,
		)

		// Add label
		label := domain.ShippingLabel{
			TrackingNumber: fmt.Sprintf("TRACK-PENDING-%d", i),
			LabelFormat:    "PDF",
			LabelData:      "base64data",
			GeneratedAt:    time.Now(),
		}
		shipment.Label = &label

		err := repo.Save(ctx, shipment)
		require.NoError(t, err)
	}

	t.Run("Find all pending shipments for manifest", func(t *testing.T) {
		shipments, err := repo.FindPendingForManifest(ctx, carrierCode)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(shipments), 4)

		// Verify all shipments are labeled and have the correct carrier
		for _, shipment := range shipments {
			assert.Equal(t, domain.ShipmentStatusLabeled, shipment.Status)
			assert.Equal(t, carrierCode, shipment.Carrier.Code)
		}
	})

	t.Run("Find for carrier with no pending shipments", func(t *testing.T) {
		shipments, err := repo.FindPendingForManifest(ctx, "CARRIER-NONE")
		assert.NoError(t, err)
		assert.Empty(t, shipments)
	})
}

// TestShipmentRepository_Delete tests deleting a shipment
func TestShipmentRepository_Delete(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Delete existing shipment", func(t *testing.T) {
		shipment := createTestShipment("SHIP-DELETE-001", "ORD-020", "WAVE-007", "UPS", domain.ShipmentStatusPending)
		err := repo.Save(ctx, shipment)
		require.NoError(t, err)

		// Delete shipment
		err = repo.Delete(ctx, "SHIP-DELETE-001")
		assert.NoError(t, err)

		// Verify it's deleted
		found, err := repo.FindByID(ctx, "SHIP-DELETE-001")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("Delete non-existent shipment", func(t *testing.T) {
		err := repo.Delete(ctx, "SHIP-NONEXISTENT")
		// Should not error, just no-op
		assert.NoError(t, err)
	})
}
