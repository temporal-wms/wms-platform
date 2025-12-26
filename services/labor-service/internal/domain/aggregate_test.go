package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewWorker tests worker creation
func TestNewWorker(t *testing.T) {
	worker := NewWorker("WORKER-001", "EMP-123", "John Doe")

	require.NotNil(t, worker)
	assert.Equal(t, "WORKER-001", worker.WorkerID)
	assert.Equal(t, "EMP-123", worker.EmployeeID)
	assert.Equal(t, "John Doe", worker.Name)
	assert.Equal(t, WorkerStatusOffline, worker.Status)
	assert.Empty(t, worker.Skills)
	assert.NotZero(t, worker.CreatedAt)
}

// TestWorkerStartShift tests shift start
func TestWorkerStartShift(t *testing.T) {
	tests := []struct {
		name        string
		setupWorker func() *Worker
		shiftID     string
		shiftType   string
		zone        string
		expectError bool
	}{
		{
			name: "Start shift for offline worker",
			setupWorker: func() *Worker {
				return NewWorker("WORKER-001", "EMP-123", "John Doe")
			},
			shiftID:     "SHIFT-001",
			shiftType:   "morning",
			zone:        "ZONE-A",
			expectError: false,
		},
		{
			name: "Cannot start shift with active shift",
			setupWorker: func() *Worker {
				worker := NewWorker("WORKER-002", "EMP-456", "Jane Smith")
				worker.StartShift("SHIFT-001", "morning", "ZONE-A")
				return worker
			},
			shiftID:     "SHIFT-002",
			shiftType:   "afternoon",
			zone:        "ZONE-B",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker := tt.setupWorker()
			err := worker.StartShift(tt.shiftID, tt.shiftType, tt.zone)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, worker.CurrentShift)
				assert.Equal(t, tt.shiftID, worker.CurrentShift.ShiftID)
				assert.Equal(t, tt.shiftType, worker.CurrentShift.ShiftType)
				assert.Equal(t, tt.zone, worker.CurrentShift.Zone)
				assert.Equal(t, tt.zone, worker.CurrentZone)
				assert.Equal(t, WorkerStatusAvailable, worker.Status)
				assert.NotZero(t, worker.CurrentShift.StartTime)
				assert.Nil(t, worker.CurrentShift.EndTime)

				// Check domain event
				events := worker.GetDomainEvents()
				assert.GreaterOrEqual(t, len(events), 1)
				event, ok := events[len(events)-1].(*ShiftStartedEvent)
				assert.True(t, ok)
				assert.Equal(t, tt.shiftID, event.ShiftID)
			}
		})
	}
}

// TestWorkerEndShift tests shift end
func TestWorkerEndShift(t *testing.T) {
	tests := []struct {
		name        string
		setupWorker func() *Worker
		expectError error
	}{
		{
			name: "End active shift",
			setupWorker: func() *Worker {
				worker := NewWorker("WORKER-001", "EMP-123", "John Doe")
				worker.StartShift("SHIFT-001", "morning", "ZONE-A")
				return worker
			},
			expectError: nil,
		},
		{
			name: "Cannot end shift with no active shift",
			setupWorker: func() *Worker {
				return NewWorker("WORKER-002", "EMP-456", "Jane Smith")
			},
			expectError: ErrNoActiveShift,
		},
		{
			name: "Cannot end already ended shift",
			setupWorker: func() *Worker {
				worker := NewWorker("WORKER-003", "EMP-789", "Bob Johnson")
				worker.StartShift("SHIFT-001", "morning", "ZONE-A")
				worker.EndShift()
				return worker
			},
			expectError: ErrShiftAlreadyEnded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker := tt.setupWorker()
			err := worker.EndShift()

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, worker.CurrentShift.EndTime)
				assert.Equal(t, WorkerStatusOffline, worker.Status)
			}
		})
	}
}

// TestWorkerBreak tests break management
func TestWorkerBreak(t *testing.T) {
	worker := NewWorker("WORKER-001", "EMP-123", "John Doe")
	worker.StartShift("SHIFT-001", "morning", "ZONE-A")

	// Start break
	err := worker.StartBreak("lunch")
	assert.NoError(t, err)
	assert.Equal(t, WorkerStatusOnBreak, worker.Status)
	assert.Len(t, worker.CurrentShift.BreaksTaken, 1)
	assert.Equal(t, "lunch", worker.CurrentShift.BreaksTaken[0].Type)
	assert.Nil(t, worker.CurrentShift.BreaksTaken[0].EndTime)

	// End break
	err = worker.EndBreak()
	assert.NoError(t, err)
	assert.Equal(t, WorkerStatusAvailable, worker.Status)
	assert.NotNil(t, worker.CurrentShift.BreaksTaken[0].EndTime)
}

// TestWorkerBreakValidation tests break validation
func TestWorkerBreakValidation(t *testing.T) {
	tests := []struct {
		name        string
		setupWorker func() *Worker
		action      func(*Worker) error
		expectError bool
	}{
		{
			name: "Cannot start break without shift",
			setupWorker: func() *Worker {
				return NewWorker("WORKER-001", "EMP-123", "John Doe")
			},
			action: func(w *Worker) error {
				return w.StartBreak("break")
			},
			expectError: true,
		},
		{
			name: "Cannot start break while on task",
			setupWorker: func() *Worker {
				worker := NewWorker("WORKER-002", "EMP-456", "Jane Smith")
				worker.StartShift("SHIFT-001", "morning", "ZONE-A")
				worker.AssignTask("TASK-001", TaskTypePicking, 5)
				return worker
			},
			action: func(w *Worker) error {
				return w.StartBreak("break")
			},
			expectError: true,
		},
		{
			name: "Cannot end break when not on break",
			setupWorker: func() *Worker {
				worker := NewWorker("WORKER-003", "EMP-789", "Bob Johnson")
				worker.StartShift("SHIFT-001", "morning", "ZONE-A")
				return worker
			},
			action: func(w *Worker) error {
				return w.EndBreak()
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker := tt.setupWorker()
			err := tt.action(worker)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestWorkerAssignTask tests task assignment
func TestWorkerAssignTask(t *testing.T) {
	tests := []struct {
		name        string
		setupWorker func() *Worker
		taskID      string
		taskType    TaskType
		priority    int
		expectError error
	}{
		{
			name: "Assign task to available worker",
			setupWorker: func() *Worker {
				worker := NewWorker("WORKER-001", "EMP-123", "John Doe")
				worker.StartShift("SHIFT-001", "morning", "ZONE-A")
				return worker
			},
			taskID:      "TASK-001",
			taskType:    TaskTypePicking,
			priority:    5,
			expectError: nil,
		},
		{
			name: "Cannot assign task to offline worker",
			setupWorker: func() *Worker {
				return NewWorker("WORKER-002", "EMP-456", "Jane Smith")
			},
			taskID:      "TASK-002",
			taskType:    TaskTypePacking,
			priority:    3,
			expectError: ErrWorkerNotAvailable,
		},
		{
			name: "Cannot assign task to worker on break",
			setupWorker: func() *Worker {
				worker := NewWorker("WORKER-003", "EMP-789", "Bob Johnson")
				worker.StartShift("SHIFT-001", "morning", "ZONE-A")
				worker.StartBreak("lunch")
				return worker
			},
			taskID:      "TASK-003",
			taskType:    TaskTypeReceiving,
			priority:    4,
			expectError: ErrWorkerNotAvailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker := tt.setupWorker()
			err := worker.AssignTask(tt.taskID, tt.taskType, tt.priority)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, worker.CurrentTask)
				assert.Equal(t, tt.taskID, worker.CurrentTask.TaskID)
				assert.Equal(t, tt.taskType, worker.CurrentTask.TaskType)
				assert.Equal(t, tt.priority, worker.CurrentTask.Priority)
				assert.Equal(t, WorkerStatusOnTask, worker.Status)
				assert.NotZero(t, worker.CurrentTask.AssignedAt)

				// Check domain event
				events := worker.GetDomainEvents()
				assert.GreaterOrEqual(t, len(events), 2) // Shift started + Task assigned
			}
		})
	}
}

// TestWorkerTaskLifecycle tests complete task lifecycle
func TestWorkerTaskLifecycle(t *testing.T) {
	worker := NewWorker("WORKER-001", "EMP-123", "John Doe")
	worker.StartShift("SHIFT-001", "morning", "ZONE-A")

	// Assign task
	err := worker.AssignTask("TASK-001", TaskTypePicking, 5)
	assert.NoError(t, err)
	assert.Equal(t, WorkerStatusOnTask, worker.Status)
	assert.NotNil(t, worker.CurrentTask)

	// Start task
	err = worker.StartTask()
	assert.NoError(t, err)
	assert.NotNil(t, worker.CurrentTask.StartedAt)

	// Complete task
	itemsProcessed := 10
	initialTasksCompleted := worker.CurrentShift.TasksCompleted
	initialItemsProcessed := worker.CurrentShift.ItemsProcessed
	initialPerfTasks := worker.PerformanceMetrics.TotalTasksCompleted
	initialPerfItems := worker.PerformanceMetrics.TotalItemsProcessed

	err = worker.CompleteTask(itemsProcessed)
	assert.NoError(t, err)
	assert.Nil(t, worker.CurrentTask)
	assert.Equal(t, WorkerStatusAvailable, worker.Status)
	assert.Equal(t, initialTasksCompleted+1, worker.CurrentShift.TasksCompleted)
	assert.Equal(t, initialItemsProcessed+itemsProcessed, worker.CurrentShift.ItemsProcessed)
	assert.Equal(t, initialPerfTasks+1, worker.PerformanceMetrics.TotalTasksCompleted)
	assert.Equal(t, initialPerfItems+itemsProcessed, worker.PerformanceMetrics.TotalItemsProcessed)
}

// TestWorkerTaskValidation tests task validation
func TestWorkerTaskValidation(t *testing.T) {
	tests := []struct {
		name        string
		setupWorker func() *Worker
		action      func(*Worker) error
		expectError bool
	}{
		{
			name: "Cannot start task without assignment",
			setupWorker: func() *Worker {
				worker := NewWorker("WORKER-001", "EMP-123", "John Doe")
				worker.StartShift("SHIFT-001", "morning", "ZONE-A")
				return worker
			},
			action: func(w *Worker) error {
				return w.StartTask()
			},
			expectError: true,
		},
		{
			name: "Cannot complete task without assignment",
			setupWorker: func() *Worker {
				worker := NewWorker("WORKER-002", "EMP-456", "Jane Smith")
				worker.StartShift("SHIFT-001", "morning", "ZONE-A")
				return worker
			},
			action: func(w *Worker) error {
				return w.CompleteTask(10)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker := tt.setupWorker()
			err := tt.action(worker)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestWorkerSkills tests skill management
func TestWorkerSkills(t *testing.T) {
	worker := NewWorker("WORKER-001", "EMP-123", "John Doe")

	// Add skill
	worker.AddSkill(TaskTypePicking, 3, true)
	assert.Len(t, worker.Skills, 1)
	assert.Equal(t, TaskTypePicking, worker.Skills[0].Type)
	assert.Equal(t, 3, worker.Skills[0].Level)
	assert.True(t, worker.Skills[0].Certified)
	assert.NotNil(t, worker.Skills[0].CertifiedAt)

	// Check skill
	hasSkill := worker.HasSkill(TaskTypePicking, 2)
	assert.True(t, hasSkill)

	hasSkill = worker.HasSkill(TaskTypePicking, 5)
	assert.False(t, hasSkill)

	hasSkill = worker.HasSkill(TaskTypePacking, 1)
	assert.False(t, hasSkill)

	// Update skill
	worker.AddSkill(TaskTypePicking, 4, true)
	assert.Len(t, worker.Skills, 1) // Should update, not add new
	assert.Equal(t, 4, worker.Skills[0].Level)

	// Add another skill
	worker.AddSkill(TaskTypePacking, 2, false)
	assert.Len(t, worker.Skills, 2)
	assert.False(t, worker.Skills[1].Certified)
	assert.Nil(t, worker.Skills[1].CertifiedAt)
}

// TestWorkerTaskTypes tests all task types
func TestWorkerTaskTypes(t *testing.T) {
	taskTypes := []TaskType{
		TaskTypePicking,
		TaskTypePacking,
		TaskTypeReceiving,
		TaskTypeConsolidation,
		TaskTypeReplenishment,
	}

	for _, taskType := range taskTypes {
		t.Run(string(taskType), func(t *testing.T) {
			worker := NewWorker("WORKER-001", "EMP-123", "John Doe")
			worker.StartShift("SHIFT-001", "morning", "ZONE-A")

			err := worker.AssignTask("TASK-001", taskType, 5)
			assert.NoError(t, err)
			assert.Equal(t, taskType, worker.CurrentTask.TaskType)
		})
	}
}

// TestWorkerUpdateZone tests zone updates
func TestWorkerUpdateZone(t *testing.T) {
	worker := NewWorker("WORKER-001", "EMP-123", "John Doe")
	worker.UpdateZone("ZONE-A")
	assert.Equal(t, "ZONE-A", worker.CurrentZone)

	worker.UpdateZone("ZONE-B")
	assert.Equal(t, "ZONE-B", worker.CurrentZone)
}

// TestWorkerPerformanceMetrics tests performance metrics
func TestWorkerPerformanceMetrics(t *testing.T) {
	worker := NewWorker("WORKER-001", "EMP-123", "John Doe")

	worker.UpdatePerformanceMetrics(5.5, 120.0, 98.5)
	assert.Equal(t, 5.5, worker.PerformanceMetrics.AverageTaskTime)
	assert.Equal(t, 120.0, worker.PerformanceMetrics.AverageItemsPerHour)
	assert.Equal(t, 98.5, worker.PerformanceMetrics.AccuracyRate)
	assert.NotZero(t, worker.PerformanceMetrics.LastUpdated)

	// Check domain event
	events := worker.GetDomainEvents()
	assert.GreaterOrEqual(t, len(events), 1)
	event, ok := events[len(events)-1].(*PerformanceRecordedEvent)
	assert.True(t, ok)
	assert.Equal(t, 5.5, event.AvgTaskTime)
}

// TestWorkerCompleteWorkflow tests complete worker workflow
func TestWorkerCompleteWorkflow(t *testing.T) {
	worker := NewWorker("WORKER-001", "EMP-123", "John Doe")
	assert.Equal(t, WorkerStatusOffline, worker.Status)

	// Start shift
	err := worker.StartShift("SHIFT-001", "morning", "ZONE-A")
	assert.NoError(t, err)
	assert.Equal(t, WorkerStatusAvailable, worker.Status)

	// Assign and complete first task
	err = worker.AssignTask("TASK-001", TaskTypePicking, 5)
	assert.NoError(t, err)
	worker.StartTask()
	err = worker.CompleteTask(10)
	assert.NoError(t, err)
	assert.Equal(t, WorkerStatusAvailable, worker.Status)
	assert.Equal(t, 1, worker.CurrentShift.TasksCompleted)

	// Take break
	err = worker.StartBreak("lunch")
	assert.NoError(t, err)
	assert.Equal(t, WorkerStatusOnBreak, worker.Status)

	err = worker.EndBreak()
	assert.NoError(t, err)
	assert.Equal(t, WorkerStatusAvailable, worker.Status)

	// Assign and complete second task
	err = worker.AssignTask("TASK-002", TaskTypePacking, 3)
	assert.NoError(t, err)
	worker.StartTask()
	err = worker.CompleteTask(5)
	assert.NoError(t, err)
	assert.Equal(t, 2, worker.CurrentShift.TasksCompleted)
	assert.Equal(t, 15, worker.CurrentShift.ItemsProcessed)

	// End shift
	err = worker.EndShift()
	assert.NoError(t, err)
	assert.Equal(t, WorkerStatusOffline, worker.Status)

	// Verify performance metrics updated
	assert.Equal(t, 2, worker.PerformanceMetrics.TotalTasksCompleted)
	assert.Equal(t, 15, worker.PerformanceMetrics.TotalItemsProcessed)
}

// TestWorkerDomainEvents tests domain event handling
func TestWorkerDomainEvents(t *testing.T) {
	worker := NewWorker("WORKER-001", "EMP-123", "John Doe")

	// Start shift
	worker.StartShift("SHIFT-001", "morning", "ZONE-A")
	events := worker.GetDomainEvents()
	assert.Len(t, events, 1)
	_, ok := events[0].(*ShiftStartedEvent)
	assert.True(t, ok)

	// Assign task
	worker.AssignTask("TASK-001", TaskTypePicking, 5)
	events = worker.GetDomainEvents()
	assert.Len(t, events, 2)

	// Complete task
	worker.StartTask()
	worker.CompleteTask(10)
	events = worker.GetDomainEvents()
	assert.Len(t, events, 3)

	// Clear events
	worker.ClearDomainEvents()
	events = worker.GetDomainEvents()
	assert.Len(t, events, 0)
}

// BenchmarkNewWorker benchmarks worker creation
func BenchmarkNewWorker(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewWorker("WORKER-001", "EMP-123", "John Doe")
	}
}

// BenchmarkAssignTask benchmarks task assignment
func BenchmarkAssignTask(b *testing.B) {
	worker := NewWorker("WORKER-001", "EMP-123", "John Doe")
	worker.StartShift("SHIFT-001", "morning", "ZONE-A")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset for benchmark
		worker.Status = WorkerStatusAvailable
		worker.CurrentTask = nil
		worker.AssignTask("TASK-001", TaskTypePicking, 5)
	}
}
