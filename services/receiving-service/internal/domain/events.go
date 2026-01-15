package domain

import "time"

// DomainEvent represents a domain event interface
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// ShipmentExpectedEvent is emitted when a new shipment is expected
type ShipmentExpectedEvent struct {
	ShipmentID      string    `json:"shipmentId"`
	ASNID           string    `json:"asnId"`
	SupplierID      string    `json:"supplierId"`
	ExpectedArrival time.Time `json:"expectedArrival"`
	ItemCount       int       `json:"itemCount"`
	OccurredAt_     time.Time `json:"occurredAt"`
}

func (e *ShipmentExpectedEvent) EventType() string     { return "receiving.shipment.expected" }
func (e *ShipmentExpectedEvent) OccurredAt() time.Time { return e.OccurredAt_ }

// ShipmentArrivedEvent is emitted when a shipment arrives at the dock
type ShipmentArrivedEvent struct {
	ShipmentID      string    `json:"shipmentId"`
	DockID          string    `json:"dockId"`
	ArrivedAt       time.Time `json:"arrivedAt"`
	ExpectedArrival time.Time `json:"expectedArrival"`
	IsOnTime        bool      `json:"isOnTime"`
}

func (e *ShipmentArrivedEvent) EventType() string     { return "receiving.shipment.arrived" }
func (e *ShipmentArrivedEvent) OccurredAt() time.Time { return e.ArrivedAt }

// ItemReceivedEvent is emitted when an item is received
type ItemReceivedEvent struct {
	ShipmentID string    `json:"shipmentId"`
	ReceiptID  string    `json:"receiptId"`
	SKU        string    `json:"sku"`
	Quantity   int       `json:"quantity"`
	Condition  string    `json:"condition"`
	ToteID     string    `json:"toteId,omitempty"`
	ReceivedBy string    `json:"receivedBy"`
	ReceivedAt time.Time `json:"receivedAt"`
	UnitIDs    []string  `json:"unitIds,omitempty"` // Generated unit UUIDs for unit-level tracking
}

func (e *ItemReceivedEvent) EventType() string     { return "receiving.item.received" }
func (e *ItemReceivedEvent) OccurredAt() time.Time { return e.ReceivedAt }

// CartonReceivedEvent is emitted when a carton is received via batch ASN
type CartonReceivedEvent struct {
	ShipmentID string    `json:"shipmentId"`
	CartonID   string    `json:"cartonId"`
	ToteID     string    `json:"toteId,omitempty"`
	ReceivedBy string    `json:"receivedBy"`
	ReceivedAt time.Time `json:"receivedAt"`
	ItemCount  int       `json:"itemCount"` // Number of distinct SKUs in carton
}

func (e *CartonReceivedEvent) EventType() string     { return "receiving.carton.received" }
func (e *CartonReceivedEvent) OccurredAt() time.Time { return e.ReceivedAt }

// ItemPrepRequiredEvent is emitted when an item needs prep/repackaging
type ItemPrepRequiredEvent struct {
	ShipmentID string    `json:"shipmentId"`
	ReceiptID  string    `json:"receiptId"`
	SKU        string    `json:"sku"`
	Quantity   int       `json:"quantity"`
	ToteID     string    `json:"toteId,omitempty"`
	Reason     string    `json:"reason"`
	ReceivedBy string    `json:"receivedBy"`
	ReceivedAt time.Time `json:"receivedAt"`
}

func (e *ItemPrepRequiredEvent) EventType() string     { return "receiving.item.prep_required" }
func (e *ItemPrepRequiredEvent) OccurredAt() time.Time { return e.ReceivedAt }

// ItemPreppedEvent is emitted when item prep is completed
type ItemPreppedEvent struct {
	ShipmentID string    `json:"shipmentId"`
	ReceiptID  string    `json:"receiptId"`
	SKU        string    `json:"sku"`
	Quantity   int       `json:"quantity"`
	ToteID     string    `json:"toteId,omitempty"`
	ReceivedBy string    `json:"receivedBy"`
	ReceivedAt time.Time `json:"receivedAt"`
}

func (e *ItemPreppedEvent) EventType() string     { return "receiving.item.prepped" }
func (e *ItemPreppedEvent) OccurredAt() time.Time { return e.ReceivedAt }

// ReceivingCompletedEvent is emitted when receiving is completed
type ReceivingCompletedEvent struct {
	ShipmentID         string    `json:"shipmentId"`
	TotalItemsExpected int       `json:"totalItemsExpected"`
	TotalItemsReceived int       `json:"totalItemsReceived"`
	TotalDamaged       int       `json:"totalDamaged"`
	DiscrepancyCount   int       `json:"discrepancyCount"`
	CompletedAt        time.Time `json:"completedAt"`
}

func (e *ReceivingCompletedEvent) EventType() string     { return "receiving.completed" }
func (e *ReceivingCompletedEvent) OccurredAt() time.Time { return e.CompletedAt }

// ReceivingDiscrepancyEvent is emitted when a discrepancy is found
type ReceivingDiscrepancyEvent struct {
	ShipmentID       string    `json:"shipmentId"`
	SKU              string    `json:"sku"`
	ExpectedQuantity int       `json:"expectedQuantity"`
	ReceivedQuantity int       `json:"receivedQuantity"`
	DamagedQuantity  int       `json:"damagedQuantity"`
	DiscrepancyType  string    `json:"discrepancyType"` // shortage, overage, damage
	OccurredAt_      time.Time `json:"occurredAt"`
}

func (e *ReceivingDiscrepancyEvent) EventType() string     { return "receiving.discrepancy" }
func (e *ReceivingDiscrepancyEvent) OccurredAt() time.Time { return e.OccurredAt_ }

// PutawayTaskCreatedEvent is emitted when putaway tasks are created
type PutawayTaskCreatedEvent struct {
	ShipmentID string    `json:"shipmentId"`
	TaskID     string    `json:"taskId"`
	SKU        string    `json:"sku"`
	Quantity   int       `json:"quantity"`
	ToteID     string    `json:"toteId"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (e *PutawayTaskCreatedEvent) EventType() string     { return "receiving.putaway.created" }
func (e *PutawayTaskCreatedEvent) OccurredAt() time.Time { return e.CreatedAt }

// ProblemCreatedEvent is emitted when a problem ticket is created
type ProblemCreatedEvent struct {
	TicketID    string    `json:"ticketId"`
	ShipmentID  string    `json:"shipmentId"`
	SKU         string    `json:"sku,omitempty"`
	ProblemType string    `json:"problemType"`
	Description string    `json:"description"`
	Quantity    int       `json:"quantity"`
	Priority    string    `json:"priority"`
	CreatedBy   string    `json:"createdBy"`
	OccurredAt_ time.Time `json:"occurredAt"`
}

func (e *ProblemCreatedEvent) EventType() string     { return "receiving.problem.created" }
func (e *ProblemCreatedEvent) OccurredAt() time.Time { return e.OccurredAt_ }

// ProblemResolvedEvent is emitted when a problem is resolved
type ProblemResolvedEvent struct {
	TicketID        string    `json:"ticketId"`
	ShipmentID      string    `json:"shipmentId"`
	SKU             string    `json:"sku,omitempty"`
	ProblemType     string    `json:"problemType"`
	Resolution      string    `json:"resolution"`
	ResolutionNotes string    `json:"resolutionNotes,omitempty"`
	ResolvedBy      string    `json:"resolvedBy"`
	OccurredAt_     time.Time `json:"occurredAt"`
}

func (e *ProblemResolvedEvent) EventType() string     { return "receiving.problem.resolved" }
func (e *ProblemResolvedEvent) OccurredAt() time.Time { return e.OccurredAt_ }

// ItemDisposedEvent is emitted when an item is disposed
type ItemDisposedEvent struct {
	TicketID    string    `json:"ticketId"`
	ShipmentID  string    `json:"shipmentId"`
	SKU         string    `json:"sku"`
	Quantity    int       `json:"quantity"`
	Reason      string    `json:"reason"`
	DisposedBy  string    `json:"disposedBy"`
	OccurredAt_ time.Time `json:"occurredAt"`
}

func (e *ItemDisposedEvent) EventType() string     { return "receiving.item.disposed" }
func (e *ItemDisposedEvent) OccurredAt() time.Time { return e.OccurredAt_ }

// ReturnCreatedEvent is emitted when a return shipment is created
type ReturnCreatedEvent struct {
	TicketID    string    `json:"ticketId"`
	ShipmentID  string    `json:"shipmentId"`
	SKU         string    `json:"sku"`
	Quantity    int       `json:"quantity"`
	Reason      string    `json:"reason"`
	CreatedBy   string    `json:"createdBy"`
	OccurredAt_ time.Time `json:"occurredAt"`
}

func (e *ReturnCreatedEvent) EventType() string     { return "receiving.return.created" }
func (e *ReturnCreatedEvent) OccurredAt() time.Time { return e.OccurredAt_ }
