package domain

import "time"

// DomainEvent is the interface for all domain events
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// PackTaskCreatedEvent is published when a pack task is created
type PackTaskCreatedEvent struct {
	TaskID    string    `json:"taskId"`
	OrderID   string    `json:"orderId"`
	ItemCount int       `json:"itemCount"`
	CreatedAt time.Time `json:"createdAt"`
}

func (e *PackTaskCreatedEvent) EventType() string    { return "wms.packing.task-created" }
func (e *PackTaskCreatedEvent) OccurredAt() time.Time { return e.CreatedAt }

// PackagingSuggestedEvent is published when packaging is selected
type PackagingSuggestedEvent struct {
	TaskID      string     `json:"taskId"`
	OrderID     string     `json:"orderId"`
	PackageID   string     `json:"packageId"`
	Type        string     `json:"type"`
	Dimensions  Dimensions `json:"dimensions"`
	SuggestedAt time.Time  `json:"suggestedAt"`
}

func (e *PackagingSuggestedEvent) EventType() string    { return "wms.packing.packaging-suggested" }
func (e *PackagingSuggestedEvent) OccurredAt() time.Time { return e.SuggestedAt }

// PackageSealedEvent is published when package is sealed
type PackageSealedEvent struct {
	TaskID    string    `json:"taskId"`
	PackageID string    `json:"packageId"`
	SealedAt  time.Time `json:"sealedAt"`
}

func (e *PackageSealedEvent) EventType() string    { return "wms.packing.package-sealed" }
func (e *PackageSealedEvent) OccurredAt() time.Time { return e.SealedAt }

// LabelAppliedEvent is published when shipping label is applied
type LabelAppliedEvent struct {
	TaskID         string    `json:"taskId"`
	PackageID      string    `json:"packageId"`
	TrackingNumber string    `json:"trackingNumber"`
	Carrier        string    `json:"carrier"`
	AppliedAt      time.Time `json:"appliedAt"`
}

func (e *LabelAppliedEvent) EventType() string    { return "wms.packing.label-applied" }
func (e *LabelAppliedEvent) OccurredAt() time.Time { return e.AppliedAt }

// PackTaskCompletedEvent is published when pack task is completed
type PackTaskCompletedEvent struct {
	TaskID         string    `json:"taskId"`
	OrderID        string    `json:"orderId"`
	PackageID      string    `json:"packageId"`
	TrackingNumber string    `json:"trackingNumber"`
	Carrier        string    `json:"carrier"`
	TotalWeight    float64   `json:"totalWeight"`
	CompletedAt    time.Time `json:"completedAt"`
}

func (e *PackTaskCompletedEvent) EventType() string    { return "wms.packing.task-completed" }
func (e *PackTaskCompletedEvent) OccurredAt() time.Time { return e.CompletedAt }
