package application

import (
	"context"
	"fmt"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"

	"github.com/wms-platform/picking-service/internal/domain"
)

// PickingApplicationService handles picking-related use cases
type PickingApplicationService struct {
	repo         domain.PickTaskRepository
	producer     *kafka.InstrumentedProducer
	eventFactory *cloudevents.EventFactory
	logger       *logging.Logger
}

// NewPickingApplicationService creates a new PickingApplicationService
func NewPickingApplicationService(
	repo domain.PickTaskRepository,
	producer *kafka.InstrumentedProducer,
	eventFactory *cloudevents.EventFactory,
	logger *logging.Logger,
) *PickingApplicationService {
	return &PickingApplicationService{
		repo:         repo,
		producer:     producer,
		eventFactory: eventFactory,
		logger:       logger,
	}
}

// CreatePickTask creates a new pick task
func (s *PickingApplicationService) CreatePickTask(ctx context.Context, cmd CreatePickTaskCommand) (*PickTaskDTO, error) {
	method := domain.PickMethodSingle
	if cmd.Method != "" {
		method = domain.PickMethod(cmd.Method)
	}

	task, err := domain.NewPickTask(cmd.TaskID, cmd.OrderID, cmd.WaveID, cmd.RouteID, method, cmd.Items)
	if err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, task); err != nil {
		s.logger.WithError(err).Error("Failed to create pick task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to create pick task: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Created pick task", "taskId", cmd.TaskID, "orderId", cmd.OrderID)
	return ToPickTaskDTO(task), nil
}

// GetPickTask retrieves a pick task by ID
func (s *PickingApplicationService) GetPickTask(ctx context.Context, query GetPickTaskQuery) (*PickTaskDTO, error) {
	task, err := s.repo.FindByID(ctx, query.TaskID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get pick task", "taskId", query.TaskID)
		return nil, fmt.Errorf("failed to get pick task: %w", err)
	}

	if task == nil {
		return nil, errors.ErrNotFound("pick task")
	}

	return ToPickTaskDTO(task), nil
}

// AssignTask assigns a pick task to a picker
func (s *PickingApplicationService) AssignTask(ctx context.Context, cmd AssignTaskCommand) (*PickTaskDTO, error) {
	task, err := s.repo.FindByID(ctx, cmd.TaskID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get pick task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to get pick task: %w", err)
	}

	if task == nil {
		return nil, errors.ErrNotFound("pick task")
	}

	if err := task.Assign(cmd.PickerID, cmd.ToteID); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, task); err != nil {
		s.logger.WithError(err).Error("Failed to save pick task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to save pick task: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	// Log business event: task assigned
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "task.assigned",
		EntityType: "pickTask",
		EntityID:   cmd.TaskID,
		Action:     "assigned",
		RelatedIDs: map[string]string{
			"workerId": cmd.PickerID,
			"toteId":   cmd.ToteID,
		},
	})

	return ToPickTaskDTO(task), nil
}

// StartTask starts a pick task
func (s *PickingApplicationService) StartTask(ctx context.Context, cmd StartTaskCommand) (*PickTaskDTO, error) {
	task, err := s.repo.FindByID(ctx, cmd.TaskID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get pick task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to get pick task: %w", err)
	}

	if task == nil {
		return nil, errors.ErrNotFound("pick task")
	}

	if err := task.Start(); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, task); err != nil {
		s.logger.WithError(err).Error("Failed to save pick task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to save pick task: %w", err)
	}

	s.logger.Info("Started pick task", "taskId", cmd.TaskID)
	return ToPickTaskDTO(task), nil
}

// ConfirmPick confirms an item was picked
func (s *PickingApplicationService) ConfirmPick(ctx context.Context, cmd ConfirmPickCommand) (*PickTaskDTO, error) {
	task, err := s.repo.FindByID(ctx, cmd.TaskID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get pick task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to get pick task: %w", err)
	}

	if task == nil {
		return nil, errors.ErrNotFound("pick task")
	}

	if err := task.ConfirmPick(cmd.SKU, cmd.LocationID, cmd.PickedQty, cmd.ToteID); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, task); err != nil {
		s.logger.WithError(err).Error("Failed to save pick task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to save pick task: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	// Log business event: item picked
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "item.picked",
		EntityType: "pickTask",
		EntityID:   cmd.TaskID,
		Action:     "picked",
		RelatedIDs: map[string]string{
			"sku":      cmd.SKU,
			"quantity": fmt.Sprintf("%d", cmd.PickedQty),
			"toteId":   cmd.ToteID,
		},
	})

	return ToPickTaskDTO(task), nil
}

// ReportException reports a pick exception
func (s *PickingApplicationService) ReportException(ctx context.Context, cmd ReportExceptionCommand) (*PickTaskDTO, error) {
	task, err := s.repo.FindByID(ctx, cmd.TaskID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get pick task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to get pick task: %w", err)
	}

	if task == nil {
		return nil, errors.ErrNotFound("pick task")
	}

	if err := task.ReportException(cmd.SKU, cmd.LocationID, cmd.Reason, cmd.RequestedQty, cmd.AvailableQty); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, task); err != nil {
		s.logger.WithError(err).Error("Failed to save pick task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to save pick task: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	// Log business event: picking exception
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "picking.exception",
		EntityType: "pickTask",
		EntityID:   cmd.TaskID,
		Action:     "exception",
		RelatedIDs: map[string]string{
			"sku":    cmd.SKU,
			"reason": cmd.Reason,
		},
	})

	return ToPickTaskDTO(task), nil
}

// CompleteTask completes a pick task
func (s *PickingApplicationService) CompleteTask(ctx context.Context, cmd CompleteTaskCommand) (*PickTaskDTO, error) {
	task, err := s.repo.FindByID(ctx, cmd.TaskID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get pick task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to get pick task: %w", err)
	}

	if task == nil {
		return nil, errors.ErrNotFound("pick task")
	}

	if err := task.Complete(); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, task); err != nil {
		s.logger.WithError(err).Error("Failed to save pick task", "taskId", cmd.TaskID)
		return nil, fmt.Errorf("failed to save pick task: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	// Log business event: task completed
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "task.completed",
		EntityType: "pickTask",
		EntityID:   cmd.TaskID,
		Action:     "completed",
		RelatedIDs: map[string]string{
			"orderId": task.OrderID,
		},
	})

	return ToPickTaskDTO(task), nil
}

// GetTasksByOrder retrieves pick tasks by order ID
func (s *PickingApplicationService) GetTasksByOrder(ctx context.Context, query GetTasksByOrderQuery) ([]PickTaskDTO, error) {
	tasks, err := s.repo.FindByOrderID(ctx, query.OrderID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get tasks by order", "orderId", query.OrderID)
		return nil, fmt.Errorf("failed to get tasks by order: %w", err)
	}

	return ToPickTaskDTOs(tasks), nil
}

// GetTasksByWave retrieves pick tasks by wave ID
func (s *PickingApplicationService) GetTasksByWave(ctx context.Context, query GetTasksByWaveQuery) ([]PickTaskDTO, error) {
	tasks, err := s.repo.FindByWaveID(ctx, query.WaveID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get tasks by wave", "waveId", query.WaveID)
		return nil, fmt.Errorf("failed to get tasks by wave: %w", err)
	}

	return ToPickTaskDTOs(tasks), nil
}

// GetTasksByPicker retrieves pick tasks by picker ID
func (s *PickingApplicationService) GetTasksByPicker(ctx context.Context, query GetTasksByPickerQuery) ([]PickTaskDTO, error) {
	tasks, err := s.repo.FindByPickerID(ctx, query.PickerID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get tasks by picker", "pickerId", query.PickerID)
		return nil, fmt.Errorf("failed to get tasks by picker: %w", err)
	}

	return ToPickTaskDTOs(tasks), nil
}

// GetActiveTask retrieves the active task for a picker
func (s *PickingApplicationService) GetActiveTask(ctx context.Context, query GetActiveTaskQuery) (*PickTaskDTO, error) {
	task, err := s.repo.FindActiveByPicker(ctx, query.PickerID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get active task", "pickerId", query.PickerID)
		return nil, fmt.Errorf("failed to get active task: %w", err)
	}

	if task == nil {
		return nil, errors.ErrNotFound("active task")
	}

	return ToPickTaskDTO(task), nil
}

// GetPendingTasks retrieves pending pick tasks
func (s *PickingApplicationService) GetPendingTasks(ctx context.Context, query GetPendingTasksQuery) ([]PickTaskDTO, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 50 // Default limit
	}

	tasks, err := s.repo.FindPendingByZone(ctx, query.Zone, limit)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get pending tasks", "zone", query.Zone)
		return nil, fmt.Errorf("failed to get pending tasks: %w", err)
	}

	return ToPickTaskDTOs(tasks), nil
}

// ListTasks retrieves pick tasks with optional status filter
func (s *PickingApplicationService) ListTasks(ctx context.Context, query ListTasksQuery) ([]PickTaskDTO, error) {
	var tasks []*domain.PickTask
	var err error

	if query.Status != "" {
		status := domain.PickTaskStatus(query.Status)
		tasks, err = s.repo.FindByStatus(ctx, status)
	} else {
		// If no status specified, return pending tasks
		limit := query.Limit
		if limit <= 0 {
			limit = 50
		}
		tasks, err = s.repo.FindPendingByZone(ctx, query.Zone, limit)
	}

	if err != nil {
		s.logger.WithError(err).Error("Failed to list tasks", "status", query.Status)
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	// Apply limit if specified
	if query.Limit > 0 && len(tasks) > query.Limit {
		tasks = tasks[:query.Limit]
	}

	return ToPickTaskDTOs(tasks), nil
}
