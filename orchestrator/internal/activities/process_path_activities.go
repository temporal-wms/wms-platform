package activities

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/wms-platform/orchestrator/internal/activities/clients"
	"go.temporal.io/sdk/activity"
)

// ProcessPathInput represents the input for determining process path
type ProcessPathInput struct {
	OrderID          string               `json:"orderId"`
	Items            []ProcessPathItem    `json:"items"`
	GiftWrap         bool                 `json:"giftWrap"`
	GiftWrapDetails  *clients.GiftWrapDetails  `json:"giftWrapDetails,omitempty"`
	HazmatDetails    *clients.HazmatDetails    `json:"hazmatDetails,omitempty"`
	ColdChainDetails *clients.ColdChainDetails `json:"coldChainDetails,omitempty"`
	TotalValue       float64              `json:"totalValue"`
}

// ProcessPathItem represents an item for process path determination
type ProcessPathItem struct {
	SKU              string  `json:"sku"`
	Quantity         int     `json:"quantity"`
	Weight           float64 `json:"weight"`
	IsFragile        bool    `json:"isFragile"`
	IsHazmat         bool    `json:"isHazmat"`
	RequiresColdChain bool   `json:"requiresColdChain"`
}

// FindCapableStationInput represents input for finding a capable station
type FindCapableStationInput struct {
	Requirements []string `json:"requirements"`
	StationType  string   `json:"stationType"`
	Zone         string   `json:"zone,omitempty"`
}

// HighValueThreshold is the threshold for high-value orders
const HighValueThreshold = 500.0

// OversizedWeightThreshold is the threshold weight for oversized items (in kg)
const OversizedWeightThreshold = 30.0

// DetermineProcessPath analyzes order characteristics and determines the process path
func (a *ProcessPathActivities) DetermineProcessPath(ctx context.Context, input ProcessPathInput) (*clients.ProcessPath, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Determining process path", "orderId", input.OrderID)

	path := &clients.ProcessPath{
		PathID:          uuid.New().String(),
		OrderID:         input.OrderID,
		Requirements:    make([]clients.ProcessRequirement, 0),
		SpecialHandling: make([]string, 0),
	}

	// Determine single vs multi-item
	totalItems := 0
	for _, item := range input.Items {
		totalItems += item.Quantity
	}

	if totalItems == 1 && len(input.Items) == 1 {
		path.Requirements = append(path.Requirements, clients.RequirementSingleItem)
		path.ConsolidationRequired = false
	} else {
		path.Requirements = append(path.Requirements, clients.RequirementMultiItem)
		path.ConsolidationRequired = true
	}

	// Check for gift wrap
	if input.GiftWrap {
		path.Requirements = append(path.Requirements, clients.RequirementGiftWrap)
		path.GiftWrapRequired = true
	}

	// Check for high value
	if input.TotalValue >= HighValueThreshold {
		path.Requirements = append(path.Requirements, clients.RequirementHighValue)
		path.SpecialHandling = append(path.SpecialHandling, "high_value_verification")
	}

	// Check for fragile items
	hasFragile := false
	for _, item := range input.Items {
		if item.IsFragile {
			hasFragile = true
			break
		}
	}
	if hasFragile {
		path.Requirements = append(path.Requirements, clients.RequirementFragile)
		path.SpecialHandling = append(path.SpecialHandling, "fragile_packing")
	}

	// Check for oversized items
	hasOversized := false
	for _, item := range input.Items {
		if item.Weight >= OversizedWeightThreshold {
			hasOversized = true
			break
		}
	}
	if hasOversized {
		path.Requirements = append(path.Requirements, clients.RequirementOversized)
		path.SpecialHandling = append(path.SpecialHandling, "oversized_handling")
	}

	// Check for hazmat items
	hasHazmat := false
	for _, item := range input.Items {
		if item.IsHazmat {
			hasHazmat = true
			break
		}
	}
	if hasHazmat || input.HazmatDetails != nil {
		path.Requirements = append(path.Requirements, clients.RequirementHazmat)
		path.SpecialHandling = append(path.SpecialHandling, "hazmat_compliance")
	}

	// Check for cold chain items
	hasColdChain := false
	for _, item := range input.Items {
		if item.RequiresColdChain {
			hasColdChain = true
			break
		}
	}
	if hasColdChain || input.ColdChainDetails != nil {
		path.Requirements = append(path.Requirements, clients.RequirementColdChain)
		path.SpecialHandling = append(path.SpecialHandling, "cold_chain_packaging")
	}

	logger.Info("Process path determined",
		"orderId", input.OrderID,
		"pathId", path.PathID,
		"requirements", path.Requirements,
		"consolidationRequired", path.ConsolidationRequired,
		"giftWrapRequired", path.GiftWrapRequired,
		"specialHandling", path.SpecialHandling,
	)

	return path, nil
}

// FindCapableStation finds a station that has all required capabilities
func (a *ProcessPathActivities) FindCapableStation(ctx context.Context, input FindCapableStationInput) (*clients.Station, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Finding capable station",
		"requirements", input.Requirements,
		"stationType", input.StationType,
		"zone", input.Zone,
	)

	req := &clients.FindCapableStationsRequest{
		Requirements: input.Requirements,
		StationType:  input.StationType,
		Zone:         input.Zone,
	}

	stations, err := a.clients.FindCapableStations(ctx, req)
	if err != nil {
		logger.Error("Failed to find capable stations", "error", err)
		return nil, fmt.Errorf("failed to find capable stations: %w", err)
	}

	if len(stations) == 0 {
		logger.Warn("No capable stations found",
			"requirements", input.Requirements,
			"stationType", input.StationType,
		)
		return nil, fmt.Errorf("no stations found with required capabilities: %v", input.Requirements)
	}

	// Select the station with the most available capacity
	var bestStation *clients.Station
	maxCapacity := -1
	for i := range stations {
		station := &stations[i]
		if station.AvailableCapacity > maxCapacity {
			maxCapacity = station.AvailableCapacity
			bestStation = station
		}
	}

	logger.Info("Capable station found",
		"stationId", bestStation.StationID,
		"stationType", bestStation.StationType,
		"availableCapacity", bestStation.AvailableCapacity,
	)

	return bestStation, nil
}

// GetStation retrieves a specific station by ID
func (a *ProcessPathActivities) GetStation(ctx context.Context, stationID string) (*clients.Station, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Getting station", "stationId", stationID)

	station, err := a.clients.GetStation(ctx, stationID)
	if err != nil {
		logger.Error("Failed to get station", "stationId", stationID, "error", err)
		return nil, fmt.Errorf("failed to get station %s: %w", stationID, err)
	}

	return station, nil
}

// GetStationsByZone retrieves all stations in a zone
func (a *ProcessPathActivities) GetStationsByZone(ctx context.Context, zone string) ([]clients.Station, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Getting stations by zone", "zone", zone)

	stations, err := a.clients.GetStationsByZone(ctx, zone)
	if err != nil {
		logger.Error("Failed to get stations by zone", "zone", zone, "error", err)
		return nil, fmt.Errorf("failed to get stations in zone %s: %w", zone, err)
	}

	return stations, nil
}
