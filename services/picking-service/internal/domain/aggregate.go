package domain

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Errors
var (
	ErrTaskAlreadyAssigned = errors.New("task is already assigned")
	ErrTaskNotAssigned     = errors.New("task is not assigned")
	ErrTaskCompleted       = errors.New("task is already completed")
	ErrInvalidQuantity     = errors.New("invalid quantity")
	ErrItemNotFound        = errors.New("item not found in task")
)

// PickTaskStatus represents the status of a pick task
type PickTaskStatus string

const (
	PickTaskStatusPending    PickTaskStatus = "pending"
	PickTaskStatusAssigned   PickTaskStatus = "assigned"
	PickTaskStatusInProgress PickTaskStatus = "in_progress"
	PickTaskStatusCompleted  PickTaskStatus = "completed"
	PickTaskStatusCancelled  PickTaskStatus = "cancelled"
	PickTaskStatusException  PickTaskStatus = "exception"
)

// PickMethod represents the picking method
type PickMethod string

const (
	PickMethodSingle PickMethod = "single" // One order at a time
	PickMethodBatch  PickMethod = "batch"  // Multiple orders
	PickMethodZone   PickMethod = "zone"   // Zone-based picking
	PickMethodWave   PickMethod = "wave"   // Wave picking
)

// PickTask is the aggregate root for the Picking bounded context
type PickTask struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	TaskID        string             `bson:"taskId"`
	TenantID      string             `bson:"tenantId"`
	FacilityID    string             `bson:"facilityId"`
	WarehouseID   string             `bson:"warehouseId"`
	OrderID       string             `bson:"orderId"`
	WaveID        string             `bson:"waveId"`
	RouteID       string             `bson:"routeId"`
	PickerID      string             `bson:"pickerId,omitempty"`
	Status        PickTaskStatus     `bson:"status"`
	Method        PickMethod         `bson:"method"`
	Items         []PickItem         `bson:"items"`
	ToteID        string             `bson:"toteId,omitempty"`
	Zone          string             `bson:"zone"`
	Priority      int                `bson:"priority"`
	TotalItems    int                `bson:"totalItems"`
	PickedItems   int                `bson:"pickedItems"`
	Exceptions    []PickException    `bson:"exceptions,omitempty"`
	CreatedAt     time.Time          `bson:"createdAt"`
	UpdatedAt     time.Time          `bson:"updatedAt"`
	AssignedAt    *time.Time         `bson:"assignedAt,omitempty"`
	StartedAt     *time.Time         `bson:"startedAt,omitempty"`
	CompletedAt   *time.Time         `bson:"completedAt,omitempty"`
	DomainEvents  []DomainEvent      `bson:"-"`

	// Multi-route support fields
	ParentOrderID      string `bson:"parentOrderId,omitempty"`      // Original order ID for multi-route orders
	RouteIndex         int    `bson:"routeIndex"`                   // Index in multi-route sequence (0, 1, 2...)
	TotalRoutesInOrder int    `bson:"totalRoutesInOrder"`           // Total routes for this order
	IsMultiRoute       bool   `bson:"isMultiRoute"`                 // Flag for multi-route order
	SourceToteID       string `bson:"sourceToteId,omitempty"`       // Unique tote for this route's items
}

// PickItem represents an item to be picked
type PickItem struct {
	SKU           string     `bson:"sku"`
	ProductName   string     `bson:"productName"`
	Quantity      int        `bson:"quantity"`
	PickedQty     int        `bson:"pickedQty"`
	Location      Location   `bson:"location"`
	Status        string     `bson:"status"` // pending, picked, short, damaged
	ToteID        string     `bson:"toteId,omitempty"`
	PickedAt      *time.Time `bson:"pickedAt,omitempty"`
	VerifiedAt    *time.Time `bson:"verifiedAt,omitempty"`
	Notes         string     `bson:"notes,omitempty"`
}

// Location represents a warehouse location
type Location struct {
	LocationID string `bson:"locationId"`
	Aisle      string `bson:"aisle"`
	Rack       int    `bson:"rack"`
	Level      int    `bson:"level"`
	Position   string `bson:"position"`
	Zone       string `bson:"zone"`
}

// PickException represents an exception during picking
type PickException struct {
	ExceptionID   string    `bson:"exceptionId"`
	SKU           string    `bson:"sku"`
	LocationID    string    `bson:"locationId"`
	Reason        string    `bson:"reason"` // item_not_found, damaged, quantity_mismatch
	RequestedQty  int       `bson:"requestedQty"`
	AvailableQty  int       `bson:"availableQty"`
	Resolution    string    `bson:"resolution,omitempty"`
	ResolvedAt    *time.Time `bson:"resolvedAt,omitempty"`
	CreatedAt     time.Time `bson:"createdAt"`
}

// NewPickTask creates a new PickTask aggregate
func NewPickTask(taskID, orderID, waveID, routeID string, method PickMethod, items []PickItem) (*PickTask, error) {
	if len(items) == 0 {
		return nil, errors.New("task must have at least one item")
	}

	now := time.Now()
	totalItems := 0
	for i := range items {
		totalItems += items[i].Quantity
		items[i].Status = "pending"
	}

	// Determine zone from first item
	zone := ""
	if len(items) > 0 {
		zone = items[0].Location.Zone
	}

	task := &PickTask{
		TaskID:       taskID,
		OrderID:      orderID,
		WaveID:       waveID,
		RouteID:      routeID,
		Status:       PickTaskStatusPending,
		Method:       method,
		Items:        items,
		Zone:         zone,
		Priority:     5, // Default medium
		TotalItems:   totalItems,
		PickedItems:  0,
		Exceptions:   make([]PickException, 0),
		CreatedAt:    now,
		UpdatedAt:    now,
		DomainEvents: make([]DomainEvent, 0),
	}

	task.AddDomainEvent(&PickTaskCreatedEvent{
		TaskID:    taskID,
		OrderID:   orderID,
		WaveID:    waveID,
		ItemCount: len(items),
		CreatedAt: now,
	})

	return task, nil
}

// NewMultiRoutePickTask creates a new PickTask with multi-route tracking fields
func NewMultiRoutePickTask(taskID, orderID, waveID, routeID string, method PickMethod, items []PickItem, routeIndex, totalRoutes int, sourceToteID string) (*PickTask, error) {
	task, err := NewPickTask(taskID, orderID, waveID, routeID, method, items)
	if err != nil {
		return nil, err
	}

	task.ParentOrderID = orderID
	task.RouteIndex = routeIndex
	task.TotalRoutesInOrder = totalRoutes
	task.IsMultiRoute = totalRoutes > 1
	task.SourceToteID = sourceToteID

	return task, nil
}

// Assign assigns the task to a picker
func (t *PickTask) Assign(pickerID, toteID string) error {
	if t.Status != PickTaskStatusPending {
		return ErrTaskAlreadyAssigned
	}

	now := time.Now()
	t.PickerID = pickerID
	t.ToteID = toteID
	t.Status = PickTaskStatusAssigned
	t.AssignedAt = &now
	t.UpdatedAt = now

	t.AddDomainEvent(&PickTaskAssignedEvent{
		TaskID:     t.TaskID,
		PickerID:   pickerID,
		ToteID:     toteID,
		AssignedAt: now,
	})

	return nil
}

// Start marks the task as in progress
func (t *PickTask) Start() error {
	if t.Status != PickTaskStatusAssigned {
		return ErrTaskNotAssigned
	}

	now := time.Now()
	t.Status = PickTaskStatusInProgress
	t.StartedAt = &now
	t.UpdatedAt = now

	return nil
}

// PickItem confirms an item has been picked
func (t *PickTask) ConfirmPick(sku string, locationID string, pickedQty int, toteID string) error {
	if t.Status != PickTaskStatusInProgress {
		return errors.New("task is not in progress")
	}

	for i := range t.Items {
		// Match by SKU, and either locationID matches or task item has no location set
		itemLocationID := t.Items[i].Location.LocationID
		locationMatches := itemLocationID == locationID || itemLocationID == ""
		if t.Items[i].SKU == sku && locationMatches {
			now := time.Now()
			t.Items[i].PickedQty = pickedQty
			t.Items[i].ToteID = toteID
			t.Items[i].PickedAt = &now

			if pickedQty >= t.Items[i].Quantity {
				t.Items[i].Status = "picked"
			} else if pickedQty > 0 {
				t.Items[i].Status = "short"
			}

			t.PickedItems += pickedQty
			t.UpdatedAt = now

			t.AddDomainEvent(&ItemPickedEvent{
				TaskID:     t.TaskID,
				SKU:        sku,
				Quantity:   pickedQty,
				LocationID: locationID,
				ToteID:     toteID,
				PickedAt:   now,
			})

			// Check if all items are picked
			allPicked := true
			for _, item := range t.Items {
				if item.Status == "pending" {
					allPicked = false
					break
				}
			}

			if allPicked {
				return t.Complete()
			}

			return nil
		}
	}

	return ErrItemNotFound
}

// ReportException reports a picking exception
func (t *PickTask) ReportException(sku, locationID, reason string, requestedQty, availableQty int) error {
	if t.Status != PickTaskStatusInProgress {
		return errors.New("task is not in progress")
	}

	now := time.Now()
	exception := PickException{
		ExceptionID:  generateExceptionID(),
		SKU:          sku,
		LocationID:   locationID,
		Reason:       reason,
		RequestedQty: requestedQty,
		AvailableQty: availableQty,
		CreatedAt:    now,
	}

	t.Exceptions = append(t.Exceptions, exception)
	t.UpdatedAt = now

	// Update item status
	for i := range t.Items {
		if t.Items[i].SKU == sku && t.Items[i].Location.LocationID == locationID {
			t.Items[i].Status = "exception"
			break
		}
	}

	t.AddDomainEvent(&PickExceptionEvent{
		TaskID:       t.TaskID,
		ExceptionID:  exception.ExceptionID,
		SKU:          sku,
		LocationID:   locationID,
		Reason:       reason,
		RequestedQty: requestedQty,
		AvailableQty: availableQty,
		ReportedAt:   now,
	})

	return nil
}

// ResolveException resolves a picking exception
func (t *PickTask) ResolveException(exceptionID, resolution string) error {
	now := time.Now()
	for i := range t.Exceptions {
		if t.Exceptions[i].ExceptionID == exceptionID {
			t.Exceptions[i].Resolution = resolution
			t.Exceptions[i].ResolvedAt = &now
			t.UpdatedAt = now
			return nil
		}
	}
	return errors.New("exception not found")
}

// Complete marks the task as completed
func (t *PickTask) Complete() error {
	if t.Status == PickTaskStatusCompleted {
		return ErrTaskCompleted
	}

	now := time.Now()
	t.Status = PickTaskStatusCompleted
	t.CompletedAt = &now
	t.UpdatedAt = now

	// Collect picked items for event
	pickedItems := make([]PickedItemInfo, 0)
	for _, item := range t.Items {
		if item.PickedQty > 0 {
			pickedItems = append(pickedItems, PickedItemInfo{
				SKU:        item.SKU,
				Quantity:   item.PickedQty,
				LocationID: item.Location.LocationID,
				ToteID:     item.ToteID,
			})
		}
	}

	t.AddDomainEvent(&PickTaskCompletedEvent{
		TaskID:      t.TaskID,
		OrderID:     t.OrderID,
		PickerID:    t.PickerID,
		TotalItems:  t.TotalItems,
		PickedItems: t.PickedItems,
		PickedList:  pickedItems,
		CompletedAt: now,
	})

	return nil
}

// Cancel cancels the task
func (t *PickTask) Cancel(reason string) error {
	if t.Status == PickTaskStatusCompleted {
		return ErrTaskCompleted
	}

	t.Status = PickTaskStatusCancelled
	t.UpdatedAt = time.Now()

	return nil
}

// GetProgress returns the completion percentage
func (t *PickTask) GetProgress() float64 {
	if t.TotalItems == 0 {
		return 0
	}
	return float64(t.PickedItems) / float64(t.TotalItems) * 100
}

// GetPendingItems returns items that haven't been picked
func (t *PickTask) GetPendingItems() []PickItem {
	pending := make([]PickItem, 0)
	for _, item := range t.Items {
		if item.Status == "pending" {
			pending = append(pending, item)
		}
	}
	return pending
}

// HasExceptions returns true if there are unresolved exceptions
func (t *PickTask) HasExceptions() bool {
	for _, ex := range t.Exceptions {
		if ex.ResolvedAt == nil {
			return true
		}
	}
	return false
}

// AddDomainEvent adds a domain event
func (t *PickTask) AddDomainEvent(event DomainEvent) {
	t.DomainEvents = append(t.DomainEvents, event)
}

// ClearDomainEvents clears all domain events
func (t *PickTask) ClearDomainEvents() {
	t.DomainEvents = make([]DomainEvent, 0)
}

// GetDomainEvents returns all domain events
func (t *PickTask) GetDomainEvents() []DomainEvent {
	return t.DomainEvents
}

// PickedItemInfo contains info about a picked item
type PickedItemInfo struct {
	SKU        string `json:"sku"`
	Quantity   int    `json:"quantity"`
	LocationID string `json:"locationId"`
	ToteID     string `json:"toteId"`
}

func generateExceptionID() string {
	return "EX-" + time.Now().Format("20060102150405")
}
