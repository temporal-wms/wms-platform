package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ConsolidationWorkflowInput represents input for the consolidation workflow
type ConsolidationWorkflowInput struct {
	OrderID     string       `json:"orderId"`
	PickedItems []PickedItem `json:"pickedItems"`
	// Unit-level tracking fields
	UnitIDs []string `json:"unitIds,omitempty"` // Specific units to consolidate
	PathID  string   `json:"pathId,omitempty"`  // Process path ID for consistency
	// Multi-route support fields
	IsMultiRoute       bool     `json:"isMultiRoute,omitempty"`       // Flag for multi-route order
	ExpectedRouteCount int      `json:"expectedRouteCount,omitempty"` // Total routes to wait for
	ExpectedTotes      []string `json:"expectedTotes,omitempty"`      // Expected tote IDs from all routes
}

// ToteArrivedSignal represents a tote arrival signal for multi-route orders
type ToteArrivedSignal struct {
	ToteID     string `json:"toteId"`
	RouteID    string `json:"routeId"`
	RouteIndex int    `json:"routeIndex"`
	ArrivedAt  string `json:"arrivedAt"`
}

// ConsolidationWorkflow coordinates the consolidation of multi-item orders
func ConsolidationWorkflow(ctx workflow.Context, input map[string]interface{}) error {
	logger := workflow.GetLogger(ctx)

	orderID, _ := input["orderId"].(string)
	waveID, _ := input["waveId"].(string)

	// Extract unit-level tracking fields
	var unitIDs []string
	var pathID string
	if ids, ok := input["unitIds"].([]interface{}); ok {
		for _, id := range ids {
			if strID, ok := id.(string); ok {
				unitIDs = append(unitIDs, strID)
			}
		}
	}
	if pid, ok := input["pathId"].(string); ok {
		pathID = pid
	}
	useUnitTracking := len(unitIDs) > 0

	// Extract multi-route support fields
	isMultiRoute := false
	expectedRouteCount := 1
	var expectedTotes []string

	if mr, ok := input["isMultiRoute"].(bool); ok {
		isMultiRoute = mr
	}
	if erc, ok := input["expectedRouteCount"].(float64); ok {
		expectedRouteCount = int(erc)
	}
	if et, ok := input["expectedTotes"].([]interface{}); ok {
		for _, t := range et {
			if toteID, ok := t.(string); ok {
				expectedTotes = append(expectedTotes, toteID)
			}
		}
	}

	logger.Info("Starting consolidation workflow",
		"orderId", orderID,
		"waveId", waveID,
		"unitCount", len(unitIDs),
		"isMultiRoute", isMultiRoute,
		"expectedRouteCount", expectedRouteCount,
		"expectedTotes", len(expectedTotes),
	)

	// Activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 0: Wait for all totes if multi-route order
	if isMultiRoute && expectedRouteCount > 1 {
		logger.Info("Multi-route order - waiting for tote arrivals",
			"orderId", orderID,
			"expectedRouteCount", expectedRouteCount,
			"expectedTotes", expectedTotes,
		)

		receivedTotes := make(map[string]bool)
		toteArrivalTimeout := 30 * time.Minute // 30 minute timeout for all totes

		// Create a channel for receiving tote arrival signals
		toteArrivedChannel := workflow.GetSignalChannel(ctx, "toteArrived")

		// Use a selector to wait for signals with timeout
		for {
			// Check if we've received all expected totes
			if len(expectedTotes) > 0 {
				allReceived := true
				for _, toteID := range expectedTotes {
					if !receivedTotes[toteID] {
						allReceived = false
						break
					}
				}
				if allReceived {
					logger.Info("All expected totes received", "orderId", orderID, "toteCount", len(receivedTotes))
					break
				}
			} else if len(receivedTotes) >= expectedRouteCount {
				// No specific totes expected, just wait for route count
				logger.Info("All routes received", "orderId", orderID, "routeCount", len(receivedTotes))
				break
			}

			// Wait for signal or timeout
			selector := workflow.NewSelector(ctx)

			selector.AddReceive(toteArrivedChannel, func(c workflow.ReceiveChannel, more bool) {
				var signal ToteArrivedSignal
				c.Receive(ctx, &signal)

				logger.Info("Tote arrived",
					"orderId", orderID,
					"toteId", signal.ToteID,
					"routeId", signal.RouteID,
					"routeIndex", signal.RouteIndex,
				)

				receivedTotes[signal.ToteID] = true
			})

			// Add timeout
			timerCtx, cancelTimer := workflow.WithCancel(ctx)
			timerFuture := workflow.NewTimer(timerCtx, toteArrivalTimeout)
			timedOut := false

			selector.AddFuture(timerFuture, func(f workflow.Future) {
				timedOut = true
				logger.Warn("Timeout waiting for totes",
					"orderId", orderID,
					"receivedCount", len(receivedTotes),
					"expectedCount", expectedRouteCount,
				)
			})

			selector.Select(ctx)
			cancelTimer()

			if timedOut {
				// Continue with partial consolidation
				logger.Warn("Proceeding with partial consolidation after timeout",
					"orderId", orderID,
					"receivedTotes", len(receivedTotes),
				)
				break
			}
		}

		logger.Info("Tote collection complete, proceeding with consolidation",
			"orderId", orderID,
			"totesReceived", len(receivedTotes),
		)
	}

	// Step 1: Create consolidation unit
	logger.Info("Creating consolidation unit", "orderId", orderID, "waveId", waveID)
	var consolidationID string
	err := workflow.ExecuteActivity(ctx, "CreateConsolidationUnit", map[string]interface{}{
		"orderId":     orderID,
		"waveId":      waveID,
		"pickedItems": input["pickedItems"],
	}).Get(ctx, &consolidationID)
	if err != nil {
		return fmt.Errorf("failed to create consolidation unit: %w", err)
	}

	// Step 2: Consolidate items from different totes
	logger.Info("Consolidating items", "orderId", orderID, "consolidationId", consolidationID)
	err = workflow.ExecuteActivity(ctx, "ConsolidateItems", map[string]interface{}{
		"consolidationId": consolidationID,
		"pickedItems":     input["pickedItems"],
	}).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to consolidate items: %w", err)
	}

	// Step 3: Verify all items consolidated
	logger.Info("Verifying consolidation", "orderId", orderID, "consolidationId", consolidationID)
	var verified bool
	err = workflow.ExecuteActivity(ctx, "VerifyConsolidation", consolidationID).Get(ctx, &verified)
	if err != nil {
		return fmt.Errorf("failed to verify consolidation: %w", err)
	}

	if !verified {
		return fmt.Errorf("consolidation verification failed for order %s", orderID)
	}

	// Step 4: Complete consolidation
	logger.Info("Completing consolidation", "orderId", orderID, "consolidationId", consolidationID)
	err = workflow.ExecuteActivity(ctx, "CompleteConsolidation", consolidationID).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to complete consolidation: %w", err)
	}

	// Step 5: Unit-level consolidation confirmation (if unit tracking enabled)
	if useUnitTracking && len(unitIDs) > 0 {
		logger.Info("Confirming unit-level consolidation", "orderId", orderID, "unitCount", len(unitIDs))

		// Use consolidationID as destination bin
		destinationBin := consolidationID
		stationID := "CONSOLIDATION-STATION-DEFAULT"
		workerID := "consolidation-workflow"

		for _, unitID := range unitIDs {
			err := workflow.ExecuteActivity(ctx, "ConfirmUnitConsolidation", map[string]interface{}{
				"unitId":         unitID,
				"destinationBin": destinationBin,
				"workerId":       workerID,
				"stationId":      stationID,
			}).Get(ctx, nil)

			if err != nil {
				logger.Warn("Failed to confirm unit consolidation",
					"orderId", orderID,
					"unitId", unitID,
					"error", err,
				)
				// Continue with other units - partial failure is handled at parent workflow level
			}
		}

		logger.Info("Unit-level consolidation confirmation completed", "orderId", orderID)
	}

	// Suppress unused variable warning
	_ = pathID

	logger.Info("Consolidation completed successfully", "orderId", orderID, "consolidationId", consolidationID)
	return nil
}
