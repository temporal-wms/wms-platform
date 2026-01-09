package domain

import (
	"context"
	"time"
)

// RouteRepository defines the interface for route persistence
type RouteRepository interface {
	// Save persists a route (create or update)
	Save(ctx context.Context, route *PickRoute) error

	// FindByID retrieves a route by its ID
	FindByID(ctx context.Context, routeID string) (*PickRoute, error)

	// FindByOrderID retrieves routes for an order
	FindByOrderID(ctx context.Context, orderID string) ([]*PickRoute, error)

	// FindByWaveID retrieves routes for a wave
	FindByWaveID(ctx context.Context, waveID string) ([]*PickRoute, error)

	// FindByPickerID retrieves routes assigned to a picker
	FindByPickerID(ctx context.Context, pickerID string) ([]*PickRoute, error)

	// FindByStatus retrieves routes by status
	FindByStatus(ctx context.Context, status RouteStatus) ([]*PickRoute, error)

	// FindByZone retrieves routes for a zone
	FindByZone(ctx context.Context, zone string) ([]*PickRoute, error)

	// FindActiveByPicker retrieves active routes for a picker
	FindActiveByPicker(ctx context.Context, pickerID string) (*PickRoute, error)

	// FindPendingRoutes retrieves pending routes ready for assignment
	FindPendingRoutes(ctx context.Context, zone string, limit int) ([]*PickRoute, error)

	// Delete removes a route
	Delete(ctx context.Context, routeID string) error

	// CountByStatus counts routes by status
	CountByStatus(ctx context.Context, status RouteStatus) (int64, error)
}

// RouteCalculator defines the interface for route calculation
type RouteCalculator interface {
	// CalculateRoute calculates an optimized route for given items
	CalculateRoute(ctx context.Context, request RouteRequest) (*PickRoute, error)

	// RecalculateRoute recalculates an existing route (e.g., after skip)
	RecalculateRoute(ctx context.Context, route *PickRoute) (*PickRoute, error)

	// SuggestStrategy suggests the best routing strategy for given items
	SuggestStrategy(ctx context.Context, items []RouteItem) (RoutingStrategy, error)
}

// RouteRequest represents a request to calculate a route
type RouteRequest struct {
	OrderID       string          `json:"orderId"`
	WaveID        string          `json:"waveId"`
	Items         []RouteItem     `json:"items"`
	Strategy      RoutingStrategy `json:"strategy,omitempty"`
	StartLocation Location        `json:"startLocation"`
	EndLocation   Location        `json:"endLocation"`
	Zone          string          `json:"zone"`
}

// WarehouseLayout provides warehouse configuration for routing
type WarehouseLayout interface {
	// GetLocation retrieves location details by ID
	GetLocation(ctx context.Context, locationID string) (*Location, error)

	// GetAisleLocations retrieves all locations in an aisle
	GetAisleLocations(ctx context.Context, aisle string) ([]Location, error)

	// GetZoneLocations retrieves all locations in a zone
	GetZoneLocations(ctx context.Context, zone string) ([]Location, error)

	// GetPickStartLocation retrieves the default pick start location
	GetPickStartLocation(ctx context.Context, zone string) Location

	// GetConsolidationLocation retrieves the consolidation area location
	GetConsolidationLocation(ctx context.Context, zone string) Location

	// GetDistance calculates distance between two locations
	GetDistance(ctx context.Context, from, to Location) float64
}

// InventoryLocator provides inventory location information
type InventoryLocator interface {
	// GetItemLocations retrieves all locations for an SKU
	GetItemLocations(ctx context.Context, sku string) ([]ItemLocation, error)

	// GetBestLocation retrieves the best location to pick from
	GetBestLocation(ctx context.Context, sku string, quantity int, zone string) (*ItemLocation, error)
}

// ItemLocation represents an item's location in the warehouse
type ItemLocation struct {
	SKU            string   `json:"sku"`
	Location       Location `json:"location"`
	QuantityOnHand int      `json:"quantityOnHand"`
	Reserved       int      `json:"reserved"`
	Available      int      `json:"available"`
}

// EventPublisher defines the interface for publishing domain events
type EventPublisher interface {
	// Publish publishes a domain event
	Publish(ctx context.Context, event DomainEvent) error

	// PublishAll publishes multiple domain events
	PublishAll(ctx context.Context, events []DomainEvent) error
}

// RouteMetrics provides route performance metrics
type RouteMetrics struct {
	TotalRoutes        int64         `json:"totalRoutes"`
	CompletedRoutes    int64         `json:"completedRoutes"`
	AvgDistance        float64       `json:"avgDistance"`
	AvgTime            time.Duration `json:"avgTime"`
	AvgItemsPerRoute   float64       `json:"avgItemsPerRoute"`
	StrategyBreakdown  map[string]int64 `json:"strategyBreakdown"`
}
