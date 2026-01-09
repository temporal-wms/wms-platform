package application

import "time"

// PickTaskDTO represents a pick task in responses
type PickTaskDTO struct {
	TaskID           string             `json:"taskId"`
	OrderID          string             `json:"orderId"`
	WaveID           string             `json:"waveId"`
	RouteID          string             `json:"routeId"`
	PickerID         string             `json:"pickerId,omitempty"`
	Status           string             `json:"status"`
	Method           string             `json:"method"`
	Items            []PickItemDTO      `json:"items"`
	ToteID           string             `json:"toteId,omitempty"`
	Zone             string             `json:"zone"`
	Priority         int                `json:"priority"`
	TotalItems       int                `json:"totalItems"`
	PickedItemsCount int                `json:"pickedItemsCount"`
	PickedItems      []PickedItemDTO    `json:"pickedItems,omitempty"`
	Exceptions       []PickExceptionDTO `json:"exceptions,omitempty"`
	CreatedAt        time.Time          `json:"createdAt"`
	UpdatedAt        time.Time          `json:"updatedAt"`
	AssignedAt       *time.Time         `json:"assignedAt,omitempty"`
	StartedAt        *time.Time         `json:"startedAt,omitempty"`
	CompletedAt      *time.Time         `json:"completedAt,omitempty"`
}

// PickItemDTO represents an item to be picked
type PickItemDTO struct {
	SKU         string       `json:"sku"`
	ProductName string       `json:"productName"`
	Quantity    int          `json:"quantity"`
	PickedQty   int          `json:"pickedQty"`
	Location    LocationDTO  `json:"location"`
	Status      string       `json:"status"`
	ToteID      string       `json:"toteId,omitempty"`
	PickedAt    *time.Time   `json:"pickedAt,omitempty"`
	VerifiedAt  *time.Time   `json:"verifiedAt,omitempty"`
	Notes       string       `json:"notes,omitempty"`
}

// LocationDTO represents a warehouse location
type LocationDTO struct {
	LocationID string `json:"locationId"`
	Aisle      string `json:"aisle"`
	Rack       int    `json:"rack"`
	Level      int    `json:"level"`
	Position   string `json:"position"`
	Zone       string `json:"zone"`
}

// PickExceptionDTO represents an exception during picking
type PickExceptionDTO struct {
	ExceptionID  string     `json:"exceptionId"`
	SKU          string     `json:"sku"`
	LocationID   string     `json:"locationId"`
	Reason       string     `json:"reason"`
	RequestedQty int        `json:"requestedQty"`
	AvailableQty int        `json:"availableQty"`
	Resolution   string     `json:"resolution,omitempty"`
	ResolvedAt   *time.Time `json:"resolvedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}

// PickedItemDTO represents a picked item in the response
type PickedItemDTO struct {
	SKU        string    `json:"sku"`
	Quantity   int       `json:"quantity"`
	LocationID string    `json:"locationId"`
	ToteID     string    `json:"toteId"`
	PickedAt   time.Time `json:"pickedAt"`
}
