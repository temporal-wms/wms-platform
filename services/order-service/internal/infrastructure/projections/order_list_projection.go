package projections

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OrderListProjection is a denormalized read model optimized for list queries
// This is the "read side" in CQRS - optimized for queries, not domain logic
type OrderListProjection struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OrderID           string             `bson:"orderId" json:"orderId"`
	CustomerID        string             `bson:"customerId" json:"customerId"`
	CustomerName      string             `bson:"customerName,omitempty" json:"customerName,omitempty"` // Denormalized
	Status            string             `bson:"status" json:"status"`
	Priority          string             `bson:"priority" json:"priority"`
	TotalItems        int                `bson:"totalItems" json:"totalItems"`
	TotalWeight       float64            `bson:"totalWeight" json:"totalWeight"`
	TotalValue        float64            `bson:"totalValue" json:"totalValue"`

	// Wave information (denormalized for quick filtering)
	WaveID            string             `bson:"waveId,omitempty" json:"waveId,omitempty"`
	WaveStatus        string             `bson:"waveStatus,omitempty" json:"waveStatus,omitempty"` // Denormalized from wave
	WaveType          string             `bson:"waveType,omitempty" json:"waveType,omitempty"`      // Denormalized from wave

	// Fulfillment information (denormalized for quick filtering)
	AssignedPicker    string             `bson:"assignedPicker,omitempty" json:"assignedPicker,omitempty"`
	PickingStartedAt  *time.Time         `bson:"pickingStartedAt,omitempty" json:"pickingStartedAt,omitempty"`
	PickingCompletedAt *time.Time        `bson:"pickingCompletedAt,omitempty" json:"pickingCompletedAt,omitempty"`

	// Shipping information (denormalized)
	TrackingNumber    string             `bson:"trackingNumber,omitempty" json:"trackingNumber,omitempty"`
	Carrier           string             `bson:"carrier,omitempty" json:"carrier,omitempty"`
	EstimatedDelivery *time.Time         `bson:"estimatedDelivery,omitempty" json:"estimatedDelivery,omitempty"`

	// Address information (denormalized for search)
	ShipToCity        string             `bson:"shipToCity" json:"shipToCity"`
	ShipToState       string             `bson:"shipToState" json:"shipToState"`
	ShipToZipCode     string             `bson:"shipToZipCode" json:"shipToZipCode"`
	ShipToCountry     string             `bson:"shipToCountry" json:"shipToCountry"`

	// Timestamps
	ReceivedAt        time.Time          `bson:"receivedAt" json:"receivedAt"`
	PromisedDeliveryAt time.Time         `bson:"promisedDeliveryAt" json:"promisedDeliveryAt"`
	CreatedAt         time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt         time.Time          `bson:"updatedAt" json:"updatedAt"`

	// Computed fields for filtering/sorting
	DaysUntilPromised int                `bson:"daysUntilPromised" json:"daysUntilPromised"` // Calculated field
	IsLate            bool               `bson:"isLate" json:"isLate"`                       // Calculated field
	IsPriority        bool               `bson:"isPriority" json:"isPriority"`               // Calculated from priority
}

// OrderListFilter represents filter criteria for order list queries
type OrderListFilter struct {
	Status            *string
	Priority          *string
	WaveID            *string
	CustomerID        *string
	AssignedPicker    *string
	ShipToState       *string
	ShipToCountry     *string
	IsLate            *bool
	IsPriority        *bool
	ReceivedAfter     *time.Time
	ReceivedBefore    *time.Time
	SearchTerm        string // For text search on orderID, customerID, etc.
}

// Pagination represents pagination parameters
type Pagination struct {
	Limit  int
	Offset int
	SortBy string // Field name to sort by
	SortOrder string // "asc" or "desc"
}

// PagedResult represents a paginated result set
type PagedResult[T any] struct {
	Items      []T   `json:"items"`
	Total      int64 `json:"total"`
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
	HasMore    bool  `json:"hasMore"`
}
