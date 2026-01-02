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
}

func (e *ItemReceivedEvent) EventType() string     { return "receiving.item.received" }
func (e *ItemReceivedEvent) OccurredAt() time.Time { return e.ReceivedAt }

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
