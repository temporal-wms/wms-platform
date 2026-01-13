package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDomainEvents_Metadata(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		eventType string
		event     DomainEvent
	}{
		{
			name:      "inventory_received",
			eventType: "wms.inventory.received",
			event:     &InventoryReceivedEvent{ReceivedAt: now},
		},
		{
			name:      "inventory_reserved",
			eventType: "wms.inventory.reserved",
			event:     &InventoryReservedEvent{ReservedAt: now},
		},
		{
			name:      "inventory_adjusted",
			eventType: "wms.inventory.adjusted",
			event:     &InventoryAdjustedEvent{AdjustedAt: now},
		},
		{
			name:      "low_stock_alert",
			eventType: "wms.inventory.low-stock-alert",
			event:     &LowStockAlertEvent{AlertedAt: now},
		},
		{
			name:      "inventory_staged",
			eventType: "wms.inventory.staged",
			event:     &InventoryStagedEvent{StagedAt: now},
		},
		{
			name:      "inventory_packed",
			eventType: "wms.inventory.packed",
			event:     &InventoryPackedEvent{PackedAt: now},
		},
		{
			name:      "inventory_shipped",
			eventType: "wms.inventory.shipped",
			event:     &InventoryShippedEvent{ShippedAt: now},
		},
		{
			name:      "inventory_returned",
			eventType: "wms.inventory.returned-to-shelf",
			event:     &InventoryReturnedToShelfEvent{ReturnedAt: now},
		},
		{
			name:      "stock_shortage",
			eventType: "wms.inventory.stock-shortage",
			event:     &StockShortageEvent{OccurredAt_: now},
		},
		{
			name:      "inventory_discrepancy",
			eventType: "wms.inventory.discrepancy",
			event:     &InventoryDiscrepancyEvent{DetectedAt: now},
		},
		{
			name:      "backorder_created",
			eventType: "wms.inventory.backorder-created",
			event:     &BackorderCreatedEvent{CreatedAt: now},
		},
		{
			name:      "velocity_changed",
			eventType: "wms.inventory.velocity-class-changed",
			event:     &VelocityClassChangedEvent{ChangedAt: now},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.eventType, tt.event.EventType())
			assert.Equal(t, now, tt.event.OccurredAt())
		})
	}
}
