package application

// CreateItemCommand represents the command to create a new inventory item
type CreateItemCommand struct {
	SKU             string
	ProductName     string
	ReorderPoint    int
	ReorderQuantity int
}

// ReceiveStockCommand represents the command to receive stock
type ReceiveStockCommand struct {
	SKU         string
	LocationID  string
	Zone        string
	Quantity    int
	ReferenceID string
	CreatedBy   string
}

// ReserveCommand represents the command to reserve stock
type ReserveCommand struct {
	SKU        string
	OrderID    string
	LocationID string
	Quantity   int
}

// PickCommand represents the command to pick stock
type PickCommand struct {
	SKU        string
	OrderID    string
	LocationID string
	Quantity   int
	CreatedBy  string
}

// ReleaseReservationCommand represents the command to release a reservation
type ReleaseReservationCommand struct {
	SKU     string
	OrderID string
}

// AdjustCommand represents the command to adjust inventory
type AdjustCommand struct {
	SKU         string
	LocationID  string
	NewQuantity int
	Reason      string
	CreatedBy   string
}

// GetItemQuery represents the query to get an item by SKU
type GetItemQuery struct {
	SKU string
}

// GetByLocationQuery represents the query to get items by location
type GetByLocationQuery struct {
	LocationID string
}

// GetByZoneQuery represents the query to get items by zone
type GetByZoneQuery struct {
	Zone string
}

// ListInventoryQuery represents the query to list inventory with pagination
type ListInventoryQuery struct {
	// Basic filters
	SKU         *string
	ProductName *string
	SearchTerm  string

	// CQRS filters
	IsLowStock      *bool
	IsOutOfStock    *bool
	MinQuantity     *int
	MaxQuantity     *int
	HasReservations *bool
	LocationID      *string
	Zone            *string

	// Pagination and sorting
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}
