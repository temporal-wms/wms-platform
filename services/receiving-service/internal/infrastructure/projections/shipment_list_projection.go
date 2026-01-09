package projections

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ShipmentListProjection is a denormalized read model optimized for list queries
// This is the "read side" in CQRS - optimized for queries, not domain logic
type ShipmentListProjection struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ShipmentID      string             `bson:"shipmentId" json:"shipmentId"`
	ASNID           string             `bson:"asnId" json:"asnId"`
	SupplierID      string             `bson:"supplierId" json:"supplierId"`
	SupplierName    string             `bson:"supplierName,omitempty" json:"supplierName,omitempty"`
	Status          string             `bson:"status" json:"status"`
	DockID          string             `bson:"dockId,omitempty" json:"dockId,omitempty"`

	// Item counts
	TotalItemsExpected int `bson:"totalItemsExpected" json:"totalItemsExpected"`
	TotalItemsReceived int `bson:"totalItemsReceived" json:"totalItemsReceived"`
	TotalDamaged       int `bson:"totalDamaged" json:"totalDamaged"`
	DiscrepancyCount   int `bson:"discrepancyCount" json:"discrepancyCount"`

	// Progress
	ReceivingProgress float64 `bson:"receivingProgress" json:"receivingProgress"` // 0-100%

	// Timestamps
	ExpectedArrival time.Time  `bson:"expectedArrival" json:"expectedArrival"`
	ArrivedAt       *time.Time `bson:"arrivedAt,omitempty" json:"arrivedAt,omitempty"`
	CompletedAt     *time.Time `bson:"completedAt,omitempty" json:"completedAt,omitempty"`
	CreatedAt       time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time  `bson:"updatedAt" json:"updatedAt"`

	// Computed fields for filtering/sorting
	IsOnTime    bool `bson:"isOnTime" json:"isOnTime"`
	IsLate      bool `bson:"isLate" json:"isLate"`
	HasIssues   bool `bson:"hasIssues" json:"hasIssues"` // Has discrepancies or damages
}

// ShipmentListFilter represents filter criteria for shipment list queries
type ShipmentListFilter struct {
	Status            *string
	SupplierID        *string
	DockID            *string
	IsLate            *bool
	HasIssues         *bool
	ExpectedAfter     *time.Time
	ExpectedBefore    *time.Time
	ArrivedAfter      *time.Time
	ArrivedBefore     *time.Time
	SearchTerm        string // For text search on shipmentID, ASNID, etc.
}

// Pagination represents pagination parameters
type Pagination struct {
	Limit     int
	Offset    int
	SortBy    string // Field name to sort by
	SortOrder string // "asc" or "desc"
}

// PagedResult represents a paginated result set
type PagedResult[T any] struct {
	Items   []T   `json:"items"`
	Total   int64 `json:"total"`
	Limit   int   `json:"limit"`
	Offset  int   `json:"offset"`
	HasMore bool  `json:"hasMore"`
}

// ReceivingDashboardStats represents dashboard statistics
type ReceivingDashboardStats struct {
	TotalShipments      int64   `json:"totalShipments"`
	ExpectedToday       int64   `json:"expectedToday"`
	ArrivedToday        int64   `json:"arrivedToday"`
	CompletedToday      int64   `json:"completedToday"`
	InProgress          int64   `json:"inProgress"`
	WithDiscrepancies   int64   `json:"withDiscrepancies"`
	LateShipments       int64   `json:"lateShipments"`
	AvgReceivingTimeMin float64 `json:"avgReceivingTimeMin"`
}
