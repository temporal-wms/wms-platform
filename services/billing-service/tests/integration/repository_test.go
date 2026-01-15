package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wms-platform/services/billing-service/internal/domain"
	"github.com/wms-platform/services/billing-service/internal/infrastructure/mongodb"
	sharedtesting "github.com/wms-platform/shared/pkg/testing"
)

// Test fixtures
func createTestActivity(sellerID string, activityType domain.ActivityType, quantity, unitPrice float64) *domain.BillableActivity {
	activity, _ := domain.NewBillableActivity(
		"TNT-001",
		sellerID,
		"FAC-001",
		activityType,
		"Test activity",
		quantity,
		unitPrice,
		"order",
		"ORD-001",
	)
	return activity
}

func createTestInvoice(sellerID string) *domain.Invoice {
	now := time.Now().UTC()
	return domain.NewInvoice(
		"TNT-001",
		sellerID,
		now.AddDate(0, -1, 0),
		now,
		"Test Seller",
		"billing@test.com",
	)
}

func setupActivityRepository(t *testing.T) (*mongodb.BillableActivityRepository, *sharedtesting.MongoDBContainer, func()) {
	ctx := context.Background()

	// Start MongoDB container
	mongoContainer, err := sharedtesting.NewMongoDBContainer(ctx)
	require.NoError(t, err)

	// Get MongoDB client
	client, err := mongoContainer.GetClient(ctx)
	require.NoError(t, err)

	// Create repository
	db := client.Database("test_billing_db")
	repo := mongodb.NewBillableActivityRepository(db)

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

func setupInvoiceRepository(t *testing.T) (*mongodb.InvoiceRepository, *sharedtesting.MongoDBContainer, func()) {
	ctx := context.Background()

	// Start MongoDB container
	mongoContainer, err := sharedtesting.NewMongoDBContainer(ctx)
	require.NoError(t, err)

	// Get MongoDB client
	client, err := mongoContainer.GetClient(ctx)
	require.NoError(t, err)

	// Create repository
	db := client.Database("test_billing_db")
	repo := mongodb.NewInvoiceRepository(db, nil)

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

func setupStorageRepository(t *testing.T) (*mongodb.StorageCalculationRepository, *sharedtesting.MongoDBContainer, func()) {
	ctx := context.Background()

	// Start MongoDB container
	mongoContainer, err := sharedtesting.NewMongoDBContainer(ctx)
	require.NoError(t, err)

	// Get MongoDB client
	client, err := mongoContainer.GetClient(ctx)
	require.NoError(t, err)

	// Create repository
	db := client.Database("test_billing_db")
	repo := mongodb.NewStorageCalculationRepository(db)

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

// TestBillableActivityRepository_Save tests the Save operation
func TestBillableActivityRepository_Save(t *testing.T) {
	repo, _, cleanup := setupActivityRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Save new activity", func(t *testing.T) {
		activity := createTestActivity("SLR-001", domain.ActivityTypePick, 10, 0.25)

		err := repo.Save(ctx, activity)
		assert.NoError(t, err)

		// Verify it was saved
		found, err := repo.FindByID(ctx, activity.ActivityID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, activity.ActivityID, found.ActivityID)
		assert.Equal(t, "SLR-001", found.SellerID)
		assert.Equal(t, domain.ActivityTypePick, found.Type)
		assert.Equal(t, 2.5, found.Amount)
	})

	t.Run("Update existing activity (upsert)", func(t *testing.T) {
		activity := createTestActivity("SLR-002", domain.ActivityTypePack, 5, 1.50)

		// Save first time
		err := repo.Save(ctx, activity)
		require.NoError(t, err)

		// Update and save again
		activity.Description = "Updated description"
		err = repo.Save(ctx, activity)
		assert.NoError(t, err)

		// Verify update
		found, err := repo.FindByID(ctx, activity.ActivityID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "Updated description", found.Description)
	})
}

// TestBillableActivityRepository_SaveAll tests saving multiple activities
func TestBillableActivityRepository_SaveAll(t *testing.T) {
	repo, _, cleanup := setupActivityRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Save multiple activities", func(t *testing.T) {
		activities := []*domain.BillableActivity{
			createTestActivity("SLR-003", domain.ActivityTypePick, 10, 0.25),
			createTestActivity("SLR-003", domain.ActivityTypePack, 5, 1.50),
			createTestActivity("SLR-003", domain.ActivityTypeShipping, 2, 5.00),
		}

		err := repo.SaveAll(ctx, activities)
		assert.NoError(t, err)

		// Verify all were saved
		pagination := domain.Pagination{Page: 1, PageSize: 10}
		found, err := repo.FindBySellerID(ctx, "SLR-003", pagination)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(found), 3)
	})

	t.Run("Save empty list", func(t *testing.T) {
		err := repo.SaveAll(ctx, []*domain.BillableActivity{})
		assert.NoError(t, err)
	})
}

// TestBillableActivityRepository_FindByID tests finding an activity by ID
func TestBillableActivityRepository_FindByID(t *testing.T) {
	repo, _, cleanup := setupActivityRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Find existing activity", func(t *testing.T) {
		activity := createTestActivity("SLR-004", domain.ActivityTypePick, 10, 0.25)
		err := repo.Save(ctx, activity)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, activity.ActivityID)
		assert.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, activity.ActivityID, found.ActivityID)
	})

	t.Run("Find non-existent activity", func(t *testing.T) {
		found, err := repo.FindByID(ctx, "ACT-NONEXISTENT")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

// TestBillableActivityRepository_FindBySellerID tests finding activities by seller
func TestBillableActivityRepository_FindBySellerID(t *testing.T) {
	repo, _, cleanup := setupActivityRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sellerID := "SLR-005"

	// Create activities for the seller
	for i := 0; i < 5; i++ {
		activity := createTestActivity(sellerID, domain.ActivityTypePick, float64(i+1), 0.25)
		err := repo.Save(ctx, activity)
		require.NoError(t, err)
	}

	t.Run("Find all activities for seller", func(t *testing.T) {
		pagination := domain.Pagination{Page: 1, PageSize: 10}
		activities, err := repo.FindBySellerID(ctx, sellerID, pagination)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(activities), 5)

		// Verify all belong to the seller
		for _, a := range activities {
			assert.Equal(t, sellerID, a.SellerID)
		}
	})

	t.Run("Find with pagination", func(t *testing.T) {
		pagination := domain.Pagination{Page: 1, PageSize: 2}
		activities, err := repo.FindBySellerID(ctx, sellerID, pagination)
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(activities), 2)
	})

	t.Run("Find for non-existent seller", func(t *testing.T) {
		pagination := domain.Pagination{Page: 1, PageSize: 10}
		activities, err := repo.FindBySellerID(ctx, "SLR-NONEXISTENT", pagination)
		assert.NoError(t, err)
		assert.Empty(t, activities)
	})
}

// TestBillableActivityRepository_FindUninvoiced tests finding uninvoiced activities
func TestBillableActivityRepository_FindUninvoiced(t *testing.T) {
	repo, _, cleanup := setupActivityRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sellerID := "SLR-006"
	now := time.Now().UTC()
	periodStart := now.AddDate(0, -1, 0)
	periodEnd := now

	// Create uninvoiced activities
	for i := 0; i < 3; i++ {
		activity := createTestActivity(sellerID, domain.ActivityTypePick, float64(i+1), 0.25)
		activity.Invoiced = false
		activity.ActivityDate = now.AddDate(0, 0, -i)
		err := repo.Save(ctx, activity)
		require.NoError(t, err)
	}

	t.Run("Find uninvoiced activities", func(t *testing.T) {
		activities, err := repo.FindUninvoiced(ctx, sellerID, periodStart, periodEnd)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(activities), 3)

		// Verify all are uninvoiced
		for _, a := range activities {
			assert.False(t, a.Invoiced)
		}
	})
}

// TestBillableActivityRepository_MarkAsInvoiced tests marking activities as invoiced
func TestBillableActivityRepository_MarkAsInvoiced(t *testing.T) {
	repo, _, cleanup := setupActivityRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create activities
	activity1 := createTestActivity("SLR-007", domain.ActivityTypePick, 10, 0.25)
	activity2 := createTestActivity("SLR-007", domain.ActivityTypePack, 5, 1.50)
	err := repo.Save(ctx, activity1)
	require.NoError(t, err)
	err = repo.Save(ctx, activity2)
	require.NoError(t, err)

	t.Run("Mark activities as invoiced", func(t *testing.T) {
		activityIDs := []string{activity1.ActivityID, activity2.ActivityID}
		invoiceID := "INV-001"

		err := repo.MarkAsInvoiced(ctx, activityIDs, invoiceID)
		assert.NoError(t, err)

		// Verify activities are marked
		found1, err := repo.FindByID(ctx, activity1.ActivityID)
		require.NoError(t, err)
		assert.True(t, found1.Invoiced)
		assert.Equal(t, &invoiceID, found1.InvoiceID)

		found2, err := repo.FindByID(ctx, activity2.ActivityID)
		require.NoError(t, err)
		assert.True(t, found2.Invoiced)
		assert.Equal(t, &invoiceID, found2.InvoiceID)
	})
}

// TestBillableActivityRepository_SumBySellerAndType tests summing by type
func TestBillableActivityRepository_SumBySellerAndType(t *testing.T) {
	repo, _, cleanup := setupActivityRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sellerID := "SLR-008"
	now := time.Now().UTC()

	// Create activities of different types
	activity1 := createTestActivity(sellerID, domain.ActivityTypePick, 10, 0.25)
	activity1.ActivityDate = now.AddDate(0, 0, -1)
	err := repo.Save(ctx, activity1)
	require.NoError(t, err)

	activity2 := createTestActivity(sellerID, domain.ActivityTypePick, 20, 0.25)
	activity2.ActivityDate = now.AddDate(0, 0, -2)
	err = repo.Save(ctx, activity2)
	require.NoError(t, err)

	activity3 := createTestActivity(sellerID, domain.ActivityTypePack, 5, 1.50)
	activity3.ActivityDate = now.AddDate(0, 0, -3)
	err = repo.Save(ctx, activity3)
	require.NoError(t, err)

	t.Run("Sum by seller and type", func(t *testing.T) {
		periodStart := now.AddDate(0, -1, 0)
		periodEnd := now

		sums, err := repo.SumBySellerAndType(ctx, sellerID, periodStart, periodEnd)
		assert.NoError(t, err)
		assert.NotEmpty(t, sums)

		// Pick: 10*0.25 + 20*0.25 = 7.5
		assert.Equal(t, 7.5, sums[domain.ActivityTypePick])
		// Pack: 5*1.50 = 7.5
		assert.Equal(t, 7.5, sums[domain.ActivityTypePack])
	})
}

// TestInvoiceRepository_Save tests the Save operation
func TestInvoiceRepository_Save(t *testing.T) {
	repo, _, cleanup := setupInvoiceRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Save new invoice", func(t *testing.T) {
		invoice := createTestInvoice("SLR-010")

		err := repo.Save(ctx, invoice)
		assert.NoError(t, err)

		// Verify it was saved
		found, err := repo.FindByID(ctx, invoice.InvoiceID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, invoice.InvoiceID, found.InvoiceID)
		assert.Equal(t, "SLR-010", found.SellerID)
		assert.Equal(t, domain.InvoiceStatusDraft, found.Status)
	})

	t.Run("Update existing invoice (upsert)", func(t *testing.T) {
		invoice := createTestInvoice("SLR-011")
		err := repo.Save(ctx, invoice)
		require.NoError(t, err)

		// Add line item and save again
		invoice.AddLineItem(domain.ActivityTypePick, "Picking fees", 100, 0.25, []string{"ACT-001"})
		err = repo.Save(ctx, invoice)
		assert.NoError(t, err)

		// Verify update
		found, err := repo.FindByID(ctx, invoice.InvoiceID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Len(t, found.LineItems, 1)
		assert.Equal(t, float64(25), found.Subtotal)
	})
}

// TestInvoiceRepository_FindByID tests finding an invoice by ID
func TestInvoiceRepository_FindByID(t *testing.T) {
	repo, _, cleanup := setupInvoiceRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Find existing invoice", func(t *testing.T) {
		invoice := createTestInvoice("SLR-012")
		err := repo.Save(ctx, invoice)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, invoice.InvoiceID)
		assert.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, invoice.InvoiceID, found.InvoiceID)
	})

	t.Run("Find non-existent invoice", func(t *testing.T) {
		found, err := repo.FindByID(ctx, "INV-NONEXISTENT")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

// TestInvoiceRepository_FindBySellerID tests finding invoices by seller
func TestInvoiceRepository_FindBySellerID(t *testing.T) {
	repo, _, cleanup := setupInvoiceRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sellerID := "SLR-013"

	// Create invoices for the seller
	for i := 0; i < 3; i++ {
		invoice := createTestInvoice(sellerID)
		err := repo.Save(ctx, invoice)
		require.NoError(t, err)
	}

	t.Run("Find all invoices for seller", func(t *testing.T) {
		pagination := domain.Pagination{Page: 1, PageSize: 10}
		invoices, err := repo.FindBySellerID(ctx, sellerID, pagination)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(invoices), 3)

		// Verify all belong to the seller
		for _, inv := range invoices {
			assert.Equal(t, sellerID, inv.SellerID)
		}
	})

	t.Run("Find for non-existent seller", func(t *testing.T) {
		pagination := domain.Pagination{Page: 1, PageSize: 10}
		invoices, err := repo.FindBySellerID(ctx, "SLR-NONEXISTENT", pagination)
		assert.NoError(t, err)
		assert.Empty(t, invoices)
	})
}

// TestInvoiceRepository_FindByStatus tests finding invoices by status
func TestInvoiceRepository_FindByStatus(t *testing.T) {
	repo, _, cleanup := setupInvoiceRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create invoices with different statuses
	draftInvoice := createTestInvoice("SLR-014")
	err := repo.Save(ctx, draftInvoice)
	require.NoError(t, err)

	finalizedInvoice := createTestInvoice("SLR-014")
	finalizedInvoice.AddLineItem(domain.ActivityTypePick, "Pick", 10, 0.25, nil)
	finalizedInvoice.Finalize()
	err = repo.Save(ctx, finalizedInvoice)
	require.NoError(t, err)

	t.Run("Find draft invoices", func(t *testing.T) {
		pagination := domain.Pagination{Page: 1, PageSize: 10}
		invoices, err := repo.FindByStatus(ctx, domain.InvoiceStatusDraft, pagination)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(invoices), 1)

		for _, inv := range invoices {
			assert.Equal(t, domain.InvoiceStatusDraft, inv.Status)
		}
	})

	t.Run("Find finalized invoices", func(t *testing.T) {
		pagination := domain.Pagination{Page: 1, PageSize: 10}
		invoices, err := repo.FindByStatus(ctx, domain.InvoiceStatusFinalized, pagination)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(invoices), 1)

		for _, inv := range invoices {
			assert.Equal(t, domain.InvoiceStatusFinalized, inv.Status)
		}
	})
}

// TestInvoiceRepository_FindOverdue tests finding overdue invoices
func TestInvoiceRepository_FindOverdue(t *testing.T) {
	repo, _, cleanup := setupInvoiceRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create overdue invoice
	overdueInvoice := createTestInvoice("SLR-015")
	overdueInvoice.AddLineItem(domain.ActivityTypePick, "Pick", 10, 0.25, nil)
	overdueInvoice.Finalize()
	overdueInvoice.DueDate = time.Now().Add(-24 * time.Hour) // Past due
	err := repo.Save(ctx, overdueInvoice)
	require.NoError(t, err)

	// Create non-overdue invoice
	currentInvoice := createTestInvoice("SLR-015")
	currentInvoice.AddLineItem(domain.ActivityTypePick, "Pick", 10, 0.25, nil)
	currentInvoice.Finalize()
	currentInvoice.DueDate = time.Now().Add(24 * time.Hour) // Future
	err = repo.Save(ctx, currentInvoice)
	require.NoError(t, err)

	t.Run("Find overdue invoices", func(t *testing.T) {
		invoices, err := repo.FindOverdue(ctx)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(invoices), 1)

		// Verify all are past due
		for _, inv := range invoices {
			assert.True(t, time.Now().After(inv.DueDate))
			assert.Equal(t, domain.InvoiceStatusFinalized, inv.Status)
		}
	})
}

// TestInvoiceRepository_FindByPeriod tests finding invoices by period
func TestInvoiceRepository_FindByPeriod(t *testing.T) {
	repo, _, cleanup := setupInvoiceRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sellerID := "SLR-016"
	now := time.Now().UTC()
	periodStart := now.AddDate(0, -1, 0)
	periodEnd := now

	// Create invoice for the period
	invoice := domain.NewInvoice("TNT-001", sellerID, periodStart, periodEnd, "Test Seller", "billing@test.com")
	err := repo.Save(ctx, invoice)
	require.NoError(t, err)

	t.Run("Find invoice by period", func(t *testing.T) {
		found, err := repo.FindByPeriod(ctx, sellerID, periodStart, periodEnd)
		assert.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, sellerID, found.SellerID)
	})

	t.Run("Find for non-existent period", func(t *testing.T) {
		differentStart := now.AddDate(0, -2, 0)
		differentEnd := now.AddDate(0, -1, 0)
		found, err := repo.FindByPeriod(ctx, sellerID, differentStart, differentEnd)
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

// TestInvoiceRepository_UpdateStatus tests updating invoice status
func TestInvoiceRepository_UpdateStatus(t *testing.T) {
	repo, _, cleanup := setupInvoiceRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Update invoice status", func(t *testing.T) {
		invoice := createTestInvoice("SLR-017")
		invoice.AddLineItem(domain.ActivityTypePick, "Pick", 10, 0.25, nil)
		err := repo.Save(ctx, invoice)
		require.NoError(t, err)

		err = repo.UpdateStatus(ctx, invoice.InvoiceID, domain.InvoiceStatusFinalized)
		assert.NoError(t, err)

		// Verify status was updated
		found, err := repo.FindByID(ctx, invoice.InvoiceID)
		require.NoError(t, err)
		assert.Equal(t, domain.InvoiceStatusFinalized, found.Status)
	})

	t.Run("Update non-existent invoice", func(t *testing.T) {
		err := repo.UpdateStatus(ctx, "INV-NONEXISTENT", domain.InvoiceStatusFinalized)
		assert.Error(t, err)
	})
}

// TestStorageCalculationRepository_Save tests the Save operation
func TestStorageCalculationRepository_Save(t *testing.T) {
	repo, _, cleanup := setupStorageRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Save new storage calculation", func(t *testing.T) {
		calc := domain.NewStorageCalculation(
			"TNT-001", "SLR-020", "FAC-001",
			time.Now().UTC(),
			500, 0.05,
		)

		err := repo.Save(ctx, calc)
		assert.NoError(t, err)

		// Verify it was saved
		found, err := repo.FindBySellerAndDate(ctx, "SLR-020", time.Now().UTC())
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "SLR-020", found.SellerID)
		assert.Equal(t, float64(25), found.TotalAmount)
	})

	t.Run("Update existing calculation (upsert by seller and date)", func(t *testing.T) {
		date := time.Now().UTC()
		calc1 := domain.NewStorageCalculation("TNT-001", "SLR-021", "FAC-001", date, 100, 0.10)
		err := repo.Save(ctx, calc1)
		require.NoError(t, err)

		// Save with same seller and date but different values
		calc2 := domain.NewStorageCalculation("TNT-001", "SLR-021", "FAC-001", date, 200, 0.10)
		err = repo.Save(ctx, calc2)
		assert.NoError(t, err)

		// Verify the update
		found, err := repo.FindBySellerAndDate(ctx, "SLR-021", date)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, float64(200), found.TotalCubicFeet)
		assert.Equal(t, float64(20), found.TotalAmount)
	})
}

// TestStorageCalculationRepository_FindBySellerAndPeriod tests finding by period
func TestStorageCalculationRepository_FindBySellerAndPeriod(t *testing.T) {
	repo, _, cleanup := setupStorageRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sellerID := "SLR-022"
	now := time.Now().UTC()

	// Create calculations for different days
	for i := 0; i < 5; i++ {
		date := now.AddDate(0, 0, -i)
		calc := domain.NewStorageCalculation("TNT-001", sellerID, "FAC-001", date, float64(100+i*10), 0.05)
		err := repo.Save(ctx, calc)
		require.NoError(t, err)
	}

	t.Run("Find calculations for period", func(t *testing.T) {
		start := now.AddDate(0, 0, -3)
		end := now

		calcs, err := repo.FindBySellerAndPeriod(ctx, sellerID, start, end)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(calcs), 3)

		for _, calc := range calcs {
			assert.Equal(t, sellerID, calc.SellerID)
		}
	})
}

// TestStorageCalculationRepository_SumByPeriod tests summing by period
func TestStorageCalculationRepository_SumByPeriod(t *testing.T) {
	repo, _, cleanup := setupStorageRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sellerID := "SLR-023"
	now := time.Now().UTC()

	// Create calculations: 100*0.05=5, 150*0.05=7.5, 200*0.05=10 = 22.5 total
	for i, cubicFeet := range []float64{100, 150, 200} {
		date := now.AddDate(0, 0, -i)
		calc := domain.NewStorageCalculation("TNT-001", sellerID, "FAC-001", date, cubicFeet, 0.05)
		err := repo.Save(ctx, calc)
		require.NoError(t, err)
	}

	t.Run("Sum by period", func(t *testing.T) {
		start := now.AddDate(0, 0, -5)
		end := now.AddDate(0, 0, 1)

		total, err := repo.SumByPeriod(ctx, sellerID, start, end)
		assert.NoError(t, err)
		assert.Equal(t, 22.5, total)
	})

	t.Run("Sum for non-existent seller", func(t *testing.T) {
		total, err := repo.SumByPeriod(ctx, "SLR-NONEXISTENT", now.AddDate(0, -1, 0), now)
		assert.NoError(t, err)
		assert.Equal(t, float64(0), total)
	})
}
