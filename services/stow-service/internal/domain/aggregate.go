package domain

import (
	"errors"
	"math/rand"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PutawayTask errors
var (
	ErrTaskNotFound           = errors.New("putaway task not found")
	ErrInvalidTaskStatus      = errors.New("invalid task status")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrNoAvailableLocations   = errors.New("no available locations for stow")
	ErrInvalidStorageStrategy = errors.New("invalid storage strategy")
	ErrTaskAlreadyAssigned    = errors.New("task already assigned to a worker")
	ErrTaskNotAssigned        = errors.New("task not assigned to any worker")
	ErrLocationCapacityExceeded = errors.New("location capacity exceeded")
)

// StorageStrategy represents the strategy for determining storage locations
type StorageStrategy string

const (
	// StorageChaotic uses random placement - Amazon-style (DEFAULT)
	StorageChaotic StorageStrategy = "chaotic"
	// StorageDirected uses system-assigned locations based on rules
	StorageDirected StorageStrategy = "directed"
	// StorageVelocity places items based on pick frequency
	StorageVelocity StorageStrategy = "velocity"
	// StorageZoneBased places items by product category
	StorageZoneBased StorageStrategy = "zone_based"
)

// IsValid checks if the storage strategy is valid
func (s StorageStrategy) IsValid() bool {
	switch s {
	case StorageChaotic, StorageDirected, StorageVelocity, StorageZoneBased:
		return true
	default:
		return false
	}
}

// PutawayStatus represents the status of a putaway task
type PutawayStatus string

const (
	PutawayStatusPending    PutawayStatus = "pending"
	PutawayStatusAssigned   PutawayStatus = "assigned"
	PutawayStatusInProgress PutawayStatus = "in_progress"
	PutawayStatusCompleted  PutawayStatus = "completed"
	PutawayStatusCancelled  PutawayStatus = "cancelled"
	PutawayStatusFailed     PutawayStatus = "failed"
)

// IsValid checks if the status is valid
func (s PutawayStatus) IsValid() bool {
	switch s {
	case PutawayStatusPending, PutawayStatusAssigned, PutawayStatusInProgress,
		PutawayStatusCompleted, PutawayStatusCancelled, PutawayStatusFailed:
		return true
	default:
		return false
	}
}

// CanTransitionTo checks if the status can transition to another status
func (s PutawayStatus) CanTransitionTo(target PutawayStatus) bool {
	validTransitions := map[PutawayStatus][]PutawayStatus{
		PutawayStatusPending:    {PutawayStatusAssigned, PutawayStatusCancelled},
		PutawayStatusAssigned:   {PutawayStatusInProgress, PutawayStatusPending, PutawayStatusCancelled},
		PutawayStatusInProgress: {PutawayStatusCompleted, PutawayStatusFailed, PutawayStatusCancelled},
		PutawayStatusCompleted:  {},
		PutawayStatusCancelled:  {},
		PutawayStatusFailed:     {PutawayStatusPending}, // Can retry
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

// StorageLocation represents a potential storage location
type StorageLocation struct {
	LocationID      string  `bson:"locationId" json:"locationId"`
	Zone            string  `bson:"zone" json:"zone"`
	Aisle           string  `bson:"aisle" json:"aisle"`
	Rack            int     `bson:"rack" json:"rack"`
	Level           int     `bson:"level" json:"level"`
	Bin             string  `bson:"bin" json:"bin"`
	Capacity        int     `bson:"capacity" json:"capacity"`
	CurrentQuantity int     `bson:"currentQuantity" json:"currentQuantity"`
	MaxWeight       float64 `bson:"maxWeight" json:"maxWeight"`
	CurrentWeight   float64 `bson:"currentWeight" json:"currentWeight"`
	// Constraints
	AllowsHazmat    bool `bson:"allowsHazmat" json:"allowsHazmat"`
	AllowsColdChain bool `bson:"allowsColdChain" json:"allowsColdChain"`
	AllowsOversized bool `bson:"allowsOversized" json:"allowsOversized"`
}

// AvailableCapacity returns the remaining capacity
func (l *StorageLocation) AvailableCapacity() int {
	return l.Capacity - l.CurrentQuantity
}

// AvailableWeight returns the remaining weight capacity
func (l *StorageLocation) AvailableWeight() float64 {
	return l.MaxWeight - l.CurrentWeight
}

// CanAcceptItem checks if the location can accept an item with given constraints
func (l *StorageLocation) CanAcceptItem(quantity int, weight float64, isHazmat, isColdChain, isOversized bool) bool {
	if l.AvailableCapacity() < quantity {
		return false
	}
	if l.AvailableWeight() < weight {
		return false
	}
	if isHazmat && !l.AllowsHazmat {
		return false
	}
	if isColdChain && !l.AllowsColdChain {
		return false
	}
	if isOversized && !l.AllowsOversized {
		return false
	}
	return true
}

// ItemConstraints represents constraints for an item being stowed
type ItemConstraints struct {
	IsHazmat        bool    `bson:"isHazmat" json:"isHazmat"`
	RequiresColdChain bool  `bson:"requiresColdChain" json:"requiresColdChain"`
	IsOversized     bool    `bson:"isOversized" json:"isOversized"`
	IsFragile       bool    `bson:"isFragile" json:"isFragile"`
	Weight          float64 `bson:"weight" json:"weight"`
}

// PutawayTask represents a task to stow items from receiving to storage
type PutawayTask struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TaskID           string             `bson:"taskId" json:"taskId"`
	ShipmentID       string             `bson:"shipmentId,omitempty" json:"shipmentId,omitempty"`
	SKU              string             `bson:"sku" json:"sku"`
	ProductName      string             `bson:"productName" json:"productName"`
	Quantity         int                `bson:"quantity" json:"quantity"`
	SourceToteID     string             `bson:"sourceToteId" json:"sourceToteId"`
	SourceLocationID string             `bson:"sourceLocationId,omitempty" json:"sourceLocationId,omitempty"`
	TargetLocationID string             `bson:"targetLocationId,omitempty" json:"targetLocationId,omitempty"`
	TargetLocation   *StorageLocation   `bson:"targetLocation,omitempty" json:"targetLocation,omitempty"`
	Strategy         StorageStrategy    `bson:"strategy" json:"strategy"`
	Constraints      ItemConstraints    `bson:"constraints" json:"constraints"`
	Status           PutawayStatus      `bson:"status" json:"status"`
	AssignedWorkerID string             `bson:"assignedWorkerId,omitempty" json:"assignedWorkerId,omitempty"`
	Priority         int                `bson:"priority" json:"priority"` // 1=highest, 5=lowest
	StowedQuantity   int                `bson:"stowedQuantity" json:"stowedQuantity"`
	FailureReason    string             `bson:"failureReason,omitempty" json:"failureReason,omitempty"`
	AssignedAt       *time.Time         `bson:"assignedAt,omitempty" json:"assignedAt,omitempty"`
	StartedAt        *time.Time         `bson:"startedAt,omitempty" json:"startedAt,omitempty"`
	CompletedAt      *time.Time         `bson:"completedAt,omitempty" json:"completedAt,omitempty"`
	CreatedAt        time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time          `bson:"updatedAt" json:"updatedAt"`
	DomainEvents     []DomainEvent      `bson:"-" json:"-"`
}

// NewPutawayTask creates a new PutawayTask with default chaotic storage
func NewPutawayTask(
	taskID, shipmentID, sku, productName string,
	quantity int,
	sourceToteID string,
	constraints ItemConstraints,
) *PutawayTask {
	now := time.Now().UTC()
	task := &PutawayTask{
		ID:           primitive.NewObjectID(),
		TaskID:       taskID,
		ShipmentID:   shipmentID,
		SKU:          sku,
		ProductName:  productName,
		Quantity:     quantity,
		SourceToteID: sourceToteID,
		Strategy:     StorageChaotic, // Default to chaotic storage
		Constraints:  constraints,
		Status:       PutawayStatusPending,
		Priority:     3, // Default priority
		CreatedAt:    now,
		UpdatedAt:    now,
		DomainEvents: make([]DomainEvent, 0),
	}

	task.addDomainEvent(&PutawayTaskCreatedEvent{
		TaskID:       taskID,
		SKU:          sku,
		Quantity:     quantity,
		SourceToteID: sourceToteID,
		Strategy:     string(StorageChaotic),
		CreatedAt:    now,
	})

	return task
}

// SetStrategy sets the storage strategy
func (t *PutawayTask) SetStrategy(strategy StorageStrategy) error {
	if !strategy.IsValid() {
		return ErrInvalidStorageStrategy
	}
	t.Strategy = strategy
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// AssignToWorker assigns the task to a worker
func (t *PutawayTask) AssignToWorker(workerID string) error {
	if t.Status != PutawayStatusPending {
		return ErrInvalidStatusTransition
	}

	now := time.Now().UTC()
	t.AssignedWorkerID = workerID
	t.Status = PutawayStatusAssigned
	t.AssignedAt = &now
	t.UpdatedAt = now

	return nil
}

// Unassign removes the worker assignment
func (t *PutawayTask) Unassign() error {
	if t.Status != PutawayStatusAssigned {
		return ErrInvalidStatusTransition
	}

	t.AssignedWorkerID = ""
	t.Status = PutawayStatusPending
	t.AssignedAt = nil
	t.UpdatedAt = time.Now().UTC()

	return nil
}

// AssignLocation assigns a target storage location
func (t *PutawayTask) AssignLocation(location StorageLocation) error {
	if !location.CanAcceptItem(t.Quantity, t.Constraints.Weight,
		t.Constraints.IsHazmat, t.Constraints.RequiresColdChain, t.Constraints.IsOversized) {
		return ErrLocationCapacityExceeded
	}

	t.TargetLocationID = location.LocationID
	t.TargetLocation = &location
	t.UpdatedAt = time.Now().UTC()

	t.addDomainEvent(&LocationAssignedEvent{
		TaskID:     t.TaskID,
		SKU:        t.SKU,
		LocationID: location.LocationID,
		Zone:       location.Zone,
		Strategy:   string(t.Strategy),
		AssignedAt: t.UpdatedAt,
	})

	return nil
}

// SelectRandomLocation implements chaotic storage by selecting a random location
func (t *PutawayTask) SelectRandomLocation(availableLocations []StorageLocation) (*StorageLocation, error) {
	// Filter locations that can accept this item
	eligibleLocations := make([]StorageLocation, 0)
	for _, loc := range availableLocations {
		if loc.CanAcceptItem(t.Quantity, t.Constraints.Weight,
			t.Constraints.IsHazmat, t.Constraints.RequiresColdChain, t.Constraints.IsOversized) {
			eligibleLocations = append(eligibleLocations, loc)
		}
	}

	if len(eligibleLocations) == 0 {
		return nil, ErrNoAvailableLocations
	}

	// Chaotic storage: select randomly from eligible locations
	randomIndex := rand.Intn(len(eligibleLocations))
	selected := &eligibleLocations[randomIndex]

	return selected, nil
}

// Start starts the stow process
func (t *PutawayTask) Start() error {
	if t.Status != PutawayStatusAssigned {
		return ErrInvalidStatusTransition
	}

	if t.TargetLocationID == "" {
		return errors.New("no target location assigned")
	}

	now := time.Now().UTC()
	t.Status = PutawayStatusInProgress
	t.StartedAt = &now
	t.UpdatedAt = now

	return nil
}

// RecordStow records stowing progress
func (t *PutawayTask) RecordStow(quantity int) error {
	if t.Status != PutawayStatusInProgress {
		return ErrInvalidStatusTransition
	}

	t.StowedQuantity += quantity
	t.UpdatedAt = time.Now().UTC()

	return nil
}

// Complete completes the putaway task
func (t *PutawayTask) Complete() error {
	if t.Status != PutawayStatusInProgress {
		return ErrInvalidStatusTransition
	}

	now := time.Now().UTC()
	t.Status = PutawayStatusCompleted
	t.CompletedAt = &now
	t.UpdatedAt = now

	t.addDomainEvent(&ItemStowedEvent{
		TaskID:     t.TaskID,
		SKU:        t.SKU,
		Quantity:   t.StowedQuantity,
		LocationID: t.TargetLocationID,
		ToteID:     t.SourceToteID,
		StowedAt:   now,
	})

	return nil
}

// Fail marks the task as failed
func (t *PutawayTask) Fail(reason string) error {
	if t.Status == PutawayStatusCompleted || t.Status == PutawayStatusCancelled {
		return ErrInvalidStatusTransition
	}

	t.Status = PutawayStatusFailed
	t.FailureReason = reason
	t.UpdatedAt = time.Now().UTC()

	return nil
}

// Cancel cancels the task
func (t *PutawayTask) Cancel(reason string) error {
	if t.Status == PutawayStatusCompleted {
		return ErrInvalidStatusTransition
	}

	t.Status = PutawayStatusCancelled
	t.FailureReason = reason
	t.UpdatedAt = time.Now().UTC()

	return nil
}

// Retry resets the task for retry
func (t *PutawayTask) Retry() error {
	if t.Status != PutawayStatusFailed {
		return ErrInvalidStatusTransition
	}

	t.Status = PutawayStatusPending
	t.AssignedWorkerID = ""
	t.AssignedAt = nil
	t.StartedAt = nil
	t.StowedQuantity = 0
	t.FailureReason = ""
	t.UpdatedAt = time.Now().UTC()

	return nil
}

// RemainingQuantity returns the quantity still to be stowed
func (t *PutawayTask) RemainingQuantity() int {
	return t.Quantity - t.StowedQuantity
}

// IsComplete checks if stowing is complete
func (t *PutawayTask) IsComplete() bool {
	return t.StowedQuantity >= t.Quantity
}

// addDomainEvent adds a domain event
func (t *PutawayTask) addDomainEvent(event DomainEvent) {
	t.DomainEvents = append(t.DomainEvents, event)
}

// GetDomainEvents returns all domain events
func (t *PutawayTask) GetDomainEvents() []DomainEvent {
	return t.DomainEvents
}

// ClearDomainEvents clears all domain events
func (t *PutawayTask) ClearDomainEvents() {
	t.DomainEvents = make([]DomainEvent, 0)
}
