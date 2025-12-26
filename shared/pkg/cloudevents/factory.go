package cloudevents

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// EventFactory creates CloudEvents for WMS domain events
type EventFactory struct {
	source string
}

// NewEventFactory creates a new EventFactory for a specific source
func NewEventFactory(source string) *EventFactory {
	return &EventFactory{source: source}
}

// CreateEvent creates a new WMSCloudEvent with the given parameters
func (f *EventFactory) CreateEvent(
	ctx context.Context,
	eventType string,
	subject string,
	data interface{},
) *WMSCloudEvent {
	event := &WMSCloudEvent{
		SpecVersion:     "1.0",
		Type:            eventType,
		Source:          f.source,
		Subject:         subject,
		ID:              uuid.New().String(),
		Time:            time.Now().UTC(),
		DataContentType: "application/json",
		Data:            data,
		Extensions:      make(map[string]interface{}),
	}

	return event
}

// CreateEventWithCorrelation creates an event with correlation tracking
func (f *EventFactory) CreateEventWithCorrelation(
	ctx context.Context,
	eventType string,
	subject string,
	data interface{},
	correlationID string,
	workflowID string,
) *WMSCloudEvent {
	event := f.CreateEvent(ctx, eventType, subject, data)
	event.CorrelationID = correlationID
	event.WorkflowID = workflowID
	return event
}

// CreateOrderReceivedEvent creates an OrderReceived event
func (f *EventFactory) CreateOrderReceivedEvent(
	ctx context.Context,
	orderID string,
	customerID string,
	orderLines []OrderLine,
	priority string,
	promisedDeliveryAt time.Time,
) *WMSCloudEvent {
	data := OrderReceivedData{
		OrderID:            orderID,
		CustomerID:         customerID,
		OrderLines:         orderLines,
		Priority:           priority,
		PromisedDeliveryAt: promisedDeliveryAt,
	}
	return f.CreateEvent(ctx, OrderReceived, "order/"+orderID, data)
}

// CreateWaveCreatedEvent creates a WaveCreated event
func (f *EventFactory) CreateWaveCreatedEvent(
	ctx context.Context,
	waveID string,
	orderIDs []string,
	scheduledStart time.Time,
	estimatedDuration string,
	waveType string,
) *WMSCloudEvent {
	data := WaveCreatedData{
		WaveID:            waveID,
		OrderIDs:          orderIDs,
		ScheduledStart:    scheduledStart,
		EstimatedDuration: estimatedDuration,
		WaveType:          waveType,
	}
	event := f.CreateEvent(ctx, WaveCreated, "wave/"+waveID, data)
	event.WaveNumber = waveID
	return event
}

// CreateRouteCalculatedEvent creates a RouteCalculated event
func (f *EventFactory) CreateRouteCalculatedEvent(
	ctx context.Context,
	routeID string,
	pickerID string,
	stops []LocationStop,
	estimatedDistance float64,
	strategy string,
) *WMSCloudEvent {
	data := RouteCalculatedData{
		RouteID:           routeID,
		PickerID:          pickerID,
		Stops:             stops,
		EstimatedDistance: estimatedDistance,
		Strategy:          strategy,
	}
	return f.CreateEvent(ctx, RouteCalculated, "route/"+routeID, data)
}

// CreateItemPickedEvent creates an ItemPicked event
func (f *EventFactory) CreateItemPickedEvent(
	ctx context.Context,
	taskID string,
	itemID string,
	sku string,
	quantity int,
	locationID string,
	toteID string,
) *WMSCloudEvent {
	data := ItemPickedData{
		TaskID:     taskID,
		ItemID:     itemID,
		SKU:        sku,
		Quantity:   quantity,
		LocationID: locationID,
		ToteID:     toteID,
	}
	return f.CreateEvent(ctx, ItemPicked, "pick-task/"+taskID, data)
}

// CreateShipmentCreatedEvent creates a ShipmentCreated event
func (f *EventFactory) CreateShipmentCreatedEvent(
	ctx context.Context,
	shipmentID string,
	orderID string,
	carrier string,
) *WMSCloudEvent {
	data := ShipmentCreatedData{
		ShipmentID: shipmentID,
		OrderID:    orderID,
		Carrier:    carrier,
	}
	return f.CreateEvent(ctx, ShipmentCreated, "shipment/"+shipmentID, data)
}

// CreateInventoryAdjustedEvent creates an InventoryAdjusted event
func (f *EventFactory) CreateInventoryAdjustedEvent(
	ctx context.Context,
	sku string,
	locationID string,
	previousQty int,
	newQty int,
	adjustmentType string,
	reason string,
) *WMSCloudEvent {
	data := InventoryAdjustedData{
		SKU:            sku,
		LocationID:     locationID,
		PreviousQty:    previousQty,
		NewQty:         newQty,
		AdjustmentType: adjustmentType,
		Reason:         reason,
	}
	return f.CreateEvent(ctx, InventoryAdjusted, "inventory/"+sku, data)
}

// CreateLaborTaskAssignedEvent creates a LaborTaskAssigned event
func (f *EventFactory) CreateLaborTaskAssignedEvent(
	ctx context.Context,
	workerID string,
	taskID string,
	taskType string,
	zone string,
) *WMSCloudEvent {
	data := LaborTaskAssignedData{
		WorkerID: workerID,
		TaskID:   taskID,
		TaskType: taskType,
		Zone:     zone,
	}
	return f.CreateEvent(ctx, LaborTaskAssigned, "worker/"+workerID, data)
}
