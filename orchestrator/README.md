# WMS Platform Orchestrator

The orchestrator service coordinates the end-to-end order fulfillment process using Temporal workflows.

## Architecture

### Main Workflows

1. **OrderFulfillmentWorkflow** (`workflows/order_fulfillment.go`)
   - Orchestrates the complete order fulfillment saga
   - Steps: Validate → Wait for Wave → Route → Pick → Consolidate → Pack → Ship
   - Includes compensation logic for failures

2. **OrderCancellationWorkflow** (`workflows/order_cancellation.go`)
   - Handles order cancellation with compensating transactions
   - Steps: Cancel order → Release inventory → Notify customer

### Child Workflows

3. **PickingWorkflow** (`workflows/picking_workflow.go`)
   - Coordinates picking process
   - Creates pick task → Assigns worker → Waits for completion

4. **ConsolidationWorkflow** (`workflows/consolidation_workflow.go`)
   - Consolidates multi-item orders from different totes
   - Creates consolidation unit → Consolidates items → Verifies → Completes

5. **PackingWorkflow** (`workflows/packing_workflow.go`)
   - Handles packing process
   - Creates task → Selects packaging → Packs items → Weighs → Generates label → Seals

6. **ShippingWorkflow** (`workflows/shipping_workflow.go`)
   - Implements SLAM (Scan, Label, Apply, Manifest) process
   - Scans → Verifies label → Places on dock → Adds to manifest → Marks shipped → Notifies

## Activities

### Order Activities (`activities/order_activities.go`)
- `ValidateOrder`: Validates order with order-service
- `CancelOrder`: Cancels an order
- `NotifyCustomerCancellation`: Sends cancellation notification

### Routing Activities (`activities/routing_activities.go`)
- `CalculateRoute`: Calculates optimal pick route

### Inventory Activities (`activities/inventory_activities.go`)
- `ReleaseInventoryReservation`: Releases reserved inventory

## Service Clients

The orchestrator uses HTTP clients to communicate with all WMS services (`activities/clients/`):

- **Order Service**: Order validation and management
- **Inventory Service**: Inventory reservations
- **Routing Service**: Route calculation
- **Picking Service**: Pick task management
- **Consolidation Service**: Multi-item consolidation
- **Packing Service**: Packing task management
- **Shipping Service**: Shipment creation and carrier integration
- **Labor Service**: Worker assignment
- **Waving Service**: Wave management

## Configuration

Service URLs are configured via environment variables:

```bash
TEMPORAL_HOST=localhost:7233
TEMPORAL_NAMESPACE=default

ORDER_SERVICE_URL=http://localhost:8001
INVENTORY_SERVICE_URL=http://localhost:8008
ROUTING_SERVICE_URL=http://localhost:8003
PICKING_SERVICE_URL=http://localhost:8004
CONSOLIDATION_SERVICE_URL=http://localhost:8005
PACKING_SERVICE_URL=http://localhost:8006
SHIPPING_SERVICE_URL=http://localhost:8007
LABOR_SERVICE_URL=http://localhost:8009
WAVING_SERVICE_URL=http://localhost:8002
```

## Running the Orchestrator

```bash
# Start the worker
go run cmd/worker/main.go
```

## Workflow Execution Flow

```
OrderFulfillmentWorkflow
├── ValidateOrder (activity)
├── Wait for waveAssigned signal
├── CalculateRoute (activity)
├── PickingWorkflow (child workflow)
│   ├── CreatePickTask
│   ├── AssignPickerToTask
│   └── Wait for pickCompleted signal
├── ConsolidationWorkflow (child workflow) [if multi-item]
│   ├── CreateConsolidationUnit
│   ├── ConsolidateItems
│   ├── VerifyConsolidation
│   └── CompleteConsolidation
├── PackingWorkflow (child workflow)
│   ├── CreatePackTask
│   ├── SelectPackagingMaterials
│   ├── PackItems
│   ├── WeighPackage
│   ├── GenerateShippingLabel
│   ├── ApplyLabelToPackage
│   └── SealPackage
└── ShippingWorkflow (child workflow)
    ├── CreateShipment
    ├── ScanPackage
    ├── VerifyShippingLabel
    ├── PlaceOnOutboundDock
    ├── AddToCarrierManifest
    ├── MarkOrderShipped
    └── NotifyCustomerShipped
```

## Error Handling

- Activities have retry policies with exponential backoff
- Workflow includes compensation logic (e.g., ReleaseInventoryReservation on failure)
- Timeouts configured based on priority (same-day: 30min, next-day: 2h, standard: 4h)

## Signals

- **waveAssigned**: Triggered when order is assigned to a wave
- **pickCompleted**: Triggered when picking is completed

## Status

✅ **Phase 2 Complete**: All workflows and activities implemented with HTTP client infrastructure
