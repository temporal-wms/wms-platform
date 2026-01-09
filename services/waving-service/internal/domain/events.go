package domain

import "time"

// DomainEvent is the interface for all domain events
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// WaveCreatedEvent is published when a new wave is created
type WaveCreatedEvent struct {
	WaveID          string    `json:"waveId"`
	WaveType        string    `json:"waveType"`
	FulfillmentMode string    `json:"fulfillmentMode"`
	CreatedAt       time.Time `json:"createdAt"`
}

func (e *WaveCreatedEvent) EventType() string   { return "wms.wave.created" }
func (e *WaveCreatedEvent) OccurredAt() time.Time { return e.CreatedAt }

// WaveScheduledEvent is published when a wave is scheduled
type WaveScheduledEvent struct {
	WaveID         string    `json:"waveId"`
	ScheduledStart time.Time `json:"scheduledStart"`
	ScheduledEnd   time.Time `json:"scheduledEnd"`
}

func (e *WaveScheduledEvent) EventType() string   { return "wms.wave.scheduled" }
func (e *WaveScheduledEvent) OccurredAt() time.Time { return e.ScheduledStart }

// WaveReleasedEvent is published when a wave is released to picking
type WaveReleasedEvent struct {
	WaveID            string        `json:"waveId"`
	OrderIDs          []string      `json:"orderIds"`
	ReleasedAt        time.Time     `json:"releasedAt"`
	EstimatedDuration time.Duration `json:"estimatedDuration"`
}

func (e *WaveReleasedEvent) EventType() string   { return "wms.wave.released" }
func (e *WaveReleasedEvent) OccurredAt() time.Time { return e.ReleasedAt }

// WaveCompletedEvent is published when all orders in a wave are fulfilled
type WaveCompletedEvent struct {
	WaveID      string     `json:"waveId"`
	CompletedAt time.Time  `json:"completedAt"`
	OrderCount  int        `json:"orderCount"`
	ActualStart *time.Time `json:"actualStart,omitempty"`
	ActualEnd   *time.Time `json:"actualEnd,omitempty"`
}

func (e *WaveCompletedEvent) EventType() string   { return "wms.wave.completed" }
func (e *WaveCompletedEvent) OccurredAt() time.Time { return e.CompletedAt }

// WaveCancelledEvent is published when a wave is cancelled
type WaveCancelledEvent struct {
	WaveID      string    `json:"waveId"`
	Reason      string    `json:"reason"`
	OrderIDs    []string  `json:"orderIds"`
	CancelledAt time.Time `json:"cancelledAt"`
}

func (e *WaveCancelledEvent) EventType() string   { return "wms.wave.cancelled" }
func (e *WaveCancelledEvent) OccurredAt() time.Time { return e.CancelledAt }

// OrderAddedToWaveEvent is published when an order is added to a wave
type OrderAddedToWaveEvent struct {
	WaveID  string    `json:"waveId"`
	OrderID string    `json:"orderId"`
	AddedAt time.Time `json:"addedAt"`
}

func (e *OrderAddedToWaveEvent) EventType() string   { return "wms.wave.order-added" }
func (e *OrderAddedToWaveEvent) OccurredAt() time.Time { return e.AddedAt }

// OrderRemovedFromWaveEvent is published when an order is removed from a wave
type OrderRemovedFromWaveEvent struct {
	WaveID    string    `json:"waveId"`
	OrderID   string    `json:"orderId"`
	RemovedAt time.Time `json:"removedAt"`
}

func (e *OrderRemovedFromWaveEvent) EventType() string   { return "wms.wave.order-removed" }
func (e *OrderRemovedFromWaveEvent) OccurredAt() time.Time { return e.RemovedAt }

// WaveOptimizedEvent is published when a wave is optimized
type WaveOptimizedEvent struct {
	WaveID             string    `json:"waveId"`
	OptimizationType   string    `json:"optimizationType"`
	OrdersReorganized  int       `json:"ordersReorganized"`
	EstimatedSavings   float64   `json:"estimatedSavings"` // in minutes
	OptimizedAt        time.Time `json:"optimizedAt"`
}

func (e *WaveOptimizedEvent) EventType() string   { return "wms.wave.optimized" }
func (e *WaveOptimizedEvent) OccurredAt() time.Time { return e.OptimizedAt }
