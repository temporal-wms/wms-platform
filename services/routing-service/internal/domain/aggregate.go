package domain

import (
	"errors"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Errors
var (
	ErrRouteEmpty          = errors.New("route must have at least one stop")
	ErrInvalidLocation     = errors.New("invalid location")
	ErrRouteAlreadyStarted = errors.New("route has already been started")
	ErrRouteCompleted      = errors.New("route is already completed")
	ErrInvalidStrategy     = errors.New("invalid routing strategy")
)

// RoutingStrategy represents the routing algorithm to use
type RoutingStrategy string

const (
	StrategyReturn     RoutingStrategy = "return"      // Enter aisle, pick, return to front
	StrategySShape     RoutingStrategy = "s_shape"     // Traverse entire aisle before moving to next
	StrategyLargestGap RoutingStrategy = "largest_gap" // Minimize travel by avoiding large gaps
	StrategyCombined   RoutingStrategy = "combined"    // Hybrid approach based on item density
	StrategyNearest    RoutingStrategy = "nearest"     // Nearest neighbor algorithm
)

// RouteStatus represents the status of a pick route
type RouteStatus string

const (
	RouteStatusPending    RouteStatus = "pending"     // Route calculated, not started
	RouteStatusInProgress RouteStatus = "in_progress" // Picker is working on route
	RouteStatusCompleted  RouteStatus = "completed"   // All stops completed
	RouteStatusCancelled  RouteStatus = "cancelled"   // Route was cancelled
	RouteStatusPaused     RouteStatus = "paused"      // Route temporarily paused
)

// PickRoute is the aggregate root for the Routing bounded context
type PickRoute struct {
	ID                primitive.ObjectID `bson:"_id,omitempty"`
	RouteID           string             `bson:"routeId"`
	OrderID           string             `bson:"orderId"`
	WaveID            string             `bson:"waveId"`
	PickerID          string             `bson:"pickerId,omitempty"`
	Status            RouteStatus        `bson:"status"`
	Strategy          RoutingStrategy    `bson:"strategy"`
	Stops             []RouteStop        `bson:"stops"`
	EstimatedDistance float64            `bson:"estimatedDistance"` // in meters
	ActualDistance    float64            `bson:"actualDistance"`    // in meters
	EstimatedTime     time.Duration      `bson:"estimatedTime"`
	ActualTime        time.Duration      `bson:"actualTime"`
	StartLocation     Location           `bson:"startLocation"`
	EndLocation       Location           `bson:"endLocation"`
	Zone              string             `bson:"zone"`
	TotalItems        int                `bson:"totalItems"`
	PickedItems       int                `bson:"pickedItems"`
	CreatedAt         time.Time          `bson:"createdAt"`
	UpdatedAt         time.Time          `bson:"updatedAt"`
	StartedAt         *time.Time         `bson:"startedAt,omitempty"`
	CompletedAt       *time.Time         `bson:"completedAt,omitempty"`
	DomainEvents      []DomainEvent      `bson:"-"`
}

// RouteStop represents a single stop in the pick route
type RouteStop struct {
	StopNumber   int       `bson:"stopNumber"`
	Location     Location  `bson:"location"`
	SKU          string    `bson:"sku"`
	Quantity     int       `bson:"quantity"`
	PickedQty    int       `bson:"pickedQty"`
	Status       string    `bson:"status"` // pending, completed, skipped
	ToteID       string    `bson:"toteId,omitempty"`
	PickedAt     *time.Time `bson:"pickedAt,omitempty"`
	Notes        string    `bson:"notes,omitempty"`
}

// Location represents a warehouse location
type Location struct {
	LocationID string  `bson:"locationId"` // e.g., "A-12-3-B"
	Aisle      string  `bson:"aisle"`      // e.g., "A"
	Rack       int     `bson:"rack"`       // e.g., 12
	Level      int     `bson:"level"`      // e.g., 3
	Position   string  `bson:"position"`   // e.g., "B" (left/right)
	Zone       string  `bson:"zone"`       // e.g., "ZONE-A"
	X          float64 `bson:"x"`          // X coordinate in warehouse
	Y          float64 `bson:"y"`          // Y coordinate in warehouse
}

// NewPickRoute creates a new PickRoute aggregate
func NewPickRoute(routeID, orderID, waveID string, strategy RoutingStrategy, items []RouteItem) (*PickRoute, error) {
	if len(items) == 0 {
		return nil, ErrRouteEmpty
	}

	if !isValidStrategy(strategy) {
		return nil, ErrInvalidStrategy
	}

	now := time.Now()

	route := &PickRoute{
		RouteID:      routeID,
		OrderID:      orderID,
		WaveID:       waveID,
		Status:       RouteStatusPending,
		Strategy:     strategy,
		Stops:        make([]RouteStop, 0, len(items)),
		CreatedAt:    now,
		UpdatedAt:    now,
		DomainEvents: make([]DomainEvent, 0),
	}

	// Convert items to stops (will be optimized later)
	totalItems := 0
	for i, item := range items {
		stop := RouteStop{
			StopNumber: i + 1,
			Location:   item.Location,
			SKU:        item.SKU,
			Quantity:   item.Quantity,
			Status:     "pending",
		}
		route.Stops = append(route.Stops, stop)
		totalItems += item.Quantity
	}

	route.TotalItems = totalItems

	// Determine zone from first item
	if len(items) > 0 {
		route.Zone = items[0].Location.Zone
	}

	route.AddDomainEvent(&RouteCreatedEvent{
		RouteID:   routeID,
		OrderID:   orderID,
		WaveID:    waveID,
		Strategy:  string(strategy),
		StopCount: len(route.Stops),
		CreatedAt: now,
	})

	return route, nil
}

// RouteItem represents an item to be picked
type RouteItem struct {
	SKU      string   `json:"sku"`
	Quantity int      `json:"quantity"`
	Location Location `json:"location"`
}

// OptimizeRoute optimizes the stop sequence based on the strategy
func (r *PickRoute) OptimizeRoute(startLoc, endLoc Location) error {
	if r.Status != RouteStatusPending {
		return ErrRouteAlreadyStarted
	}

	r.StartLocation = startLoc
	r.EndLocation = endLoc

	// Apply routing strategy
	switch r.Strategy {
	case StrategyReturn:
		r.optimizeReturn()
	case StrategySShape:
		r.optimizeSShape()
	case StrategyLargestGap:
		r.optimizeLargestGap()
	case StrategyCombined:
		r.optimizeCombined()
	case StrategyNearest:
		r.optimizeNearest()
	}

	// Recalculate stop numbers after optimization
	for i := range r.Stops {
		r.Stops[i].StopNumber = i + 1
	}

	// Calculate estimated distance and time
	r.EstimatedDistance = r.calculateTotalDistance()
	r.EstimatedTime = r.calculateEstimatedTime()

	r.UpdatedAt = time.Now()

	r.AddDomainEvent(&RouteOptimizedEvent{
		RouteID:           r.RouteID,
		Strategy:          string(r.Strategy),
		EstimatedDistance: r.EstimatedDistance,
		EstimatedTime:     r.EstimatedTime,
		OptimizedAt:       time.Now(),
	})

	return nil
}

// optimizeReturn implements the Return Strategy
// Picker enters aisle, picks items, returns to the front
func (r *PickRoute) optimizeReturn() {
	// Group stops by aisle
	aisleGroups := make(map[string][]RouteStop)
	for _, stop := range r.Stops {
		aisleGroups[stop.Location.Aisle] = append(aisleGroups[stop.Location.Aisle], stop)
	}

	// Sort each aisle by position (depth)
	for aisle := range aisleGroups {
		stops := aisleGroups[aisle]
		sortByRack(stops)
		aisleGroups[aisle] = stops
	}

	// Build optimized route: process each aisle, return to front
	optimized := make([]RouteStop, 0, len(r.Stops))
	aisles := getSortedAisles(aisleGroups)

	for _, aisle := range aisles {
		optimized = append(optimized, aisleGroups[aisle]...)
	}

	r.Stops = optimized
}

// optimizeSShape implements the S-Shape Strategy
// Picker traverses entire aisle before moving to next
func (r *PickRoute) optimizeSShape() {
	// Group stops by aisle
	aisleGroups := make(map[string][]RouteStop)
	for _, stop := range r.Stops {
		aisleGroups[stop.Location.Aisle] = append(aisleGroups[stop.Location.Aisle], stop)
	}

	// Build S-shape route
	optimized := make([]RouteStop, 0, len(r.Stops))
	aisles := getSortedAisles(aisleGroups)

	for i, aisle := range aisles {
		stops := aisleGroups[aisle]
		sortByRack(stops)

		// Alternate direction for S-shape
		if i%2 == 1 {
			reverse(stops)
		}
		optimized = append(optimized, stops...)
	}

	r.Stops = optimized
}

// optimizeLargestGap implements the Largest Gap Strategy
// Minimizes travel by avoiding large gaps between picks
func (r *PickRoute) optimizeLargestGap() {
	// Group stops by aisle
	aisleGroups := make(map[string][]RouteStop)
	for _, stop := range r.Stops {
		aisleGroups[stop.Location.Aisle] = append(aisleGroups[stop.Location.Aisle], stop)
	}

	optimized := make([]RouteStop, 0, len(r.Stops))
	aisles := getSortedAisles(aisleGroups)

	for _, aisle := range aisles {
		stops := aisleGroups[aisle]
		sortByRack(stops)

		if len(stops) <= 1 {
			optimized = append(optimized, stops...)
			continue
		}

		// Find largest gap
		maxGap := 0
		maxGapIndex := -1
		for i := 0; i < len(stops)-1; i++ {
			gap := stops[i+1].Location.Rack - stops[i].Location.Rack
			if gap > maxGap {
				maxGap = gap
				maxGapIndex = i
			}
		}

		// If largest gap is significant, split the aisle
		if maxGapIndex > 0 && maxGap > 3 {
			// Pick from front to gap
			optimized = append(optimized, stops[:maxGapIndex+1]...)
			// Reverse remaining and append
			remaining := stops[maxGapIndex+1:]
			reverse(remaining)
			optimized = append(optimized, remaining...)
		} else {
			optimized = append(optimized, stops...)
		}
	}

	r.Stops = optimized
}

// optimizeCombined implements a hybrid strategy
func (r *PickRoute) optimizeCombined() {
	// Analyze item density per aisle
	aisleGroups := make(map[string][]RouteStop)
	for _, stop := range r.Stops {
		aisleGroups[stop.Location.Aisle] = append(aisleGroups[stop.Location.Aisle], stop)
	}

	optimized := make([]RouteStop, 0, len(r.Stops))
	aisles := getSortedAisles(aisleGroups)

	for _, aisle := range aisles {
		stops := aisleGroups[aisle]
		sortByRack(stops)

		// Use S-shape for aisles with many items, return for few items
		if len(stops) > 3 {
			optimized = append(optimized, stops...)
		} else {
			// Return strategy - pick and return
			optimized = append(optimized, stops...)
		}
	}

	r.Stops = optimized
}

// optimizeNearest implements nearest neighbor algorithm
func (r *PickRoute) optimizeNearest() {
	if len(r.Stops) <= 1 {
		return
	}

	optimized := make([]RouteStop, 0, len(r.Stops))
	remaining := make([]RouteStop, len(r.Stops))
	copy(remaining, r.Stops)

	// Start from start location
	currentX, currentY := r.StartLocation.X, r.StartLocation.Y

	for len(remaining) > 0 {
		// Find nearest stop
		nearestIdx := 0
		nearestDist := math.MaxFloat64

		for i, stop := range remaining {
			dist := distance(currentX, currentY, stop.Location.X, stop.Location.Y)
			if dist < nearestDist {
				nearestDist = dist
				nearestIdx = i
			}
		}

		// Add nearest stop to route
		nearest := remaining[nearestIdx]
		optimized = append(optimized, nearest)

		// Update current position
		currentX, currentY = nearest.Location.X, nearest.Location.Y

		// Remove from remaining
		remaining = append(remaining[:nearestIdx], remaining[nearestIdx+1:]...)
	}

	r.Stops = optimized
}

// Start marks the route as in progress
func (r *PickRoute) Start(pickerID string) error {
	if r.Status != RouteStatusPending && r.Status != RouteStatusPaused {
		return ErrRouteAlreadyStarted
	}

	now := time.Now()
	r.Status = RouteStatusInProgress
	r.PickerID = pickerID
	r.StartedAt = &now
	r.UpdatedAt = now

	r.AddDomainEvent(&RouteStartedEvent{
		RouteID:   r.RouteID,
		PickerID:  pickerID,
		StartedAt: now,
	})

	return nil
}

// CompleteStop marks a stop as completed
func (r *PickRoute) CompleteStop(stopNumber int, pickedQty int, toteID string) error {
	if r.Status != RouteStatusInProgress {
		return errors.New("route is not in progress")
	}

	for i := range r.Stops {
		if r.Stops[i].StopNumber == stopNumber {
			now := time.Now()
			r.Stops[i].PickedQty = pickedQty
			r.Stops[i].Status = "completed"
			r.Stops[i].ToteID = toteID
			r.Stops[i].PickedAt = &now
			r.PickedItems += pickedQty
			r.UpdatedAt = now

			r.AddDomainEvent(&StopCompletedEvent{
				RouteID:    r.RouteID,
				StopNumber: stopNumber,
				SKU:        r.Stops[i].SKU,
				PickedQty:  pickedQty,
				ToteID:     toteID,
				LocationID: r.Stops[i].Location.LocationID,
				PickedAt:   now,
			})

			// Check if all stops are completed
			allCompleted := true
			for _, stop := range r.Stops {
				if stop.Status != "completed" && stop.Status != "skipped" {
					allCompleted = false
					break
				}
			}

			if allCompleted {
				return r.Complete()
			}

			return nil
		}
	}

	return errors.New("stop not found")
}

// SkipStop marks a stop as skipped
func (r *PickRoute) SkipStop(stopNumber int, reason string) error {
	if r.Status != RouteStatusInProgress {
		return errors.New("route is not in progress")
	}

	for i := range r.Stops {
		if r.Stops[i].StopNumber == stopNumber {
			r.Stops[i].Status = "skipped"
			r.Stops[i].Notes = reason
			r.UpdatedAt = time.Now()
			return nil
		}
	}

	return errors.New("stop not found")
}

// Complete marks the route as completed
func (r *PickRoute) Complete() error {
	if r.Status == RouteStatusCompleted {
		return ErrRouteCompleted
	}

	now := time.Now()
	r.Status = RouteStatusCompleted
	r.CompletedAt = &now
	r.UpdatedAt = now

	// Calculate actual time
	if r.StartedAt != nil {
		r.ActualTime = now.Sub(*r.StartedAt)
	}

	r.AddDomainEvent(&RouteCompletedEvent{
		RouteID:        r.RouteID,
		OrderID:        r.OrderID,
		PickerID:       r.PickerID,
		TotalItems:     r.TotalItems,
		PickedItems:    r.PickedItems,
		ActualDistance: r.ActualDistance,
		ActualTime:     r.ActualTime,
		CompletedAt:    now,
	})

	return nil
}

// Pause pauses the route
func (r *PickRoute) Pause() error {
	if r.Status != RouteStatusInProgress {
		return errors.New("can only pause in-progress routes")
	}

	r.Status = RouteStatusPaused
	r.UpdatedAt = time.Now()
	return nil
}

// Cancel cancels the route
func (r *PickRoute) Cancel(reason string) error {
	if r.Status == RouteStatusCompleted {
		return ErrRouteCompleted
	}

	r.Status = RouteStatusCancelled
	r.UpdatedAt = time.Now()

	r.AddDomainEvent(&RouteCancelledEvent{
		RouteID:     r.RouteID,
		Reason:      reason,
		CancelledAt: time.Now(),
	})

	return nil
}

// GetProgress returns the completion percentage
func (r *PickRoute) GetProgress() float64 {
	if len(r.Stops) == 0 {
		return 0
	}

	completed := 0
	for _, stop := range r.Stops {
		if stop.Status == "completed" || stop.Status == "skipped" {
			completed++
		}
	}

	return float64(completed) / float64(len(r.Stops)) * 100
}

// GetCurrentStop returns the next pending stop
func (r *PickRoute) GetCurrentStop() *RouteStop {
	for i := range r.Stops {
		if r.Stops[i].Status == "pending" {
			return &r.Stops[i]
		}
	}
	return nil
}

// calculateTotalDistance calculates the total route distance
func (r *PickRoute) calculateTotalDistance() float64 {
	if len(r.Stops) == 0 {
		return 0
	}

	totalDistance := 0.0

	// Distance from start to first stop
	totalDistance += distance(
		r.StartLocation.X, r.StartLocation.Y,
		r.Stops[0].Location.X, r.Stops[0].Location.Y,
	)

	// Distance between stops
	for i := 0; i < len(r.Stops)-1; i++ {
		totalDistance += distance(
			r.Stops[i].Location.X, r.Stops[i].Location.Y,
			r.Stops[i+1].Location.X, r.Stops[i+1].Location.Y,
		)
	}

	// Distance from last stop to end
	lastStop := r.Stops[len(r.Stops)-1]
	totalDistance += distance(
		lastStop.Location.X, lastStop.Location.Y,
		r.EndLocation.X, r.EndLocation.Y,
	)

	return totalDistance
}

// calculateEstimatedTime estimates time based on distance and items
func (r *PickRoute) calculateEstimatedTime() time.Duration {
	// Assume walking speed of 1.2 m/s and 10 seconds per pick
	walkingTime := r.EstimatedDistance / 1.2
	pickTime := float64(r.TotalItems) * 10

	return time.Duration(walkingTime+pickTime) * time.Second
}

// AddDomainEvent adds a domain event
func (r *PickRoute) AddDomainEvent(event DomainEvent) {
	r.DomainEvents = append(r.DomainEvents, event)
}

// ClearDomainEvents clears all domain events
func (r *PickRoute) ClearDomainEvents() {
	r.DomainEvents = make([]DomainEvent, 0)
}

// GetDomainEvents returns all domain events
func (r *PickRoute) GetDomainEvents() []DomainEvent {
	return r.DomainEvents
}

// Helper functions

func isValidStrategy(s RoutingStrategy) bool {
	switch s {
	case StrategyReturn, StrategySShape, StrategyLargestGap, StrategyCombined, StrategyNearest:
		return true
	default:
		return false
	}
}

func distance(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	return math.Sqrt(dx*dx + dy*dy)
}

func sortByRack(stops []RouteStop) {
	for i := 0; i < len(stops)-1; i++ {
		for j := i + 1; j < len(stops); j++ {
			if stops[j].Location.Rack < stops[i].Location.Rack {
				stops[i], stops[j] = stops[j], stops[i]
			}
		}
	}
}

func reverse(stops []RouteStop) {
	for i, j := 0, len(stops)-1; i < j; i, j = i+1, j-1 {
		stops[i], stops[j] = stops[j], stops[i]
	}
}

func getSortedAisles(groups map[string][]RouteStop) []string {
	aisles := make([]string, 0, len(groups))
	for aisle := range groups {
		aisles = append(aisles, aisle)
	}
	// Simple string sort
	for i := 0; i < len(aisles)-1; i++ {
		for j := i + 1; j < len(aisles); j++ {
			if aisles[j] < aisles[i] {
				aisles[i], aisles[j] = aisles[j], aisles[i]
			}
		}
	}
	return aisles
}
