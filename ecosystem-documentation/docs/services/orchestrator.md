---
sidebar_position: 10
---

# Orchestrator

The Orchestrator is a Temporal worker that executes workflow orchestration.

## Overview

| Property | Value |
|----------|-------|
| **Port** | 8080 (metrics) |
| **Type** | Worker (no HTTP API) |
| **Task Queue** | orchestrator-queue |

## Responsibilities

- Execute Temporal workflows
- Coordinate service activities
- Handle saga compensation
- Manage workflow signals

## Workflows

### OrderFulfillmentWorkflow

Main workflow for end-to-end order processing:

```mermaid
graph LR
    Start[Start] --> Validate[Validate Order]
    Validate --> Reserve[Reserve Inventory]
    Reserve --> WaitWave[Wait for Wave Signal]
    WaitWave --> Route[Calculate Route]
    Route --> Pick[Picking Workflow]
    Pick --> Consolidate{Multi-item?}
    Consolidate -->|Yes| Consol[Consolidation Workflow]
    Consolidate -->|No| Pack[Packing Workflow]
    Consol --> Pack
    Pack --> Ship[Shipping Workflow]
    Ship --> Complete[Complete Order]
```

### Child Workflows

| Workflow | Purpose |
|----------|---------|
| PickingWorkflow | Coordinate picking operations |
| ConsolidationWorkflow | Combine multi-item orders |
| PackingWorkflow | Package preparation |
| ShippingWorkflow | SLAM process |

### OrderCancellationWorkflow

Compensation workflow:

```mermaid
graph LR
    Start[Cancel Request] --> CancelOrder[Cancel Order]
    CancelOrder --> ReleaseInventory[Release Inventory]
    ReleaseInventory --> Notify[Notify Customer]
    Notify --> Complete[Complete]
```

## Activities

| Activity | Service | Operation |
|----------|---------|-----------|
| ValidateOrder | Order Service | Validate order |
| ReserveInventory | Inventory Service | Reserve stock |
| ReleaseInventory | Inventory Service | Release reservation |
| CreatePickTask | Picking Service | Create pick task |
| GetPickTaskStatus | Picking Service | Check status |
| CreateConsolidation | Consolidation Service | Start consolidation |
| CreatePackTask | Packing Service | Create pack task |
| CreateShipment | Shipping Service | Create shipment |
| MarkOrderShipped | Order Service | Update status |

## Signals

| Signal | Description | Source |
|--------|-------------|--------|
| waveAssigned | Order assigned to wave | Waving Service |
| pickCompleted | Picking task done | Picking Service |
| packCompleted | Packing task done | Packing Service |

## Workflow Configuration

```go
workflow.ExecuteActivity(ctx, ValidateOrderActivity, activityOptions{
    StartToCloseTimeout: 5 * time.Minute,
    RetryPolicy: &temporal.RetryPolicy{
        MaximumAttempts:    3,
        InitialInterval:    1 * time.Second,
        BackoffCoefficient: 2.0,
    },
})
```

## Priority-Based Timeouts

| Priority | Wave Wait Timeout |
|----------|------------------|
| same_day | 30 minutes |
| next_day | 2 hours |
| standard | 4 hours |

## Service Clients

```mermaid
graph TB
    subgraph "Orchestrator"
        Workflows[Workflows]
        Activities[Activities]
    end

    subgraph "Service Clients"
        OrderClient[Order Client]
        InventoryClient[Inventory Client]
        WavingClient[Waving Client]
        RoutingClient[Routing Client]
        PickingClient[Picking Client]
        ConsolidationClient[Consolidation Client]
        PackingClient[Packing Client]
        ShippingClient[Shipping Client]
    end

    Activities --> OrderClient
    Activities --> InventoryClient
    Activities --> WavingClient
    Activities --> RoutingClient
    Activities --> PickingClient
    Activities --> ConsolidationClient
    Activities --> PackingClient
    Activities --> ShippingClient
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| SERVICE_NAME | Service identifier | orchestrator |
| TEMPORAL_HOST | Temporal server | Required |
| TEMPORAL_NAMESPACE | Temporal namespace | wms |
| TEMPORAL_TASK_QUEUE | Task queue | orchestrator-queue |
| ORDER_SERVICE_URL | Order service URL | Required |
| INVENTORY_SERVICE_URL | Inventory service URL | Required |
| ... | Other service URLs | Required |

## Health Endpoints

- `GET /health` - Liveness (worker running)
- `GET /metrics` - Prometheus metrics

## Temporal UI

Monitor workflows at: `http://temporal-ui:8080`

```mermaid
graph TB
    subgraph "Temporal"
        Frontend[Frontend]
        History[History Service]
        Matching[Matching Service]
        Worker[Worker Service]
        UI[Temporal UI]
    end

    Orchestrator -->|Register| Frontend
    Frontend --> History
    Frontend --> Matching
    Frontend --> Worker
    UI --> Frontend
```

## Related Documentation

- [Order Fulfillment](/architecture/sequence-diagrams/order-fulfillment) - Main workflow
- [Order Cancellation](/architecture/sequence-diagrams/order-cancellation) - Compensation
- [Temporal Infrastructure](/infrastructure/temporal) - Server setup
