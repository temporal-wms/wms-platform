package domain

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// InboundShipment errors
var (
	ErrShipmentNotFound         = errors.New("shipment not found")
	ErrInvalidShipmentStatus    = errors.New("invalid shipment status")
	ErrInvalidStatusTransition  = errors.New("invalid status transition")
	ErrNoExpectedItems          = errors.New("shipment must have at least one expected item")
	ErrItemNotFound             = errors.New("item not found in shipment")
	ErrQuantityExceedsExpected  = errors.New("received quantity exceeds expected quantity")
	ErrShipmentAlreadyCompleted = errors.New("shipment already completed")
	ErrInvalidASN               = errors.New("invalid advance shipping notice")
)

// ShipmentStatus represents the status of an inbound shipment
type ShipmentStatus string

const (
	ShipmentStatusExpected   ShipmentStatus = "expected"
	ShipmentStatusArrived    ShipmentStatus = "arrived"
	ShipmentStatusReceiving  ShipmentStatus = "receiving"
	ShipmentStatusInspection ShipmentStatus = "inspection"
	ShipmentStatusCompleted  ShipmentStatus = "completed"
	ShipmentStatusCancelled  ShipmentStatus = "cancelled"
)

// IsValid checks if the status is valid
func (s ShipmentStatus) IsValid() bool {
	switch s {
	case ShipmentStatusExpected, ShipmentStatusArrived, ShipmentStatusReceiving,
		ShipmentStatusInspection, ShipmentStatusCompleted, ShipmentStatusCancelled:
		return true
	default:
		return false
	}
}

// CanTransitionTo checks if the status can transition to another status
func (s ShipmentStatus) CanTransitionTo(target ShipmentStatus) bool {
	validTransitions := map[ShipmentStatus][]ShipmentStatus{
		ShipmentStatusExpected:   {ShipmentStatusArrived, ShipmentStatusCancelled},
		ShipmentStatusArrived:    {ShipmentStatusReceiving, ShipmentStatusCancelled},
		ShipmentStatusReceiving:  {ShipmentStatusInspection, ShipmentStatusCompleted, ShipmentStatusCancelled},
		ShipmentStatusInspection: {ShipmentStatusCompleted, ShipmentStatusCancelled},
		ShipmentStatusCompleted:  {},
		ShipmentStatusCancelled:  {},
	}

	allowedTargets, exists := validTransitions[s]
	if !exists {
		return false
	}

	for _, allowed := range allowedTargets {
		if target == allowed {
			return true
		}
	}
	return false
}

// ExpectedItem represents an item expected in the shipment
type ExpectedItem struct {
	SKU              string  `bson:"sku" json:"sku"`
	ProductName      string  `bson:"productName" json:"productName"`
	ExpectedQuantity int     `bson:"expectedQuantity" json:"expectedQuantity"`
	ReceivedQuantity int     `bson:"receivedQuantity" json:"receivedQuantity"`
	DamagedQuantity  int     `bson:"damagedQuantity" json:"damagedQuantity"`
	UnitCost         float64 `bson:"unitCost" json:"unitCost"`
	Weight           float64 `bson:"weight" json:"weight"`
	IsHazmat         bool    `bson:"isHazmat" json:"isHazmat"`
	RequiresColdChain bool   `bson:"requiresColdChain" json:"requiresColdChain"`
}

// RemainingQuantity returns the quantity still to be received
func (e *ExpectedItem) RemainingQuantity() int {
	return e.ExpectedQuantity - e.ReceivedQuantity - e.DamagedQuantity
}

// IsFullyReceived returns true if the item is fully received
func (e *ExpectedItem) IsFullyReceived() bool {
	return e.ReceivedQuantity+e.DamagedQuantity >= e.ExpectedQuantity
}

// ReceiptRecord represents a single receipt action
type ReceiptRecord struct {
	ReceiptID    string    `bson:"receiptId" json:"receiptId"`
	SKU          string    `bson:"sku" json:"sku"`
	Quantity     int       `bson:"quantity" json:"quantity"`
	ToteID       string    `bson:"toteId,omitempty" json:"toteId,omitempty"`
	Condition    string    `bson:"condition" json:"condition"` // good, damaged
	ReceivedBy   string    `bson:"receivedBy" json:"receivedBy"`
	ReceivedAt   time.Time `bson:"receivedAt" json:"receivedAt"`
	Notes        string    `bson:"notes,omitempty" json:"notes,omitempty"`
}

// Discrepancy represents a difference between expected and actual
type Discrepancy struct {
	SKU               string    `bson:"sku" json:"sku"`
	ExpectedQuantity  int       `bson:"expectedQuantity" json:"expectedQuantity"`
	ReceivedQuantity  int       `bson:"receivedQuantity" json:"receivedQuantity"`
	DamagedQuantity   int       `bson:"damagedQuantity" json:"damagedQuantity"`
	DiscrepancyType   string    `bson:"discrepancyType" json:"discrepancyType"` // shortage, overage, damage
	RecordedAt        time.Time `bson:"recordedAt" json:"recordedAt"`
	Notes             string    `bson:"notes,omitempty" json:"notes,omitempty"`
}

// AdvanceShippingNotice (ASN) contains pre-arrival shipment information
type AdvanceShippingNotice struct {
	ASNID            string    `bson:"asnId" json:"asnId"`
	CarrierName      string    `bson:"carrierName" json:"carrierName"`
	TrackingNumber   string    `bson:"trackingNumber,omitempty" json:"trackingNumber,omitempty"`
	ExpectedArrival  time.Time `bson:"expectedArrival" json:"expectedArrival"`
	ContainerCount   int       `bson:"containerCount" json:"containerCount"`
	TotalWeight      float64   `bson:"totalWeight" json:"totalWeight"`
	SpecialHandling  []string  `bson:"specialHandling,omitempty" json:"specialHandling,omitempty"`
}

// Supplier represents the supplier sending the shipment
type Supplier struct {
	SupplierID   string `bson:"supplierId" json:"supplierId"`
	SupplierName string `bson:"supplierName" json:"supplierName"`
	ContactName  string `bson:"contactName,omitempty" json:"contactName,omitempty"`
	ContactPhone string `bson:"contactPhone,omitempty" json:"contactPhone,omitempty"`
	ContactEmail string `bson:"contactEmail,omitempty" json:"contactEmail,omitempty"`
}

// InboundShipment is the aggregate root for the Receiving bounded context
type InboundShipment struct {
	ID               primitive.ObjectID    `bson:"_id,omitempty" json:"id"`
	ShipmentID       string                `bson:"shipmentId" json:"shipmentId"`
	ASN              AdvanceShippingNotice `bson:"asn" json:"asn"`
	PurchaseOrderID  string                `bson:"purchaseOrderId,omitempty" json:"purchaseOrderId,omitempty"`
	Supplier         Supplier              `bson:"supplier" json:"supplier"`
	ExpectedItems    []ExpectedItem        `bson:"expectedItems" json:"expectedItems"`
	ReceiptRecords   []ReceiptRecord       `bson:"receiptRecords" json:"receiptRecords"`
	Discrepancies    []Discrepancy         `bson:"discrepancies" json:"discrepancies"`
	Status           ShipmentStatus        `bson:"status" json:"status"`
	ReceivingDockID  string                `bson:"receivingDockId,omitempty" json:"receivingDockId,omitempty"`
	AssignedWorkerID string                `bson:"assignedWorkerId,omitempty" json:"assignedWorkerId,omitempty"`
	ArrivedAt        *time.Time            `bson:"arrivedAt,omitempty" json:"arrivedAt,omitempty"`
	CompletedAt      *time.Time            `bson:"completedAt,omitempty" json:"completedAt,omitempty"`
	CreatedAt        time.Time             `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time             `bson:"updatedAt" json:"updatedAt"`
	DomainEvents     []DomainEvent         `bson:"-" json:"-"`
}

// NewInboundShipment creates a new InboundShipment aggregate
func NewInboundShipment(
	shipmentID string,
	asn AdvanceShippingNotice,
	supplier Supplier,
	expectedItems []ExpectedItem,
	purchaseOrderID string,
) (*InboundShipment, error) {
	if len(expectedItems) == 0 {
		return nil, ErrNoExpectedItems
	}

	if asn.ASNID == "" {
		return nil, ErrInvalidASN
	}

	now := time.Now().UTC()
	shipment := &InboundShipment{
		ID:              primitive.NewObjectID(),
		ShipmentID:      shipmentID,
		ASN:             asn,
		PurchaseOrderID: purchaseOrderID,
		Supplier:        supplier,
		ExpectedItems:   expectedItems,
		ReceiptRecords:  make([]ReceiptRecord, 0),
		Discrepancies:   make([]Discrepancy, 0),
		Status:          ShipmentStatusExpected,
		CreatedAt:       now,
		UpdatedAt:       now,
		DomainEvents:    make([]DomainEvent, 0),
	}

	shipment.addDomainEvent(&ShipmentExpectedEvent{
		ShipmentID:      shipmentID,
		ASNID:           asn.ASNID,
		SupplierID:      supplier.SupplierID,
		ExpectedArrival: asn.ExpectedArrival,
		ItemCount:       len(expectedItems),
		OccurredAt_:     now,
	})

	return shipment, nil
}

// MarkArrived marks the shipment as arrived at the receiving dock
func (s *InboundShipment) MarkArrived(dockID string) error {
	if !s.Status.CanTransitionTo(ShipmentStatusArrived) {
		return ErrInvalidStatusTransition
	}

	now := time.Now().UTC()
	s.Status = ShipmentStatusArrived
	s.ReceivingDockID = dockID
	s.ArrivedAt = &now
	s.UpdatedAt = now

	s.addDomainEvent(&ShipmentArrivedEvent{
		ShipmentID:      s.ShipmentID,
		DockID:          dockID,
		ArrivedAt:       now,
		ExpectedArrival: s.ASN.ExpectedArrival,
		IsOnTime:        now.Before(s.ASN.ExpectedArrival) || now.Equal(s.ASN.ExpectedArrival),
	})

	return nil
}

// StartReceiving starts the receiving process
func (s *InboundShipment) StartReceiving(workerID string) error {
	if !s.Status.CanTransitionTo(ShipmentStatusReceiving) {
		return ErrInvalidStatusTransition
	}

	s.Status = ShipmentStatusReceiving
	s.AssignedWorkerID = workerID
	s.UpdatedAt = time.Now().UTC()

	return nil
}

// ReceiveItem records the receipt of items
func (s *InboundShipment) ReceiveItem(sku string, quantity int, condition string, toteID, workerID, notes string) error {
	if s.Status != ShipmentStatusReceiving {
		return ErrInvalidStatusTransition
	}

	// Find the expected item
	var expectedItem *ExpectedItem
	for i := range s.ExpectedItems {
		if s.ExpectedItems[i].SKU == sku {
			expectedItem = &s.ExpectedItems[i]
			break
		}
	}

	if expectedItem == nil {
		return ErrItemNotFound
	}

	now := time.Now().UTC()

	// Update quantities based on condition
	if condition == "damaged" {
		expectedItem.DamagedQuantity += quantity
	} else {
		expectedItem.ReceivedQuantity += quantity
	}

	// Create receipt record
	receiptID := generateReceiptID()
	s.ReceiptRecords = append(s.ReceiptRecords, ReceiptRecord{
		ReceiptID:  receiptID,
		SKU:        sku,
		Quantity:   quantity,
		ToteID:     toteID,
		Condition:  condition,
		ReceivedBy: workerID,
		ReceivedAt: now,
		Notes:      notes,
	})

	s.UpdatedAt = now

	s.addDomainEvent(&ItemReceivedEvent{
		ShipmentID: s.ShipmentID,
		ReceiptID:  receiptID,
		SKU:        sku,
		Quantity:   quantity,
		Condition:  condition,
		ToteID:     toteID,
		ReceivedBy: workerID,
		ReceivedAt: now,
	})

	return nil
}

// StartInspection starts quality inspection
func (s *InboundShipment) StartInspection() error {
	if !s.Status.CanTransitionTo(ShipmentStatusInspection) {
		return ErrInvalidStatusTransition
	}

	s.Status = ShipmentStatusInspection
	s.UpdatedAt = time.Now().UTC()

	return nil
}

// Complete completes the receiving process
func (s *InboundShipment) Complete() error {
	if s.Status != ShipmentStatusReceiving && s.Status != ShipmentStatusInspection {
		return ErrInvalidStatusTransition
	}

	// Calculate and record discrepancies
	s.calculateDiscrepancies()

	now := time.Now().UTC()
	s.Status = ShipmentStatusCompleted
	s.CompletedAt = &now
	s.UpdatedAt = now

	s.addDomainEvent(&ReceivingCompletedEvent{
		ShipmentID:         s.ShipmentID,
		TotalItemsExpected: s.TotalExpectedQuantity(),
		TotalItemsReceived: s.TotalReceivedQuantity(),
		TotalDamaged:       s.TotalDamagedQuantity(),
		DiscrepancyCount:   len(s.Discrepancies),
		CompletedAt:        now,
	})

	// Emit discrepancy events if any
	for _, disc := range s.Discrepancies {
		s.addDomainEvent(&ReceivingDiscrepancyEvent{
			ShipmentID:       s.ShipmentID,
			SKU:              disc.SKU,
			ExpectedQuantity: disc.ExpectedQuantity,
			ReceivedQuantity: disc.ReceivedQuantity,
			DamagedQuantity:  disc.DamagedQuantity,
			DiscrepancyType:  disc.DiscrepancyType,
			OccurredAt_:      now,
		})
	}

	return nil
}

// Cancel cancels the shipment
func (s *InboundShipment) Cancel(reason string) error {
	if s.Status == ShipmentStatusCompleted {
		return ErrShipmentAlreadyCompleted
	}

	if !s.Status.CanTransitionTo(ShipmentStatusCancelled) {
		return ErrInvalidStatusTransition
	}

	s.Status = ShipmentStatusCancelled
	s.UpdatedAt = time.Now().UTC()

	return nil
}

// calculateDiscrepancies calculates discrepancies between expected and received
func (s *InboundShipment) calculateDiscrepancies() {
	now := time.Now().UTC()
	s.Discrepancies = make([]Discrepancy, 0)

	for _, item := range s.ExpectedItems {
		received := item.ReceivedQuantity
		damaged := item.DamagedQuantity
		expected := item.ExpectedQuantity

		if received+damaged != expected || damaged > 0 {
			discType := "exact"
			if received+damaged < expected {
				discType = "shortage"
			} else if received+damaged > expected {
				discType = "overage"
			} else if damaged > 0 {
				discType = "damage"
			}

			if discType != "exact" {
				s.Discrepancies = append(s.Discrepancies, Discrepancy{
					SKU:              item.SKU,
					ExpectedQuantity: expected,
					ReceivedQuantity: received,
					DamagedQuantity:  damaged,
					DiscrepancyType:  discType,
					RecordedAt:       now,
				})
			}
		}
	}
}

// TotalExpectedQuantity returns total expected items
func (s *InboundShipment) TotalExpectedQuantity() int {
	total := 0
	for _, item := range s.ExpectedItems {
		total += item.ExpectedQuantity
	}
	return total
}

// TotalReceivedQuantity returns total received items (good condition)
func (s *InboundShipment) TotalReceivedQuantity() int {
	total := 0
	for _, item := range s.ExpectedItems {
		total += item.ReceivedQuantity
	}
	return total
}

// TotalDamagedQuantity returns total damaged items
func (s *InboundShipment) TotalDamagedQuantity() int {
	total := 0
	for _, item := range s.ExpectedItems {
		total += item.DamagedQuantity
	}
	return total
}

// IsFullyReceived checks if all items are fully received
func (s *InboundShipment) IsFullyReceived() bool {
	for _, item := range s.ExpectedItems {
		if !item.IsFullyReceived() {
			return false
		}
	}
	return true
}

// HasDiscrepancies checks if there are any discrepancies
func (s *InboundShipment) HasDiscrepancies() bool {
	return len(s.Discrepancies) > 0
}

// GetExpectedItem returns an expected item by SKU
func (s *InboundShipment) GetExpectedItem(sku string) *ExpectedItem {
	for i := range s.ExpectedItems {
		if s.ExpectedItems[i].SKU == sku {
			return &s.ExpectedItems[i]
		}
	}
	return nil
}

// addDomainEvent adds a domain event
func (s *InboundShipment) addDomainEvent(event DomainEvent) {
	s.DomainEvents = append(s.DomainEvents, event)
}

// GetDomainEvents returns all domain events
func (s *InboundShipment) GetDomainEvents() []DomainEvent {
	return s.DomainEvents
}

// ClearDomainEvents clears all domain events
func (s *InboundShipment) ClearDomainEvents() {
	s.DomainEvents = make([]DomainEvent, 0)
}

func generateReceiptID() string {
	return "RCV-" + time.Now().Format("20060102150405.000")
}
