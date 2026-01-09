package application

import "time"

// PickRouteDTO represents a pick route in responses
type PickRouteDTO struct {
	RouteID           string          `json:"routeId"`
	OrderID           string          `json:"orderId"`
	WaveID            string          `json:"waveId"`
	PickerID          string          `json:"pickerId,omitempty"`
	Status            string          `json:"status"`
	Strategy          string          `json:"strategy"`
	Stops             []RouteStopDTO  `json:"stops"`
	EstimatedDistance float64         `json:"estimatedDistance"`
	ActualDistance    float64         `json:"actualDistance"`
	EstimatedTime     int64           `json:"estimatedTime"` // Duration in seconds
	ActualTime        int64           `json:"actualTime"`    // Duration in seconds
	StartLocation     LocationDTO     `json:"startLocation"`
	EndLocation       LocationDTO     `json:"endLocation"`
	Zone              string          `json:"zone"`
	TotalItems        int             `json:"totalItems"`
	PickedItems       int             `json:"pickedItems"`
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
	StartedAt         *time.Time      `json:"startedAt,omitempty"`
	CompletedAt       *time.Time      `json:"completedAt,omitempty"`
}

// RouteStopDTO represents a stop in the route
type RouteStopDTO struct {
	StopNumber int          `json:"stopNumber"`
	Location   LocationDTO  `json:"location"`
	SKU        string       `json:"sku"`
	Quantity   int          `json:"quantity"`
	PickedQty  int          `json:"pickedQty"`
	Status     string       `json:"status"`
	ToteID     string       `json:"toteId,omitempty"`
	PickedAt   *time.Time   `json:"pickedAt,omitempty"`
	Notes      string       `json:"notes,omitempty"`
}

// LocationDTO represents a warehouse location
type LocationDTO struct {
	LocationID string  `json:"locationId"`
	Aisle      string  `json:"aisle"`
	Rack       int     `json:"rack"`
	Level      int     `json:"level"`
	Position   string  `json:"position"`
	Zone       string  `json:"zone"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
}

// MultiRouteResultDTO contains the result of multi-route calculation
type MultiRouteResultDTO struct {
	OrderID       string           `json:"orderId"`
	Routes        []PickRouteDTO   `json:"routes"`
	TotalRoutes   int              `json:"totalRoutes"`
	SplitReason   string           `json:"splitReason"`
	ZoneBreakdown map[string]int   `json:"zoneBreakdown"`
	TotalItems    int              `json:"totalItems"`
	CreatedAt     time.Time        `json:"createdAt"`
}
