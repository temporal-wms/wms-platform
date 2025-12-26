package cloudevents

import (
	"time"
)

// EventType constants for WMS domain events
const (
	// Order events
	OrderReceived   = "wms.order.received"
	OrderValidated  = "wms.order.validated"
	OrderCancelled  = "wms.order.cancelled"
	OrderCompleted  = "wms.order.completed"

	// Wave events
	WaveCreated   = "wms.wave.created"
	WaveReleased  = "wms.wave.released"
	WaveCompleted = "wms.wave.completed"

	// Routing events
	RouteCalculated = "wms.routing.route-calculated"
	RouteOptimized  = "wms.routing.route-optimized"

	// Picking events
	PickTaskAssigned  = "wms.picking.task-assigned"
	ItemPicked        = "wms.picking.item-picked"
	PickTaskCompleted = "wms.picking.task-completed"
	PickException     = "wms.picking.exception"

	// Consolidation events
	ConsolidationStarted   = "wms.consolidation.started"
	ItemConsolidated       = "wms.consolidation.item-consolidated"
	ConsolidationCompleted = "wms.consolidation.completed"

	// Packing events
	PackTaskCreated    = "wms.packing.task-created"
	PackagingSuggested = "wms.packing.packaging-suggested"
	PackageSealed      = "wms.packing.package-sealed"
	LabelApplied       = "wms.packing.label-applied"
	PackTaskCompleted  = "wms.packing.task-completed"

	// Shipping events
	ShipmentCreated    = "wms.shipping.shipment-created"
	ShipmentLabeled    = "wms.shipping.label-generated"
	ShipmentManifested = "wms.shipping.manifested"
	ShipConfirmed      = "wms.shipping.confirmed"

	// Inventory events
	InventoryReceived      = "wms.inventory.received"
	InventoryAdjusted      = "wms.inventory.adjusted"
	LowStockAlert          = "wms.inventory.low-stock-alert"
	CycleCountCompleted    = "wms.inventory.cycle-count-completed"

	// Labor events
	ShiftStarted        = "wms.labor.shift-started"
	ShiftEnded          = "wms.labor.shift-ended"
	LaborTaskAssigned   = "wms.labor.task-assigned"
	PerformanceRecorded = "wms.labor.performance-recorded"
)

// Source constants for event sources
const (
	SourceOrderManagement = "/wms/order-service"
	SourceWaving          = "/wms/waving-service"
	SourceRouting         = "/wms/routing-service"
	SourcePicking         = "/wms/picking-service"
	SourceConsolidation   = "/wms/consolidation-service"
	SourcePacking         = "/wms/packing-service"
	SourceShipping        = "/wms/shipping-service"
	SourceInventory       = "/wms/inventory-service"
	SourceLabor           = "/wms/labor-service"
)

// WMSCloudEvent represents a CloudEvents v1.0 compliant event for WMS
type WMSCloudEvent struct {
	SpecVersion     string                 `json:"specversion"`
	Type            string                 `json:"type"`
	Source          string                 `json:"source"`
	Subject         string                 `json:"subject,omitempty"`
	ID              string                 `json:"id"`
	Time            time.Time              `json:"time"`
	DataContentType string                 `json:"datacontenttype"`
	Data            interface{}            `json:"data"`
	Extensions      map[string]interface{} `json:"-"`

	// WMS-specific extensions
	CorrelationID string `json:"wmscorrelationid,omitempty"`
	WaveNumber    string `json:"wmswavenumber,omitempty"`
	WorkflowID    string `json:"wmsworkflowid,omitempty"`
}

// OrderReceivedData represents the data payload for OrderReceived event
type OrderReceivedData struct {
	OrderID             string      `json:"orderId"`
	CustomerID          string      `json:"customerId"`
	OrderLines          []OrderLine `json:"orderLines"`
	Priority            string      `json:"priority"`
	PromisedDeliveryAt  time.Time   `json:"promisedDeliveryDate"`
}

// OrderLine represents an item in an order
type OrderLine struct {
	SKU      string  `json:"sku"`
	Quantity int     `json:"quantity"`
	Weight   float64 `json:"weight"`
}

// WaveCreatedData represents the data payload for WaveCreated event
type WaveCreatedData struct {
	WaveID            string    `json:"waveId"`
	OrderIDs          []string  `json:"orderIds"`
	ScheduledStart    time.Time `json:"scheduledStart"`
	EstimatedDuration string    `json:"estimatedDuration"`
	WaveType          string    `json:"waveType"` // "digital" | "wholesale"
}

// RouteCalculatedData represents the data payload for RouteCalculated event
type RouteCalculatedData struct {
	RouteID           string         `json:"routeId"`
	PickerID          string         `json:"pickerId"`
	Stops             []LocationStop `json:"stops"`
	EstimatedDistance float64        `json:"estimatedDistance"`
	Strategy          string         `json:"strategy"`
}

// LocationStop represents a stop in a pick route
type LocationStop struct {
	LocationID string `json:"locationId"`
	Zone       string `json:"zone"`
	Aisle      string `json:"aisle"`
	Rack       string `json:"rack"`
	Level      string `json:"level"`
	Position   string `json:"position"`
}

// ItemPickedData represents the data payload for ItemPicked event
type ItemPickedData struct {
	TaskID     string `json:"taskId"`
	ItemID     string `json:"itemId"`
	SKU        string `json:"sku"`
	Quantity   int    `json:"quantity"`
	LocationID string `json:"locationId"`
	ToteID     string `json:"toteId"`
}

// ShipmentCreatedData represents the data payload for ShipmentCreated event
type ShipmentCreatedData struct {
	ShipmentID     string `json:"shipmentId"`
	OrderID        string `json:"orderId"`
	Carrier        string `json:"carrier"`
	TrackingNumber string `json:"trackingNumber,omitempty"`
}

// InventoryAdjustedData represents the data payload for InventoryAdjusted event
type InventoryAdjustedData struct {
	SKU            string `json:"sku"`
	LocationID     string `json:"locationId"`
	PreviousQty    int    `json:"previousQuantity"`
	NewQty         int    `json:"newQuantity"`
	AdjustmentType string `json:"adjustmentType"` // "pick", "receive", "cycle_count", "damage"
	Reason         string `json:"reason,omitempty"`
}

// LaborTaskAssignedData represents the data payload for LaborTaskAssigned event
type LaborTaskAssignedData struct {
	WorkerID string `json:"workerId"`
	TaskID   string `json:"taskId"`
	TaskType string `json:"taskType"` // "picking", "packing", "receiving"
	Zone     string `json:"zone"`
}
