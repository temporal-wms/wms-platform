package domain

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Station errors
var (
	ErrStationNotActive      = errors.New("station is not active")
	ErrStationAtCapacity     = errors.New("station is at maximum capacity")
	ErrCapabilityNotFound    = errors.New("capability not found on station")
	ErrCapabilityExists      = errors.New("capability already exists on station")
	ErrInvalidStationType    = errors.New("invalid station type")
	ErrInvalidStationStatus  = errors.New("invalid station status")
)

// StationCapability represents a capability that a station supports
type StationCapability string

const (
	// Item count capabilities
	CapabilitySingleItem StationCapability = "single_item"
	CapabilityMultiItem  StationCapability = "multi_item"

	// Special handling capabilities
	CapabilityGiftWrap  StationCapability = "gift_wrap"
	CapabilityHazmat    StationCapability = "hazmat"
	CapabilityOversized StationCapability = "oversized"
	CapabilityFragile   StationCapability = "fragile"
	CapabilityColdChain StationCapability = "cold_chain"
	CapabilityHighValue StationCapability = "high_value"
)

// IsValid checks if the capability is valid
func (c StationCapability) IsValid() bool {
	switch c {
	case CapabilitySingleItem, CapabilityMultiItem,
		CapabilityGiftWrap, CapabilityHazmat,
		CapabilityOversized, CapabilityFragile,
		CapabilityColdChain, CapabilityHighValue:
		return true
	default:
		return false
	}
}

// StationStatus represents the operational status of a station
type StationStatus string

const (
	StationStatusActive      StationStatus = "active"
	StationStatusInactive    StationStatus = "inactive"
	StationStatusMaintenance StationStatus = "maintenance"
)

// IsValid checks if the status is valid
func (s StationStatus) IsValid() bool {
	switch s {
	case StationStatusActive, StationStatusInactive, StationStatusMaintenance:
		return true
	default:
		return false
	}
}

// StationType represents the type of station
type StationType string

const (
	StationTypePacking       StationType = "packing"
	StationTypeConsolidation StationType = "consolidation"
	StationTypeShipping      StationType = "shipping"
	StationTypeReceiving     StationType = "receiving"
)

// IsValid checks if the station type is valid
func (t StationType) IsValid() bool {
	switch t {
	case StationTypePacking, StationTypeConsolidation, StationTypeShipping, StationTypeReceiving:
		return true
	default:
		return false
	}
}

// StationEquipment represents equipment at a station
type StationEquipment struct {
	EquipmentID   string `bson:"equipmentId" json:"equipmentId"`
	EquipmentType string `bson:"equipmentType" json:"equipmentType"` // scale, printer, cold_storage, hazmat_cabinet
	Status        string `bson:"status" json:"status"`               // active, inactive, maintenance
}

// Station represents a packing/consolidation/shipping station with capabilities
type Station struct {
	ID                 primitive.ObjectID  `bson:"_id,omitempty"`
	StationID          string              `bson:"stationId"`
	Name               string              `bson:"name"`
	Zone               string              `bson:"zone"`
	StationType        StationType         `bson:"stationType"`
	Status             StationStatus       `bson:"status"`
	Capabilities       []StationCapability `bson:"capabilities"`
	MaxConcurrentTasks int                 `bson:"maxConcurrentTasks"`
	CurrentTasks       int                 `bson:"currentTasks"`
	AssignedWorkerID   string              `bson:"assignedWorkerId,omitempty"`
	Equipment          []StationEquipment  `bson:"equipment"`
	CreatedAt          time.Time           `bson:"createdAt"`
	UpdatedAt          time.Time           `bson:"updatedAt"`
	DomainEvents       []DomainEvent       `bson:"-"`
}

// NewStation creates a new Station aggregate
func NewStation(stationID, name, zone string, stationType StationType, maxConcurrentTasks int) (*Station, error) {
	if !stationType.IsValid() {
		return nil, ErrInvalidStationType
	}

	if maxConcurrentTasks <= 0 {
		maxConcurrentTasks = 1
	}

	now := time.Now()
	station := &Station{
		StationID:          stationID,
		Name:               name,
		Zone:               zone,
		StationType:        stationType,
		Status:             StationStatusActive,
		Capabilities:       make([]StationCapability, 0),
		MaxConcurrentTasks: maxConcurrentTasks,
		CurrentTasks:       0,
		Equipment:          make([]StationEquipment, 0),
		CreatedAt:          now,
		UpdatedAt:          now,
		DomainEvents:       make([]DomainEvent, 0),
	}

	station.AddDomainEvent(&StationCreatedEvent{
		StationID:   stationID,
		Name:        name,
		Zone:        zone,
		StationType: string(stationType),
		CreatedAt:   now,
	})

	return station, nil
}

// HasCapability checks if the station has a specific capability
func (s *Station) HasCapability(cap StationCapability) bool {
	for _, existing := range s.Capabilities {
		if existing == cap {
			return true
		}
	}
	return false
}

// HasAllCapabilities checks if the station has ALL required capabilities
func (s *Station) HasAllCapabilities(required []StationCapability) bool {
	for _, req := range required {
		if !s.HasCapability(req) {
			return false
		}
	}
	return true
}

// AddCapability adds a capability to the station
func (s *Station) AddCapability(cap StationCapability) error {
	if !cap.IsValid() {
		return errors.New("invalid capability")
	}

	if s.HasCapability(cap) {
		return ErrCapabilityExists
	}

	s.Capabilities = append(s.Capabilities, cap)
	s.UpdatedAt = time.Now()

	s.AddDomainEvent(&StationCapabilityAddedEvent{
		StationID:  s.StationID,
		Capability: string(cap),
		AddedAt:    s.UpdatedAt,
	})

	return nil
}

// RemoveCapability removes a capability from the station
func (s *Station) RemoveCapability(cap StationCapability) error {
	for i, existing := range s.Capabilities {
		if existing == cap {
			s.Capabilities = append(s.Capabilities[:i], s.Capabilities[i+1:]...)
			s.UpdatedAt = time.Now()

			s.AddDomainEvent(&StationCapabilityRemovedEvent{
				StationID:  s.StationID,
				Capability: string(cap),
				RemovedAt:  s.UpdatedAt,
			})

			return nil
		}
	}
	return ErrCapabilityNotFound
}

// SetCapabilities replaces all capabilities with the provided list
func (s *Station) SetCapabilities(capabilities []StationCapability) error {
	for _, cap := range capabilities {
		if !cap.IsValid() {
			return errors.New("invalid capability: " + string(cap))
		}
	}

	s.Capabilities = capabilities
	s.UpdatedAt = time.Now()

	s.AddDomainEvent(&StationCapabilitiesUpdatedEvent{
		StationID:    s.StationID,
		Capabilities: s.GetCapabilityStrings(),
		UpdatedAt:    s.UpdatedAt,
	})

	return nil
}

// GetCapabilityStrings returns capabilities as strings
func (s *Station) GetCapabilityStrings() []string {
	caps := make([]string, len(s.Capabilities))
	for i, cap := range s.Capabilities {
		caps[i] = string(cap)
	}
	return caps
}

// CanAcceptTask checks if the station can accept a new task
func (s *Station) CanAcceptTask() bool {
	return s.Status == StationStatusActive && s.CurrentTasks < s.MaxConcurrentTasks
}

// IncrementTasks increments the current task count
func (s *Station) IncrementTasks() error {
	if !s.CanAcceptTask() {
		if s.Status != StationStatusActive {
			return ErrStationNotActive
		}
		return ErrStationAtCapacity
	}

	s.CurrentTasks++
	s.UpdatedAt = time.Now()
	return nil
}

// DecrementTasks decrements the current task count
func (s *Station) DecrementTasks() {
	if s.CurrentTasks > 0 {
		s.CurrentTasks--
		s.UpdatedAt = time.Now()
	}
}

// AssignWorker assigns a worker to the station
func (s *Station) AssignWorker(workerID string) error {
	if s.Status != StationStatusActive {
		return ErrStationNotActive
	}

	s.AssignedWorkerID = workerID
	s.UpdatedAt = time.Now()

	s.AddDomainEvent(&WorkerAssignedToStationEvent{
		StationID:  s.StationID,
		WorkerID:   workerID,
		AssignedAt: s.UpdatedAt,
	})

	return nil
}

// UnassignWorker removes the worker assignment from the station
func (s *Station) UnassignWorker() {
	s.AssignedWorkerID = ""
	s.UpdatedAt = time.Now()
}

// SetStatus updates the station status
func (s *Station) SetStatus(status StationStatus) error {
	if !status.IsValid() {
		return ErrInvalidStationStatus
	}

	oldStatus := s.Status
	s.Status = status
	s.UpdatedAt = time.Now()

	s.AddDomainEvent(&StationStatusChangedEvent{
		StationID: s.StationID,
		OldStatus: string(oldStatus),
		NewStatus: string(status),
		ChangedAt: s.UpdatedAt,
	})

	return nil
}

// Activate sets the station to active status
func (s *Station) Activate() error {
	return s.SetStatus(StationStatusActive)
}

// Deactivate sets the station to inactive status
func (s *Station) Deactivate() error {
	return s.SetStatus(StationStatusInactive)
}

// SetMaintenance sets the station to maintenance status
func (s *Station) SetMaintenance() error {
	return s.SetStatus(StationStatusMaintenance)
}

// AddEquipment adds equipment to the station
func (s *Station) AddEquipment(equipment StationEquipment) {
	s.Equipment = append(s.Equipment, equipment)
	s.UpdatedAt = time.Now()
}

// RemoveEquipment removes equipment from the station by ID
func (s *Station) RemoveEquipment(equipmentID string) bool {
	for i, eq := range s.Equipment {
		if eq.EquipmentID == equipmentID {
			s.Equipment = append(s.Equipment[:i], s.Equipment[i+1:]...)
			s.UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// GetAvailableCapacity returns the number of additional tasks the station can accept
func (s *Station) GetAvailableCapacity() int {
	if s.Status != StationStatusActive {
		return 0
	}
	return s.MaxConcurrentTasks - s.CurrentTasks
}

// AddDomainEvent adds a domain event
func (s *Station) AddDomainEvent(event DomainEvent) {
	s.DomainEvents = append(s.DomainEvents, event)
}

// ClearDomainEvents clears all domain events
func (s *Station) ClearDomainEvents() {
	s.DomainEvents = make([]DomainEvent, 0)
}

// GetDomainEvents returns all domain events
func (s *Station) GetDomainEvents() []DomainEvent {
	return s.DomainEvents
}

// Station Domain Events

// StationCreatedEvent is emitted when a station is created
type StationCreatedEvent struct {
	StationID   string    `json:"stationId"`
	Name        string    `json:"name"`
	Zone        string    `json:"zone"`
	StationType string    `json:"stationType"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (e *StationCreatedEvent) EventType() string    { return "station.created" }
func (e *StationCreatedEvent) OccurredAt() time.Time { return e.CreatedAt }

// StationCapabilityAddedEvent is emitted when a capability is added
type StationCapabilityAddedEvent struct {
	StationID  string    `json:"stationId"`
	Capability string    `json:"capability"`
	AddedAt    time.Time `json:"addedAt"`
}

func (e *StationCapabilityAddedEvent) EventType() string    { return "station.capability.added" }
func (e *StationCapabilityAddedEvent) OccurredAt() time.Time { return e.AddedAt }

// StationCapabilityRemovedEvent is emitted when a capability is removed
type StationCapabilityRemovedEvent struct {
	StationID  string    `json:"stationId"`
	Capability string    `json:"capability"`
	RemovedAt  time.Time `json:"removedAt"`
}

func (e *StationCapabilityRemovedEvent) EventType() string    { return "station.capability.removed" }
func (e *StationCapabilityRemovedEvent) OccurredAt() time.Time { return e.RemovedAt }

// StationCapabilitiesUpdatedEvent is emitted when capabilities are bulk updated
type StationCapabilitiesUpdatedEvent struct {
	StationID    string    `json:"stationId"`
	Capabilities []string  `json:"capabilities"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (e *StationCapabilitiesUpdatedEvent) EventType() string    { return "station.capabilities.updated" }
func (e *StationCapabilitiesUpdatedEvent) OccurredAt() time.Time { return e.UpdatedAt }

// StationStatusChangedEvent is emitted when station status changes
type StationStatusChangedEvent struct {
	StationID string    `json:"stationId"`
	OldStatus string    `json:"oldStatus"`
	NewStatus string    `json:"newStatus"`
	ChangedAt time.Time `json:"changedAt"`
}

func (e *StationStatusChangedEvent) EventType() string    { return "station.status.changed" }
func (e *StationStatusChangedEvent) OccurredAt() time.Time { return e.ChangedAt }

// WorkerAssignedToStationEvent is emitted when a worker is assigned to a station
type WorkerAssignedToStationEvent struct {
	StationID  string    `json:"stationId"`
	WorkerID   string    `json:"workerId"`
	AssignedAt time.Time `json:"assignedAt"`
}

func (e *WorkerAssignedToStationEvent) EventType() string    { return "station.worker.assigned" }
func (e *WorkerAssignedToStationEvent) OccurredAt() time.Time { return e.AssignedAt }
