package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// PlanningWorkflowInput represents the input for the planning workflow
type PlanningWorkflowInput struct {
	OrderID            string                 `json:"orderId"`
	CustomerID         string                 `json:"customerId"`
	Items              []Item                 `json:"items"`
	Priority           string                 `json:"priority"`
	PromisedDeliveryAt time.Time              `json:"promisedDeliveryAt"`
	IsMultiItem        bool                   `json:"isMultiItem"`
	GiftWrap           bool                   `json:"giftWrap"`
	GiftWrapDetails    *GiftWrapDetailsInput  `json:"giftWrapDetails,omitempty"`
	HazmatDetails      *HazmatDetailsInput    `json:"hazmatDetails,omitempty"`
	ColdChainDetails   *ColdChainDetailsInput `json:"coldChainDetails,omitempty"`
	TotalValue         float64                `json:"totalValue"`
	UnitIDs            []string               `json:"unitIds,omitempty"` // Unit tracking now always enabled
	// Multi-tenant context
	TenantID    string `json:"tenantId"`
	FacilityID  string `json:"facilityId"`
	WarehouseID string `json:"warehouseId"`
}

// PlanningWorkflowResult represents the consolidated planning output
type PlanningWorkflowResult struct {
	ProcessPath        ProcessPathResult `json:"processPath"`
	PathID             string            `json:"pathId,omitempty"`
	WaveID             string            `json:"waveId"`
	WaveScheduledStart time.Time         `json:"waveScheduledStart"`
	ReservedUnitIDs    []string          `json:"reservedUnitIds,omitempty"`
	TargetStationID    string            `json:"targetStationId,omitempty"`    // Pre-assigned station
	RequiredSkills     []string          `json:"requiredSkills,omitempty"`     // Required worker skills
	RequiredEquipment  []string          `json:"requiredEquipment,omitempty"`  // Required equipment types
	EquipmentReserved  map[string]string `json:"equipmentReserved,omitempty"`  // Equipment type -> reservation ID
	Success            bool              `json:"success"`
	Error              string            `json:"error,omitempty"`
}

// PlanningWorkflow coordinates process path determination and wave assignment
// This workflow is executed as a child workflow of OrderFulfillmentWorkflow
func PlanningWorkflow(ctx workflow.Context, input PlanningWorkflowInput) (*PlanningWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting planning workflow", "orderId", input.OrderID)

	result := &PlanningWorkflowResult{
		Success: false,
	}

	// Activity options with retry policy
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: PlanningActivityTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    DefaultRetryInitialInterval,
			BackoffCoefficient: DefaultRetryBackoffCoefficient,
			MaximumInterval:    DefaultRetryMaxInterval,
			MaximumAttempts:    DefaultMaxRetryAttempts,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Create disconnected context for compensation (runs even if workflow cancelled)
	compensationCtx, _ := workflow.NewDisconnectedContext(ctx)
	compensationCtx = workflow.WithActivityOptions(compensationCtx, workflow.ActivityOptions{
		StartToCloseTimeout: PlanningActivityTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    DefaultRetryInitialInterval,
			BackoffCoefficient: DefaultRetryBackoffCoefficient,
			MaximumInterval:    DefaultRetryMaxInterval,
			MaximumAttempts:    DefaultMaxRetryAttempts,
		},
	})

	// Track whether we need to compensate unit reservations
	var needsUnitCompensation bool

	// ========================================
	// Step 1: Determine Process Path
	// ========================================
	logger.Info("Planning Step 1: Determining process path", "orderId", input.OrderID)

	// Build process path items
	processPathItems := make([]map[string]interface{}, len(input.Items))
	for i, item := range input.Items {
		processPathItems[i] = map[string]interface{}{
			"sku":               item.SKU,
			"quantity":          item.Quantity,
			"weight":            item.Weight,
			"isFragile":         item.IsFragile,
			"isHazmat":          item.IsHazmat,
			"requiresColdChain": item.RequiresColdChain,
		}
	}

	processPathInput := map[string]interface{}{
		"orderId":    input.OrderID,
		"items":      processPathItems,
		"giftWrap":   input.GiftWrap,
		"totalValue": input.TotalValue,
	}
	if input.GiftWrapDetails != nil {
		processPathInput["giftWrapDetails"] = input.GiftWrapDetails
	}
	if input.HazmatDetails != nil {
		processPathInput["hazmatDetails"] = input.HazmatDetails
	}
	if input.ColdChainDetails != nil {
		processPathInput["coldChainDetails"] = input.ColdChainDetails
	}

	var processPath ProcessPathResult
	err := workflow.ExecuteActivity(ctx, "DetermineProcessPath", processPathInput).Get(ctx, &processPath)
	if err != nil {
		result.Error = fmt.Sprintf("process path determination failed: %v", err)
		return result, err
	}

	result.ProcessPath = processPath
	logger.Info("Process path determined",
		"orderId", input.OrderID,
		"pathId", processPath.PathID,
		"requirements", processPath.Requirements,
		"consolidationRequired", processPath.ConsolidationRequired,
		"giftWrapRequired", processPath.GiftWrapRequired,
	)

	// ========================================
	// Step 2: Persist Process Path (always enabled)
	// ========================================
	logger.Info("Planning Step 2: Persisting process path", "orderId", input.OrderID)

	var persistPathResult map[string]string
	err = workflow.ExecuteActivity(ctx, "PersistProcessPath", map[string]interface{}{
		"orderId":    input.OrderID,
		"items":      processPathItems,
		"giftWrap":   input.GiftWrap,
		"totalValue": input.TotalValue,
	}).Get(ctx, &persistPathResult)
	if err != nil {
		logger.Warn("Failed to persist process path", "orderId", input.OrderID, "error", err)
		// Non-fatal: generate default path ID for unit tracking
		result.PathID = fmt.Sprintf("path-%s", input.OrderID)
		processPath.PathID = result.PathID
	} else if pathID, ok := persistPathResult["pathId"]; ok {
		result.PathID = pathID
		processPath.PathID = pathID
	} else {
		// PersistProcessPath succeeded but didn't return pathId, generate default
		result.PathID = fmt.Sprintf("path-%s", input.OrderID)
		processPath.PathID = result.PathID
	}

	// ========================================
	// Step 2a: Find Capable Station (Process Path Integration with ML Optimization)
	// ========================================
	logger.Info("Planning Step 2a: Finding optimal station with dynamic routing", "orderId", input.OrderID)

	var targetStationID string
	var routingScore float64
	var routingConfidence float64

	if len(processPath.Requirements) > 0 {
		// Determine station type based on requirements
		stationType := determineStationType(processPath.Requirements)

		// Try ML-based routing optimizer first (ATROPS-like)
		var optimizedResult map[string]interface{}
		optimizeStationInput := map[string]interface{}{
			"orderId":            input.OrderID,
			"priority":           input.Priority,
			"requirements":       processPath.Requirements,
			"specialHandling":    processPath.SpecialHandling,
			"itemCount":          len(input.Items),
			"promisedDeliveryAt": input.PromisedDeliveryAt,
			"requiredSkills":     result.RequiredSkills,
			"requiredEquipment":  result.RequiredEquipment,
			"zone":               "", // Could be enhanced
			"stationType":        stationType,
		}

		err = workflow.ExecuteActivity(ctx, "OptimizeStationSelection", optimizeStationInput).Get(ctx, &optimizedResult)
		if err != nil {
			logger.Warn("ML routing optimizer unavailable, falling back to basic selection",
				"orderId", input.OrderID,
				"error", err,
			)

			// Fallback to basic FindCapableStation
			var stationResult map[string]interface{}
			findStationInput := map[string]interface{}{
				"requirements": processPath.Requirements,
				"stationType":  stationType,
				"zone":         "",
			}

			err = workflow.ExecuteActivity(ctx, "FindCapableStation", findStationInput).Get(ctx, &stationResult)
			if err != nil {
				logger.Warn("Failed to find capable station", "orderId", input.OrderID, "error", err)
				// Non-fatal: continue without station pre-assignment
			} else if stationID, ok := stationResult["stationId"].(string); ok {
				targetStationID = stationID
				result.TargetStationID = stationID
				logger.Info("Station assigned (basic selection)",
					"orderId", input.OrderID,
					"stationId", stationID,
					"requirements", processPath.Requirements,
				)
			}
		} else if success, ok := optimizedResult["success"].(bool); ok && success {
			// Use ML-optimized result
			if stationID, ok := optimizedResult["selectedStationId"].(string); ok {
				targetStationID = stationID
				result.TargetStationID = stationID

				if score, ok := optimizedResult["score"].(float64); ok {
					routingScore = score
				}
				if confidence, ok := optimizedResult["confidence"].(float64); ok {
					routingConfidence = confidence
				}

				logger.Info("Station optimized with ML routing",
					"orderId", input.OrderID,
					"stationId", stationID,
					"score", routingScore,
					"confidence", routingConfidence,
					"reasoning", optimizedResult["reasoning"],
				)
			}
		}
	}

	// Extract required skills from requirements
	result.RequiredSkills = extractRequiredSkills(processPath.Requirements, processPath.SpecialHandling)

	// Extract required equipment from requirements
	result.RequiredEquipment = extractRequiredEquipment(processPath.Requirements)
	result.EquipmentReserved = make(map[string]string)

	// ========================================
	// Step 2b: Reserve Station Capacity (if station assigned)
	// ========================================
	var stationReservationID string
	var needsStationCompensation bool

	if targetStationID != "" {
		logger.Info("Planning Step 2b: Reserving station capacity", "orderId", input.OrderID, "stationId", targetStationID)

		// Generate reservation ID
		stationReservationID = fmt.Sprintf("res-%s-%s", input.OrderID, targetStationID)

		// Determine required slots based on order complexity
		requiredSlots := determineRequiredSlots(len(input.Items), processPath.Requirements)

		var capacityResult map[string]interface{}
		reserveCapacityInput := map[string]interface{}{
			"stationId":     targetStationID,
			"orderId":       input.OrderID,
			"requiredSlots": requiredSlots,
			"reservationId": stationReservationID,
		}

		err = workflow.ExecuteActivity(ctx, "ReserveStationCapacity", reserveCapacityInput).Get(ctx, &capacityResult)
		if err != nil {
			logger.Warn("Failed to reserve station capacity - triggering escalation",
				"orderId", input.OrderID,
				"stationId", targetStationID,
				"error", err,
			)

			// Trigger escalation and find fallback
			newTier, fallbackStations, escalateErr := handleEscalation(
				ctx,
				input.OrderID,
				processPath.PathID,
				false, // station not unavailable, just capacity exceeded
				true,  // capacity exceeded
				false,
				false,
				processPath.Requirements,
			)

			if escalateErr == nil && len(fallbackStations) > 0 {
				// Try fallback station
				fallbackStationID, fbErr := findFallbackStationOnFailure(
					ctx,
					input.OrderID,
					processPath.PathID,
					targetStationID,
					processPath.Requirements,
					processPath.SpecialHandling,
					input.FacilityID,
				)

				if fbErr == nil && fallbackStationID != "" {
					logger.Info("Using fallback station after capacity failure",
						"orderId", input.OrderID,
						"originalStation", targetStationID,
						"fallbackStation", fallbackStationID,
						"newTier", newTier,
					)
					targetStationID = fallbackStationID
					result.TargetStationID = fallbackStationID

					// Try to reserve capacity on fallback station
					stationReservationID = fmt.Sprintf("res-%s-%s", input.OrderID, fallbackStationID)
					reserveCapacityInput["stationId"] = fallbackStationID
					reserveCapacityInput["reservationId"] = stationReservationID

					err = workflow.ExecuteActivity(ctx, "ReserveStationCapacity", reserveCapacityInput).Get(ctx, &capacityResult)
					if err == nil {
						if success, ok := capacityResult["success"].(bool); ok && success {
							needsStationCompensation = true
							logger.Info("Fallback station capacity reserved",
								"orderId", input.OrderID,
								"fallbackStationId", fallbackStationID,
							)
						}
					}
				}
			}

			// If escalation failed or no fallback found, clear target station
			if targetStationID == "" {
				result.TargetStationID = ""
			}
		} else if success, ok := capacityResult["success"].(bool); ok && success {
			needsStationCompensation = true
			logger.Info("Station capacity reserved",
				"orderId", input.OrderID,
				"stationId", targetStationID,
				"reservationId", stationReservationID,
				"remainingCapacity", capacityResult["remainingCapacity"],
			)
		}
	}

	// ========================================
	// Step 2c: Validate Worker Certifications (if skills required)
	// ========================================
	if len(result.RequiredSkills) > 0 {
		logger.Info("Planning Step 2c: Validating worker certifications",
			"orderId", input.OrderID,
			"requiredSkills", result.RequiredSkills,
		)

		var certValidationResult map[string]interface{}
		validateCertInput := map[string]interface{}{
			"requiredSkills": result.RequiredSkills,
			"zone":           "", // Could be enhanced to pass zone
			"minWorkers":     1,  // At least 1 certified worker needed
		}

		err = workflow.ExecuteActivity(ctx, "ValidateWorkerCertification", validateCertInput).Get(ctx, &certValidationResult)
		if err != nil {
			logger.Warn("Failed to validate worker certifications - triggering escalation",
				"orderId", input.OrderID,
				"requiredSkills", result.RequiredSkills,
				"error", err,
			)

			// Trigger escalation for worker unavailability
			_, _, escalateErr := handleEscalation(
				ctx,
				input.OrderID,
				processPath.PathID,
				false,
				false,
				false,
				true, // worker unavailable
				processPath.Requirements,
			)

			if escalateErr != nil {
				logger.Error("Escalation failed for worker unavailability",
					"orderId", input.OrderID,
					"error", escalateErr,
				)
			}
		} else if sufficientLabor, ok := certValidationResult["sufficientLabor"].(bool); ok {
			if !sufficientLabor {
				missingSkills := certValidationResult["missingSkills"]
				logger.Warn("Insufficient certified labor available - triggering escalation",
					"orderId", input.OrderID,
					"requiredSkills", result.RequiredSkills,
					"missingSkills", missingSkills,
					"availableWorkers", certValidationResult["certifiedWorkersAvailable"],
				)

				// Trigger escalation for worker unavailability
				_, _, escalateErr := handleEscalation(
					ctx,
					input.OrderID,
					processPath.PathID,
					false,
					false,
					false,
					true, // worker unavailable
					processPath.Requirements,
				)

				if escalateErr != nil {
					logger.Error("Escalation failed for worker unavailability",
						"orderId", input.OrderID,
						"error", escalateErr,
					)
				}
			} else {
				logger.Info("Worker certifications validated",
					"orderId", input.OrderID,
					"certifiedWorkers", certValidationResult["certifiedWorkersAvailable"],
				)
			}
		}
	}

	// ========================================
	// Step 2d: Check and Reserve Equipment (if equipment required)
	// ========================================
	var equipmentReservations []string
	var needsEquipmentCompensation bool

	if len(result.RequiredEquipment) > 0 {
		logger.Info("Planning Step 2d: Checking equipment availability",
			"orderId", input.OrderID,
			"requiredEquipment", result.RequiredEquipment,
		)

		// Check equipment availability first
		var equipmentAvailability map[string]interface{}
		checkEquipmentInput := map[string]interface{}{
			"equipmentTypes": result.RequiredEquipment,
			"zone":           "", // Could be enhanced to pass zone
			"quantity":       1,  // At least 1 unit of each type needed
		}

		err = workflow.ExecuteActivity(ctx, "CheckEquipmentAvailability", checkEquipmentInput).Get(ctx, &equipmentAvailability)
		if err != nil {
			logger.Warn("Failed to check equipment availability - triggering escalation",
				"orderId", input.OrderID,
				"requiredEquipment", result.RequiredEquipment,
				"error", err,
			)

			// Trigger escalation for equipment unavailability
			_, _, escalateErr := handleEscalation(
				ctx,
				input.OrderID,
				processPath.PathID,
				false,
				false,
				true, // equipment unavailable
				false,
				processPath.Requirements,
			)

			if escalateErr != nil {
				logger.Error("Escalation failed for equipment unavailability",
					"orderId", input.OrderID,
					"error", escalateErr,
				)
			}
		} else if allAvailable, ok := equipmentAvailability["allAvailable"].(bool); ok {
			if !allAvailable {
				insufficientEquipment := equipmentAvailability["insufficientEquipment"]
				logger.Warn("Insufficient equipment available - triggering escalation",
					"orderId", input.OrderID,
					"requiredEquipment", result.RequiredEquipment,
					"insufficientEquipment", insufficientEquipment,
				)

				// Trigger escalation for equipment unavailability
				_, _, escalateErr := handleEscalation(
					ctx,
					input.OrderID,
					processPath.PathID,
					false,
					false,
					true, // equipment unavailable
					false,
					processPath.Requirements,
				)

				if escalateErr != nil {
					logger.Error("Escalation failed for equipment unavailability",
						"orderId", input.OrderID,
						"error", escalateErr,
					)
				}
			} else {
				logger.Info("All required equipment available",
					"orderId", input.OrderID,
					"equipmentTypes", result.RequiredEquipment,
				)

				// Reserve equipment for each type
				for _, equipType := range result.RequiredEquipment {
					reservationID := fmt.Sprintf("eq-res-%s-%s", input.OrderID, equipType)

					var reserveResult map[string]interface{}
					reserveEquipmentInput := map[string]interface{}{
						"equipmentType": equipType,
						"orderId":       input.OrderID,
						"quantity":      1,
						"reservationId": reservationID,
					}

					err = workflow.ExecuteActivity(ctx, "ReserveEquipment", reserveEquipmentInput).Get(ctx, &reserveResult)
					if err != nil {
						logger.Warn("Failed to reserve equipment",
							"orderId", input.OrderID,
							"equipmentType", equipType,
							"error", err,
						)
						// Non-fatal: continue without this equipment
					} else if success, ok := reserveResult["success"].(bool); ok && success {
						result.EquipmentReserved[equipType] = reservationID
						equipmentReservations = append(equipmentReservations, reservationID)
						needsEquipmentCompensation = true
						logger.Info("Equipment reserved",
							"orderId", input.OrderID,
							"equipmentType", equipType,
							"reservationId", reservationID,
						)
					}
				}
			}
		}
	}

	// ========================================
	// Step 3: Reserve Units (always enabled)
	// ========================================
	logger.Info("Planning Step 3: Reserving units", "orderId", input.OrderID)

	if len(input.UnitIDs) > 0 {
		// Use pre-existing unit IDs (units already created and passed in)
		result.ReservedUnitIDs = input.UnitIDs
		logger.Info("Using pre-existing units", "orderId", input.OrderID, "unitCount", len(input.UnitIDs))
	} else {
		// Reserve units from available inventory (units should already exist from receiving)
		reserveItems := make([]map[string]interface{}, len(input.Items))
		for i, item := range input.Items {
			reserveItems[i] = map[string]interface{}{
				"sku":      item.SKU,
				"quantity": item.Quantity,
			}
		}

		var reserveResult map[string]interface{}
		err = workflow.ExecuteActivity(ctx, "ReserveUnits", map[string]interface{}{
			"orderId":   input.OrderID,
			"pathId":    result.PathID,
			"items":     reserveItems,
			"handlerId": "planning-workflow",
		}).Get(ctx, &reserveResult)
		if err != nil {
			result.Error = fmt.Sprintf("unit reservation failed: %v", err)
			return result, err
		}

		// Extract reserved unit IDs
		if reserved, ok := reserveResult["reservedUnits"].([]interface{}); ok {
			for _, u := range reserved {
				if unit, ok := u.(map[string]interface{}); ok {
					if id, ok := unit["unitId"].(string); ok {
						result.ReservedUnitIDs = append(result.ReservedUnitIDs, id)
					}
				}
			}
		}

		// Check for failed reservations
		if failed, ok := reserveResult["failedItems"].([]interface{}); ok && len(failed) > 0 {
			logger.Warn("Some units could not be reserved", "orderId", input.OrderID, "failedCount", len(failed))
			// Continue with partial reservation - workflow will handle partial completion
		}
	}

	logger.Info("Units reserved for order", "orderId", input.OrderID, "unitCount", len(result.ReservedUnitIDs))

	// Mark that we now have unit reservations that may need compensation
	needsUnitCompensation = true

	// ========================================
	// Step 3a: Reserve Inventory in Inventory Service
	// ========================================
	// Create matching reservations in inventory-service for staging
	logger.Info("Planning Step 3a: Reserving inventory in inventory service", "orderId", input.OrderID)

	reserveInventoryItems := make([]map[string]interface{}, len(input.Items))
	for i, item := range input.Items {
		reserveInventoryItems[i] = map[string]interface{}{
			"sku":      item.SKU,
			"quantity": item.Quantity,
		}
	}

	// Set up compensation to release units and station capacity if workflow fails
	defer func() {
		if !workflow.IsReplaying(ctx) && err != nil {
			// Compensate unit reservations
			if needsUnitCompensation {
				logger.Warn("Compensating: releasing unit reservations due to workflow failure",
					"orderId", input.OrderID,
					"error", err)

				releaseErr := workflow.ExecuteActivity(compensationCtx, "ReleaseUnits", map[string]interface{}{
					"orderId": input.OrderID,
					"reason":  "workflow_failed",
				}).Get(compensationCtx, nil)

				if releaseErr != nil {
					logger.Error("Compensation failed: could not release unit reservations",
						"orderId", input.OrderID,
						"error", releaseErr)
				} else {
					logger.Info("Compensation successful: unit reservations released",
						"orderId", input.OrderID)
				}
			}

			// Compensate station capacity reservation
			if needsStationCompensation && targetStationID != "" && stationReservationID != "" {
				logger.Warn("Compensating: releasing station capacity due to workflow failure",
					"orderId", input.OrderID,
					"stationId", targetStationID,
					"error", err)

				releaseCapacityErr := workflow.ExecuteActivity(compensationCtx, "ReleaseStationCapacity", map[string]interface{}{
					"stationId":     targetStationID,
					"orderId":       input.OrderID,
					"reservationId": stationReservationID,
					"reason":        "workflow_failed",
				}).Get(compensationCtx, nil)

				if releaseCapacityErr != nil {
					logger.Error("Compensation failed: could not release station capacity",
						"orderId", input.OrderID,
						"stationId", targetStationID,
						"error", releaseCapacityErr)
				} else {
					logger.Info("Compensation successful: station capacity released",
						"orderId", input.OrderID,
						"stationId", targetStationID)
				}
			}

			// Compensate equipment reservations
			if needsEquipmentCompensation && len(equipmentReservations) > 0 {
				logger.Warn("Compensating: releasing equipment reservations due to workflow failure",
					"orderId", input.OrderID,
					"reservationCount", len(equipmentReservations),
					"error", err)

				for equipType, reservationID := range result.EquipmentReserved {
					releaseEquipErr := workflow.ExecuteActivity(compensationCtx, "ReleaseEquipment", map[string]interface{}{
						"reservationId": reservationID,
						"equipmentType": equipType,
						"orderId":       input.OrderID,
						"reason":        "workflow_failed",
					}).Get(compensationCtx, nil)

					if releaseEquipErr != nil {
						logger.Error("Compensation failed: could not release equipment",
							"orderId", input.OrderID,
							"equipmentType", equipType,
							"reservationId", reservationID,
							"error", releaseEquipErr)
					} else {
						logger.Info("Compensation successful: equipment released",
							"orderId", input.OrderID,
							"equipmentType", equipType,
							"reservationId", reservationID)
					}
				}
			}
		}
	}()

	// Execute ReserveInventory - failure will now be fatal
	err = workflow.ExecuteActivity(ctx, "ReserveInventory", map[string]interface{}{
		"orderId": input.OrderID,
		"items":   reserveInventoryItems,
	}).Get(ctx, nil)
	if err != nil {
		// Create specific error with context
		itemErrors := make([]ItemError, len(input.Items))
		for i, item := range input.Items {
			itemErrors[i] = ItemError{
				SKU:      item.SKU,
				Quantity: item.Quantity,
				Reason:   "reservation_failed",
			}
		}

		inventoryErr := NewInventoryReservationError(
			input.OrderID,
			itemErrors,
			err,
			"Failed to create soft reservation in inventory service",
		)

		logger.Error("Inventory reservation failed - workflow will fail",
			"orderId", input.OrderID,
			"error", inventoryErr)

		result.Error = fmt.Sprintf("inventory reservation failed: %v", inventoryErr)
		return result, inventoryErr
	}

	logger.Info("Inventory reserved in inventory service", "orderId", input.OrderID)
	// Clear compensation flag since we succeeded
	needsUnitCompensation = false

	// ========================================
	// Step 4: Wait for Wave Assignment
	// ========================================
	logger.Info("Planning Step 4: Waiting for wave assignment", "orderId", input.OrderID)

	// Set up signal channel for wave assignment
	var waveAssignment WaveAssignment
	waveSignal := workflow.GetSignalChannel(ctx, "waveAssigned")

	// Wait for wave assignment with timeout based on priority
	waveTimeout := getWaveTimeout(input.Priority)
	waveCtx, cancelWave := workflow.WithCancel(ctx)
	defer cancelWave()

	selector := workflow.NewSelector(waveCtx)

	var waveAssigned bool
	selector.AddReceive(waveSignal, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(waveCtx, &waveAssignment)
		waveAssigned = true
	})

	selector.AddFuture(workflow.NewTimer(waveCtx, waveTimeout), func(f workflow.Future) {
		// Timeout - order not assigned to wave in time
		logger.Warn("Wave assignment timeout", "orderId", input.OrderID, "timeout", waveTimeout)
	})

	selector.Select(waveCtx)

	if !waveAssigned {
		result.Error = "wave assignment timeout"
		return result, fmt.Errorf("wave assignment timeout for order %s", input.OrderID)
	}

	result.WaveID = waveAssignment.WaveID
	result.WaveScheduledStart = waveAssignment.ScheduledStart
	logger.Info("Order assigned to wave", "orderId", input.OrderID, "waveId", waveAssignment.WaveID)

	// ========================================
	// Step 5: Update Order Status
	// ========================================
	logger.Info("Planning Step 5: Updating order status to wave_assigned", "orderId", input.OrderID)

	err = workflow.ExecuteActivity(ctx, "AssignToWave", input.OrderID, waveAssignment.WaveID).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to update order status to wave_assigned", "orderId", input.OrderID, "error", err)
		// Non-fatal: continue
	}

	result.Success = true

	logger.Info("Planning workflow completed",
		"orderId", input.OrderID,
		"waveId", result.WaveID,
		"pathId", processPath.PathID,
	)

	return result, nil
}

// Helper Functions for Station Assignment and Skill Extraction

// determineStationType determines the appropriate station type based on process path requirements
func determineStationType(requirements []string) string {
	// Priority order: specialized requirements first
	for _, req := range requirements {
		switch req {
		case "hazmat":
			return "hazmat_handling"
		case "cold_chain":
			return "cold_storage"
		case "high_value":
			return "secure_packing"
		case "oversized":
			return "oversized_handling"
		case "gift_wrap":
			return "gift_wrap"
		}
	}

	// Default to packing station for multi-item orders
	for _, req := range requirements {
		if req == "multi_item" {
			return "packing"
		}
	}

	// Default station type
	return "packing"
}

// extractRequiredSkills extracts worker skills required from process path requirements and special handling
func extractRequiredSkills(requirements []string, specialHandling []string) []string {
	skillsMap := make(map[string]bool)

	// Map requirements to skills
	for _, req := range requirements {
		switch req {
		case "hazmat":
			skillsMap["hazmat_certification"] = true
		case "cold_chain":
			skillsMap["cold_chain_handling"] = true
		case "high_value":
			skillsMap["high_value_verification"] = true
		case "fragile":
			skillsMap["fragile_handling"] = true
		case "oversized":
			skillsMap["heavy_lifting"] = true
		case "gift_wrap":
			skillsMap["gift_wrapping"] = true
		}
	}

	// Map special handling to skills
	for _, handling := range specialHandling {
		switch handling {
		case "hazmat_compliance":
			skillsMap["hazmat_certification"] = true
		case "cold_chain_packaging":
			skillsMap["cold_chain_handling"] = true
		case "high_value_verification":
			skillsMap["high_value_verification"] = true
		case "fragile_packing":
			skillsMap["fragile_handling"] = true
		case "oversized_handling":
			skillsMap["heavy_lifting"] = true
		}
	}

	// Convert map to slice
	skills := make([]string, 0, len(skillsMap))
	for skill := range skillsMap {
		skills = append(skills, skill)
	}

	return skills
}

// determineRequiredSlots calculates the number of station capacity slots needed for an order
func determineRequiredSlots(itemCount int, requirements []string) int {
	baseSlots := 1

	// Add slots based on item count (more items = more space/time needed)
	if itemCount > 10 {
		baseSlots += 2
	} else if itemCount > 5 {
		baseSlots += 1
	}

	// Add slots for special handling requirements
	for _, req := range requirements {
		switch req {
		case "hazmat", "cold_chain", "high_value":
			// High-priority requirements need extra capacity
			baseSlots += 1
		case "oversized":
			// Oversized items need more physical space
			baseSlots += 2
		case "gift_wrap", "fragile":
			// Extra time/care requirements
			baseSlots += 1
		}
	}

	// Cap at reasonable maximum
	if baseSlots > 5 {
		baseSlots = 5
	}

	return baseSlots
}

// extractRequiredEquipment extracts required equipment types from process path requirements
func extractRequiredEquipment(requirements []string) []string {
	equipmentMap := make(map[string]bool)

	// Map requirements to equipment types
	for _, req := range requirements {
		switch req {
		case "hazmat":
			equipmentMap["hazmat_kit"] = true
			equipmentMap["hazmat_ppe"] = true // Personal protective equipment
		case "cold_chain":
			equipmentMap["cold_storage_unit"] = true
			equipmentMap["temperature_monitor"] = true
		case "oversized":
			equipmentMap["forklift"] = true
			equipmentMap["pallet_jack"] = true
		case "gift_wrap":
			equipmentMap["gift_wrap_station"] = true
		case "fragile":
			equipmentMap["fragile_handling_kit"] = true
		case "high_value":
			equipmentMap["secure_container"] = true
		}
	}

	// Convert map to slice
	equipment := make([]string, 0, len(equipmentMap))
	for equip := range equipmentMap {
		equipment = append(equipment, equip)
	}

	return equipment
}

// handleEscalation handles process path escalation when constraints are detected
func handleEscalation(
	ctx workflow.Context,
	orderID string,
	pathID string,
	stationUnavailable bool,
	capacityExceeded bool,
	equipmentUnavailable bool,
	workerUnavailable bool,
	requirements []string,
) (string, []string, error) {
	logger := workflow.GetLogger(ctx)

	// Step 1: Determine recommended escalation tier
	var tierResult map[string]interface{}
	determineTierInput := map[string]interface{}{
		"pathId":               pathID,
		"orderId":              orderID,
		"stationUnavailable":   stationUnavailable,
		"capacityExceeded":     capacityExceeded,
		"equipmentUnavailable": equipmentUnavailable,
		"workerUnavailable":    workerUnavailable,
		"requirements":         requirements,
	}

	err := workflow.ExecuteActivity(ctx, "DetermineEscalationTier", determineTierInput).Get(ctx, &tierResult)
	if err != nil {
		logger.Error("Failed to determine escalation tier", "orderId", orderID, "error", err)
		return "", nil, err
	}

	recommendedTier, _ := tierResult["recommendedTier"].(string)
	trigger, _ := tierResult["trigger"].(string)
	reason, _ := tierResult["reason"].(string)

	// If no escalation needed, return
	if recommendedTier == "optimal" {
		logger.Info("No escalation needed", "orderId", orderID)
		return "", nil, nil
	}

	logger.Info("Escalating process path",
		"orderId", orderID,
		"pathId", pathID,
		"toTier", recommendedTier,
		"trigger", trigger,
	)

	// Step 2: Execute escalation
	var escalateResult map[string]interface{}
	escalateInput := map[string]interface{}{
		"pathId":      pathID,
		"orderId":     orderID,
		"toTier":      recommendedTier,
		"trigger":     trigger,
		"reason":      reason,
		"escalatedBy": "planning_workflow",
	}

	err = workflow.ExecuteActivity(ctx, "EscalateProcessPath", escalateInput).Get(ctx, &escalateResult)
	if err != nil {
		logger.Error("Failed to escalate process path", "orderId", orderID, "error", err)
		return "", nil, err
	}

	newTier, _ := escalateResult["newTier"].(string)
	fallbackStations := []string{}
	if fb, ok := escalateResult["fallbackStations"].([]interface{}); ok {
		for _, s := range fb {
			if station, ok := s.(string); ok {
				fallbackStations = append(fallbackStations, station)
			}
		}
	}

	logger.Info("Process path escalated",
		"orderId", orderID,
		"pathId", pathID,
		"newTier", newTier,
		"fallbackStationsCount", len(fallbackStations),
	)

	return newTier, fallbackStations, nil
}

// findFallbackStationOnFailure finds fallback stations when primary station fails
func findFallbackStationOnFailure(
	ctx workflow.Context,
	orderID string,
	pathID string,
	failedStationID string,
	requirements []string,
	specialHandling []string,
	facilityID string,
) (string, error) {
	logger := workflow.GetLogger(ctx)

	logger.Info("Finding fallback station",
		"orderId", orderID,
		"failedStationId", failedStationID,
	)

	// Find fallback stations
	var fallbackResult map[string]interface{}
	findFallbackInput := map[string]interface{}{
		"pathId":          pathID,
		"orderId":         orderID,
		"failedStationId": failedStationID,
		"requirements":    requirements,
		"specialHandling": specialHandling,
		"facilityId":      facilityID,
		"maxAlternates":   3,
	}

	err := workflow.ExecuteActivity(ctx, "FindFallbackStations", findFallbackInput).Get(ctx, &fallbackResult)
	if err != nil {
		logger.Error("Failed to find fallback stations", "orderId", orderID, "error", err)
		return "", err
	}

	success, _ := fallbackResult["success"].(bool)
	if !success {
		logger.Warn("No fallback stations available", "orderId", orderID)
		return "", fmt.Errorf("no fallback stations available for order %s", orderID)
	}

	// Extract first fallback station
	if fallbacks, ok := fallbackResult["fallbackStations"].([]interface{}); ok && len(fallbacks) > 0 {
		if fallback, ok := fallbacks[0].(map[string]interface{}); ok {
			if stationID, ok := fallback["stationId"].(string); ok {
				logger.Info("Fallback station found",
					"orderId", orderID,
					"fallbackStationId", stationID,
					"score", fallback["score"],
				)
				return stationID, nil
			}
		}
	}

	return "", fmt.Errorf("no valid fallback stations found for order %s", orderID)
}
