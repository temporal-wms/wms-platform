---
sidebar_position: 0
---

# OpenAPI Specifications

This directory contains OpenAPI 3.0.3 specifications for all WMS Platform REST APIs.

## Specifications

| Service | File | Port | Description |
|---------|------|------|-------------|
| Order Service | [order-service.yaml](./order-service.yaml) | 8001 | Order intake and lifecycle |
| Waving Service | [waving-service.yaml](./waving-service.yaml) | 8002 | Wave management |
| WES Service | [wes-service.yaml](./wes-service.yaml) | 8016 | Warehouse Execution System |
| Walling Service | [walling-service.yaml](./walling-service.yaml) | 8017 | Put-wall sorting |
| Picking Service | [picking-service.yaml](./picking-service.yaml) | 8004 | Pick task execution |
| Packing Service | [packing-service.yaml](./packing-service.yaml) | 8006 | Pack task execution |
| Shipping Service | [shipping-service.yaml](./shipping-service.yaml) | 8007 | Carrier integration |
| Inventory Service | [inventory-service.yaml](./inventory-service.yaml) | 8008 | Stock management |
| Routing Service | [routing-service.yaml](./routing-service.yaml) | 8005 | Route optimization |
| Labor Service | [labor-service.yaml](./labor-service.yaml) | 8009 | Workforce management |
| Consolidation Service | [consolidation-service.yaml](./consolidation-service.yaml) | 8012 | Multi-zone consolidation |
| Facility Service | [facility-service.yaml](./facility-service.yaml) | 8011 | Warehouse layout |
| Receiving Service | [receiving-service.yaml](./receiving-service.yaml) | 8013 | Inbound receiving |
| Stow Service | [stow-service.yaml](./stow-service.yaml) | 8014 | Putaway operations |
| Sortation Service | [sortation-service.yaml](./sortation-service.yaml) | 8015 | Sortation operations |
| Orchestrator | [orchestrator.yaml](./orchestrator.yaml) | 8080 | Workflow triggers |

## OpenAPI Version

All specifications use **OpenAPI 3.0.3** with the following conventions:

- **Servers**: Development (localhost) and Production URLs
- **Tags**: Endpoints grouped by resource type
- **Examples**: Request/response examples for key operations
- **Components**: Reusable schemas, parameters, and headers

## Common Headers

All APIs accept the following correlation headers:

```yaml
X-WMS-Correlation-ID: UUID for distributed tracing
X-WMS-Wave-Number: Wave identifier for batch processing
X-WMS-Workflow-ID: Temporal workflow ID
```

## Using the Specifications

### View in Swagger UI

```bash
docker run -p 8081:8080 -e SWAGGER_JSON=/specs/order-service.yaml \
  -v $(pwd):/specs swaggerapi/swagger-ui
```

### Generate Client SDKs

```bash
# TypeScript
openapi-generator generate -i order-service.yaml -g typescript-axios -o ./client

# Go
openapi-generator generate -i order-service.yaml -g go -o ./client
```

### Validate Specifications

```bash
npx @redocly/cli lint order-service.yaml
```
