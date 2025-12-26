package activities

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/wms-platform/packing-service/internal/domain"
	"go.temporal.io/sdk/activity"
)

// PackingActivities contains activities for the packing workflow
type PackingActivities struct {
	repo   domain.PackTaskRepository
	logger *slog.Logger
}

// NewPackingActivities creates a new PackingActivities instance
func NewPackingActivities(repo domain.PackTaskRepository, logger *slog.Logger) *PackingActivities {
	return &PackingActivities{
		repo:   repo,
		logger: logger,
	}
}

// CreatePackTask creates a new pack task
func (a *PackingActivities) CreatePackTask(ctx context.Context, orderID string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Creating pack task", "orderId", orderID)

	// Generate task ID
	taskID := "PK-" + uuid.New().String()[:8]

	// In a real implementation, we'd get the items from a previous step
	// For now, create sample items
	items := []domain.PackItem{
		{
			SKU:         "SKU-001",
			ProductName: "Sample Product",
			Quantity:    1,
			Weight:      0.5,
			Fragile:     false,
			Verified:    false,
		},
	}

	waveID := "WAVE-" + uuid.New().String()[:8]

	// Create the pack task
	task, err := domain.NewPackTask(taskID, orderID, waveID, items)
	if err != nil {
		logger.Error("Failed to create pack task", "error", err)
		return "", fmt.Errorf("failed to create pack task: %w", err)
	}

	// Save to repository
	if err := a.repo.Save(ctx, task); err != nil {
		logger.Error("Failed to save pack task", "error", err)
		return "", fmt.Errorf("failed to save pack task: %w", err)
	}

	logger.Info("Pack task created", "taskId", taskID, "itemCount", len(items))
	return taskID, nil
}

// AssignPacker assigns a packer to a pack task
func (a *PackingActivities) AssignPacker(ctx context.Context, input map[string]string) error {
	logger := activity.GetLogger(ctx)

	taskID := input["taskId"]
	packerID := input["packerId"]
	station := input["station"]

	logger.Info("Assigning packer", "taskId", taskID, "packerId", packerID)

	task, err := a.repo.FindByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to find pack task: %w", err)
	}

	if task == nil {
		return fmt.Errorf("pack task not found: %s", taskID)
	}

	if err := task.Assign(packerID, station); err != nil {
		return fmt.Errorf("failed to assign packer: %w", err)
	}

	if err := a.repo.Save(ctx, task); err != nil {
		return fmt.Errorf("failed to save pack task: %w", err)
	}

	logger.Info("Packer assigned", "taskId", taskID, "packerId", packerID)
	return nil
}

// StartPacking starts the packing process
func (a *PackingActivities) StartPacking(ctx context.Context, taskID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting packing", "taskId", taskID)

	task, err := a.repo.FindByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to find pack task: %w", err)
	}

	if task == nil {
		return fmt.Errorf("pack task not found: %s", taskID)
	}

	if err := task.Start(); err != nil {
		return fmt.Errorf("failed to start packing: %w", err)
	}

	if err := a.repo.Save(ctx, task); err != nil {
		return fmt.Errorf("failed to save pack task: %w", err)
	}

	logger.Info("Packing started", "taskId", taskID)
	return nil
}

// VerifyItem verifies an item in the pack task
func (a *PackingActivities) VerifyItem(ctx context.Context, taskID, sku string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Verifying item", "taskId", taskID, "sku", sku)

	task, err := a.repo.FindByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to find pack task: %w", err)
	}

	if task == nil {
		return fmt.Errorf("pack task not found: %s", taskID)
	}

	if err := task.VerifyItem(sku); err != nil {
		return fmt.Errorf("failed to verify item: %w", err)
	}

	if err := a.repo.Save(ctx, task); err != nil {
		return fmt.Errorf("failed to save pack task: %w", err)
	}

	logger.Info("Item verified", "taskId", taskID, "sku", sku)
	return nil
}

// SelectPackaging selects the packaging for a task
func (a *PackingActivities) SelectPackaging(ctx context.Context, taskID string, packageType string, dimensions domain.Dimensions, materials []string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Selecting packaging", "taskId", taskID, "type", packageType)

	task, err := a.repo.FindByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to find pack task: %w", err)
	}

	if task == nil {
		return fmt.Errorf("pack task not found: %s", taskID)
	}

	if err := task.SelectPackaging(domain.PackageType(packageType), dimensions, materials); err != nil {
		return fmt.Errorf("failed to select packaging: %w", err)
	}

	if err := a.repo.Save(ctx, task); err != nil {
		return fmt.Errorf("failed to save pack task: %w", err)
	}

	logger.Info("Packaging selected", "taskId", taskID, "type", packageType)
	return nil
}

// SealPackage seals the package
func (a *PackingActivities) SealPackage(ctx context.Context, taskID string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Sealing package", "taskId", taskID)

	task, err := a.repo.FindByID(ctx, taskID)
	if err != nil {
		return "", fmt.Errorf("failed to find pack task: %w", err)
	}

	if task == nil {
		return "", fmt.Errorf("pack task not found: %s", taskID)
	}

	if err := task.SealPackage(); err != nil {
		return "", fmt.Errorf("failed to seal package: %w", err)
	}

	if err := a.repo.Save(ctx, task); err != nil {
		return "", fmt.Errorf("failed to save pack task: %w", err)
	}

	logger.Info("Package sealed", "taskId", taskID, "packageId", task.Package.PackageID)
	return task.Package.PackageID, nil
}

// ApplyLabel applies a shipping label
func (a *PackingActivities) ApplyLabel(ctx context.Context, taskID, trackingNumber, carrier, serviceType, labelURL string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Applying label", "taskId", taskID, "tracking", trackingNumber)

	task, err := a.repo.FindByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to find pack task: %w", err)
	}

	if task == nil {
		return fmt.Errorf("pack task not found: %s", taskID)
	}

	label := domain.ShippingLabel{
		TrackingNumber: trackingNumber,
		Carrier:        carrier,
		ServiceType:    serviceType,
		LabelURL:       labelURL,
	}

	if err := task.ApplyLabel(label); err != nil {
		return fmt.Errorf("failed to apply label: %w", err)
	}

	if err := a.repo.Save(ctx, task); err != nil {
		return fmt.Errorf("failed to save pack task: %w", err)
	}

	logger.Info("Label applied", "taskId", taskID, "tracking", trackingNumber)
	return nil
}

// CompletePackTask marks a pack task as completed
func (a *PackingActivities) CompletePackTask(ctx context.Context, taskID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Completing pack task", "taskId", taskID)

	task, err := a.repo.FindByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to find pack task: %w", err)
	}

	if task == nil {
		return fmt.Errorf("pack task not found: %s", taskID)
	}

	// If the task is labeled, complete it
	if task.Status == domain.PackTaskStatusLabeled {
		if err := task.Complete(); err != nil {
			logger.Warn("Failed to complete pack task", "taskId", taskID, "error", err)
		}
	}

	if err := a.repo.Save(ctx, task); err != nil {
		return fmt.Errorf("failed to save pack task: %w", err)
	}

	logger.Info("Pack task completed", "taskId", taskID)
	return nil
}
