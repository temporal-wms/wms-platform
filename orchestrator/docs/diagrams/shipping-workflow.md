# Shipping Workflow (SLAM Process)

This diagram shows the shipping child workflow implementing the SLAM process: Scan, Label, Apply, Manifest.

## SLAM Process Overview

```mermaid
graph LR
    S[Scan] --> L[Label]
    L --> A[Apply]
    A --> M[Manifest]

    style S fill:#e1f5fe
    style L fill:#fff3e0
    style A fill:#e8f5e9
    style M fill:#fce4ec
```

## Shipping Sequence Diagram

```mermaid
sequenceDiagram
    autonumber
    participant Parent as OrderFulfillmentWorkflow
    participant Shipping as ShippingWorkflow
    participant ShippingSvc as Shipping Service
    participant OrderSvc as Order Service
    participant InventorySvc as Inventory Service
    participant Worker as Shipping Worker
    participant Scanner as Barcode Scanner
    participant Carrier as Carrier System
    participant Customer

    Parent->>Shipping: Start ShippingWorkflow
    Note over Shipping: WorkflowID: shipping-{orderId}

    rect rgb(225, 245, 254)
        Note over Shipping,ShippingSvc: Step 1: Create Shipment
        Shipping->>ShippingSvc: CreateShipment Activity
        ShippingSvc->>ShippingSvc: Create shipment record
        ShippingSvc->>ShippingSvc: Select carrier & service
        ShippingSvc-->>Shipping: ShipmentID
    end

    rect rgb(255, 243, 224)
        Note over Shipping,Scanner: Step 2: Scan Package (SLAM - S)
        Shipping->>ShippingSvc: ScanPackage Activity
        Worker->>Scanner: Scan Package Barcode
        Scanner->>ShippingSvc: Verify Package
        ShippingSvc->>ShippingSvc: Match to shipment
        ShippingSvc-->>Shipping: Package Verified
    end

    rect rgb(232, 245, 233)
        Note over Shipping,ShippingSvc: Step 3: Verify Label (SLAM - L)
        Shipping->>ShippingSvc: VerifyShippingLabel Activity
        Worker->>Scanner: Scan Label Barcode
        Scanner->>ShippingSvc: Read tracking number
        ShippingSvc->>ShippingSvc: Validate label data
        ShippingSvc-->>Shipping: Label Verified
    end

    rect rgb(252, 228, 236)
        Note over Shipping,Worker: Step 4: Place on Outbound Dock (SLAM - A)
        Shipping->>ShippingSvc: PlaceOnOutboundDock Activity
        Worker->>Worker: Move package to carrier lane
        Worker->>Scanner: Scan dock location
        ShippingSvc->>ShippingSvc: Record dock placement
        ShippingSvc-->>Shipping: Package Staged
    end

    rect rgb(243, 229, 245)
        Note over Shipping,Carrier: Step 5: Add to Manifest (SLAM - M)
        Shipping->>ShippingSvc: AddToCarrierManifest Activity
        ShippingSvc->>Carrier: Add to pickup manifest
        Carrier-->>ShippingSvc: Manifest Confirmed
        ShippingSvc-->>Shipping: Manifested
    end

    rect rgb(255, 253, 231)
        Note over Shipping,OrderSvc: Step 6: Mark Order Shipped
        Shipping->>OrderSvc: MarkOrderShipped Activity
        OrderSvc->>OrderSvc: Update order status
        OrderSvc-->>Shipping: Order Updated
    end

    rect rgb(255, 245, 230)
        Note over Shipping,InventorySvc: Step 7: Ship Inventory (if allocations)
        alt Has AllocationIDs
            Shipping->>InventorySvc: ShipInventory Activity
            Note right of Shipping: Finalize hard allocation - remove from system
            InventorySvc->>InventorySvc: POST /inventory/ship
            InventorySvc->>InventorySvc: Remove items from inventory
            InventorySvc-->>Shipping: Inventory Shipped
        end
    end

    rect rgb(224, 247, 250)
        Note over Shipping,Customer: Step 8: Notify Customer (Best Effort)
        Shipping->>ShippingSvc: NotifyCustomerShipped Activity
        ShippingSvc->>Customer: Email: Tracking Number
        Note right of Customer: Non-critical step
        ShippingSvc-->>Shipping: Notification Sent
    end

    Shipping-->>Parent: ShippingResult

    Note over Parent: Workflow Complete
```

## SLAM Station Layout

```mermaid
graph TD
    subgraph "SLAM Station"
        subgraph "Input"
            Conveyor[Incoming Conveyor]
        end

        subgraph "Scan Station"
            Scanner1[Package Scanner]
            Display1[Verification Display]
        end

        subgraph "Label Station"
            Scanner2[Label Scanner]
            Display2[Label Verification]
        end

        subgraph "Apply/Staging"
            Lanes[Carrier Lanes]
        end

        subgraph "Carriers"
            UPS[UPS Lane]
            FedEx[FedEx Lane]
            USPS[USPS Lane]
            DHL[DHL Lane]
        end

        Conveyor --> Scanner1
        Scanner1 --> Display1
        Display1 --> Scanner2
        Scanner2 --> Display2
        Display2 --> Lanes
        Lanes --> UPS
        Lanes --> FedEx
        Lanes --> USPS
        Lanes --> DHL
    end
```

## Shipment State Machine

```mermaid
stateDiagram-v2
    [*] --> Pending: Shipment Created
    Pending --> Scanned: Package Scanned
    Scanned --> Labeled: Label Verified
    Labeled --> Staged: On Outbound Dock
    Staged --> Manifested: Added to Manifest
    Manifested --> Shipped: Carrier Pickup
    Shipped --> InTransit: Carrier Scan
    InTransit --> Delivered: Delivery Confirmed
    Delivered --> [*]

    Pending --> Cancelled: Order Cancelled
    Scanned --> Exception: Scan Error
    Labeled --> Exception: Label Error
    Exception --> Pending: Error Resolved
    Cancelled --> [*]
```

## Data Structures

### Shipment
| Field | Type | Description |
|-------|------|-------------|
| ShipmentID | string | Unique identifier |
| OrderID | string | Associated order |
| PackageID | string | Package being shipped |
| Carrier | string | UPS/FedEx/USPS/DHL |
| Service | string | Service level |
| TrackingNumber | string | Tracking number |
| Status | string | Current status |
| Weight | float64 | Package weight |
| Dimensions | Dimensions | Package dimensions |
| ShippingAddress | Address | Destination |
| Label | ShippingLabel | Label info |

### Carrier Options
| Carrier | Services | Features |
|---------|----------|----------|
| UPS | Ground, 2-Day, Next Day | Full tracking, pickup |
| FedEx | Ground, Express, Priority | Real-time tracking |
| USPS | Priority, First Class | Residential delivery |
| DHL | Express, eCommerce | International |

### Manifest
| Field | Type | Description |
|-------|------|-------------|
| ManifestID | string | Unique identifier |
| Carrier | string | Carrier code |
| PickupDate | date | Scheduled pickup |
| Shipments | []string | Shipment IDs |
| Status | string | open/closed/picked_up |
| TotalPackages | int | Package count |
| TotalWeight | float64 | Combined weight |

## Error Handling

```mermaid
flowchart TD
    Scan[Scan Package] --> Check1{Valid?}
    Check1 -->|Yes| Label[Verify Label]
    Check1 -->|No| Error1[Rescan/Investigate]
    Error1 --> Scan

    Label --> Check2{Label OK?}
    Check2 -->|Yes| Stage[Stage Package]
    Check2 -->|No| Reprint[Reprint Label]
    Reprint --> Label

    Stage --> Check3{Correct Lane?}
    Check3 -->|Yes| Manifest[Add to Manifest]
    Check3 -->|No| Move[Move to Correct Lane]
    Move --> Stage

    Manifest --> Complete[Ship Confirmed]
```

## Related Diagrams

- [Order Fulfillment Flow](order-fulfillment.md) - Parent workflow
- [Packing Workflow](packing-workflow.md) - Previous step
- [Cancellation Workflow](cancellation-workflow.md) - Compensation workflow
