package domain

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Tote errors
var (
	ErrToteNotAvailable    = errors.New("tote is not available")
	ErrToteAtCapacity      = errors.New("tote is at maximum capacity")
	ErrToteWeightExceeded  = errors.New("tote weight limit exceeded")
	ErrItemNotInTote       = errors.New("item not found in tote")
	ErrInvalidToteType     = errors.New("invalid tote type")
	ErrInvalidToteStatus   = errors.New("invalid tote status")
	ErrToteAlreadyAssigned = errors.New("tote already assigned to an order")
)

// ToteType represents the type of tote
type ToteType string

const (
	ToteTypeStandard  ToteType = "standard"
	ToteTypeColdChain ToteType = "cold_chain"
	ToteTypeHazmat    ToteType = "hazmat"
	ToteTypeOversized ToteType = "oversized"
)

// IsValid checks if the tote type is valid
func (t ToteType) IsValid() bool {
	switch t {
	case ToteTypeStandard, ToteTypeColdChain, ToteTypeHazmat, ToteTypeOversized:
		return true
	default:
		return false
	}
}

// ToteStatus represents the operational status of a tote
type ToteStatus string

const (
	ToteStatusAvailable     ToteStatus = "available"
	ToteStatusInUse         ToteStatus = "in_use"
	ToteStatusNeedsCleaning ToteStatus = "needs_cleaning"
	ToteStatusMaintenance   ToteStatus = "maintenance"
	ToteStatusRetired       ToteStatus = "retired"
)

// IsValid checks if the status is valid
func (s ToteStatus) IsValid() bool {
	switch s {
	case ToteStatusAvailable, ToteStatusInUse, ToteStatusNeedsCleaning, ToteStatusMaintenance, ToteStatusRetired:
		return true
	default:
		return false
	}
}

// ToteItem represents an item currently in a tote
type ToteItem struct {
	SKU        string    `bson:"sku" json:"sku"`
	Quantity   int       `bson:"quantity" json:"quantity"`
	OrderID    string    `bson:"orderId,omitempty" json:"orderId,omitempty"`
	LocationID string    `bson:"locationId,omitempty" json:"locationId,omitempty"`
	Weight     float64   `bson:"weight" json:"weight"`
	AddedAt    time.Time `bson:"addedAt" json:"addedAt"`
}

// ToteCapacity defines capacity limits for different tote types
type ToteCapacity struct {
	MaxItems  int     `bson:"maxItems" json:"maxItems"`
	MaxWeight float64 `bson:"maxWeight" json:"maxWeight"` // in kg
}

// DefaultCapacities for tote types
var DefaultCapacities = map[ToteType]ToteCapacity{
	ToteTypeStandard:  {MaxItems: 20, MaxWeight: 15.0},
	ToteTypeColdChain: {MaxItems: 10, MaxWeight: 10.0},
	ToteTypeHazmat:    {MaxItems: 5, MaxWeight: 8.0},
	ToteTypeOversized: {MaxItems: 5, MaxWeight: 25.0},
}

// Tote represents a transport container used throughout the fulfillment process
// This is an entity with identity and mutable state
type Tote struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ToteID          string             `bson:"toteId" json:"toteId"`
	Barcode         string             `bson:"barcode" json:"barcode"`
	ToteType        ToteType           `bson:"toteType" json:"toteType"`
	CurrentLocation string             `bson:"currentLocation" json:"currentLocation"`
	CurrentZone     string             `bson:"currentZone" json:"currentZone"`
	Contents        []ToteItem         `bson:"contents" json:"contents"`
	Status          ToteStatus         `bson:"status" json:"status"`
	Capacity        ToteCapacity       `bson:"capacity" json:"capacity"`
	CurrentWeight   float64            `bson:"currentWeight" json:"currentWeight"`
	AssignedOrderID string             `bson:"assignedOrderId,omitempty" json:"assignedOrderId,omitempty"`
	AssignedWaveID  string             `bson:"assignedWaveId,omitempty" json:"assignedWaveId,omitempty"`
	AssignedWorkerID string            `bson:"assignedWorkerId,omitempty" json:"assignedWorkerId,omitempty"`
	LastUsedAt      *time.Time         `bson:"lastUsedAt,omitempty" json:"lastUsedAt,omitempty"`
	CreatedAt       time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time          `bson:"updatedAt" json:"updatedAt"`
	DomainEvents    []DomainEvent      `bson:"-" json:"-"`
}

// DomainEvent interface for tote domain events
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// NewTote creates a new Tote entity
func NewTote(toteID, barcode string, toteType ToteType) (*Tote, error) {
	if !toteType.IsValid() {
		return nil, ErrInvalidToteType
	}

	capacity, exists := DefaultCapacities[toteType]
	if !exists {
		capacity = DefaultCapacities[ToteTypeStandard]
	}

	now := time.Now().UTC()
	tote := &Tote{
		ID:            primitive.NewObjectID(),
		ToteID:        toteID,
		Barcode:       barcode,
		ToteType:      toteType,
		Contents:      make([]ToteItem, 0),
		Status:        ToteStatusAvailable,
		Capacity:      capacity,
		CurrentWeight: 0,
		CreatedAt:     now,
		UpdatedAt:     now,
		DomainEvents:  make([]DomainEvent, 0),
	}

	tote.addDomainEvent(&ToteCreatedEvent{
		ToteID:    toteID,
		ToteType:  string(toteType),
		CreatedAt: now,
	})

	return tote, nil
}

// IsAvailable checks if the tote is available for use
func (t *Tote) IsAvailable() bool {
	return t.Status == ToteStatusAvailable
}

// IsEmpty checks if the tote has no contents
func (t *Tote) IsEmpty() bool {
	return len(t.Contents) == 0
}

// ItemCount returns the total number of items in the tote
func (t *Tote) ItemCount() int {
	total := 0
	for _, item := range t.Contents {
		total += item.Quantity
	}
	return total
}

// RemainingCapacity returns the number of additional items the tote can hold
func (t *Tote) RemainingCapacity() int {
	return t.Capacity.MaxItems - t.ItemCount()
}

// RemainingWeight returns the additional weight the tote can hold
func (t *Tote) RemainingWeight() float64 {
	return t.Capacity.MaxWeight - t.CurrentWeight
}

// CanAddItem checks if an item can be added to the tote
func (t *Tote) CanAddItem(quantity int, weight float64) bool {
	if t.Status != ToteStatusAvailable && t.Status != ToteStatusInUse {
		return false
	}
	if t.ItemCount()+quantity > t.Capacity.MaxItems {
		return false
	}
	if t.CurrentWeight+weight > t.Capacity.MaxWeight {
		return false
	}
	return true
}

// AddItem adds an item to the tote
func (t *Tote) AddItem(sku string, quantity int, weight float64, orderID, locationID string) error {
	if t.Status != ToteStatusAvailable && t.Status != ToteStatusInUse {
		return ErrToteNotAvailable
	}

	if t.ItemCount()+quantity > t.Capacity.MaxItems {
		return ErrToteAtCapacity
	}

	if t.CurrentWeight+weight > t.Capacity.MaxWeight {
		return ErrToteWeightExceeded
	}

	// Check if item already exists in tote
	for i := range t.Contents {
		if t.Contents[i].SKU == sku && t.Contents[i].OrderID == orderID {
			t.Contents[i].Quantity += quantity
			t.Contents[i].Weight += weight
			t.CurrentWeight += weight
			t.UpdatedAt = time.Now().UTC()
			return nil
		}
	}

	// Add new item
	t.Contents = append(t.Contents, ToteItem{
		SKU:        sku,
		Quantity:   quantity,
		OrderID:    orderID,
		LocationID: locationID,
		Weight:     weight,
		AddedAt:    time.Now().UTC(),
	})

	t.CurrentWeight += weight
	t.Status = ToteStatusInUse
	t.UpdatedAt = time.Now().UTC()

	t.addDomainEvent(&ItemAddedToToteEvent{
		ToteID:   t.ToteID,
		SKU:      sku,
		Quantity: quantity,
		OrderID:  orderID,
		AddedAt:  t.UpdatedAt,
	})

	return nil
}

// RemoveItem removes an item from the tote
func (t *Tote) RemoveItem(sku string, orderID string) error {
	for i := range t.Contents {
		if t.Contents[i].SKU == sku && t.Contents[i].OrderID == orderID {
			t.CurrentWeight -= t.Contents[i].Weight
			t.Contents = append(t.Contents[:i], t.Contents[i+1:]...)
			t.UpdatedAt = time.Now().UTC()

			if len(t.Contents) == 0 {
				t.Status = ToteStatusAvailable
			}

			t.addDomainEvent(&ItemRemovedFromToteEvent{
				ToteID:    t.ToteID,
				SKU:       sku,
				OrderID:   orderID,
				RemovedAt: t.UpdatedAt,
			})

			return nil
		}
	}
	return ErrItemNotInTote
}

// Clear removes all items from the tote
func (t *Tote) Clear() {
	t.Contents = make([]ToteItem, 0)
	t.CurrentWeight = 0
	t.Status = ToteStatusAvailable
	t.AssignedOrderID = ""
	t.AssignedWaveID = ""
	now := time.Now().UTC()
	t.UpdatedAt = now

	t.addDomainEvent(&ToteClearedEvent{
		ToteID:    t.ToteID,
		ClearedAt: now,
	})
}

// AssignToOrder assigns the tote to a specific order
func (t *Tote) AssignToOrder(orderID string) error {
	if t.AssignedOrderID != "" && t.AssignedOrderID != orderID {
		return ErrToteAlreadyAssigned
	}

	t.AssignedOrderID = orderID
	t.Status = ToteStatusInUse
	t.UpdatedAt = time.Now().UTC()

	return nil
}

// AssignToWave assigns the tote to a wave
func (t *Tote) AssignToWave(waveID string) {
	t.AssignedWaveID = waveID
	t.Status = ToteStatusInUse
	t.UpdatedAt = time.Now().UTC()
}

// AssignToWorker assigns the tote to a worker
func (t *Tote) AssignToWorker(workerID string) {
	t.AssignedWorkerID = workerID
	t.UpdatedAt = time.Now().UTC()
}

// UnassignWorker removes the worker assignment
func (t *Tote) UnassignWorker() {
	t.AssignedWorkerID = ""
	t.UpdatedAt = time.Now().UTC()
}

// MoveTo updates the tote's current location
func (t *Tote) MoveTo(locationID, zone string) {
	t.CurrentLocation = locationID
	t.CurrentZone = zone
	t.UpdatedAt = time.Now().UTC()

	t.addDomainEvent(&ToteMovedEvent{
		ToteID:     t.ToteID,
		LocationID: locationID,
		Zone:       zone,
		MovedAt:    t.UpdatedAt,
	})
}

// SetStatus sets the tote status
func (t *Tote) SetStatus(status ToteStatus) error {
	if !status.IsValid() {
		return ErrInvalidToteStatus
	}

	oldStatus := t.Status
	t.Status = status
	now := time.Now().UTC()
	t.UpdatedAt = now

	if status == ToteStatusAvailable || status == ToteStatusNeedsCleaning {
		t.LastUsedAt = &now
	}

	t.addDomainEvent(&ToteStatusChangedEvent{
		ToteID:    t.ToteID,
		OldStatus: string(oldStatus),
		NewStatus: string(status),
		ChangedAt: now,
	})

	return nil
}

// MarkNeedsCleaning marks the tote as needing cleaning
func (t *Tote) MarkNeedsCleaning() error {
	return t.SetStatus(ToteStatusNeedsCleaning)
}

// MarkAvailable marks the tote as available
func (t *Tote) MarkAvailable() error {
	return t.SetStatus(ToteStatusAvailable)
}

// Retire marks the tote as retired
func (t *Tote) Retire() error {
	return t.SetStatus(ToteStatusRetired)
}

// GetContentsForOrder returns items in the tote for a specific order
func (t *Tote) GetContentsForOrder(orderID string) []ToteItem {
	items := make([]ToteItem, 0)
	for _, item := range t.Contents {
		if item.OrderID == orderID {
			items = append(items, item)
		}
	}
	return items
}

// addDomainEvent adds a domain event
func (t *Tote) addDomainEvent(event DomainEvent) {
	t.DomainEvents = append(t.DomainEvents, event)
}

// GetDomainEvents returns all domain events
func (t *Tote) GetDomainEvents() []DomainEvent {
	return t.DomainEvents
}

// ClearDomainEvents clears all domain events
func (t *Tote) ClearDomainEvents() {
	t.DomainEvents = make([]DomainEvent, 0)
}

// Tote Domain Events

// ToteCreatedEvent is emitted when a tote is created
type ToteCreatedEvent struct {
	ToteID    string    `json:"toteId"`
	ToteType  string    `json:"toteType"`
	CreatedAt time.Time `json:"createdAt"`
}

func (e *ToteCreatedEvent) EventType() string     { return "tote.created" }
func (e *ToteCreatedEvent) OccurredAt() time.Time { return e.CreatedAt }

// ItemAddedToToteEvent is emitted when an item is added to a tote
type ItemAddedToToteEvent struct {
	ToteID   string    `json:"toteId"`
	SKU      string    `json:"sku"`
	Quantity int       `json:"quantity"`
	OrderID  string    `json:"orderId,omitempty"`
	AddedAt  time.Time `json:"addedAt"`
}

func (e *ItemAddedToToteEvent) EventType() string     { return "tote.item.added" }
func (e *ItemAddedToToteEvent) OccurredAt() time.Time { return e.AddedAt }

// ItemRemovedFromToteEvent is emitted when an item is removed from a tote
type ItemRemovedFromToteEvent struct {
	ToteID    string    `json:"toteId"`
	SKU       string    `json:"sku"`
	OrderID   string    `json:"orderId,omitempty"`
	RemovedAt time.Time `json:"removedAt"`
}

func (e *ItemRemovedFromToteEvent) EventType() string     { return "tote.item.removed" }
func (e *ItemRemovedFromToteEvent) OccurredAt() time.Time { return e.RemovedAt }

// ToteClearedEvent is emitted when a tote is cleared
type ToteClearedEvent struct {
	ToteID    string    `json:"toteId"`
	ClearedAt time.Time `json:"clearedAt"`
}

func (e *ToteClearedEvent) EventType() string     { return "tote.cleared" }
func (e *ToteClearedEvent) OccurredAt() time.Time { return e.ClearedAt }

// ToteMovedEvent is emitted when a tote is moved to a new location
type ToteMovedEvent struct {
	ToteID     string    `json:"toteId"`
	LocationID string    `json:"locationId"`
	Zone       string    `json:"zone"`
	MovedAt    time.Time `json:"movedAt"`
}

func (e *ToteMovedEvent) EventType() string     { return "tote.moved" }
func (e *ToteMovedEvent) OccurredAt() time.Time { return e.MovedAt }

// ToteStatusChangedEvent is emitted when tote status changes
type ToteStatusChangedEvent struct {
	ToteID    string    `json:"toteId"`
	OldStatus string    `json:"oldStatus"`
	NewStatus string    `json:"newStatus"`
	ChangedAt time.Time `json:"changedAt"`
}

func (e *ToteStatusChangedEvent) EventType() string     { return "tote.status.changed" }
func (e *ToteStatusChangedEvent) OccurredAt() time.Time { return e.ChangedAt }
