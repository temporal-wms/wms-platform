package activities

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/wms-platform/picking-service/internal/domain"
	"go.temporal.io/sdk/activity"
)

// PickingActivities contains activities for the picking workflow
type PickingActivities struct {
	repo   domain.PickTaskRepository
	logger *slog.Logger
}

// NewPickingActivities creates a new PickingActivities instance
func NewPickingActivities(repo domain.PickTaskRepository, logger *slog.Logger) *PickingActivities {
	return &PickingActivities{
		repo:   repo,
		logger: logger,
	}
}

// CreatePickTaskInput represents input for creating a pick task
type CreatePickTaskInput struct {
	OrderID string        `json:"orderId"`
	WaveID  string        `json:"waveId"`
	RouteID string        `json:"routeId"`
	Stops   []interface{} `json:"stops"`
}

// CreatePickTask creates a new pick task
func (a *PickingActivities) CreatePickTask(ctx context.Context, input map[string]interface{}) (string, error) {
	logger := activity.GetLogger(ctx)

	orderID, _ := input["orderId"].(string)
	waveID, _ := input["waveId"].(string)
	routeID, _ := input["routeId"].(string)
	stopsRaw, _ := input["stops"].([]interface{})

	logger.Info("Creating pick task", "orderId", orderID, "waveId", waveID)

	// Generate task ID
	taskID := "PT-" + uuid.New().String()[:8]

	// Convert stops to pick items
	items := make([]domain.PickItem, 0)
	for i, stopRaw := range stopsRaw {
		if stop, ok := stopRaw.(map[string]interface{}); ok {
			sku, _ := stop["sku"].(string)
			locationID, _ := stop["locationId"].(string)
			quantity, _ := stop["quantity"].(float64)

			items = append(items, domain.PickItem{
				SKU:       sku,
				Quantity:  int(quantity),
				PickedQty: 0,
				Status:    "pending",
				Location: domain.Location{
					LocationID: locationID,
					Zone:       fmt.Sprintf("ZONE-%d", (i%4)+1),
				},
			})
		}
	}

	// Create the pick task
	task, err := domain.NewPickTask(taskID, orderID, waveID, routeID, domain.PickMethodSingle, items)
	if err != nil {
		logger.Error("Failed to create pick task", "error", err)
		return "", fmt.Errorf("failed to create pick task: %w", err)
	}

	// Save to repository
	if err := a.repo.Save(ctx, task); err != nil {
		logger.Error("Failed to save pick task", "error", err)
		return "", fmt.Errorf("failed to save pick task: %w", err)
	}

	logger.Info("Pick task created", "taskId", taskID, "itemCount", len(items))
	return taskID, nil
}

// CompletePickTask marks a pick task as completed
func (a *PickingActivities) CompletePickTask(ctx context.Context, taskID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Completing pick task", "taskId", taskID)

	task, err := a.repo.FindByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to find pick task: %w", err)
	}

	if task == nil {
		return fmt.Errorf("pick task not found: %s", taskID)
	}

	// If task is in progress, complete it
	if task.Status == domain.PickTaskStatusInProgress {
		if err := task.Complete(); err != nil {
			logger.Warn("Failed to complete task", "taskId", taskID, "error", err)
		}
	}

	// Save updated task
	if err := a.repo.Save(ctx, task); err != nil {
		return fmt.Errorf("failed to save pick task: %w", err)
	}

	logger.Info("Pick task completed", "taskId", taskID)
	return nil
}

// AssignWorker assigns a worker to a pick task
func (a *PickingActivities) AssignWorker(ctx context.Context, taskID, workerID, toteID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Assigning worker to pick task", "taskId", taskID, "workerId", workerID)

	task, err := a.repo.FindByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to find pick task: %w", err)
	}

	if task == nil {
		return fmt.Errorf("pick task not found: %s", taskID)
	}

	if err := task.Assign(workerID, toteID); err != nil {
		return fmt.Errorf("failed to assign worker: %w", err)
	}

	if err := a.repo.Save(ctx, task); err != nil {
		return fmt.Errorf("failed to save pick task: %w", err)
	}

	logger.Info("Worker assigned to pick task", "taskId", taskID, "workerId", workerID)
	return nil
}

// StartPicking starts the picking process
func (a *PickingActivities) StartPicking(ctx context.Context, taskID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting picking", "taskId", taskID)

	task, err := a.repo.FindByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to find pick task: %w", err)
	}

	if task == nil {
		return fmt.Errorf("pick task not found: %s", taskID)
	}

	if err := task.Start(); err != nil {
		return fmt.Errorf("failed to start picking: %w", err)
	}

	if err := a.repo.Save(ctx, task); err != nil {
		return fmt.Errorf("failed to save pick task: %w", err)
	}

	logger.Info("Picking started", "taskId", taskID)
	return nil
}

// ConfirmPick confirms an item has been picked
func (a *PickingActivities) ConfirmPick(ctx context.Context, taskID, sku, locationID, toteID string, quantity int) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Confirming pick", "taskId", taskID, "sku", sku, "quantity", quantity)

	task, err := a.repo.FindByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to find pick task: %w", err)
	}

	if task == nil {
		return fmt.Errorf("pick task not found: %s", taskID)
	}

	if err := task.ConfirmPick(sku, locationID, quantity, toteID); err != nil {
		return fmt.Errorf("failed to confirm pick: %w", err)
	}

	if err := a.repo.Save(ctx, task); err != nil {
		return fmt.Errorf("failed to save pick task: %w", err)
	}

	logger.Info("Pick confirmed", "taskId", taskID, "sku", sku, "quantity", quantity)
	return nil
}

// ReportException reports a picking exception
func (a *PickingActivities) ReportException(ctx context.Context, taskID, sku, locationID, reason string, requestedQty, availableQty int) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Reporting pick exception", "taskId", taskID, "sku", sku, "reason", reason)

	task, err := a.repo.FindByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to find pick task: %w", err)
	}

	if task == nil {
		return fmt.Errorf("pick task not found: %s", taskID)
	}

	if err := task.ReportException(sku, locationID, reason, requestedQty, availableQty); err != nil {
		return fmt.Errorf("failed to report exception: %w", err)
	}

	if err := a.repo.Save(ctx, task); err != nil {
		return fmt.Errorf("failed to save pick task: %w", err)
	}

	logger.Info("Exception reported", "taskId", taskID, "sku", sku, "reason", reason)
	return nil
}
