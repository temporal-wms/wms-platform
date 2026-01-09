package domain

import "context"

// ProcessPathRepository defines the interface for process path persistence
type ProcessPathRepository interface {
	// Save persists a process path
	Save(ctx context.Context, processPath *ProcessPath) error

	// FindByID retrieves a process path by its ID
	FindByID(ctx context.Context, id string) (*ProcessPath, error)

	// FindByPathID retrieves a process path by pathId
	FindByPathID(ctx context.Context, pathID string) (*ProcessPath, error)

	// FindByOrderID retrieves a process path by order ID
	FindByOrderID(ctx context.Context, orderID string) (*ProcessPath, error)

	// Update updates an existing process path
	Update(ctx context.Context, processPath *ProcessPath) error
}
