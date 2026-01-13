package activities

import (
	"context"
	"fmt"

	"github.com/wms-platform/orchestrator/internal/activities/clients"
	"go.temporal.io/sdk/activity"
)

// ProcessPathInput represents the input for determining process path
type ProcessPathInput struct {
	OrderID          string                    `json:"orderId"`
	Items            []ProcessPathItem         `json:"items"`
	GiftWrap         bool                      `json:"giftWrap"`
	GiftWrapDetails  *clients.GiftWrapDetails  `json:"giftWrapDetails,omitempty"`
	HazmatDetails    *clients.HazmatDetails    `json:"hazmatDetails,omitempty"`
	ColdChainDetails *clients.ColdChainDetails `json:"coldChainDetails,omitempty"`
	TotalValue       float64                   `json:"totalValue"`
}

// ProcessPathItem represents an item for process path determination
type ProcessPathItem struct {
	SKU               string  `json:"sku"`
	Quantity          int     `json:"quantity"`
	Weight            float64 `json:"weight"`
	IsFragile         bool    `json:"isFragile"`
	IsHazmat          bool    `json:"isHazmat"`
	RequiresColdChain bool    `json:"requiresColdChain"`
}

// FindCapableStationInput represents input for finding a capable station
type FindCapableStationInput struct {
	Requirements []string `json:"requirements"`
	StationType  string   `json:"stationType"`
	Zone         string   `json:"zone,omitempty"`
}

// DetermineProcessPath calls the process-path-service to determine the process path
func (a *ProcessPathActivities) DetermineProcessPath(ctx context.Context, input ProcessPathInput) (*clients.ProcessPath, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Determining process path via process-path-service", "orderId", input.OrderID)

	// Convert activity input items to client request items
	items := make([]clients.ProcessPathItem, len(input.Items))
	for i, item := range input.Items {
		items[i] = clients.ProcessPathItem{
			SKU:               item.SKU,
			Quantity:          item.Quantity,
			Weight:            item.Weight,
			IsFragile:         item.IsFragile,
			IsHazmat:          item.IsHazmat,
			RequiresColdChain: item.RequiresColdChain,
		}
	}

	req := &clients.DetermineProcessPathRequest{
		OrderID:          input.OrderID,
		Items:            items,
		GiftWrap:         input.GiftWrap,
		GiftWrapDetails:  input.GiftWrapDetails,
		HazmatDetails:    input.HazmatDetails,
		ColdChainDetails: input.ColdChainDetails,
		TotalValue:       input.TotalValue,
	}

	path, err := a.clients.DetermineProcessPathViaService(ctx, req)
	if err != nil {
		logger.Error("Failed to determine process path", "orderId", input.OrderID, "error", err)
		return nil, fmt.Errorf("failed to determine process path: %w", err)
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

// Station Capacity Management Activities

// ReserveStationCapacityInput represents input for reserving station capacity
type ReserveStationCapacityInput struct {
	StationID      string `json:"stationId"`
	OrderID        string `json:"orderId"`
	RequiredSlots  int    `json:"requiredSlots"`  // Number of concurrent task slots needed
	ReservationID  string `json:"reservationId"`  // Unique reservation identifier
}

// ReserveStationCapacityResult represents the result of reserving station capacity
type ReserveStationCapacityResult struct {
	ReservationID      string `json:"reservationId"`
	StationID          string `json:"stationId"`
	ReservedSlots      int    `json:"reservedSlots"`
	RemainingCapacity  int    `json:"remainingCapacity"`
	Success            bool   `json:"success"`
}

// ReserveStationCapacity reserves capacity on a station for an order
func (a *ProcessPathActivities) ReserveStationCapacity(ctx context.Context, input ReserveStationCapacityInput) (*ReserveStationCapacityResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Reserving station capacity",
		"stationId", input.StationID,
		"orderId", input.OrderID,
		"requiredSlots", input.RequiredSlots,
		"reservationId", input.ReservationID,
	)

	// Call facility service to reserve capacity
	req := &clients.ReserveStationCapacityRequest{
		StationID:     input.StationID,
		OrderID:       input.OrderID,
		RequiredSlots: input.RequiredSlots,
		ReservationID: input.ReservationID,
	}

	reservation, err := a.clients.ReserveStationCapacity(ctx, req)
	if err != nil {
		logger.Error("Failed to reserve station capacity",
			"stationId", input.StationID,
			"orderId", input.OrderID,
			"error", err,
		)
		return &ReserveStationCapacityResult{
			ReservationID: input.ReservationID,
			StationID:     input.StationID,
			Success:       false,
		}, fmt.Errorf("failed to reserve station capacity: %w", err)
	}

	logger.Info("Station capacity reserved",
		"stationId", input.StationID,
		"orderId", input.OrderID,
		"reservationId", reservation.ReservationID,
		"reservedSlots", reservation.ReservedSlots,
		"remainingCapacity", reservation.RemainingCapacity,
	)

	return &ReserveStationCapacityResult{
		ReservationID:     reservation.ReservationID,
		StationID:         reservation.StationID,
		ReservedSlots:     reservation.ReservedSlots,
		RemainingCapacity: reservation.RemainingCapacity,
		Success:           true,
	}, nil
}

// ReleaseStationCapacityInput represents input for releasing station capacity
type ReleaseStationCapacityInput struct {
	StationID     string `json:"stationId"`
	OrderID       string `json:"orderId"`
	ReservationID string `json:"reservationId"`
	Reason        string `json:"reason,omitempty"` // Why capacity is being released
}

// ReleaseStationCapacity releases previously reserved station capacity
func (a *ProcessPathActivities) ReleaseStationCapacity(ctx context.Context, input ReleaseStationCapacityInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Releasing station capacity",
		"stationId", input.StationID,
		"orderId", input.OrderID,
		"reservationId", input.ReservationID,
		"reason", input.Reason,
	)

	// Call facility service to release capacity
	req := &clients.ReleaseStationCapacityRequest{
		StationID:     input.StationID,
		OrderID:       input.OrderID,
		ReservationID: input.ReservationID,
		Reason:        input.Reason,
	}

	err := a.clients.ReleaseStationCapacity(ctx, req)
	if err != nil {
		logger.Error("Failed to release station capacity",
			"stationId", input.StationID,
			"orderId", input.OrderID,
			"reservationId", input.ReservationID,
			"error", err,
		)
		return fmt.Errorf("failed to release station capacity: %w", err)
	}

	logger.Info("Station capacity released",
		"stationId", input.StationID,
		"orderId", input.OrderID,
		"reservationId", input.ReservationID,
	)

	return nil
}
