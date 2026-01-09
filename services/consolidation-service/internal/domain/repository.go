package domain

import "context"

// ConsolidationRepository defines the interface for consolidation persistence
type ConsolidationRepository interface {
	Save(ctx context.Context, unit *ConsolidationUnit) error
	FindByID(ctx context.Context, consolidationID string) (*ConsolidationUnit, error)
	FindByOrderID(ctx context.Context, orderID string) (*ConsolidationUnit, error)
	FindByWaveID(ctx context.Context, waveID string) ([]*ConsolidationUnit, error)
	FindByStatus(ctx context.Context, status ConsolidationStatus) ([]*ConsolidationUnit, error)
	FindByStation(ctx context.Context, station string) ([]*ConsolidationUnit, error)
	FindPending(ctx context.Context, limit int) ([]*ConsolidationUnit, error)
	Delete(ctx context.Context, consolidationID string) error
}

// EventPublisher defines the interface for publishing domain events
type EventPublisher interface {
	Publish(ctx context.Context, event DomainEvent) error
	PublishAll(ctx context.Context, events []DomainEvent) error
}
