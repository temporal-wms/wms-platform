package domain

import "time"

// DomainEvent represents a domain event interface
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// PutawayTaskCreatedEvent is emitted when a putaway task is created
type PutawayTaskCreatedEvent struct {
	TaskID       string    `json:"taskId"`
	SKU          string    `json:"sku"`
	Quantity     int       `json:"quantity"`
	SourceToteID string    `json:"sourceToteId"`
	Strategy     string    `json:"strategy"`
	CreatedAt    time.Time `json:"createdAt"`
}

func (e *PutawayTaskCreatedEvent) EventType() string     { return "stow.task.created" }
func (e *PutawayTaskCreatedEvent) OccurredAt() time.Time { return e.CreatedAt }

// LocationAssignedEvent is emitted when a location is assigned to a task
type LocationAssignedEvent struct {
	TaskID     string    `json:"taskId"`
	SKU        string    `json:"sku"`
	LocationID string    `json:"locationId"`
	Zone       string    `json:"zone"`
	Strategy   string    `json:"strategy"`
	AssignedAt time.Time `json:"assignedAt"`
}

func (e *LocationAssignedEvent) EventType() string     { return "stow.location.assigned" }
func (e *LocationAssignedEvent) OccurredAt() time.Time { return e.AssignedAt }

// ItemStowedEvent is emitted when items are stowed
type ItemStowedEvent struct {
	TaskID     string    `json:"taskId"`
	SKU        string    `json:"sku"`
	Quantity   int       `json:"quantity"`
	LocationID string    `json:"locationId"`
	ToteID     string    `json:"toteId"`
	StowedAt   time.Time `json:"stowedAt"`
}

func (e *ItemStowedEvent) EventType() string     { return "stow.item.stowed" }
func (e *ItemStowedEvent) OccurredAt() time.Time { return e.StowedAt }

// PutawayTaskAssignedEvent is emitted when a task is assigned to a worker
type PutawayTaskAssignedEvent struct {
	TaskID     string    `json:"taskId"`
	WorkerID   string    `json:"workerId"`
	AssignedAt time.Time `json:"assignedAt"`
}

func (e *PutawayTaskAssignedEvent) EventType() string     { return "stow.task.assigned" }
func (e *PutawayTaskAssignedEvent) OccurredAt() time.Time { return e.AssignedAt }

// PutawayTaskCompletedEvent is emitted when a task is completed
type PutawayTaskCompletedEvent struct {
	TaskID       string    `json:"taskId"`
	SKU          string    `json:"sku"`
	Quantity     int       `json:"quantity"`
	LocationID   string    `json:"locationId"`
	WorkerID     string    `json:"workerId"`
	CompletedAt  time.Time `json:"completedAt"`
	DurationMins float64   `json:"durationMins"`
}

func (e *PutawayTaskCompletedEvent) EventType() string     { return "stow.task.completed" }
func (e *PutawayTaskCompletedEvent) OccurredAt() time.Time { return e.CompletedAt }

// PutawayTaskFailedEvent is emitted when a task fails
type PutawayTaskFailedEvent struct {
	TaskID    string    `json:"taskId"`
	SKU       string    `json:"sku"`
	Reason    string    `json:"reason"`
	FailedAt  time.Time `json:"failedAt"`
}

func (e *PutawayTaskFailedEvent) EventType() string     { return "stow.task.failed" }
func (e *PutawayTaskFailedEvent) OccurredAt() time.Time { return e.FailedAt }
