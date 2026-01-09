---
sidebar_position: 1
---

# C4 Level 1: System Context

The System Context diagram shows the WMS Platform as a whole and its relationships with external actors and systems.

## System Context Diagram

```mermaid
C4Context
    title System Context Diagram - WMS Platform

    Person(customer, "Customer", "Places orders for warehouse fulfillment")
    Person(warehouse_worker, "Warehouse Worker", "Picks, packs, and ships orders")
    Person(warehouse_manager, "Warehouse Manager", "Manages operations and monitors performance")

    System(wms, "WMS Platform", "Manages end-to-end warehouse operations including order fulfillment, inventory, and workforce")

    System_Ext(ecommerce, "E-Commerce Platform", "Online store systems that submit orders")
    System_Ext(erp, "ERP System", "Enterprise resource planning for inventory sync")
    System_Ext(carriers, "Carrier Systems", "UPS, FedEx, USPS, DHL for shipping")
    System_Ext(labor_system, "HR/Labor System", "Workforce scheduling and management")

    Rel(customer, ecommerce, "Places orders")
    Rel(ecommerce, wms, "Submits orders", "REST API")
    Rel(erp, wms, "Syncs inventory", "REST API")
    Rel(wms, carriers, "Ships packages", "REST API")
    Rel(warehouse_worker, wms, "Performs warehouse tasks", "Mobile/Web UI")
    Rel(warehouse_manager, wms, "Monitors operations", "Web Dashboard")
    Rel(labor_system, wms, "Provides worker schedules", "REST API")
```

## Context Description

### Primary Actors

| Actor | Description | Interactions |
|-------|-------------|--------------|
| **Customer** | End consumer who places orders through e-commerce platforms | Orders are received and fulfilled by WMS |
| **Warehouse Worker** | Employee who performs picking, packing, and shipping | Uses mobile/web interface to complete tasks |
| **Warehouse Manager** | Operations manager who oversees warehouse performance | Monitors dashboards, manages waves, adjusts priorities |

### External Systems

| System | Description | Integration |
|--------|-------------|-------------|
| **E-Commerce Platform** | Shopify, Magento, custom storefronts | REST API for order submission |
| **ERP System** | SAP, Oracle, NetSuite | Inventory synchronization |
| **Carrier Systems** | UPS, FedEx, USPS, DHL | Label generation, tracking, manifesting |
| **HR/Labor System** | Workforce management | Worker schedules, time tracking |

### WMS Platform Capabilities

The WMS Platform provides:

1. **Order Management**
   - Receive and validate orders
   - Track order status through fulfillment
   - Handle order modifications and cancellations

2. **Warehouse Operations**
   - Wave planning and release
   - Pick path optimization
   - Picking, consolidation, packing workflows

3. **Inventory Management**
   - Real-time stock levels
   - Location management
   - Reservation and allocation

4. **Shipping**
   - Carrier selection
   - Label generation (SLAM)
   - Manifest creation

5. **Workforce Management**
   - Task assignment
   - Performance tracking
   - Workload balancing

## Data Flows

### Inbound Flows
```mermaid
sequenceDiagram
    participant EC as E-Commerce
    participant WMS as WMS Platform
    participant ERP as ERP System

    EC->>WMS: Submit Order
    WMS-->>EC: Order Confirmation

    ERP->>WMS: Inventory Update
    WMS-->>ERP: Acknowledgment
```

### Outbound Flows
```mermaid
sequenceDiagram
    participant WMS as WMS Platform
    participant Carrier as Carrier System
    participant EC as E-Commerce

    WMS->>Carrier: Generate Label
    Carrier-->>WMS: Tracking Number

    WMS->>Carrier: Manifest Package
    Carrier-->>WMS: Pickup Confirmed

    WMS->>EC: Shipment Notification
```

## Security Boundary

```mermaid
graph TB
    subgraph "Internet Zone"
        EC[E-Commerce]
        Carriers[Carriers]
    end

    subgraph "DMZ"
        Gateway[API Gateway]
        WAF[Web Application Firewall]
    end

    subgraph "Internal Zone"
        WMS[WMS Platform]
        DB[(Databases)]
    end

    EC --> WAF
    Carriers --> WAF
    WAF --> Gateway
    Gateway --> WMS
    WMS --> DB
```

## Related Diagrams

- [Container Diagram](./containers) - Internal structure of the WMS Platform
- [Architecture Overview](../overview) - Detailed architecture description
