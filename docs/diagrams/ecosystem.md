# WMS Platform Ecosystem

This diagram provides a high-level overview of the WMS (Warehouse Management System) platform architecture, showing all microservices, orchestration layer, and infrastructure components.

## Architecture Overview

```mermaid
graph TB
    subgraph "External"
        Customer[Customer]
        Carrier[Carriers<br/>UPS/FedEx/USPS/DHL]
    end

    subgraph "WMS Platform"
        subgraph "API Gateway"
            Gateway[API Gateway]
        end

        subgraph "Orchestration Layer"
            Temporal[Temporal Server]
            Orchestrator[Orchestrator Service<br/>Port 8010]
        end

        subgraph "Order Management"
            Order[Order Service<br/>Port 8001]
            Waving[Waving Service<br/>Port 8003]
        end

        subgraph "Warehouse Operations"
            Inventory[Inventory Service<br/>Port 8002]
            Routing[Routing Service<br/>Port 8004]
            Picking[Picking Service<br/>Port 8005]
            Consolidation[Consolidation Service<br/>Port 8006]
        end

        subgraph "Fulfillment"
            Packing[Packing Service<br/>Port 8007]
            Shipping[Shipping Service<br/>Port 8008]
        end

        subgraph "Workforce"
            Labor[Labor Service<br/>Port 8009]
        end

        subgraph "Infrastructure"
            Kafka[Apache Kafka<br/>Event Streaming]
            MongoDB[(MongoDB<br/>Document Store)]
            OTEL[OpenTelemetry<br/>Observability]
            Prometheus[Prometheus<br/>Metrics]
        end
    end

    Customer --> Gateway
    Gateway --> Order

    Orchestrator --> Temporal
    Orchestrator --> Order
    Orchestrator --> Inventory
    Orchestrator --> Waving
    Orchestrator --> Routing
    Orchestrator --> Picking
    Orchestrator --> Consolidation
    Orchestrator --> Packing
    Orchestrator --> Shipping

    Order --> Kafka
    Waving --> Kafka
    Picking --> Kafka
    Packing --> Kafka
    Shipping --> Kafka
    Inventory --> Kafka
    Labor --> Kafka

    Order --> MongoDB
    Inventory --> MongoDB
    Waving --> MongoDB
    Routing --> MongoDB
    Picking --> MongoDB
    Consolidation --> MongoDB
    Packing --> MongoDB
    Shipping --> MongoDB
    Labor --> MongoDB

    Shipping --> Carrier
    Labor --> Picking
    Labor --> Packing
```

## Service Communication Patterns

```mermaid
graph LR
    subgraph "Synchronous (HTTP)"
        Orch[Orchestrator] -->|REST API| Services[Domain Services]
    end

    subgraph "Asynchronous (Events)"
        Services2[Domain Services] -->|Publish| Kafka[Apache Kafka]
        Kafka -->|Subscribe| Services3[Domain Services]
    end

    subgraph "Workflow (Temporal)"
        Temporal[Temporal] -->|Activities| Orch2[Orchestrator]
        Temporal -->|Signals| Orch2
    end
```

## Legend

| Symbol | Meaning |
|--------|---------|
| Rectangle | Microservice |
| Cylinder | Database |
| Arrow | Communication flow |
| Subgraph | Logical grouping |

## Related Diagrams

- [Order Fulfillment Flow](order-fulfillment-flow.md) - End-to-end order processing
- [Order Cancellation Flow](order-cancellation-flow.md) - Compensation pattern
- [Context Map](ddd/context-map.md) - DDD bounded contexts
