package application

import (
	"context"
	"fmt"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"

	"github.com/wms-platform/labor-service/internal/domain"
)

// LaborApplicationService handles labor-related use cases
type LaborApplicationService struct {
	repo         domain.WorkerRepository
	producer     *kafka.InstrumentedProducer
	eventFactory *cloudevents.EventFactory
	logger       *logging.Logger
}

// NewLaborApplicationService creates a new LaborApplicationService
func NewLaborApplicationService(
	repo domain.WorkerRepository,
	producer *kafka.InstrumentedProducer,
	eventFactory *cloudevents.EventFactory,
	logger *logging.Logger,
) *LaborApplicationService {
	return &LaborApplicationService{
		repo:         repo,
		producer:     producer,
		eventFactory: eventFactory,
		logger:       logger,
	}
}

// CreateWorker creates a new worker
func (s *LaborApplicationService) CreateWorker(ctx context.Context, cmd CreateWorkerCommand) (*WorkerDTO, error) {
	worker := domain.NewWorker(cmd.WorkerID, cmd.EmployeeID, cmd.Name)

	if err := s.repo.Save(ctx, worker); err != nil {
		s.logger.WithError(err).Error("Failed to save worker", "workerId", worker.WorkerID)
		return nil, fmt.Errorf("failed to save worker: %w", err)
	}

	s.logger.Info("Created worker", "workerId", worker.WorkerID)
	return ToWorkerDTO(worker), nil
}

// GetWorker retrieves a worker by ID
func (s *LaborApplicationService) GetWorker(ctx context.Context, query GetWorkerQuery) (*WorkerDTO, error) {
	worker, err := s.repo.FindByID(ctx, query.WorkerID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get worker", "workerId", query.WorkerID)
		return nil, fmt.Errorf("failed to get worker: %w", err)
	}

	if worker == nil {
		return nil, errors.ErrNotFound("worker")
	}

	return ToWorkerDTO(worker), nil
}

// StartShift starts a shift for a worker
func (s *LaborApplicationService) StartShift(ctx context.Context, cmd StartShiftCommand) (*WorkerDTO, error) {
	worker, err := s.repo.FindByID(ctx, cmd.WorkerID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get worker", "workerId", cmd.WorkerID)
		return nil, fmt.Errorf("failed to get worker: %w", err)
	}

	if worker == nil {
		return nil, errors.ErrNotFound("worker")
	}

	if err := worker.StartShift(cmd.ShiftID, cmd.ShiftType, cmd.Zone); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, worker); err != nil {
		s.logger.WithError(err).Error("Failed to save worker", "workerId", cmd.WorkerID)
		return nil, fmt.Errorf("failed to save worker: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Started shift", "workerId", cmd.WorkerID, "shiftId", cmd.ShiftID)
	return ToWorkerDTO(worker), nil
}

// EndShift ends a shift for a worker
func (s *LaborApplicationService) EndShift(ctx context.Context, cmd EndShiftCommand) (*WorkerDTO, error) {
	worker, err := s.repo.FindByID(ctx, cmd.WorkerID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get worker", "workerId", cmd.WorkerID)
		return nil, fmt.Errorf("failed to get worker: %w", err)
	}

	if worker == nil {
		return nil, errors.ErrNotFound("worker")
	}

	if err := worker.EndShift(); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, worker); err != nil {
		s.logger.WithError(err).Error("Failed to save worker", "workerId", cmd.WorkerID)
		return nil, fmt.Errorf("failed to save worker: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Ended shift", "workerId", cmd.WorkerID)
	return ToWorkerDTO(worker), nil
}

// StartBreak starts a break for a worker
func (s *LaborApplicationService) StartBreak(ctx context.Context, cmd StartBreakCommand) (*WorkerDTO, error) {
	worker, err := s.repo.FindByID(ctx, cmd.WorkerID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get worker", "workerId", cmd.WorkerID)
		return nil, fmt.Errorf("failed to get worker: %w", err)
	}

	if worker == nil {
		return nil, errors.ErrNotFound("worker")
	}

	if err := worker.StartBreak(cmd.BreakType); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, worker); err != nil {
		s.logger.WithError(err).Error("Failed to save worker", "workerId", cmd.WorkerID)
		return nil, fmt.Errorf("failed to save worker: %w", err)
	}

	s.logger.Info("Started break", "workerId", cmd.WorkerID, "breakType", cmd.BreakType)
	return ToWorkerDTO(worker), nil
}

// EndBreak ends a break for a worker
func (s *LaborApplicationService) EndBreak(ctx context.Context, cmd EndBreakCommand) (*WorkerDTO, error) {
	worker, err := s.repo.FindByID(ctx, cmd.WorkerID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get worker", "workerId", cmd.WorkerID)
		return nil, fmt.Errorf("failed to get worker: %w", err)
	}

	if worker == nil {
		return nil, errors.ErrNotFound("worker")
	}

	if err := worker.EndBreak(); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, worker); err != nil {
		s.logger.WithError(err).Error("Failed to save worker", "workerId", cmd.WorkerID)
		return nil, fmt.Errorf("failed to save worker: %w", err)
	}

	s.logger.Info("Ended break", "workerId", cmd.WorkerID)
	return ToWorkerDTO(worker), nil
}

// AssignTask assigns a task to a worker
func (s *LaborApplicationService) AssignTask(ctx context.Context, cmd AssignTaskCommand) (*WorkerDTO, error) {
	worker, err := s.repo.FindByID(ctx, cmd.WorkerID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get worker", "workerId", cmd.WorkerID)
		return nil, fmt.Errorf("failed to get worker: %w", err)
	}

	if worker == nil {
		return nil, errors.ErrNotFound("worker")
	}

	if err := worker.AssignTask(cmd.TaskID, cmd.TaskType, cmd.Priority); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, worker); err != nil {
		s.logger.WithError(err).Error("Failed to save worker", "workerId", cmd.WorkerID)
		return nil, fmt.Errorf("failed to save worker: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Assigned task", "workerId", cmd.WorkerID, "taskId", cmd.TaskID)
	return ToWorkerDTO(worker), nil
}

// StartTask starts the current task for a worker
func (s *LaborApplicationService) StartTask(ctx context.Context, cmd StartTaskCommand) (*WorkerDTO, error) {
	worker, err := s.repo.FindByID(ctx, cmd.WorkerID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get worker", "workerId", cmd.WorkerID)
		return nil, fmt.Errorf("failed to get worker: %w", err)
	}

	if worker == nil {
		return nil, errors.ErrNotFound("worker")
	}

	if err := worker.StartTask(); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, worker); err != nil {
		s.logger.WithError(err).Error("Failed to save worker", "workerId", cmd.WorkerID)
		return nil, fmt.Errorf("failed to save worker: %w", err)
	}

	s.logger.Info("Started task", "workerId", cmd.WorkerID)
	return ToWorkerDTO(worker), nil
}

// CompleteTask completes the current task for a worker
func (s *LaborApplicationService) CompleteTask(ctx context.Context, cmd CompleteTaskCommand) (*WorkerDTO, error) {
	worker, err := s.repo.FindByID(ctx, cmd.WorkerID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get worker", "workerId", cmd.WorkerID)
		return nil, fmt.Errorf("failed to get worker: %w", err)
	}

	if worker == nil {
		return nil, errors.ErrNotFound("worker")
	}

	if err := worker.CompleteTask(cmd.ItemsProcessed); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, worker); err != nil {
		s.logger.WithError(err).Error("Failed to save worker", "workerId", cmd.WorkerID)
		return nil, fmt.Errorf("failed to save worker: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Completed task", "workerId", cmd.WorkerID, "itemsProcessed", cmd.ItemsProcessed)
	return ToWorkerDTO(worker), nil
}

// AddSkill adds a skill to a worker
func (s *LaborApplicationService) AddSkill(ctx context.Context, cmd AddSkillCommand) (*WorkerDTO, error) {
	worker, err := s.repo.FindByID(ctx, cmd.WorkerID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get worker", "workerId", cmd.WorkerID)
		return nil, fmt.Errorf("failed to get worker: %w", err)
	}

	if worker == nil {
		return nil, errors.ErrNotFound("worker")
	}

	worker.AddSkill(cmd.TaskType, cmd.Level, cmd.Certified)

	if err := s.repo.Save(ctx, worker); err != nil {
		s.logger.WithError(err).Error("Failed to save worker", "workerId", cmd.WorkerID)
		return nil, fmt.Errorf("failed to save worker: %w", err)
	}

	s.logger.Info("Added skill", "workerId", cmd.WorkerID, "taskType", cmd.TaskType)
	return ToWorkerDTO(worker), nil
}

// GetByStatus retrieves workers by status
func (s *LaborApplicationService) GetByStatus(ctx context.Context, query GetByStatusQuery) ([]WorkerDTO, error) {
	workers, err := s.repo.FindByStatus(ctx, query.Status)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get workers by status", "status", query.Status)
		return nil, fmt.Errorf("failed to get workers by status: %w", err)
	}

	return ToWorkerDTOs(workers), nil
}

// GetByZone retrieves workers by zone
func (s *LaborApplicationService) GetByZone(ctx context.Context, query GetByZoneQuery) ([]WorkerDTO, error) {
	workers, err := s.repo.FindByZone(ctx, query.Zone)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get workers by zone", "zone", query.Zone)
		return nil, fmt.Errorf("failed to get workers by zone: %w", err)
	}

	return ToWorkerDTOs(workers), nil
}

// GetAvailable retrieves available workers
func (s *LaborApplicationService) GetAvailable(ctx context.Context, query GetAvailableQuery) ([]WorkerDTO, error) {
	workers, err := s.repo.FindByStatus(ctx, domain.WorkerStatusAvailable)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get available workers")
		return nil, fmt.Errorf("failed to get available workers: %w", err)
	}

	// Filter by zone if specified
	if query.Zone != "" {
		filtered := make([]*domain.Worker, 0)
		for _, worker := range workers {
			if worker.CurrentZone == query.Zone {
				filtered = append(filtered, worker)
			}
		}
		workers = filtered
	}

	return ToWorkerDTOs(workers), nil
}

// ListWorkers retrieves all workers
func (s *LaborApplicationService) ListWorkers(ctx context.Context, query ListWorkersQuery) ([]WorkerDTO, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 50 // Default limit
	}

	workers, err := s.repo.FindAll(ctx, limit, query.Offset)
	if err != nil {
		s.logger.WithError(err).Error("Failed to list workers")
		return nil, fmt.Errorf("failed to list workers: %w", err)
	}

	return ToWorkerDTOs(workers), nil
}
