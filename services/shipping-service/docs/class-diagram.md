# Shipping Service - Class Diagram

This diagram shows the domain model for the Shipping Service bounded context.

## Domain Model

```mermaid
classDiagram
    class Shipment {
        <<Aggregate Root>>
        +ShipmentID string
        +OrderID string
        +PackageID string
        +Carrier Carrier
        +Service string
        +TrackingNumber string
        +Status ShipmentStatus
        +Weight float64
        +Dimensions Dimensions
        +ShippingAddress Address
        +Label ShippingLabel
        +ManifestID string
        +CreatedAt time.Time
        +ShippedAt time.Time
        +DeliveredAt time.Time
        +GenerateLabel()
        +AddToManifest(manifestID string)
        +ConfirmShipment()
        +ConfirmDelivery()
        +Cancel()
    }

    class Carrier {
        <<Value Object>>
        +Code CarrierCode
        +Name string
        +AccountID string
        +ServiceType string
        +SupportsInternational() bool
        +RequiresCustomsDocs() bool
    }

    class ShippingLabel {
        <<Entity>>
        +LabelID string
        +TrackingNumber string
        +LabelFormat LabelFormat
        +LabelData []byte
        +GeneratedAt time.Time
        +Voided bool
    }

    class Manifest {
        <<Entity>>
        +ManifestID string
        +Carrier CarrierCode
        +PickupDate time.Time
        +Status ManifestStatus
        +Shipments []string
        +TotalPackages int
        +TotalWeight float64
        +ClosedAt time.Time
        +AddShipment(shipmentID string)
        +Close()
        +ConfirmPickup()
    }

    class Address {
        <<Value Object>>
        +Name string
        +Company string
        +Street1 string
        +Street2 string
        +City string
        +State string
        +PostalCode string
        +Country string
        +Phone string
        +IsResidential bool
    }

    class Dimensions {
        <<Value Object>>
        +Length float64
        +Width float64
        +Height float64
    }

    class CarrierCode {
        <<Enumeration>>
        UPS
        FEDEX
        USPS
        DHL
    }

    class ShipmentStatus {
        <<Enumeration>>
        pending
        labeled
        manifested
        shipped
        in_transit
        delivered
        cancelled
    }

    class ManifestStatus {
        <<Enumeration>>
        open
        closed
        picked_up
    }

    class LabelFormat {
        <<Enumeration>>
        PDF
        ZPL
        PNG
    }

    Shipment "1" *-- "1" Carrier : uses
    Shipment "1" *-- "1" ShippingLabel : has
    Shipment "1" *-- "1" Address : ships to
    Shipment "1" *-- "1" Dimensions : sized
    Shipment "*" --o "1" Manifest : included in
    Shipment --> ShipmentStatus : has
    Manifest --> ManifestStatus : has
    Carrier --> CarrierCode : identified by
    ShippingLabel --> LabelFormat : format
```

## Shipment Flow

```mermaid
stateDiagram-v2
    [*] --> pending: Create Shipment
    pending --> labeled: GenerateLabel()
    labeled --> manifested: AddToManifest()
    manifested --> shipped: ConfirmShipment()
    shipped --> in_transit: Carrier Scan
    in_transit --> delivered: ConfirmDelivery()
    delivered --> [*]

    pending --> cancelled: Cancel()
    labeled --> cancelled: Void Label
    cancelled --> [*]
```

## Related Diagrams

- [Aggregate Diagram](ddd/aggregates.md) - DDD aggregate structure
- [Shipping Workflow](../../../orchestrator/docs/diagrams/shipping-workflow.md) - SLAM process
