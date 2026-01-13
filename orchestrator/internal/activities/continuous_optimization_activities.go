package activities

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/orchestrator/internal/activities/clients"
	"go.temporal.io/sdk/activity"
)

// ContinuousOptimizationActivities handles continuous optimization and rebalancing
type ContinuousOptimizationActivities struct {
	clients *clients.ServiceClients
}

// NewContinuousOptimizationActivities creates a new ContinuousOptimizationActivities instance
func NewContinuousOptimizationActivities(clients *clients.ServiceClients) *ContinuousOptimizationActivities {
	return &ContinuousOptimizationActivities{
		clients: clients,
	}
}

// MonitorSystemHealthInput represents input for system health monitoring
type MonitorSystemHealthInput struct {
	FacilityID string `json:"facilityId,omitempty"`
	Zone       string `json:"zone,omitempty"`
	TimeWindow string `json:"timeWindow"` // e.g., "1h", "24h"
}

// MonitorSystemHealthResult represents system health status
type MonitorSystemHealthResult struct {
	OverallHealth           string             `json:"overallHealth"` // "healthy", "degraded", "critical"
	StationUtilization      map[string]float64 `json:"stationUtilization"`
	OverloadedStations      []string           `json:"overloadedStations"`
	UnderutilizedStations   []string           `json:"underutilizedStations"`
	AverageConfidence       float64            `json:"averageConfidence"`
	CapacityConstrainedRate float64            `json:"capacityConstrainedRate"`
	RebalancingRecommended  bool               `json:"rebalancingRecommended"`
	ReroutingOpportunities  int                `json:"reroutingOpportunities"`
	Timestamp               time.Time          `json:"timestamp"`
}

// MonitorSystemHealth monitors overall system health and routing efficiency
func (a *ContinuousOptimizationActivities) MonitorSystemHealth(ctx context.Context, input MonitorSystemHealthInput) (*MonitorSystemHealthResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Monitoring system health",
		"facilityId", input.FacilityID,
		"zone", input.Zone,
		"timeWindow", input.TimeWindow,
	)

	// Get routing metrics from process-path-service
	req := &clients.GetRoutingMetricsRequest{
		FacilityID: input.FacilityID,
		Zone:       input.Zone,
		TimeWindow: input.TimeWindow,
	}

	metrics, err := a.clients.GetRoutingMetrics(ctx, req)
	if err != nil {
		logger.Error("Failed to get routing metrics",
			"facilityId", input.FacilityID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get routing metrics: %w", err)
	}

	// Analyze station utilization
	overloaded := make([]string, 0)
	underutilized := make([]string, 0)

	for stationID, utilization := range metrics.StationUtilization {
		if utilization > 0.90 {
			// Over 90% utilization - overloaded
			overloaded = append(overloaded, stationID)
		} else if utilization < 0.30 {
			// Under 30% utilization - underutilized
			underutilized = append(underutilized, stationID)
		}
	}

	// Determine overall health
	overallHealth := "healthy"
	if len(overloaded) > 0 || metrics.CapacityConstrainedRate > 0.20 {
		overallHealth = "degraded"
	}
	if len(overloaded) > 3 || metrics.CapacityConstrainedRate > 0.40 {
		overallHealth = "critical"
	}

	// Count rerouting opportunities (orders on overloaded stations that could move to underutilized)
	reroutingOpportunities := 0
	if len(overloaded) > 0 && len(underutilized) > 0 {
		reroutingOpportunities = len(overloaded) * 5 // Estimate 5 orders per overloaded station
	}

	result := &MonitorSystemHealthResult{
		OverallHealth:           overallHealth,
		StationUtilization:      metrics.StationUtilization,
		OverloadedStations:      overloaded,
		UnderutilizedStations:   underutilized,
		AverageConfidence:       metrics.AverageConfidence,
		CapacityConstrainedRate: metrics.CapacityConstrainedRate,
		RebalancingRecommended:  metrics.RebalancingRecommended,
		ReroutingOpportunities:  reroutingOpportunities,
		Timestamp:               time.Now(),
	}

	logger.Info("System health monitored",
		"overallHealth", overallHealth,
		"overloadedStations", len(overloaded),
		"underutilizedStations", len(underutilized),
		"rebalancingRecommended", metrics.RebalancingRecommended,
	)

	return result, nil
}

// RebalanceWavesInput represents input for wave rebalancing
type RebalanceWavesInput struct {
	FacilityID            string   `json:"facilityId"`
	OverloadedStations    []string `json:"overloadedStations"`
	UnderutilizedStations []string `json:"underutilizedStations"`
	MaxOrdersToRebalance  int      `json:"maxOrdersToRebalance"` // Max orders to move
}

// RebalanceWavesResult represents the result of wave rebalancing
type RebalanceWavesResult struct {
	OrdersRebalanced  int                       `json:"ordersRebalanced"`
	StationChanges    map[string]StationChange  `json:"stationChanges"`
	NewUtilization    map[string]float64        `json:"newUtilization"`
	RebalancedAt      time.Time                 `json:"rebalancedAt"`
	Success           bool                      `json:"success"`
}

// StationChange represents a station load change
type StationChange struct {
	StationID          string `json:"stationId"`
	OrdersAdded        int    `json:"ordersAdded"`
	OrdersRemoved      int    `json:"ordersRemoved"`
	UtilizationBefore  float64 `json:"utilizationBefore"`
	UtilizationAfter   float64 `json:"utilizationAfter"`
}

// RebalanceWaves redistributes orders across stations to balance load
func (a *ContinuousOptimizationActivities) RebalanceWaves(ctx context.Context, input RebalanceWavesInput) (*RebalanceWavesResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Rebalancing waves",
		"facilityId", input.FacilityID,
		"overloadedStations", len(input.OverloadedStations),
		"underutilizedStations", len(input.UnderutilizedStations),
	)

	if len(input.OverloadedStations) == 0 || len(input.UnderutilizedStations) == 0 {
		logger.Info("No rebalancing needed - no station imbalance detected")
		return &RebalanceWavesResult{
			OrdersRebalanced: 0,
			StationChanges:   make(map[string]StationChange),
			RebalancedAt:     time.Now(),
			Success:          true,
		}, nil
	}

	// Get pending orders from overloaded stations
	// In a real implementation, this would query the waving-service or order-service
	// For now, we'll simulate the rebalancing logic

	ordersRebalanced := 0
	stationChanges := make(map[string]StationChange)
	maxOrders := input.MaxOrdersToRebalance
	if maxOrders == 0 {
		maxOrders = 50 // Default limit
	}

	// Simulate rerouting orders from overloaded to underutilized stations
	ordersPerStation := maxOrders / len(input.OverloadedStations)
	if ordersPerStation == 0 {
		ordersPerStation = 1
	}

	for i, overloadedStation := range input.OverloadedStations {
		if ordersRebalanced >= maxOrders {
			break
		}

		// Pick underutilized station in round-robin fashion
		targetStation := input.UnderutilizedStations[i%len(input.UnderutilizedStations)]

		// Simulate moving orders
		ordersMoved := ordersPerStation
		if ordersRebalanced+ordersMoved > maxOrders {
			ordersMoved = maxOrders - ordersRebalanced
		}

		// Record changes for overloaded station
		stationChanges[overloadedStation] = StationChange{
			StationID:         overloadedStation,
			OrdersRemoved:     ordersMoved,
			UtilizationBefore: 0.95, // Simulated
			UtilizationAfter:  0.75, // Simulated
		}

		// Record changes for underutilized station
		if existing, ok := stationChanges[targetStation]; ok {
			existing.OrdersAdded += ordersMoved
			stationChanges[targetStation] = existing
		} else {
			stationChanges[targetStation] = StationChange{
				StationID:         targetStation,
				OrdersAdded:       ordersMoved,
				UtilizationBefore: 0.25, // Simulated
				UtilizationAfter:  0.50, // Simulated
			}
		}

		ordersRebalanced += ordersMoved

		logger.Info("Orders rebalanced",
			"from", overloadedStation,
			"to", targetStation,
			"count", ordersMoved,
		)
	}

	result := &RebalanceWavesResult{
		OrdersRebalanced: ordersRebalanced,
		StationChanges:   stationChanges,
		NewUtilization:   make(map[string]float64), // Would be populated from actual metrics
		RebalancedAt:     time.Now(),
		Success:          ordersRebalanced > 0,
	}

	logger.Info("Wave rebalancing completed",
		"ordersRebalanced", ordersRebalanced,
		"stationsAffected", len(stationChanges),
	)

	return result, nil
}

// TriggerDynamicReroutingInput represents input for dynamic rerouting
type TriggerDynamicReroutingInput struct {
	FacilityID   string   `json:"facilityId"`
	OrderIDs     []string `json:"orderIds,omitempty"`     // Specific orders to reroute
	StationID    string   `json:"stationId,omitempty"`    // Reroute all orders from this station
	Reason       string   `json:"reason"`                 // Reason for rerouting
	Priority     string   `json:"priority"`               // "high", "medium", "low"
	ForceReroute bool     `json:"forceReroute"`           // Force even if not optimal
}

// TriggerDynamicReroutingResult represents the result of dynamic rerouting
type TriggerDynamicReroutingResult struct {
	OrdersRerouted     int                  `json:"ordersRerouted"`
	ReroutingDecisions []ReroutingDecision  `json:"reroutingDecisions"`
	AverageConfidence  float64              `json:"averageConfidence"`
	ReroutedAt         time.Time            `json:"reroutedAt"`
	Success            bool                 `json:"success"`
}

// ReroutingDecision represents a single rerouting decision
type ReroutingDecision struct {
	OrderID          string  `json:"orderId"`
	OldStationID     string  `json:"oldStationId"`
	NewStationID     string  `json:"newStationId"`
	Score            float64 `json:"score"`
	Confidence       float64 `json:"confidence"`
	ImprovementScore float64 `json:"improvementScore"` // How much better is the new route
}

// TriggerDynamicRerouting triggers rerouting of in-flight orders based on changing conditions
func (a *ContinuousOptimizationActivities) TriggerDynamicRerouting(ctx context.Context, input TriggerDynamicReroutingInput) (*TriggerDynamicReroutingResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Triggering dynamic rerouting",
		"facilityId", input.FacilityID,
		"orderCount", len(input.OrderIDs),
		"stationId", input.StationID,
		"reason", input.Reason,
	)

	if len(input.OrderIDs) == 0 && input.StationID == "" {
		logger.Info("No orders or station specified for rerouting")
		return &TriggerDynamicReroutingResult{
			OrdersRerouted:     0,
			ReroutingDecisions: []ReroutingDecision{},
			ReroutedAt:         time.Now(),
			Success:            true,
		}, nil
	}

	reroutingDecisions := make([]ReroutingDecision, 0)
	totalConfidence := 0.0

	// Reroute each order
	for _, orderID := range input.OrderIDs {
		// Call routing optimizer to find better route
		req := &clients.RerouteOrderRequest{
			OrderID:      orderID,
			CurrentPath:  input.StationID, // Current station
			Reason:       input.Reason,
			Requirements: []string{}, // Would be populated from order data
			Priority:     input.Priority,
			ForceReroute: input.ForceReroute,
		}

		rerouteResp, err := a.clients.RerouteOrder(ctx, req)
		if err != nil {
			logger.Warn("Failed to reroute order",
				"orderId", orderID,
				"error", err,
			)
			continue
		}

		// Record decision
		decision := ReroutingDecision{
			OrderID:          orderID,
			OldStationID:     input.StationID,
			NewStationID:     rerouteResp.NewStationID,
			Score:            rerouteResp.Score,
			Confidence:       rerouteResp.Confidence,
			ImprovementScore: rerouteResp.Score * rerouteResp.Confidence, // Simple improvement metric
		}
		reroutingDecisions = append(reroutingDecisions, decision)
		totalConfidence += rerouteResp.Confidence

		logger.Info("Order rerouted",
			"orderId", orderID,
			"newStationId", rerouteResp.NewStationID,
			"confidence", rerouteResp.Confidence,
		)
	}

	avgConfidence := 0.0
	if len(reroutingDecisions) > 0 {
		avgConfidence = totalConfidence / float64(len(reroutingDecisions))
	}

	result := &TriggerDynamicReroutingResult{
		OrdersRerouted:     len(reroutingDecisions),
		ReroutingDecisions: reroutingDecisions,
		AverageConfidence:  avgConfidence,
		ReroutedAt:         time.Now(),
		Success:            len(reroutingDecisions) > 0,
	}

	logger.Info("Dynamic rerouting completed",
		"ordersRerouted", len(reroutingDecisions),
		"averageConfidence", avgConfidence,
	)

	return result, nil
}

// PredictCapacityNeedsInput represents input for capacity prediction
type PredictCapacityNeedsInput struct {
	FacilityID       string    `json:"facilityId"`
	Zone             string    `json:"zone,omitempty"`
	PredictionWindow string    `json:"predictionWindow"` // e.g., "1h", "4h", "24h"
	HistoricalWindow string    `json:"historicalWindow"` // e.g., "7d", "30d"
}

// PredictCapacityNeedsResult represents predicted capacity requirements
type PredictCapacityNeedsResult struct {
	PredictedOrderVolume     int                `json:"predictedOrderVolume"`
	PredictedStationLoad     map[string]float64 `json:"predictedStationLoad"`
	RecommendedStaffing      map[string]int     `json:"recommendedStaffing"` // Station -> worker count
	PredictedBottlenecks     []string           `json:"predictedBottlenecks"`
	ConfidenceScore          float64            `json:"confidenceScore"`
	RecommendedActions       []string           `json:"recommendedActions"`
	PredictionTime           time.Time          `json:"predictionTime"`
	ForecastHorizon          time.Time          `json:"forecastHorizon"`
}

// PredictCapacityNeeds forecasts capacity requirements based on historical patterns
func (a *ContinuousOptimizationActivities) PredictCapacityNeeds(ctx context.Context, input PredictCapacityNeedsInput) (*PredictCapacityNeedsResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Predicting capacity needs",
		"facilityId", input.FacilityID,
		"predictionWindow", input.PredictionWindow,
	)

	// In a real implementation, this would use historical data and ML models
	// For now, we'll provide a simple rule-based prediction

	// Get current routing metrics
	req := &clients.GetRoutingMetricsRequest{
		FacilityID: input.FacilityID,
		Zone:       input.Zone,
		TimeWindow: input.HistoricalWindow,
	}

	metrics, err := a.clients.GetRoutingMetrics(ctx, req)
	if err != nil {
		logger.Error("Failed to get routing metrics for prediction",
			"facilityId", input.FacilityID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get routing metrics: %w", err)
	}

	// Simple prediction: assume 20% growth in next window
	growthFactor := 1.20
	predictedVolume := int(float64(metrics.TotalRoutingDecisions) * growthFactor)

	// Predict station load based on current utilization + growth
	predictedLoad := make(map[string]float64)
	bottlenecks := make([]string, 0)
	for stationID, utilization := range metrics.StationUtilization {
		predictedUtil := utilization * growthFactor
		predictedLoad[stationID] = predictedUtil

		if predictedUtil > 0.85 {
			bottlenecks = append(bottlenecks, stationID)
		}
	}

	// Recommend staffing based on predicted load
	recommendedStaffing := make(map[string]int)
	for stationID, predictedUtil := range predictedLoad {
		// Simple staffing rule: 1 worker per 20% utilization
		workers := int(predictedUtil / 0.20)
		if workers < 1 {
			workers = 1
		}
		recommendedStaffing[stationID] = workers
	}

	// Generate recommended actions
	recommendedActions := make([]string, 0)
	if len(bottlenecks) > 0 {
		recommendedActions = append(recommendedActions,
			fmt.Sprintf("Increase staffing at %d predicted bottleneck stations", len(bottlenecks)))
		recommendedActions = append(recommendedActions,
			"Consider preemptive rebalancing to distribute load")
	}
	if predictedVolume > metrics.TotalRoutingDecisions {
		recommendedActions = append(recommendedActions,
			fmt.Sprintf("Prepare for %.0f%% volume increase", (growthFactor-1)*100))
	}

	// Calculate forecast horizon based on prediction window
	forecastHorizon := time.Now().Add(1 * time.Hour) // Default 1 hour
	switch input.PredictionWindow {
	case "4h":
		forecastHorizon = time.Now().Add(4 * time.Hour)
	case "24h":
		forecastHorizon = time.Now().Add(24 * time.Hour)
	}

	result := &PredictCapacityNeedsResult{
		PredictedOrderVolume: predictedVolume,
		PredictedStationLoad: predictedLoad,
		RecommendedStaffing:  recommendedStaffing,
		PredictedBottlenecks: bottlenecks,
		ConfidenceScore:      0.75, // Simulated confidence
		RecommendedActions:   recommendedActions,
		PredictionTime:       time.Now(),
		ForecastHorizon:      forecastHorizon,
	}

	logger.Info("Capacity needs predicted",
		"predictedVolume", predictedVolume,
		"predictedBottlenecks", len(bottlenecks),
		"confidenceScore", 0.75,
	)

	return result, nil
}
