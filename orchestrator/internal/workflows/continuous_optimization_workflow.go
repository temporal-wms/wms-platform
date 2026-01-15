package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ContinuousOptimizationWorkflowInput represents input for the continuous optimization workflow
type ContinuousOptimizationWorkflowInput struct {
	FacilityID              string `json:"facilityId"`
	Zone                    string `json:"zone,omitempty"`
	MonitoringInterval      string `json:"monitoringInterval"`      // e.g., "5m", "10m"
	CapacityThreshold       float64 `json:"capacityThreshold"`       // Trigger rebalancing above this %
	UnderutilizationThreshold float64 `json:"underutilizationThreshold"` // Consider underutilized below this %
	MaxOrdersPerRebalance   int    `json:"maxOrdersPerRebalance"`   // Max orders to move per rebalancing cycle
	EnableAutoRebalancing   bool   `json:"enableAutoRebalancing"`   // Auto-trigger rebalancing
	EnableAutoRerouting     bool   `json:"enableAutoRerouting"`     // Auto-trigger rerouting
	EnableCapacityPrediction bool  `json:"enableCapacityPrediction"` // Generate capacity predictions
}

// ContinuousOptimizationWorkflowResult represents the result of optimization workflow
type ContinuousOptimizationWorkflowResult struct {
	TotalCyclesRun         int       `json:"totalCyclesRun"`
	TotalRebalancingEvents int       `json:"totalRebalancingEvents"`
	TotalReroutingEvents   int       `json:"totalReroutingEvents"`
	TotalOrdersRebalanced  int       `json:"totalOrdersRebalanced"`
	TotalOrdersRerouted    int       `json:"totalOrdersRerouted"`
	AverageSystemHealth    string    `json:"averageSystemHealth"`
	StartTime              time.Time `json:"startTime"`
	LastCycleTime          time.Time `json:"lastCycleTime"`
}

// ContinuousOptimizationWorkflow continuously monitors and optimizes warehouse operations
func ContinuousOptimizationWorkflow(ctx workflow.Context, input ContinuousOptimizationWorkflowInput) (*ContinuousOptimizationWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting continuous optimization workflow",
		"facilityId", input.FacilityID,
		"monitoringInterval", input.MonitoringInterval,
		"autoRebalancing", input.EnableAutoRebalancing,
		"autoRerouting", input.EnableAutoRerouting,
	)

	// Set default values
	if input.MonitoringInterval == "" {
		input.MonitoringInterval = "5m"
	}
	if input.CapacityThreshold == 0 {
		input.CapacityThreshold = 0.85 // 85% utilization triggers rebalancing
	}
	if input.UnderutilizationThreshold == 0 {
		input.UnderutilizationThreshold = 0.30 // 30% utilization is considered underutilized
	}
	if input.MaxOrdersPerRebalance == 0 {
		input.MaxOrdersPerRebalance = 50
	}

	// Parse monitoring interval
	monitoringDuration, err := time.ParseDuration(input.MonitoringInterval)
	if err != nil {
		logger.Error("Invalid monitoring interval", "interval", input.MonitoringInterval, "error", err)
		monitoringDuration = 5 * time.Minute // Default to 5 minutes
	}

	// Configure activity options
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	result := &ContinuousOptimizationWorkflowResult{
		StartTime: workflow.Now(ctx),
	}

	// Run optimization cycles continuously
	for {
		cycleStart := workflow.Now(ctx)
		logger.Info("Starting optimization cycle", "cycle", result.TotalCyclesRun+1)

		// ========================================
		// Step 1: Monitor System Health
		// ========================================
		var healthResult map[string]interface{}
		monitorInput := map[string]interface{}{
			"facilityId": input.FacilityID,
			"zone":       input.Zone,
			"timeWindow": "1h", // Monitor last hour
		}

		err := workflow.ExecuteActivity(ctx, "MonitorSystemHealth", monitorInput).Get(ctx, &healthResult)
		if err != nil {
			logger.Error("Failed to monitor system health", "error", err)
			// Continue to next cycle even if monitoring fails
			workflow.Sleep(ctx, monitoringDuration)
			continue
		}

		overallHealth, _ := healthResult["overallHealth"].(string)
		rebalancingRecommended, _ := healthResult["rebalancingRecommended"].(bool)
		reroutingOpportunities, _ := healthResult["reroutingOpportunities"].(float64)

		logger.Info("System health checked",
			"overallHealth", overallHealth,
			"rebalancingRecommended", rebalancingRecommended,
			"reroutingOpportunities", int(reroutingOpportunities),
		)

		// Track average health (simplified)
		if result.AverageSystemHealth == "" {
			result.AverageSystemHealth = overallHealth
		}

		// ========================================
		// Step 2: Trigger Rebalancing (if needed and enabled)
		// ========================================
		if input.EnableAutoRebalancing && rebalancingRecommended {
			logger.Info("Triggering automatic rebalancing")

			// Extract overloaded and underutilized stations
			overloadedStations := []string{}
			underutilizedStations := []string{}

			if overloaded, ok := healthResult["overloadedStations"].([]interface{}); ok {
				for _, s := range overloaded {
					if station, ok := s.(string); ok {
						overloadedStations = append(overloadedStations, station)
					}
				}
			}

			if underutilized, ok := healthResult["underutilizedStations"].([]interface{}); ok {
				for _, s := range underutilized {
					if station, ok := s.(string); ok {
						underutilizedStations = append(underutilizedStations, station)
					}
				}
			}

			if len(overloadedStations) > 0 && len(underutilizedStations) > 0 {
				var rebalanceResult map[string]interface{}
				rebalanceInput := map[string]interface{}{
					"facilityId":            input.FacilityID,
					"overloadedStations":    overloadedStations,
					"underutilizedStations": underutilizedStations,
					"maxOrdersToRebalance":  input.MaxOrdersPerRebalance,
				}

				err := workflow.ExecuteActivity(ctx, "RebalanceWaves", rebalanceInput).Get(ctx, &rebalanceResult)
				if err != nil {
					logger.Error("Failed to rebalance waves", "error", err)
				} else if success, ok := rebalanceResult["success"].(bool); ok && success {
					ordersRebalanced, _ := rebalanceResult["ordersRebalanced"].(float64)
					result.TotalRebalancingEvents++
					result.TotalOrdersRebalanced += int(ordersRebalanced)
					logger.Info("Rebalancing completed",
						"ordersRebalanced", int(ordersRebalanced),
						"totalRebalancingEvents", result.TotalRebalancingEvents,
					)
				}
			}
		}

		// ========================================
		// Step 3: Trigger Dynamic Rerouting (if opportunities exist and enabled)
		// ========================================
		if input.EnableAutoRerouting && reroutingOpportunities > 0 {
			logger.Info("Triggering dynamic rerouting",
				"opportunities", int(reroutingOpportunities),
			)

			// Extract overloaded stations that need rerouting
			overloadedStations := []string{}
			if overloaded, ok := healthResult["overloadedStations"].([]interface{}); ok {
				for _, s := range overloaded {
					if station, ok := s.(string); ok {
						overloadedStations = append(overloadedStations, station)
					}
				}
			}

			// Reroute orders from the first overloaded station
			if len(overloadedStations) > 0 {
				var reroutingResult map[string]interface{}
				reroutingInput := map[string]interface{}{
					"facilityId":   input.FacilityID,
					"stationId":    overloadedStations[0], // Reroute from most overloaded
					"reason":       "automatic_load_balancing",
					"priority":     "medium",
					"forceReroute": false,
					"orderIds":     []string{}, // Would be populated from actual orders
				}

				err := workflow.ExecuteActivity(ctx, "TriggerDynamicRerouting", reroutingInput).Get(ctx, &reroutingResult)
				if err != nil {
					logger.Error("Failed to trigger dynamic rerouting", "error", err)
				} else if success, ok := reroutingResult["success"].(bool); ok && success {
					ordersRerouted, _ := reroutingResult["ordersRerouted"].(float64)
					result.TotalReroutingEvents++
					result.TotalOrdersRerouted += int(ordersRerouted)
					logger.Info("Dynamic rerouting completed",
						"ordersRerouted", int(ordersRerouted),
						"totalReroutingEvents", result.TotalReroutingEvents,
					)
				}
			}
		}

		// ========================================
		// Step 4: Generate Capacity Predictions (if enabled)
		// ========================================
		if input.EnableCapacityPrediction {
			var predictionResult map[string]interface{}
			predictionInput := map[string]interface{}{
				"facilityId":       input.FacilityID,
				"zone":             input.Zone,
				"predictionWindow": "4h",  // Predict next 4 hours
				"historicalWindow": "7d",  // Based on last 7 days
			}

			err := workflow.ExecuteActivity(ctx, "PredictCapacityNeeds", predictionInput).Get(ctx, &predictionResult)
			if err != nil {
				logger.Error("Failed to predict capacity needs", "error", err)
			} else {
				predictedVolume, _ := predictionResult["predictedOrderVolume"].(float64)
				confidenceScore, _ := predictionResult["confidenceScore"].(float64)
				logger.Info("Capacity prediction generated",
					"predictedVolume", int(predictedVolume),
					"confidenceScore", confidenceScore,
				)

				// Log bottlenecks if any
				if bottlenecks, ok := predictionResult["predictedBottlenecks"].([]interface{}); ok && len(bottlenecks) > 0 {
					logger.Warn("Predicted bottlenecks detected",
						"bottleneckCount", len(bottlenecks),
						"action", "Consider preemptive capacity adjustments",
					)
				}
			}
		}

		// ========================================
		// Cycle Complete
		// ========================================
		result.TotalCyclesRun++
		result.LastCycleTime = workflow.Now(ctx)

		cycleDuration := workflow.Now(ctx).Sub(cycleStart)
		logger.Info("Optimization cycle completed",
			"cycle", result.TotalCyclesRun,
			"duration", cycleDuration,
			"overallHealth", overallHealth,
		)

		// Check for completion signal (for graceful shutdown)
		selector := workflow.NewSelector(ctx)

		// Add timer for next cycle
		timerFired := false
		timer := workflow.NewTimer(ctx, monitoringDuration)
		selector.AddFuture(timer, func(f workflow.Future) {
			timerFired = true
		})

		// Add signal handler for stop command
		var stopSignal string
		stopChannel := workflow.GetSignalChannel(ctx, "stop-optimization")
		selector.AddReceive(stopChannel, func(c workflow.ReceiveChannel, more bool) {
			c.Receive(ctx, &stopSignal)
			logger.Info("Received stop signal, shutting down optimization workflow")
		})

		// Wait for either timer or stop signal
		selector.Select(ctx)

		if stopSignal != "" {
			// Stop signal received, exit gracefully
			logger.Info("Continuous optimization workflow stopped",
				"totalCyclesRun", result.TotalCyclesRun,
				"totalRebalancingEvents", result.TotalRebalancingEvents,
				"totalReroutingEvents", result.TotalReroutingEvents,
			)
			return result, nil
		}

		if !timerFired {
			// Shouldn't happen, but handle just in case
			workflow.Sleep(ctx, monitoringDuration)
		}

		// Continue to next cycle
	}
}

// StopContinuousOptimization is a helper to send stop signal to the workflow
func StopContinuousOptimization(ctx workflow.Context) error {
	return workflow.SignalExternalWorkflow(ctx, "", "", "stop-optimization", "stop").Get(ctx, nil)
}
