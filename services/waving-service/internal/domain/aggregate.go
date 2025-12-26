package domain

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Errors
var (
	ErrWaveEmpty           = errors.New("wave must contain at least one order")
	ErrWaveAlreadyReleased = errors.New("wave has already been released")
	ErrWaveAlreadyClosed   = errors.New("wave is already closed")
	ErrInvalidWaveType     = errors.New("invalid wave type")
	ErrOrderAlreadyInWave  = errors.New("order is already in this wave")
)

// WaveType represents the type of wave
type WaveType string

const (
	WaveTypeDigital   WaveType = "digital"   // B2C, e-commerce orders
	WaveTypeWholesale WaveType = "wholesale" // B2B, bulk orders
	WaveTypePriority  WaveType = "priority"  // Same-day, next-day orders
	WaveTypeMixed     WaveType = "mixed"     // Combined wave types
)

// WaveStatus represents the status of a wave
type WaveStatus string

const (
	WaveStatusPlanning   WaveStatus = "planning"    // Wave is being planned
	WaveStatusScheduled  WaveStatus = "scheduled"   // Wave is scheduled for release
	WaveStatusReleased   WaveStatus = "released"    // Wave has been released to picking
	WaveStatusInProgress WaveStatus = "in_progress" // Wave is being picked
	WaveStatusCompleted  WaveStatus = "completed"   // All orders in wave are fulfilled
	WaveStatusCancelled  WaveStatus = "cancelled"   // Wave was cancelled
)

// FulfillmentMode represents the fulfillment mode
type FulfillmentMode string

const (
	FulfillmentModeWave       FulfillmentMode = "wave"       // Traditional wave-based
	FulfillmentModeWaveless   FulfillmentMode = "waveless"   // Continuous/real-time
	FulfillmentModeHybrid     FulfillmentMode = "hybrid"     // Mixed mode
)

// Wave is the aggregate root for the Waving bounded context
type Wave struct {
	ID                primitive.ObjectID `bson:"_id,omitempty"`
	WaveID            string             `bson:"waveId"`
	WaveType          WaveType           `bson:"waveType"`
	Status            WaveStatus         `bson:"status"`
	FulfillmentMode   FulfillmentMode    `bson:"fulfillmentMode"`
	Orders            []WaveOrder        `bson:"orders"`
	Configuration     WaveConfiguration  `bson:"configuration"`
	LaborAllocation   LaborAllocation    `bson:"laborAllocation"`
	ScheduledStart    time.Time          `bson:"scheduledStart"`
	ScheduledEnd      time.Time          `bson:"scheduledEnd"`
	ActualStart       *time.Time         `bson:"actualStart,omitempty"`
	ActualEnd         *time.Time         `bson:"actualEnd,omitempty"`
	EstimatedDuration time.Duration      `bson:"estimatedDuration"`
	Priority          int                `bson:"priority"` // 1 = highest
	Zone              string             `bson:"zone"`     // Warehouse zone
	CreatedAt         time.Time          `bson:"createdAt"`
	UpdatedAt         time.Time          `bson:"updatedAt"`
	ReleasedAt        *time.Time         `bson:"releasedAt,omitempty"`
	CompletedAt       *time.Time         `bson:"completedAt,omitempty"`
	DomainEvents      []DomainEvent      `bson:"-"` // Transient
}

// WaveOrder represents an order within a wave
type WaveOrder struct {
	OrderID            string    `bson:"orderId"`
	CustomerID         string    `bson:"customerId"`
	Priority           string    `bson:"priority"`
	ItemCount          int       `bson:"itemCount"`
	TotalWeight        float64   `bson:"totalWeight"`
	PromisedDeliveryAt time.Time `bson:"promisedDeliveryAt"`
	CarrierCutoff      time.Time `bson:"carrierCutoff"`
	Zone               string    `bson:"zone"`
	Status             string    `bson:"status"` // pending, picking, completed
	AddedAt            time.Time `bson:"addedAt"`
}

// WaveConfiguration holds the wave planning parameters
type WaveConfiguration struct {
	MaxOrders           int           `bson:"maxOrders"`
	MaxItems            int           `bson:"maxItems"`
	MaxWeight           float64       `bson:"maxWeight"`
	CarrierFilter       []string      `bson:"carrierFilter,omitempty"`
	PriorityFilter      []string      `bson:"priorityFilter,omitempty"`
	ZoneFilter          []string      `bson:"zoneFilter,omitempty"`
	CutoffTime          time.Time     `bson:"cutoffTime"`
	ReleaseDelay        time.Duration `bson:"releaseDelay"`
	AutoRelease         bool          `bson:"autoRelease"`
	OptimizeForCarrier  bool          `bson:"optimizeForCarrier"`
	OptimizeForZone     bool          `bson:"optimizeForZone"`
	OptimizeForPriority bool          `bson:"optimizeForPriority"`
}

// LaborAllocation represents the labor assigned to a wave
type LaborAllocation struct {
	PickersRequired   int      `bson:"pickersRequired"`
	PickersAssigned   int      `bson:"pickersAssigned"`
	PackersRequired   int      `bson:"packersRequired"`
	PackersAssigned   int      `bson:"packersAssigned"`
	AssignedWorkerIDs []string `bson:"assignedWorkerIds,omitempty"`
}

// NewWave creates a new Wave aggregate
func NewWave(waveID string, waveType WaveType, mode FulfillmentMode, config WaveConfiguration) (*Wave, error) {
	if waveType != WaveTypeDigital && waveType != WaveTypeWholesale &&
		waveType != WaveTypePriority && waveType != WaveTypeMixed {
		return nil, ErrInvalidWaveType
	}

	now := time.Now()
	wave := &Wave{
		WaveID:          waveID,
		WaveType:        waveType,
		Status:          WaveStatusPlanning,
		FulfillmentMode: mode,
		Orders:          make([]WaveOrder, 0),
		Configuration:   config,
		LaborAllocation: LaborAllocation{},
		Priority:        5, // Default medium priority
		CreatedAt:       now,
		UpdatedAt:       now,
		DomainEvents:    make([]DomainEvent, 0),
	}

	wave.AddDomainEvent(&WaveCreatedEvent{
		WaveID:          waveID,
		WaveType:        string(waveType),
		FulfillmentMode: string(mode),
		CreatedAt:       now,
	})

	return wave, nil
}

// AddOrder adds an order to the wave
func (w *Wave) AddOrder(order WaveOrder) error {
	if w.Status == WaveStatusReleased || w.Status == WaveStatusInProgress ||
		w.Status == WaveStatusCompleted || w.Status == WaveStatusCancelled {
		return ErrWaveAlreadyReleased
	}

	// Check if order already exists
	for _, o := range w.Orders {
		if o.OrderID == order.OrderID {
			return ErrOrderAlreadyInWave
		}
	}

	// Check capacity constraints
	if w.Configuration.MaxOrders > 0 && len(w.Orders) >= w.Configuration.MaxOrders {
		return errors.New("wave has reached maximum order capacity")
	}

	order.AddedAt = time.Now()
	order.Status = "pending"
	w.Orders = append(w.Orders, order)
	w.UpdatedAt = time.Now()

	w.AddDomainEvent(&OrderAddedToWaveEvent{
		WaveID:  w.WaveID,
		OrderID: order.OrderID,
		AddedAt: order.AddedAt,
	})

	return nil
}

// RemoveOrder removes an order from the wave
func (w *Wave) RemoveOrder(orderID string) error {
	if w.Status == WaveStatusReleased || w.Status == WaveStatusInProgress {
		return ErrWaveAlreadyReleased
	}

	for i, o := range w.Orders {
		if o.OrderID == orderID {
			w.Orders = append(w.Orders[:i], w.Orders[i+1:]...)
			w.UpdatedAt = time.Now()

			w.AddDomainEvent(&OrderRemovedFromWaveEvent{
				WaveID:    w.WaveID,
				OrderID:   orderID,
				RemovedAt: time.Now(),
			})
			return nil
		}
	}

	return errors.New("order not found in wave")
}

// Schedule schedules the wave for release
func (w *Wave) Schedule(startTime, endTime time.Time) error {
	if w.Status != WaveStatusPlanning {
		return errors.New("wave can only be scheduled from planning status")
	}

	if len(w.Orders) == 0 {
		return ErrWaveEmpty
	}

	w.ScheduledStart = startTime
	w.ScheduledEnd = endTime
	w.EstimatedDuration = endTime.Sub(startTime)
	w.Status = WaveStatusScheduled
	w.UpdatedAt = time.Now()

	w.AddDomainEvent(&WaveScheduledEvent{
		WaveID:         w.WaveID,
		ScheduledStart: startTime,
		ScheduledEnd:   endTime,
	})

	return nil
}

// Release releases the wave to picking
func (w *Wave) Release() error {
	if w.Status != WaveStatusScheduled && w.Status != WaveStatusPlanning {
		return ErrWaveAlreadyReleased
	}

	if len(w.Orders) == 0 {
		return ErrWaveEmpty
	}

	now := time.Now()
	w.Status = WaveStatusReleased
	w.ReleasedAt = &now
	w.ActualStart = &now
	w.UpdatedAt = now

	// Update all orders to picking status
	for i := range w.Orders {
		w.Orders[i].Status = "picking"
	}

	orderIDs := make([]string, len(w.Orders))
	for i, o := range w.Orders {
		orderIDs[i] = o.OrderID
	}

	w.AddDomainEvent(&WaveReleasedEvent{
		WaveID:            w.WaveID,
		OrderIDs:          orderIDs,
		ReleasedAt:        now,
		EstimatedDuration: w.EstimatedDuration,
	})

	return nil
}

// StartProgress marks the wave as in progress
func (w *Wave) StartProgress() error {
	if w.Status != WaveStatusReleased {
		return errors.New("wave must be released before starting progress")
	}

	now := time.Now()
	w.Status = WaveStatusInProgress
	if w.ActualStart == nil {
		w.ActualStart = &now
	}
	w.UpdatedAt = now

	return nil
}

// CompleteOrder marks an order in the wave as completed
func (w *Wave) CompleteOrder(orderID string) error {
	for i, o := range w.Orders {
		if o.OrderID == orderID {
			w.Orders[i].Status = "completed"
			w.UpdatedAt = time.Now()

			// Check if all orders are completed
			allCompleted := true
			for _, order := range w.Orders {
				if order.Status != "completed" {
					allCompleted = false
					break
				}
			}

			if allCompleted {
				w.Complete()
			}

			return nil
		}
	}

	return errors.New("order not found in wave")
}

// Complete marks the wave as completed
func (w *Wave) Complete() error {
	if w.Status == WaveStatusCompleted {
		return ErrWaveAlreadyClosed
	}

	now := time.Now()
	w.Status = WaveStatusCompleted
	w.ActualEnd = &now
	w.CompletedAt = &now
	w.UpdatedAt = now

	w.AddDomainEvent(&WaveCompletedEvent{
		WaveID:       w.WaveID,
		CompletedAt:  now,
		OrderCount:   len(w.Orders),
		ActualStart:  w.ActualStart,
		ActualEnd:    &now,
	})

	return nil
}

// Cancel cancels the wave
func (w *Wave) Cancel(reason string) error {
	if w.Status == WaveStatusCompleted {
		return ErrWaveAlreadyClosed
	}

	w.Status = WaveStatusCancelled
	w.UpdatedAt = time.Now()

	orderIDs := make([]string, len(w.Orders))
	for i, o := range w.Orders {
		orderIDs[i] = o.OrderID
	}

	w.AddDomainEvent(&WaveCancelledEvent{
		WaveID:      w.WaveID,
		Reason:      reason,
		OrderIDs:    orderIDs,
		CancelledAt: time.Now(),
	})

	return nil
}

// AllocateLabor sets the labor allocation for the wave
func (w *Wave) AllocateLabor(allocation LaborAllocation) {
	w.LaborAllocation = allocation
	w.UpdatedAt = time.Now()
}

// SetPriority sets the wave priority
func (w *Wave) SetPriority(priority int) {
	w.Priority = priority
	w.UpdatedAt = time.Now()
}

// SetZone sets the warehouse zone for the wave
func (w *Wave) SetZone(zone string) {
	w.Zone = zone
	w.UpdatedAt = time.Now()
}

// GetOrderCount returns the number of orders in the wave
func (w *Wave) GetOrderCount() int {
	return len(w.Orders)
}

// GetTotalItems returns the total number of items across all orders
func (w *Wave) GetTotalItems() int {
	total := 0
	for _, o := range w.Orders {
		total += o.ItemCount
	}
	return total
}

// GetTotalWeight returns the total weight of all orders
func (w *Wave) GetTotalWeight() float64 {
	total := 0.0
	for _, o := range w.Orders {
		total += o.TotalWeight
	}
	return total
}

// GetCompletedOrderCount returns the number of completed orders
func (w *Wave) GetCompletedOrderCount() int {
	count := 0
	for _, o := range w.Orders {
		if o.Status == "completed" {
			count++
		}
	}
	return count
}

// GetProgress returns the completion percentage
func (w *Wave) GetProgress() float64 {
	if len(w.Orders) == 0 {
		return 0
	}
	return float64(w.GetCompletedOrderCount()) / float64(len(w.Orders)) * 100
}

// AddDomainEvent adds a domain event
func (w *Wave) AddDomainEvent(event DomainEvent) {
	w.DomainEvents = append(w.DomainEvents, event)
}

// ClearDomainEvents clears all domain events
func (w *Wave) ClearDomainEvents() {
	w.DomainEvents = make([]DomainEvent, 0)
}

// GetDomainEvents returns all domain events
func (w *Wave) GetDomainEvents() []DomainEvent {
	return w.DomainEvents
}
