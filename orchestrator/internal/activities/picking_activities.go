package activities

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/wms-platform/orchestrator/internal/activities/clients"
	"go.temporal.io/sdk/activity"
)

// CreatePickTask creates a pick task from route information
func (a *PickingActivities) CreatePickTask(ctx context.Context, input map[string]interface{}) (string, error) {
	logger := activity.GetLogger(ctx)

	orderID, _ := input["orderId"].(string)
	waveID, _ := input["waveId"].(string)
	routeRaw, _ := input["route"].(map[string]interface{})

	logger.Info("Creating pick task", "orderId", orderID, "waveId", waveID)

	// Extract route ID and stops from route data
	routeID, _ := routeRaw["routeId"].(string)
	stopsRaw, _ := routeRaw["stops"].([]interface{})

	// Convert stops to pick items
	items := make([]clients.PickItem, 0)
	for _, stopRaw := range stopsRaw {
		if stop, ok := stopRaw.(map[string]interface{}); ok {
			sku, _ := stop["sku"].(string)
			quantity, _ := stop["quantity"].(float64)
			locationID, _ := stop["locationId"].(string)
			items = append(items, clients.PickItem{
				SKU:        sku,
				Quantity:   int(quantity),
				LocationID: locationID,
			})
		}
	}

	// Generate task ID
	taskID := "PT-" + uuid.New().String()[:8]

	// Call picking-service to create task
	task, err := a.clients.CreatePickTask(ctx, &clients.CreatePickTaskRequest{
		TaskID:  taskID,
		OrderID: orderID,
		WaveID:  waveID,
		RouteID: routeID,
		Items:   items,
	})
	if err != nil {
		logger.Error("Failed to create pick task", "orderId", orderID, "error", err)
		return "", fmt.Errorf("failed to create pick task: %w", err)
	}

	logger.Info("Pick task created successfully",
		"orderId", orderID,
		"taskId", task.TaskID,
		"itemCount", len(items),
	)

	return task.TaskID, nil
}

// AssignPickerToTask assigns an available worker to a pick task
func (a *PickingActivities) AssignPickerToTask(ctx context.Context, input map[string]interface{}) (string, error) {
	logger := activity.GetLogger(ctx)

	taskID, _ := input["taskId"].(string)
	waveID, _ := input["waveId"].(string)

	logger.Info("Assigning picker to task", "taskId", taskID, "waveId", waveID)

	// Get available workers from labor service
	workers, err := a.clients.GetAvailableWorkers(ctx, "picking", "")
	if err != nil {
		logger.Warn("Failed to get available workers, using default worker", "error", err)
		workers = nil
	}

	// Select a worker (first available or default)
	var pickerID string
	if len(workers) > 0 {
		pickerID = workers[0].WorkerID
	} else {
		// Use a default picker ID for simulation
		pickerID = "PK-" + uuid.New().String()[:8]
		logger.Info("Using simulated picker", "pickerId", pickerID)
	}

	// Generate a tote ID for the pick task
	toteID := "TOTE-" + uuid.New().String()[:8]

	// Assign the picker to the task with a tote
	err = a.clients.AssignPickTask(ctx, taskID, pickerID, toteID)
	if err != nil {
		logger.Error("Failed to assign picker to task", "taskId", taskID, "pickerId", pickerID, "toteId", toteID, "error", err)
		return "", fmt.Errorf("failed to assign worker: %w", err)
	}

	logger.Info("Picker assigned successfully",
		"taskId", taskID,
		"pickerId", pickerID,
		"toteId", toteID,
	)

	return pickerID, nil
}
