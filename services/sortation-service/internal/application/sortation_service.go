package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/wms-platform/services/sortation-service/internal/domain"
	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/logging"
)

// SortationService implements the application layer for sortation operations
type SortationService struct {
	batchRepo domain.SortationBatchRepository
	logger    *logging.Logger
}

// NewSortationService creates a new SortationService
func NewSortationService(batchRepo domain.SortationBatchRepository, logger *logging.Logger) *SortationService {
	return &SortationService{
		batchRepo: batchRepo,
		logger:    logger,
	}
}

// CreateBatchCommand represents the command to create a batch
type CreateBatchCommand struct {
	SortationCenter  string
	DestinationGroup string
	CarrierID        string
}

// CreateBatch creates a new sortation batch
func (s *SortationService) CreateBatch(ctx context.Context, cmd CreateBatchCommand) (*domain.SortationBatch, error) {
	batchID := fmt.Sprintf("SRT-%s", uuid.New().String()[:8])

	batch := domain.NewSortationBatch(
		batchID,
		cmd.SortationCenter,
		cmd.DestinationGroup,
		cmd.CarrierID,
	)

	if err := s.batchRepo.Save(ctx, batch); err != nil {
		s.logger.WithError(err).Error("Failed to save sortation batch")
		return nil, errors.ErrInternal("failed to save batch").Wrap(err)
	}

	s.logger.Info("Created sortation batch",
		"batchId", batch.BatchID,
		"destination", batch.DestinationGroup,
		"carrier", batch.CarrierID,
	)

	return batch, nil
}

// AddPackageCommand represents the command to add a package
type AddPackageCommand struct {
	BatchID        string
	PackageID      string
	OrderID        string
	TrackingNumber string
	Destination    string
	CarrierID      string
	Weight         float64
}

// AddPackage adds a package to a batch
func (s *SortationService) AddPackage(ctx context.Context, cmd AddPackageCommand) (*domain.SortationBatch, error) {
	batch, err := s.batchRepo.FindByID(ctx, cmd.BatchID)
	if err != nil {
		return nil, errors.ErrInternal("failed to find batch").Wrap(err)
	}
	if batch == nil {
		return nil, errors.ErrNotFound("batch")
	}

	pkg := domain.SortedPackage{
		PackageID:      cmd.PackageID,
		OrderID:        cmd.OrderID,
		TrackingNumber: cmd.TrackingNumber,
		Destination:    cmd.Destination,
		CarrierID:      cmd.CarrierID,
		Weight:         cmd.Weight,
	}

	if err := batch.AddPackage(pkg); err != nil {
		return nil, errors.ErrValidation("cannot add package").Wrap(err)
	}

	if err := s.batchRepo.Save(ctx, batch); err != nil {
		return nil, errors.ErrInternal("failed to save batch").Wrap(err)
	}

	return batch, nil
}

// SortPackageCommand represents the command to sort a package
type SortPackageCommand struct {
	BatchID   string
	PackageID string
	ChuteID   string
	WorkerID  string
}

// SortPackage sorts a package to a chute
func (s *SortationService) SortPackage(ctx context.Context, cmd SortPackageCommand) (*domain.SortationBatch, error) {
	batch, err := s.batchRepo.FindByID(ctx, cmd.BatchID)
	if err != nil {
		return nil, errors.ErrInternal("failed to find batch").Wrap(err)
	}
	if batch == nil {
		return nil, errors.ErrNotFound("batch")
	}

	if err := batch.SortPackage(cmd.PackageID, cmd.ChuteID, cmd.WorkerID); err != nil {
		return nil, errors.ErrValidation("cannot sort package").Wrap(err)
	}

	// Auto-mark as ready if all packages sorted
	if batch.IsFullySorted() && batch.Status == domain.SortationStatusSorting {
		_ = batch.MarkReady()
	}

	if err := s.batchRepo.Save(ctx, batch); err != nil {
		return nil, errors.ErrInternal("failed to save batch").Wrap(err)
	}

	s.logger.Info("Sorted package",
		"batchId", batch.BatchID,
		"packageId", cmd.PackageID,
		"chuteId", cmd.ChuteID,
		"progress", batch.GetSortingProgress(),
	)

	return batch, nil
}

// MarkReadyCommand represents the command to mark batch ready
type MarkReadyCommand struct {
	BatchID string
}

// MarkReady marks a batch as ready for dispatch
func (s *SortationService) MarkReady(ctx context.Context, cmd MarkReadyCommand) (*domain.SortationBatch, error) {
	batch, err := s.batchRepo.FindByID(ctx, cmd.BatchID)
	if err != nil {
		return nil, errors.ErrInternal("failed to find batch").Wrap(err)
	}
	if batch == nil {
		return nil, errors.ErrNotFound("batch")
	}

	if err := batch.MarkReady(); err != nil {
		return nil, errors.ErrValidation("cannot mark batch ready").Wrap(err)
	}

	if err := s.batchRepo.Save(ctx, batch); err != nil {
		return nil, errors.ErrInternal("failed to save batch").Wrap(err)
	}

	return batch, nil
}

// DispatchBatchCommand represents the command to dispatch a batch
type DispatchBatchCommand struct {
	BatchID      string
	TrailerID    string
	DispatchDock string
}

// DispatchBatch dispatches a batch
func (s *SortationService) DispatchBatch(ctx context.Context, cmd DispatchBatchCommand) (*domain.SortationBatch, error) {
	batch, err := s.batchRepo.FindByID(ctx, cmd.BatchID)
	if err != nil {
		return nil, errors.ErrInternal("failed to find batch").Wrap(err)
	}
	if batch == nil {
		return nil, errors.ErrNotFound("batch")
	}

	// Assign to trailer first
	if err := batch.AssignToTrailer(cmd.TrailerID, cmd.DispatchDock); err != nil {
		return nil, errors.ErrValidation("cannot assign to trailer").Wrap(err)
	}

	// Then dispatch
	if err := batch.Dispatch(); err != nil {
		return nil, errors.ErrValidation("cannot dispatch batch").Wrap(err)
	}

	if err := s.batchRepo.Save(ctx, batch); err != nil {
		return nil, errors.ErrInternal("failed to save batch").Wrap(err)
	}

	s.logger.Info("Dispatched batch",
		"batchId", batch.BatchID,
		"trailerId", cmd.TrailerID,
		"dock", cmd.DispatchDock,
		"packages", batch.TotalPackages,
	)

	return batch, nil
}

// GetBatch retrieves a batch by ID
func (s *SortationService) GetBatch(ctx context.Context, batchID string) (*domain.SortationBatch, error) {
	batch, err := s.batchRepo.FindByID(ctx, batchID)
	if err != nil {
		return nil, errors.ErrInternal("failed to find batch").Wrap(err)
	}
	if batch == nil {
		return nil, errors.ErrNotFound("batch")
	}
	return batch, nil
}

// GetBatchesByStatus retrieves batches by status
func (s *SortationService) GetBatchesByStatus(ctx context.Context, status domain.SortationStatus) ([]*domain.SortationBatch, error) {
	return s.batchRepo.FindByStatus(ctx, status)
}

// GetReadyBatches retrieves batches ready for dispatch
func (s *SortationService) GetReadyBatches(ctx context.Context, carrierID string, limit int) ([]*domain.SortationBatch, error) {
	return s.batchRepo.FindReadyForDispatch(ctx, carrierID, limit)
}

// ListBatches retrieves all batches
func (s *SortationService) ListBatches(ctx context.Context, limit int) ([]*domain.SortationBatch, error) {
	return s.batchRepo.FindAll(ctx, limit)
}
