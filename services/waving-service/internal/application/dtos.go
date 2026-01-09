package application

import "time"

// WaveDTO represents a wave in responses
type WaveDTO struct {
	WaveID            string                  `json:"waveId"`
	WaveType          string                  `json:"waveType"`
	Status            string                  `json:"status"`
	FulfillmentMode   string                  `json:"fulfillmentMode"`
	Orders            []WaveOrderDTO          `json:"orders"`
	Configuration     WaveConfigurationDTO    `json:"configuration"`
	LaborAllocation   LaborAllocationDTO      `json:"laborAllocation"`
	ScheduledStart    time.Time               `json:"scheduledStart,omitempty"`
	ScheduledEnd      time.Time               `json:"scheduledEnd,omitempty"`
	ActualStart       *time.Time              `json:"actualStart,omitempty"`
	ActualEnd         *time.Time              `json:"actualEnd,omitempty"`
	EstimatedDuration string                  `json:"estimatedDuration,omitempty"`
	Priority          int                     `json:"priority"`
	Zone              string                  `json:"zone"`
	CreatedAt         time.Time               `json:"createdAt"`
	UpdatedAt         time.Time               `json:"updatedAt"`
	ReleasedAt        *time.Time              `json:"releasedAt,omitempty"`
	CompletedAt       *time.Time              `json:"completedAt,omitempty"`
	OrderCount        int                     `json:"orderCount"`
	TotalItems        int                     `json:"totalItems"`
	TotalWeight       float64                 `json:"totalWeight"`
}

// WaveOrderDTO represents an order in a wave
type WaveOrderDTO struct {
	OrderID            string    `json:"orderId"`
	CustomerID         string    `json:"customerId"`
	Priority           string    `json:"priority"`
	ItemCount          int       `json:"itemCount"`
	TotalWeight        float64   `json:"totalWeight"`
	PromisedDeliveryAt time.Time `json:"promisedDeliveryAt"`
	CarrierCutoff      time.Time `json:"carrierCutoff"`
	Zone               string    `json:"zone"`
	Status             string    `json:"status"`
	AddedAt            time.Time `json:"addedAt"`
}

// WaveConfigurationDTO represents wave configuration
type WaveConfigurationDTO struct {
	MaxOrders           int           `json:"maxOrders"`
	MaxItems            int           `json:"maxItems"`
	MaxWeight           float64       `json:"maxWeight"`
	CarrierFilter       []string      `json:"carrierFilter,omitempty"`
	PriorityFilter      []string      `json:"priorityFilter,omitempty"`
	ZoneFilter          []string      `json:"zoneFilter,omitempty"`
	CutoffTime          time.Time     `json:"cutoffTime,omitempty"`
	ReleaseDelay        string        `json:"releaseDelay,omitempty"`
	AutoRelease         bool          `json:"autoRelease"`
	OptimizeForCarrier  bool          `json:"optimizeForCarrier"`
	OptimizeForZone     bool          `json:"optimizeForZone"`
	OptimizeForPriority bool          `json:"optimizeForPriority"`
}

// LaborAllocationDTO represents labor allocation
type LaborAllocationDTO struct {
	PickersRequired   int      `json:"pickersRequired"`
	PickersAssigned   int      `json:"pickersAssigned"`
	PackersRequired   int      `json:"packersRequired"`
	PackersAssigned   int      `json:"packersAssigned"`
	AssignedWorkerIDs []string `json:"assignedWorkerIds,omitempty"`
}

// WaveListDTO represents a simplified wave for list operations
type WaveListDTO struct {
	WaveID          string     `json:"waveId"`
	WaveType        string     `json:"waveType"`
	Status          string     `json:"status"`
	FulfillmentMode string     `json:"fulfillmentMode"`
	OrderCount      int        `json:"orderCount"`
	Priority        int        `json:"priority"`
	Zone            string     `json:"zone"`
	ScheduledStart  time.Time  `json:"scheduledStart,omitempty"`
	ScheduledEnd    time.Time  `json:"scheduledEnd,omitempty"`
	ReleasedAt      *time.Time `json:"releasedAt,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

// CreateWaveFromOrdersRequest is the API request for creating a wave from order IDs
type CreateWaveFromOrdersRequest struct {
	OrderIDs        []string             `json:"orderIds" binding:"required,min=1"`
	WaveType        string               `json:"waveType" binding:"required"`
	FulfillmentMode string               `json:"fulfillmentMode"`
	Zone            string               `json:"zone"`
	Configuration   WaveConfigurationDTO `json:"configuration"`
}

// CreateWaveFromOrdersResponse is the API response for creating a wave from order IDs
type CreateWaveFromOrdersResponse struct {
	Wave         WaveDTO  `json:"wave"`
	FailedOrders []string `json:"failedOrders,omitempty"`
}

// WaveAssignedSignal is the payload for the waveAssigned Temporal signal
type WaveAssignedSignal struct {
	WaveID         string    `json:"waveId"`
	ScheduledStart time.Time `json:"scheduledStart"`
}
