package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"

	"github.com/wms-platform/shipping-service/internal/domain"
)

// ManifestRepository defines the interface for manifest persistence
type ManifestRepository interface {
	Save(ctx context.Context, manifest *domain.OutboundManifest) error
	FindByID(ctx context.Context, manifestID string) (*domain.OutboundManifest, error)
	FindByCarrierID(ctx context.Context, carrierID string) ([]*domain.OutboundManifest, error)
	FindByStatus(ctx context.Context, status domain.ManifestStatus) ([]*domain.OutboundManifest, error)
	FindOpenByCarrier(ctx context.Context, carrierID string) (*domain.OutboundManifest, error)
	FindClosedByCarrier(ctx context.Context, carrierID string) ([]*domain.OutboundManifest, error)
	FindByDateRange(ctx context.Context, start, end time.Time) ([]*domain.OutboundManifest, error)
	FindDispatchedToday(ctx context.Context) ([]*domain.OutboundManifest, error)
	CountByStatus(ctx context.Context, status domain.ManifestStatus) (int64, error)
	Delete(ctx context.Context, manifestID string) error
}

// ManifestApplicationService handles manifest-related use cases
type ManifestApplicationService struct {
	repo         ManifestRepository
	producer     *kafka.InstrumentedProducer
	eventFactory *cloudevents.EventFactory
	logger       *logging.Logger
}

// NewManifestApplicationService creates a new ManifestApplicationService
func NewManifestApplicationService(
	repo ManifestRepository,
	producer *kafka.InstrumentedProducer,
	eventFactory *cloudevents.EventFactory,
	logger *logging.Logger,
) *ManifestApplicationService {
	return &ManifestApplicationService{
		repo:         repo,
		producer:     producer,
		eventFactory: eventFactory,
		logger:       logger,
	}
}

// CreateManifestCommand represents the command to create a manifest
type CreateManifestCommand struct {
	CarrierID       string     `json:"carrierId"`
	CarrierName     string     `json:"carrierName"`
	ServiceType     string     `json:"serviceType"`
	ScheduledPickup *time.Time `json:"scheduledPickup,omitempty"`
}

// AddPackageCommand represents the command to add a package to a manifest
type AddPackageCommand struct {
	ManifestID     string  `json:"manifestId"`
	PackageID      string  `json:"packageId"`
	ShipmentID     string  `json:"shipmentId"`
	OrderID        string  `json:"orderId"`
	TrackingNumber string  `json:"trackingNumber"`
	Weight         float64 `json:"weight"`
}

// CloseManifestCommand represents the command to close a manifest
type CloseManifestCommand struct {
	ManifestID string `json:"manifestId"`
}

// AssignTrailerCommand represents the command to assign a trailer to a manifest
type AssignTrailerCommand struct {
	ManifestID   string `json:"manifestId"`
	TrailerID    string `json:"trailerId"`
	DispatchDock string `json:"dispatchDock"`
}

// DispatchManifestCommand represents the command to dispatch a manifest
type DispatchManifestCommand struct {
	ManifestID string `json:"manifestId"`
}

// GetManifestQuery represents a query to get a manifest
type GetManifestQuery struct {
	ManifestID string `json:"manifestId"`
}

// GetManifestsByCarrierQuery represents a query to get manifests by carrier
type GetManifestsByCarrierQuery struct {
	CarrierID string `json:"carrierId"`
}

// GetManifestsByStatusQuery represents a query to get manifests by status
type GetManifestsByStatusQuery struct {
	Status string `json:"status"`
}

// ManifestDTO represents a manifest response
type ManifestDTO struct {
	ManifestID      string            `json:"manifestId"`
	CarrierID       string            `json:"carrierId"`
	CarrierName     string            `json:"carrierName"`
	ServiceType     string            `json:"serviceType"`
	TrailerID       string            `json:"trailerId,omitempty"`
	DispatchDock    string            `json:"dispatchDock,omitempty"`
	Packages        []ManifestPkgDTO  `json:"packages"`
	TotalPackages   int               `json:"totalPackages"`
	TotalWeight     float64           `json:"totalWeight"`
	Status          string            `json:"status"`
	ScheduledPickup *time.Time        `json:"scheduledPickup,omitempty"`
	ClosedAt        *time.Time        `json:"closedAt,omitempty"`
	DispatchedAt    *time.Time        `json:"dispatchedAt,omitempty"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
}

// ManifestPkgDTO represents a package in a manifest response
type ManifestPkgDTO struct {
	PackageID      string    `json:"packageId"`
	ShipmentID     string    `json:"shipmentId"`
	OrderID        string    `json:"orderId"`
	TrackingNumber string    `json:"trackingNumber"`
	Weight         float64   `json:"weight"`
	AddedAt        time.Time `json:"addedAt"`
}

// CreateManifest creates a new outbound manifest
func (s *ManifestApplicationService) CreateManifest(ctx context.Context, cmd CreateManifestCommand) (*ManifestDTO, error) {
	manifestID := fmt.Sprintf("MAN-%s-%s", cmd.CarrierID, uuid.New().String()[:8])

	manifest := domain.NewOutboundManifest(
		manifestID,
		cmd.CarrierID,
		cmd.CarrierName,
		cmd.ServiceType,
	)

	if cmd.ScheduledPickup != nil {
		manifest.ScheduledPickup = cmd.ScheduledPickup
	}

	if err := s.repo.Save(ctx, manifest); err != nil {
		s.logger.WithError(err).Error("Failed to create manifest", "manifestId", manifestID)
		return nil, fmt.Errorf("failed to create manifest: %w", err)
	}

	s.logger.Info("Created manifest", "manifestId", manifestID, "carrierId", cmd.CarrierID)
	return toManifestDTO(manifest), nil
}

// GetManifest retrieves a manifest by ID
func (s *ManifestApplicationService) GetManifest(ctx context.Context, query GetManifestQuery) (*ManifestDTO, error) {
	manifest, err := s.repo.FindByID(ctx, query.ManifestID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get manifest", "manifestId", query.ManifestID)
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	if manifest == nil {
		return nil, errors.ErrNotFound("manifest")
	}

	return toManifestDTO(manifest), nil
}

// GetOrCreateOpenManifest gets an open manifest for carrier or creates one
func (s *ManifestApplicationService) GetOrCreateOpenManifest(ctx context.Context, carrierID, carrierName, serviceType string) (*ManifestDTO, error) {
	manifest, err := s.repo.FindOpenByCarrier(ctx, carrierID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to find open manifest", "carrierId", carrierID)
		return nil, fmt.Errorf("failed to find open manifest: %w", err)
	}

	if manifest != nil {
		return toManifestDTO(manifest), nil
	}

	// Create new manifest
	return s.CreateManifest(ctx, CreateManifestCommand{
		CarrierID:   carrierID,
		CarrierName: carrierName,
		ServiceType: serviceType,
	})
}

// AddPackage adds a package to a manifest
func (s *ManifestApplicationService) AddPackage(ctx context.Context, cmd AddPackageCommand) (*ManifestDTO, error) {
	manifest, err := s.repo.FindByID(ctx, cmd.ManifestID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get manifest", "manifestId", cmd.ManifestID)
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	if manifest == nil {
		return nil, errors.ErrNotFound("manifest")
	}

	pkg := domain.ManifestPackage{
		PackageID:      cmd.PackageID,
		ShipmentID:     cmd.ShipmentID,
		OrderID:        cmd.OrderID,
		TrackingNumber: cmd.TrackingNumber,
		Weight:         cmd.Weight,
	}

	if err := manifest.AddPackage(pkg); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, manifest); err != nil {
		s.logger.WithError(err).Error("Failed to save manifest", "manifestId", cmd.ManifestID)
		return nil, fmt.Errorf("failed to save manifest: %w", err)
	}

	s.logger.Info("Added package to manifest",
		"manifestId", cmd.ManifestID,
		"packageId", cmd.PackageID,
		"totalPackages", manifest.TotalPackages,
	)
	return toManifestDTO(manifest), nil
}

// CloseManifest closes a manifest
func (s *ManifestApplicationService) CloseManifest(ctx context.Context, cmd CloseManifestCommand) (*ManifestDTO, error) {
	manifest, err := s.repo.FindByID(ctx, cmd.ManifestID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get manifest", "manifestId", cmd.ManifestID)
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	if manifest == nil {
		return nil, errors.ErrNotFound("manifest")
	}

	if err := manifest.Close(); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, manifest); err != nil {
		s.logger.WithError(err).Error("Failed to save manifest", "manifestId", cmd.ManifestID)
		return nil, fmt.Errorf("failed to save manifest: %w", err)
	}

	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "manifest.closed",
		EntityType: "manifest",
		EntityID:   cmd.ManifestID,
		Action:     "closed",
		RelatedIDs: map[string]string{
			"carrierId":    manifest.CarrierID,
			"packageCount": fmt.Sprintf("%d", manifest.TotalPackages),
		},
	})

	return toManifestDTO(manifest), nil
}

// AssignTrailer assigns a trailer to a manifest
func (s *ManifestApplicationService) AssignTrailer(ctx context.Context, cmd AssignTrailerCommand) (*ManifestDTO, error) {
	manifest, err := s.repo.FindByID(ctx, cmd.ManifestID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get manifest", "manifestId", cmd.ManifestID)
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	if manifest == nil {
		return nil, errors.ErrNotFound("manifest")
	}

	if err := manifest.AssignTrailer(cmd.TrailerID, cmd.DispatchDock); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, manifest); err != nil {
		s.logger.WithError(err).Error("Failed to save manifest", "manifestId", cmd.ManifestID)
		return nil, fmt.Errorf("failed to save manifest: %w", err)
	}

	s.logger.Info("Assigned trailer to manifest",
		"manifestId", cmd.ManifestID,
		"trailerId", cmd.TrailerID,
		"dispatchDock", cmd.DispatchDock,
	)
	return toManifestDTO(manifest), nil
}

// DispatchManifest dispatches a manifest
func (s *ManifestApplicationService) DispatchManifest(ctx context.Context, cmd DispatchManifestCommand) (*ManifestDTO, error) {
	manifest, err := s.repo.FindByID(ctx, cmd.ManifestID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get manifest", "manifestId", cmd.ManifestID)
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	if manifest == nil {
		return nil, errors.ErrNotFound("manifest")
	}

	if err := manifest.Dispatch(); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, manifest); err != nil {
		s.logger.WithError(err).Error("Failed to save manifest", "manifestId", cmd.ManifestID)
		return nil, fmt.Errorf("failed to save manifest: %w", err)
	}

	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "manifest.dispatched",
		EntityType: "manifest",
		EntityID:   cmd.ManifestID,
		Action:     "dispatched",
		RelatedIDs: map[string]string{
			"carrierId":    manifest.CarrierID,
			"trailerId":    manifest.TrailerID,
			"dispatchDock": manifest.DispatchDock,
		},
	})

	return toManifestDTO(manifest), nil
}

// GetByCarrier retrieves manifests by carrier
func (s *ManifestApplicationService) GetByCarrier(ctx context.Context, query GetManifestsByCarrierQuery) ([]ManifestDTO, error) {
	manifests, err := s.repo.FindByCarrierID(ctx, query.CarrierID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get manifests by carrier", "carrierId", query.CarrierID)
		return nil, fmt.Errorf("failed to get manifests by carrier: %w", err)
	}

	return toManifestDTOs(manifests), nil
}

// GetByStatus retrieves manifests by status
func (s *ManifestApplicationService) GetByStatus(ctx context.Context, query GetManifestsByStatusQuery) ([]ManifestDTO, error) {
	status := domain.ManifestStatus(query.Status)
	manifests, err := s.repo.FindByStatus(ctx, status)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get manifests by status", "status", query.Status)
		return nil, fmt.Errorf("failed to get manifests by status: %w", err)
	}

	return toManifestDTOs(manifests), nil
}

// GetClosedByCarrier retrieves closed manifests ready for dispatch
func (s *ManifestApplicationService) GetClosedByCarrier(ctx context.Context, carrierID string) ([]ManifestDTO, error) {
	manifests, err := s.repo.FindClosedByCarrier(ctx, carrierID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get closed manifests", "carrierId", carrierID)
		return nil, fmt.Errorf("failed to get closed manifests: %w", err)
	}

	return toManifestDTOs(manifests), nil
}

// GetDispatchedToday retrieves all manifests dispatched today
func (s *ManifestApplicationService) GetDispatchedToday(ctx context.Context) ([]ManifestDTO, error) {
	manifests, err := s.repo.FindDispatchedToday(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get dispatched manifests")
		return nil, fmt.Errorf("failed to get dispatched manifests: %w", err)
	}

	return toManifestDTOs(manifests), nil
}

// Helper functions
func toManifestDTO(m *domain.OutboundManifest) *ManifestDTO {
	packages := make([]ManifestPkgDTO, len(m.Packages))
	for i, pkg := range m.Packages {
		packages[i] = ManifestPkgDTO{
			PackageID:      pkg.PackageID,
			ShipmentID:     pkg.ShipmentID,
			OrderID:        pkg.OrderID,
			TrackingNumber: pkg.TrackingNumber,
			Weight:         pkg.Weight,
			AddedAt:        pkg.AddedAt,
		}
	}

	return &ManifestDTO{
		ManifestID:      m.ManifestID,
		CarrierID:       m.CarrierID,
		CarrierName:     m.CarrierName,
		ServiceType:     m.ServiceType,
		TrailerID:       m.TrailerID,
		DispatchDock:    m.DispatchDock,
		Packages:        packages,
		TotalPackages:   m.TotalPackages,
		TotalWeight:     m.TotalWeight,
		Status:          string(m.Status),
		ScheduledPickup: m.ScheduledPickup,
		ClosedAt:        m.ClosedAt,
		DispatchedAt:    m.DispatchedAt,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

func toManifestDTOs(manifests []*domain.OutboundManifest) []ManifestDTO {
	dtos := make([]ManifestDTO, len(manifests))
	for i, m := range manifests {
		dtos[i] = *toManifestDTO(m)
	}
	return dtos
}
