package domain

import (
	"context"
)

// PutawayTaskRepository defines the interface for putaway task persistence
type PutawayTaskRepository interface {
	// Save persists a putaway task (upsert)
	Save(ctx context.Context, task *PutawayTask) error

	// FindByID retrieves a task by its TaskID
	FindByID(ctx context.Context, taskID string) (*PutawayTask, error)

	// FindByStatus retrieves tasks by status
	FindByStatus(ctx context.Context, status PutawayStatus, pagination Pagination) ([]*PutawayTask, error)

	// FindByWorkerID retrieves tasks assigned to a worker
	FindByWorkerID(ctx context.Context, workerID string, pagination Pagination) ([]*PutawayTask, error)

	// FindPendingTasks retrieves pending tasks for assignment
	FindPendingTasks(ctx context.Context, limit int) ([]*PutawayTask, error)

	// FindByShipmentID retrieves tasks for a shipment
	FindByShipmentID(ctx context.Context, shipmentID string) ([]*PutawayTask, error)

	// FindBySKU retrieves tasks for a specific SKU
	FindBySKU(ctx context.Context, sku string, pagination Pagination) ([]*PutawayTask, error)

	// UpdateStatus updates the task status
	UpdateStatus(ctx context.Context, taskID string, status PutawayStatus) error

	// Delete deletes a task
	Delete(ctx context.Context, taskID string) error

	// Count returns the total number of tasks matching the filter
	Count(ctx context.Context, filter TaskFilter) (int64, error)
}

// StorageLocationRepository defines the interface for storage location persistence
type StorageLocationRepository interface {
	// FindAvailableLocations finds locations with available capacity
	FindAvailableLocations(ctx context.Context, constraints LocationConstraints, limit int) ([]StorageLocation, error)

	// FindByID retrieves a location by ID
	FindByID(ctx context.Context, locationID string) (*StorageLocation, error)

	// FindByZone retrieves locations in a zone
	FindByZone(ctx context.Context, zone string) ([]StorageLocation, error)

	// UpdateCapacity updates the location capacity
	UpdateCapacity(ctx context.Context, locationID string, quantityChange int, weightChange float64) error
}

// LocationConstraints represents constraints for finding locations
type LocationConstraints struct {
	MinCapacity     int
	MinWeight       float64
	RequiresHazmat  bool
	RequiresColdChain bool
	RequiresOversized bool
	PreferredZone   string
}

// Pagination represents pagination options
type Pagination struct {
	Page     int64
	PageSize int64
}

// DefaultPagination returns default pagination options
func DefaultPagination() Pagination {
	return Pagination{
		Page:     1,
		PageSize: 20,
	}
}

// Skip returns the number of documents to skip
func (p Pagination) Skip() int64 {
	return (p.Page - 1) * p.PageSize
}

// Limit returns the maximum number of documents to return
func (p Pagination) Limit() int64 {
	return p.PageSize
}

// TaskFilter represents filter options for querying tasks
type TaskFilter struct {
	Status     *PutawayStatus
	WorkerID   *string
	ShipmentID *string
	SKU        *string
	Strategy   *StorageStrategy
	FromDate   *string
	ToDate     *string
}

// EventPublisher defines the interface for publishing domain events
type EventPublisher interface {
	// Publish publishes a domain event
	Publish(ctx context.Context, event DomainEvent) error

	// PublishAll publishes multiple domain events
	PublishAll(ctx context.Context, events []DomainEvent) error
}
