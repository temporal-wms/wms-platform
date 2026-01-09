package application

import (
	"time"

	"github.com/wms-platform/services/order-service/internal/domain"
)

// CreateOrderCommand represents the command to create a new order
type CreateOrderCommand struct {
	CustomerID         string
	Items              []OrderItemInput
	ShippingAddress    AddressInput
	Priority           string
	PromisedDeliveryAt time.Time
}

// OrderItemInput represents an order item in a command
type OrderItemInput struct {
	SKU      string
	Quantity int
	Weight   float64
}

// AddressInput represents an address in a command
type AddressInput struct {
	Street  string
	City    string
	State   string
	ZipCode string
	Country string
}

// ValidateOrderCommand represents the command to validate an order
type ValidateOrderCommand struct {
	OrderID string
}

// CancelOrderCommand represents the command to cancel an order
type CancelOrderCommand struct {
	OrderID string
	Reason  string
}

// AssignToWaveCommand represents the command to assign an order to a wave
type AssignToWaveCommand struct {
	OrderID string
	WaveID  string
}

// MarkShippedCommand represents the command to mark an order as shipped
type MarkShippedCommand struct {
	OrderID        string
	TrackingNumber string
}

// ListOrdersQuery represents the query to list orders with filters and pagination
type ListOrdersQuery struct {
	// Basic filters
	CustomerID *string
	Status     *string
	Priority   *string

	// Extended CQRS filters
	WaveID         *string
	AssignedPicker *string
	ShipToState    *string
	ShipToCountry  *string
	IsLate         *bool
	IsPriority     *bool
	SearchTerm     string

	// Pagination
	Page     int64
	PageSize int64
	Limit    int
	Offset   int
	SortBy   string
	SortOrder string
}

// GetOrderQuery represents the query to get a single order
type GetOrderQuery struct {
	OrderID string
}

// Helper methods to convert inputs to domain models

// ToDomainOrderItems converts OrderItemInput slice to domain.OrderItem slice
func (c *CreateOrderCommand) ToDomainOrderItems() []domain.OrderItem {
	items := make([]domain.OrderItem, 0, len(c.Items))
	for _, item := range c.Items {
		items = append(items, domain.OrderItem{
			SKU:      item.SKU,
			Quantity: item.Quantity,
			Weight:   item.Weight,
		})
	}
	return items
}

// ToDomainAddress converts AddressInput to domain.Address
func (a *AddressInput) ToDomainAddress() domain.Address {
	return domain.Address{
		Street:  a.Street,
		City:    a.City,
		State:   a.State,
		ZipCode: a.ZipCode,
		Country: a.Country,
	}
}

// ToDomainPriority converts string priority to domain.Priority
func (c *CreateOrderCommand) ToDomainPriority() domain.Priority {
	return domain.Priority(c.Priority)
}

// StartPickingCommand represents the command to mark an order as picking in progress
type StartPickingCommand struct {
	OrderID string
}

// MarkConsolidatedCommand represents the command to mark an order as consolidated
type MarkConsolidatedCommand struct {
	OrderID string
}

// MarkPackedCommand represents the command to mark an order as packed
type MarkPackedCommand struct {
	OrderID string
}
