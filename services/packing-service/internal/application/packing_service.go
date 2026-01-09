package application

import (
	"context"
	"fmt"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"

	"github.com/wms-platform/packing-service/internal/domain"
)

// PackingApplicationService handles packing-related use cases
type PackingApplicationService struct {
	repo         domain.PackTaskRepository
	producer     *kafka.InstrumentedProducer
	eventFactory *cloudevents.EventFactory
	logger       *logging.Logger
}

// NewPackingApplicationService creates a new PackingApplicationService
func NewPackingApplicationService(
	repo domain.PackTaskRepository,
	producer *kafka.InstrumentedProducer,
	eventFactory *cloudevents.EventFactory,
	logger *logging.Logger,
) *PackingApplicationService {
	return &PackingApplicationService{
		repo:         repo,
		producer:     producer,
		eventFactory: eventFactory,
		logger:       logger,
	}
}

// CreatePackTask creates a new packing task
func (s *PackingApplicationService) CreatePackTask(ctx context.Context, cmd CreatePackTaskCommand) (*PackTaskDTO, error) {
	task, err := domain.NewPackTask(cmd.TaskID, cmd.OrderID, cmd.WaveID, cmd.Items)
	if err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, task); err != nil {
		s.logger.WithError(err).Error("Failed to create pack task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to create pack task: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	// Log business event: packing task created
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "packing.task_created",
		EntityType: "packTask",
		EntityID:   cmd.TaskID,
		Action:     "created",
		RelatedIDs: map[string]string{
			"orderId": cmd.OrderID,
			"waveId":  cmd.WaveID,
		},
	})

	return ToPackTaskDTO(task), nil
}

// GetPackTask retrieves a packing task by ID
func (s *PackingApplicationService) GetPackTask(ctx context.Context, query GetPackTaskQuery) (*PackTaskDTO, error) {
	task, err := s.repo.FindByID(ctx, query.TaskID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get pack task", "taskId", query.TaskID)
		return nil, fmt.Errorf("failed to get pack task: %w", err)
	}

	if task == nil {
		return nil, errors.ErrNotFound("pack task")
	}

	return ToPackTaskDTO(task), nil
}

// AssignPackTask assigns a packing task to a packer
func (s *PackingApplicationService) AssignPackTask(ctx context.Context, cmd AssignPackTaskCommand) (*PackTaskDTO, error) {
	task, err := s.repo.FindByID(ctx, cmd.TaskID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get pack task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to get pack task: %w", err)
	}

	if task == nil {
		return nil, errors.ErrNotFound("pack task")
	}

	if err := task.Assign(cmd.PackerID, cmd.Station); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, task); err != nil {
		s.logger.WithError(err).Error("Failed to save pack task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to save pack task: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Assigned pack task", "taskId", cmd.TaskID, "packerId", cmd.PackerID)
	return ToPackTaskDTO(task), nil
}

// StartPackTask starts a packing task
func (s *PackingApplicationService) StartPackTask(ctx context.Context, cmd StartPackTaskCommand) (*PackTaskDTO, error) {
	task, err := s.repo.FindByID(ctx, cmd.TaskID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get pack task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to get pack task: %w", err)
	}

	if task == nil {
		return nil, errors.ErrNotFound("pack task")
	}

	if err := task.Start(); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, task); err != nil {
		s.logger.WithError(err).Error("Failed to save pack task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to save pack task: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Started pack task", "taskId", cmd.TaskID)
	return ToPackTaskDTO(task), nil
}

// VerifyItem verifies an item during packing
func (s *PackingApplicationService) VerifyItem(ctx context.Context, cmd VerifyItemCommand) (*PackTaskDTO, error) {
	task, err := s.repo.FindByID(ctx, cmd.TaskID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get pack task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to get pack task: %w", err)
	}

	if task == nil {
		return nil, errors.ErrNotFound("pack task")
	}

	if err := task.VerifyItem(cmd.SKU); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, task); err != nil {
		s.logger.WithError(err).Error("Failed to save pack task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to save pack task: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Verified item", "taskId", cmd.TaskID, "sku", cmd.SKU)
	return ToPackTaskDTO(task), nil
}

// SelectPackaging selects packaging for the task
func (s *PackingApplicationService) SelectPackaging(ctx context.Context, cmd SelectPackagingCommand) (*PackTaskDTO, error) {
	task, err := s.repo.FindByID(ctx, cmd.TaskID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get pack task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to get pack task: %w", err)
	}

	if task == nil {
		return nil, errors.ErrNotFound("pack task")
	}

	if err := task.SelectPackaging(cmd.PackageType, cmd.Dimensions, cmd.Materials); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, task); err != nil {
		s.logger.WithError(err).Error("Failed to save pack task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to save pack task: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Selected packaging", "taskId", cmd.TaskID, "packageType", cmd.PackageType)
	return ToPackTaskDTO(task), nil
}

// SealPackage seals the package
func (s *PackingApplicationService) SealPackage(ctx context.Context, cmd SealPackageCommand) (*PackTaskDTO, error) {
	task, err := s.repo.FindByID(ctx, cmd.TaskID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get pack task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to get pack task: %w", err)
	}

	if task == nil {
		return nil, errors.ErrNotFound("pack task")
	}

	if err := task.SealPackage(); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, task); err != nil {
		s.logger.WithError(err).Error("Failed to save pack task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to save pack task: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Sealed package", "taskId", cmd.TaskID, "packageId", task.Package.PackageID)
	return ToPackTaskDTO(task), nil
}

// ApplyLabel applies a shipping label to the package
func (s *PackingApplicationService) ApplyLabel(ctx context.Context, cmd ApplyLabelCommand) (*PackTaskDTO, error) {
	task, err := s.repo.FindByID(ctx, cmd.TaskID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get pack task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to get pack task: %w", err)
	}

	if task == nil {
		return nil, errors.ErrNotFound("pack task")
	}

	if err := task.ApplyLabel(cmd.Label); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, task); err != nil {
		s.logger.WithError(err).Error("Failed to save pack task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to save pack task: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	// Log business event: label applied
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "packing.label_applied",
		EntityType: "packTask",
		EntityID:   cmd.TaskID,
		Action:     "label_applied",
		RelatedIDs: map[string]string{
			"packageId":      task.Package.PackageID,
			"trackingNumber": cmd.Label.TrackingNumber,
		},
	})

	return ToPackTaskDTO(task), nil
}

// CompletePackTask completes a packing task
func (s *PackingApplicationService) CompletePackTask(ctx context.Context, cmd CompletePackTaskCommand) (*PackTaskDTO, error) {
	task, err := s.repo.FindByID(ctx, cmd.TaskID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get pack task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to get pack task: %w", err)
	}

	if task == nil {
		return nil, errors.ErrNotFound("pack task")
	}

	if err := task.Complete(); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, task); err != nil {
		s.logger.WithError(err).Error("Failed to save pack task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to save pack task: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	// Log business event: packing task completed
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "packing.task_completed",
		EntityType: "packTask",
		EntityID:   cmd.TaskID,
		Action:     "completed",
		RelatedIDs: map[string]string{
			"packageId": task.Package.PackageID,
		},
	})

	return ToPackTaskDTO(task), nil
}

// GetByOrder retrieves a packing task by order ID
func (s *PackingApplicationService) GetByOrder(ctx context.Context, query GetByOrderQuery) (*PackTaskDTO, error) {
	task, err := s.repo.FindByOrderID(ctx, query.OrderID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get pack task by order", "orderId", query.OrderID)
		return nil, fmt.Errorf("failed to get pack task by order: %w", err)
	}

	if task == nil {
		return nil, errors.ErrNotFound("pack task")
	}

	return ToPackTaskDTO(task), nil
}

// GetByWave retrieves packing tasks by wave ID
func (s *PackingApplicationService) GetByWave(ctx context.Context, query GetByWaveQuery) ([]PackTaskDTO, error) {
	tasks, err := s.repo.FindByWaveID(ctx, query.WaveID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get pack tasks by wave", "waveId", query.WaveID)
		return nil, fmt.Errorf("failed to get pack tasks by wave: %w", err)
	}

	return ToPackTaskDTOs(tasks), nil
}

// GetByTracking retrieves a packing task by tracking number
func (s *PackingApplicationService) GetByTracking(ctx context.Context, query GetByTrackingQuery) (*PackTaskDTO, error) {
	task, err := s.repo.FindByTrackingNumber(ctx, query.TrackingNumber)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get pack task by tracking", "trackingNumber", query.TrackingNumber)
		return nil, fmt.Errorf("failed to get pack task by tracking: %w", err)
	}

	if task == nil {
		return nil, errors.ErrNotFound("pack task")
	}

	return ToPackTaskDTO(task), nil
}

// GetPending retrieves pending packing tasks
func (s *PackingApplicationService) GetPending(ctx context.Context, query GetPendingQuery) ([]PackTaskDTO, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 50 // Default limit
	}

	tasks, err := s.repo.FindPending(ctx, limit)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get pending pack tasks")
		return nil, fmt.Errorf("failed to get pending pack tasks: %w", err)
	}

	return ToPackTaskDTOs(tasks), nil
}
