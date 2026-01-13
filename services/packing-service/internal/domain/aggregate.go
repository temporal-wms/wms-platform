package domain

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Errors
var (
	ErrPackTaskCompleted   = errors.New("pack task is already completed")
	ErrPackageSealed       = errors.New("package is already sealed")
	ErrNoItemsToPack       = errors.New("no items to pack")
	ErrInvalidPackageType  = errors.New("invalid package type")
)

// PackTaskStatus represents the status of a pack task
type PackTaskStatus string

const (
	PackTaskStatusPending    PackTaskStatus = "pending"
	PackTaskStatusInProgress PackTaskStatus = "in_progress"
	PackTaskStatusPacked     PackTaskStatus = "packed"
	PackTaskStatusLabeled    PackTaskStatus = "labeled"
	PackTaskStatusCompleted  PackTaskStatus = "completed"
	PackTaskStatusCancelled  PackTaskStatus = "cancelled"
)

// PackageType represents the type of packaging
type PackageType string

const (
	PackageTypeBox       PackageType = "box"
	PackageTypeEnvelope  PackageType = "envelope"
	PackageTypeBag       PackageType = "bag"
	PackageTypePadded    PackageType = "padded_envelope"
	PackageTypeCustom    PackageType = "custom"
)

// PackTask is the aggregate root for the Packing bounded context
type PackTask struct {
	ID               primitive.ObjectID `bson:"_id,omitempty"`
	TaskID           string             `bson:"taskId"`
	TenantID         string             `bson:"tenantId"`
	FacilityID       string             `bson:"facilityId"`
	WarehouseID      string             `bson:"warehouseId"`
	OrderID          string             `bson:"orderId"`
	ConsolidationID  string             `bson:"consolidationId,omitempty"`
	WaveID           string             `bson:"waveId"`
	PackerID         string             `bson:"packerId,omitempty"`
	Status           PackTaskStatus     `bson:"status"`
	Items            []PackItem         `bson:"items"`
	Package          Package            `bson:"package"`
	ShippingLabel    *ShippingLabel     `bson:"shippingLabel,omitempty"`
	Station          string             `bson:"station"`
	Priority         int                `bson:"priority"`
	CreatedAt        time.Time          `bson:"createdAt"`
	UpdatedAt        time.Time          `bson:"updatedAt"`
	StartedAt        *time.Time         `bson:"startedAt,omitempty"`
	PackedAt         *time.Time         `bson:"packedAt,omitempty"`
	LabeledAt        *time.Time         `bson:"labeledAt,omitempty"`
	CompletedAt      *time.Time         `bson:"completedAt,omitempty"`
	DomainEvents     []DomainEvent      `bson:"-"`
}

// PackItem represents an item to be packed
type PackItem struct {
	SKU         string `bson:"sku"`
	ProductName string `bson:"productName"`
	Quantity    int    `bson:"quantity"`
	Weight      float64 `bson:"weight"` // in kg
	Fragile     bool   `bson:"fragile"`
	Verified    bool   `bson:"verified"`
}

// Package represents the packaging used
type Package struct {
	PackageID     string      `bson:"packageId"`
	Type          PackageType `bson:"type"`
	SuggestedType PackageType `bson:"suggestedType"`
	Dimensions    Dimensions  `bson:"dimensions"`
	Weight        float64     `bson:"weight"`     // Package weight in kg
	TotalWeight   float64     `bson:"totalWeight"` // Package + contents
	Materials     []string    `bson:"materials"`  // Bubble wrap, paper, etc.
	Sealed        bool        `bson:"sealed"`
	SealedAt      *time.Time  `bson:"sealedAt,omitempty"`
}

// Dimensions represents package dimensions
type Dimensions struct {
	Length float64 `bson:"length"` // in cm
	Width  float64 `bson:"width"`
	Height float64 `bson:"height"`
}

// ShippingLabel represents the shipping label
type ShippingLabel struct {
	TrackingNumber string    `bson:"trackingNumber"`
	Carrier        string    `bson:"carrier"`
	ServiceType    string    `bson:"serviceType"`
	LabelURL       string    `bson:"labelUrl,omitempty"`
	LabelData      string    `bson:"labelData,omitempty"` // Base64 encoded
	GeneratedAt    time.Time `bson:"generatedAt"`
	AppliedAt      *time.Time `bson:"appliedAt,omitempty"`
}

// NewPackTask creates a new PackTask aggregate
func NewPackTask(taskID, orderID, waveID string, items []PackItem) (*PackTask, error) {
	if len(items) == 0 {
		return nil, ErrNoItemsToPack
	}

	now := time.Now()

	// Calculate total weight
	totalWeight := 0.0
	for _, item := range items {
		totalWeight += item.Weight * float64(item.Quantity)
	}

	task := &PackTask{
		TaskID:       taskID,
		OrderID:      orderID,
		WaveID:       waveID,
		Status:       PackTaskStatusPending,
		Items:        items,
		Package:      Package{},
		Priority:     5,
		CreatedAt:    now,
		UpdatedAt:    now,
		DomainEvents: make([]DomainEvent, 0),
	}

	// Suggest package type
	task.Package.SuggestedType = suggestPackageType(items, totalWeight)

	task.AddDomainEvent(&PackTaskCreatedEvent{
		TaskID:    taskID,
		OrderID:   orderID,
		ItemCount: len(items),
		CreatedAt: now,
	})

	return task, nil
}

// Assign assigns the task to a packer
func (t *PackTask) Assign(packerID, station string) error {
	if t.Status == PackTaskStatusCompleted {
		return ErrPackTaskCompleted
	}

	t.PackerID = packerID
	t.Station = station
	t.UpdatedAt = time.Now()

	return nil
}

// Start marks the task as in progress
func (t *PackTask) Start() error {
	if t.Status != PackTaskStatusPending {
		return errors.New("task already started")
	}

	now := time.Now()
	t.Status = PackTaskStatusInProgress
	t.StartedAt = &now
	t.UpdatedAt = now

	return nil
}

// VerifyItem verifies an item is correct
func (t *PackTask) VerifyItem(sku string) error {
	if t.Status == PackTaskStatusCompleted {
		return ErrPackTaskCompleted
	}

	for i := range t.Items {
		if t.Items[i].SKU == sku {
			t.Items[i].Verified = true
			t.UpdatedAt = time.Now()
			return nil
		}
	}

	return errors.New("item not found")
}

// SelectPackaging selects the packaging to use
func (t *PackTask) SelectPackaging(packageType PackageType, dimensions Dimensions, materials []string) error {
	if t.Status == PackTaskStatusCompleted {
		return ErrPackTaskCompleted
	}

	// Calculate total weight
	totalWeight := 0.0
	for _, item := range t.Items {
		totalWeight += item.Weight * float64(item.Quantity)
	}

	t.Package = Package{
		PackageID:     generatePackageID(t.OrderID),
		Type:          packageType,
		SuggestedType: t.Package.SuggestedType,
		Dimensions:    dimensions,
		Materials:     materials,
		TotalWeight:   totalWeight + estimatePackageWeight(packageType),
	}

	t.UpdatedAt = time.Now()

	t.AddDomainEvent(&PackagingSuggestedEvent{
		TaskID:     t.TaskID,
		OrderID:    t.OrderID,
		PackageID:  t.Package.PackageID,
		Type:       string(packageType),
		Dimensions: dimensions,
		SuggestedAt: time.Now(),
	})

	return nil
}

// SealPackage marks the package as sealed
func (t *PackTask) SealPackage() error {
	if t.Package.Sealed {
		return ErrPackageSealed
	}

	// Verify all items
	for _, item := range t.Items {
		if !item.Verified {
			return errors.New("all items must be verified before sealing")
		}
	}

	now := time.Now()
	t.Package.Sealed = true
	t.Package.SealedAt = &now
	t.Status = PackTaskStatusPacked
	t.PackedAt = &now
	t.UpdatedAt = now

	t.AddDomainEvent(&PackageSealedEvent{
		TaskID:    t.TaskID,
		PackageID: t.Package.PackageID,
		SealedAt:  now,
	})

	return nil
}

// ApplyLabel applies the shipping label
func (t *PackTask) ApplyLabel(label ShippingLabel) error {
	if t.Status == PackTaskStatusCompleted {
		return ErrPackTaskCompleted
	}

	if !t.Package.Sealed {
		return errors.New("package must be sealed before applying label")
	}

	now := time.Now()
	label.AppliedAt = &now
	t.ShippingLabel = &label
	t.Status = PackTaskStatusLabeled
	t.LabeledAt = &now
	t.UpdatedAt = now

	t.AddDomainEvent(&LabelAppliedEvent{
		TaskID:         t.TaskID,
		PackageID:      t.Package.PackageID,
		TrackingNumber: label.TrackingNumber,
		Carrier:        label.Carrier,
		AppliedAt:      now,
	})

	return nil
}

// Complete marks the task as completed
func (t *PackTask) Complete() error {
	if t.Status == PackTaskStatusCompleted {
		return ErrPackTaskCompleted
	}

	if t.ShippingLabel == nil {
		return errors.New("label must be applied before completing")
	}

	now := time.Now()
	t.Status = PackTaskStatusCompleted
	t.CompletedAt = &now
	t.UpdatedAt = now

	t.AddDomainEvent(&PackTaskCompletedEvent{
		TaskID:         t.TaskID,
		OrderID:        t.OrderID,
		PackageID:      t.Package.PackageID,
		TrackingNumber: t.ShippingLabel.TrackingNumber,
		Carrier:        t.ShippingLabel.Carrier,
		TotalWeight:    t.Package.TotalWeight,
		CompletedAt:    now,
	})

	return nil
}

// Cancel cancels the task
func (t *PackTask) Cancel(reason string) error {
	if t.Status == PackTaskStatusCompleted {
		return ErrPackTaskCompleted
	}

	t.Status = PackTaskStatusCancelled
	t.UpdatedAt = time.Now()

	return nil
}

// GetProgress returns the progress status
func (t *PackTask) GetProgress() string {
	switch t.Status {
	case PackTaskStatusPending:
		return "Waiting to start"
	case PackTaskStatusInProgress:
		verified := 0
		for _, item := range t.Items {
			if item.Verified {
				verified++
			}
		}
		return "Verifying items: " + string(rune(verified)) + "/" + string(rune(len(t.Items)))
	case PackTaskStatusPacked:
		return "Packed, waiting for label"
	case PackTaskStatusLabeled:
		return "Labeled, ready for shipping"
	case PackTaskStatusCompleted:
		return "Completed"
	default:
		return "Unknown"
	}
}

// AddDomainEvent adds a domain event
func (t *PackTask) AddDomainEvent(event DomainEvent) {
	t.DomainEvents = append(t.DomainEvents, event)
}

// ClearDomainEvents clears all domain events
func (t *PackTask) ClearDomainEvents() {
	t.DomainEvents = make([]DomainEvent, 0)
}

// GetDomainEvents returns all domain events
func (t *PackTask) GetDomainEvents() []DomainEvent {
	return t.DomainEvents
}

// Helper functions

func suggestPackageType(items []PackItem, totalWeight float64) PackageType {
	// Check if any items are fragile
	hasFragile := false
	for _, item := range items {
		if item.Fragile {
			hasFragile = true
			break
		}
	}

	// Simple heuristics for package suggestion
	if len(items) == 1 && totalWeight < 0.5 && !hasFragile {
		return PackageTypeEnvelope
	}

	if totalWeight < 1.0 && !hasFragile {
		return PackageTypeBag
	}

	if hasFragile || totalWeight > 5.0 {
		return PackageTypeBox
	}

	return PackageTypePadded
}

func generatePackageID(orderID string) string {
	return "PKG-" + orderID + "-" + time.Now().Format("150405")
}

func estimatePackageWeight(packageType PackageType) float64 {
	switch packageType {
	case PackageTypeEnvelope:
		return 0.02
	case PackageTypeBag:
		return 0.05
	case PackageTypePadded:
		return 0.1
	case PackageTypeBox:
		return 0.3
	default:
		return 0.2
	}
}
