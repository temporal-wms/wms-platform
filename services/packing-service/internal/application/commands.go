package application

import "github.com/wms-platform/packing-service/internal/domain"

// CreatePackTaskCommand creates a new packing task
type CreatePackTaskCommand struct {
	TaskID  string
	OrderID string
	WaveID  string
	Items   []domain.PackItem
}

// AssignPackTaskCommand assigns a packing task to a packer
type AssignPackTaskCommand struct {
	TaskID   string
	PackerID string
	Station  string
}

// StartPackTaskCommand starts a packing task
type StartPackTaskCommand struct {
	TaskID string
}

// VerifyItemCommand verifies an item during packing
type VerifyItemCommand struct {
	TaskID string
	SKU    string
}

// SelectPackagingCommand selects packaging for the task
type SelectPackagingCommand struct {
	TaskID      string
	PackageType domain.PackageType
	Dimensions  domain.Dimensions
	Materials   []string
}

// SealPackageCommand seals the package
type SealPackageCommand struct {
	TaskID string
}

// ApplyLabelCommand applies a shipping label
type ApplyLabelCommand struct {
	TaskID string
	Label  domain.ShippingLabel
}

// CompletePackTaskCommand completes a packing task
type CompletePackTaskCommand struct {
	TaskID string
}

// GetPackTaskQuery retrieves a packing task by ID
type GetPackTaskQuery struct {
	TaskID string
}

// GetByOrderQuery retrieves a packing task by order ID
type GetByOrderQuery struct {
	OrderID string
}

// GetByWaveQuery retrieves packing tasks by wave ID
type GetByWaveQuery struct {
	WaveID string
}

// GetByTrackingQuery retrieves a packing task by tracking number
type GetByTrackingQuery struct {
	TrackingNumber string
}

// GetPendingQuery retrieves pending packing tasks
type GetPendingQuery struct {
	Limit int
}
