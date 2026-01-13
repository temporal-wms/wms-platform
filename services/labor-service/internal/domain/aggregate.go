package domain

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Errors
var (
	ErrWorkerNotAvailable = errors.New("worker is not available")
	ErrShiftAlreadyEnded  = errors.New("shift has already ended")
	ErrNoActiveShift      = errors.New("no active shift")
)

// WorkerStatus represents the status of a worker
type WorkerStatus string

const (
	WorkerStatusAvailable   WorkerStatus = "available"
	WorkerStatusOnTask      WorkerStatus = "on_task"
	WorkerStatusOnBreak     WorkerStatus = "on_break"
	WorkerStatusOffline     WorkerStatus = "offline"
)

// TaskType represents the type of warehouse task
type TaskType string

const (
	TaskTypePicking       TaskType = "picking"
	TaskTypePacking       TaskType = "packing"
	TaskTypeReceiving     TaskType = "receiving"
	TaskTypeConsolidation TaskType = "consolidation"
	TaskTypeReplenishment TaskType = "replenishment"
	TaskTypeWalling       TaskType = "walling"
)

// Worker is the aggregate root for the Labor bounded context
type Worker struct {
	ID                primitive.ObjectID  `bson:"_id,omitempty"`
	WorkerID          string              `bson:"workerId"`
	TenantID    string `bson:"tenantId" json:"tenantId"`
	FacilityID  string `bson:"facilityId" json:"facilityId"`
	WarehouseID string `bson:"warehouseId" json:"warehouseId"`
	EmployeeID        string              `bson:"employeeId"`
	Name              string              `bson:"name"`
	Status            WorkerStatus        `bson:"status"`
	CurrentZone       string              `bson:"currentZone"`
	Skills            []Skill             `bson:"skills"`
	CurrentShift      *Shift              `bson:"currentShift,omitempty"`
	CurrentTask       *TaskAssignment     `bson:"currentTask,omitempty"`
	PerformanceMetrics PerformanceMetrics `bson:"performanceMetrics"`
	CreatedAt         time.Time           `bson:"createdAt"`
	UpdatedAt         time.Time           `bson:"updatedAt"`
	DomainEvents      []DomainEvent       `bson:"-"`
}

// Skill represents a worker's skill
type Skill struct {
	Type        TaskType `bson:"type"`
	Level       int      `bson:"level"` // 1-5
	Certified   bool     `bson:"certified"`
	CertifiedAt *time.Time `bson:"certifiedAt,omitempty"`
}

// Shift represents a work shift
type Shift struct {
	ShiftID     string     `bson:"shiftId"`
	ShiftType   string     `bson:"shiftType"` // morning, afternoon, night
	Zone        string     `bson:"zone"`
	StartTime   time.Time  `bson:"startTime"`
	EndTime     *time.Time `bson:"endTime,omitempty"`
	BreaksTaken []Break    `bson:"breaksTaken"`
	TasksCompleted int     `bson:"tasksCompleted"`
	ItemsProcessed int     `bson:"itemsProcessed"`
}

// Break represents a break taken
type Break struct {
	StartTime time.Time  `bson:"startTime"`
	EndTime   *time.Time `bson:"endTime,omitempty"`
	Type      string     `bson:"type"` // break, lunch
}

// TaskAssignment represents an assigned task
type TaskAssignment struct {
	TaskID     string    `bson:"taskId"`
	TaskType   TaskType  `bson:"taskType"`
	Priority   int       `bson:"priority"`
	AssignedAt time.Time `bson:"assignedAt"`
	StartedAt  *time.Time `bson:"startedAt,omitempty"`
	CompletedAt *time.Time `bson:"completedAt,omitempty"`
}

// PerformanceMetrics represents worker performance
type PerformanceMetrics struct {
	TotalTasksCompleted   int       `bson:"totalTasksCompleted"`
	TotalItemsProcessed   int       `bson:"totalItemsProcessed"`
	AverageTaskTime       float64   `bson:"averageTaskTime"` // in minutes
	AverageItemsPerHour   float64   `bson:"averageItemsPerHour"`
	AccuracyRate          float64   `bson:"accuracyRate"` // percentage
	LastUpdated           time.Time `bson:"lastUpdated"`
}

// NewWorker creates a new Worker aggregate
func NewWorker(workerID, employeeID, name string) *Worker {
	now := time.Now()
	return &Worker{
		WorkerID:   workerID,
		EmployeeID: employeeID,
		Name:       name,
		Status:     WorkerStatusOffline,
		Skills:     make([]Skill, 0),
		PerformanceMetrics: PerformanceMetrics{LastUpdated: now},
		CreatedAt:  now,
		UpdatedAt:  now,
		DomainEvents: make([]DomainEvent, 0),
	}
}

// StartShift starts a new shift
func (w *Worker) StartShift(shiftID, shiftType, zone string) error {
	if w.CurrentShift != nil && w.CurrentShift.EndTime == nil {
		return errors.New("worker already has an active shift")
	}

	now := time.Now()
	w.CurrentShift = &Shift{
		ShiftID:   shiftID,
		ShiftType: shiftType,
		Zone:      zone,
		StartTime: now,
		BreaksTaken: make([]Break, 0),
	}
	w.CurrentZone = zone
	w.Status = WorkerStatusAvailable
	w.UpdatedAt = now

	w.AddDomainEvent(&ShiftStartedEvent{
		WorkerID:  w.WorkerID,
		ShiftID:   shiftID,
		Zone:      zone,
		ShiftType: shiftType,
		StartedAt: now,
	})

	return nil
}

// EndShift ends the current shift
func (w *Worker) EndShift() error {
	if w.CurrentShift == nil {
		return ErrNoActiveShift
	}
	if w.CurrentShift.EndTime != nil {
		return ErrShiftAlreadyEnded
	}

	now := time.Now()
	w.CurrentShift.EndTime = &now
	w.Status = WorkerStatusOffline
	w.UpdatedAt = now

	w.AddDomainEvent(&ShiftEndedEvent{
		WorkerID:       w.WorkerID,
		ShiftID:        w.CurrentShift.ShiftID,
		TasksCompleted: w.CurrentShift.TasksCompleted,
		ItemsProcessed: w.CurrentShift.ItemsProcessed,
		EndedAt:        now,
	})

	return nil
}

// StartBreak starts a break
func (w *Worker) StartBreak(breakType string) error {
	if w.CurrentShift == nil {
		return ErrNoActiveShift
	}
	if w.Status == WorkerStatusOnTask {
		return errors.New("cannot start break while on task")
	}

	now := time.Now()
	w.CurrentShift.BreaksTaken = append(w.CurrentShift.BreaksTaken, Break{
		StartTime: now,
		Type:      breakType,
	})
	w.Status = WorkerStatusOnBreak
	w.UpdatedAt = now

	return nil
}

// EndBreak ends a break
func (w *Worker) EndBreak() error {
	if w.Status != WorkerStatusOnBreak {
		return errors.New("worker is not on break")
	}

	now := time.Now()
	if len(w.CurrentShift.BreaksTaken) > 0 {
		lastBreak := &w.CurrentShift.BreaksTaken[len(w.CurrentShift.BreaksTaken)-1]
		lastBreak.EndTime = &now
	}
	w.Status = WorkerStatusAvailable
	w.UpdatedAt = now

	return nil
}

// AssignTask assigns a task to the worker
func (w *Worker) AssignTask(taskID string, taskType TaskType, priority int) error {
	if w.Status != WorkerStatusAvailable {
		return ErrWorkerNotAvailable
	}

	now := time.Now()
	w.CurrentTask = &TaskAssignment{
		TaskID:     taskID,
		TaskType:   taskType,
		Priority:   priority,
		AssignedAt: now,
	}
	w.Status = WorkerStatusOnTask
	w.UpdatedAt = now

	w.AddDomainEvent(&TaskAssignedEvent{
		WorkerID:   w.WorkerID,
		TaskID:     taskID,
		TaskType:   string(taskType),
		AssignedAt: now,
	})

	return nil
}

// StartTask marks the current task as started
func (w *Worker) StartTask() error {
	if w.CurrentTask == nil {
		return errors.New("no task assigned")
	}

	now := time.Now()
	w.CurrentTask.StartedAt = &now
	w.UpdatedAt = now

	return nil
}

// CompleteTask completes the current task
func (w *Worker) CompleteTask(itemsProcessed int) error {
	if w.CurrentTask == nil {
		return errors.New("no task assigned")
	}

	now := time.Now()
	w.CurrentTask.CompletedAt = &now

	// Update shift metrics
	if w.CurrentShift != nil {
		w.CurrentShift.TasksCompleted++
		w.CurrentShift.ItemsProcessed += itemsProcessed
	}

	// Update performance metrics
	w.PerformanceMetrics.TotalTasksCompleted++
	w.PerformanceMetrics.TotalItemsProcessed += itemsProcessed
	w.PerformanceMetrics.LastUpdated = now

	taskID := w.CurrentTask.TaskID
	w.CurrentTask = nil
	w.Status = WorkerStatusAvailable
	w.UpdatedAt = now

	w.AddDomainEvent(&TaskCompletedEvent{
		WorkerID:       w.WorkerID,
		TaskID:         taskID,
		ItemsProcessed: itemsProcessed,
		CompletedAt:    now,
	})

	return nil
}

// UpdateZone updates the worker's current zone
func (w *Worker) UpdateZone(zone string) {
	w.CurrentZone = zone
	w.UpdatedAt = time.Now()
}

// AddSkill adds a skill to the worker
func (w *Worker) AddSkill(taskType TaskType, level int, certified bool) {
	now := time.Now()
	skill := Skill{
		Type:      taskType,
		Level:     level,
		Certified: certified,
	}
	if certified {
		skill.CertifiedAt = &now
	}

	// Update existing or add new
	found := false
	for i := range w.Skills {
		if w.Skills[i].Type == taskType {
			w.Skills[i] = skill
			found = true
			break
		}
	}
	if !found {
		w.Skills = append(w.Skills, skill)
	}
	w.UpdatedAt = now
}

// HasSkill checks if worker has a skill
func (w *Worker) HasSkill(taskType TaskType, minLevel int) bool {
	for _, skill := range w.Skills {
		if skill.Type == taskType && skill.Level >= minLevel {
			return true
		}
	}
	return false
}

// UpdatePerformanceMetrics updates performance metrics
func (w *Worker) UpdatePerformanceMetrics(avgTaskTime, avgItemsPerHour, accuracyRate float64) {
	w.PerformanceMetrics.AverageTaskTime = avgTaskTime
	w.PerformanceMetrics.AverageItemsPerHour = avgItemsPerHour
	w.PerformanceMetrics.AccuracyRate = accuracyRate
	w.PerformanceMetrics.LastUpdated = time.Now()
	w.UpdatedAt = time.Now()

	w.AddDomainEvent(&PerformanceRecordedEvent{
		WorkerID:        w.WorkerID,
		AvgTaskTime:     avgTaskTime,
		AvgItemsPerHour: avgItemsPerHour,
		AccuracyRate:    accuracyRate,
		RecordedAt:      time.Now(),
	})
}

// AddDomainEvent adds a domain event
func (w *Worker) AddDomainEvent(event DomainEvent) {
	w.DomainEvents = append(w.DomainEvents, event)
}

// ClearDomainEvents clears all domain events
func (w *Worker) ClearDomainEvents() {
	w.DomainEvents = make([]DomainEvent, 0)
}

// GetDomainEvents returns all domain events
func (w *Worker) GetDomainEvents() []DomainEvent {
	return w.DomainEvents
}
