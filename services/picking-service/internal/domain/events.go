package domain

import "time"

// DomainEvent is the interface for all domain events
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// PickTaskCreatedEvent is published when a pick task is created
type PickTaskCreatedEvent struct {
	TaskID    string    `json:"taskId"`
	OrderID   string    `json:"orderId"`
	WaveID    string    `json:"waveId"`
	ItemCount int       `json:"itemCount"`
	CreatedAt time.Time `json:"createdAt"`
}

func (e *PickTaskCreatedEvent) EventType() string    { return "wms.picking.task-created" }
func (e *PickTaskCreatedEvent) OccurredAt() time.Time { return e.CreatedAt }

// PickTaskAssignedEvent is published when a task is assigned to a picker
type PickTaskAssignedEvent struct {
	TaskID     string    `json:"taskId"`
	PickerID   string    `json:"pickerId"`
	ToteID     string    `json:"toteId"`
	AssignedAt time.Time `json:"assignedAt"`
}

func (e *PickTaskAssignedEvent) EventType() string    { return "wms.picking.task-assigned" }
func (e *PickTaskAssignedEvent) OccurredAt() time.Time { return e.AssignedAt }

// ItemPickedEvent is published when an item is picked
type ItemPickedEvent struct {
	TaskID     string    `json:"taskId"`
	SKU        string    `json:"sku"`
	Quantity   int       `json:"quantity"`
	LocationID string    `json:"locationId"`
	ToteID     string    `json:"toteId"`
	PickedAt   time.Time `json:"pickedAt"`
}

func (e *ItemPickedEvent) EventType() string    { return "wms.picking.item-picked" }
func (e *ItemPickedEvent) OccurredAt() time.Time { return e.PickedAt }

// PickTaskCompletedEvent is published when a task is completed
type PickTaskCompletedEvent struct {
	TaskID      string           `json:"taskId"`
	OrderID     string           `json:"orderId"`
	PickerID    string           `json:"pickerId"`
	TotalItems  int              `json:"totalItems"`
	PickedItems int              `json:"pickedItems"`
	PickedList  []PickedItemInfo `json:"pickedItems"`
	CompletedAt time.Time        `json:"completedAt"`
}

func (e *PickTaskCompletedEvent) EventType() string    { return "wms.picking.task-completed" }
func (e *PickTaskCompletedEvent) OccurredAt() time.Time { return e.CompletedAt }

// PickExceptionEvent is published when a picking exception occurs
type PickExceptionEvent struct {
	TaskID       string    `json:"taskId"`
	ExceptionID  string    `json:"exceptionId"`
	SKU          string    `json:"sku"`
	LocationID   string    `json:"locationId"`
	Reason       string    `json:"reason"`
	RequestedQty int       `json:"requestedQty"`
	AvailableQty int       `json:"availableQty"`
	ReportedAt   time.Time `json:"reportedAt"`
}

func (e *PickExceptionEvent) EventType() string    { return "wms.picking.exception" }
func (e *PickExceptionEvent) OccurredAt() time.Time { return e.ReportedAt }
