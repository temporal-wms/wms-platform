package domain

import (
	"context"
)

// SortationBatchRepository defines the interface for sortation batch persistence
type SortationBatchRepository interface {
	// Save persists a sortation batch (upsert)
	Save(ctx context.Context, batch *SortationBatch) error

	// FindByID retrieves a batch by its BatchID
	FindByID(ctx context.Context, batchID string) (*SortationBatch, error)

	// FindByStatus retrieves batches by status
	FindByStatus(ctx context.Context, status SortationStatus) ([]*SortationBatch, error)

	// FindByCarrier retrieves batches for a carrier
	FindByCarrier(ctx context.Context, carrierID string) ([]*SortationBatch, error)

	// FindByDestination retrieves batches for a destination
	FindByDestination(ctx context.Context, destinationGroup string) ([]*SortationBatch, error)

	// FindByCenter retrieves batches for a sortation center
	FindByCenter(ctx context.Context, centerID string) ([]*SortationBatch, error)

	// FindReadyForDispatch retrieves batches ready for dispatch
	FindReadyForDispatch(ctx context.Context, carrierID string, limit int) ([]*SortationBatch, error)

	// FindAll retrieves all batches with limit
	FindAll(ctx context.Context, limit int) ([]*SortationBatch, error)

	// Delete deletes a batch
	Delete(ctx context.Context, batchID string) error
}

// ChuteRepository defines the interface for chute persistence
type ChuteRepository interface {
	// FindByID retrieves a chute by ID
	FindByID(ctx context.Context, chuteID string) (*Chute, error)

	// FindByDestination retrieves chutes handling a destination
	FindByDestination(ctx context.Context, destination string) ([]Chute, error)

	// FindAvailable retrieves available chutes
	FindAvailable(ctx context.Context) ([]Chute, error)

	// UpdateCount updates the package count for a chute
	UpdateCount(ctx context.Context, chuteID string, countChange int) error

	// Save saves or updates a chute
	Save(ctx context.Context, chute *Chute) error
}
