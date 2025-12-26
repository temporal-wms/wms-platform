package outbox

import "context"

// Repository defines the interface for outbox event persistence
type Repository interface {
	// Save saves an outbox event
	Save(ctx context.Context, event *OutboxEvent) error

	// SaveAll saves multiple outbox events in a single operation
	SaveAll(ctx context.Context, events []*OutboxEvent) error

	// FindUnpublished retrieves unpublished events up to the specified limit
	FindUnpublished(ctx context.Context, limit int) ([]*OutboxEvent, error)

	// MarkPublished marks an event as published
	MarkPublished(ctx context.Context, eventID string) error

	// IncrementRetry increments the retry count and updates last error
	IncrementRetry(ctx context.Context, eventID string, errorMsg string) error

	// DeletePublished deletes published events older than the specified duration
	DeletePublished(ctx context.Context, olderThan int64) error

	// GetByID retrieves an outbox event by ID
	GetByID(ctx context.Context, eventID string) (*OutboxEvent, error)

	// FindByAggregateID retrieves all events for a specific aggregate
	FindByAggregateID(ctx context.Context, aggregateID string) ([]*OutboxEvent, error)
}
