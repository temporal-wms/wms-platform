package domain

import "context"

// PickTaskRepository defines the interface for pick task persistence
type PickTaskRepository interface {
	Save(ctx context.Context, task *PickTask) error
	FindByID(ctx context.Context, taskID string) (*PickTask, error)
	FindByOrderID(ctx context.Context, orderID string) ([]*PickTask, error)
	FindByWaveID(ctx context.Context, waveID string) ([]*PickTask, error)
	FindByPickerID(ctx context.Context, pickerID string) ([]*PickTask, error)
	FindByStatus(ctx context.Context, status PickTaskStatus) ([]*PickTask, error)
	FindActiveByPicker(ctx context.Context, pickerID string) (*PickTask, error)
	FindPendingByZone(ctx context.Context, zone string, limit int) ([]*PickTask, error)
	Delete(ctx context.Context, taskID string) error
}

// EventPublisher defines the interface for publishing domain events
type EventPublisher interface {
	Publish(ctx context.Context, event DomainEvent) error
	PublishAll(ctx context.Context, events []DomainEvent) error
}
