---
sidebar_position: 1
---

# OpenAPI Specifications

All WMS Platform REST APIs are documented using OpenAPI 3.0.3. Each service exposes a RESTful API with consistent conventions.

## API Standards

| Standard | Value |
|----------|-------|
| **OpenAPI Version** | 3.0.3 |
| **API Version** | 1.0.0 |
| **Base Path** | `/api/v1` |
| **Content-Type** | `application/json` |

## Service Specifications

### Core Fulfillment

| Service | Port | Spec | Endpoints | Description |
|---------|------|------|-----------|-------------|
| Order Service | 8001 | [openapi.yaml](./order-service.yaml) | 15 | Order lifecycle, reprocessing, DLQ |
| Waving Service | 8002 | [openapi.yaml](./waving-service.yaml) | 20+ | Wave planning, scheduling, optimization |
| WES Service | 8016 | [openapi.yaml](./wes-service.yaml) | 12 | Execution plans, task routes, stages |
| Orchestrator | 30010 | [openapi.yaml](./orchestrator.yaml) | 8 | Workflow signals, health |

### Warehouse Operations

| Service | Port | Spec | Endpoints | Description |
|---------|------|------|-----------|-------------|
| Routing Service | 8003 | [openapi.yaml](./routing-service.yaml) | 17 | Route calculation, strategies |
| Picking Service | 8004 | [openapi.yaml](./picking-service.yaml) | 13 | Pick tasks, exceptions |
| Walling Service | 8017 | [openapi.yaml](./walling-service.yaml) | 8 | Put-wall sorting, bin management |
| Consolidation Service | 8005 | [openapi.yaml](./consolidation-service.yaml) | 13 | Multi-item consolidation |
| Packing Service | 8006 | [openapi.yaml](./packing-service.yaml) | 13 | Packaging, labeling |
| Shipping Service | 8007 | [openapi.yaml](./shipping-service.yaml) | 13 | SLAM, carrier integration |

### Inventory & Inbound

| Service | Port | Spec | Endpoints | Description |
|---------|------|------|-----------|-------------|
| Inventory Service | 8008 | [openapi.yaml](./inventory-service.yaml) | 12 | Stock management, reservations |
| Receiving Service | 8011 | [openapi.yaml](./receiving-service.yaml) | 9 | ASN, inbound receiving |
| Stow Service | 8012 | [openapi.yaml](./stow-service.yaml) | 12 | Putaway tasks, storage strategies |

### Infrastructure & Support

| Service | Port | Spec | Endpoints | Description |
|---------|------|------|-----------|-------------|
| Labor Service | 8009 | [openapi.yaml](./labor-service.yaml) | 20 | Workers, shifts, productivity |
| Facility Service | 8010 | [openapi.yaml](./facility-service.yaml) | 20 | Stations, zones, capabilities |
| Sortation Service | 8013 | [openapi.yaml](./sortation-service.yaml) | 9 | Package sortation, dispatch |

## Shared Components

- [common-components.yaml](./common-components.yaml) - Reusable schemas, parameters, and responses

## Common Patterns

### Request Headers

```yaml
X-WMS-Correlation-ID:
  description: Distributed tracing correlation ID
  schema:
    type: string
    format: uuid

Authorization:
  description: Bearer token for authentication
  schema:
    type: string
    pattern: "^Bearer .+"
```

### Standard Error Response

```yaml
ErrorResponse:
  type: object
  properties:
    error:
      type: string
      description: Error message
    code:
      type: string
      description: Error code
    details:
      type: object
      description: Additional error details
    timestamp:
      type: string
      format: date-time
```

### Pagination

```yaml
PaginatedResponse:
  type: object
  properties:
    data:
      type: array
      items: {}
    meta:
      type: object
      properties:
        page:
          type: integer
        pageSize:
          type: integer
        totalItems:
          type: integer
        totalPages:
          type: integer
```

## Tags by Service

Each service organizes endpoints using consistent tags:

| Tag Pattern | Description |
|-------------|-------------|
| `{Resource}` | Primary resource operations (Orders, Tasks, Routes) |
| `Queries` | Read-only query operations |
| `Operations` | State-changing operations |
| `Health` | Health and readiness checks |

## Viewing Specifications

You can view these specifications using:

- **Swagger UI**: Load the YAML file into [Swagger Editor](https://editor.swagger.io/)
- **Redoc**: Generate documentation with [Redoc](https://redocly.github.io/redoc/)
- **Postman**: Import the YAML file into Postman collections
