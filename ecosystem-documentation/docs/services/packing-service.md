---
sidebar_position: 6
---

# Packing Service

The Packing Service handles package preparation and labeling.

## Overview

| Property | Value |
|----------|-------|
| **Port** | 8006 |
| **Database** | packing_db |
| **Aggregate Root** | PackTask |
| **Bounded Context** | Packing |

## Responsibilities

- Create and manage pack tasks
- Select appropriate packaging
- Record package weights/dimensions
- Apply shipping labels
- Seal packages

## API Endpoints

### Create Pack Task

```http
POST /api/v1/pack-tasks
Content-Type: application/json

{
  "orderId": "ORD-12345",
  "stationId": "STATION-01",
  "items": [
    { "sku": "SKU-001", "quantity": 2 }
  ]
}
```

### Get Pack Task

```http
GET /api/v1/pack-tasks/{id}
```

### Select Packaging

```http
PUT /api/v1/pack-tasks/{id}/packaging
Content-Type: application/json

{
  "packageType": "box",
  "dimensions": {
    "length": 30,
    "width": 20,
    "height": 15,
    "unit": "cm"
  }
}
```

### Pack Item

```http
POST /api/v1/pack-tasks/{id}/items/{itemId}/pack
Content-Type: application/json

{
  "quantity": 2
}
```

### Record Weight

```http
PUT /api/v1/pack-tasks/{id}/weight
Content-Type: application/json

{
  "weight": { "value": 2.5, "unit": "kg" }
}
```

### Apply Label

```http
PUT /api/v1/pack-tasks/{id}/label
Content-Type: application/json

{
  "trackingNumber": "1Z999AA10123456784"
}
```

### Seal Package

```http
PUT /api/v1/pack-tasks/{id}/seal
```

## Domain Events Published

| Event | Topic | Description |
|-------|-------|-------------|
| PackTaskCreatedEvent | wms.packing.events | Task created |
| PackagingSuggestedEvent | wms.packing.events | Package selected |
| PackageSealedEvent | wms.packing.events | Package sealed |
| LabelAppliedEvent | wms.packing.events | Label affixed |
| PackTaskCompletedEvent | wms.packing.events | Complete |

## Package Types

| Type | Use Case | Max Weight |
|------|----------|------------|
| envelope | Documents | 0.5 kg |
| padded_envelope | Small fragile | 1 kg |
| bag | Soft goods | 5 kg |
| box | General | 30 kg |
| custom | Oversize | Varies |

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| SERVICE_NAME | Service identifier | packing-service |
| MONGODB_DATABASE | Database name | packing_db |

## Related Documentation

- [PackTask Aggregate](/domain-driven-design/aggregates/pack-task) - Domain model
- [Packing Workflow](/architecture/sequence-diagrams/packing-workflow) - Workflow
