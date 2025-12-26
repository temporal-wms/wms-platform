package domain

import "time"

// DomainEvent is the interface for all domain events
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// InventoryReceivedEvent is published when inventory is received
type InventoryReceivedEvent struct {
	SKU        string    `json:"sku"`
	Quantity   int       `json:"quantity"`
	LocationID string    `json:"locationId"`
	ReceivedAt time.Time `json:"receivedAt"`
}

func (e *InventoryReceivedEvent) EventType() string    { return "wms.inventory.received" }
func (e *InventoryReceivedEvent) OccurredAt() time.Time { return e.ReceivedAt }

// InventoryAdjustedEvent is published when inventory is adjusted
type InventoryAdjustedEvent struct {
	SKU         string    `json:"sku"`
	LocationID  string    `json:"locationId"`
	OldQuantity int       `json:"oldQuantity"`
	NewQuantity int       `json:"newQuantity"`
	Reason      string    `json:"reason"`
	AdjustedAt  time.Time `json:"adjustedAt"`
}

func (e *InventoryAdjustedEvent) EventType() string    { return "wms.inventory.adjusted" }
func (e *InventoryAdjustedEvent) OccurredAt() time.Time { return e.AdjustedAt }

// LowStockAlertEvent is published when stock falls below reorder point
type LowStockAlertEvent struct {
	SKU             string    `json:"sku"`
	CurrentQuantity int       `json:"currentQuantity"`
	ReorderPoint    int       `json:"reorderPoint"`
	AlertedAt       time.Time `json:"alertedAt"`
}

func (e *LowStockAlertEvent) EventType() string    { return "wms.inventory.low-stock-alert" }
func (e *LowStockAlertEvent) OccurredAt() time.Time { return e.AlertedAt }
