package application

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"

	"github.com/wms-platform/shipping-service/internal/domain"
)

// ShippingApplicationService handles shipping-related use cases
type ShippingApplicationService struct {
	repo         domain.ShipmentRepository
	producer     *kafka.InstrumentedProducer
	eventFactory *cloudevents.EventFactory
	logger       *logging.Logger
}

// NewShippingApplicationService creates a new ShippingApplicationService
func NewShippingApplicationService(
	repo domain.ShipmentRepository,
	producer *kafka.InstrumentedProducer,
	eventFactory *cloudevents.EventFactory,
	logger *logging.Logger,
) *ShippingApplicationService {
	return &ShippingApplicationService{
		repo:         repo,
		producer:     producer,
		eventFactory: eventFactory,
		logger:       logger,
	}
}

// CreateShipment creates a new shipment
func (s *ShippingApplicationService) CreateShipment(ctx context.Context, cmd CreateShipmentCommand) (*ShipmentDTO, error) {
	shipment := domain.NewShipment(
		cmd.ShipmentID,
		cmd.OrderID,
		cmd.PackageID,
		cmd.WaveID,
		cmd.Carrier,
		cmd.Package,
		cmd.Recipient,
		cmd.Shipper,
	)

	if err := s.repo.Save(ctx, shipment); err != nil {
		s.logger.WithError(err).Error("Failed to create shipment", "shipmentId", cmd.ShipmentID)
		return nil, fmt.Errorf("failed to create shipment: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Created shipment", "shipmentId", cmd.ShipmentID, "orderId", cmd.OrderID)
	return ToShipmentDTO(shipment), nil
}

// GetShipment retrieves a shipment by ID
func (s *ShippingApplicationService) GetShipment(ctx context.Context, query GetShipmentQuery) (*ShipmentDTO, error) {
	shipment, err := s.repo.FindByID(ctx, query.ShipmentID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get shipment", "shipmentId", query.ShipmentID)
		return nil, fmt.Errorf("failed to get shipment: %w", err)
	}

	if shipment == nil {
		return nil, errors.ErrNotFound("shipment")
	}

	return ToShipmentDTO(shipment), nil
}

// GenerateLabel generates a shipping label for a shipment
func (s *ShippingApplicationService) GenerateLabel(ctx context.Context, cmd GenerateLabelCommand) (*ShipmentDTO, error) {
	shipment, err := s.repo.FindByID(ctx, cmd.ShipmentID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get shipment", "shipmentId", cmd.ShipmentID)
		return nil, fmt.Errorf("failed to get shipment: %w", err)
	}

	if shipment == nil {
		return nil, errors.ErrNotFound("shipment")
	}

	// Set generated time
	label := cmd.Label
	label.GeneratedAt = time.Now()

	if err := shipment.GenerateLabel(label); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, shipment); err != nil {
		s.logger.WithError(err).Error("Failed to save shipment", "shipmentId", cmd.ShipmentID)
		return nil, fmt.Errorf("failed to save shipment: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Generated label for shipment", "shipmentId", cmd.ShipmentID, "trackingNumber", label.TrackingNumber)
	return ToShipmentDTO(shipment), nil
}

// AddToManifest adds a shipment to a manifest
func (s *ShippingApplicationService) AddToManifest(ctx context.Context, cmd AddToManifestCommand) (*ShipmentDTO, error) {
	shipment, err := s.repo.FindByID(ctx, cmd.ShipmentID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get shipment", "shipmentId", cmd.ShipmentID)
		return nil, fmt.Errorf("failed to get shipment: %w", err)
	}

	if shipment == nil {
		return nil, errors.ErrNotFound("shipment")
	}

	// Set generated time
	manifest := cmd.Manifest
	manifest.GeneratedAt = time.Now()

	if err := shipment.AddToManifest(manifest); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, shipment); err != nil {
		s.logger.WithError(err).Error("Failed to save shipment", "shipmentId", cmd.ShipmentID)
		return nil, fmt.Errorf("failed to save shipment: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Added shipment to manifest", "shipmentId", cmd.ShipmentID, "manifestId", manifest.ManifestID)
	return ToShipmentDTO(shipment), nil
}

// ConfirmShipment confirms a shipment has been shipped
func (s *ShippingApplicationService) ConfirmShipment(ctx context.Context, cmd ConfirmShipmentCommand) (*ShipmentDTO, error) {
	shipment, err := s.repo.FindByID(ctx, cmd.ShipmentID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get shipment", "shipmentId", cmd.ShipmentID)
		return nil, fmt.Errorf("failed to get shipment: %w", err)
	}

	if shipment == nil {
		return nil, errors.ErrNotFound("shipment")
	}

	if err := shipment.ConfirmShipment(cmd.EstimatedDelivery); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.repo.Save(ctx, shipment); err != nil {
		s.logger.WithError(err).Error("Failed to save shipment", "shipmentId", cmd.ShipmentID)
		return nil, fmt.Errorf("failed to save shipment: %w", err)
	}

	// Events are saved to outbox by repository in transaction

	s.logger.Info("Confirmed shipment", "shipmentId", cmd.ShipmentID)
	return ToShipmentDTO(shipment), nil
}

// GetByOrder retrieves a shipment by order ID
func (s *ShippingApplicationService) GetByOrder(ctx context.Context, query GetByOrderQuery) (*ShipmentDTO, error) {
	shipment, err := s.repo.FindByOrderID(ctx, query.OrderID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get shipment by order", "orderId", query.OrderID)
		return nil, fmt.Errorf("failed to get shipment by order: %w", err)
	}

	if shipment == nil {
		return nil, errors.ErrNotFound("shipment")
	}

	return ToShipmentDTO(shipment), nil
}

// GetByTracking retrieves a shipment by tracking number
func (s *ShippingApplicationService) GetByTracking(ctx context.Context, query GetByTrackingQuery) (*ShipmentDTO, error) {
	shipment, err := s.repo.FindByTrackingNumber(ctx, query.TrackingNumber)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get shipment by tracking", "trackingNumber", query.TrackingNumber)
		return nil, fmt.Errorf("failed to get shipment by tracking: %w", err)
	}

	if shipment == nil {
		return nil, errors.ErrNotFound("shipment")
	}

	return ToShipmentDTO(shipment), nil
}

// GetByStatus retrieves shipments by status
func (s *ShippingApplicationService) GetByStatus(ctx context.Context, query GetByStatusQuery) ([]ShipmentDTO, error) {
	status := domain.ShipmentStatus(query.Status)
	shipments, err := s.repo.FindByStatus(ctx, status)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get shipments by status", "status", status)
		return nil, fmt.Errorf("failed to get shipments by status: %w", err)
	}

	return ToShipmentDTOs(shipments), nil
}

// GetByCarrier retrieves shipments by carrier
func (s *ShippingApplicationService) GetByCarrier(ctx context.Context, query GetByCarrierQuery) ([]ShipmentDTO, error) {
	shipments, err := s.repo.FindByCarrier(ctx, query.CarrierCode)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get shipments by carrier", "carrierCode", query.CarrierCode)
		return nil, fmt.Errorf("failed to get shipments by carrier: %w", err)
	}

	return ToShipmentDTOs(shipments), nil
}

// GetPendingForManifest retrieves pending shipments for a carrier manifest
func (s *ShippingApplicationService) GetPendingForManifest(ctx context.Context, query GetPendingForManifestQuery) ([]ShipmentDTO, error) {
	shipments, err := s.repo.FindPendingForManifest(ctx, query.CarrierCode)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get pending shipments for manifest", "carrierCode", query.CarrierCode)
		return nil, fmt.Errorf("failed to get pending shipments for manifest: %w", err)
	}

	return ToShipmentDTOs(shipments), nil
}
