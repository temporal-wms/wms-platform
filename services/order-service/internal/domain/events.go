package domain

import (
	"time"

	"github.com/google/uuid"
)

// DomainEvent represents a domain event
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
	AggregateID() string
}

// BaseDomainEvent contains common event fields
type BaseDomainEvent struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	AggregateId string    `json:"aggregateId"`
	Timestamp   time.Time `json:"timestamp"`
}

func (e BaseDomainEvent) EventType() string    { return e.Type }
func (e BaseDomainEvent) OccurredAt() time.Time { return e.Timestamp }
func (e BaseDomainEvent) AggregateID() string   { return e.AggregateId }

// OrderReceivedEvent is raised when a new order is created
type OrderReceivedEvent struct {
	BaseDomainEvent
	OrderID            string      `json:"orderId"`
	CustomerID         string      `json:"customerId"`
	Items              []OrderItem `json:"items"`
	Priority           Priority    `json:"priority"`
	PromisedDeliveryAt time.Time   `json:"promisedDeliveryAt"`
}

// NewOrderReceivedEvent creates a new OrderReceivedEvent
func NewOrderReceivedEvent(order *Order) *OrderReceivedEvent {
	return &OrderReceivedEvent{
		BaseDomainEvent: BaseDomainEvent{
			ID:          uuid.New().String(),
			Type:        "wms.order.received",
			AggregateId: order.OrderID,
			Timestamp:   time.Now().UTC(),
		},
		OrderID:            order.OrderID,
		CustomerID:         order.CustomerID,
		Items:              order.Items,
		Priority:           order.Priority,
		PromisedDeliveryAt: order.PromisedDeliveryAt,
	}
}

// OrderValidatedEvent is raised when an order is validated
type OrderValidatedEvent struct {
	BaseDomainEvent
	OrderID    string `json:"orderId"`
	CustomerID string `json:"customerId"`
	TotalItems int    `json:"totalItems"`
}

// NewOrderValidatedEvent creates a new OrderValidatedEvent
func NewOrderValidatedEvent(order *Order) *OrderValidatedEvent {
	return &OrderValidatedEvent{
		BaseDomainEvent: BaseDomainEvent{
			ID:          uuid.New().String(),
			Type:        "wms.order.validated",
			AggregateId: order.OrderID,
			Timestamp:   time.Now().UTC(),
		},
		OrderID:    order.OrderID,
		CustomerID: order.CustomerID,
		TotalItems: order.TotalItems(),
	}
}

// OrderAssignedToWaveEvent is raised when an order is assigned to a wave
type OrderAssignedToWaveEvent struct {
	BaseDomainEvent
	OrderID string `json:"orderId"`
	WaveID  string `json:"waveId"`
}

// NewOrderAssignedToWaveEvent creates a new OrderAssignedToWaveEvent
func NewOrderAssignedToWaveEvent(order *Order, waveID string) *OrderAssignedToWaveEvent {
	return &OrderAssignedToWaveEvent{
		BaseDomainEvent: BaseDomainEvent{
			ID:          uuid.New().String(),
			Type:        "wms.order.wave-assigned",
			AggregateId: order.OrderID,
			Timestamp:   time.Now().UTC(),
		},
		OrderID: order.OrderID,
		WaveID:  waveID,
	}
}

// OrderShippedEvent is raised when an order is shipped
type OrderShippedEvent struct {
	BaseDomainEvent
	OrderID        string `json:"orderId"`
	CustomerID     string `json:"customerId"`
	TrackingNumber string `json:"trackingNumber"`
}

// NewOrderShippedEvent creates a new OrderShippedEvent
func NewOrderShippedEvent(order *Order) *OrderShippedEvent {
	return &OrderShippedEvent{
		BaseDomainEvent: BaseDomainEvent{
			ID:          uuid.New().String(),
			Type:        "wms.order.shipped",
			AggregateId: order.OrderID,
			Timestamp:   time.Now().UTC(),
		},
		OrderID:        order.OrderID,
		CustomerID:     order.CustomerID,
		TrackingNumber: order.TrackingNumber,
	}
}

// OrderCancelledEvent is raised when an order is cancelled
type OrderCancelledEvent struct {
	BaseDomainEvent
	OrderID    string `json:"orderId"`
	CustomerID string `json:"customerId"`
	Reason     string `json:"reason"`
}

// NewOrderCancelledEvent creates a new OrderCancelledEvent
func NewOrderCancelledEvent(order *Order, reason string) *OrderCancelledEvent {
	return &OrderCancelledEvent{
		BaseDomainEvent: BaseDomainEvent{
			ID:          uuid.New().String(),
			Type:        "wms.order.cancelled",
			AggregateId: order.OrderID,
			Timestamp:   time.Now().UTC(),
		},
		OrderID:    order.OrderID,
		CustomerID: order.CustomerID,
		Reason:     reason,
	}
}

// OrderCompletedEvent is raised when an order is completed (delivered)
type OrderCompletedEvent struct {
	BaseDomainEvent
	OrderID    string    `json:"orderId"`
	CustomerID string    `json:"customerId"`
	DeliveredAt time.Time `json:"deliveredAt"`
}

// NewOrderCompletedEvent creates a new OrderCompletedEvent
func NewOrderCompletedEvent(order *Order) *OrderCompletedEvent {
	return &OrderCompletedEvent{
		BaseDomainEvent: BaseDomainEvent{
			ID:          uuid.New().String(),
			Type:        "wms.order.completed",
			AggregateId: order.OrderID,
			Timestamp:   time.Now().UTC(),
		},
		OrderID:     order.OrderID,
		CustomerID:  order.CustomerID,
		DeliveredAt: time.Now().UTC(),
	}
}
