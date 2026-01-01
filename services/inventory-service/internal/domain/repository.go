package domain

import "context"

// InventoryRepository defines the interface for inventory persistence
type InventoryRepository interface {
	Save(ctx context.Context, item *InventoryItem) error
	FindBySKU(ctx context.Context, sku string) (*InventoryItem, error)
	FindByLocation(ctx context.Context, locationID string) ([]*InventoryItem, error)
	FindByZone(ctx context.Context, zone string) ([]*InventoryItem, error)
	FindByOrderID(ctx context.Context, orderID string) ([]*InventoryItem, error)
	FindLowStock(ctx context.Context) ([]*InventoryItem, error)
	FindAll(ctx context.Context, limit, offset int) ([]*InventoryItem, error)
	Delete(ctx context.Context, sku string) error
}

// EventPublisher defines the interface for publishing domain events
type EventPublisher interface {
	Publish(ctx context.Context, event DomainEvent) error
	PublishAll(ctx context.Context, events []DomainEvent) error
}
