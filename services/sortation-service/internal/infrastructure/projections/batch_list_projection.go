package projections

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BatchListProjection is a denormalized read model optimized for list queries
type BatchListProjection struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BatchID          string             `bson:"batchId" json:"batchId"`
	SortationCenter  string             `bson:"sortationCenter" json:"sortationCenter"`
	DestinationGroup string             `bson:"destinationGroup" json:"destinationGroup"`
	CarrierID        string             `bson:"carrierId" json:"carrierId"`
	CarrierName      string             `bson:"carrierName,omitempty" json:"carrierName,omitempty"`
	Status           string             `bson:"status" json:"status"`

	// Chute assignment
	AssignedChuteID  string             `bson:"assignedChuteId,omitempty" json:"assignedChuteId,omitempty"`
	AssignedChuteNum int                `bson:"assignedChuteNum,omitempty" json:"assignedChuteNum,omitempty"`

	// Package counts
	TotalPackages    int                `bson:"totalPackages" json:"totalPackages"`
	SortedPackages   int                `bson:"sortedPackages" json:"sortedPackages"`
	PendingPackages  int                `bson:"pendingPackages" json:"pendingPackages"`
	TotalWeight      float64            `bson:"totalWeight" json:"totalWeight"`

	// Progress
	SortingProgress  float64            `bson:"sortingProgress" json:"sortingProgress"` // 0-100%

	// Dispatch information
	TrailerID        string             `bson:"trailerId,omitempty" json:"trailerId,omitempty"`
	DispatchDock     string             `bson:"dispatchDock,omitempty" json:"dispatchDock,omitempty"`

	// Timestamps
	CreatedAt        time.Time          `bson:"createdAt" json:"createdAt"`
	ReadyAt          *time.Time         `bson:"readyAt,omitempty" json:"readyAt,omitempty"`
	DispatchedAt     *time.Time         `bson:"dispatchedAt,omitempty" json:"dispatchedAt,omitempty"`
	UpdatedAt        time.Time          `bson:"updatedAt" json:"updatedAt"`

	// Computed fields
	IsReady          bool               `bson:"isReady" json:"isReady"`
	IsDispatched     bool               `bson:"isDispatched" json:"isDispatched"`
}

// BatchListFilter represents filter criteria for batch list queries
type BatchListFilter struct {
	Status           *string
	SortationCenter  *string
	DestinationGroup *string
	CarrierID        *string
	ChuteID          *string
	TrailerID        *string
	IsReady          *bool
	IsDispatched     *bool
	CreatedAfter     *time.Time
	CreatedBefore    *time.Time
	SearchTerm       string
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

// SortationDashboardStats represents dashboard statistics
type SortationDashboardStats struct {
	TotalBatches        int64              `json:"totalBatches"`
	OpenBatches         int64              `json:"openBatches"`
	ReadyBatches        int64              `json:"readyBatches"`
	DispatchedToday     int64              `json:"dispatchedToday"`
	TotalPackagesSorted int64              `json:"totalPackagesSorted"`
	TotalWeightKg       float64            `json:"totalWeightKg"`
	BatchesByCarrier    map[string]int64   `json:"batchesByCarrier"`
	BatchesByChute      map[string]int64   `json:"batchesByChute"`
	AvgSortTimeMin      float64            `json:"avgSortTimeMin"`
}

// ChuteStatus represents the status of a sortation chute
type ChuteStatus struct {
	ChuteID          string  `json:"chuteId"`
	ChuteNumber      int     `json:"chuteNumber"`
	Zone             string  `json:"zone"`
	CurrentBatchID   string  `json:"currentBatchId,omitempty"`
	DestinationGroup string  `json:"destinationGroup,omitempty"`
	CarrierID        string  `json:"carrierId,omitempty"`
	PackageCount     int     `json:"packageCount"`
	IsActive         bool    `json:"isActive"`
	IsFull           bool    `json:"isFull"`
}

// DestinationSummary represents a summary by destination
type DestinationSummary struct {
	DestinationGroup string  `json:"destinationGroup"`
	TotalBatches     int64   `json:"totalBatches"`
	TotalPackages    int64   `json:"totalPackages"`
	TotalWeight      float64 `json:"totalWeight"`
	ReadyBatches     int64   `json:"readyBatches"`
}
