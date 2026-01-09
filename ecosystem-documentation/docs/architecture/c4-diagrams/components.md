---
sidebar_position: 3
---

# C4 Level 3: Component Diagrams

Component diagrams show the internal structure of each container, revealing the major building blocks and their interactions.

## Order Service Components

```mermaid
C4Component
    title Component Diagram - Order Service

    Container_Boundary(order, "Order Service") {
        Component(api, "API Layer", "Gin Handlers", "HTTP endpoints for order operations")
        Component(service, "Order Service", "Go", "Business logic and use cases")
        Component(domain, "Domain Layer", "Go", "Order aggregate, entities, events")
        Component(repo, "Repository", "Go", "MongoDB data access")
        Component(publisher, "Event Publisher", "Go", "Outbox-based Kafka publishing")
        Component(client, "HTTP Client", "Go", "Resilient external calls")
    }

    ContainerDb(mongo, "MongoDB", "orders_db")
    Container(kafka, "Kafka", "Event streaming")

    Rel(api, service, "Uses")
    Rel(service, domain, "Uses")
    Rel(service, repo, "Uses")
    Rel(service, publisher, "Uses")
    Rel(repo, mongo, "Reads/Writes")
    Rel(publisher, mongo, "Writes to outbox")
    Rel(publisher, kafka, "Publishes events")
```

### Order Service Components

| Component | Responsibility |
|-----------|---------------|
| **API Layer** | HTTP handlers, request validation, response mapping |
| **Order Service** | Use cases: CreateOrder, ValidateOrder, CancelOrder |
| **Domain Layer** | Order aggregate, OrderItem entity, domain events |
| **Repository** | CRUD operations, query implementations |
| **Event Publisher** | Outbox pattern implementation, Kafka producer |
| **HTTP Client** | Circuit breaker, retries, timeout handling |

## Picking Service Components

```mermaid
C4Component
    title Component Diagram - Picking Service

    Container_Boundary(picking, "Picking Service") {
        Component(api, "API Layer", "Gin Handlers", "HTTP endpoints for pick operations")
        Component(service, "Pick Service", "Go", "Pick task management")
        Component(domain, "Domain Layer", "Go", "PickTask aggregate, entities")
        Component(repo, "Repository", "Go", "MongoDB data access")
        Component(publisher, "Event Publisher", "Go", "Outbox-based publishing")
        Component(consumer, "Event Consumer", "Go", "Kafka consumer for waves")
        Component(labor_client, "Labor Client", "Go", "Worker assignment integration")
    }

    ContainerDb(mongo, "MongoDB", "picking_db")
    Container(kafka, "Kafka", "Event streaming")
    Container(labor, "Labor Service", "Workforce management")

    Rel(api, service, "Uses")
    Rel(service, domain, "Uses")
    Rel(service, repo, "Uses")
    Rel(service, publisher, "Uses")
    Rel(service, labor_client, "Uses")
    Rel(consumer, service, "Triggers")
    Rel(repo, mongo, "Reads/Writes")
    Rel(publisher, kafka, "Publishes events")
    Rel(consumer, kafka, "Consumes events")
    Rel(labor_client, labor, "REST API")
```

## Waving Service Components

```mermaid
C4Component
    title Component Diagram - Waving Service

    Container_Boundary(waving, "Waving Service") {
        Component(api, "API Layer", "Gin Handlers", "Wave management endpoints")
        Component(service, "Wave Service", "Go", "Wave lifecycle management")
        Component(scheduler, "Continuous Waving", "Go", "Automatic wave scheduling")
        Component(domain, "Domain Layer", "Go", "Wave aggregate")
        Component(repo, "Repository", "Go", "MongoDB data access")
        Component(publisher, "Event Publisher", "Go", "Outbox-based publishing")
        Component(temporal_client, "Temporal Client", "Go", "Workflow signal sender")
    }

    ContainerDb(mongo, "MongoDB", "waves_db")
    Container(kafka, "Kafka", "Event streaming")
    Container(temporal, "Temporal", "Workflow engine")

    Rel(api, service, "Uses")
    Rel(scheduler, service, "Uses")
    Rel(service, domain, "Uses")
    Rel(service, repo, "Uses")
    Rel(service, publisher, "Uses")
    Rel(service, temporal_client, "Uses")
    Rel(repo, mongo, "Reads/Writes")
    Rel(publisher, kafka, "Publishes events")
    Rel(temporal_client, temporal, "Sends signals")
```

## Orchestrator Components

```mermaid
C4Component
    title Component Diagram - Orchestrator

    Container_Boundary(orchestrator, "Orchestrator") {
        Component(worker, "Temporal Worker", "Go", "Workflow and activity registration")
        Component(workflows, "Workflows", "Go", "Order fulfillment workflows")
        Component(activities, "Activities", "Go", "Service integration activities")
        Component(clients, "Service Clients", "Go", "HTTP clients for each service")
        Component(health, "Health Server", "Go", "Liveness/readiness probes")
    }

    Container(temporal, "Temporal Server", "Workflow engine")
    Container(order, "Order Service", "Order management")
    Container(inventory, "Inventory Service", "Stock management")
    Container(picking, "Picking Service", "Pick operations")
    Container(packing, "Packing Service", "Pack operations")
    Container(shipping, "Shipping Service", "Ship operations")

    Rel(worker, temporal, "Registers with")
    Rel(worker, workflows, "Executes")
    Rel(workflows, activities, "Calls")
    Rel(activities, clients, "Uses")
    Rel(clients, order, "HTTP")
    Rel(clients, inventory, "HTTP")
    Rel(clients, picking, "HTTP")
    Rel(clients, packing, "HTTP")
    Rel(clients, shipping, "HTTP")
```

### Orchestrator Workflows

| Workflow | Description |
|----------|-------------|
| **OrderFulfillmentWorkflow** | Main workflow for end-to-end order processing |
| **PickingWorkflow** | Child workflow for picking operations |
| **ConsolidationWorkflow** | Child workflow for multi-item consolidation |
| **PackingWorkflow** | Child workflow for packing operations |
| **ShippingWorkflow** | Child workflow for shipping and SLAM |

### Orchestrator Activities

| Activity | Service | Operation |
|----------|---------|-----------|
| ValidateOrder | Order Service | Validate order data |
| ReserveInventory | Inventory Service | Reserve stock |
| CreatePickTask | Picking Service | Create picking task |
| GetPickTaskStatus | Picking Service | Check pick status |
| CreateConsolidation | Consolidation Service | Start consolidation |
| CreatePackTask | Packing Service | Create packing task |
| CreateShipment | Shipping Service | Create shipment |

## Shared Components

All services share common infrastructure components:

```mermaid
graph TB
    subgraph "Shared Package"
        Logging[Structured Logging]
        Metrics[Prometheus Metrics]
        Tracing[OpenTelemetry Tracing]
        CloudEvents[CloudEvent Types]
        MongoDB[MongoDB Client]
        Kafka[Kafka Client]
        Resilience[Circuit Breaker]
        Outbox[Outbox Publisher]
    end

    subgraph "Services"
        OrderSvc[Order Service]
        WavingSvc[Waving Service]
        PickingSvc[Picking Service]
    end

    OrderSvc --> Logging
    OrderSvc --> Metrics
    OrderSvc --> CloudEvents
    OrderSvc --> MongoDB
    OrderSvc --> Kafka
    OrderSvc --> Outbox

    WavingSvc --> Logging
    WavingSvc --> Metrics
    WavingSvc --> Tracing

    PickingSvc --> Resilience
    PickingSvc --> Outbox
```

## Component Interactions

### Order Creation Flow

```mermaid
sequenceDiagram
    participant Client
    participant API as API Layer
    participant Service as Order Service
    participant Domain as Domain Layer
    participant Repo as Repository
    participant Publisher as Event Publisher
    participant DB as MongoDB
    participant Kafka

    Client->>API: POST /orders
    API->>Service: CreateOrder(cmd)
    Service->>Domain: Order.Create()
    Domain-->>Service: Order + OrderReceivedEvent
    Service->>Repo: Save(order)
    Repo->>DB: Insert(order)
    Service->>Publisher: PublishEvent(event)
    Publisher->>DB: Insert into outbox
    Note over Publisher,Kafka: Async via Outbox Publisher
    Publisher->>Kafka: Publish OrderReceivedEvent
    Publisher->>DB: Mark as published
    API-->>Client: 201 Created
```

## Related Diagrams

- [Container Diagram](./containers) - High-level containers
- [Code Diagram](./code) - Class-level details
- [Domain Events](/domain-driven-design/domain-events) - Event catalog
