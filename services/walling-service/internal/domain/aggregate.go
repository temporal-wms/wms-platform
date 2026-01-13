package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Errors
var (
	ErrWallingTaskCompleted = errors.New("walling task is already completed")
	ErrWallingTaskCancelled = errors.New("walling task is cancelled")
	ErrNoItemsToSort        = errors.New("no items to sort")
	ErrItemNotFound         = errors.New("item not found in task")
	ErrAllItemsSorted       = errors.New("all items already sorted")
)

// WallingTaskStatus represents the status of a walling task
type WallingTaskStatus string

const (
	WallingTaskStatusPending    WallingTaskStatus = "pending"
	WallingTaskStatusAssigned   WallingTaskStatus = "assigned"
	WallingTaskStatusInProgress WallingTaskStatus = "in_progress"
	WallingTaskStatusCompleted  WallingTaskStatus = "completed"
	WallingTaskStatusCancelled  WallingTaskStatus = "cancelled"
)

// WallingTask is the aggregate root for the Walling bounded context
type WallingTask struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	TaskID         string             `bson:"taskId"`
	TenantID    string `bson:"tenantId" json:"tenantId"`
	FacilityID  string `bson:"facilityId" json:"facilityId"`
	WarehouseID string `bson:"warehouseId" json:"warehouseId"`
	OrderID        string             `bson:"orderId"`
	WaveID         string             `bson:"waveId"`
	RouteID        string             `bson:"routeId,omitempty"` // Link to WES route
	WallinerID     string             `bson:"wallinerId,omitempty"`
	Status         WallingTaskStatus  `bson:"status"`
	SourceTotes    []SourceTote       `bson:"sourceTotes"`
	DestinationBin string             `bson:"destinationBin"`
	PutWallID      string             `bson:"putWallId"`
	ItemsToSort    []ItemToSort       `bson:"itemsToSort"`
	SortedItems    []SortedItem       `bson:"sortedItems"`
	Station        string             `bson:"station"`
	Priority       int                `bson:"priority"`
	CreatedAt      time.Time          `bson:"createdAt"`
	UpdatedAt      time.Time          `bson:"updatedAt"`
	AssignedAt     *time.Time         `bson:"assignedAt,omitempty"`
	StartedAt      *time.Time         `bson:"startedAt,omitempty"`
	CompletedAt    *time.Time         `bson:"completedAt,omitempty"`
	DomainEvents   []DomainEvent      `bson:"-"`
}

// SourceTote represents a tote from the picking stage
type SourceTote struct {
	ToteID     string `bson:"toteId" json:"toteId"`
	PickTaskID string `bson:"pickTaskId" json:"pickTaskId"`
	ItemCount  int    `bson:"itemCount" json:"itemCount"`
}

// ItemToSort represents an item that needs to be sorted to a bin
type ItemToSort struct {
	SKU          string `bson:"sku" json:"sku"`
	Quantity     int    `bson:"quantity" json:"quantity"`
	FromToteID   string `bson:"fromToteId" json:"fromToteId"`
	SortedQty    int    `bson:"sortedQty" json:"sortedQty"` // How many have been sorted
}

// SortedItem represents an item that has been sorted to a bin
type SortedItem struct {
	SKU        string    `bson:"sku" json:"sku"`
	Quantity   int       `bson:"quantity" json:"quantity"`
	FromToteID string    `bson:"fromToteId" json:"fromToteId"`
	ToBinID    string    `bson:"toBinId" json:"toBinId"`
	SortedAt   time.Time `bson:"sortedAt" json:"sortedAt"`
	Verified   bool      `bson:"verified" json:"verified"`
}

// NewWallingTask creates a new WallingTask
func NewWallingTask(orderID, waveID, putWallID, destinationBin string, sourceTotes []SourceTote, items []ItemToSort) (*WallingTask, error) {
	if len(items) == 0 {
		return nil, ErrNoItemsToSort
	}

	now := time.Now()
	taskID := "WT-" + uuid.New().String()[:8]

	task := &WallingTask{
		TaskID:         taskID,
		OrderID:        orderID,
		WaveID:         waveID,
		Status:         WallingTaskStatusPending,
		SourceTotes:    sourceTotes,
		DestinationBin: destinationBin,
		PutWallID:      putWallID,
		ItemsToSort:    items,
		SortedItems:    make([]SortedItem, 0),
		Priority:       5,
		CreatedAt:      now,
		UpdatedAt:      now,
		DomainEvents:   make([]DomainEvent, 0),
	}

	task.AddDomainEvent(&WallingTaskCreatedEvent{
		TaskID:         taskID,
		OrderID:        orderID,
		WaveID:         waveID,
		PutWallID:      putWallID,
		DestinationBin: destinationBin,
		ItemCount:      len(items),
		CreatedAt:      now,
	})

	return task, nil
}

// SetRouteID sets the WES route ID
func (t *WallingTask) SetRouteID(routeID string) {
	t.RouteID = routeID
	t.UpdatedAt = time.Now()
}

// Assign assigns the task to a walliner
func (t *WallingTask) Assign(wallinerID, station string) error {
	if t.Status == WallingTaskStatusCompleted {
		return ErrWallingTaskCompleted
	}
	if t.Status == WallingTaskStatusCancelled {
		return ErrWallingTaskCancelled
	}

	now := time.Now()
	t.WallinerID = wallinerID
	t.Station = station
	t.Status = WallingTaskStatusAssigned
	t.AssignedAt = &now
	t.UpdatedAt = now

	t.AddDomainEvent(&WallingTaskAssignedEvent{
		TaskID:     t.TaskID,
		OrderID:    t.OrderID,
		WallinerID: wallinerID,
		Station:    station,
		AssignedAt: now,
	})

	return nil
}

// Start marks the task as in progress
func (t *WallingTask) Start() error {
	if t.Status != WallingTaskStatusAssigned {
		return errors.New("task must be assigned before starting")
	}

	now := time.Now()
	t.Status = WallingTaskStatusInProgress
	t.StartedAt = &now
	t.UpdatedAt = now

	return nil
}

// SortItem sorts an item to the destination bin
func (t *WallingTask) SortItem(sku string, quantity int, fromToteID string) error {
	if t.Status == WallingTaskStatusCompleted {
		return ErrWallingTaskCompleted
	}
	if t.Status == WallingTaskStatusCancelled {
		return ErrWallingTaskCancelled
	}

	// Start the task if not started
	if t.Status == WallingTaskStatusAssigned {
		if err := t.Start(); err != nil {
			return err
		}
	}

	// Find the item in items to sort
	var itemFound *ItemToSort
	for i := range t.ItemsToSort {
		if t.ItemsToSort[i].SKU == sku && t.ItemsToSort[i].FromToteID == fromToteID {
			itemFound = &t.ItemsToSort[i]
			break
		}
	}

	if itemFound == nil {
		return ErrItemNotFound
	}

	// Calculate remaining to sort
	remaining := itemFound.Quantity - itemFound.SortedQty
	if remaining <= 0 {
		return errors.New("item already fully sorted")
	}

	// Limit quantity to remaining
	if quantity > remaining {
		quantity = remaining
	}

	// Record sorted item
	now := time.Now()
	sortedItem := SortedItem{
		SKU:        sku,
		Quantity:   quantity,
		FromToteID: fromToteID,
		ToBinID:    t.DestinationBin,
		SortedAt:   now,
		Verified:   true,
	}
	t.SortedItems = append(t.SortedItems, sortedItem)
	itemFound.SortedQty += quantity
	t.UpdatedAt = now

	t.AddDomainEvent(&ItemSortedEvent{
		TaskID:   t.TaskID,
		OrderID:  t.OrderID,
		SKU:      sku,
		Quantity: quantity,
		ToteID:   fromToteID,
		BinID:    t.DestinationBin,
		SortedAt: now,
	})

	// Check if all items are sorted
	if t.AllItemsSorted() {
		return t.Complete()
	}

	return nil
}

// AllItemsSorted checks if all items have been sorted
func (t *WallingTask) AllItemsSorted() bool {
	for _, item := range t.ItemsToSort {
		if item.SortedQty < item.Quantity {
			return false
		}
	}
	return true
}

// Complete marks the task as completed
func (t *WallingTask) Complete() error {
	if t.Status == WallingTaskStatusCompleted {
		return ErrWallingTaskCompleted
	}

	now := time.Now()
	t.Status = WallingTaskStatusCompleted
	t.CompletedAt = &now
	t.UpdatedAt = now

	// Count total sorted
	totalSorted := 0
	for _, item := range t.SortedItems {
		totalSorted += item.Quantity
	}

	t.AddDomainEvent(&WallingTaskCompletedEvent{
		TaskID:         t.TaskID,
		OrderID:        t.OrderID,
		WallinerID:     t.WallinerID,
		DestinationBin: t.DestinationBin,
		ItemsSorted:    totalSorted,
		CompletedAt:    now,
	})

	return nil
}

// Cancel cancels the task
func (t *WallingTask) Cancel(reason string) error {
	if t.Status == WallingTaskStatusCompleted {
		return ErrWallingTaskCompleted
	}

	t.Status = WallingTaskStatusCancelled
	t.UpdatedAt = time.Now()

	return nil
}

// GetProgress returns the sorting progress
func (t *WallingTask) GetProgress() (sorted int, total int) {
	for _, item := range t.ItemsToSort {
		total += item.Quantity
		sorted += item.SortedQty
	}
	return sorted, total
}

// AddDomainEvent adds a domain event
func (t *WallingTask) AddDomainEvent(event DomainEvent) {
	t.DomainEvents = append(t.DomainEvents, event)
}

// ClearDomainEvents clears all domain events
func (t *WallingTask) ClearDomainEvents() {
	t.DomainEvents = make([]DomainEvent, 0)
}

// GetDomainEvents returns all domain events
func (t *WallingTask) GetDomainEvents() []DomainEvent {
	return t.DomainEvents
}
