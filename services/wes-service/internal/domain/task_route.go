package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Errors
var (
	ErrRouteAlreadyCompleted = errors.New("route is already completed")
	ErrRouteAlreadyFailed    = errors.New("route has already failed")
	ErrInvalidStageIndex     = errors.New("invalid stage index")
	ErrStageNotPending       = errors.New("stage is not in pending status")
	ErrNoCurrentStage        = errors.New("no current stage available")
	ErrStageNotInProgress    = errors.New("stage is not in progress")
)

// RouteStatus represents the status of a task route
type RouteStatus string

const (
	RouteStatusPending    RouteStatus = "pending"
	RouteStatusInProgress RouteStatus = "in_progress"
	RouteStatusCompleted  RouteStatus = "completed"
	RouteStatusFailed     RouteStatus = "failed"
)

// TaskRoute is the aggregate root for tracking order execution through stages
type TaskRoute struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	RouteID          string             `bson:"routeId" json:"routeId"`
	TenantID    string `bson:"tenantId" json:"tenantId"`
	FacilityID  string `bson:"facilityId" json:"facilityId"`
	WarehouseID string `bson:"warehouseId" json:"warehouseId"`
	OrderID          string             `bson:"orderId" json:"orderId"`
	WaveID           string             `bson:"waveId" json:"waveId"`
	PathTemplateID   string             `bson:"pathTemplateId" json:"pathTemplateId"`
	PathType         ProcessPathType    `bson:"pathType" json:"pathType"`
	CurrentStageIdx  int                `bson:"currentStageIdx" json:"currentStageIdx"`
	Stages           []StageStatus      `bson:"stages" json:"stages"`
	Status           RouteStatus        `bson:"status" json:"status"`
	SpecialHandling  []string           `bson:"specialHandling" json:"specialHandling"`
	ProcessPathID    string             `bson:"processPathId,omitempty" json:"processPathId,omitempty"` // Reference to process-path-service
	CreatedAt        time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time          `bson:"updatedAt" json:"updatedAt"`
	CompletedAt      *time.Time         `bson:"completedAt,omitempty" json:"completedAt,omitempty"`
	DomainEvents     []DomainEvent      `bson:"-" json:"-"`
}

// NewTaskRoute creates a new task route from a stage template
func NewTaskRoute(orderID, waveID string, template *StageTemplate, specialHandling []string, processPathID string) *TaskRoute {
	now := time.Now()
	routeID := "RT-" + uuid.New().String()[:8]

	// Initialize stages from template
	stages := make([]StageStatus, len(template.Stages))
	for i, stageDef := range template.Stages {
		stages[i] = StageStatus{
			StageType: stageDef.StageType,
			Status:    StageStatusPending,
		}
	}

	route := &TaskRoute{
		RouteID:         routeID,
		OrderID:         orderID,
		WaveID:          waveID,
		PathTemplateID:  template.TemplateID,
		PathType:        template.PathType,
		CurrentStageIdx: 0,
		Stages:          stages,
		Status:          RouteStatusPending,
		SpecialHandling: specialHandling,
		ProcessPathID:   processPathID,
		CreatedAt:       now,
		UpdatedAt:       now,
		DomainEvents:    make([]DomainEvent, 0),
	}

	route.AddDomainEvent(&RouteCreatedEvent{
		RouteID:        routeID,
		OrderID:        orderID,
		WaveID:         waveID,
		PathTemplateID: template.TemplateID,
		PathType:       string(template.PathType),
		StageCount:     len(stages),
		CreatedAt:      now,
	})

	return route
}

// GetCurrentStage returns the current stage status
func (r *TaskRoute) GetCurrentStage() *StageStatus {
	if r.CurrentStageIdx >= len(r.Stages) {
		return nil
	}
	return &r.Stages[r.CurrentStageIdx]
}

// AssignWorkerToCurrentStage assigns a worker to the current stage
func (r *TaskRoute) AssignWorkerToCurrentStage(workerID, taskID string) error {
	if r.Status == RouteStatusCompleted {
		return ErrRouteAlreadyCompleted
	}
	if r.Status == RouteStatusFailed {
		return ErrRouteAlreadyFailed
	}

	stage := r.GetCurrentStage()
	if stage == nil {
		return ErrNoCurrentStage
	}
	if stage.Status != StageStatusPending {
		return ErrStageNotPending
	}

	now := time.Now().UnixMilli()
	stage.WorkerID = workerID
	stage.TaskID = taskID
	stage.Status = StageStatusAssigned
	r.Status = RouteStatusInProgress
	r.UpdatedAt = time.Now()

	r.AddDomainEvent(&StageAssignedEvent{
		RouteID:   r.RouteID,
		OrderID:   r.OrderID,
		StageType: string(stage.StageType),
		WorkerID:  workerID,
		TaskID:    taskID,
		Timestamp: now,
	})

	return nil
}

// StartCurrentStage marks the current stage as in progress
func (r *TaskRoute) StartCurrentStage() error {
	if r.Status == RouteStatusCompleted {
		return ErrRouteAlreadyCompleted
	}

	stage := r.GetCurrentStage()
	if stage == nil {
		return ErrNoCurrentStage
	}
	if stage.Status != StageStatusAssigned {
		return errors.New("stage must be assigned before starting")
	}

	now := time.Now().UnixMilli()
	stage.Status = StageStatusInProgress
	stage.StartedAt = &now
	r.UpdatedAt = time.Now()

	r.AddDomainEvent(&StageStartedEvent{
		RouteID:   r.RouteID,
		OrderID:   r.OrderID,
		StageType: string(stage.StageType),
		TaskID:    stage.TaskID,
		WorkerID:  stage.WorkerID,
		Timestamp: now,
	})

	return nil
}

// CompleteCurrentStage marks the current stage as completed and advances to the next
func (r *TaskRoute) CompleteCurrentStage() error {
	if r.Status == RouteStatusCompleted {
		return ErrRouteAlreadyCompleted
	}

	stage := r.GetCurrentStage()
	if stage == nil {
		return ErrNoCurrentStage
	}
	if stage.Status != StageStatusInProgress {
		return ErrStageNotInProgress
	}

	now := time.Now().UnixMilli()
	stage.Status = StageStatusCompleted
	stage.CompletedAt = &now
	r.UpdatedAt = time.Now()

	r.AddDomainEvent(&StageCompletedEvent{
		RouteID:   r.RouteID,
		OrderID:   r.OrderID,
		StageType: string(stage.StageType),
		TaskID:    stage.TaskID,
		WorkerID:  stage.WorkerID,
		Timestamp: now,
	})

	// Advance to next stage or complete route
	r.CurrentStageIdx++
	if r.CurrentStageIdx >= len(r.Stages) {
		r.completeRoute()
	}

	return nil
}

// FailCurrentStage marks the current stage as failed
func (r *TaskRoute) FailCurrentStage(errorMsg string) error {
	if r.Status == RouteStatusCompleted {
		return ErrRouteAlreadyCompleted
	}

	stage := r.GetCurrentStage()
	if stage == nil {
		return ErrNoCurrentStage
	}

	now := time.Now().UnixMilli()
	stage.Status = StageStatusFailed
	stage.Error = errorMsg
	stage.CompletedAt = &now
	r.Status = RouteStatusFailed
	r.UpdatedAt = time.Now()

	r.AddDomainEvent(&StageFailedEvent{
		RouteID:   r.RouteID,
		OrderID:   r.OrderID,
		StageType: string(stage.StageType),
		TaskID:    stage.TaskID,
		Error:     errorMsg,
		Timestamp: now,
	})

	return nil
}

// completeRoute marks the entire route as completed
func (r *TaskRoute) completeRoute() {
	now := time.Now()
	r.Status = RouteStatusCompleted
	r.CompletedAt = &now
	r.UpdatedAt = now

	r.AddDomainEvent(&RouteCompletedEvent{
		RouteID:     r.RouteID,
		OrderID:     r.OrderID,
		PathType:    string(r.PathType),
		StageCount:  len(r.Stages),
		CompletedAt: now,
	})
}

// GetProgress returns the progress as a fraction (completed/total)
func (r *TaskRoute) GetProgress() (completed int, total int) {
	total = len(r.Stages)
	for _, stage := range r.Stages {
		if stage.Status == StageStatusCompleted {
			completed++
		}
	}
	return completed, total
}

// IsCompleted checks if the route is completed
func (r *TaskRoute) IsCompleted() bool {
	return r.Status == RouteStatusCompleted
}

// IsFailed checks if the route has failed
func (r *TaskRoute) IsFailed() bool {
	return r.Status == RouteStatusFailed
}

// AddDomainEvent adds a domain event
func (r *TaskRoute) AddDomainEvent(event DomainEvent) {
	r.DomainEvents = append(r.DomainEvents, event)
}

// ClearDomainEvents clears all domain events
func (r *TaskRoute) ClearDomainEvents() {
	r.DomainEvents = make([]DomainEvent, 0)
}

// GetDomainEvents returns all domain events
func (r *TaskRoute) GetDomainEvents() []DomainEvent {
	return r.DomainEvents
}
