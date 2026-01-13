package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewInventoryTransaction(t *testing.T) {
	txn := NewInventoryTransaction(
		"TXN-1",
		"SKU-1",
		"receive",
		5,
		"LOC-1",
		"REF-1",
		"reason",
		"user1",
		&TransactionTenantInfo{
			TenantID:    "T-1",
			FacilityID:  "F-1",
			WarehouseID: "W-1",
			SellerID:    "S-1",
		},
	)

	assert.Equal(t, "TXN-1", txn.TransactionID)
	assert.Equal(t, "SKU-1", txn.SKU)
	assert.Equal(t, "receive", txn.Type)
	assert.Equal(t, "T-1", txn.TenantID)
}

func TestInventoryTransactionAggregate_EventsLifecycle(t *testing.T) {
	txn := NewInventoryTransaction("TXN-2", "SKU-2", "adjust", 1, "LOC-2", "REF-2", "", "user1", nil)
	txn.AddDomainEvent(&InventoryAdjustedEvent{SKU: "SKU-2"})
	assert.Len(t, txn.DomainEvents, 1)
	txn.ClearDomainEvents()
	assert.Len(t, txn.DomainEvents, 0)
}
