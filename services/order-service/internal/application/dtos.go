package application

import "time"

// OrderDTO represents an order in the application layer responses
type OrderDTO struct {
	OrderID            string         `json:"orderId"`
	CustomerID         string         `json:"customerId"`
	Items              []OrderItemDTO `json:"items"`
	ShippingAddress    AddressDTO     `json:"shippingAddress"`
	Priority           string         `json:"priority"`
	Status             string         `json:"status"`
	PromisedDeliveryAt time.Time      `json:"promisedDeliveryAt"`
	WaveID             string         `json:"waveId,omitempty"`
	TrackingNumber     string         `json:"trackingNumber,omitempty"`
	TotalItems         int            `json:"totalItems"`
	TotalWeight        float64        `json:"totalWeight"`
	IsMultiItem        bool           `json:"isMultiItem"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
}

// OrderItemDTO represents an order item in responses
type OrderItemDTO struct {
	SKU      string  `json:"sku"`
	Quantity int     `json:"quantity"`
	Weight   float64 `json:"weight"`
}

// AddressDTO represents an address in responses
type AddressDTO struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	State   string `json:"state"`
	ZipCode string `json:"zipCode"`
	Country string `json:"country"`
}

// OrderListDTO represents a simplified order for list operations (denormalized for CQRS)
type OrderListDTO struct {
	OrderID            string    `json:"orderId"`
	CustomerID         string    `json:"customerId"`
	CustomerName       string    `json:"customerName,omitempty"`       // Denormalized
	Status             string    `json:"status"`
	Priority           string    `json:"priority"`
	TotalItems         int       `json:"totalItems"`
	TotalWeight        float64   `json:"totalWeight"`
	TotalValue         float64   `json:"totalValue"`

	// Wave information (denormalized)
	WaveID             string    `json:"waveId,omitempty"`
	WaveStatus         string    `json:"waveStatus,omitempty"`         // Denormalized
	WaveType           string    `json:"waveType,omitempty"`           // Denormalized

	// Fulfillment information
	AssignedPicker     string    `json:"assignedPicker,omitempty"`
	TrackingNumber     string    `json:"trackingNumber,omitempty"`
	Carrier            string    `json:"carrier,omitempty"`

	// Address information
	ShipToCity         string    `json:"shipToCity"`
	ShipToState        string    `json:"shipToState"`
	ShipToZipCode      string    `json:"shipToZipCode"`

	// Computed fields
	DaysUntilPromised  int       `json:"daysUntilPromised"`
	IsLate             bool      `json:"isLate"`
	IsPriority         bool      `json:"isPriority"`

	// Timestamps
	ReceivedAt         string    `json:"receivedAt"`                   // ISO8601 string
	PromisedDeliveryAt string    `json:"promisedDeliveryAt"`           // ISO8601 string
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// PagedOrdersResult represents a paginated list of orders
type PagedOrdersResult struct {
	Data       []OrderListDTO `json:"data"`
	Page       int64          `json:"page"`
	PageSize   int64          `json:"pageSize"`
	TotalItems int64          `json:"totalItems"`
	TotalPages int64          `json:"totalPages"`
}

// OrderCreatedResponse represents the response after creating an order
type OrderCreatedResponse struct {
	Order      OrderDTO `json:"order"`
	WorkflowID string   `json:"workflowId,omitempty"`
}
