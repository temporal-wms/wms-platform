package domain

import "context"

// WorkerRepository defines the interface for worker persistence
type WorkerRepository interface {
	Save(ctx context.Context, worker *Worker) error
	FindByID(ctx context.Context, workerID string) (*Worker, error)
	FindByEmployeeID(ctx context.Context, employeeID string) (*Worker, error)
	FindByStatus(ctx context.Context, status WorkerStatus) ([]*Worker, error)
	FindByZone(ctx context.Context, zone string) ([]*Worker, error)
	FindAvailableBySkill(ctx context.Context, taskType TaskType, zone string) ([]*Worker, error)
	FindAll(ctx context.Context, limit, offset int) ([]*Worker, error)
	Delete(ctx context.Context, workerID string) error
}

// EventPublisher defines the interface for publishing domain events
type EventPublisher interface {
	Publish(ctx context.Context, event DomainEvent) error
	PublishAll(ctx context.Context, events []DomainEvent) error
}
