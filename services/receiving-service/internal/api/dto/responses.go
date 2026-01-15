package dto

import "time"

// ShipmentResponse represents the response for a shipment
type ShipmentResponse struct {
	ID               string                `json:"id"`
	ShipmentID       string                `json:"shipmentId"`
	ASN              ASNResponse           `json:"asn"`
	PurchaseOrderID  string                `json:"purchaseOrderId,omitempty"`
	Supplier         SupplierResponse      `json:"supplier"`
	ExpectedItems    []ExpectedItemResponse `json:"expectedItems"`
	ReceiptRecords   []ReceiptRecordResponse `json:"receiptRecords"`
	Discrepancies    []DiscrepancyResponse `json:"discrepancies"`
	Status           string                `json:"status"`
	ReceivingDockID  string                `json:"receivingDockId,omitempty"`
	AssignedWorkerID string                `json:"assignedWorkerId,omitempty"`
	ArrivedAt        *time.Time            `json:"arrivedAt,omitempty"`
	CompletedAt      *time.Time            `json:"completedAt,omitempty"`
	CreatedAt        time.Time             `json:"createdAt"`
	UpdatedAt        time.Time             `json:"updatedAt"`
	// Summary fields
	TotalExpected    int                   `json:"totalExpected"`
	TotalReceived    int                   `json:"totalReceived"`
	TotalDamaged     int                   `json:"totalDamaged"`
	IsFullyReceived  bool                  `json:"isFullyReceived"`
}

// ASNResponse represents the advance shipping notice in response
type ASNResponse struct {
	ASNID            string    `json:"asnId"`
	CarrierName      string    `json:"carrierName"`
	TrackingNumber   string    `json:"trackingNumber,omitempty"`
	ExpectedArrival  time.Time `json:"expectedArrival"`
	ContainerCount   int       `json:"containerCount"`
	TotalWeight      float64   `json:"totalWeight"`
	SpecialHandling  []string  `json:"specialHandling,omitempty"`
}

// SupplierResponse represents the supplier in response
type SupplierResponse struct {
	SupplierID   string `json:"supplierId"`
	SupplierName string `json:"supplierName"`
	ContactName  string `json:"contactName,omitempty"`
	ContactPhone string `json:"contactPhone,omitempty"`
	ContactEmail string `json:"contactEmail,omitempty"`
}

// ExpectedItemResponse represents an expected item in response
type ExpectedItemResponse struct {
	SKU               string  `json:"sku"`
	ProductName       string  `json:"productName"`
	ExpectedQuantity  int     `json:"expectedQuantity"`
	ReceivedQuantity  int     `json:"receivedQuantity"`
	DamagedQuantity   int     `json:"damagedQuantity"`
	PrepQuantity      int     `json:"prepQuantity"`      // Items needing prep
	RemainingQuantity int     `json:"remainingQuantity"`
	UnitCost          float64 `json:"unitCost"`
	Weight            float64 `json:"weight"`
	IsHazmat          bool    `json:"isHazmat"`
	RequiresColdChain bool    `json:"requiresColdChain"`
	IsFullyReceived   bool    `json:"isFullyReceived"`
}

// ReceiptRecordResponse represents a receipt record in response
type ReceiptRecordResponse struct {
	ReceiptID  string    `json:"receiptId"`
	SKU        string    `json:"sku"`
	Quantity   int       `json:"quantity"`
	ToteID     string    `json:"toteId,omitempty"`
	Condition  string    `json:"condition"`
	ReceivedBy string    `json:"receivedBy"`
	ReceivedAt time.Time `json:"receivedAt"`
	Notes      string    `json:"notes,omitempty"`
}

// DiscrepancyResponse represents a discrepancy in response
type DiscrepancyResponse struct {
	SKU              string    `json:"sku"`
	ExpectedQuantity int       `json:"expectedQuantity"`
	ReceivedQuantity int       `json:"receivedQuantity"`
	DamagedQuantity  int       `json:"damagedQuantity"`
	DiscrepancyType  string    `json:"discrepancyType"`
	RecordedAt       time.Time `json:"recordedAt"`
	Notes            string    `json:"notes,omitempty"`
}

// ShipmentListResponse represents a list of shipments
type ShipmentListResponse struct {
	Shipments []ShipmentSummary `json:"shipments"`
	Total     int               `json:"total"`
}

// ShipmentSummary represents a summary of a shipment for list views
type ShipmentSummary struct {
	ID              string    `json:"id"`
	ShipmentID      string    `json:"shipmentId"`
	ASNID           string    `json:"asnId"`
	SupplierName    string    `json:"supplierName"`
	Status          string    `json:"status"`
	ExpectedArrival time.Time `json:"expectedArrival"`
	TotalExpected   int       `json:"totalExpected"`
	TotalReceived   int       `json:"totalReceived"`
	CreatedAt       time.Time `json:"createdAt"`
}

// ProblemTicketResponse represents the response for a problem ticket
type ProblemTicketResponse struct {
	ID              string     `json:"id"`
	TicketID        string     `json:"ticketId"`
	ShipmentID      string     `json:"shipmentId"`
	SKU             string     `json:"sku,omitempty"`
	ProductName     string     `json:"productName,omitempty"`
	ProblemType     string     `json:"problemType"`
	Description     string     `json:"description"`
	Quantity        int        `json:"quantity"`
	AffectedUnitIDs []string   `json:"affectedUnitIds,omitempty"`
	Resolution      string     `json:"resolution"`
	ResolutionNotes string     `json:"resolutionNotes,omitempty"`
	CreatedBy       string     `json:"createdBy"`
	AssignedTo      string     `json:"assignedTo,omitempty"`
	ResolvedBy      string     `json:"resolvedBy,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	ResolvedAt      *time.Time `json:"resolvedAt,omitempty"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	Priority        string     `json:"priority"`
	ImageURLs       []string   `json:"imageUrls,omitempty"`
	IsPending       bool       `json:"isPending"`
	IsResolved      bool       `json:"isResolved"`
}

// ProblemTicketListResponse represents a list of problem tickets
type ProblemTicketListResponse struct {
	Tickets []ProblemTicketResponse `json:"tickets"`
	Total   int                     `json:"total"`
}
