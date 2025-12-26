# Shipping Service

The Shipping Service manages carrier integration, shipment creation, and the SLAM (Scan, Label, Apply, Manifest) process.

## Overview

- **Port**: 8007
- **Database**: shipping_db (MongoDB)
- **Aggregate Root**: Shipment

## Features

- Shipment creation and management
- Carrier integration (UPS, FedEx, USPS)
- Shipping label generation
- Manifest management
- Tracking code generation
- Anti-Corruption Layer for external carriers

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/shipments` | Create shipment |
| GET | `/api/v1/shipments/:shipmentId` | Get shipment |
| POST | `/api/v1/shipments/:shipmentId/label` | Generate label |
| POST | `/api/v1/shipments/:shipmentId/ship` | Mark as shipped |
| GET | `/api/v1/shipments/order/:orderId` | Get by order ID |
| POST | `/api/v1/manifests` | Create manifest |
| POST | `/api/v1/manifests/:manifestId/shipments` | Add to manifest |
| POST | `/api/v1/manifests/:manifestId/close` | Close manifest |

## Events Published

| Event | Topic | Description |
|-------|-------|-------------|
| `ShipmentCreated` | wms.shipping.events | Shipment created |
| `LabelGenerated` | wms.shipping.events | Label generated |
| `ShipmentManifested` | wms.shipping.events | Added to manifest |
| `ShipConfirmed` | wms.shipping.events | Shipment confirmed |
| `DeliveryConfirmed` | wms.shipping.events | Delivery confirmed |

## Domain Model

```go
type Shipment struct {
    ID           string
    OrderID      string
    PackageID    string
    Carrier      Carrier
    TrackingCode string
    Label        *ShippingLabel
    Address      Address
    Status       ShipmentStatus
    ManifestID   *string
    ShippedAt    *time.Time
    DeliveredAt  *time.Time
    CreatedAt    time.Time
}

type Carrier struct {
    Code        string  // UPS, FEDEX, USPS
    ServiceType string  // ground, express, overnight
    AccountID   string
}

type ShipmentStatus string
const (
    ShipmentStatusCreated   ShipmentStatus = "created"
    ShipmentStatusLabeled   ShipmentStatus = "labeled"
    ShipmentStatusManifested ShipmentStatus = "manifested"
    ShipmentStatusShipped   ShipmentStatus = "shipped"
    ShipmentStatusDelivered ShipmentStatus = "delivered"
)
```

## Running Locally

```bash
go run cmd/api/main.go
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8007` |
| `MONGODB_URI` | MongoDB connection | `mongodb://localhost:27017` |
| `MONGODB_DATABASE` | Database name | `shipping_db` |
| `KAFKA_BROKERS` | Kafka brokers | `localhost:9092` |

## Testing

```bash
go test ./...
```

## Related Services

- **packing-service**: Provides packed packages
- **order-service**: Receives shipment confirmations
