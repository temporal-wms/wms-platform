package projections

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TaskListProjection is a denormalized read model optimized for list queries
type TaskListProjection struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TaskID          string             `bson:"taskId" json:"taskId"`
	SKU             string             `bson:"sku" json:"sku"`
	ProductName     string             `bson:"productName,omitempty" json:"productName,omitempty"`
	Quantity        int                `bson:"quantity" json:"quantity"`
	Status          string             `bson:"status" json:"status"`

	// Source information
	SourceToteID    string             `bson:"sourceToteId" json:"sourceToteId"`
	ShipmentID      string             `bson:"shipmentId,omitempty" json:"shipmentId,omitempty"`

	// Target location (denormalized)
	TargetLocationID string            `bson:"targetLocationId,omitempty" json:"targetLocationId,omitempty"`
	TargetZone       string            `bson:"targetZone,omitempty" json:"targetZone,omitempty"`
	TargetAisle      string            `bson:"targetAisle,omitempty" json:"targetAisle,omitempty"`
	TargetRack       int               `bson:"targetRack,omitempty" json:"targetRack,omitempty"`
	TargetLevel      int               `bson:"targetLevel,omitempty" json:"targetLevel,omitempty"`
	TargetBin        string            `bson:"targetBin,omitempty" json:"targetBin,omitempty"`

	// Strategy and constraints
	Strategy         string             `bson:"strategy" json:"strategy"`
	IsHazmat         bool               `bson:"isHazmat" json:"isHazmat"`
	RequiresColdChain bool              `bson:"requiresColdChain" json:"requiresColdChain"`
	IsOversized      bool               `bson:"isOversized" json:"isOversized"`

	// Worker assignment
	AssignedWorkerID string             `bson:"assignedWorkerId,omitempty" json:"assignedWorkerId,omitempty"`
	AssignedWorkerName string           `bson:"assignedWorkerName,omitempty" json:"assignedWorkerName,omitempty"`

	// Timestamps
	CreatedAt        time.Time          `bson:"createdAt" json:"createdAt"`
	AssignedAt       *time.Time         `bson:"assignedAt,omitempty" json:"assignedAt,omitempty"`
	StartedAt        *time.Time         `bson:"startedAt,omitempty" json:"startedAt,omitempty"`
	CompletedAt      *time.Time         `bson:"completedAt,omitempty" json:"completedAt,omitempty"`
	UpdatedAt        time.Time          `bson:"updatedAt" json:"updatedAt"`

	// Computed fields
	DurationMins     float64            `bson:"durationMins,omitempty" json:"durationMins,omitempty"`
	IsOverdue        bool               `bson:"isOverdue" json:"isOverdue"`
}

// TaskListFilter represents filter criteria for task list queries
type TaskListFilter struct {
	Status            *string
	Strategy          *string
	Zone              *string
	AssignedWorkerID  *string
	ShipmentID        *string
	IsHazmat          *bool
	RequiresColdChain *bool
	IsOversized       *bool
	IsOverdue         *bool
	CreatedAfter      *time.Time
	CreatedBefore     *time.Time
	SearchTerm        string
}

// Pagination represents pagination parameters
type Pagination struct {
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

// PagedResult represents a paginated result set
type PagedResult[T any] struct {
	Items   []T   `json:"items"`
	Total   int64 `json:"total"`
	Limit   int   `json:"limit"`
	Offset  int   `json:"offset"`
	HasMore bool  `json:"hasMore"`
}

// StowDashboardStats represents dashboard statistics for stow operations
type StowDashboardStats struct {
	TotalTasks          int64   `json:"totalTasks"`
	PendingTasks        int64   `json:"pendingTasks"`
	AssignedTasks       int64   `json:"assignedTasks"`
	InProgressTasks     int64   `json:"inProgressTasks"`
	CompletedToday      int64   `json:"completedToday"`
	FailedTasks         int64   `json:"failedTasks"`
	OverdueTasks        int64   `json:"overdueTasks"`
	AvgCompletionTimeMins float64 `json:"avgCompletionTimeMins"`
	TasksByZone         map[string]int64 `json:"tasksByZone"`
}

// ZoneCapacity represents zone capacity information
type ZoneCapacity struct {
	Zone            string  `json:"zone"`
	TotalLocations  int64   `json:"totalLocations"`
	UsedLocations   int64   `json:"usedLocations"`
	AvailableLocations int64 `json:"availableLocations"`
	UtilizationPct  float64 `json:"utilizationPct"`
}
