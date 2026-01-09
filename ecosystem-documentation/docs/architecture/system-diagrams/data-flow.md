---
sidebar_position: 3
---

# Data Flow Diagrams

This document describes the data flows within the WMS Platform, including synchronous API calls and asynchronous event flows.

## Order Data Flow

```mermaid
graph LR
    subgraph "External"
        Client[Client Application]
    end

    subgraph "WMS Platform"
        Order[Order Service]
        Kafka[Kafka]
        Waving[Waving Service]
        Orchestrator[Orchestrator]
        Temporal[Temporal]
    end

    Client -->|1. POST /orders| Order
    Order -->|2. OrderReceivedEvent| Kafka
    Order -->|3. Start Workflow| Orchestrator
    Orchestrator -->|4. Register| Temporal
    Kafka -->|5. Subscribe| Waving
    Waving -->|6. Signal| Orchestrator
```

## Event Flow Architecture

```mermaid
graph TB
    subgraph "Publishers"
        OrderSvc[Order Service]
        WavingSvc[Waving Service]
        PickingSvc[Picking Service]
        PackingSvc[Packing Service]
        ShippingSvc[Shipping Service]
        InventorySvc[Inventory Service]
    end

    subgraph "Kafka Topics"
        OrderEvents[wms.orders.events]
        WaveEvents[wms.waves.events]
        PickingEvents[wms.picking.events]
        PackingEvents[wms.packing.events]
        ShippingEvents[wms.shipping.events]
        InventoryEvents[wms.inventory.events]
    end

    subgraph "Consumers"
        WavingConsumer[Waving Service]
        InventoryConsumer[Inventory Service]
        AnalyticsConsumer[Analytics]
    end

    OrderSvc --> OrderEvents
    WavingSvc --> WaveEvents
    PickingSvc --> PickingEvents
    PackingSvc --> PackingEvents
    ShippingSvc --> ShippingEvents
    InventorySvc --> InventoryEvents

    OrderEvents --> WavingConsumer
    OrderEvents --> InventoryConsumer
    PickingEvents --> InventoryConsumer
    WaveEvents --> AnalyticsConsumer
```

## Transactional Outbox Pattern

```mermaid
sequenceDiagram
    participant API as API Handler
    participant Service as Domain Service
    participant DB as MongoDB
    participant Publisher as Outbox Publisher
    participant Kafka

    API->>Service: Create Order
    Service->>Service: Business Logic

    rect rgb(240, 248, 255)
        Note over Service,DB: Single Transaction
        Service->>DB: Save Order
        Service->>DB: Save to Outbox
        DB-->>Service: Transaction Committed
    end

    Service-->>API: Order Created

    rect rgb(255, 250, 240)
        Note over Publisher,Kafka: Async Publishing
        Publisher->>DB: Poll Outbox
        DB-->>Publisher: Unpublished Events
        Publisher->>Kafka: Publish Event
        Publisher->>DB: Mark Published
    end
```

## Database Data Flow

```mermaid
graph TB
    subgraph "Application Layer"
        OrderSvc[Order Service]
        WavingSvc[Waving Service]
        PickingSvc[Picking Service]
    end

    subgraph "MongoDB Cluster"
        subgraph "orders_db"
            Orders[(orders)]
            OrderOutbox[(outbox)]
        end

        subgraph "waves_db"
            Waves[(waves)]
            WaveOutbox[(outbox)]
        end

        subgraph "picking_db"
            PickTasks[(pick_tasks)]
            PickOutbox[(outbox)]
        end
    end

    OrderSvc -->|Read/Write| Orders
    OrderSvc -->|Write| OrderOutbox

    WavingSvc -->|Read/Write| Waves
    WavingSvc -->|Write| WaveOutbox

    PickingSvc -->|Read/Write| PickTasks
    PickingSvc -->|Write| PickOutbox
```

## Workflow Data Flow

```mermaid
sequenceDiagram
    participant Client
    participant Order as Order Service
    participant Orch as Orchestrator
    participant Temporal
    participant Inventory as Inventory Service
    participant Picking as Picking Service

    Client->>Order: Create Order
    Order->>Orch: Start Workflow

    Orch->>Temporal: RegisterWorkflow
    Temporal-->>Orch: WorkflowID

    Orch->>Order: ValidateOrder
    Order-->>Orch: Valid

    Orch->>Inventory: ReserveStock
    Inventory-->>Orch: Reserved

    Note over Orch: Wait for Wave Signal

    Orch->>Picking: CreatePickTask
    Picking-->>Orch: TaskID

    Note over Orch: Wait for Pick Complete

    Orch->>Temporal: WorkflowComplete
```

## Read vs Write Paths

### Write Path

```mermaid
graph LR
    subgraph "Write Path"
        API[API Request]
        Validation[Validation]
        Business[Business Logic]
        Aggregate[Aggregate Update]
        DB[(Database)]
        Outbox[(Outbox)]
        Kafka[Kafka]
    end

    API --> Validation
    Validation --> Business
    Business --> Aggregate
    Aggregate --> DB
    Aggregate --> Outbox
    Outbox -.->|Async| Kafka
```

### Read Path

```mermaid
graph LR
    subgraph "Read Path"
        API[API Request]
        Query[Query Handler]
        DB[(Database)]
        Cache[Cache Layer]
        Response[Response]
    end

    API --> Query
    Query --> Cache
    Cache -->|Miss| DB
    DB --> Cache
    Cache --> Response
```

## Event Sourcing (Domain Events)

```mermaid
graph TB
    subgraph "Order Aggregate"
        Create[OrderReceivedEvent]
        Validate[OrderValidatedEvent]
        Wave[OrderWaveAssignedEvent]
        Ship[OrderShippedEvent]
        Complete[OrderCompletedEvent]
    end

    subgraph "Event Store"
        Kafka[Kafka Topic]
    end

    subgraph "Projections"
        OrderView[Order View]
        Analytics[Analytics]
        Reporting[Reporting]
    end

    Create --> Kafka
    Validate --> Kafka
    Wave --> Kafka
    Ship --> Kafka
    Complete --> Kafka

    Kafka --> OrderView
    Kafka --> Analytics
    Kafka --> Reporting
```

## Cross-Service Data Flow

```mermaid
sequenceDiagram
    participant Order as Order Context
    participant Kafka
    participant Waving as Waving Context
    participant Picking as Picking Context
    participant Inventory as Inventory Context

    Order->>Kafka: OrderReceivedEvent

    par Parallel Processing
        Kafka->>Waving: OrderReceivedEvent
        Waving->>Waving: Add to Wave
    and
        Kafka->>Inventory: OrderReceivedEvent
        Inventory->>Inventory: Check Stock
    end

    Waving->>Kafka: WaveReleasedEvent
    Kafka->>Picking: WaveReleasedEvent
    Picking->>Picking: Create Pick Tasks

    Picking->>Kafka: PickTaskCompletedEvent
    Kafka->>Inventory: PickTaskCompletedEvent
    Inventory->>Inventory: Update Stock
```

## Data Consistency

### Saga Pattern

```mermaid
graph TD
    subgraph "Forward Flow"
        T1[Reserve Inventory]
        T2[Create Pick Task]
        T3[Create Pack Task]
        T4[Create Shipment]
    end

    subgraph "Compensation Flow"
        C4[Cancel Shipment]
        C3[Cancel Pack Task]
        C2[Cancel Pick Task]
        C1[Release Inventory]
    end

    T1 --> T2
    T2 --> T3
    T3 --> T4

    T4 -.->|Failure| C4
    C4 --> C3
    C3 --> C2
    C2 --> C1
```

### Eventual Consistency

```mermaid
sequenceDiagram
    participant Order as Order Service
    participant Kafka
    participant Inventory as Inventory Service

    Order->>Order: Create Order (Immediate)
    Order->>Kafka: OrderReceivedEvent

    Note over Kafka: Event in Transit

    Kafka->>Inventory: Consume Event
    Inventory->>Inventory: Reserve Stock (Eventually)

    Note over Order,Inventory: Eventual Consistency<br/>Window: ~100ms
```

## Data Retention

| Data Type | Retention Period | Storage |
|-----------|-----------------|---------|
| Orders | 7 years | MongoDB + Archive |
| Events | 30 days | Kafka |
| Traces | 7 days | Tempo |
| Logs | 14 days | Loki |
| Metrics | 15 days | Prometheus |

## Related Diagrams

- [Infrastructure](./infrastructure) - System topology
- [Deployment](./deployment) - Kubernetes resources
- [Domain Events](/domain-driven-design/domain-events) - Event catalog
