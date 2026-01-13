package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDomainEventTypes(t *testing.T) {
	startedAt := time.Now()
	started := &ConsolidationStartedEvent{
		ConsolidationID: "CONS-100",
		OrderID:         "ORD-100",
		ExpectedItems:   3,
		SourceTotes:     []string{"TOTE-1"},
		StartedAt:       startedAt,
	}
	assert.Equal(t, "wms.consolidation.started", started.EventType())
	assert.Equal(t, startedAt, started.OccurredAt())

	consolidatedAt := time.Now().Add(time.Second)
	consolidated := &ItemConsolidatedEvent{
		ConsolidationID: "CONS-100",
		SKU:             "SKU-100",
		Quantity:        1,
		SourceToteID:    "TOTE-1",
		DestinationBin:  "BIN-1",
		ConsolidatedAt:  consolidatedAt,
	}
	assert.Equal(t, "wms.consolidation.item-consolidated", consolidated.EventType())
	assert.Equal(t, consolidatedAt, consolidated.OccurredAt())

	completedAt := time.Now().Add(2 * time.Second)
	completed := &ConsolidationCompletedEvent{
		ConsolidationID:   "CONS-100",
		OrderID:           "ORD-100",
		DestinationBin:    "BIN-1",
		TotalConsolidated: 3,
		ReadyForPacking:   true,
		CompletedAt:       completedAt,
	}
	assert.Equal(t, "wms.consolidation.completed", completed.EventType())
	assert.Equal(t, completedAt, completed.OccurredAt())

	receivedAt := time.Now().Add(3 * time.Second)
	received := &ToteReceivedEvent{
		ConsolidationID: "CONS-100",
		ToteID:          "TOTE-1",
		RouteID:         "ROUTE-1",
		ReceivedAt:      receivedAt,
	}
	assert.Equal(t, "wms.consolidation.tote-received", received.EventType())
	assert.Equal(t, receivedAt, received.OccurredAt())
}
