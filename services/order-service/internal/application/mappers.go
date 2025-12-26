package application

import (
	"time"

	"github.com/wms-platform/services/order-service/internal/domain"
)

// ToOrderDTO converts a domain Order to OrderDTO
func ToOrderDTO(order *domain.Order) *OrderDTO {
	if order == nil {
		return nil
	}

	items := make([]OrderItemDTO, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, OrderItemDTO{
			SKU:      item.SKU,
			Quantity: item.Quantity,
			Weight:   item.Weight,
		})
	}

	return &OrderDTO{
		OrderID:   order.OrderID,
		CustomerID: order.CustomerID,
		Items:     items,
		ShippingAddress: AddressDTO{
			Street:  order.ShippingAddress.Street,
			City:    order.ShippingAddress.City,
			State:   order.ShippingAddress.State,
			ZipCode: order.ShippingAddress.ZipCode,
			Country: order.ShippingAddress.Country,
		},
		Priority:           string(order.Priority),
		Status:             string(order.Status),
		PromisedDeliveryAt: order.PromisedDeliveryAt,
		WaveID:             order.WaveID,
		TrackingNumber:     order.TrackingNumber,
		TotalItems:         order.TotalItems(),
		TotalWeight:        order.TotalWeight(),
		IsMultiItem:        order.IsMultiItem(),
		CreatedAt:          order.CreatedAt,
		UpdatedAt:          order.UpdatedAt,
	}
}

// ToOrderListDTO converts a domain Order to OrderListDTO (simplified)
func ToOrderListDTO(order *domain.Order) *OrderListDTO {
	if order == nil {
		return nil
	}

	// Calculate derived fields for CQRS
	totalValue := 0.0
	for _, item := range order.Items {
		totalValue += item.UnitPrice * float64(item.Quantity)
	}

	daysUntilPromised := int(time.Until(order.PromisedDeliveryAt).Hours() / 24)
	isLate := time.Now().After(order.PromisedDeliveryAt) && order.Status != "delivered" && order.Status != "cancelled"
	isPriority := order.Priority == "same_day" || order.Priority == "next_day"

	return &OrderListDTO{
		OrderID:            order.OrderID,
		CustomerID:         order.CustomerID,
		Status:             string(order.Status),
		Priority:           string(order.Priority),
		TotalItems:         order.TotalItems(),
		TotalWeight:        order.TotalWeight(),
		TotalValue:         totalValue,
		WaveID:             order.WaveID,
		TrackingNumber:     order.TrackingNumber,
		ShipToCity:         order.ShippingAddress.City,
		ShipToState:        order.ShippingAddress.State,
		ShipToZipCode:      order.ShippingAddress.ZipCode,
		DaysUntilPromised:  daysUntilPromised,
		IsLate:             isLate,
		IsPriority:         isPriority,
		ReceivedAt:         order.CreatedAt.Format("2006-01-02T15:04:05Z"),
		PromisedDeliveryAt: order.PromisedDeliveryAt.Format("2006-01-02T15:04:05Z"),
		CreatedAt:          order.CreatedAt,
		UpdatedAt:          order.UpdatedAt,
	}
}

// ToOrderListDTOs converts a slice of domain Orders to OrderListDTOs
func ToOrderListDTOs(orders []*domain.Order) []OrderListDTO {
	dtos := make([]OrderListDTO, 0, len(orders))
	for _, order := range orders {
		if dto := ToOrderListDTO(order); dto != nil {
			dtos = append(dtos, *dto)
		}
	}
	return dtos
}

// ToDomainPagination converts query pagination to domain Pagination
func (q *ListOrdersQuery) ToDomainPagination() domain.Pagination {
	return domain.Pagination{
		Page:     q.Page,
		PageSize: q.PageSize,
	}
}

// ToDomainFilter converts query filters to domain OrderFilter
func (q *ListOrdersQuery) ToDomainFilter() domain.OrderFilter {
	filter := domain.OrderFilter{}

	if q.CustomerID != nil {
		filter.CustomerID = q.CustomerID
	}

	if q.Status != nil {
		status := domain.Status(*q.Status)
		filter.Status = &status
	}

	if q.Priority != nil {
		priority := domain.Priority(*q.Priority)
		filter.Priority = &priority
	}

	return filter
}
