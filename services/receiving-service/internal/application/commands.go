package application

import (
	"github.com/wms-platform/services/receiving-service/internal/domain"
)

// CreateShipmentCommand represents a command to create a new inbound shipment
type CreateShipmentCommand struct {
	// Optional explicit shipment ID (will be generated if not provided)
	ShipmentID string `json:"shipmentId"`

	// Purchase order
	PurchaseOrderID string `json:"purchaseOrderId"`

	// ASN - can be provided as nested struct or flat fields
	ASN domain.AdvanceShippingNotice `json:"asn"`

	// Supplier - can be provided as nested struct or flat fields
	Supplier domain.Supplier `json:"supplier"`

	// Expected items
	ExpectedItems []domain.ExpectedItem `json:"expectedItems" binding:"required,min=1"`
}

// ExpectedItemInput represents an expected item in a shipment
type ExpectedItemInput struct {
	SKU               string  `json:"sku" binding:"required"`
	ProductName       string  `json:"productName" binding:"required"`
	ExpectedQuantity  int     `json:"expectedQuantity" binding:"required,min=1"`
	UnitCost          float64 `json:"unitCost"`
	Weight            float64 `json:"weight"`
	IsHazmat          bool    `json:"isHazmat"`
	RequiresColdChain bool    `json:"requiresColdChain"`
}

// ReceiveItemCommand represents a command to receive an item
type ReceiveItemCommand struct {
	ShipmentID string `json:"shipmentId" binding:"required"`
	SKU        string `json:"sku" binding:"required"`
	Quantity   int    `json:"quantity" binding:"required,min=1"`
	Condition  string `json:"condition" binding:"required,oneof=good damaged"` // good or damaged
	ToteID     string `json:"toteId"`
	WorkerID   string `json:"workerId" binding:"required"`
	Notes      string `json:"notes"`
}

// MarkArrivedCommand represents a command to mark a shipment as arrived
type MarkArrivedCommand struct {
	ShipmentID string `json:"shipmentId" binding:"required"`
	DockID     string `json:"dockId" binding:"required"`
}

// StartReceivingCommand represents a command to start receiving
type StartReceivingCommand struct {
	ShipmentID string `json:"shipmentId" binding:"required"`
	WorkerID   string `json:"workerId" binding:"required"`
}

// CompleteReceivingCommand represents a command to complete receiving
type CompleteReceivingCommand struct {
	ShipmentID string `json:"shipmentId" binding:"required"`
}
