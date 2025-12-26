# Shipping Service - DDD Aggregates

This document describes the aggregate structure for the Shipping bounded context.

## Aggregate: Shipment

The Shipment aggregate manages the SLAM process and carrier integration.

```mermaid
graph TD
    subgraph "Shipment Aggregate"
        Shipment[Shipment<br/><<Aggregate Root>>]

        subgraph "Entities"
            Label[ShippingLabel]
        end

        subgraph "Value Objects"
            Carrier[Carrier]
            Address[Address]
            Dimensions[Dimensions]
            ShipmentStatus[ShipmentStatus]
            CarrierCode[CarrierCode]
            LabelFormat[LabelFormat]
        end

        Shipment -->|has| Label
        Shipment -->|uses| Carrier
        Shipment -->|ships to| Address
        Shipment -->|sized| Dimensions
        Shipment -->|status| ShipmentStatus
        Carrier -->|code| CarrierCode
        Label -->|format| LabelFormat
    end

    style Shipment fill:#f9f,stroke:#333,stroke-width:4px
```

## Aggregate: Manifest

```mermaid
graph TD
    subgraph "Manifest Aggregate"
        Manifest[Manifest<br/><<Aggregate Root>>]

        subgraph "Value Objects"
            ManifestStatus[ManifestStatus]
        end

        Manifest -->|status| ManifestStatus
        Manifest -->|contains| ShipmentRef[ShipmentID References]
    end

    style Manifest fill:#f9f,stroke:#333,stroke-width:4px
```

## Aggregate Boundaries

```mermaid
graph LR
    subgraph "Shipment Aggregate Boundary"
        S[Shipment]
        SL[ShippingLabel]
        C[Carrier]
        A[Address]
    end

    subgraph "Manifest Aggregate Boundary"
        M[Manifest]
    end

    subgraph "External References"
        O[OrderID]
        P[PackageID]
    end

    S -.->|for| O
    S -.->|ships| P
    S -.->|in| M

    style S fill:#f9f,stroke:#333,stroke-width:2px
    style M fill:#f9f,stroke:#333,stroke-width:2px
```

## Invariants

| Invariant | Description |
|-----------|-------------|
| Label required | Cannot manifest without label |
| Valid carrier | Carrier must support service type |
| Address valid | Shipping address must be complete |
| Manifest open | Can only add to open manifest |

## Domain Events

```mermaid
graph LR
    Ship[Shipment] -->|emits| E1[ShipmentCreatedEvent]
    Ship -->|emits| E2[LabelGeneratedEvent]
    Ship -->|emits| E3[ShipmentManifestedEvent]
    Ship -->|emits| E4[ShipConfirmedEvent]
    Ship -->|emits| E5[DeliveryConfirmedEvent]
```

## Related Documentation

- [Class Diagram](../class-diagram.md) - Full domain model
- [Shipping Workflow](../../../../orchestrator/docs/diagrams/shipping-workflow.md) - SLAM process
