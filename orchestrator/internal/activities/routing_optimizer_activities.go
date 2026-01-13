package activities

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/orchestrator/internal/activities/clients"
	"go.temporal.io/sdk/activity"
)

// RoutingOptimizerActivities contains dynamic routing optimization activities
type RoutingOptimizerActivities struct {
	clients *clients.ServiceClients
}

// NewRoutingOptimizerActivities creates a new RoutingOptimizerActivities instance
func NewRoutingOptimizerActivities(clients *clients.ServiceClients) *RoutingOptimizerActivities {
	return &RoutingOptimizerActivities{
		clients: clients,
	}
}

// OptimizeStationSelectionInput represents input for optimizing station selection
type OptimizeStationSelectionInput struct {
	OrderID            string    `json:"orderId"`
	Priority           string    `json:"priority"`
	Requirements       []string  `json:"requirements"`
	SpecialHandling    []string  `json:"specialHandling"`
	ItemCount          int       `json:"itemCount"`
	TotalWeight        float64   `json:"totalWeight"`
	PromisedDeliveryAt time.Time `json:"promisedDeliveryAt"`
	RequiredSkills     []string  `json:"requiredSkills"`
	RequiredEquipment  []string  `json:"requiredEquipment"`
	Zone               string    `json:"zone,omitempty"`
	StationType        string    `json:"stationType"`
}

// OptimizeStationSelectionResult represents the result of station optimization
type OptimizeStationSelectionResult struct {
	SelectedStationID string                 `json:"selectedStationId"`
	Score             float64                `json:"score"`
	Reasoning         map[string]float64     `json:"reasoning"`
	AlternateStations []AlternateStationInfo `json:"alternateStations"`
	Confidence        float64                `json:"confidence"`
	Success           bool                   `json:"success"`
}

// AlternateStationInfo represents an alternate station option
type AlternateStationInfo struct {
	StationID string  `json:"stationId"`
	Score     float64 `json:"score"`
	Rank      int     `json:"rank"`
}

// OptimizeStationSelection uses ML-like optimization to select the best station
func (a *RoutingOptimizerActivities) OptimizeStationSelection(ctx context.Context, input OptimizeStationSelectionInput) (*OptimizeStationSelectionResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Optimizing station selection with dynamic routing",
		"orderId", input.OrderID,
		"priority", input.Priority,
		"requirements", input.Requirements,
	)

	// Call process-path-service routing optimizer
	req := &clients.OptimizeRoutingRequest{
		OrderID:            input.OrderID,
		Priority:           input.Priority,
		Requirements:       input.Requirements,
		SpecialHandling:    input.SpecialHandling,
		ItemCount:          input.ItemCount,
		TotalWeight:        input.TotalWeight,
		PromisedDeliveryAt: input.PromisedDeliveryAt,
		RequiredSkills:     input.RequiredSkills,
		RequiredEquipment:  input.RequiredEquipment,
		Zone:               input.Zone,
		StationType:        input.StationType,
	}

	decision, err := a.clients.OptimizeRouting(ctx, req)
	if err != nil {
		logger.Error("Failed to optimize station selection",
			"orderId", input.OrderID,
			"error", err,
		)
		return &OptimizeStationSelectionResult{
			Success: false,
		}, fmt.Errorf("failed to optimize routing: %w", err)
	}

	// Convert alternates
	alternates := make([]AlternateStationInfo, len(decision.AlternateStations))
	for i, alt := range decision.AlternateStations {
		alternates[i] = AlternateStationInfo{
			StationID: alt.StationID,
			Score:     alt.Score,
			Rank:      alt.Rank,
		}
	}

	logger.Info("Station selection optimized",
		"orderId", input.OrderID,
		"selectedStationId", decision.SelectedStationID,
		"score", decision.Score,
		"confidence", decision.Confidence,
		"alternateCount", len(alternates),
	)

	return &OptimizeStationSelectionResult{
		SelectedStationID: decision.SelectedStationID,
		Score:             decision.Score,
		Reasoning:         decision.Reasoning,
		AlternateStations: alternates,
		Confidence:        decision.Confidence,
		Success:           true,
	}, nil
}

// GetRoutingMetricsInput represents input for getting routing metrics
type GetRoutingMetricsInput struct {
	FacilityID string `json:"facilityId,omitempty"`
	Zone       string `json:"zone,omitempty"`
	TimeWindow string `json:"timeWindow,omitempty"` // e.g., "1h", "24h"
}

// GetRoutingMetricsResult represents routing metrics
type GetRoutingMetricsResult struct {
	TotalRoutingDecisions   int                `json:"totalRoutingDecisions"`
	AverageDecisionTime     int64              `json:"averageDecisionTimeMs"` // milliseconds
	AverageConfidence       float64            `json:"averageConfidence"`
	StationUtilization      map[string]float64 `json:"stationUtilization"`
	CapacityConstrainedRate float64            `json:"capacityConstrainedRate"`
	RouteChanges            int                `json:"routeChanges"`
	RebalancingRecommended  bool               `json:"rebalancingRecommended"`
	LastUpdated             time.Time          `json:"lastUpdated"`
}

// GetRoutingMetrics retrieves current routing metrics
func (a *RoutingOptimizerActivities) GetRoutingMetrics(ctx context.Context, input GetRoutingMetricsInput) (*GetRoutingMetricsResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Getting routing metrics",
		"facilityId", input.FacilityID,
		"zone", input.Zone,
		"timeWindow", input.TimeWindow,
	)

	// Call process-path-service for metrics
	req := &clients.GetRoutingMetricsRequest{
		FacilityID: input.FacilityID,
		Zone:       input.Zone,
		TimeWindow: input.TimeWindow,
	}

	metrics, err := a.clients.GetRoutingMetrics(ctx, req)
	if err != nil {
		logger.Error("Failed to get routing metrics",
			"error", err,
		)
		return nil, fmt.Errorf("failed to get routing metrics: %w", err)
	}

	logger.Info("Routing metrics retrieved",
		"totalDecisions", metrics.TotalRoutingDecisions,
		"averageConfidence", metrics.AverageConfidence,
		"rebalancingRecommended", metrics.RebalancingRecommended,
	)

	return &GetRoutingMetricsResult{
		TotalRoutingDecisions:   metrics.TotalRoutingDecisions,
		AverageDecisionTime:     metrics.AverageDecisionTimeMs,
		AverageConfidence:       metrics.AverageConfidence,
		StationUtilization:      metrics.StationUtilization,
		CapacityConstrainedRate: metrics.CapacityConstrainedRate,
		RouteChanges:            metrics.RouteChanges,
		RebalancingRecommended:  metrics.RebalancingRecommended,
		LastUpdated:             metrics.LastUpdated,
	}, nil
}

// RerouteOrderInput represents input for rerouting an order
type RerouteOrderInput struct {
	OrderID       string   `json:"orderId"`
	CurrentPath   string   `json:"currentPath"`   // Current station/path
	Reason        string   `json:"reason"`        // Why rerouting
	Requirements  []string `json:"requirements"`
	Priority      string   `json:"priority"`
	ForceReroute  bool     `json:"forceReroute"`  // Force even if not optimal
}

// RerouteOrderResult represents the result of rerouting
type RerouteOrderResult struct {
	NewStationID      string  `json:"newStationId"`
	PreviousStationID string  `json:"previousStationId"`
	Score             float64 `json:"score"`
	Confidence        float64 `json:"confidence"`
	RerouteTime       time.Time `json:"rerouteTime"`
	Success           bool    `json:"success"`
}

// RerouteOrder dynamically reroutes an order to a better station
func (a *RoutingOptimizerActivities) RerouteOrder(ctx context.Context, input RerouteOrderInput) (*RerouteOrderResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Rerouting order",
		"orderId", input.OrderID,
		"currentPath", input.CurrentPath,
		"reason", input.Reason,
		"forceReroute", input.ForceReroute,
	)

	// Call process-path-service for rerouting decision
	req := &clients.RerouteOrderRequest{
		OrderID:      input.OrderID,
		CurrentPath:  input.CurrentPath,
		Reason:       input.Reason,
		Requirements: input.Requirements,
		Priority:     input.Priority,
		ForceReroute: input.ForceReroute,
	}

	rerouteDecision, err := a.clients.RerouteOrder(ctx, req)
	if err != nil {
		logger.Error("Failed to reroute order",
			"orderId", input.OrderID,
			"error", err,
		)
		return &RerouteOrderResult{
			Success: false,
		}, fmt.Errorf("failed to reroute order: %w", err)
	}

	logger.Info("Order rerouted",
		"orderId", input.OrderID,
		"newStationId", rerouteDecision.NewStationID,
		"previousStationId", input.CurrentPath,
		"confidence", rerouteDecision.Confidence,
	)

	return &RerouteOrderResult{
		NewStationID:      rerouteDecision.NewStationID,
		PreviousStationID: input.CurrentPath,
		Score:             rerouteDecision.Score,
		Confidence:        rerouteDecision.Confidence,
		RerouteTime:       time.Now(),
		Success:           true,
	}, nil
}
