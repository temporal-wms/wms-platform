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

// OrderRetryScheduledEvent is raised when an order is scheduled for retry after a transient failure
type OrderRetryScheduledEvent struct {
	BaseDomainEvent
	OrderID       string `json:"orderId"`
	CustomerID    string `json:"customerId"`
	RetryNumber   int    `json:"retryNumber"`
	FailureStatus string `json:"failureStatus"`
	FailureReason string `json:"failureReason"`
}

// NewOrderRetryScheduledEvent creates a new OrderRetryScheduledEvent
func NewOrderRetryScheduledEvent(order *Order, retryNumber int, failureStatus string, failureReason string) *OrderRetryScheduledEvent {
	return &OrderRetryScheduledEvent{
		BaseDomainEvent: BaseDomainEvent{
			ID:          uuid.New().String(),
			Type:        "wms.order.retry-scheduled",
			AggregateId: order.OrderID,
			Timestamp:   time.Now().UTC(),
		},
		OrderID:       order.OrderID,
		CustomerID:    order.CustomerID,
		RetryNumber:   retryNumber,
		FailureStatus: failureStatus,
		FailureReason: failureReason,
	}
}

// OrderMovedToDLQEvent is raised when an order is moved to the dead letter queue
type OrderMovedToDLQEvent struct {
	BaseDomainEvent
	OrderID            string `json:"orderId"`
	CustomerID         string `json:"customerId"`
	FinalFailureStatus string `json:"finalFailureStatus"`
	FinalFailureReason string `json:"finalFailureReason"`
	TotalRetryAttempts int    `json:"totalRetryAttempts"`
}

// NewOrderMovedToDLQEvent creates a new OrderMovedToDLQEvent
func NewOrderMovedToDLQEvent(order *Order, failureStatus string, failureReason string, totalRetries int) *OrderMovedToDLQEvent {
	return &OrderMovedToDLQEvent{
		BaseDomainEvent: BaseDomainEvent{
			ID:          uuid.New().String(),
			Type:        "wms.order.moved-to-dlq",
			AggregateId: order.OrderID,
			Timestamp:   time.Now().UTC(),
		},
		OrderID:            order.OrderID,
		CustomerID:         order.CustomerID,
		FinalFailureStatus: failureStatus,
		FinalFailureReason: failureReason,
		TotalRetryAttempts: totalRetries,
	}
}

// OrderPartiallyFulfilledEvent is raised when an order is partially fulfilled due to stock shortage
type OrderPartiallyFulfilledEvent struct {
	BaseDomainEvent
	OrderID          string           `json:"orderId"`
	CustomerID       string           `json:"customerId"`
	FulfilledItems   []PartialItem    `json:"fulfilledItems"`
	BackorderedItems []BackorderedItem `json:"backorderedItems"`
	FulfillmentRatio float64          `json:"fulfillmentRatio"` // Percentage of order fulfilled (0.0-1.0)
}

// PartialItem represents an item that was fulfilled
type PartialItem struct {
	SKU              string `json:"sku"`
	QuantityOrdered  int    `json:"quantityOrdered"`
	QuantityFulfilled int   `json:"quantityFulfilled"`
}

// BackorderedItem represents an item that was backordered
type BackorderedItem struct {
	SKU              string `json:"sku"`
	QuantityOrdered  int    `json:"quantityOrdered"`
	QuantityShort    int    `json:"quantityShort"`
	Reason           string `json:"reason"` // not_found, damaged, quantity_mismatch
}

// NewOrderPartiallyFulfilledEvent creates a new OrderPartiallyFulfilledEvent
func NewOrderPartiallyFulfilledEvent(order *Order, fulfilledItems []PartialItem, backorderedItems []BackorderedItem) *OrderPartiallyFulfilledEvent {
	totalOrdered := 0
	totalFulfilled := 0
	for _, item := range fulfilledItems {
		totalOrdered += item.QuantityOrdered
		totalFulfilled += item.QuantityFulfilled
	}
	for _, item := range backorderedItems {
		totalOrdered += item.QuantityOrdered
	}

	ratio := 0.0
	if totalOrdered > 0 {
		ratio = float64(totalFulfilled) / float64(totalOrdered)
	}

	return &OrderPartiallyFulfilledEvent{
		BaseDomainEvent: BaseDomainEvent{
			ID:          uuid.New().String(),
			Type:        "wms.order.partially-fulfilled",
			AggregateId: order.OrderID,
			Timestamp:   time.Now().UTC(),
		},
		OrderID:          order.OrderID,
		CustomerID:       order.CustomerID,
		FulfilledItems:   fulfilledItems,
		BackorderedItems: backorderedItems,
		FulfillmentRatio: ratio,
	}
}

// BackorderCreatedEvent is raised when a backorder is created for short items
type BackorderCreatedEvent struct {
	BaseDomainEvent
	BackorderID     string            `json:"backorderId"`
	OriginalOrderID string            `json:"originalOrderId"`
	CustomerID      string            `json:"customerId"`
	Items           []BackorderedItem `json:"items"`
	Priority        Priority          `json:"priority"` // Inherits from original order
}

// NewBackorderCreatedEvent creates a new BackorderCreatedEvent
func NewBackorderCreatedEvent(order *Order, backorderID string, items []BackorderedItem) *BackorderCreatedEvent {
	return &BackorderCreatedEvent{
		BaseDomainEvent: BaseDomainEvent{
			ID:          uuid.New().String(),
			Type:        "wms.order.backorder-created",
			AggregateId: backorderID,
			Timestamp:   time.Now().UTC(),
		},
		BackorderID:     backorderID,
		OriginalOrderID: order.OrderID,
		CustomerID:      order.CustomerID,
		Items:           items,
		Priority:        order.Priority,
	}
}

// BackorderFulfilledEvent is raised when a backorder is fulfilled
type BackorderFulfilledEvent struct {
	BaseDomainEvent
	BackorderID     string `json:"backorderId"`
	OriginalOrderID string `json:"originalOrderId"`
	CustomerID      string `json:"customerId"`
}

// NewBackorderFulfilledEvent creates a new BackorderFulfilledEvent
func NewBackorderFulfilledEvent(backorderID, originalOrderID, customerID string) *BackorderFulfilledEvent {
	return &BackorderFulfilledEvent{
		BaseDomainEvent: BaseDomainEvent{
			ID:          uuid.New().String(),
			Type:        "wms.order.backorder-fulfilled",
			AggregateId: backorderID,
			Timestamp:   time.Now().UTC(),
		},
		BackorderID:     backorderID,
		OriginalOrderID: originalOrderID,
		CustomerID:      customerID,
	}
}
