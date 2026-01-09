package application

import (
	"context"
	"fmt"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"

	"github.com/wms-platform/facility-service/internal/domain"
)

// StationRepository interface for station persistence
type StationRepository interface {
	Save(ctx context.Context, station *domain.Station) error
	FindByID(ctx context.Context, stationID string) (*domain.Station, error)
	FindByZone(ctx context.Context, zone string) ([]*domain.Station, error)
	FindByType(ctx context.Context, stationType domain.StationType) ([]*domain.Station, error)
	FindByStatus(ctx context.Context, status domain.StationStatus) ([]*domain.Station, error)
	FindCapableStations(ctx context.Context, requirements []domain.StationCapability, stationType domain.StationType, zone string) ([]*domain.Station, error)
	FindByCapability(ctx context.Context, capability domain.StationCapability) ([]*domain.Station, error)
	FindByWorkerID(ctx context.Context, workerID string) (*domain.Station, error)
	FindAll(ctx context.Context, limit, offset int) ([]*domain.Station, error)
	Delete(ctx context.Context, stationID string) error
}

// StationApplicationService handles station-related use cases
type StationApplicationService struct {
	repo         StationRepository
	producer     *kafka.InstrumentedProducer
	eventFactory *cloudevents.EventFactory
	logger       *logging.Logger
}

// NewStationApplicationService creates a new StationApplicationService
func NewStationApplicationService(
	repo StationRepository,
	producer *kafka.InstrumentedProducer,
	eventFactory *cloudevents.EventFactory,
	logger *logging.Logger,
) *StationApplicationService {
	return &StationApplicationService{
		repo:         repo,
		producer:     producer,
		eventFactory: eventFactory,
		logger:       logger,
	}
}

// CreateStation creates a new station
func (s *StationApplicationService) CreateStation(ctx context.Context, cmd CreateStationCommand) (*StationDTO, error) {
	station, err := domain.NewStation(cmd.StationID, cmd.Name, cmd.Zone, domain.StationType(cmd.StationType), cmd.MaxConcurrentTasks)
	if err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	// Add initial capabilities
	for _, capStr := range cmd.Capabilities {
		cap := domain.StationCapability(capStr)
		if err := station.AddCapability(cap); err != nil {
			return nil, errors.ErrValidation(fmt.Sprintf("invalid capability: %s", capStr))
		}
	}

	if err := s.repo.Save(ctx, station); err != nil {
		s.logger.WithError(err).Error("Failed to save station", "stationId", station.StationID)
		return nil, fmt.Errorf("failed to save station: %w", err)
	}

	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "station.created",
		EntityType: "station",
		EntityID:   station.StationID,
		Action:     "created",
		RelatedIDs: map[string]string{
			"zone":        cmd.Zone,
			"stationType": cmd.StationType,
		},
	})

	return ToStationDTO(station), nil
}

// GetStation retrieves a station by ID
func (s *StationApplicationService) GetStation(ctx context.Context, query GetStationQuery) (*StationDTO, error) {
	station, err := s.repo.FindByID(ctx, query.StationID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get station", "stationId", query.StationID)
		return nil, fmt.Errorf("failed to get station: %w", err)
	}

	if station == nil {
		return nil, errors.ErrNotFound("station")
	}

	return ToStationDTO(station), nil
}

// UpdateStation updates a station
func (s *StationApplicationService) UpdateStation(ctx context.Context, cmd UpdateStationCommand) (*StationDTO, error) {
	station, err := s.repo.FindByID(ctx, cmd.StationID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get station", "stationId", cmd.StationID)
		return nil, fmt.Errorf("failed to get station: %w", err)
	}

	if station == nil {
		return nil, errors.ErrNotFound("station")
	}

	if cmd.Name != "" {
		station.Name = cmd.Name
	}
	if cmd.Zone != "" {
		station.Zone = cmd.Zone
	}
	if cmd.MaxConcurrentTasks > 0 {
		station.MaxConcurrentTasks = cmd.MaxConcurrentTasks
	}

	if err := s.repo.Save(ctx, station); err != nil {
		s.logger.WithError(err).Error("Failed to save station", "stationId", cmd.StationID)
		return nil, fmt.Errorf("failed to save station: %w", err)
	}

	return ToStationDTO(station), nil
}

// AddCapability adds a capability to a station
func (s *StationApplicationService) AddCapability(ctx context.Context, cmd AddCapabilityCommand) (*StationDTO, error) {
	station, err := s.repo.FindByID(ctx, cmd.StationID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get station", "stationId", cmd.StationID)
		return nil, fmt.Errorf("failed to get station: %w", err)
	}

	if station == nil {
		return nil, errors.ErrNotFound("station")
	}

	cap := domain.StationCapability(cmd.Capability)
	if err := station.AddCapability(cap); err != nil {
		if err == domain.ErrCapabilityExists {
			return nil, errors.ErrConflict("capability already exists")
		}
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, station); err != nil {
		s.logger.WithError(err).Error("Failed to save station", "stationId", cmd.StationID)
		return nil, fmt.Errorf("failed to save station: %w", err)
	}

	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "station.capability.added",
		EntityType: "station",
		EntityID:   cmd.StationID,
		Action:     "capability_added",
		RelatedIDs: map[string]string{
			"capability": cmd.Capability,
		},
	})

	return ToStationDTO(station), nil
}

// RemoveCapability removes a capability from a station
func (s *StationApplicationService) RemoveCapability(ctx context.Context, cmd RemoveCapabilityCommand) (*StationDTO, error) {
	station, err := s.repo.FindByID(ctx, cmd.StationID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get station", "stationId", cmd.StationID)
		return nil, fmt.Errorf("failed to get station: %w", err)
	}

	if station == nil {
		return nil, errors.ErrNotFound("station")
	}

	cap := domain.StationCapability(cmd.Capability)
	if err := station.RemoveCapability(cap); err != nil {
		if err == domain.ErrCapabilityNotFound {
			return nil, errors.ErrNotFound("capability")
		}
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, station); err != nil {
		s.logger.WithError(err).Error("Failed to save station", "stationId", cmd.StationID)
		return nil, fmt.Errorf("failed to save station: %w", err)
	}

	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "station.capability.removed",
		EntityType: "station",
		EntityID:   cmd.StationID,
		Action:     "capability_removed",
		RelatedIDs: map[string]string{
			"capability": cmd.Capability,
		},
	})

	return ToStationDTO(station), nil
}

// SetCapabilities sets all capabilities for a station
func (s *StationApplicationService) SetCapabilities(ctx context.Context, cmd SetCapabilitiesCommand) (*StationDTO, error) {
	station, err := s.repo.FindByID(ctx, cmd.StationID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get station", "stationId", cmd.StationID)
		return nil, fmt.Errorf("failed to get station: %w", err)
	}

	if station == nil {
		return nil, errors.ErrNotFound("station")
	}

	capabilities := make([]domain.StationCapability, len(cmd.Capabilities))
	for i, capStr := range cmd.Capabilities {
		capabilities[i] = domain.StationCapability(capStr)
	}

	if err := station.SetCapabilities(capabilities); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, station); err != nil {
		s.logger.WithError(err).Error("Failed to save station", "stationId", cmd.StationID)
		return nil, fmt.Errorf("failed to save station: %w", err)
	}

	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "station.capabilities.updated",
		EntityType: "station",
		EntityID:   cmd.StationID,
		Action:     "capabilities_updated",
	})

	return ToStationDTO(station), nil
}

// SetStatus updates the station status
func (s *StationApplicationService) SetStatus(ctx context.Context, cmd SetStationStatusCommand) (*StationDTO, error) {
	station, err := s.repo.FindByID(ctx, cmd.StationID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get station", "stationId", cmd.StationID)
		return nil, fmt.Errorf("failed to get station: %w", err)
	}

	if station == nil {
		return nil, errors.ErrNotFound("station")
	}

	if err := station.SetStatus(domain.StationStatus(cmd.Status)); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, station); err != nil {
		s.logger.WithError(err).Error("Failed to save station", "stationId", cmd.StationID)
		return nil, fmt.Errorf("failed to save station: %w", err)
	}

	return ToStationDTO(station), nil
}

// FindCapableStations finds stations that have all required capabilities
func (s *StationApplicationService) FindCapableStations(ctx context.Context, query FindCapableStationsQuery) ([]StationDTO, error) {
	requirements := make([]domain.StationCapability, len(query.Requirements))
	for i, req := range query.Requirements {
		requirements[i] = domain.StationCapability(req)
	}

	stations, err := s.repo.FindCapableStations(ctx, requirements, domain.StationType(query.StationType), query.Zone)
	if err != nil {
		s.logger.WithError(err).Error("Failed to find capable stations", "requirements", query.Requirements)
		return nil, fmt.Errorf("failed to find capable stations: %w", err)
	}

	return ToStationDTOs(stations), nil
}

// ListStations retrieves all stations
func (s *StationApplicationService) ListStations(ctx context.Context, query ListStationsQuery) ([]StationDTO, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}

	stations, err := s.repo.FindAll(ctx, limit, query.Offset)
	if err != nil {
		s.logger.WithError(err).Error("Failed to list stations")
		return nil, fmt.Errorf("failed to list stations: %w", err)
	}

	return ToStationDTOs(stations), nil
}

// GetByZone retrieves stations by zone
func (s *StationApplicationService) GetByZone(ctx context.Context, query GetStationsByZoneQuery) ([]StationDTO, error) {
	stations, err := s.repo.FindByZone(ctx, query.Zone)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get stations by zone", "zone", query.Zone)
		return nil, fmt.Errorf("failed to get stations by zone: %w", err)
	}

	return ToStationDTOs(stations), nil
}

// GetByType retrieves stations by type
func (s *StationApplicationService) GetByType(ctx context.Context, query GetStationsByTypeQuery) ([]StationDTO, error) {
	stations, err := s.repo.FindByType(ctx, domain.StationType(query.StationType))
	if err != nil {
		s.logger.WithError(err).Error("Failed to get stations by type", "type", query.StationType)
		return nil, fmt.Errorf("failed to get stations by type: %w", err)
	}

	return ToStationDTOs(stations), nil
}

// GetByStatus retrieves stations by status
func (s *StationApplicationService) GetByStatus(ctx context.Context, query GetStationsByStatusQuery) ([]StationDTO, error) {
	stations, err := s.repo.FindByStatus(ctx, domain.StationStatus(query.Status))
	if err != nil {
		s.logger.WithError(err).Error("Failed to get stations by status", "status", query.Status)
		return nil, fmt.Errorf("failed to get stations by status: %w", err)
	}

	return ToStationDTOs(stations), nil
}

// DeleteStation deletes a station
func (s *StationApplicationService) DeleteStation(ctx context.Context, cmd DeleteStationCommand) error {
	station, err := s.repo.FindByID(ctx, cmd.StationID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get station", "stationId", cmd.StationID)
		return fmt.Errorf("failed to get station: %w", err)
	}

	if station == nil {
		return errors.ErrNotFound("station")
	}

	if err := s.repo.Delete(ctx, cmd.StationID); err != nil {
		s.logger.WithError(err).Error("Failed to delete station", "stationId", cmd.StationID)
		return fmt.Errorf("failed to delete station: %w", err)
	}

	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "station.deleted",
		EntityType: "station",
		EntityID:   cmd.StationID,
		Action:     "deleted",
	})

	return nil
}
