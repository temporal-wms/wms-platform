---
sidebar_position: 2
---

# C4 Level 2: Container Diagram

The Container diagram shows the high-level technology choices and how responsibilities are distributed across containers (applications, databases, etc.).

## Container Diagram

```mermaid
C4Container
    title Container Diagram - WMS Platform

    Person(worker, "Warehouse Worker", "Performs warehouse tasks")
    Person(manager, "Warehouse Manager", "Monitors operations")

    System_Boundary(wms, "WMS Platform") {
        Container(api_gateway, "API Gateway", "Kong/Nginx", "Routes requests, handles auth")

        Container(order_svc, "Order Service", "Go, Gin", "Order lifecycle management")
        Container(waving_svc, "Waving Service", "Go, Gin", "Batch order grouping")
        Container(routing_svc, "Routing Service", "Go, Gin", "Pick path optimization")
        Container(picking_svc, "Picking Service", "Go, Gin", "Picking operations")
        Container(consolidation_svc, "Consolidation Service", "Go, Gin", "Multi-item combining")
        Container(packing_svc, "Packing Service", "Go, Gin", "Package preparation")
        Container(shipping_svc, "Shipping Service", "Go, Gin", "Carrier integration")
        Container(inventory_svc, "Inventory Service", "Go, Gin", "Stock management")
        Container(labor_svc, "Labor Service", "Go, Gin", "Workforce management")

        Container(orchestrator, "Orchestrator", "Go, Temporal SDK", "Workflow orchestration")

        ContainerDb(mongodb, "MongoDB", "MongoDB 6.0", "Document storage for all services")
        Container(kafka, "Apache Kafka", "Kafka 3.0", "Event streaming platform")
        Container(temporal, "Temporal Server", "Temporal", "Workflow engine")

        Container(prometheus, "Prometheus", "Prometheus", "Metrics collection")
        Container(tempo, "Tempo", "Grafana Tempo", "Distributed tracing")
    }

    System_Ext(carriers, "Carrier Systems", "UPS, FedEx, USPS")

    Rel(worker, api_gateway, "Uses", "HTTPS")
    Rel(manager, api_gateway, "Uses", "HTTPS")

    Rel(api_gateway, order_svc, "Routes to", "HTTP")
    Rel(api_gateway, waving_svc, "Routes to", "HTTP")
    Rel(api_gateway, picking_svc, "Routes to", "HTTP")

    Rel(orchestrator, temporal, "Registers workflows", "gRPC")
    Rel(orchestrator, order_svc, "Calls", "HTTP")
    Rel(orchestrator, waving_svc, "Calls", "HTTP")
    Rel(orchestrator, routing_svc, "Calls", "HTTP")
    Rel(orchestrator, picking_svc, "Calls", "HTTP")
    Rel(orchestrator, consolidation_svc, "Calls", "HTTP")
    Rel(orchestrator, packing_svc, "Calls", "HTTP")
    Rel(orchestrator, shipping_svc, "Calls", "HTTP")
    Rel(orchestrator, inventory_svc, "Calls", "HTTP")

    Rel(order_svc, mongodb, "Reads/Writes", "MongoDB Protocol")
    Rel(order_svc, kafka, "Publishes events", "Kafka Protocol")

    Rel(shipping_svc, carriers, "Integrates with", "HTTPS")
```

## Container Descriptions

### Application Services

| Container | Technology | Port | Description |
|-----------|------------|------|-------------|
| **Order Service** | Go, Gin | 8001 | Manages order lifecycle from receipt to completion |
| **Waving Service** | Go, Gin | 8002 | Groups orders into waves for efficient picking |
| **Routing Service** | Go, Gin | 8003 | Calculates optimal pick paths through warehouse |
| **Picking Service** | Go, Gin | 8004 | Manages picking task execution |
| **Consolidation Service** | Go, Gin | 8005 | Combines items from multi-item orders |
| **Packing Service** | Go, Gin | 8006 | Handles package preparation and labeling |
| **Shipping Service** | Go, Gin | 8007 | Integrates with carriers for shipping |
| **Inventory Service** | Go, Gin | 8008 | Tracks stock levels and locations |
| **Labor Service** | Go, Gin | 8009 | Manages workforce and task assignments |
| **Orchestrator** | Go, Temporal SDK | 8080 | Executes Temporal workflows |

### Infrastructure Containers

| Container | Technology | Purpose |
|-----------|------------|---------|
| **MongoDB** | MongoDB 6.0 | Document storage (database per service) |
| **Apache Kafka** | Kafka 3.0 | Event streaming and messaging |
| **Temporal Server** | Temporal | Workflow execution engine |
| **Prometheus** | Prometheus | Metrics collection and alerting |
| **Tempo** | Grafana Tempo | Distributed trace storage |

## Service Interactions

### Synchronous Communication

```mermaid
graph LR
    subgraph "Orchestrator Activities"
        O[Orchestrator]
        O -->|ValidateOrder| Order[Order Service]
        O -->|ReserveInventory| Inv[Inventory Service]
        O -->|CreatePickTask| Pick[Picking Service]
        O -->|CreatePackTask| Pack[Packing Service]
        O -->|CreateShipment| Ship[Shipping Service]
    end
```

### Asynchronous Communication

```mermaid
graph TB
    subgraph "Event Publishing"
        Order[Order Service] -->|OrderReceivedEvent| K1[orders.events]
        Waving[Waving Service] -->|WaveReleasedEvent| K2[waves.events]
        Picking[Picking Service] -->|PickTaskCompletedEvent| K3[picking.events]
        Packing[Packing Service] -->|PackageSealedEvent| K4[packing.events]
        Shipping[Shipping Service] -->|ShipConfirmedEvent| K5[shipping.events]
    end

    subgraph "Kafka Topics"
        K1
        K2
        K3
        K4
        K5
    end
```

## Database Architecture

Each service has its own database:

```mermaid
graph TB
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

        subgraph "inventory_db"
            InventoryItems[(inventory_items)]
            Reservations[(reservations)]
        end
    end

    OrderSvc[Order Service] --> Orders
    WavingSvc[Waving Service] --> Waves
    PickingSvc[Picking Service] --> PickTasks
    InventorySvc[Inventory Service] --> InventoryItems
```

## Network Topology

```mermaid
graph TB
    subgraph "External Network"
        LB[Load Balancer]
    end

    subgraph "Service Mesh"
        Gateway[API Gateway]

        subgraph "Service Network"
            Services[Domain Services]
            Orchestrator[Orchestrator]
        end

        subgraph "Data Network"
            MongoDB[(MongoDB)]
            Kafka[Kafka]
            Temporal[Temporal]
        end
    end

    LB --> Gateway
    Gateway --> Services
    Services --> MongoDB
    Services --> Kafka
    Orchestrator --> Temporal
    Orchestrator --> Services
```

## Scalability Considerations

| Container | Scaling Strategy | Notes |
|-----------|-----------------|-------|
| Order Service | Horizontal | Stateless, scale based on order volume |
| Picking Service | Horizontal | Scale based on warehouse zones |
| Orchestrator | Horizontal | Multiple workers for workflow processing |
| MongoDB | ReplicaSet + Sharding | Shard by tenant/warehouse |
| Kafka | Partition-based | Increase partitions for throughput |

## Related Diagrams

- [System Context](./context) - External view
- [Component Diagram](./components) - Internal service structure
- [Infrastructure](../system-diagrams/infrastructure) - Deployment details
