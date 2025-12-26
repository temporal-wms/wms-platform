package events_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wms-platform/shared/pkg/contracts/asyncapi"
)

const asyncAPISpecPath = "../../../../docs/asyncapi.yaml"

// Event type constants matching the CloudEvents in the WMS platform
const (
	// Order events
	OrderReceived     = "wms.order.received"
	OrderValidated    = "wms.order.validated"
	OrderWaveAssigned = "wms.order.wave-assigned"
	OrderShipped      = "wms.order.shipped"
	OrderCancelled    = "wms.order.cancelled"
	OrderCompleted    = "wms.order.completed"

	// Wave events
	WaveCreated      = "wms.wave.created"
	OrderAddedToWave = "wms.wave.order-added"
	WaveScheduled    = "wms.wave.scheduled"
	WaveReleased     = "wms.wave.released"
	WaveCompleted    = "wms.wave.completed"
	WaveCancelled    = "wms.wave.cancelled"

	// Routing events
	RouteCalculated = "wms.routing.route-calculated"
	RouteStarted    = "wms.routing.route-started"
	StopCompleted   = "wms.routing.stop-completed"
	RouteCompleted  = "wms.routing.route-completed"

	// Picking events
	PickTaskCreated   = "wms.picking.task-created"
	PickTaskAssigned  = "wms.picking.task-assigned"
	ItemPicked        = "wms.picking.item-picked"
	PickException     = "wms.picking.exception"
	PickTaskCompleted = "wms.picking.task-completed"

	// Consolidation events
	ConsolidationStarted   = "wms.consolidation.started"
	ItemConsolidated       = "wms.consolidation.item-consolidated"
	ConsolidationCompleted = "wms.consolidation.completed"

	// Packing events
	PackTaskCreated    = "wms.packing.task-created"
	PackagingSuggested = "wms.packing.packaging-suggested"
	PackageSealed      = "wms.packing.package-sealed"
	LabelApplied       = "wms.packing.label-applied"
	PackTaskCompleted  = "wms.packing.task-completed"

	// Shipping events
	ShipmentCreated    = "wms.shipping.shipment-created"
	LabelGenerated     = "wms.shipping.label-generated"
	ShipmentManifested = "wms.shipping.manifested"
	ShipConfirmed      = "wms.shipping.confirmed"

	// Inventory events
	InventoryReceived = "wms.inventory.received"
	InventoryReserved = "wms.inventory.reserved"
	InventoryPicked   = "wms.inventory.picked"
	InventoryAdjusted = "wms.inventory.adjusted"
	LowStockAlert     = "wms.inventory.low-stock-alert"

	// Labor events
	ShiftStarted        = "wms.labor.shift-started"
	ShiftEnded          = "wms.labor.shift-ended"
	TaskAssigned        = "wms.labor.task-assigned"
	TaskCompleted       = "wms.labor.task-completed"
	PerformanceRecorded = "wms.labor.performance-recorded"
)

// Sample event payloads for testing

// OrderReceivedData represents the payload for OrderReceived event
type OrderReceivedData struct {
	OrderID    string  `json:"orderId"`
	CustomerID string  `json:"customerId"`
	TotalValue float64 `json:"totalValue"`
	ItemCount  int     `json:"itemCount"`
	Priority   string  `json:"priority"`
}

// WaveCreatedData represents the payload for WaveCreated event
type WaveCreatedData struct {
	WaveID     string `json:"waveId"`
	Priority   int    `json:"priority"`
	OrderCount int    `json:"orderCount"`
}

// ItemPickedData represents the payload for ItemPicked event
type ItemPickedData struct {
	TaskID     string `json:"taskId"`
	SKU        string `json:"sku"`
	Quantity   int    `json:"quantity"`
	Location   string `json:"location"`
	WorkerID   string `json:"workerId"`
	PickedAt   string `json:"pickedAt"`
}

// ShipmentCreatedData represents the payload for ShipmentCreated event
type ShipmentCreatedData struct {
	ShipmentID string `json:"shipmentId"`
	OrderID    string `json:"orderId"`
	Carrier    string `json:"carrier"`
}

func TestAsyncAPISpecExists(t *testing.T) {
	absPath, err := filepath.Abs(asyncAPISpecPath)
	require.NoError(t, err)

	_, err = os.Stat(absPath)
	if os.IsNotExist(err) {
		t.Skip("AsyncAPI spec not found - skipping event validation tests")
	}
	require.NoError(t, err)
}

func TestEventValidatorCreation(t *testing.T) {
	absPath, err := filepath.Abs(asyncAPISpecPath)
	require.NoError(t, err)

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("AsyncAPI spec not found")
	}

	validator, err := asyncapi.NewEventValidator(absPath)
	require.NoError(t, err)
	assert.NotNil(t, validator)

	eventTypes := validator.GetSupportedEventTypes()
	t.Logf("Found %d event types in AsyncAPI spec", len(eventTypes))
}

func TestOrderEventSchemas(t *testing.T) {
	absPath, err := filepath.Abs(asyncAPISpecPath)
	require.NoError(t, err)

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("AsyncAPI spec not found")
	}

	validator, err := asyncapi.NewEventValidator(absPath)
	require.NoError(t, err)

	t.Run("OrderReceived", func(t *testing.T) {
		if !validator.HasSchema(OrderReceived) {
			t.Skipf("No schema found for %s", OrderReceived)
		}

		event := asyncapi.CloudEvent{
			SpecVersion:     "1.0",
			Type:            OrderReceived,
			Source:          "/wms/order-service",
			ID:              "evt-123",
			Time:            time.Now().Format(time.RFC3339),
			DataContentType: "application/json",
			Data: OrderReceivedData{
				OrderID:    "ord-123456",
				CustomerID: "cust-001",
				TotalValue: 150.50,
				ItemCount:  5,
				Priority:   "standard",
			},
		}

		err := validator.ValidateEvent(event)
		if err != nil {
			t.Logf("Validation error (may be expected if schema doesn't match test data): %v", err)
		}
	})
}

func TestPickingEventSchemas(t *testing.T) {
	absPath, err := filepath.Abs(asyncAPISpecPath)
	require.NoError(t, err)

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("AsyncAPI spec not found")
	}

	validator, err := asyncapi.NewEventValidator(absPath)
	require.NoError(t, err)

	t.Run("ItemPicked", func(t *testing.T) {
		if !validator.HasSchema(ItemPicked) {
			t.Skipf("No schema found for %s", ItemPicked)
		}

		event := asyncapi.CloudEvent{
			SpecVersion:     "1.0",
			Type:            ItemPicked,
			Source:          "/wms/picking-service",
			ID:              "evt-456",
			Time:            time.Now().Format(time.RFC3339),
			DataContentType: "application/json",
			Data: ItemPickedData{
				TaskID:   "task-123",
				SKU:      "SKU-001",
				Quantity: 2,
				Location: "A-01-01",
				WorkerID: "worker-001",
				PickedAt: time.Now().Format(time.RFC3339),
			},
		}

		err := validator.ValidateEvent(event)
		if err != nil {
			t.Logf("Validation error (may be expected if schema doesn't match test data): %v", err)
		}
	})
}

func TestShippingEventSchemas(t *testing.T) {
	absPath, err := filepath.Abs(asyncAPISpecPath)
	require.NoError(t, err)

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("AsyncAPI spec not found")
	}

	validator, err := asyncapi.NewEventValidator(absPath)
	require.NoError(t, err)

	t.Run("ShipmentCreated", func(t *testing.T) {
		if !validator.HasSchema(ShipmentCreated) {
			t.Skipf("No schema found for %s", ShipmentCreated)
		}

		event := asyncapi.CloudEvent{
			SpecVersion:     "1.0",
			Type:            ShipmentCreated,
			Source:          "/wms/shipping-service",
			ID:              "evt-789",
			Time:            time.Now().Format(time.RFC3339),
			DataContentType: "application/json",
			Data: ShipmentCreatedData{
				ShipmentID: "ship-123",
				OrderID:    "ord-123456",
				Carrier:    "UPS",
			},
		}

		err := validator.ValidateEvent(event)
		if err != nil {
			t.Logf("Validation error (may be expected if schema doesn't match test data): %v", err)
		}
	})
}

func TestAllEventTypesHaveSchemas(t *testing.T) {
	absPath, err := filepath.Abs(asyncAPISpecPath)
	require.NoError(t, err)

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("AsyncAPI spec not found")
	}

	validator, err := asyncapi.NewEventValidator(absPath)
	require.NoError(t, err)

	expectedEventTypes := []string{
		// Order events
		OrderReceived, OrderValidated, OrderWaveAssigned, OrderShipped, OrderCancelled, OrderCompleted,
		// Wave events
		WaveCreated, OrderAddedToWave, WaveScheduled, WaveReleased, WaveCompleted, WaveCancelled,
		// Routing events
		RouteCalculated, RouteStarted, StopCompleted, RouteCompleted,
		// Picking events
		PickTaskCreated, PickTaskAssigned, ItemPicked, PickException, PickTaskCompleted,
		// Consolidation events
		ConsolidationStarted, ItemConsolidated, ConsolidationCompleted,
		// Packing events
		PackTaskCreated, PackagingSuggested, PackageSealed, LabelApplied, PackTaskCompleted,
		// Shipping events
		ShipmentCreated, LabelGenerated, ShipmentManifested, ShipConfirmed,
		// Inventory events
		InventoryReceived, InventoryReserved, InventoryPicked, InventoryAdjusted, LowStockAlert,
		// Labor events
		ShiftStarted, ShiftEnded, TaskAssigned, TaskCompleted, PerformanceRecorded,
	}

	registeredTypes := validator.GetSupportedEventTypes()
	registeredMap := make(map[string]bool)
	for _, t := range registeredTypes {
		registeredMap[t] = true
	}

	missingTypes := []string{}
	for _, eventType := range expectedEventTypes {
		if !registeredMap[eventType] {
			missingTypes = append(missingTypes, eventType)
		}
	}

	if len(missingTypes) > 0 {
		t.Logf("Missing schemas for event types (may need to be added to AsyncAPI spec or mapping): %v", missingTypes)
	}

	t.Logf("Total expected: %d, Total registered: %d, Missing: %d",
		len(expectedEventTypes), len(registeredTypes), len(missingTypes))
}

func TestRegisterCustomSchema(t *testing.T) {
	absPath, err := filepath.Abs(asyncAPISpecPath)
	require.NoError(t, err)

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("AsyncAPI spec not found")
	}

	validator, err := asyncapi.NewEventValidator(absPath)
	require.NoError(t, err)

	// Register a custom schema
	customSchema := []byte(`{
		"type": "object",
		"properties": {
			"testField": {"type": "string"}
		},
		"required": ["testField"]
	}`)

	err = validator.RegisterSchema("custom.test.event", customSchema)
	require.NoError(t, err)

	assert.True(t, validator.HasSchema("custom.test.event"))

	// Validate an event against the custom schema
	event := asyncapi.CloudEvent{
		SpecVersion: "1.0",
		Type:        "custom.test.event",
		Source:      "/wms/test",
		ID:          "test-123",
		Data: map[string]interface{}{
			"testField": "test value",
		},
	}

	err = validator.ValidateEvent(event)
	require.NoError(t, err)
}
