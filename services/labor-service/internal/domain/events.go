package domain

import "time"

// DomainEvent is the interface for all domain events
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// ShiftStartedEvent is published when a worker starts a shift
type ShiftStartedEvent struct {
	WorkerID  string    `json:"workerId"`
	ShiftID   string    `json:"shiftId"`
	Zone      string    `json:"zone"`
	ShiftType string    `json:"shiftType"`
	StartedAt time.Time `json:"startedAt"`
}

func (e *ShiftStartedEvent) EventType() string    { return "wms.labor.shift-started" }
func (e *ShiftStartedEvent) OccurredAt() time.Time { return e.StartedAt }

// ShiftEndedEvent is published when a worker ends a shift
type ShiftEndedEvent struct {
	WorkerID       string    `json:"workerId"`
	ShiftID        string    `json:"shiftId"`
	TasksCompleted int       `json:"tasksCompleted"`
	ItemsProcessed int       `json:"itemsProcessed"`
	EndedAt        time.Time `json:"endedAt"`
}

func (e *ShiftEndedEvent) EventType() string    { return "wms.labor.shift-ended" }
func (e *ShiftEndedEvent) OccurredAt() time.Time { return e.EndedAt }

// TaskAssignedEvent is published when a task is assigned to a worker
type TaskAssignedEvent struct {
	WorkerID   string    `json:"workerId"`
	TaskID     string    `json:"taskId"`
	TaskType   string    `json:"taskType"`
	AssignedAt time.Time `json:"assignedAt"`
}

func (e *TaskAssignedEvent) EventType() string    { return "wms.labor.task-assigned" }
func (e *TaskAssignedEvent) OccurredAt() time.Time { return e.AssignedAt }

// TaskCompletedEvent is published when a worker completes a task
type TaskCompletedEvent struct {
	WorkerID       string    `json:"workerId"`
	TaskID         string    `json:"taskId"`
	ItemsProcessed int       `json:"itemsProcessed"`
	CompletedAt    time.Time `json:"completedAt"`
}

func (e *TaskCompletedEvent) EventType() string    { return "wms.labor.task-completed" }
func (e *TaskCompletedEvent) OccurredAt() time.Time { return e.CompletedAt }

// PerformanceRecordedEvent is published when performance metrics are updated
type PerformanceRecordedEvent struct {
	WorkerID        string    `json:"workerId"`
	AvgTaskTime     float64   `json:"avgTaskTime"`
	AvgItemsPerHour float64   `json:"avgItemsPerHour"`
	AccuracyRate    float64   `json:"accuracyRate"`
	RecordedAt      time.Time `json:"recordedAt"`
}

func (e *PerformanceRecordedEvent) EventType() string    { return "wms.labor.performance-recorded" }
func (e *PerformanceRecordedEvent) OccurredAt() time.Time { return e.RecordedAt }
