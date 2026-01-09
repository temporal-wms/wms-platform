package application

import (
	"context"
	"fmt"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"

	"github.com/wms-platform/consolidation-service/internal/domain"
)

// ConsolidationApplicationService handles consolidation-related use cases
type ConsolidationApplicationService struct {
	repo         domain.ConsolidationRepository
	producer     *kafka.InstrumentedProducer
	eventFactory *cloudevents.EventFactory
	logger       *logging.Logger
}

// NewConsolidationApplicationService creates a new ConsolidationApplicationService
func NewConsolidationApplicationService(
	repo domain.ConsolidationRepository,
	producer *kafka.InstrumentedProducer,
	eventFactory *cloudevents.EventFactory,
	logger *logging.Logger,
) *ConsolidationApplicationService {
	return &ConsolidationApplicationService{
		repo:         repo,
		producer:     producer,
		eventFactory: eventFactory,
		logger:       logger,
	}
}

// CreateConsolidation creates a new consolidation unit
func (s *ConsolidationApplicationService) CreateConsolidation(ctx context.Context, cmd CreateConsolidationCommand) (*ConsolidationDTO, error) {
	strategy := domain.StrategyOrderBased
	if cmd.Strategy != "" {
		strategy = domain.ConsolidationStrategy(cmd.Strategy)
	}

	unit, err := domain.NewConsolidationUnit(cmd.ConsolidationID, cmd.OrderID, cmd.WaveID, strategy, cmd.Items)
	if err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, unit); err != nil {
		s.logger.WithError(err).Error("Failed to create consolidation", "consolidationId", cmd.ConsolidationID)
		return nil, fmt.Errorf("failed to create consolidation: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	// Log business event: consolidation started
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "consolidation.started",
		EntityType: "consolidation",
		EntityID:   cmd.ConsolidationID,
		Action:     "started",
		RelatedIDs: map[string]string{
			"orderId": cmd.OrderID,
			"waveId":  cmd.WaveID,
		},
	})

	return ToConsolidationDTO(unit), nil
}

// GetConsolidation retrieves a consolidation unit by ID
func (s *ConsolidationApplicationService) GetConsolidation(ctx context.Context, query GetConsolidationQuery) (*ConsolidationDTO, error) {
	unit, err := s.repo.FindByID(ctx, query.ConsolidationID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get consolidation", "consolidationId", query.ConsolidationID)
		return nil, fmt.Errorf("failed to get consolidation: %w", err)
	}

	if unit == nil {
		return nil, errors.ErrNotFound("consolidation")
	}

	return ToConsolidationDTO(unit), nil
}

// AssignStation assigns a station to a consolidation unit
func (s *ConsolidationApplicationService) AssignStation(ctx context.Context, cmd AssignStationCommand) (*ConsolidationDTO, error) {
	unit, err := s.repo.FindByID(ctx, cmd.ConsolidationID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get consolidation", "consolidationId", cmd.ConsolidationID)
		return nil, fmt.Errorf("failed to get consolidation: %w", err)
	}

	if unit == nil {
		return nil, errors.ErrNotFound("consolidation")
	}

	if err := unit.AssignStation(cmd.Station, cmd.WorkerID, cmd.DestinationBin); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, unit); err != nil {
		s.logger.WithError(err).Error("Failed to save consolidation", "consolidationId", cmd.ConsolidationID)
		return nil, fmt.Errorf("failed to save consolidation: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Assigned station to consolidation", "consolidationId", cmd.ConsolidationID, "station", cmd.Station)
	return ToConsolidationDTO(unit), nil
}

// ConsolidateItem consolidates an item into the unit
func (s *ConsolidationApplicationService) ConsolidateItem(ctx context.Context, cmd ConsolidateItemCommand) (*ConsolidationDTO, error) {
	unit, err := s.repo.FindByID(ctx, cmd.ConsolidationID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get consolidation", "consolidationId", cmd.ConsolidationID)
		return nil, fmt.Errorf("failed to get consolidation: %w", err)
	}

	if unit == nil {
		return nil, errors.ErrNotFound("consolidation")
	}

	if err := unit.ConsolidateItem(cmd.SKU, cmd.Quantity, cmd.SourceToteID, cmd.VerifiedBy); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, unit); err != nil {
		s.logger.WithError(err).Error("Failed to save consolidation", "consolidationId", cmd.ConsolidationID)
		return nil, fmt.Errorf("failed to save consolidation: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Consolidated item", "consolidationId", cmd.ConsolidationID, "sku", cmd.SKU, "quantity", cmd.Quantity)
	return ToConsolidationDTO(unit), nil
}

// CompleteConsolidation completes a consolidation unit
func (s *ConsolidationApplicationService) CompleteConsolidation(ctx context.Context, cmd CompleteConsolidationCommand) (*ConsolidationDTO, error) {
	unit, err := s.repo.FindByID(ctx, cmd.ConsolidationID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get consolidation", "consolidationId", cmd.ConsolidationID)
		return nil, fmt.Errorf("failed to get consolidation: %w", err)
	}

	if unit == nil {
		return nil, errors.ErrNotFound("consolidation")
	}

	if err := unit.Complete(); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, unit); err != nil {
		s.logger.WithError(err).Error("Failed to save consolidation", "consolidationId", cmd.ConsolidationID)
		return nil, fmt.Errorf("failed to save consolidation: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	// Log business event: consolidation completed
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "consolidation.completed",
		EntityType: "consolidation",
		EntityID:   cmd.ConsolidationID,
		Action:     "completed",
	})

	return ToConsolidationDTO(unit), nil
}

// GetByOrder retrieves a consolidation unit by order ID
func (s *ConsolidationApplicationService) GetByOrder(ctx context.Context, query GetByOrderQuery) (*ConsolidationDTO, error) {
	unit, err := s.repo.FindByOrderID(ctx, query.OrderID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get consolidation by order", "orderId", query.OrderID)
		return nil, fmt.Errorf("failed to get consolidation by order: %w", err)
	}

	if unit == nil {
		return nil, errors.ErrNotFound("consolidation")
	}

	return ToConsolidationDTO(unit), nil
}

// GetByWave retrieves consolidation units by wave ID
func (s *ConsolidationApplicationService) GetByWave(ctx context.Context, query GetByWaveQuery) ([]ConsolidationDTO, error) {
	units, err := s.repo.FindByWaveID(ctx, query.WaveID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get consolidations by wave", "waveId", query.WaveID)
		return nil, fmt.Errorf("failed to get consolidations by wave: %w", err)
	}

	return ToConsolidationDTOs(units), nil
}

// GetByStation retrieves consolidation units by station
func (s *ConsolidationApplicationService) GetByStation(ctx context.Context, query GetByStationQuery) ([]ConsolidationDTO, error) {
	units, err := s.repo.FindByStation(ctx, query.Station)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get consolidations by station", "station", query.Station)
		return nil, fmt.Errorf("failed to get consolidations by station: %w", err)
	}

	return ToConsolidationDTOs(units), nil
}

// GetPending retrieves pending consolidation units
func (s *ConsolidationApplicationService) GetPending(ctx context.Context, query GetPendingQuery) ([]ConsolidationDTO, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 50 // Default limit
	}

	units, err := s.repo.FindPending(ctx, limit)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get pending consolidations")
		return nil, fmt.Errorf("failed to get pending consolidations: %w", err)
	}

	return ToConsolidationDTOs(units), nil
}
