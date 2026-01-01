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

// StockShortageEvent is published when a confirmed shortage is discovered during picking
type StockShortageEvent struct {
	SKU              string    `json:"sku"`
	LocationID       string    `json:"locationId"`
	OrderID          string    `json:"orderId"`
	ExpectedQuantity int       `json:"expectedQuantity"`
	ActualQuantity   int       `json:"actualQuantity"`
	ShortageQuantity int       `json:"shortageQuantity"`
	ReportedBy       string    `json:"reportedBy"`
	Reason           string    `json:"reason"` // not_found, damaged, quantity_mismatch
	OccurredAt_      time.Time `json:"occurredAt"`
}

func (e *StockShortageEvent) EventType() string     { return "wms.inventory.stock-shortage" }
func (e *StockShortageEvent) OccurredAt() time.Time { return e.OccurredAt_ }

// InventoryDiscrepancyEvent is published for audit trail when actual != expected
type InventoryDiscrepancyEvent struct {
	SKU             string    `json:"sku"`
	LocationID      string    `json:"locationId"`
	SystemQuantity  int       `json:"systemQuantity"`
	ActualQuantity  int       `json:"actualQuantity"`
	DiscrepancyType string    `json:"discrepancyType"` // shortage, overage
	Source          string    `json:"source"`          // picking, cycle_count, receiving
	ReferenceID     string    `json:"referenceId"`     // orderId, countId, etc.
	DetectedAt      time.Time `json:"detectedAt"`
}

func (e *InventoryDiscrepancyEvent) EventType() string     { return "wms.inventory.discrepancy" }
func (e *InventoryDiscrepancyEvent) OccurredAt() time.Time { return e.DetectedAt }

// BackorderCreatedEvent is published when a backorder is created for shortage items
type BackorderCreatedEvent struct {
	BackorderID     string    `json:"backorderId"`
	OriginalOrderID string    `json:"originalOrderId"`
	SKU             string    `json:"sku"`
	Quantity        int       `json:"quantity"`
	Priority        int       `json:"priority"`
	CreatedAt       time.Time `json:"createdAt"`
}

func (e *BackorderCreatedEvent) EventType() string     { return "wms.inventory.backorder-created" }
func (e *BackorderCreatedEvent) OccurredAt() time.Time { return e.CreatedAt }
