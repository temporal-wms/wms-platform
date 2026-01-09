package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test fixtures
func createTestPickItems() []PickItem {
	return []PickItem{
		{
			SKU:         "SKU-001",
			ProductName: "Test Product 1",
			Quantity:    5,
			PickedQty:   0,
			Location: Location{
				LocationID: "LOC-A1",
				Aisle:      "A",
				Rack:       1,
				Level:      2,
				Position:   "01",
				Zone:       "ZONE-A",
			},
			Status: "pending",
		},
		{
			SKU:         "SKU-002",
			ProductName: "Test Product 2",
			Quantity:    3,
			PickedQty:   0,
			Location: Location{
				LocationID: "LOC-A2",
				Aisle:      "A",
				Rack:       1,
				Level:      3,
				Position:   "02",
				Zone:       "ZONE-A",
			},
			Status: "pending",
		},
	}
}

// TestNewPickTask tests pick task creation
func TestNewPickTask(t *testing.T) {
	tests := []struct {
		name        string
		taskID      string
		orderID     string
		waveID      string
		routeID     string
		method      PickMethod
		items       []PickItem
		expectError bool
	}{
		{
			name:        "Valid pick task creation",
			taskID:      "TASK-001",
			orderID:     "ORD-001",
			waveID:      "WAVE-001",
			routeID:     "ROUTE-001",
			method:      PickMethodWave,
			items:       createTestPickItems(),
			expectError: false,
		},
		{
			name:        "Pick task with no items",
			taskID:      "TASK-002",
			orderID:     "ORD-002",
			waveID:      "WAVE-001",
			routeID:     "ROUTE-001",
			method:      PickMethodWave,
			items:       []PickItem{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := NewPickTask(tt.taskID, tt.orderID, tt.waveID, tt.routeID, tt.method, tt.items)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, task)
			} else {
				require.NoError(t, err)
				require.NotNil(t, task)
				assert.Equal(t, tt.taskID, task.TaskID)
				assert.Equal(t, tt.orderID, task.OrderID)
				assert.Equal(t, tt.waveID, task.WaveID)
				assert.Equal(t, PickTaskStatusPending, task.Status)
				assert.Equal(t, 8, task.TotalItems) // 5 + 3
				assert.Equal(t, 0, task.PickedItems)
				assert.NotZero(t, task.CreatedAt)

				// Check domain event
				events := task.GetDomainEvents()
				assert.Len(t, events, 1)
				event, ok := events[0].(*PickTaskCreatedEvent)
				assert.True(t, ok)
				assert.Equal(t, tt.taskID, event.TaskID)
			}
		})
	}
}

// TestPickTaskAssign tests task assignment
func TestPickTaskAssign(t *testing.T) {
	tests := []struct {
		name        string
		setupTask   func() *PickTask
		pickerID    string
		toteID      string
		expectError error
	}{
		{
			name: "Assign pending task",
			setupTask: func() *PickTask {
				task, _ := NewPickTask("TASK-001", "ORD-001", "WAVE-001", "ROUTE-001", PickMethodWave, createTestPickItems())
				return task
			},
			pickerID:    "PICKER-123",
			toteID:      "TOTE-456",
			expectError: nil,
		},
		{
			name: "Cannot assign already assigned task",
			setupTask: func() *PickTask {
				task, _ := NewPickTask("TASK-002", "ORD-002", "WAVE-001", "ROUTE-001", PickMethodWave, createTestPickItems())
				task.Assign("PICKER-111", "TOTE-222")
				return task
			},
			pickerID:    "PICKER-999",
			toteID:      "TOTE-999",
			expectError: ErrTaskAlreadyAssigned,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := tt.setupTask()
			err := task.Assign(tt.pickerID, tt.toteID)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.pickerID, task.PickerID)
				assert.Equal(t, tt.toteID, task.ToteID)
				assert.Equal(t, PickTaskStatusAssigned, task.Status)
				assert.NotNil(t, task.AssignedAt)
			}
		})
	}
}

// TestPickTaskStart tests starting a task
func TestPickTaskStart(t *testing.T) {
	tests := []struct {
		name        string
		setupTask   func() *PickTask
		expectError error
	}{
		{
			name: "Start assigned task",
			setupTask: func() *PickTask {
				task, _ := NewPickTask("TASK-001", "ORD-001", "WAVE-001", "ROUTE-001", PickMethodWave, createTestPickItems())
				task.Assign("PICKER-123", "TOTE-456")
				return task
			},
			expectError: nil,
		},
		{
			name: "Cannot start unassigned task",
			setupTask: func() *PickTask {
				task, _ := NewPickTask("TASK-002", "ORD-002", "WAVE-001", "ROUTE-001", PickMethodWave, createTestPickItems())
				return task
			},
			expectError: ErrTaskNotAssigned,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := tt.setupTask()
			err := task.Start()

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, PickTaskStatusInProgress, task.Status)
				assert.NotNil(t, task.StartedAt)
			}
		})
	}
}

// TestPickTaskConfirmPick tests confirming item picks
func TestPickTaskConfirmPick(t *testing.T) {
	tests := []struct {
		name        string
		setupTask   func() *PickTask
		sku         string
		locationID  string
		pickedQty   int
		toteID      string
		expectError bool
	}{
		{
			name: "Confirm pick successfully",
			setupTask: func() *PickTask {
				task, _ := NewPickTask("TASK-001", "ORD-001", "WAVE-001", "ROUTE-001", PickMethodWave, createTestPickItems())
				task.Assign("PICKER-123", "TOTE-456")
				task.Start()
				return task
			},
			sku:         "SKU-001",
			locationID:  "LOC-A1",
			pickedQty:   5,
			toteID:      "TOTE-456",
			expectError: false,
		},
		{
			name: "Confirm partial pick",
			setupTask: func() *PickTask {
				task, _ := NewPickTask("TASK-002", "ORD-002", "WAVE-001", "ROUTE-001", PickMethodWave, createTestPickItems())
				task.Assign("PICKER-123", "TOTE-456")
				task.Start()
				return task
			},
			sku:         "SKU-001",
			locationID:  "LOC-A1",
			pickedQty:   3, // Less than required 5
			toteID:      "TOTE-456",
			expectError: false,
		},
		{
			name: "Item not found",
			setupTask: func() *PickTask {
				task, _ := NewPickTask("TASK-003", "ORD-003", "WAVE-001", "ROUTE-001", PickMethodWave, createTestPickItems())
				task.Assign("PICKER-123", "TOTE-456")
				task.Start()
				return task
			},
			sku:         "SKU-999",
			locationID:  "LOC-A1",
			pickedQty:   5,
			toteID:      "TOTE-456",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := tt.setupTask()
			initialPicked := task.PickedItems
			err := task.ConfirmPick(tt.sku, tt.locationID, tt.pickedQty, tt.toteID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, initialPicked+tt.pickedQty, task.PickedItems)

				// Verify item status
				for _, item := range task.Items {
					if item.SKU == tt.sku && item.Location.LocationID == tt.locationID {
						assert.Equal(t, tt.pickedQty, item.PickedQty)
						if tt.pickedQty >= item.Quantity {
							assert.Equal(t, "picked", item.Status)
						} else if tt.pickedQty > 0 {
							assert.Equal(t, "short", item.Status)
						}
					}
				}
			}
		})
	}
}

// TestPickTaskAutoComplete tests automatic completion when all items are picked
func TestPickTaskAutoComplete(t *testing.T) {
	task, _ := NewPickTask("TASK-001", "ORD-001", "WAVE-001", "ROUTE-001", PickMethodWave, createTestPickItems())
	task.Assign("PICKER-123", "TOTE-456")
	task.Start()

	// Pick all items
	err := task.ConfirmPick("SKU-001", "LOC-A1", 5, "TOTE-456")
	assert.NoError(t, err)
	assert.Equal(t, PickTaskStatusInProgress, task.Status)

	// Pick last item - should auto-complete
	err = task.ConfirmPick("SKU-002", "LOC-A2", 3, "TOTE-456")
	assert.NoError(t, err)
	assert.Equal(t, PickTaskStatusCompleted, task.Status)
	assert.NotNil(t, task.CompletedAt)
}

// TestPickTaskReportException tests exception reporting
func TestPickTaskReportException(t *testing.T) {
	tests := []struct {
		name        string
		setupTask   func() *PickTask
		sku         string
		locationID  string
		reason      string
		requestedQty int
		availableQty int
		expectError bool
	}{
		{
			name: "Report exception during picking",
			setupTask: func() *PickTask {
				task, _ := NewPickTask("TASK-001", "ORD-001", "WAVE-001", "ROUTE-001", PickMethodWave, createTestPickItems())
				task.Assign("PICKER-123", "TOTE-456")
				task.Start()
				return task
			},
			sku:          "SKU-001",
			locationID:   "LOC-A1",
			reason:       "item_not_found",
			requestedQty: 5,
			availableQty: 0,
			expectError:  false,
		},
		{
			name: "Cannot report exception when not in progress",
			setupTask: func() *PickTask {
				task, _ := NewPickTask("TASK-002", "ORD-002", "WAVE-001", "ROUTE-001", PickMethodWave, createTestPickItems())
				return task
			},
			sku:          "SKU-001",
			locationID:   "LOC-A1",
			reason:       "damaged",
			requestedQty: 5,
			availableQty: 0,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := tt.setupTask()
			initialExceptions := len(task.Exceptions)
			err := task.ReportException(tt.sku, tt.locationID, tt.reason, tt.requestedQty, tt.availableQty)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, task.Exceptions, initialExceptions+1)
				assert.Equal(t, tt.sku, task.Exceptions[len(task.Exceptions)-1].SKU)
				assert.Equal(t, tt.reason, task.Exceptions[len(task.Exceptions)-1].Reason)
			}
		})
	}
}

// TestPickTaskCancel tests task cancellation
func TestPickTaskCancel(t *testing.T) {
	task, _ := NewPickTask("TASK-001", "ORD-001", "WAVE-001", "ROUTE-001", PickMethodWave, createTestPickItems())
	task.Assign("PICKER-123", "TOTE-456")
	task.Start()

	err := task.Cancel("Order cancelled")
	assert.NoError(t, err)
	assert.Equal(t, PickTaskStatusCancelled, task.Status)
}

// TestPickTaskGetProgress tests progress calculation
func TestPickTaskGetProgress(t *testing.T) {
	task, _ := NewPickTask("TASK-001", "ORD-001", "WAVE-001", "ROUTE-001", PickMethodWave, createTestPickItems())
	task.Assign("PICKER-123", "TOTE-456")
	task.Start()

	// Initially 0%
	progress := task.GetProgress()
	assert.Equal(t, 0.0, progress)

	// Pick partial
	task.ConfirmPick("SKU-001", "LOC-A1", 5, "TOTE-456")
	progress = task.GetProgress()
	assert.InDelta(t, 62.5, progress, 0.1) // 5 out of 8 items

	// Pick remaining
	task.ConfirmPick("SKU-002", "LOC-A2", 3, "TOTE-456")
	progress = task.GetProgress()
	assert.Equal(t, 100.0, progress)
}

// TestPickTaskDomainEvents tests domain event handling
func TestPickTaskDomainEvents(t *testing.T) {
	task, _ := NewPickTask("TASK-001", "ORD-001", "WAVE-001", "ROUTE-001", PickMethodWave, createTestPickItems())

	// Check initial event
	events := task.GetDomainEvents()
	assert.Len(t, events, 1)
	_, ok := events[0].(*PickTaskCreatedEvent)
	assert.True(t, ok)

	// Assign
	task.Assign("PICKER-123", "TOTE-456")
	events = task.GetDomainEvents()
	assert.Len(t, events, 2)

	// Start doesn't generate event
	task.Start()
	events = task.GetDomainEvents()
	assert.Len(t, events, 2)

	// Pick item
	task.ConfirmPick("SKU-001", "LOC-A1", 5, "TOTE-456")
	events = task.GetDomainEvents()
	assert.Len(t, events, 3)

	// Clear events
	task.ClearDomainEvents()
	events = task.GetDomainEvents()
	assert.Len(t, events, 0)
}

// BenchmarkNewPickTask benchmarks pick task creation
func BenchmarkNewPickTask(b *testing.B) {
	items := createTestPickItems()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewPickTask("TASK-001", "ORD-001", "WAVE-001", "ROUTE-001", PickMethodWave, items)
	}
}

// BenchmarkConfirmPick benchmarks pick confirmation
func BenchmarkConfirmPick(b *testing.B) {
	task, _ := NewPickTask("TASK-001", "ORD-001", "WAVE-001", "ROUTE-001", PickMethodWave, createTestPickItems())
	task.Assign("PICKER-123", "TOTE-456")
	task.Start()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Note: This will complete the task quickly, measuring first pick
		task.ConfirmPick("SKU-001", "LOC-A1", 5, "TOTE-456")
	}
}
