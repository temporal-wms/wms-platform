package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wms-platform/services/stow-service/internal/domain"
	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/logging"
)

// StowService implements the application layer for stow operations
type StowService struct {
	taskRepo     domain.PutawayTaskRepository
	locationRepo domain.StorageLocationRepository
	logger       *logging.Logger
}

// NewStowService creates a new StowService
func NewStowService(
	taskRepo domain.PutawayTaskRepository,
	locationRepo domain.StorageLocationRepository,
	logger *logging.Logger,
) *StowService {
	return &StowService{
		taskRepo:     taskRepo,
		locationRepo: locationRepo,
		logger:       logger,
	}
}

// CreatePutawayTaskCommand represents the command to create a putaway task
type CreatePutawayTaskCommand struct {
	ShipmentID        string
	SKU               string
	ProductName       string
	Quantity          int
	SourceToteID      string
	SourceLocationID  string
	IsHazmat          bool
	RequiresColdChain bool
	IsOversized       bool
	IsFragile         bool
	Weight            float64
	Priority          int
	Strategy          string
}

// CreatePutawayTask creates a new putaway task
func (s *StowService) CreatePutawayTask(ctx context.Context, cmd CreatePutawayTaskCommand) (*domain.PutawayTask, error) {
	taskID := fmt.Sprintf("PTW-%s", uuid.New().String()[:8])

	constraints := domain.ItemConstraints{
		IsHazmat:          cmd.IsHazmat,
		RequiresColdChain: cmd.RequiresColdChain,
		IsOversized:       cmd.IsOversized,
		IsFragile:         cmd.IsFragile,
		Weight:            cmd.Weight,
	}

	task := domain.NewPutawayTask(
		taskID,
		cmd.ShipmentID,
		cmd.SKU,
		cmd.ProductName,
		cmd.Quantity,
		cmd.SourceToteID,
		constraints,
	)

	task.SourceLocationID = cmd.SourceLocationID

	if cmd.Priority > 0 && cmd.Priority <= 5 {
		task.Priority = cmd.Priority
	}

	if cmd.Strategy != "" {
		strategy := domain.StorageStrategy(cmd.Strategy)
		if err := task.SetStrategy(strategy); err != nil {
			return nil, errors.NewValidationError("invalid storage strategy", err)
		}
	}

	if err := s.taskRepo.Save(ctx, task); err != nil {
		s.logger.WithError(err).Error("Failed to save putaway task")
		return nil, errors.NewInternalError("failed to save putaway task", err)
	}

	s.logger.Info("Created putaway task",
		"taskId", task.TaskID,
		"sku", task.SKU,
		"quantity", task.Quantity,
		"strategy", task.Strategy,
	)

	return task, nil
}

// AssignTaskCommand represents the command to assign a task
type AssignTaskCommand struct {
	TaskID   string
	WorkerID string
}

// AssignTask assigns a task to a worker and finds a storage location
func (s *StowService) AssignTask(ctx context.Context, cmd AssignTaskCommand) (*domain.PutawayTask, error) {
	task, err := s.taskRepo.FindByID(ctx, cmd.TaskID)
	if err != nil {
		return nil, errors.NewInternalError("failed to find task", err)
	}
	if task == nil {
		return nil, errors.NewNotFoundError("task not found", nil)
	}

	// Assign worker
	if err := task.AssignToWorker(cmd.WorkerID); err != nil {
		return nil, errors.NewValidationError("cannot assign task", err)
	}

	// Find storage location based on strategy
	if task.TargetLocationID == "" {
		location, err := s.findStorageLocation(ctx, task)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to find storage location", "taskId", task.TaskID)
		} else if location != nil {
			if err := task.AssignLocation(*location); err != nil {
				s.logger.WithError(err).Warn("Failed to assign location", "taskId", task.TaskID)
			}
		}
	}

	if err := s.taskRepo.Save(ctx, task); err != nil {
		return nil, errors.NewInternalError("failed to save task", err)
	}

	s.logger.Info("Assigned putaway task",
		"taskId", task.TaskID,
		"workerId", cmd.WorkerID,
		"locationId", task.TargetLocationID,
	)

	return task, nil
}

// findStorageLocation finds a storage location based on the task's strategy
func (s *StowService) findStorageLocation(ctx context.Context, task *domain.PutawayTask) (*domain.StorageLocation, error) {
	constraints := domain.LocationConstraints{
		MinCapacity:       task.Quantity,
		MinWeight:         task.Constraints.Weight,
		RequiresHazmat:    task.Constraints.IsHazmat,
		RequiresColdChain: task.Constraints.RequiresColdChain,
		RequiresOversized: task.Constraints.IsOversized,
	}

	availableLocations, err := s.locationRepo.FindAvailableLocations(ctx, constraints, 20)
	if err != nil {
		return nil, err
	}

	if len(availableLocations) == 0 {
		return nil, domain.ErrNoAvailableLocations
	}

	switch task.Strategy {
	case domain.StorageChaotic:
		// Use random selection for chaotic storage
		return task.SelectRandomLocation(availableLocations)
	default:
		// For other strategies, return the first available
		return &availableLocations[0], nil
	}
}

// StartTaskCommand represents the command to start a task
type StartTaskCommand struct {
	TaskID string
}

// StartTask starts a putaway task
func (s *StowService) StartTask(ctx context.Context, cmd StartTaskCommand) (*domain.PutawayTask, error) {
	task, err := s.taskRepo.FindByID(ctx, cmd.TaskID)
	if err != nil {
		return nil, errors.NewInternalError("failed to find task", err)
	}
	if task == nil {
		return nil, errors.NewNotFoundError("task not found", nil)
	}

	if err := task.Start(); err != nil {
		return nil, errors.NewValidationError("cannot start task", err)
	}

	if err := s.taskRepo.Save(ctx, task); err != nil {
		return nil, errors.NewInternalError("failed to save task", err)
	}

	return task, nil
}

// RecordStowCommand represents the command to record stowing progress
type RecordStowCommand struct {
	TaskID   string
	Quantity int
}

// RecordStow records stowing progress
func (s *StowService) RecordStow(ctx context.Context, cmd RecordStowCommand) (*domain.PutawayTask, error) {
	task, err := s.taskRepo.FindByID(ctx, cmd.TaskID)
	if err != nil {
		return nil, errors.NewInternalError("failed to find task", err)
	}
	if task == nil {
		return nil, errors.NewNotFoundError("task not found", nil)
	}

	if err := task.RecordStow(cmd.Quantity); err != nil {
		return nil, errors.NewValidationError("cannot record stow", err)
	}

	if err := s.taskRepo.Save(ctx, task); err != nil {
		return nil, errors.NewInternalError("failed to save task", err)
	}

	return task, nil
}

// CompleteTaskCommand represents the command to complete a task
type CompleteTaskCommand struct {
	TaskID string
}

// CompleteTask completes a putaway task
func (s *StowService) CompleteTask(ctx context.Context, cmd CompleteTaskCommand) (*domain.PutawayTask, error) {
	task, err := s.taskRepo.FindByID(ctx, cmd.TaskID)
	if err != nil {
		return nil, errors.NewInternalError("failed to find task", err)
	}
	if task == nil {
		return nil, errors.NewNotFoundError("task not found", nil)
	}

	// If not all items stowed yet, stow the remaining
	if task.RemainingQuantity() > 0 {
		if err := task.RecordStow(task.RemainingQuantity()); err != nil {
			return nil, errors.NewValidationError("cannot record final stow", err)
		}
	}

	if err := task.Complete(); err != nil {
		return nil, errors.NewValidationError("cannot complete task", err)
	}

	// Update location capacity
	if task.TargetLocationID != "" {
		if err := s.locationRepo.UpdateCapacity(ctx, task.TargetLocationID, task.StowedQuantity, task.Constraints.Weight*float64(task.StowedQuantity)); err != nil {
			s.logger.WithError(err).Warn("Failed to update location capacity", "locationId", task.TargetLocationID)
		}
	}

	if err := s.taskRepo.Save(ctx, task); err != nil {
		return nil, errors.NewInternalError("failed to save task", err)
	}

	s.logger.Info("Completed putaway task",
		"taskId", task.TaskID,
		"sku", task.SKU,
		"stowedQuantity", task.StowedQuantity,
		"locationId", task.TargetLocationID,
		"duration", time.Since(*task.StartedAt),
	)

	return task, nil
}

// FailTaskCommand represents the command to fail a task
type FailTaskCommand struct {
	TaskID string
	Reason string
}

// FailTask marks a task as failed
func (s *StowService) FailTask(ctx context.Context, cmd FailTaskCommand) (*domain.PutawayTask, error) {
	task, err := s.taskRepo.FindByID(ctx, cmd.TaskID)
	if err != nil {
		return nil, errors.NewInternalError("failed to find task", err)
	}
	if task == nil {
		return nil, errors.NewNotFoundError("task not found", nil)
	}

	if err := task.Fail(cmd.Reason); err != nil {
		return nil, errors.NewValidationError("cannot fail task", err)
	}

	if err := s.taskRepo.Save(ctx, task); err != nil {
		return nil, errors.NewInternalError("failed to save task", err)
	}

	return task, nil
}

// GetTask retrieves a task by ID
func (s *StowService) GetTask(ctx context.Context, taskID string) (*domain.PutawayTask, error) {
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, errors.NewInternalError("failed to find task", err)
	}
	if task == nil {
		return nil, errors.NewNotFoundError("task not found", nil)
	}
	return task, nil
}

// GetPendingTasks retrieves pending tasks
func (s *StowService) GetPendingTasks(ctx context.Context, limit int) ([]*domain.PutawayTask, error) {
	return s.taskRepo.FindPendingTasks(ctx, limit)
}

// GetTasksByStatus retrieves tasks by status
func (s *StowService) GetTasksByStatus(ctx context.Context, status domain.PutawayStatus, pagination domain.Pagination) ([]*domain.PutawayTask, error) {
	return s.taskRepo.FindByStatus(ctx, status, pagination)
}

// GetTasksByWorker retrieves tasks by worker
func (s *StowService) GetTasksByWorker(ctx context.Context, workerID string, pagination domain.Pagination) ([]*domain.PutawayTask, error) {
	return s.taskRepo.FindByWorkerID(ctx, workerID, pagination)
}

// GetTasksByShipment retrieves tasks by shipment
func (s *StowService) GetTasksByShipment(ctx context.Context, shipmentID string) ([]*domain.PutawayTask, error) {
	return s.taskRepo.FindByShipmentID(ctx, shipmentID)
}
