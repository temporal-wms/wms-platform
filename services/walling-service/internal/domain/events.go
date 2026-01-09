package domain

import "time"

// DomainEvent represents a domain event
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// WallingTaskCreatedEvent is emitted when a walling task is created
type WallingTaskCreatedEvent struct {
	TaskID         string    `json:"taskId"`
	OrderID        string    `json:"orderId"`
	WaveID         string    `json:"waveId"`
	PutWallID      string    `json:"putWallId"`
	DestinationBin string    `json:"destinationBin"`
	ItemCount      int       `json:"itemCount"`
	CreatedAt      time.Time `json:"createdAt"`
}

func (e *WallingTaskCreatedEvent) EventType() string     { return "wms.walling.task-created" }
func (e *WallingTaskCreatedEvent) OccurredAt() time.Time { return e.CreatedAt }

// WallingTaskAssignedEvent is emitted when a walling task is assigned to a walliner
type WallingTaskAssignedEvent struct {
	TaskID     string    `json:"taskId"`
	OrderID    string    `json:"orderId"`
	WallinerID string    `json:"wallinerId"`
	Station    string    `json:"station"`
	AssignedAt time.Time `json:"assignedAt"`
}

func (e *WallingTaskAssignedEvent) EventType() string     { return "wms.walling.task-assigned" }
func (e *WallingTaskAssignedEvent) OccurredAt() time.Time { return e.AssignedAt }

// ItemSortedEvent is emitted when an item is sorted to a bin
type ItemSortedEvent struct {
	TaskID   string    `json:"taskId"`
	OrderID  string    `json:"orderId"`
	SKU      string    `json:"sku"`
	Quantity int       `json:"quantity"`
	ToteID   string    `json:"toteId"`
	BinID    string    `json:"binId"`
	SortedAt time.Time `json:"sortedAt"`
}

func (e *ItemSortedEvent) EventType() string     { return "wms.walling.item-sorted" }
func (e *ItemSortedEvent) OccurredAt() time.Time { return e.SortedAt }

// WallingTaskCompletedEvent is emitted when a walling task is completed
type WallingTaskCompletedEvent struct {
	TaskID         string    `json:"taskId"`
	OrderID        string    `json:"orderId"`
	WallinerID     string    `json:"wallinerId"`
	DestinationBin string    `json:"destinationBin"`
	ItemsSorted    int       `json:"itemsSorted"`
	CompletedAt    time.Time `json:"completedAt"`
}

func (e *WallingTaskCompletedEvent) EventType() string     { return "wms.walling.task-completed" }
func (e *WallingTaskCompletedEvent) OccurredAt() time.Time { return e.CompletedAt }
