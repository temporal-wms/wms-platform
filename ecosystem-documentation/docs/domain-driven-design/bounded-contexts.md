---
sidebar_position: 2
---

# Bounded Contexts

This document describes the bounded contexts within the WMS Platform and their responsibilities.

## Context Overview

```mermaid
graph TB
    subgraph "Core Domain"
        Picking[Picking Context<br/>Pick Operations]
        Routing[Routing Context<br/>Path Optimization]
        Waving[Waving Context<br/>Batch Grouping]
    end

    subgraph "Supporting Domain"
        Inventory[Inventory Context<br/>Stock Management]
        Labor[Labor Context<br/>Workforce]
        Consolidation[Consolidation Context<br/>Item Combining]
    end

    subgraph "Generic Domain"
        Order[Order Context<br/>Order Lifecycle]
        Packing[Packing Context<br/>Packaging]
        Shipping[Shipping Context<br/>Carrier Integration]
    end

    style Picking fill:#ff9999
    style Routing fill:#ff9999
    style Waving fill:#ff9999
```

## Core Domain Contexts

### Picking Context

The Picking context handles warehouse picking operations - the physical retrieval of items from storage locations.

| Aspect | Description |
|--------|-------------|
| **Responsibility** | Manage picking task execution |
| **Aggregate Root** | PickTask |
| **Key Entities** | PickItem |
| **Value Objects** | Location, ToteID |
| **Domain Events** | PickTaskCreated, ItemPicked, PickTaskCompleted |

```mermaid
classDiagram
    class PickTask {
        <<Aggregate Root>>
        +ID string
        +OrderID string
        +WaveID string
        +WorkerID string
        +Items []PickItem
        +Status PickTaskStatus
        +AssignWorker()
        +PickItem()
        +Complete()
    }
```

**Why Core Domain?**
- Direct competitive advantage through pick efficiency
- High complexity in optimization
- Unique business rules per warehouse

### Routing Context

The Routing context calculates optimal pick paths through the warehouse.

| Aspect | Description |
|--------|-------------|
| **Responsibility** | Optimize pick path for efficiency |
| **Aggregate Root** | PickRoute |
| **Key Entities** | RouteStop |
| **Value Objects** | Location, Distance |
| **Domain Events** | RouteCalculated, RouteStarted, RouteCompleted |

```mermaid
classDiagram
    class PickRoute {
        <<Aggregate Root>>
        +ID string
        +TaskID string
        +Stops []RouteStop
        +TotalDistance float64
        +EstimatedTime Duration
        +Calculate()
        +Optimize()
    }
```

**Why Core Domain?**
- Directly impacts warehouse throughput
- Complex algorithms (TSP variants)
- Warehouse-specific optimizations

### Waving Context

The Waving context groups orders into batches (waves) for efficient processing.

| Aspect | Description |
|--------|-------------|
| **Responsibility** | Batch orders for picking |
| **Aggregate Root** | Wave |
| **Key Entities** | WaveOrder |
| **Value Objects** | WaveNumber, Priority |
| **Domain Events** | WaveCreated, OrderAddedToWave, WaveReleased |

```mermaid
classDiagram
    class Wave {
        <<Aggregate Root>>
        +ID string
        +WaveNumber string
        +Orders []WaveOrder
        +Status WaveStatus
        +AddOrder()
        +Schedule()
        +Release()
    }
```

**Why Core Domain?**
- Impacts overall fulfillment efficiency
- Complex scheduling algorithms
- Business-critical SLA management

## Supporting Domain Contexts

### Inventory Context

The Inventory context manages stock levels and locations.

| Aspect | Description |
|--------|-------------|
| **Responsibility** | Track stock quantities and locations |
| **Aggregate Root** | InventoryItem |
| **Key Entities** | Reservation |
| **Value Objects** | Location, SKU, Quantity |
| **Domain Events** | InventoryReceived, InventoryReserved, InventoryPicked |

```mermaid
classDiagram
    class InventoryItem {
        <<Aggregate Root>>
        +ID string
        +SKU string
        +Location Location
        +Quantity int
        +ReservedQuantity int
        +Reserve()
        +Pick()
        +Adjust()
    }
```

**Why Supporting Domain?**
- Essential for operations but not differentiating
- Standard inventory management patterns
- Could potentially use off-the-shelf solutions

### Labor Context

The Labor context manages workforce and task assignments.

| Aspect | Description |
|--------|-------------|
| **Responsibility** | Workforce management and assignment |
| **Aggregate Root** | Worker |
| **Key Entities** | TaskAssignment, Shift |
| **Value Objects** | EmployeeID, Certification |
| **Domain Events** | ShiftStarted, TaskAssigned, TaskCompleted |

```mermaid
classDiagram
    class Worker {
        <<Aggregate Root>>
        +ID string
        +EmployeeID string
        +Role WorkerRole
        +Status WorkerStatus
        +CurrentTask TaskAssignment
        +StartShift()
        +AssignTask()
        +CompleteTask()
    }
```

**Why Supporting Domain?**
- Necessary for operations
- Standard HR/WFM patterns
- Enables core domain efficiency

### Consolidation Context

The Consolidation context combines picked items for multi-item orders.

| Aspect | Description |
|--------|-------------|
| **Responsibility** | Combine items from multiple picks |
| **Aggregate Root** | ConsolidationUnit |
| **Key Entities** | ConsolidationItem |
| **Value Objects** | ToteID, OrderID |
| **Domain Events** | ConsolidationStarted, ItemConsolidated, ConsolidationCompleted |

```mermaid
classDiagram
    class ConsolidationUnit {
        <<Aggregate Root>>
        +ID string
        +OrderID string
        +Items []ConsolidationItem
        +Status ConsolidationStatus
        +AddItem()
        +Verify()
        +Complete()
    }
```

**Why Supporting Domain?**
- Supports picking/packing workflow
- Relatively straightforward logic
- Enables multi-item order efficiency

## Generic Domain Contexts

### Order Context

The Order context manages the order lifecycle from receipt to completion.

| Aspect | Description |
|--------|-------------|
| **Responsibility** | Order lifecycle management |
| **Aggregate Root** | Order |
| **Key Entities** | OrderItem |
| **Value Objects** | Address, Money, Priority |
| **Domain Events** | OrderReceived, OrderValidated, OrderShipped |

```mermaid
classDiagram
    class Order {
        <<Aggregate Root>>
        +ID string
        +CustomerID string
        +Items []OrderItem
        +Status OrderStatus
        +ShippingAddress Address
        +Validate()
        +AssignToWave()
        +Ship()
    }
```

**Why Generic Domain?**
- Standard e-commerce patterns
- Well-understood domain
- Could use third-party OMS

### Packing Context

The Packing context handles package preparation and labeling.

| Aspect | Description |
|--------|-------------|
| **Responsibility** | Package items for shipping |
| **Aggregate Root** | PackTask |
| **Key Entities** | PackItem, Package |
| **Value Objects** | Dimensions, Weight |
| **Domain Events** | PackTaskCreated, PackageSealed, LabelApplied |

```mermaid
classDiagram
    class PackTask {
        <<Aggregate Root>>
        +ID string
        +OrderID string
        +Items []PackItem
        +Package Package
        +Status PackTaskStatus
        +SelectPackaging()
        +Pack()
        +Seal()
    }
```

**Why Generic Domain?**
- Standard packing processes
- Industry-standard practices
- Straightforward implementation

### Shipping Context

The Shipping context handles carrier integration and the SLAM process.

| Aspect | Description |
|--------|-------------|
| **Responsibility** | Carrier integration, label generation, manifesting |
| **Aggregate Root** | Shipment |
| **Key Entities** | Package, ShippingLabel |
| **Value Objects** | TrackingNumber, Carrier |
| **Domain Events** | ShipmentCreated, LabelGenerated, ShipConfirmed |

```mermaid
classDiagram
    class Shipment {
        <<Aggregate Root>>
        +ID string
        +OrderID string
        +Carrier Carrier
        +TrackingNumber string
        +Label ShippingLabel
        +GenerateLabel()
        +Manifest()
        +Confirm()
    }
```

**Why Generic Domain?**
- Standard carrier integration patterns
- Well-defined external APIs
- Could use shipping aggregators

## Context Boundaries

### Strict Boundaries

Each context maintains strict boundaries:

1. **Separate Database** - Each context has its own database
2. **API Communication** - Synchronous calls via REST API
3. **Event Communication** - Asynchronous via Kafka events
4. **No Shared Tables** - No direct database access between contexts

### Shared Concepts

Some concepts are shared across contexts:

| Concept | Contexts | Implementation |
|---------|----------|----------------|
| Location | Routing, Picking | Shared Kernel |
| OrderID | All | Reference by ID |
| WaveID | Waving, Picking, Routing | Reference by ID |

## Related Documentation

- [Context Map](./context-map) - Context relationships
- [Aggregates](./aggregates/order) - Aggregate details
- [Domain Events](./domain-events) - Event catalog
