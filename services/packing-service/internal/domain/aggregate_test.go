package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test fixtures
func createTestPackItems() []PackItem {
	return []PackItem{
		{
			SKU:         "SKU-001",
			ProductName: "Test Product 1",
			Quantity:    2,
			Weight:      0.5,
			Fragile:     false,
			Verified:    false,
		},
		{
			SKU:         "SKU-002",
			ProductName: "Test Product 2",
			Quantity:    1,
			Weight:      0.3,
			Fragile:     true,
			Verified:    false,
		},
	}
}

func createTestDimensions() Dimensions {
	return Dimensions{
		Length: 30,
		Width:  20,
		Height: 10,
	}
}

func createTestShippingLabel() ShippingLabel {
	return ShippingLabel{
		TrackingNumber: "1Z999AA10123456784",
		Carrier:        "UPS",
		ServiceType:    "GROUND",
		LabelURL:       "https://example.com/label.pdf",
		LabelData:      "base64encodeddata==",
	}
}

// TestNewPackTask tests pack task creation
func TestNewPackTask(t *testing.T) {
	tests := []struct {
		name        string
		taskID      string
		orderID     string
		waveID      string
		items       []PackItem
		expectError error
	}{
		{
			name:        "Valid pack task creation",
			taskID:      "PACK-001",
			orderID:     "ORD-001",
			waveID:      "WAVE-001",
			items:       createTestPackItems(),
			expectError: nil,
		},
		{
			name:        "Cannot create with no items",
			taskID:      "PACK-002",
			orderID:     "ORD-002",
			waveID:      "WAVE-001",
			items:       []PackItem{},
			expectError: ErrNoItemsToPack,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := NewPackTask(tt.taskID, tt.orderID, tt.waveID, tt.items)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
				assert.Nil(t, task)
			} else {
				require.NoError(t, err)
				require.NotNil(t, task)
				assert.Equal(t, tt.taskID, task.TaskID)
				assert.Equal(t, tt.orderID, task.OrderID)
				assert.Equal(t, tt.waveID, task.WaveID)
				assert.Equal(t, PackTaskStatusPending, task.Status)
				assert.Equal(t, len(tt.items), len(task.Items))
				assert.NotZero(t, task.CreatedAt)
				assert.NotEmpty(t, task.Package.SuggestedType)

				// Check domain event
				events := task.GetDomainEvents()
				assert.Len(t, events, 1)
				event, ok := events[0].(*PackTaskCreatedEvent)
				assert.True(t, ok)
				assert.Equal(t, tt.taskID, event.TaskID)
			}
		})
	}
}

// TestPackTaskAssign tests task assignment
func TestPackTaskAssign(t *testing.T) {
	tests := []struct {
		name        string
		setupTask   func() *PackTask
		packerID    string
		station     string
		expectError error
	}{
		{
			name: "Assign pending task",
			setupTask: func() *PackTask {
				task, _ := NewPackTask("PACK-001", "ORD-001", "WAVE-001", createTestPackItems())
				return task
			},
			packerID:    "PACKER-123",
			station:     "STATION-A1",
			expectError: nil,
		},
		{
			name: "Cannot assign completed task",
			setupTask: func() *PackTask {
				task, _ := NewPackTask("PACK-002", "ORD-002", "WAVE-001", createTestPackItems())
				task.Status = PackTaskStatusCompleted
				return task
			},
			packerID:    "PACKER-456",
			station:     "STATION-A2",
			expectError: ErrPackTaskCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := tt.setupTask()
			err := task.Assign(tt.packerID, tt.station)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.packerID, task.PackerID)
				assert.Equal(t, tt.station, task.Station)
			}
		})
	}
}

// TestPackTaskStart tests starting a task
func TestPackTaskStart(t *testing.T) {
	tests := []struct {
		name        string
		setupTask   func() *PackTask
		expectError bool
	}{
		{
			name: "Start pending task",
			setupTask: func() *PackTask {
				task, _ := NewPackTask("PACK-001", "ORD-001", "WAVE-001", createTestPackItems())
				task.Assign("PACKER-123", "STATION-A1")
				return task
			},
			expectError: false,
		},
		{
			name: "Cannot start already started task",
			setupTask: func() *PackTask {
				task, _ := NewPackTask("PACK-002", "ORD-002", "WAVE-001", createTestPackItems())
				task.Assign("PACKER-123", "STATION-A1")
				task.Start()
				return task
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := tt.setupTask()
			err := task.Start()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, PackTaskStatusInProgress, task.Status)
				assert.NotNil(t, task.StartedAt)
			}
		})
	}
}

// TestPackTaskVerifyItem tests item verification
func TestPackTaskVerifyItem(t *testing.T) {
	tests := []struct {
		name        string
		setupTask   func() *PackTask
		sku         string
		expectError bool
	}{
		{
			name: "Verify existing item",
			setupTask: func() *PackTask {
				task, _ := NewPackTask("PACK-001", "ORD-001", "WAVE-001", createTestPackItems())
				task.Assign("PACKER-123", "STATION-A1")
				task.Start()
				return task
			},
			sku:         "SKU-001",
			expectError: false,
		},
		{
			name: "Cannot verify non-existent item",
			setupTask: func() *PackTask {
				task, _ := NewPackTask("PACK-002", "ORD-002", "WAVE-001", createTestPackItems())
				task.Assign("PACKER-123", "STATION-A1")
				task.Start()
				return task
			},
			sku:         "SKU-999",
			expectError: true,
		},
		{
			name: "Cannot verify item in completed task",
			setupTask: func() *PackTask {
				task, _ := NewPackTask("PACK-003", "ORD-003", "WAVE-001", createTestPackItems())
				task.Status = PackTaskStatusCompleted
				return task
			},
			sku:         "SKU-001",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := tt.setupTask()
			err := task.VerifyItem(tt.sku)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify item is marked as verified
				for _, item := range task.Items {
					if item.SKU == tt.sku {
						assert.True(t, item.Verified)
					}
				}
			}
		})
	}
}

// TestPackTaskSelectPackaging tests packaging selection
func TestPackTaskSelectPackaging(t *testing.T) {
	tests := []struct {
		name         string
		setupTask    func() *PackTask
		packageType  PackageType
		dimensions   Dimensions
		materials    []string
		expectError  error
	}{
		{
			name: "Select box packaging",
			setupTask: func() *PackTask {
				task, _ := NewPackTask("PACK-001", "ORD-001", "WAVE-001", createTestPackItems())
				task.Assign("PACKER-123", "STATION-A1")
				task.Start()
				return task
			},
			packageType: PackageTypeBox,
			dimensions:  createTestDimensions(),
			materials:   []string{"bubble_wrap", "tape"},
			expectError: nil,
		},
		{
			name: "Select envelope packaging",
			setupTask: func() *PackTask {
				task, _ := NewPackTask("PACK-002", "ORD-002", "WAVE-001", createTestPackItems())
				task.Assign("PACKER-123", "STATION-A1")
				task.Start()
				return task
			},
			packageType: PackageTypeEnvelope,
			dimensions:  Dimensions{Length: 25, Width: 18, Height: 2},
			materials:   []string{"paper"},
			expectError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := tt.setupTask()
			err := task.SelectPackaging(tt.packageType, tt.dimensions, tt.materials)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.packageType, task.Package.Type)
				assert.Equal(t, tt.dimensions, task.Package.Dimensions)
				assert.Equal(t, tt.materials, task.Package.Materials)
				assert.NotEmpty(t, task.Package.PackageID)
				assert.Greater(t, task.Package.TotalWeight, 0.0)
			}
		})
	}
}

// TestPackTaskSealPackage tests package sealing
func TestPackTaskSealPackage(t *testing.T) {
	tests := []struct {
		name        string
		setupTask   func() *PackTask
		expectError bool
	}{
		{
			name: "Seal package with all items verified",
			setupTask: func() *PackTask {
				task, _ := NewPackTask("PACK-001", "ORD-001", "WAVE-001", createTestPackItems())
				task.Assign("PACKER-123", "STATION-A1")
				task.Start()
				task.SelectPackaging(PackageTypeBox, createTestDimensions(), []string{"tape"})
				task.VerifyItem("SKU-001")
				task.VerifyItem("SKU-002")
				return task
			},
			expectError: false,
		},
		{
			name: "Cannot seal with unverified items",
			setupTask: func() *PackTask {
				task, _ := NewPackTask("PACK-002", "ORD-002", "WAVE-001", createTestPackItems())
				task.Assign("PACKER-123", "STATION-A1")
				task.Start()
				task.SelectPackaging(PackageTypeBox, createTestDimensions(), []string{"tape"})
				// Not verifying items
				return task
			},
			expectError: true,
		},
		{
			name: "Cannot seal already sealed package",
			setupTask: func() *PackTask {
				task, _ := NewPackTask("PACK-003", "ORD-003", "WAVE-001", createTestPackItems())
				task.Assign("PACKER-123", "STATION-A1")
				task.Start()
				task.SelectPackaging(PackageTypeBox, createTestDimensions(), []string{"tape"})
				task.VerifyItem("SKU-001")
				task.VerifyItem("SKU-002")
				task.SealPackage()
				return task
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := tt.setupTask()
			err := task.SealPackage()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, task.Package.Sealed)
				assert.NotNil(t, task.Package.SealedAt)
				assert.Equal(t, PackTaskStatusPacked, task.Status)
				assert.NotNil(t, task.PackedAt)
			}
		})
	}
}

// TestPackTaskApplyLabel tests label application
func TestPackTaskApplyLabel(t *testing.T) {
	tests := []struct {
		name        string
		setupTask   func() *PackTask
		label       ShippingLabel
		expectError bool
	}{
		{
			name: "Apply label to sealed package",
			setupTask: func() *PackTask {
				task, _ := NewPackTask("PACK-001", "ORD-001", "WAVE-001", createTestPackItems())
				task.Assign("PACKER-123", "STATION-A1")
				task.Start()
				task.SelectPackaging(PackageTypeBox, createTestDimensions(), []string{"tape"})
				task.VerifyItem("SKU-001")
				task.VerifyItem("SKU-002")
				task.SealPackage()
				return task
			},
			label:       createTestShippingLabel(),
			expectError: false,
		},
		{
			name: "Cannot apply label to unsealed package",
			setupTask: func() *PackTask {
				task, _ := NewPackTask("PACK-002", "ORD-002", "WAVE-001", createTestPackItems())
				task.Assign("PACKER-123", "STATION-A1")
				task.Start()
				task.SelectPackaging(PackageTypeBox, createTestDimensions(), []string{"tape"})
				// Not sealing package
				return task
			},
			label:       createTestShippingLabel(),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := tt.setupTask()
			err := task.ApplyLabel(tt.label)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, task.ShippingLabel)
				assert.Equal(t, tt.label.TrackingNumber, task.ShippingLabel.TrackingNumber)
				assert.Equal(t, tt.label.Carrier, task.ShippingLabel.Carrier)
				assert.Equal(t, PackTaskStatusLabeled, task.Status)
				assert.NotNil(t, task.LabeledAt)
				assert.NotNil(t, task.ShippingLabel.AppliedAt)
			}
		})
	}
}

// TestPackTaskComplete tests task completion
func TestPackTaskComplete(t *testing.T) {
	tests := []struct {
		name        string
		setupTask   func() *PackTask
		expectError bool
	}{
		{
			name: "Complete labeled task",
			setupTask: func() *PackTask {
				task, _ := NewPackTask("PACK-001", "ORD-001", "WAVE-001", createTestPackItems())
				task.Assign("PACKER-123", "STATION-A1")
				task.Start()
				task.SelectPackaging(PackageTypeBox, createTestDimensions(), []string{"tape"})
				task.VerifyItem("SKU-001")
				task.VerifyItem("SKU-002")
				task.SealPackage()
				task.ApplyLabel(createTestShippingLabel())
				return task
			},
			expectError: false,
		},
		{
			name: "Cannot complete without label",
			setupTask: func() *PackTask {
				task, _ := NewPackTask("PACK-002", "ORD-002", "WAVE-001", createTestPackItems())
				task.Assign("PACKER-123", "STATION-A1")
				task.Start()
				task.SelectPackaging(PackageTypeBox, createTestDimensions(), []string{"tape"})
				task.VerifyItem("SKU-001")
				task.VerifyItem("SKU-002")
				task.SealPackage()
				// Not applying label
				return task
			},
			expectError: true,
		},
		{
			name: "Cannot complete already completed task",
			setupTask: func() *PackTask {
				task, _ := NewPackTask("PACK-003", "ORD-003", "WAVE-001", createTestPackItems())
				task.Assign("PACKER-123", "STATION-A1")
				task.Start()
				task.SelectPackaging(PackageTypeBox, createTestDimensions(), []string{"tape"})
				task.VerifyItem("SKU-001")
				task.VerifyItem("SKU-002")
				task.SealPackage()
				task.ApplyLabel(createTestShippingLabel())
				task.Complete()
				return task
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := tt.setupTask()
			err := task.Complete()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, PackTaskStatusCompleted, task.Status)
				assert.NotNil(t, task.CompletedAt)
			}
		})
	}
}

// TestPackTaskCancel tests task cancellation
func TestPackTaskCancel(t *testing.T) {
	tests := []struct {
		name        string
		setupTask   func() *PackTask
		reason      string
		expectError error
	}{
		{
			name: "Cancel pending task",
			setupTask: func() *PackTask {
				task, _ := NewPackTask("PACK-001", "ORD-001", "WAVE-001", createTestPackItems())
				return task
			},
			reason:      "Order cancelled",
			expectError: nil,
		},
		{
			name: "Cancel in-progress task",
			setupTask: func() *PackTask {
				task, _ := NewPackTask("PACK-002", "ORD-002", "WAVE-001", createTestPackItems())
				task.Assign("PACKER-123", "STATION-A1")
				task.Start()
				return task
			},
			reason:      "Incorrect items",
			expectError: nil,
		},
		{
			name: "Cannot cancel completed task",
			setupTask: func() *PackTask {
				task, _ := NewPackTask("PACK-003", "ORD-003", "WAVE-001", createTestPackItems())
				task.Status = PackTaskStatusCompleted
				return task
			},
			reason:      "Too late",
			expectError: ErrPackTaskCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := tt.setupTask()
			err := task.Cancel(tt.reason)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, PackTaskStatusCancelled, task.Status)
			}
		})
	}
}

// TestPackTaskWorkflow tests complete packing workflow
func TestPackTaskWorkflow(t *testing.T) {
	// Create task
	task, err := NewPackTask("PACK-001", "ORD-001", "WAVE-001", createTestPackItems())
	assert.NoError(t, err)
	assert.Equal(t, PackTaskStatusPending, task.Status)

	// Assign task
	err = task.Assign("PACKER-123", "STATION-A1")
	assert.NoError(t, err)

	// Start task
	err = task.Start()
	assert.NoError(t, err)
	assert.Equal(t, PackTaskStatusInProgress, task.Status)

	// Verify items
	err = task.VerifyItem("SKU-001")
	assert.NoError(t, err)
	err = task.VerifyItem("SKU-002")
	assert.NoError(t, err)

	// Select packaging
	err = task.SelectPackaging(PackageTypeBox, createTestDimensions(), []string{"bubble_wrap", "tape"})
	assert.NoError(t, err)

	// Seal package
	err = task.SealPackage()
	assert.NoError(t, err)
	assert.Equal(t, PackTaskStatusPacked, task.Status)

	// Apply label
	err = task.ApplyLabel(createTestShippingLabel())
	assert.NoError(t, err)
	assert.Equal(t, PackTaskStatusLabeled, task.Status)

	// Complete task
	err = task.Complete()
	assert.NoError(t, err)
	assert.Equal(t, PackTaskStatusCompleted, task.Status)

	// Verify all events generated
	events := task.GetDomainEvents()
	assert.GreaterOrEqual(t, len(events), 4) // Created, Packaging, Sealed, Label, Completed
}

// TestPackTaskGetProgress tests progress status
func TestPackTaskGetProgress(t *testing.T) {
	task, _ := NewPackTask("PACK-001", "ORD-001", "WAVE-001", createTestPackItems())

	// Pending
	progress := task.GetProgress()
	assert.Contains(t, progress, "Waiting")

	// In progress
	task.Start()
	progress = task.GetProgress()
	assert.Contains(t, progress, "Verifying")

	// Packed
	task.VerifyItem("SKU-001")
	task.VerifyItem("SKU-002")
	task.SelectPackaging(PackageTypeBox, createTestDimensions(), []string{"tape"})
	task.SealPackage()
	progress = task.GetProgress()
	assert.Contains(t, progress, "Packed")

	// Labeled
	task.ApplyLabel(createTestShippingLabel())
	progress = task.GetProgress()
	assert.Contains(t, progress, "Labeled")

	// Completed
	task.Complete()
	progress = task.GetProgress()
	assert.Contains(t, progress, "Completed")
}

// TestPackTaskDomainEvents tests domain event handling
func TestPackTaskDomainEvents(t *testing.T) {
	task, _ := NewPackTask("PACK-001", "ORD-001", "WAVE-001", createTestPackItems())

	// Check initial event
	events := task.GetDomainEvents()
	assert.Len(t, events, 1)
	_, ok := events[0].(*PackTaskCreatedEvent)
	assert.True(t, ok)

	// Seal package
	task.Assign("PACKER-123", "STATION-A1")
	task.Start()
	task.SelectPackaging(PackageTypeBox, createTestDimensions(), []string{"tape"})
	task.VerifyItem("SKU-001")
	task.VerifyItem("SKU-002")
	task.SealPackage()
	events = task.GetDomainEvents()
	assert.GreaterOrEqual(t, len(events), 3) // Created, Packaging, Sealed

	// Clear events
	task.ClearDomainEvents()
	events = task.GetDomainEvents()
	assert.Len(t, events, 0)
}

// BenchmarkNewPackTask benchmarks pack task creation
func BenchmarkNewPackTask(b *testing.B) {
	items := createTestPackItems()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewPackTask("PACK-001", "ORD-001", "WAVE-001", items)
	}
}

// BenchmarkVerifyItem benchmarks item verification
func BenchmarkVerifyItem(b *testing.B) {
	task, _ := NewPackTask("PACK-001", "ORD-001", "WAVE-001", createTestPackItems())
	task.Assign("PACKER-123", "STATION-A1")
	task.Start()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset verification for benchmark
		task.Items[0].Verified = false
		task.VerifyItem("SKU-001")
	}
}
