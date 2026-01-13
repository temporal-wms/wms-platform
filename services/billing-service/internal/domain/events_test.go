package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInvoiceEvents(t *testing.T) {
	now := time.Now().UTC()

	created := &InvoiceCreatedEvent{
		InvoiceID:   "INV-001",
		SellerID:    "SLR-001",
		PeriodStart: now.Add(-24 * time.Hour),
		PeriodEnd:   now,
		CreatedAt:   now,
	}
	assert.Equal(t, "billing.invoice.created", created.EventType())
	assert.Equal(t, created.CreatedAt, created.OccurredAt())

	finalized := &InvoiceFinalizedEvent{
		InvoiceID:   "INV-001",
		SellerID:    "SLR-001",
		Total:       100,
		DueDate:     now.Add(30 * 24 * time.Hour),
		FinalizedAt: now,
	}
	assert.Equal(t, "billing.invoice.finalized", finalized.EventType())
	assert.Equal(t, finalized.FinalizedAt, finalized.OccurredAt())

	paid := &InvoicePaidEvent{
		InvoiceID:     "INV-001",
		SellerID:      "SLR-001",
		Amount:        100,
		PaymentMethod: "card",
		PaidAt:        now,
	}
	assert.Equal(t, "billing.invoice.paid", paid.EventType())
	assert.Equal(t, paid.PaidAt, paid.OccurredAt())

	overdue := &InvoiceOverdueEvent{
		InvoiceID: "INV-001",
		SellerID:  "SLR-001",
		Total:     100,
		DueDate:   now.Add(-24 * time.Hour),
	}
	before := time.Now()
	assert.Equal(t, "billing.invoice.overdue", overdue.EventType())
	assert.WithinDuration(t, before, overdue.OccurredAt(), time.Second)
}

func TestActivityAndStorageEvents(t *testing.T) {
	now := time.Now().UTC()

	activity := &ActivityRecordedEvent{
		ActivityID:    "ACT-001",
		SellerID:      "SLR-001",
		Type:          ActivityTypePick,
		Amount:        12.5,
		ReferenceType: "order",
		ReferenceID:   "ORD-001",
		RecordedAt:    now,
	}
	assert.Equal(t, "billing.activity.recorded", activity.EventType())
	assert.Equal(t, activity.RecordedAt, activity.OccurredAt())

	storage := &StorageCalculatedEvent{
		CalculationID:  "STO-001",
		SellerID:       "SLR-001",
		FacilityID:     "FAC-001",
		TotalCubicFeet: 100,
		TotalAmount:    10,
		CalculatedAt:   now,
	}
	assert.Equal(t, "billing.storage.calculated", storage.EventType())
	assert.Equal(t, storage.CalculatedAt, storage.OccurredAt())
}
