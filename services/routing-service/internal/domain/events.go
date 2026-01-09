package domain

import "time"

// DomainEvent is the interface for all domain events
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// RouteCreatedEvent is published when a new route is created
type RouteCreatedEvent struct {
	RouteID   string    `json:"routeId"`
	OrderID   string    `json:"orderId"`
	WaveID    string    `json:"waveId"`
	Strategy  string    `json:"strategy"`
	StopCount int       `json:"stopCount"`
	CreatedAt time.Time `json:"createdAt"`
}

func (e *RouteCreatedEvent) EventType() string    { return "wms.routing.route-created" }
func (e *RouteCreatedEvent) OccurredAt() time.Time { return e.CreatedAt }

// RouteOptimizedEvent is published when a route is optimized
type RouteOptimizedEvent struct {
	RouteID           string        `json:"routeId"`
	Strategy          string        `json:"strategy"`
	EstimatedDistance float64       `json:"estimatedDistance"`
	EstimatedTime     time.Duration `json:"estimatedTime"`
	OptimizedAt       time.Time     `json:"optimizedAt"`
}

func (e *RouteOptimizedEvent) EventType() string    { return "wms.routing.route-optimized" }
func (e *RouteOptimizedEvent) OccurredAt() time.Time { return e.OptimizedAt }

// RouteStartedEvent is published when picking on a route begins
type RouteStartedEvent struct {
	RouteID   string    `json:"routeId"`
	PickerID  string    `json:"pickerId"`
	StartedAt time.Time `json:"startedAt"`
}

func (e *RouteStartedEvent) EventType() string    { return "wms.routing.route-started" }
func (e *RouteStartedEvent) OccurredAt() time.Time { return e.StartedAt }

// StopCompletedEvent is published when a stop is completed
type StopCompletedEvent struct {
	RouteID    string    `json:"routeId"`
	StopNumber int       `json:"stopNumber"`
	SKU        string    `json:"sku"`
	PickedQty  int       `json:"pickedQty"`
	ToteID     string    `json:"toteId"`
	LocationID string    `json:"locationId"`
	PickedAt   time.Time `json:"pickedAt"`
}

func (e *StopCompletedEvent) EventType() string    { return "wms.routing.stop-completed" }
func (e *StopCompletedEvent) OccurredAt() time.Time { return e.PickedAt }

// RouteCompletedEvent is published when a route is completed
type RouteCompletedEvent struct {
	RouteID        string        `json:"routeId"`
	OrderID        string        `json:"orderId"`
	PickerID       string        `json:"pickerId"`
	TotalItems     int           `json:"totalItems"`
	PickedItems    int           `json:"pickedItems"`
	ActualDistance float64       `json:"actualDistance"`
	ActualTime     time.Duration `json:"actualTime"`
	CompletedAt    time.Time     `json:"completedAt"`
}

func (e *RouteCompletedEvent) EventType() string    { return "wms.routing.route-completed" }
func (e *RouteCompletedEvent) OccurredAt() time.Time { return e.CompletedAt }

// RouteCancelledEvent is published when a route is cancelled
type RouteCancelledEvent struct {
	RouteID     string    `json:"routeId"`
	Reason      string    `json:"reason"`
	CancelledAt time.Time `json:"cancelledAt"`
}

func (e *RouteCancelledEvent) EventType() string    { return "wms.routing.route-cancelled" }
func (e *RouteCancelledEvent) OccurredAt() time.Time { return e.CancelledAt }

// RouteRecalculatedEvent is published when a route is recalculated
type RouteRecalculatedEvent struct {
	RouteID       string    `json:"routeId"`
	Reason        string    `json:"reason"`
	OldDistance   float64   `json:"oldDistance"`
	NewDistance   float64   `json:"newDistance"`
	RecalculatedAt time.Time `json:"recalculatedAt"`
}

func (e *RouteRecalculatedEvent) EventType() string    { return "wms.routing.route-recalculated" }
func (e *RouteRecalculatedEvent) OccurredAt() time.Time { return e.RecalculatedAt }
