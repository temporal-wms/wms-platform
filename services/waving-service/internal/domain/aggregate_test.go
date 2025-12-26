package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test fixtures
func createTestWaveConfiguration() WaveConfiguration {
	return WaveConfiguration{
		MaxOrders:           50,
		MaxItems:            500,
		MaxWeight:           1000.0,
		CutoffTime:          time.Now().Add(8 * time.Hour),
		ReleaseDelay:        30 * time.Minute,
		AutoRelease:         false,
		OptimizeForCarrier:  true,
		OptimizeForZone:     true,
		OptimizeForPriority: true,
	}
}

func createTestWaveOrder(orderID string) WaveOrder {
	return WaveOrder{
		OrderID:            orderID,
		CustomerID:         "CUST-001",
		Priority:           "same_day",
		ItemCount:          5,
		TotalWeight:        10.5,
		PromisedDeliveryAt: time.Now().Add(24 * time.Hour),
		CarrierCutoff:      time.Now().Add(8 * time.Hour),
		Zone:               "ZONE-A",
		Status:             "pending",
	}
}

// TestNewWave tests wave creation
func TestNewWave(t *testing.T) {
	tests := []struct {
		name        string
		waveID      string
		waveType    WaveType
		mode        FulfillmentMode
		config      WaveConfiguration
		expectError error
	}{
		{
			name:        "Valid digital wave creation",
			waveID:      "WAVE-001",
			waveType:    WaveTypeDigital,
			mode:        FulfillmentModeWave,
			config:      createTestWaveConfiguration(),
			expectError: nil,
		},
		{
			name:        "Valid priority wave creation",
			waveID:      "WAVE-002",
			waveType:    WaveTypePriority,
			mode:        FulfillmentModeWave,
			config:      createTestWaveConfiguration(),
			expectError: nil,
		},
		{
			name:        "Invalid wave type",
			waveID:      "WAVE-003",
			waveType:    WaveType("invalid"),
			mode:        FulfillmentModeWave,
			config:      createTestWaveConfiguration(),
			expectError: ErrInvalidWaveType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wave, err := NewWave(tt.waveID, tt.waveType, tt.mode, tt.config)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
				assert.Nil(t, wave)
			} else {
				require.NoError(t, err)
				require.NotNil(t, wave)
				assert.Equal(t, tt.waveID, wave.WaveID)
				assert.Equal(t, tt.waveType, wave.WaveType)
				assert.Equal(t, WaveStatusPlanning, wave.Status)
				assert.Equal(t, 5, wave.Priority) // Default priority
				assert.NotZero(t, wave.CreatedAt)
				assert.NotZero(t, wave.UpdatedAt)

				// Check domain event was created
				events := wave.GetDomainEvents()
				assert.Len(t, events, 1)
				event, ok := events[0].(*WaveCreatedEvent)
				assert.True(t, ok)
				assert.Equal(t, tt.waveID, event.WaveID)
			}
		})
	}
}

// TestWaveAddOrder tests adding orders to a wave
func TestWaveAddOrder(t *testing.T) {
	tests := []struct {
		name        string
		setupWave   func() *Wave
		order       WaveOrder
		expectError bool
		errorMsg    string
	}{
		{
			name: "Add order to planning wave",
			setupWave: func() *Wave {
				wave, _ := NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, createTestWaveConfiguration())
				return wave
			},
			order:       createTestWaveOrder("ORD-001"),
			expectError: false,
		},
		{
			name: "Cannot add duplicate order",
			setupWave: func() *Wave {
				wave, _ := NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, createTestWaveConfiguration())
				wave.AddOrder(createTestWaveOrder("ORD-001"))
				return wave
			},
			order:       createTestWaveOrder("ORD-001"),
			expectError: true,
			errorMsg:    "order is already in this wave",
		},
		{
			name: "Cannot add order to released wave",
			setupWave: func() *Wave {
				wave, _ := NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, createTestWaveConfiguration())
				wave.AddOrder(createTestWaveOrder("ORD-001"))
				wave.Schedule(time.Now().Add(1*time.Hour), time.Now().Add(3*time.Hour))
				wave.Release()
				return wave
			},
			order:       createTestWaveOrder("ORD-002"),
			expectError: true,
			errorMsg:    "wave has already been released",
		},
		{
			name: "Cannot exceed max orders",
			setupWave: func() *Wave {
				config := createTestWaveConfiguration()
				config.MaxOrders = 1
				wave, _ := NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, config)
				wave.AddOrder(createTestWaveOrder("ORD-001"))
				return wave
			},
			order:       createTestWaveOrder("ORD-002"),
			expectError: true,
			errorMsg:    "wave has reached maximum order capacity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wave := tt.setupWave()
			err := wave.AddOrder(tt.order)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Greater(t, len(wave.Orders), 0)
			}
		})
	}
}

// TestWaveRemoveOrder tests removing orders from a wave
func TestWaveRemoveOrder(t *testing.T) {
	tests := []struct {
		name        string
		setupWave   func() *Wave
		orderID     string
		expectError bool
		errorMsg    string
	}{
		{
			name: "Remove order from planning wave",
			setupWave: func() *Wave {
				wave, _ := NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, createTestWaveConfiguration())
				wave.AddOrder(createTestWaveOrder("ORD-001"))
				wave.AddOrder(createTestWaveOrder("ORD-002"))
				return wave
			},
			orderID:     "ORD-001",
			expectError: false,
		},
		{
			name: "Cannot remove order from released wave",
			setupWave: func() *Wave {
				wave, _ := NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, createTestWaveConfiguration())
				wave.AddOrder(createTestWaveOrder("ORD-001"))
				wave.Schedule(time.Now().Add(1*time.Hour), time.Now().Add(3*time.Hour))
				wave.Release()
				return wave
			},
			orderID:     "ORD-001",
			expectError: true,
			errorMsg:    "wave has already been released",
		},
		{
			name: "Order not found in wave",
			setupWave: func() *Wave {
				wave, _ := NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, createTestWaveConfiguration())
				wave.AddOrder(createTestWaveOrder("ORD-001"))
				return wave
			},
			orderID:     "ORD-999",
			expectError: true,
			errorMsg:    "order not found in wave",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wave := tt.setupWave()
			initialCount := len(wave.Orders)
			err := wave.RemoveOrder(tt.orderID)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, initialCount-1, len(wave.Orders))
			}
		})
	}
}

// TestWaveSchedule tests wave scheduling
func TestWaveSchedule(t *testing.T) {
	tests := []struct {
		name        string
		setupWave   func() *Wave
		startTime   time.Time
		endTime     time.Time
		expectError bool
		errorMsg    string
	}{
		{
			name: "Schedule planning wave with orders",
			setupWave: func() *Wave {
				wave, _ := NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, createTestWaveConfiguration())
				wave.AddOrder(createTestWaveOrder("ORD-001"))
				return wave
			},
			startTime:   time.Now().Add(1 * time.Hour),
			endTime:     time.Now().Add(3 * time.Hour),
			expectError: false,
		},
		{
			name: "Cannot schedule empty wave",
			setupWave: func() *Wave {
				wave, _ := NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, createTestWaveConfiguration())
				return wave
			},
			startTime:   time.Now().Add(1 * time.Hour),
			endTime:     time.Now().Add(3 * time.Hour),
			expectError: true,
			errorMsg:    "wave must contain at least one order",
		},
		{
			name: "Cannot schedule already scheduled wave",
			setupWave: func() *Wave {
				wave, _ := NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, createTestWaveConfiguration())
				wave.AddOrder(createTestWaveOrder("ORD-001"))
				wave.Schedule(time.Now().Add(1*time.Hour), time.Now().Add(3*time.Hour))
				return wave
			},
			startTime:   time.Now().Add(2 * time.Hour),
			endTime:     time.Now().Add(4 * time.Hour),
			expectError: true,
			errorMsg:    "wave can only be scheduled from planning status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wave := tt.setupWave()
			err := wave.Schedule(tt.startTime, tt.endTime)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, WaveStatusScheduled, wave.Status)
				assert.Equal(t, tt.startTime, wave.ScheduledStart)
				assert.Equal(t, tt.endTime, wave.ScheduledEnd)
			}
		})
	}
}

// TestWaveRelease tests wave release
func TestWaveRelease(t *testing.T) {
	tests := []struct {
		name        string
		setupWave   func() *Wave
		expectError bool
		errorMsg    string
	}{
		{
			name: "Release scheduled wave",
			setupWave: func() *Wave {
				wave, _ := NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, createTestWaveConfiguration())
				wave.AddOrder(createTestWaveOrder("ORD-001"))
				wave.Schedule(time.Now().Add(1*time.Hour), time.Now().Add(3*time.Hour))
				return wave
			},
			expectError: false,
		},
		{
			name: "Release planning wave directly",
			setupWave: func() *Wave {
				wave, _ := NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, createTestWaveConfiguration())
				wave.AddOrder(createTestWaveOrder("ORD-001"))
				return wave
			},
			expectError: false,
		},
		{
			name: "Cannot release empty wave",
			setupWave: func() *Wave {
				wave, _ := NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, createTestWaveConfiguration())
				return wave
			},
			expectError: true,
			errorMsg:    "wave must contain at least one order",
		},
		{
			name: "Cannot release already released wave",
			setupWave: func() *Wave {
				wave, _ := NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, createTestWaveConfiguration())
				wave.AddOrder(createTestWaveOrder("ORD-001"))
				wave.Schedule(time.Now().Add(1*time.Hour), time.Now().Add(3*time.Hour))
				wave.Release()
				return wave
			},
			expectError: true,
			errorMsg:    "wave has already been released",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wave := tt.setupWave()
			err := wave.Release()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, WaveStatusReleased, wave.Status)
				assert.NotNil(t, wave.ReleasedAt)
				assert.NotNil(t, wave.ActualStart)
				// Check all orders are in picking status
				for _, order := range wave.Orders {
					assert.Equal(t, "picking", order.Status)
				}
			}
		})
	}
}

// TestWaveComplete tests wave completion
func TestWaveComplete(t *testing.T) {
	t.Run("Complete wave successfully", func(t *testing.T) {
		wave, _ := NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, createTestWaveConfiguration())
		wave.AddOrder(createTestWaveOrder("ORD-001"))
		wave.Release()

		err := wave.Complete()
		assert.NoError(t, err)
		assert.Equal(t, WaveStatusCompleted, wave.Status)
		assert.NotNil(t, wave.CompletedAt)
		assert.NotNil(t, wave.ActualEnd)
	})

	t.Run("Cannot complete already completed wave", func(t *testing.T) {
		wave, _ := NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, createTestWaveConfiguration())
		wave.AddOrder(createTestWaveOrder("ORD-001"))
		wave.Release()
		wave.Complete()

		err := wave.Complete()
		assert.Error(t, err)
		assert.Equal(t, ErrWaveAlreadyClosed, err)
	})
}

// TestWaveCompleteOrder tests completing individual orders
func TestWaveCompleteOrder(t *testing.T) {
	t.Run("Complete order in wave", func(t *testing.T) {
		wave, _ := NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, createTestWaveConfiguration())
		wave.AddOrder(createTestWaveOrder("ORD-001"))
		wave.AddOrder(createTestWaveOrder("ORD-002"))
		wave.Release()

		err := wave.CompleteOrder("ORD-001")
		assert.NoError(t, err)
		assert.Equal(t, "completed", wave.Orders[0].Status)
		assert.Equal(t, WaveStatusReleased, wave.Status) // Not all orders complete
	})

	t.Run("Auto-complete wave when all orders done", func(t *testing.T) {
		wave, _ := NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, createTestWaveConfiguration())
		wave.AddOrder(createTestWaveOrder("ORD-001"))
		wave.Release()

		err := wave.CompleteOrder("ORD-001")
		assert.NoError(t, err)
		assert.Equal(t, WaveStatusCompleted, wave.Status) // Auto-completed
	})

	t.Run("Order not found", func(t *testing.T) {
		wave, _ := NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, createTestWaveConfiguration())
		wave.AddOrder(createTestWaveOrder("ORD-001"))
		wave.Release()

		err := wave.CompleteOrder("ORD-999")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "order not found in wave")
	})
}

// TestWaveCancel tests wave cancellation
func TestWaveCancel(t *testing.T) {
	t.Run("Cancel planning wave", func(t *testing.T) {
		wave, _ := NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, createTestWaveConfiguration())
		wave.AddOrder(createTestWaveOrder("ORD-001"))

		err := wave.Cancel("Test cancellation")
		assert.NoError(t, err)
		assert.Equal(t, WaveStatusCancelled, wave.Status)
	})

	t.Run("Cannot cancel completed wave", func(t *testing.T) {
		wave, _ := NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, createTestWaveConfiguration())
		wave.AddOrder(createTestWaveOrder("ORD-001"))
		wave.Release()
		wave.Complete()

		err := wave.Cancel("Test cancellation")
		assert.Error(t, err)
		assert.Equal(t, ErrWaveAlreadyClosed, err)
	})
}

// TestWaveMetrics tests wave metric methods
func TestWaveMetrics(t *testing.T) {
	wave, _ := NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, createTestWaveConfiguration())
	wave.AddOrder(createTestWaveOrder("ORD-001"))
	wave.AddOrder(createTestWaveOrder("ORD-002"))
	wave.AddOrder(createTestWaveOrder("ORD-003"))

	t.Run("GetOrderCount", func(t *testing.T) {
		assert.Equal(t, 3, wave.GetOrderCount())
	})

	t.Run("GetTotalItems", func(t *testing.T) {
		assert.Equal(t, 15, wave.GetTotalItems()) // 3 orders * 5 items each
	})

	t.Run("GetTotalWeight", func(t *testing.T) {
		assert.Equal(t, 31.5, wave.GetTotalWeight()) // 3 orders * 10.5 weight each
	})

	t.Run("GetCompletedOrderCount", func(t *testing.T) {
		wave.Release()
		wave.CompleteOrder("ORD-001")
		wave.CompleteOrder("ORD-002")
		assert.Equal(t, 2, wave.GetCompletedOrderCount())
	})

	t.Run("GetProgress", func(t *testing.T) {
		progress := wave.GetProgress()
		assert.InDelta(t, 66.67, progress, 0.1) // 2 out of 3 complete
	})
}

// TestWaveDomainEvents tests domain event handling
func TestWaveDomainEvents(t *testing.T) {
	wave, _ := NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, createTestWaveConfiguration())

	// Check initial event
	events := wave.GetDomainEvents()
	assert.Len(t, events, 1)
	_, ok := events[0].(*WaveCreatedEvent)
	assert.True(t, ok)

	// Add order
	wave.AddOrder(createTestWaveOrder("ORD-001"))
	events = wave.GetDomainEvents()
	assert.Len(t, events, 2)

	// Schedule
	wave.Schedule(time.Now().Add(1*time.Hour), time.Now().Add(3*time.Hour))
	events = wave.GetDomainEvents()
	assert.Len(t, events, 3)

	// Release
	wave.Release()
	events = wave.GetDomainEvents()
	assert.Len(t, events, 4)

	// Clear events
	wave.ClearDomainEvents()
	events = wave.GetDomainEvents()
	assert.Len(t, events, 0)
}

// BenchmarkNewWave benchmarks wave creation
func BenchmarkNewWave(b *testing.B) {
	config := createTestWaveConfiguration()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, config)
	}
}

// BenchmarkAddOrder benchmarks adding orders to wave
func BenchmarkAddOrder(b *testing.B) {
	wave, _ := NewWave("WAVE-001", WaveTypeDigital, FulfillmentModeWave, createTestWaveConfiguration())
	order := createTestWaveOrder("ORD-001")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Note: This will fail after first iteration, but measures the happy path
		wave.AddOrder(order)
	}
}
