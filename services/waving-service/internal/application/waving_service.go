package application

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/temporal"

	"github.com/wms-platform/waving-service/internal/domain"
	"github.com/wms-platform/waving-service/internal/infrastructure/clients"
)

// WavingApplicationService handles wave-related use cases
type WavingApplicationService struct {
	repo           domain.WaveRepository
	producer       *kafka.InstrumentedProducer
	eventFactory   *cloudevents.EventFactory
	logger         *logging.Logger
	orderClient    *clients.OrderServiceClient
	temporalClient *temporal.Client
}

// NewWavingApplicationService creates a new WavingApplicationService
func NewWavingApplicationService(
	repo domain.WaveRepository,
	producer *kafka.InstrumentedProducer,
	eventFactory *cloudevents.EventFactory,
	logger *logging.Logger,
	orderClient *clients.OrderServiceClient,
	temporalClient *temporal.Client,
) *WavingApplicationService {
	return &WavingApplicationService{
		repo:           repo,
		producer:       producer,
		eventFactory:   eventFactory,
		logger:         logger,
		orderClient:    orderClient,
		temporalClient: temporalClient,
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

	// Log business event: wave created
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "wave.created",
		EntityType: "wave",
		EntityID:   waveID,
		Action:     "created",
		RelatedIDs: map[string]string{
			"waveType": cmd.WaveType,
			"zone":     cmd.Zone,
		},
	})

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

	// Log business event: wave released
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "wave.released",
		EntityType: "wave",
		EntityID:   cmd.WaveID,
		Action:     "released",
		RelatedIDs: map[string]string{
			"orderCount": fmt.Sprintf("%d", wave.GetOrderCount()),
		},
	})

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

	// Log business event: wave cancelled
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "wave.cancelled",
		EntityType: "wave",
		EntityID:   cmd.WaveID,
		Action:     "cancelled",
		RelatedIDs: map[string]string{
			"reason": cmd.Reason,
		},
	})

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

// CreateWaveFromOrders creates a wave from a list of order IDs
func (s *WavingApplicationService) CreateWaveFromOrders(ctx context.Context, cmd CreateWaveFromOrdersCommand) (*CreateWaveFromOrdersResponse, error) {
	if len(cmd.OrderIDs) == 0 {
		return nil, errors.ErrValidation("at least one order ID is required")
	}

	// Generate wave ID and create wave
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

	// Track failed orders
	var failedOrders []string
	scheduledStart := time.Now().Add(30 * time.Minute)

	// Fetch and add each order
	for _, orderID := range cmd.OrderIDs {
		// Fetch order from order-service
		order, err := s.orderClient.GetOrder(ctx, orderID)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to fetch order", "orderId", orderID)
			failedOrders = append(failedOrders, orderID)
			continue
		}

		// Validate order is in correct status
		if order.Status != "validated" {
			s.logger.Warn("Order not in validated status", "orderId", orderID, "status", order.Status)
			failedOrders = append(failedOrders, orderID)
			continue
		}

		// Convert to WaveOrder
		waveOrder := domain.WaveOrder{
			OrderID:            order.OrderID,
			CustomerID:         order.CustomerID,
			Priority:           order.Priority,
			ItemCount:          order.TotalItems,
			TotalWeight:        order.TotalWeight,
			PromisedDeliveryAt: order.PromisedDeliveryAt,
			CarrierCutoff:      order.PromisedDeliveryAt.Add(-4 * time.Hour),
			Zone:               cmd.Zone,
			Status:             "pending",
		}

		if err := wave.AddOrder(waveOrder); err != nil {
			s.logger.WithError(err).Warn("Failed to add order to wave", "orderId", orderID)
			failedOrders = append(failedOrders, orderID)
			continue
		}
	}

	// Validate we have at least one order
	if wave.GetOrderCount() == 0 {
		return nil, errors.ErrValidation("no valid orders to add to wave")
	}

	// Save the wave
	if err := s.repo.Save(ctx, wave); err != nil {
		s.logger.WithError(err).Error("Failed to save wave", "waveId", waveID)
		return nil, fmt.Errorf("failed to save wave: %w", err)
	}

	// Signal each order's Planning workflow with waveAssigned
	if s.temporalClient != nil {
		for _, order := range wave.Orders {
			workflowID := "planning-" + order.OrderID
			signalPayload := WaveAssignedSignal{
				WaveID:         waveID,
				ScheduledStart: scheduledStart,
			}

			err := s.temporalClient.SignalWorkflow(ctx, workflowID, "", "waveAssigned", signalPayload)
			if err != nil {
				s.logger.WithError(err).Warn("Failed to signal workflow",
					"orderId", order.OrderID,
					"workflowId", workflowID)
				// Continue - don't fail the whole operation for signal failures
			} else {
				s.logger.Info("Signaled workflow", "orderId", order.OrderID, "waveId", waveID)
			}
		}
	}

	// Log business event: wave created from orders
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "wave.created",
		EntityType: "wave",
		EntityID:   waveID,
		Action:     "created",
		RelatedIDs: map[string]string{
			"waveType":    cmd.WaveType,
			"zone":        cmd.Zone,
			"orderCount":  fmt.Sprintf("%d", wave.GetOrderCount()),
			"failedCount": fmt.Sprintf("%d", len(failedOrders)),
		},
	})

	return &CreateWaveFromOrdersResponse{
		Wave:         *ToWaveDTO(wave),
		FailedOrders: failedOrders,
	}, nil
}

