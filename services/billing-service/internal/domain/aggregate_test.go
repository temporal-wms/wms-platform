package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewBillableActivity tests billable activity creation
func TestNewBillableActivity(t *testing.T) {
	tests := []struct {
		name          string
		tenantID      string
		sellerID      string
		facilityID    string
		activityType  ActivityType
		description   string
		quantity      float64
		unitPrice     float64
		referenceType string
		referenceID   string
		expectError   error
	}{
		{
			name:          "Valid pick activity",
			tenantID:      "TNT-001",
			sellerID:      "SLR-001",
			facilityID:    "FAC-001",
			activityType:  ActivityTypePick,
			description:   "Pick fee",
			quantity:      10,
			unitPrice:     0.25,
			referenceType: "order",
			referenceID:   "ORD-001",
			expectError:   nil,
		},
		{
			name:          "Valid storage activity",
			tenantID:      "TNT-001",
			sellerID:      "SLR-001",
			facilityID:    "FAC-001",
			activityType:  ActivityTypeStorage,
			description:   "Daily storage",
			quantity:      500,
			unitPrice:     0.05,
			referenceType: "inventory",
			referenceID:   "INV-001",
			expectError:   nil,
		},
		{
			name:          "Invalid activity type",
			tenantID:      "TNT-001",
			sellerID:      "SLR-001",
			facilityID:    "FAC-001",
			activityType:  ActivityType("invalid"),
			description:   "Invalid",
			quantity:      10,
			unitPrice:     1.0,
			referenceType: "order",
			referenceID:   "ORD-001",
			expectError:   ErrInvalidActivityType,
		},
		{
			name:          "Negative quantity with positive price is valid",
			tenantID:      "TNT-001",
			sellerID:      "SLR-001",
			facilityID:    "FAC-001",
			activityType:  ActivityTypePick,
			description:   "Credit",
			quantity:      -5,
			unitPrice:     0.25,
			referenceType: "order",
			referenceID:   "ORD-001",
			expectError:   ErrInvalidAmount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			activity, err := NewBillableActivity(
				tt.tenantID, tt.sellerID, tt.facilityID,
				tt.activityType, tt.description,
				tt.quantity, tt.unitPrice,
				tt.referenceType, tt.referenceID,
			)

			if tt.expectError != nil {
				assert.Equal(t, tt.expectError, err)
				assert.Nil(t, activity)
			} else {
				require.NoError(t, err)
				require.NotNil(t, activity)
				assert.NotEmpty(t, activity.ActivityID)
				assert.Equal(t, tt.tenantID, activity.TenantID)
				assert.Equal(t, tt.sellerID, activity.SellerID)
				assert.Equal(t, tt.facilityID, activity.FacilityID)
				assert.Equal(t, tt.activityType, activity.Type)
				assert.Equal(t, tt.description, activity.Description)
				assert.Equal(t, tt.quantity, activity.Quantity)
				assert.Equal(t, tt.unitPrice, activity.UnitPrice)
				assert.Equal(t, tt.quantity*tt.unitPrice, activity.Amount)
				assert.Equal(t, "USD", activity.Currency)
				assert.False(t, activity.Invoiced)
				assert.NotZero(t, activity.CreatedAt)
			}
		})
	}
}

// TestActivityTypeIsValid tests activity type validation
func TestActivityTypeIsValid(t *testing.T) {
	validTypes := []ActivityType{
		ActivityTypeStorage,
		ActivityTypePick,
		ActivityTypePack,
		ActivityTypeReceiving,
		ActivityTypeShipping,
		ActivityTypeReturnProcessing,
		ActivityTypeGiftWrap,
		ActivityTypeHazmat,
		ActivityTypeOversized,
		ActivityTypeColdChain,
		ActivityTypeFragile,
		ActivityTypeSpecialHandling,
	}

	for _, at := range validTypes {
		assert.True(t, at.IsValid(), "Expected %s to be valid", at)
	}

	assert.False(t, ActivityType("invalid").IsValid())
}

// TestNewInvoice tests invoice creation
func TestNewInvoice(t *testing.T) {
	now := time.Now().UTC()
	periodStart := now.AddDate(0, -1, 0)
	periodEnd := now

	invoice := NewInvoice("TNT-001", "SLR-001", periodStart, periodEnd, "Acme Corp", "billing@acme.com")

	require.NotNil(t, invoice)
	assert.NotEmpty(t, invoice.InvoiceID)
	assert.NotEmpty(t, invoice.InvoiceNumber)
	assert.Equal(t, "TNT-001", invoice.TenantID)
	assert.Equal(t, "SLR-001", invoice.SellerID)
	assert.Equal(t, InvoiceStatusDraft, invoice.Status)
	assert.Equal(t, periodStart, invoice.PeriodStart)
	assert.Equal(t, periodEnd, invoice.PeriodEnd)
	assert.Empty(t, invoice.LineItems)
	assert.Equal(t, float64(0), invoice.Subtotal)
	assert.Equal(t, float64(0), invoice.Total)
	assert.Equal(t, "USD", invoice.Currency)
	assert.NotZero(t, invoice.DueDate)
	assert.Equal(t, "Acme Corp", invoice.SellerName)
	assert.Equal(t, "billing@acme.com", invoice.SellerEmail)

	// Should have creation event
	events := invoice.DomainEvents()
	assert.Len(t, events, 1)
}

// TestInvoiceAddLineItem tests adding line items
func TestInvoiceAddLineItem(t *testing.T) {
	invoice := createTestInvoice()

	// Add first line item
	err := invoice.AddLineItem(ActivityTypePick, "Picking fees", 100, 0.25, []string{"ACT-001"})
	assert.NoError(t, err)
	assert.Len(t, invoice.LineItems, 1)
	assert.Equal(t, float64(25), invoice.Subtotal)
	assert.Equal(t, float64(25), invoice.Total)

	// Add different type
	err = invoice.AddLineItem(ActivityTypePack, "Packing fees", 50, 1.50, []string{"ACT-002"})
	assert.NoError(t, err)
	assert.Len(t, invoice.LineItems, 2)
	assert.Equal(t, float64(100), invoice.Subtotal) // 25 + 75

	// Add same type - should aggregate
	err = invoice.AddLineItem(ActivityTypePick, "More picking", 50, 0.25, []string{"ACT-003"})
	assert.NoError(t, err)
	assert.Len(t, invoice.LineItems, 2) // Still 2 line items
	assert.Equal(t, float64(112.50), invoice.Subtotal) // 100 + 12.50

	// Verify aggregation
	for _, item := range invoice.LineItems {
		if item.ActivityType == ActivityTypePick {
			assert.Equal(t, float64(150), item.Quantity)
			assert.Equal(t, float64(37.50), item.Amount)
			assert.Len(t, item.ActivityIDs, 2)
		}
	}
}

// TestInvoiceAddLineItemFinalized tests adding to finalized invoice
func TestInvoiceAddLineItemFinalized(t *testing.T) {
	invoice := createTestInvoice()
	invoice.AddLineItem(ActivityTypePick, "Picking", 10, 0.25, []string{"ACT-001"})
	invoice.Finalize()

	err := invoice.AddLineItem(ActivityTypePack, "Packing", 5, 1.50, nil)
	assert.Equal(t, ErrInvoiceAlreadyFinalized, err)
}

// TestInvoiceSetTaxRate tests setting tax rate
func TestInvoiceSetTaxRate(t *testing.T) {
	invoice := createTestInvoice()
	invoice.AddLineItem(ActivityTypePick, "Picking", 100, 1.00, nil)

	err := invoice.SetTaxRate(0.08)
	assert.NoError(t, err)
	assert.Equal(t, 0.08, invoice.TaxRate)
	assert.Equal(t, float64(8), invoice.TaxAmount) // 100 * 0.08
	assert.Equal(t, float64(108), invoice.Total)

	// Cannot set on finalized
	invoice.Finalize()
	err = invoice.SetTaxRate(0.10)
	assert.Equal(t, ErrInvoiceAlreadyFinalized, err)
}

// TestInvoiceApplyDiscount tests applying discount
func TestInvoiceApplyDiscount(t *testing.T) {
	invoice := createTestInvoice()
	invoice.AddLineItem(ActivityTypePick, "Picking", 100, 1.00, nil)
	invoice.SetTaxRate(0.08)

	err := invoice.ApplyDiscount(10)
	assert.NoError(t, err)
	assert.Equal(t, float64(10), invoice.Discount)
	assert.Equal(t, float64(98), invoice.Total) // 100 + 8 - 10

	// Cannot apply on finalized
	invoice.Finalize()
	err = invoice.ApplyDiscount(5)
	assert.Equal(t, ErrInvoiceAlreadyFinalized, err)
}

// TestInvoiceFinalize tests invoice finalization
func TestInvoiceFinalize(t *testing.T) {
	invoice := createTestInvoice()
	invoice.AddLineItem(ActivityTypePick, "Picking", 100, 1.00, nil)
	invoice.ClearDomainEvents()

	err := invoice.Finalize()
	assert.NoError(t, err)
	assert.Equal(t, InvoiceStatusFinalized, invoice.Status)
	assert.NotNil(t, invoice.FinalizedAt)
	assert.Len(t, invoice.DomainEvents(), 1)

	// Cannot finalize again
	err = invoice.Finalize()
	assert.Equal(t, ErrInvoiceAlreadyFinalized, err)
}

// TestInvoiceMarkPaid tests marking invoice as paid
func TestInvoiceMarkPaid(t *testing.T) {
	invoice := createTestInvoice()
	invoice.AddLineItem(ActivityTypePick, "Picking", 100, 1.00, nil)

	// Cannot pay draft invoice
	err := invoice.MarkPaid("bank_transfer", "TXN-001")
	assert.Equal(t, ErrInvoiceNotFinalized, err)

	// Finalize and pay
	invoice.Finalize()
	invoice.ClearDomainEvents()

	err = invoice.MarkPaid("bank_transfer", "TXN-001")
	assert.NoError(t, err)
	assert.Equal(t, InvoiceStatusPaid, invoice.Status)
	assert.NotNil(t, invoice.PaidAt)
	assert.Equal(t, "bank_transfer", invoice.PaymentMethod)
	assert.Equal(t, "TXN-001", invoice.PaymentRef)
	assert.Len(t, invoice.DomainEvents(), 1)
}

// TestInvoiceMarkOverdue tests marking invoice as overdue
func TestInvoiceMarkOverdue(t *testing.T) {
	invoice := createTestInvoice()
	invoice.AddLineItem(ActivityTypePick, "Picking", 100, 1.00, nil)
	invoice.Finalize()
	invoice.DueDate = time.Now().Add(-24 * time.Hour) // Past due
	invoice.ClearDomainEvents()

	invoice.MarkOverdue()
	assert.Equal(t, InvoiceStatusOverdue, invoice.Status)
	assert.Len(t, invoice.DomainEvents(), 1)
}

// TestInvoiceMarkOverdueNotPastDue tests overdue with future due date
func TestInvoiceMarkOverdueNotPastDue(t *testing.T) {
	invoice := createTestInvoice()
	invoice.AddLineItem(ActivityTypePick, "Picking", 100, 1.00, nil)
	invoice.Finalize()
	invoice.DueDate = time.Now().Add(24 * time.Hour) // Future

	invoice.MarkOverdue()
	assert.Equal(t, InvoiceStatusFinalized, invoice.Status) // Should not change
}

// TestInvoiceVoid tests voiding invoice
func TestInvoiceVoid(t *testing.T) {
	invoice := createTestInvoice()
	invoice.AddLineItem(ActivityTypePick, "Picking", 100, 1.00, nil)
	invoice.Finalize()

	err := invoice.Void("Duplicate invoice")
	assert.NoError(t, err)
	assert.Equal(t, InvoiceStatusVoided, invoice.Status)
	assert.Equal(t, "Duplicate invoice", invoice.Notes)

	// Cannot void paid invoice
	paidInvoice := createTestInvoice()
	paidInvoice.AddLineItem(ActivityTypePick, "Picking", 100, 1.00, nil)
	paidInvoice.Finalize()
	paidInvoice.MarkPaid("bank_transfer", "TXN-001")

	err = paidInvoice.Void("Test")
	assert.Error(t, err)
}

// TestInvoicePayOverdue tests paying overdue invoice
func TestInvoicePayOverdue(t *testing.T) {
	invoice := createTestInvoice()
	invoice.AddLineItem(ActivityTypePick, "Picking", 100, 1.00, nil)
	invoice.Finalize()
	invoice.DueDate = time.Now().Add(-24 * time.Hour)
	invoice.MarkOverdue()

	// Should be able to pay overdue
	err := invoice.MarkPaid("bank_transfer", "TXN-001")
	assert.NoError(t, err)
	assert.Equal(t, InvoiceStatusPaid, invoice.Status)
}

// TestInvoiceDomainEvents tests domain event handling
func TestInvoiceDomainEvents(t *testing.T) {
	invoice := createTestInvoice()

	events := invoice.DomainEvents()
	assert.Len(t, events, 1)

	invoice.ClearDomainEvents()
	events = invoice.DomainEvents()
	assert.Empty(t, events)
}

// TestNewStorageCalculation tests storage calculation creation
func TestNewStorageCalculation(t *testing.T) {
	calc := NewStorageCalculation(
		"TNT-001", "SLR-001", "FAC-001",
		time.Now().UTC(),
		500, 0.05,
	)

	require.NotNil(t, calc)
	assert.NotEmpty(t, calc.CalculationID)
	assert.Equal(t, "TNT-001", calc.TenantID)
	assert.Equal(t, "SLR-001", calc.SellerID)
	assert.Equal(t, "FAC-001", calc.FacilityID)
	assert.Equal(t, float64(500), calc.TotalCubicFeet)
	assert.Equal(t, 0.05, calc.RatePerCubicFt)
	assert.Equal(t, float64(25), calc.TotalAmount) // 500 * 0.05
	assert.NotZero(t, calc.CreatedAt)
}

// TestInvoiceRecalculateTotals tests total recalculation
func TestInvoiceRecalculateTotals(t *testing.T) {
	invoice := createTestInvoice()

	// Add items
	invoice.AddLineItem(ActivityTypePick, "Picking", 100, 0.25, nil)
	invoice.AddLineItem(ActivityTypePack, "Packing", 50, 1.50, nil)

	assert.Equal(t, float64(100), invoice.Subtotal) // 25 + 75

	// Add tax
	invoice.SetTaxRate(0.10)
	assert.Equal(t, float64(10), invoice.TaxAmount)
	assert.Equal(t, float64(110), invoice.Total)

	// Add discount
	invoice.ApplyDiscount(5)
	assert.Equal(t, float64(105), invoice.Total) // 100 + 10 - 5
}

// Helper function
func createTestInvoice() *Invoice {
	now := time.Now().UTC()
	return NewInvoice("TNT-001", "SLR-001", now.AddDate(0, -1, 0), now, "Acme Corp", "billing@acme.com")
}

// BenchmarkNewBillableActivity benchmarks activity creation
func BenchmarkNewBillableActivity(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewBillableActivity(
			"TNT-001", "SLR-001", "FAC-001",
			ActivityTypePick, "Pick fee",
			10, 0.25,
			"order", "ORD-001",
		)
	}
}

// BenchmarkInvoiceAddLineItem benchmarks adding line items
func BenchmarkInvoiceAddLineItem(b *testing.B) {
	invoice := createTestInvoice()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		invoice.AddLineItem(ActivityTypePick, "Pick", 1, 0.25, nil)
	}
}

// BenchmarkInvoiceRecalculateTotals benchmarks recalculation
func BenchmarkInvoiceRecalculateTotals(b *testing.B) {
	invoice := createTestInvoice()
	invoice.AddLineItem(ActivityTypePick, "Picking", 100, 0.25, nil)
	invoice.AddLineItem(ActivityTypePack, "Packing", 50, 1.50, nil)
	invoice.AddLineItem(ActivityTypeShipping, "Shipping", 10, 5.00, nil)
	invoice.SetTaxRate(0.08)
	invoice.ApplyDiscount(10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		invoice.recalculateTotals()
	}
}
