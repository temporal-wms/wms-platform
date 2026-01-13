package domain

import (
	"context"
	"time"
)

// WaveRepository defines the interface for wave persistence
type WaveRepository interface {
	// Save persists a wave (create or update)
	Save(ctx context.Context, wave *Wave) error

	// FindByID retrieves a wave by its ID
	FindByID(ctx context.Context, waveID string) (*Wave, error)

	// FindByStatus retrieves waves by status
	FindByStatus(ctx context.Context, status WaveStatus) ([]*Wave, error)

	// FindByType retrieves waves by type
	FindByType(ctx context.Context, waveType WaveType) ([]*Wave, error)

	// FindByZone retrieves waves by warehouse zone
	FindByZone(ctx context.Context, zone string) ([]*Wave, error)

	// FindScheduledBefore retrieves waves scheduled before a given time
	FindScheduledBefore(ctx context.Context, before time.Time) ([]*Wave, error)

	// FindReadyForRelease retrieves waves that are ready to be released
	FindReadyForRelease(ctx context.Context) ([]*Wave, error)

	// FindByOrderID retrieves the wave containing a specific order
	FindByOrderID(ctx context.Context, orderID string) (*Wave, error)

	// FindActive retrieves all active waves (not completed or cancelled)
	FindActive(ctx context.Context) ([]*Wave, error)

	// FindByDateRange retrieves waves created within a date range
	FindByDateRange(ctx context.Context, start, end time.Time) ([]*Wave, error)

	// Delete removes a wave
	Delete(ctx context.Context, waveID string) error

	// Count returns the total number of waves matching a status
	Count(ctx context.Context, status WaveStatus) (int64, error)
}

// WavePlanner defines the interface for wave planning algorithms
type WavePlanner interface {
	// PlanWave creates an optimized wave from available orders
	PlanWave(ctx context.Context, config WavePlanningConfig) (*Wave, error)

	// OptimizeWave optimizes an existing wave
	OptimizeWave(ctx context.Context, wave *Wave) (*Wave, error)

	// SuggestOrders suggests orders to add to a wave
	SuggestOrders(ctx context.Context, wave *Wave, limit int) ([]WaveOrder, error)
}

// WavePlanningConfig holds configuration for wave planning
type WavePlanningConfig struct {
	WaveType        WaveType        `json:"waveType"`
	FulfillmentMode FulfillmentMode `json:"fulfillmentMode"`
	MaxOrders       int             `json:"maxOrders"`
	MaxItems        int             `json:"maxItems"`
	MaxWeight       float64         `json:"maxWeight"`
	Zone            string          `json:"zone,omitempty"`
	CarrierFilter   []string        `json:"carrierFilter,omitempty"`
	PriorityFilter  []string        `json:"priorityFilter,omitempty"`
	CutoffTime      time.Time       `json:"cutoffTime"`
	// Process Path Filters
	RequiredProcessPaths  []string `json:"requiredProcessPaths,omitempty"`  // Specific process path requirements to include
	ExcludedProcessPaths  []string `json:"excludedProcessPaths,omitempty"`  // Process path requirements to exclude
	SpecialHandlingFilter []string `json:"specialHandlingFilter,omitempty"` // Filter by special handling types
	GroupByProcessPath    bool     `json:"groupByProcessPath"`              // Whether to group orders by process path compatibility
}

// EventPublisher defines the interface for publishing domain events
type EventPublisher interface {
	// Publish publishes a domain event
	Publish(ctx context.Context, event DomainEvent) error

	// PublishAll publishes multiple domain events
	PublishAll(ctx context.Context, events []DomainEvent) error
}

// OrderService defines the interface for interacting with the Order service
type OrderService interface {
	// GetOrdersReadyForWaving retrieves orders ready to be waved
	GetOrdersReadyForWaving(ctx context.Context, filter OrderFilter) ([]WaveOrder, error)

	// NotifyWaveAssignment notifies the order service of wave assignment
	NotifyWaveAssignment(ctx context.Context, orderID, waveID string, scheduledStart time.Time) error
}

// OrderFilter defines criteria for filtering orders
type OrderFilter struct {
	Priority                []string  `json:"priority,omitempty"`
	Zone                    []string  `json:"zone,omitempty"`
	Carrier                 []string  `json:"carrier,omitempty"`
	MinItems                int       `json:"minItems,omitempty"`
	MaxItems                int       `json:"maxItems,omitempty"`
	CutoffBefore            time.Time `json:"cutoffBefore,omitempty"`
	Limit                   int       `json:"limit,omitempty"`
	// Process Path Filters
	ProcessPathRequirements []string `json:"processPathRequirements,omitempty"` // Filter by process path requirements
	SpecialHandling         []string `json:"specialHandling,omitempty"`         // Filter by special handling types
	ExcludeProcessPaths     []string `json:"excludeProcessPaths,omitempty"`     // Exclude orders with these process paths
}
