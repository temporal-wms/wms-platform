package application

import "github.com/wms-platform/consolidation-service/internal/domain"

// CreateConsolidationCommand represents the command to create a new consolidation unit
type CreateConsolidationCommand struct {
	ConsolidationID string
	OrderID         string
	WaveID          string
	Strategy        string
	Items           []domain.ExpectedItem
}

// AssignStationCommand represents the command to assign a station to consolidation
type AssignStationCommand struct {
	ConsolidationID string
	Station         string
	WorkerID        string
	DestinationBin  string
}

// ConsolidateItemCommand represents the command to consolidate an item
type ConsolidateItemCommand struct {
	ConsolidationID string
	SKU             string
	Quantity        int
	SourceToteID    string
	VerifiedBy      string
}

// CompleteConsolidationCommand represents the command to complete consolidation
type CompleteConsolidationCommand struct {
	ConsolidationID string
}

// GetConsolidationQuery represents the query to get a consolidation by ID
type GetConsolidationQuery struct {
	ConsolidationID string
}

// GetByOrderQuery represents the query to get consolidation by order ID
type GetByOrderQuery struct {
	OrderID string
}

// GetByWaveQuery represents the query to get consolidations by wave ID
type GetByWaveQuery struct {
	WaveID string
}

// GetByStationQuery represents the query to get consolidations by station
type GetByStationQuery struct {
	Station string
}

// GetPendingQuery represents the query to get pending consolidations
type GetPendingQuery struct {
	Limit int
}
