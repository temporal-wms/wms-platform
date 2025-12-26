package application

import "github.com/wms-platform/labor-service/internal/domain"

// CreateWorkerCommand creates a new worker
type CreateWorkerCommand struct {
	WorkerID   string
	EmployeeID string
	Name       string
}

// GetWorkerQuery retrieves a worker by ID
type GetWorkerQuery struct {
	WorkerID string
}

// StartShiftCommand starts a shift
type StartShiftCommand struct {
	WorkerID  string
	ShiftID   string
	ShiftType string
	Zone      string
}

// EndShiftCommand ends a shift
type EndShiftCommand struct {
	WorkerID string
}

// StartBreakCommand starts a break
type StartBreakCommand struct {
	WorkerID  string
	BreakType string
}

// EndBreakCommand ends a break
type EndBreakCommand struct {
	WorkerID string
}

// AssignTaskCommand assigns a task to a worker
type AssignTaskCommand struct {
	WorkerID string
	TaskID   string
	TaskType domain.TaskType
	Priority int
}

// StartTaskCommand starts the current task
type StartTaskCommand struct {
	WorkerID string
}

// CompleteTaskCommand completes the current task
type CompleteTaskCommand struct {
	WorkerID       string
	ItemsProcessed int
}

// AddSkillCommand adds a skill to a worker
type AddSkillCommand struct {
	WorkerID  string
	TaskType  domain.TaskType
	Level     int
	Certified bool
}

// GetByStatusQuery retrieves workers by status
type GetByStatusQuery struct {
	Status domain.WorkerStatus
}

// GetByZoneQuery retrieves workers by zone
type GetByZoneQuery struct {
	Zone string
}

// GetAvailableQuery retrieves available workers
type GetAvailableQuery struct {
	Zone string // Optional filter by zone
}

// ListWorkersQuery retrieves all workers
type ListWorkersQuery struct {
	Limit  int
	Offset int
}
