package domain

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Errors for Order aggregate
var (
	ErrNoItems           = errors.New("order must have at least one item")
	ErrInvalidPriority   = errors.New("invalid order priority")
	ErrInvalidStatus     = errors.New("invalid status transition")
	ErrOrderCancelled    = errors.New("order has been cancelled")
	ErrOrderAlreadyWaved = errors.New("order already assigned to a wave")
)

// Priority represents order priority levels
type Priority string

const (
	PrioritySameDay  Priority = "same_day"
	PriorityNextDay  Priority = "next_day"
	PriorityStandard Priority = "standard"
)

// IsValid checks if the priority is valid
func (p Priority) IsValid() bool {
	switch p {
	case PrioritySameDay, PriorityNextDay, PriorityStandard:
		return true
	default:
		return false
	}
}

// Status represents order status
type Status string

const (
	StatusReceived     Status = "received"
	StatusValidated    Status = "validated"
	StatusWaveAssigned Status = "wave_assigned"
	StatusPicking      Status = "picking"
	StatusConsolidated Status = "consolidated"
	StatusPacked       Status = "packed"
	StatusShipped      Status = "shipped"
	StatusDelivered    Status = "delivered"
	StatusCancelled    Status = "cancelled"
	StatusPendingRetry Status = "pending_retry"
	StatusDeadLetter   Status = "dead_letter"
)

// HighValueThreshold is the order value threshold for high-value handling (in currency units)
const HighValueThreshold = 500.0

// Order is the aggregate root for the Order Management bounded context
type Order struct {
	ID                  primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OrderID             string             `bson:"orderId" json:"orderId"`
	CustomerID          string             `bson:"customerId" json:"customerId"`
	Items               []OrderItem        `bson:"items" json:"items"`
	ShippingAddress     Address            `bson:"shippingAddress" json:"shippingAddress"`
	Priority            Priority           `bson:"priority" json:"priority"`
	Status              Status             `bson:"status" json:"status"`
	PromisedDeliveryAt  time.Time          `bson:"promisedDeliveryAt" json:"promisedDeliveryAt"`
	WaveID              string             `bson:"waveId,omitempty" json:"waveId,omitempty"`
	TrackingNumber      string             `bson:"trackingNumber,omitempty" json:"trackingNumber,omitempty"`
	CreatedAt           time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt           time.Time          `bson:"updatedAt" json:"updatedAt"`
	GiftWrap            bool               `bson:"giftWrap" json:"giftWrap"`
	ProcessRequirements OrderRequirements  `bson:"processRequirements" json:"processRequirements"`

	// Domain events - transient, not persisted
	domainEvents []DomainEvent `bson:"-" json:"-"`
}

// OrderItem represents an item in an order
type OrderItem struct {
	SKU               string  `bson:"sku" json:"sku"`
	Name              string  `bson:"name" json:"name"`
	Quantity          int     `bson:"quantity" json:"quantity"`
	Weight            float64 `bson:"weight" json:"weight"`
	Dimensions        Dims    `bson:"dimensions" json:"dimensions"`
	UnitPrice         float64 `bson:"unitPrice" json:"unitPrice"`
	PickedQty         int     `bson:"pickedQty" json:"pickedQty"`
	LocationID        string  `bson:"locationId,omitempty" json:"locationId,omitempty"`
	IsFragile         bool    `bson:"isFragile" json:"isFragile"`
	IsHazmat          bool    `bson:"isHazmat" json:"isHazmat"`
	RequiresColdChain bool    `bson:"requiresColdChain" json:"requiresColdChain"`
}

// IsOversized returns true if the item exceeds standard shipping dimensions
// Oversized threshold: any dimension > 100cm or total weight > 30kg
func (i *OrderItem) IsOversized() bool {
	const maxDimension = 100.0 // cm
	const maxWeight = 30.0    // kg

	return i.Dimensions.Length > maxDimension ||
		i.Dimensions.Width > maxDimension ||
		i.Dimensions.Height > maxDimension ||
		i.Weight > maxWeight
}

// Dims represents item dimensions
type Dims struct {
	Length float64 `bson:"length" json:"length"`
	Width  float64 `bson:"width" json:"width"`
	Height float64 `bson:"height" json:"height"`
}

// Address represents a shipping address
type Address struct {
	Street     string `bson:"street" json:"street"`
	City       string `bson:"city" json:"city"`
	State      string `bson:"state" json:"state"`
	ZipCode    string `bson:"zipCode" json:"zipCode"`
	Country    string `bson:"country" json:"country"`
	Phone      string `bson:"phone,omitempty" json:"phone,omitempty"`
	RecipientName string `bson:"recipientName" json:"recipientName"`
}

// NewOrder creates a new Order aggregate
func NewOrder(
	orderID string,
	customerID string,
	items []OrderItem,
	shippingAddress Address,
	priority Priority,
	promisedDeliveryAt time.Time,
) (*Order, error) {
	if len(items) == 0 {
		return nil, ErrNoItems
	}

	if !priority.IsValid() {
		return nil, ErrInvalidPriority
	}

	now := time.Now().UTC()
	order := &Order{
		ID:                 primitive.NewObjectID(),
		OrderID:            orderID,
		CustomerID:         customerID,
		Items:              items,
		ShippingAddress:    shippingAddress,
		Priority:           priority,
		Status:             StatusReceived,
		PromisedDeliveryAt: promisedDeliveryAt,
		CreatedAt:          now,
		UpdatedAt:          now,
		domainEvents:       make([]DomainEvent, 0),
	}

	// Calculate process path requirements based on order characteristics
	order.CalculateRequirements()

	order.addDomainEvent(NewOrderReceivedEvent(order))

	return order, nil
}

// Validate validates the order and transitions it to validated status
// This method is idempotent - if already validated, it returns success
func (o *Order) Validate() error {
	if o.Status == StatusCancelled {
		return ErrOrderCancelled
	}

	// Idempotency: if already validated or further along, return success
	if o.Status == StatusValidated || o.Status == StatusWaveAssigned ||
		o.Status == StatusPicking || o.Status == StatusConsolidated ||
		o.Status == StatusPacked || o.Status == StatusShipped {
		return nil
	}

	if o.Status != StatusReceived {
		return ErrInvalidStatus
	}

	if len(o.Items) == 0 {
		return ErrNoItems
	}

	o.Status = StatusValidated
	o.UpdatedAt = time.Now().UTC()
	o.addDomainEvent(NewOrderValidatedEvent(o))

	return nil
}

// AssignToWave assigns the order to a wave
func (o *Order) AssignToWave(waveID string) error {
	if o.Status == StatusCancelled {
		return ErrOrderCancelled
	}

	if o.Status != StatusValidated {
		return ErrInvalidStatus
	}

	if o.WaveID != "" {
		return ErrOrderAlreadyWaved
	}

	o.WaveID = waveID
	o.Status = StatusWaveAssigned
	o.UpdatedAt = time.Now().UTC()
	o.addDomainEvent(NewOrderAssignedToWaveEvent(o, waveID))

	return nil
}

// StartPicking transitions the order to picking status
func (o *Order) StartPicking() error {
	if o.Status != StatusWaveAssigned {
		return ErrInvalidStatus
	}

	o.Status = StatusPicking
	o.UpdatedAt = time.Now().UTC()

	return nil
}

// MarkItemPicked marks an item as picked
func (o *Order) MarkItemPicked(sku string, quantity int) error {
	for i := range o.Items {
		if o.Items[i].SKU == sku {
			o.Items[i].PickedQty += quantity
			o.UpdatedAt = time.Now().UTC()
			return nil
		}
	}
	return errors.New("item not found in order")
}

// MarkConsolidated marks the order as consolidated
func (o *Order) MarkConsolidated() error {
	if o.Status != StatusPicking {
		return ErrInvalidStatus
	}

	o.Status = StatusConsolidated
	o.UpdatedAt = time.Now().UTC()

	return nil
}

// MarkPacked marks the order as packed
func (o *Order) MarkPacked() error {
	if o.Status != StatusConsolidated && o.Status != StatusPicking {
		return ErrInvalidStatus
	}

	o.Status = StatusPacked
	o.UpdatedAt = time.Now().UTC()

	return nil
}

// MarkShipped marks the order as shipped with tracking number
func (o *Order) MarkShipped(trackingNumber string) error {
	if o.Status != StatusPacked {
		return ErrInvalidStatus
	}

	o.Status = StatusShipped
	o.TrackingNumber = trackingNumber
	o.UpdatedAt = time.Now().UTC()
	o.addDomainEvent(NewOrderShippedEvent(o))

	return nil
}

// Cancel cancels the order
func (o *Order) Cancel(reason string) error {
	if o.Status == StatusShipped || o.Status == StatusDelivered {
		return errors.New("cannot cancel shipped or delivered order")
	}

	if o.Status == StatusCancelled {
		return nil // Already cancelled
	}

	o.Status = StatusCancelled
	o.UpdatedAt = time.Now().UTC()
	o.addDomainEvent(NewOrderCancelledEvent(o, reason))

	return nil
}

// TotalItems returns the total number of items in the order
func (o *Order) TotalItems() int {
	total := 0
	for _, item := range o.Items {
		total += item.Quantity
	}
	return total
}

// TotalWeight returns the total weight of the order
func (o *Order) TotalWeight() float64 {
	total := 0.0
	for _, item := range o.Items {
		total += item.Weight * float64(item.Quantity)
	}
	return total
}

// IsMultiItem returns true if the order has multiple items
func (o *Order) IsMultiItem() bool {
	return len(o.Items) > 1 || (len(o.Items) == 1 && o.Items[0].Quantity > 1)
}

// IsSingleItem returns true if the order has a single item
func (o *Order) IsSingleItem() bool {
	return len(o.Items) == 1 && o.Items[0].Quantity == 1
}

// TotalValue returns the total value of the order
func (o *Order) TotalValue() float64 {
	total := 0.0
	for _, item := range o.Items {
		total += item.UnitPrice * float64(item.Quantity)
	}
	return total
}

// CalculateRequirements analyzes the order and populates ProcessRequirements
// This should be called when the order is created or modified
func (o *Order) CalculateRequirements() {
	requirements := make([]ProcessRequirement, 0)

	// Single vs Multi item requirement
	if o.IsSingleItem() {
		requirements = append(requirements, RequirementSingleItem)
	} else {
		requirements = append(requirements, RequirementMultiItem)
	}

	// Gift wrap requirement (order-level)
	if o.GiftWrap {
		requirements = append(requirements, RequirementGiftWrap)
	}

	// Item-level requirements (deduplicated)
	hasFragile := false
	hasHazmat := false
	hasColdChain := false
	hasOversized := false

	for _, item := range o.Items {
		if item.IsFragile && !hasFragile {
			requirements = append(requirements, RequirementFragile)
			hasFragile = true
		}
		if item.IsHazmat && !hasHazmat {
			requirements = append(requirements, RequirementHazmat)
			hasHazmat = true
		}
		if item.RequiresColdChain && !hasColdChain {
			requirements = append(requirements, RequirementColdChain)
			hasColdChain = true
		}
		if item.IsOversized() && !hasOversized {
			requirements = append(requirements, RequirementOversized)
			hasOversized = true
		}
	}

	// High value requirement (order-level threshold)
	if o.TotalValue() >= HighValueThreshold {
		requirements = append(requirements, RequirementHighValue)
	}

	o.ProcessRequirements.Requirements = requirements
}

// SetGiftWrapDetails sets the gift wrap details and recalculates requirements
func (o *Order) SetGiftWrapDetails(details *GiftWrapDetails) {
	o.GiftWrap = true
	o.ProcessRequirements.GiftWrapDetails = details
	o.CalculateRequirements()
	o.UpdatedAt = time.Now().UTC()
}

// SetHazmatDetails sets the hazmat details for the order
func (o *Order) SetHazmatDetails(details *HazmatDetails) {
	o.ProcessRequirements.HazmatDetails = details
	o.UpdatedAt = time.Now().UTC()
}

// SetColdChainDetails sets the cold chain details for the order
func (o *Order) SetColdChainDetails(details *ColdChainDetails) {
	o.ProcessRequirements.ColdChainDetails = details
	o.UpdatedAt = time.Now().UTC()
}

// addDomainEvent adds a domain event to the order
func (o *Order) addDomainEvent(event DomainEvent) {
	o.domainEvents = append(o.domainEvents, event)
}

// DomainEvents returns all pending domain events
func (o *Order) DomainEvents() []DomainEvent {
	return o.domainEvents
}

// ClearDomainEvents clears all pending domain events
func (o *Order) ClearDomainEvents() {
	o.domainEvents = make([]DomainEvent, 0)
}

// ResetForRetry prepares the order for reprocessing after a transient failure
// This clears the wave assignment and sets the order to pending_retry status
func (o *Order) ResetForRetry() error {
	// Cannot reset orders in terminal states
	if o.Status == StatusCancelled || o.Status == StatusShipped || o.Status == StatusDelivered {
		return errors.New("cannot reset order in terminal status")
	}

	// Cannot reset orders already in dead letter
	if o.Status == StatusDeadLetter {
		return errors.New("cannot reset order in dead letter queue")
	}

	o.WaveID = ""
	o.Status = StatusPendingRetry
	o.UpdatedAt = time.Now().UTC()

	return nil
}

// MoveToDeadLetter marks the order as moved to dead letter queue after exhausting retries
func (o *Order) MoveToDeadLetter(reason string) error {
	// Cannot move shipped/delivered orders to DLQ
	if o.Status == StatusShipped || o.Status == StatusDelivered {
		return errors.New("cannot move shipped or delivered order to dead letter queue")
	}

	// Already in DLQ
	if o.Status == StatusDeadLetter {
		return nil
	}

	o.Status = StatusDeadLetter
	o.UpdatedAt = time.Now().UTC()

	return nil
}
