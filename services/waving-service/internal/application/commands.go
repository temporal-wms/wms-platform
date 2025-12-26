package application

import (
	"time"

	"github.com/wms-platform/waving-service/internal/domain"
)

// CreateWaveCommand represents the command to create a new wave
type CreateWaveCommand struct {
	WaveType        string
	FulfillmentMode string
	Zone            string
	Configuration   domain.WaveConfiguration
}

// UpdateWaveCommand represents the command to update a wave
type UpdateWaveCommand struct {
	WaveID   string
	Priority *int
	Zone     *string
}

// AddOrderToWaveCommand represents the command to add an order to a wave
type AddOrderToWaveCommand struct {
	WaveID string
	Order  domain.WaveOrder
}

// RemoveOrderFromWaveCommand represents the command to remove an order from a wave
type RemoveOrderFromWaveCommand struct {
	WaveID  string
	OrderID string
}

// ScheduleWaveCommand represents the command to schedule a wave
type ScheduleWaveCommand struct {
	WaveID         string
	ScheduledStart time.Time
	ScheduledEnd   time.Time
}

// ReleaseWaveCommand represents the command to release a wave
type ReleaseWaveCommand struct {
	WaveID string
}

// CancelWaveCommand represents the command to cancel a wave
type CancelWaveCommand struct {
	WaveID string
	Reason string
}

// DeleteWaveCommand represents the command to delete a wave
type DeleteWaveCommand struct {
	WaveID string
}

// GetWaveQuery represents the query to get a wave by ID
type GetWaveQuery struct {
	WaveID string
}

// GetWavesByStatusQuery represents the query to get waves by status
type GetWavesByStatusQuery struct {
	Status string
}

// GetWavesByZoneQuery represents the query to get waves by zone
type GetWavesByZoneQuery struct {
	Zone string
}

// GetWaveByOrderQuery represents the query to get a wave by order ID
type GetWaveByOrderQuery struct {
	OrderID string
}
