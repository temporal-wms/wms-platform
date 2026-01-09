package domain

import "context"

// WallingTaskRepository defines the interface for walling task persistence
type WallingTaskRepository interface {
	// Save saves a walling task
	Save(ctx context.Context, task *WallingTask) error

	// FindByID finds a task by its MongoDB ObjectID
	FindByID(ctx context.Context, id string) (*WallingTask, error)

	// FindByTaskID finds a task by its task ID
	FindByTaskID(ctx context.Context, taskID string) (*WallingTask, error)

	// FindByOrderID finds a task by order ID
	FindByOrderID(ctx context.Context, orderID string) (*WallingTask, error)

	// FindByWalliner finds active task for a walliner
	FindActiveByWalliner(ctx context.Context, wallinerID string) (*WallingTask, error)

	// FindPendingByPutWall finds pending tasks for a put wall
	FindPendingByPutWall(ctx context.Context, putWallID string, limit int) ([]*WallingTask, error)

	// FindByStatus finds tasks by status
	FindByStatus(ctx context.Context, status WallingTaskStatus) ([]*WallingTask, error)

	// Update updates a walling task
	Update(ctx context.Context, task *WallingTask) error
}
