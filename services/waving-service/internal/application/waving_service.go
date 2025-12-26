package application

import (
	"context"
	"fmt"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"

	"github.com/wms-platform/waving-service/internal/domain"
)

// WavingApplicationService handles wave-related use cases
type WavingApplicationService struct {
	repo         domain.WaveRepository
	producer     *kafka.InstrumentedProducer
	eventFactory *cloudevents.EventFactory
	logger       *logging.Logger
}

// NewWavingApplicationService creates a new WavingApplicationService
func NewWavingApplicationService(
	repo domain.WaveRepository,
	producer *kafka.InstrumentedProducer,
	eventFactory *cloudevents.EventFactory,
	logger *logging.Logger,
) *WavingApplicationService {
	return &WavingApplicationService{
		repo:         repo,
		producer:     producer,
		eventFactory: eventFactory,
		logger:       logger,
	}
}

// CreateWave creates a new wave
func (s *WavingApplicationService) CreateWave(ctx context.Context, cmd CreateWaveCommand) (*WaveDTO, error) {
	waveType := domain.WaveType(cmd.WaveType)
	mode := domain.FulfillmentModeWave
	if cmd.FulfillmentMode != "" {
		mode = domain.FulfillmentMode(cmd.FulfillmentMode)
	}

	waveID := generateWaveID(waveType)
	wave, err := domain.NewWave(waveID, waveType, mode, cmd.Configuration)
	if err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if cmd.Zone != "" {
		wave.SetZone(cmd.Zone)
	}

	if err := s.repo.Save(ctx, wave); err != nil {
		s.logger.WithError(err).Error("Failed to create wave", "waveId", waveID)
		return nil, fmt.Errorf("failed to create wave: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Created wave", "waveId", waveID, "waveType", cmd.WaveType)
	return ToWaveDTO(wave), nil
}

// GetWave retrieves a wave by ID
func (s *WavingApplicationService) GetWave(ctx context.Context, query GetWaveQuery) (*WaveDTO, error) {
	wave, err := s.repo.FindByID(ctx, query.WaveID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get wave", "waveId", query.WaveID)
		return nil, fmt.Errorf("failed to get wave: %w", err)
	}

	if wave == nil {
		return nil, errors.ErrNotFound("wave")
	}

	return ToWaveDTO(wave), nil
}

// ListActiveWaves lists all active waves
func (s *WavingApplicationService) ListActiveWaves(ctx context.Context) ([]WaveListDTO, error) {
	waves, err := s.repo.FindActive(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to list waves")
		return nil, fmt.Errorf("failed to list waves: %w", err)
	}

	return ToWaveListDTOs(waves), nil
}

// UpdateWave updates a wave
func (s *WavingApplicationService) UpdateWave(ctx context.Context, cmd UpdateWaveCommand) (*WaveDTO, error) {
	wave, err := s.repo.FindByID(ctx, cmd.WaveID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get wave", "waveId", cmd.WaveID)
		return nil, fmt.Errorf("failed to get wave: %w", err)
	}

	if wave == nil {
		return nil, errors.ErrNotFound("wave")
	}

	if cmd.Priority != nil && *cmd.Priority > 0 {
		wave.SetPriority(*cmd.Priority)
	}
	if cmd.Zone != nil && *cmd.Zone != "" {
		wave.SetZone(*cmd.Zone)
	}

	if err := s.repo.Save(ctx, wave); err != nil {
		s.logger.WithError(err).Error("Failed to update wave", "waveId", cmd.WaveID)
		return nil, fmt.Errorf("failed to update wave: %w", err)
	}

	s.logger.Info("Updated wave", "waveId", cmd.WaveID)
	return ToWaveDTO(wave), nil
}

// DeleteWave deletes a wave
func (s *WavingApplicationService) DeleteWave(ctx context.Context, cmd DeleteWaveCommand) error {
	if err := s.repo.Delete(ctx, cmd.WaveID); err != nil {
		s.logger.WithError(err).Error("Failed to delete wave", "waveId", cmd.WaveID)
		return fmt.Errorf("failed to delete wave: %w", err)
	}

	s.logger.Info("Deleted wave", "waveId", cmd.WaveID)
	return nil
}

// AddOrderToWave adds an order to a wave
func (s *WavingApplicationService) AddOrderToWave(ctx context.Context, cmd AddOrderToWaveCommand) (*WaveDTO, error) {
	wave, err := s.repo.FindByID(ctx, cmd.WaveID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get wave", "waveId", cmd.WaveID)
		return nil, fmt.Errorf("failed to get wave: %w", err)
	}

	if wave == nil {
		return nil, errors.ErrNotFound("wave")
	}

	if err := wave.AddOrder(cmd.Order); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, wave); err != nil {
		s.logger.WithError(err).Error("Failed to add order to wave", "waveId", cmd.WaveID)
		return nil, fmt.Errorf("failed to add order to wave: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Added order to wave", "waveId", cmd.WaveID, "orderId", cmd.Order.OrderID)
	return ToWaveDTO(wave), nil
}

// RemoveOrderFromWave removes an order from a wave
func (s *WavingApplicationService) RemoveOrderFromWave(ctx context.Context, cmd RemoveOrderFromWaveCommand) (*WaveDTO, error) {
	wave, err := s.repo.FindByID(ctx, cmd.WaveID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get wave", "waveId", cmd.WaveID)
		return nil, fmt.Errorf("failed to get wave: %w", err)
	}

	if wave == nil {
		return nil, errors.ErrNotFound("wave")
	}

	if err := wave.RemoveOrder(cmd.OrderID); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, wave); err != nil {
		s.logger.WithError(err).Error("Failed to remove order from wave", "waveId", cmd.WaveID)
		return nil, fmt.Errorf("failed to remove order from wave: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Removed order from wave", "waveId", cmd.WaveID, "orderId", cmd.OrderID)
	return ToWaveDTO(wave), nil
}

// ScheduleWave schedules a wave
func (s *WavingApplicationService) ScheduleWave(ctx context.Context, cmd ScheduleWaveCommand) (*WaveDTO, error) {
	wave, err := s.repo.FindByID(ctx, cmd.WaveID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get wave", "waveId", cmd.WaveID)
		return nil, fmt.Errorf("failed to get wave: %w", err)
	}

	if wave == nil {
		return nil, errors.ErrNotFound("wave")
	}

	if err := wave.Schedule(cmd.ScheduledStart, cmd.ScheduledEnd); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, wave); err != nil {
		s.logger.WithError(err).Error("Failed to schedule wave", "waveId", cmd.WaveID)
		return nil, fmt.Errorf("failed to schedule wave: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Scheduled wave", "waveId", cmd.WaveID, "start", cmd.ScheduledStart)
	return ToWaveDTO(wave), nil
}

// ReleaseWave releases a wave
func (s *WavingApplicationService) ReleaseWave(ctx context.Context, cmd ReleaseWaveCommand) (*WaveDTO, error) {
	wave, err := s.repo.FindByID(ctx, cmd.WaveID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get wave", "waveId", cmd.WaveID)
		return nil, fmt.Errorf("failed to get wave: %w", err)
	}

	if wave == nil {
		return nil, errors.ErrNotFound("wave")
	}

	if err := wave.Release(); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, wave); err != nil {
		s.logger.WithError(err).Error("Failed to release wave", "waveId", cmd.WaveID)
		return nil, fmt.Errorf("failed to release wave: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Released wave", "waveId", cmd.WaveID)
	return ToWaveDTO(wave), nil
}

// CancelWave cancels a wave
func (s *WavingApplicationService) CancelWave(ctx context.Context, cmd CancelWaveCommand) (*WaveDTO, error) {
	wave, err := s.repo.FindByID(ctx, cmd.WaveID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get wave", "waveId", cmd.WaveID)
		return nil, fmt.Errorf("failed to get wave: %w", err)
	}

	if wave == nil {
		return nil, errors.ErrNotFound("wave")
	}

	if err := wave.Cancel(cmd.Reason); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, wave); err != nil {
		s.logger.WithError(err).Error("Failed to cancel wave", "waveId", cmd.WaveID)
		return nil, fmt.Errorf("failed to cancel wave: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Cancelled wave", "waveId", cmd.WaveID, "reason", cmd.Reason)
	return ToWaveDTO(wave), nil
}

// GetWavesByStatus retrieves waves by status
func (s *WavingApplicationService) GetWavesByStatus(ctx context.Context, query GetWavesByStatusQuery) ([]WaveDTO, error) {
	status := domain.WaveStatus(query.Status)
	waves, err := s.repo.FindByStatus(ctx, status)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get waves by status", "status", status)
		return nil, fmt.Errorf("failed to get waves by status: %w", err)
	}

	return ToWaveDTOs(waves), nil
}

// GetWavesByZone retrieves waves by zone
func (s *WavingApplicationService) GetWavesByZone(ctx context.Context, query GetWavesByZoneQuery) ([]WaveDTO, error) {
	waves, err := s.repo.FindByZone(ctx, query.Zone)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get waves by zone", "zone", query.Zone)
		return nil, fmt.Errorf("failed to get waves by zone: %w", err)
	}

	return ToWaveDTOs(waves), nil
}

// GetWaveByOrder retrieves a wave by order ID
func (s *WavingApplicationService) GetWaveByOrder(ctx context.Context, query GetWaveByOrderQuery) (*WaveDTO, error) {
	wave, err := s.repo.FindByOrderID(ctx, query.OrderID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get wave by order", "orderId", query.OrderID)
		return nil, fmt.Errorf("failed to get wave by order: %w", err)
	}

	if wave == nil {
		return nil, errors.ErrNotFound("wave")
	}

	return ToWaveDTO(wave), nil
}

// GetReadyForRelease retrieves waves ready for release
func (s *WavingApplicationService) GetReadyForRelease(ctx context.Context) ([]WaveDTO, error) {
	waves, err := s.repo.FindReadyForRelease(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get waves ready for release")
		return nil, fmt.Errorf("failed to get waves ready for release: %w", err)
	}

	return ToWaveDTOs(waves), nil
}

