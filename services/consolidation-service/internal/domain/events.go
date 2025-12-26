package domain

import "time"

// DomainEvent is the interface for all domain events
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// ConsolidationStartedEvent is published when consolidation starts
type ConsolidationStartedEvent struct {
	ConsolidationID string    `json:"consolidationId"`
	OrderID         string    `json:"orderId"`
	ExpectedItems   int       `json:"expectedItems"`
	SourceTotes     []string  `json:"sourceTotes"`
	StartedAt       time.Time `json:"startedAt"`
}

func (e *ConsolidationStartedEvent) EventType() string    { return "wms.consolidation.started" }
func (e *ConsolidationStartedEvent) OccurredAt() time.Time { return e.StartedAt }

// ItemConsolidatedEvent is published when an item is consolidated
type ItemConsolidatedEvent struct {
	ConsolidationID string    `json:"consolidationId"`
	SKU             string    `json:"sku"`
	Quantity        int       `json:"quantity"`
	SourceToteID    string    `json:"sourceToteId"`
	DestinationBin  string    `json:"destinationBin"`
	ConsolidatedAt  time.Time `json:"consolidatedAt"`
}

func (e *ItemConsolidatedEvent) EventType() string    { return "wms.consolidation.item-consolidated" }
func (e *ItemConsolidatedEvent) OccurredAt() time.Time { return e.ConsolidatedAt }

// ConsolidationCompletedEvent is published when consolidation is complete
type ConsolidationCompletedEvent struct {
	ConsolidationID   string    `json:"consolidationId"`
	OrderID           string    `json:"orderId"`
	DestinationBin    string    `json:"destinationBin"`
	TotalConsolidated int       `json:"totalConsolidated"`
	ReadyForPacking   bool      `json:"readyForPacking"`
	CompletedAt       time.Time `json:"completedAt"`
}

func (e *ConsolidationCompletedEvent) EventType() string    { return "wms.consolidation.completed" }
func (e *ConsolidationCompletedEvent) OccurredAt() time.Time { return e.CompletedAt }
