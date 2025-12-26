package application

import "time"

// InventoryItemDTO represents an inventory item in responses
type InventoryItemDTO struct {
	SKU               string               `json:"sku"`
	ProductName       string               `json:"productName"`
	Locations         []StockLocationDTO   `json:"locations"`
	TotalQuantity     int                  `json:"totalQuantity"`
	ReservedQuantity  int                  `json:"reservedQuantity"`
	AvailableQuantity int                  `json:"availableQuantity"`
	ReorderPoint      int                  `json:"reorderPoint"`
	ReorderQuantity   int                  `json:"reorderQuantity"`
	Reservations      []ReservationDTO     `json:"reservations,omitempty"`
	LastCycleCount    *time.Time           `json:"lastCycleCount,omitempty"`
	CreatedAt         time.Time            `json:"createdAt"`
	UpdatedAt         time.Time            `json:"updatedAt"`
}

// StockLocationDTO represents stock at a specific location
type StockLocationDTO struct {
	LocationID string `json:"locationId"`
	Zone       string `json:"zone"`
	Aisle      string `json:"aisle"`
	Rack       int    `json:"rack"`
	Level      int    `json:"level"`
	Quantity   int    `json:"quantity"`
	Reserved   int    `json:"reserved"`
	Available  int    `json:"available"`
}

// ReservationDTO represents a stock reservation
type ReservationDTO struct {
	ReservationID string    `json:"reservationId"`
	OrderID       string    `json:"orderId"`
	Quantity      int       `json:"quantity"`
	LocationID    string    `json:"locationId"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	ExpiresAt     time.Time `json:"expiresAt"`
}

// InventoryListDTO represents a simplified inventory item for list operations
type InventoryListDTO struct {
	SKU               string    `json:"sku"`
	ProductName       string    `json:"productName"`
	TotalQuantity     int       `json:"totalQuantity"`
	ReservedQuantity  int       `json:"reservedQuantity"`
	AvailableQuantity int       `json:"availableQuantity"`
	ReorderPoint      int       `json:"reorderPoint"`
	ReorderQuantity   int       `json:"reorderQuantity"`

	// CQRS computed fields
	IsLowStock         bool     `json:"isLowStock"`
	IsOutOfStock       bool     `json:"isOutOfStock"`
	LocationCount      int      `json:"locationCount"`
	PrimaryLocation    string   `json:"primaryLocation,omitempty"`
	AvailableLocations []string `json:"availableLocations"`
	ActiveReservations int      `json:"activeReservations"`
	ReservedOrders     []string `json:"reservedOrders,omitempty"`

	UpdatedAt time.Time `json:"updatedAt"`
}
