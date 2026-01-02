package dto

import "time"

// CreateShipmentRequest represents the request to create a new inbound shipment
type CreateShipmentRequest struct {
	ShipmentID      string              `json:"shipmentId" binding:"required"`
	PurchaseOrderID string              `json:"purchaseOrderId,omitempty"`
	ASN             ASNRequest          `json:"asn" binding:"required"`
	Supplier        SupplierRequest     `json:"supplier" binding:"required"`
	ExpectedItems   []ExpectedItemRequest `json:"expectedItems" binding:"required,min=1"`
}

// ASNRequest represents the advance shipping notice
type ASNRequest struct {
	ASNID            string    `json:"asnId" binding:"required"`
	CarrierName      string    `json:"carrierName" binding:"required"`
	TrackingNumber   string    `json:"trackingNumber,omitempty"`
	ExpectedArrival  time.Time `json:"expectedArrival" binding:"required"`
	ContainerCount   int       `json:"containerCount"`
	TotalWeight      float64   `json:"totalWeight"`
	SpecialHandling  []string  `json:"specialHandling,omitempty"`
}

// SupplierRequest represents the supplier information
type SupplierRequest struct {
	SupplierID   string `json:"supplierId" binding:"required"`
	SupplierName string `json:"supplierName" binding:"required"`
	ContactName  string `json:"contactName,omitempty"`
	ContactPhone string `json:"contactPhone,omitempty"`
	ContactEmail string `json:"contactEmail,omitempty"`
}

// ExpectedItemRequest represents an expected item in the shipment
type ExpectedItemRequest struct {
	SKU               string  `json:"sku" binding:"required"`
	ProductName       string  `json:"productName" binding:"required"`
	ExpectedQuantity  int     `json:"expectedQuantity" binding:"required,min=1"`
	UnitCost          float64 `json:"unitCost"`
	Weight            float64 `json:"weight"`
	IsHazmat          bool    `json:"isHazmat"`
	RequiresColdChain bool    `json:"requiresColdChain"`
}

// MarkArrivedRequest represents the request to mark a shipment as arrived
type MarkArrivedRequest struct {
	DockID string `json:"dockId" binding:"required"`
}

// StartReceivingRequest represents the request to start receiving
type StartReceivingRequest struct {
	WorkerID string `json:"workerId" binding:"required"`
}

// ReceiveItemRequest represents the request to receive an item
type ReceiveItemRequest struct {
	SKU       string `json:"sku" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required,min=1"`
	Condition string `json:"condition" binding:"required,oneof=good damaged"`
	ToteID    string `json:"toteId,omitempty"`
	WorkerID  string `json:"workerId" binding:"required"`
	Notes     string `json:"notes,omitempty"`
}
