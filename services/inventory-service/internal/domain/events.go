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

func (e *LowStockAlertEvent) EventType() string     { return "wms.inventory.low-stock-alert" }
func (e *LowStockAlertEvent) OccurredAt() time.Time { return e.AlertedAt }

// InventoryStagedEvent is published when inventory is physically staged
type InventoryStagedEvent struct {
	SKU               string    `json:"sku"`
	AllocationID      string    `json:"allocationId"`
	OrderID           string    `json:"orderId"`
	Quantity          int       `json:"quantity"`
	SourceLocationID  string    `json:"sourceLocationId"`
	StagingLocationID string    `json:"stagingLocationId"`
	StagedBy          string    `json:"stagedBy"`
	StagedAt          time.Time `json:"stagedAt"`
}

func (e *InventoryStagedEvent) EventType() string     { return "wms.inventory.staged" }
func (e *InventoryStagedEvent) OccurredAt() time.Time { return e.StagedAt }

// InventoryPackedEvent is published when staged inventory is packed
type InventoryPackedEvent struct {
	SKU          string    `json:"sku"`
	AllocationID string    `json:"allocationId"`
	OrderID      string    `json:"orderId"`
	PackedBy     string    `json:"packedBy"`
	PackedAt     time.Time `json:"packedAt"`
}

func (e *InventoryPackedEvent) EventType() string     { return "wms.inventory.packed" }
func (e *InventoryPackedEvent) OccurredAt() time.Time { return e.PackedAt }

// InventoryShippedEvent is published when inventory is shipped
type InventoryShippedEvent struct {
	SKU          string    `json:"sku"`
	AllocationID string    `json:"allocationId"`
	OrderID      string    `json:"orderId"`
	Quantity     int       `json:"quantity"`
	ShippedAt    time.Time `json:"shippedAt"`
}

func (e *InventoryShippedEvent) EventType() string     { return "wms.inventory.shipped" }
func (e *InventoryShippedEvent) OccurredAt() time.Time { return e.ShippedAt }

// InventoryReturnedToShelfEvent is published when hard allocated inventory is returned
type InventoryReturnedToShelfEvent struct {
	SKU              string    `json:"sku"`
	AllocationID     string    `json:"allocationId"`
	OrderID          string    `json:"orderId"`
	Quantity         int       `json:"quantity"`
	SourceLocationID string    `json:"sourceLocationId"`
	ReturnedBy       string    `json:"returnedBy"`
	Reason           string    `json:"reason"`
	ReturnedAt       time.Time `json:"returnedAt"`
}

func (e *InventoryReturnedToShelfEvent) EventType() string     { return "wms.inventory.returned-to-shelf" }
func (e *InventoryReturnedToShelfEvent) OccurredAt() time.Time { return e.ReturnedAt }
