package dto

import (
	"time"

	"github.com/wms-platform/services/order-service/internal/domain"
)

// CreateOrderRequest represents the request to create an order
type CreateOrderRequest struct {
	CustomerID         string    `json:"customerId" binding:"required" example:"CUST-123"`
	Items              []OrderItemRequest `json:"items" binding:"required,min=1,dive"`
	ShippingAddress    AddressRequest `json:"shippingAddress" binding:"required"`
	Priority           string    `json:"priority" binding:"required,oneof=same_day next_day standard" example:"same_day"`
	PromisedDeliveryAt time.Time `json:"promisedDeliveryAt" binding:"required" example:"2025-12-25T15:00:00Z"`
}

// OrderItemRequest represents an order item in the request
type OrderItemRequest struct {
	SKU      string  `json:"sku" binding:"required" example:"SKU-12345"`
	Quantity int     `json:"quantity" binding:"required,min=1" example:"2"`
	Weight   float64 `json:"weight" binding:"required,min=0" example:"1.5"`
}

// AddressRequest represents a shipping address in the request
type AddressRequest struct {
	Street     string `json:"street" binding:"required" example:"123 Main St"`
	City       string `json:"city" binding:"required" example:"San Francisco"`
	State      string `json:"state" binding:"required,len=2" example:"CA"`
	PostalCode string `json:"postalCode" binding:"required" example:"94105"`
	Country    string `json:"country" binding:"required,len=2" example:"US"`
}

// CancelOrderRequest represents the request to cancel an order
type CancelOrderRequest struct {
	Reason string `json:"reason" binding:"required,min=3,max=500" example:"Customer requested cancellation"`
}

// OrderResponse represents an order in the response
type OrderResponse struct {
	OrderID            string              `json:"orderId" example:"ORD-a1b2c3d4"`
	CustomerID         string              `json:"customerId" example:"CUST-123"`
	Items              []OrderItemResponse `json:"items"`
	ShippingAddress    AddressResponse     `json:"shippingAddress"`
	Priority           string              `json:"priority" example:"same_day"`
	Status             string              `json:"status" example:"received"`
	PromisedDeliveryAt time.Time           `json:"promisedDeliveryAt" example:"2025-12-25T15:00:00Z"`
	CreatedAt          time.Time           `json:"createdAt" example:"2025-12-23T10:00:00Z"`
	UpdatedAt          time.Time           `json:"updatedAt" example:"2025-12-23T10:00:00Z"`
}

// OrderItemResponse represents an order item in the response
type OrderItemResponse struct {
	SKU      string  `json:"sku" example:"SKU-12345"`
	Quantity int     `json:"quantity" example:"2"`
	Weight   float64 `json:"weight" example:"1.5"`
}

// AddressResponse represents a shipping address in the response
type AddressResponse struct {
	Street     string `json:"street" example:"123 Main St"`
	City       string `json:"city" example:"San Francisco"`
	State      string `json:"state" example:"CA"`
	PostalCode string `json:"postalCode" example:"94105"`
	Country    string `json:"country" example:"US"`
}

// OrderListResponse represents a paginated list of orders
type OrderListResponse struct {
	Data       []OrderResponse `json:"data"`
	Page       int64          `json:"page" example:"1"`
	PageSize   int64          `json:"pageSize" example:"20"`
	TotalItems int64          `json:"totalItems" example:"100"`
	TotalPages int64          `json:"totalPages" example:"5"`
	HasNext    bool           `json:"hasNext" example:"true"`
	HasPrev    bool           `json:"hasPrev" example:"false"`
}

// ToOrderResponse converts a domain Order to OrderResponse DTO
func ToOrderResponse(order *domain.Order) OrderResponse {
	items := make([]OrderItemResponse, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, OrderItemResponse{
			SKU:      item.SKU,
			Quantity: item.Quantity,
			Weight:   item.Weight,
		})
	}

	return OrderResponse{
		OrderID:    order.OrderID,
		CustomerID: order.CustomerID,
		Items:      items,
		ShippingAddress: AddressResponse{
			Street:     order.ShippingAddress.Street,
			City:       order.ShippingAddress.City,
			State:      order.ShippingAddress.State,
			PostalCode: order.ShippingAddress.ZipCode,
			Country:    order.ShippingAddress.Country,
		},
		Priority:           string(order.Priority),
		Status:             string(order.Status),
		PromisedDeliveryAt: order.PromisedDeliveryAt,
		CreatedAt:          order.CreatedAt,
		UpdatedAt:          order.UpdatedAt,
	}
}

// ToOrderListResponse converts a list of domain Orders to OrderListResponse DTO
func ToOrderListResponse(orders []*domain.Order, page, pageSize, totalItems int64) OrderListResponse {
	data := make([]OrderResponse, 0, len(orders))
	for _, order := range orders {
		data = append(data, ToOrderResponse(order))
	}

	totalPages := (totalItems + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	return OrderListResponse{
		Data:       data,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// ToDomainOrderItems converts OrderItemRequest to domain OrderItems
func (r *CreateOrderRequest) ToDomainOrderItems() []domain.OrderItem {
	items := make([]domain.OrderItem, 0, len(r.Items))
	for _, item := range r.Items {
		items = append(items, domain.OrderItem{
			SKU:      item.SKU,
			Quantity: item.Quantity,
			Weight:   item.Weight,
		})
	}
	return items
}

// ToDomainAddress converts AddressRequest to domain Address
func (r *CreateOrderRequest) ToDomainAddress() domain.Address {
	return domain.Address{
		Street:  r.ShippingAddress.Street,
		City:    r.ShippingAddress.City,
		State:   r.ShippingAddress.State,
		ZipCode: r.ShippingAddress.PostalCode,
		Country: r.ShippingAddress.Country,
	}
}

// ToDomainPriority converts string priority to domain Priority
func (r *CreateOrderRequest) ToDomainPriority() domain.Priority {
	return domain.Priority(r.Priority)
}
