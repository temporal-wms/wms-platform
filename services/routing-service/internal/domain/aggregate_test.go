package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test fixtures
func createTestLocation(locationID, aisle string, rack, level int, x, y float64) Location {
	return Location{
		LocationID: locationID,
		Aisle:      aisle,
		Rack:       rack,
		Level:      level,
		Position:   "A",
		Zone:       "ZONE-A",
		X:          x,
		Y:          y,
	}
}

func createTestRouteItems() []RouteItem {
	return []RouteItem{
		{
			SKU:      "SKU-001",
			Quantity: 2,
			Location: createTestLocation("A-05-1-A", "A", 5, 1, 10.0, 5.0),
		},
		{
			SKU:      "SKU-002",
			Quantity: 3,
			Location: createTestLocation("A-10-2-A", "A", 10, 2, 20.0, 10.0),
		},
		{
			SKU:      "SKU-003",
			Quantity: 1,
			Location: createTestLocation("B-03-1-A", "B", 3, 1, 15.0, 30.0),
		},
	}
}

// TestNewPickRoute tests pick route creation
func TestNewPickRoute(t *testing.T) {
	tests := []struct {
		name        string
		routeID     string
		orderID     string
		waveID      string
		strategy    RoutingStrategy
		items       []RouteItem
		expectError error
	}{
		{
			name:        "Valid route creation",
			routeID:     "ROUTE-001",
			orderID:     "ORD-001",
			waveID:      "WAVE-001",
			strategy:    StrategySShape,
			items:       createTestRouteItems(),
			expectError: nil,
		},
		{
			name:        "Cannot create empty route",
			routeID:     "ROUTE-002",
			orderID:     "ORD-002",
			waveID:      "WAVE-001",
			strategy:    StrategySShape,
			items:       []RouteItem{},
			expectError: ErrRouteEmpty,
		},
		{
			name:        "Cannot create route with invalid strategy",
			routeID:     "ROUTE-003",
			orderID:     "ORD-003",
			waveID:      "WAVE-001",
			strategy:    RoutingStrategy("invalid"),
			items:       createTestRouteItems(),
			expectError: ErrInvalidStrategy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route, err := NewPickRoute(tt.routeID, tt.orderID, tt.waveID, tt.strategy, tt.items)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
				assert.Nil(t, route)
			} else {
				require.NoError(t, err)
				require.NotNil(t, route)
				assert.Equal(t, tt.routeID, route.RouteID)
				assert.Equal(t, tt.orderID, route.OrderID)
				assert.Equal(t, tt.waveID, route.WaveID)
				assert.Equal(t, RouteStatusPending, route.Status)
				assert.Equal(t, tt.strategy, route.Strategy)
				assert.Len(t, route.Stops, len(tt.items))
				assert.Equal(t, 6, route.TotalItems) // 2 + 3 + 1
				assert.NotZero(t, route.CreatedAt)

				// Check domain event
				events := route.GetDomainEvents()
				assert.Len(t, events, 1)
				event, ok := events[0].(*RouteCreatedEvent)
				assert.True(t, ok)
				assert.Equal(t, tt.routeID, event.RouteID)
			}
		})
	}
}

// TestPickRouteOptimize tests route optimization
func TestPickRouteOptimize(t *testing.T) {
	strategies := []RoutingStrategy{
		StrategyReturn,
		StrategySShape,
		StrategyLargestGap,
		StrategyCombined,
		StrategyNearest,
	}

	startLoc := createTestLocation("START", "START", 0, 0, 0.0, 0.0)
	endLoc := createTestLocation("END", "END", 0, 0, 0.0, 0.0)

	for _, strategy := range strategies {
		t.Run(string(strategy), func(t *testing.T) {
			route, _ := NewPickRoute("ROUTE-001", "ORD-001", "WAVE-001", strategy, createTestRouteItems())

			err := route.OptimizeRoute(startLoc, endLoc)
			assert.NoError(t, err)
			assert.Equal(t, startLoc, route.StartLocation)
			assert.Equal(t, endLoc, route.EndLocation)
			assert.Greater(t, route.EstimatedDistance, 0.0)
			assert.Greater(t, route.EstimatedTime, int64(0))

			// Verify stops are numbered sequentially
			for i, stop := range route.Stops {
				assert.Equal(t, i+1, stop.StopNumber)
			}

			// Check domain event
			events := route.GetDomainEvents()
			assert.GreaterOrEqual(t, len(events), 2) // Created + Optimized
		})
	}
}

// TestPickRouteOptimizeValidation tests optimization validation
func TestPickRouteOptimizeValidation(t *testing.T) {
	tests := []struct {
		name        string
		setupRoute  func() *PickRoute
		expectError error
	}{
		{
			name: "Cannot optimize already started route",
			setupRoute: func() *PickRoute {
				route, _ := NewPickRoute("ROUTE-001", "ORD-001", "WAVE-001", StrategySShape, createTestRouteItems())
				route.Status = RouteStatusInProgress
				return route
			},
			expectError: ErrRouteAlreadyStarted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := tt.setupRoute()
			startLoc := createTestLocation("START", "START", 0, 0, 0.0, 0.0)
			endLoc := createTestLocation("END", "END", 0, 0, 0.0, 0.0)

			err := route.OptimizeRoute(startLoc, endLoc)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPickRouteStart tests route starting
func TestPickRouteStart(t *testing.T) {
	tests := []struct {
		name        string
		setupRoute  func() *PickRoute
		pickerID    string
		expectError bool
	}{
		{
			name: "Start pending route",
			setupRoute: func() *PickRoute {
				route, _ := NewPickRoute("ROUTE-001", "ORD-001", "WAVE-001", StrategySShape, createTestRouteItems())
				return route
			},
			pickerID:    "PICKER-123",
			expectError: false,
		},
		{
			name: "Resume paused route",
			setupRoute: func() *PickRoute {
				route, _ := NewPickRoute("ROUTE-002", "ORD-002", "WAVE-001", StrategySShape, createTestRouteItems())
				route.Status = RouteStatusPaused
				return route
			},
			pickerID:    "PICKER-456",
			expectError: false,
		},
		{
			name: "Cannot start already started route",
			setupRoute: func() *PickRoute {
				route, _ := NewPickRoute("ROUTE-003", "ORD-003", "WAVE-001", StrategySShape, createTestRouteItems())
				route.Start("PICKER-111")
				return route
			},
			pickerID:    "PICKER-999",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := tt.setupRoute()
			err := route.Start(tt.pickerID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, RouteStatusInProgress, route.Status)
				assert.Equal(t, tt.pickerID, route.PickerID)
				assert.NotNil(t, route.StartedAt)
			}
		})
	}
}

// TestPickRouteCompleteStop tests stop completion
func TestPickRouteCompleteStop(t *testing.T) {
	tests := []struct {
		name        string
		setupRoute  func() *PickRoute
		stopNumber  int
		pickedQty   int
		toteID      string
		expectError bool
	}{
		{
			name: "Complete valid stop",
			setupRoute: func() *PickRoute {
				route, _ := NewPickRoute("ROUTE-001", "ORD-001", "WAVE-001", StrategySShape, createTestRouteItems())
				route.Start("PICKER-123")
				return route
			},
			stopNumber:  1,
			pickedQty:   2,
			toteID:      "TOTE-001",
			expectError: false,
		},
		{
			name: "Cannot complete stop on pending route",
			setupRoute: func() *PickRoute {
				route, _ := NewPickRoute("ROUTE-002", "ORD-002", "WAVE-001", StrategySShape, createTestRouteItems())
				return route
			},
			stopNumber:  1,
			pickedQty:   2,
			toteID:      "TOTE-001",
			expectError: true,
		},
		{
			name: "Cannot complete invalid stop number",
			setupRoute: func() *PickRoute {
				route, _ := NewPickRoute("ROUTE-003", "ORD-003", "WAVE-001", StrategySShape, createTestRouteItems())
				route.Start("PICKER-123")
				return route
			},
			stopNumber:  999,
			pickedQty:   2,
			toteID:      "TOTE-001",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := tt.setupRoute()
			initialPickedItems := route.PickedItems

			err := route.CompleteStop(tt.stopNumber, tt.pickedQty, tt.toteID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, initialPickedItems+tt.pickedQty, route.PickedItems)

				// Verify stop is marked as completed
				for _, stop := range route.Stops {
					if stop.StopNumber == tt.stopNumber {
						assert.Equal(t, "completed", stop.Status)
						assert.Equal(t, tt.pickedQty, stop.PickedQty)
						assert.Equal(t, tt.toteID, stop.ToteID)
						assert.NotNil(t, stop.PickedAt)
					}
				}
			}
		})
	}
}

// TestPickRouteAutoComplete tests automatic route completion
func TestPickRouteAutoComplete(t *testing.T) {
	route, _ := NewPickRoute("ROUTE-001", "ORD-001", "WAVE-001", StrategySShape, createTestRouteItems())
	route.Start("PICKER-123")

	// Complete stop 1
	err := route.CompleteStop(1, 2, "TOTE-001")
	assert.NoError(t, err)
	assert.Equal(t, RouteStatusInProgress, route.Status)

	// Complete stop 2
	err = route.CompleteStop(2, 3, "TOTE-001")
	assert.NoError(t, err)
	assert.Equal(t, RouteStatusInProgress, route.Status)

	// Complete last stop - should auto-complete
	err = route.CompleteStop(3, 1, "TOTE-001")
	assert.NoError(t, err)
	assert.Equal(t, RouteStatusCompleted, route.Status)
	assert.NotNil(t, route.CompletedAt)
	assert.Equal(t, 6, route.PickedItems)
}

// TestPickRouteSkipStop tests stop skipping
func TestPickRouteSkipStop(t *testing.T) {
	tests := []struct {
		name        string
		setupRoute  func() *PickRoute
		stopNumber  int
		reason      string
		expectError bool
	}{
		{
			name: "Skip stop on in-progress route",
			setupRoute: func() *PickRoute {
				route, _ := NewPickRoute("ROUTE-001", "ORD-001", "WAVE-001", StrategySShape, createTestRouteItems())
				route.Start("PICKER-123")
				return route
			},
			stopNumber:  1,
			reason:      "Item not found",
			expectError: false,
		},
		{
			name: "Cannot skip stop on pending route",
			setupRoute: func() *PickRoute {
				route, _ := NewPickRoute("ROUTE-002", "ORD-002", "WAVE-001", StrategySShape, createTestRouteItems())
				return route
			},
			stopNumber:  1,
			reason:      "Item not found",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := tt.setupRoute()
			err := route.SkipStop(tt.stopNumber, tt.reason)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify stop is marked as skipped
				for _, stop := range route.Stops {
					if stop.StopNumber == tt.stopNumber {
						assert.Equal(t, "skipped", stop.Status)
						assert.Equal(t, tt.reason, stop.Notes)
					}
				}
			}
		})
	}
}

// TestPickRouteComplete tests manual route completion
func TestPickRouteComplete(t *testing.T) {
	tests := []struct {
		name        string
		setupRoute  func() *PickRoute
		expectError error
	}{
		{
			name: "Complete in-progress route",
			setupRoute: func() *PickRoute {
				route, _ := NewPickRoute("ROUTE-001", "ORD-001", "WAVE-001", StrategySShape, createTestRouteItems())
				route.Start("PICKER-123")
				return route
			},
			expectError: nil,
		},
		{
			name: "Cannot complete already completed route",
			setupRoute: func() *PickRoute {
				route, _ := NewPickRoute("ROUTE-002", "ORD-002", "WAVE-001", StrategySShape, createTestRouteItems())
				route.Start("PICKER-123")
				route.Complete()
				return route
			},
			expectError: ErrRouteCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := tt.setupRoute()
			err := route.Complete()

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, RouteStatusCompleted, route.Status)
				assert.NotNil(t, route.CompletedAt)
				assert.Greater(t, route.ActualTime, int64(0))
			}
		})
	}
}

// TestPickRoutePause tests route pausing
func TestPickRoutePause(t *testing.T) {
	tests := []struct {
		name        string
		setupRoute  func() *PickRoute
		expectError bool
	}{
		{
			name: "Pause in-progress route",
			setupRoute: func() *PickRoute {
				route, _ := NewPickRoute("ROUTE-001", "ORD-001", "WAVE-001", StrategySShape, createTestRouteItems())
				route.Start("PICKER-123")
				return route
			},
			expectError: false,
		},
		{
			name: "Cannot pause pending route",
			setupRoute: func() *PickRoute {
				route, _ := NewPickRoute("ROUTE-002", "ORD-002", "WAVE-001", StrategySShape, createTestRouteItems())
				return route
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := tt.setupRoute()
			err := route.Pause()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, RouteStatusPaused, route.Status)
			}
		})
	}
}

// TestPickRouteCancel tests route cancellation
func TestPickRouteCancel(t *testing.T) {
	tests := []struct {
		name        string
		setupRoute  func() *PickRoute
		reason      string
		expectError error
	}{
		{
			name: "Cancel pending route",
			setupRoute: func() *PickRoute {
				route, _ := NewPickRoute("ROUTE-001", "ORD-001", "WAVE-001", StrategySShape, createTestRouteItems())
				return route
			},
			reason:      "Order cancelled",
			expectError: nil,
		},
		{
			name: "Cancel in-progress route",
			setupRoute: func() *PickRoute {
				route, _ := NewPickRoute("ROUTE-002", "ORD-002", "WAVE-001", StrategySShape, createTestRouteItems())
				route.Start("PICKER-123")
				return route
			},
			reason:      "Picker unavailable",
			expectError: nil,
		},
		{
			name: "Cannot cancel completed route",
			setupRoute: func() *PickRoute {
				route, _ := NewPickRoute("ROUTE-003", "ORD-003", "WAVE-001", StrategySShape, createTestRouteItems())
				route.Start("PICKER-123")
				route.Complete()
				return route
			},
			reason:      "Too late",
			expectError: ErrRouteCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := tt.setupRoute()
			err := route.Cancel(tt.reason)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, RouteStatusCancelled, route.Status)
			}
		})
	}
}

// TestPickRouteGetProgress tests progress calculation
func TestPickRouteGetProgress(t *testing.T) {
	route, _ := NewPickRoute("ROUTE-001", "ORD-001", "WAVE-001", StrategySShape, createTestRouteItems())
	route.Start("PICKER-123")

	// Initially 0%
	progress := route.GetProgress()
	assert.Equal(t, 0.0, progress)

	// Complete one stop
	route.CompleteStop(1, 2, "TOTE-001")
	progress = route.GetProgress()
	assert.InDelta(t, 33.33, progress, 0.1) // 1 out of 3 stops

	// Complete second stop
	route.CompleteStop(2, 3, "TOTE-001")
	progress = route.GetProgress()
	assert.InDelta(t, 66.67, progress, 0.1) // 2 out of 3 stops

	// Complete last stop
	route.CompleteStop(3, 1, "TOTE-001")
	progress = route.GetProgress()
	assert.Equal(t, 100.0, progress)
}

// TestPickRouteGetCurrentStop tests getting current stop
func TestPickRouteGetCurrentStop(t *testing.T) {
	route, _ := NewPickRoute("ROUTE-001", "ORD-001", "WAVE-001", StrategySShape, createTestRouteItems())
	route.Start("PICKER-123")

	// First stop should be current
	currentStop := route.GetCurrentStop()
	assert.NotNil(t, currentStop)
	assert.Equal(t, 1, currentStop.StopNumber)

	// Complete first stop
	route.CompleteStop(1, 2, "TOTE-001")
	currentStop = route.GetCurrentStop()
	assert.NotNil(t, currentStop)
	assert.Equal(t, 2, currentStop.StopNumber)

	// Complete all stops
	route.CompleteStop(2, 3, "TOTE-001")
	route.CompleteStop(3, 1, "TOTE-001")
	currentStop = route.GetCurrentStop()
	assert.Nil(t, currentStop)
}

// TestPickRouteWorkflow tests complete route workflow
func TestPickRouteWorkflow(t *testing.T) {
	// Create route
	route, err := NewPickRoute("ROUTE-001", "ORD-001", "WAVE-001", StrategySShape, createTestRouteItems())
	assert.NoError(t, err)
	assert.Equal(t, RouteStatusPending, route.Status)

	// Optimize route
	startLoc := createTestLocation("START", "START", 0, 0, 0.0, 0.0)
	endLoc := createTestLocation("END", "END", 0, 0, 0.0, 0.0)
	err = route.OptimizeRoute(startLoc, endLoc)
	assert.NoError(t, err)
	assert.Greater(t, route.EstimatedDistance, 0.0)

	// Start route
	err = route.Start("PICKER-123")
	assert.NoError(t, err)
	assert.Equal(t, RouteStatusInProgress, route.Status)

	// Complete stops
	err = route.CompleteStop(1, 2, "TOTE-001")
	assert.NoError(t, err)
	err = route.CompleteStop(2, 3, "TOTE-001")
	assert.NoError(t, err)
	err = route.CompleteStop(3, 1, "TOTE-001")
	assert.NoError(t, err)

	// Route should be auto-completed
	assert.Equal(t, RouteStatusCompleted, route.Status)
	assert.Equal(t, 6, route.PickedItems)
	assert.NotNil(t, route.CompletedAt)
}

// TestPickRouteDomainEvents tests domain event handling
func TestPickRouteDomainEvents(t *testing.T) {
	route, _ := NewPickRoute("ROUTE-001", "ORD-001", "WAVE-001", StrategySShape, createTestRouteItems())

	// Check initial event
	events := route.GetDomainEvents()
	assert.Len(t, events, 1)
	_, ok := events[0].(*RouteCreatedEvent)
	assert.True(t, ok)

	// Optimize
	startLoc := createTestLocation("START", "START", 0, 0, 0.0, 0.0)
	endLoc := createTestLocation("END", "END", 0, 0, 0.0, 0.0)
	route.OptimizeRoute(startLoc, endLoc)
	events = route.GetDomainEvents()
	assert.Len(t, events, 2)

	// Start
	route.Start("PICKER-123")
	events = route.GetDomainEvents()
	assert.Len(t, events, 3)

	// Clear events
	route.ClearDomainEvents()
	events = route.GetDomainEvents()
	assert.Len(t, events, 0)
}

// BenchmarkNewPickRoute benchmarks route creation
func BenchmarkNewPickRoute(b *testing.B) {
	items := createTestRouteItems()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewPickRoute("ROUTE-001", "ORD-001", "WAVE-001", StrategySShape, items)
	}
}

// BenchmarkOptimizeRoute benchmarks route optimization
func BenchmarkOptimizeRoute(b *testing.B) {
	route, _ := NewPickRoute("ROUTE-001", "ORD-001", "WAVE-001", StrategySShape, createTestRouteItems())
	startLoc := createTestLocation("START", "START", 0, 0, 0.0, 0.0)
	endLoc := createTestLocation("END", "END", 0, 0, 0.0, 0.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset status for benchmark
		route.Status = RouteStatusPending
		route.OptimizeRoute(startLoc, endLoc)
	}
}
