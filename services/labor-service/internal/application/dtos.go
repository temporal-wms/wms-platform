package application

import "time"

// WorkerDTO represents a worker in responses
type WorkerDTO struct {
	WorkerID           string                 `json:"workerId"`
	EmployeeID         string                 `json:"employeeId"`
	Name               string                 `json:"name"`
	Status             string                 `json:"status"`
	CurrentZone        string                 `json:"currentZone"`
	Skills             []SkillDTO             `json:"skills"`
	CurrentShift       *ShiftDTO              `json:"currentShift,omitempty"`
	CurrentTask        *TaskAssignmentDTO     `json:"currentTask,omitempty"`
	PerformanceMetrics PerformanceMetricsDTO  `json:"performanceMetrics"`
	CreatedAt          time.Time              `json:"createdAt"`
	UpdatedAt          time.Time              `json:"updatedAt"`
}

// SkillDTO represents a worker skill
type SkillDTO struct {
	Type        string     `json:"type"`
	Level       int        `json:"level"`
	Certified   bool       `json:"certified"`
	CertifiedAt *time.Time `json:"certifiedAt,omitempty"`
}

// ShiftDTO represents a worker shift
type ShiftDTO struct {
	ShiftID        string     `json:"shiftId"`
	ShiftType      string     `json:"shiftType"`
	Zone           string     `json:"zone"`
	StartTime      time.Time  `json:"startTime"`
	EndTime        *time.Time `json:"endTime,omitempty"`
	BreaksTaken    []BreakDTO `json:"breaksTaken"`
	TasksCompleted int        `json:"tasksCompleted"`
	ItemsProcessed int        `json:"itemsProcessed"`
}

// BreakDTO represents a break
type BreakDTO struct {
	Type      string     `json:"type"`
	StartTime time.Time  `json:"startTime"`
	EndTime   *time.Time `json:"endTime,omitempty"`
}

// TaskAssignmentDTO represents a task assignment
type TaskAssignmentDTO struct {
	TaskID      string     `json:"taskId"`
	TaskType    string     `json:"taskType"`
	Priority    int        `json:"priority"`
	AssignedAt  time.Time  `json:"assignedAt"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// PerformanceMetricsDTO represents performance metrics
type PerformanceMetricsDTO struct {
	TotalTasksCompleted  int       `json:"totalTasksCompleted"`
	TotalItemsProcessed  int       `json:"totalItemsProcessed"`
	AverageTaskTime      float64   `json:"averageTaskTime"`
	AverageItemsPerHour  float64   `json:"averageItemsPerHour"`
	AccuracyRate         float64   `json:"accuracyRate"`
	LastUpdated          time.Time `json:"lastUpdated"`
}
