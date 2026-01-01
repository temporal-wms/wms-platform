---
sidebar_position: 1
slug: /
---

# WMS Platform

Welcome to the **Warehouse Management System (WMS) Platform** documentation. This comprehensive guide covers the architecture, design decisions, and implementation details of a production-ready warehouse management system built with microservices.

## Overview

The WMS Platform is an enterprise-grade warehouse management system designed to handle the complete order fulfillment lifecycle:

```
Order → Validation → Waving → Routing → Picking → Consolidation → Packing → Shipping
```

## Key Features

- **9 Domain-Driven Microservices** - Each service owns its bounded context with rich aggregate models
- **Temporal Workflow Orchestration** - Reliable, durable execution of complex business processes
- **Event-Driven Architecture** - 58 CloudEvent types across 11 Kafka topics
- **Saga Pattern** - Distributed transactions with automatic compensation
- **Circuit Breakers** - Resilient inter-service communication
- **Full Observability** - OpenTelemetry tracing, Prometheus metrics, structured logging

## Technology Stack

| Category | Technology |
|----------|------------|
| **Language** | Go 1.24 |
| **Web Framework** | Gin |
| **Database** | MongoDB 6.0+ |
| **Message Broker** | Apache Kafka 3.0+ |
| **Workflow Engine** | Temporal |
| **Tracing** | OpenTelemetry, Jaeger |
| **Metrics** | Prometheus |
| **Container** | Docker, Kubernetes |

## Services Overview

| Service | Port | Description | Aggregate Root |
|---------|------|-------------|----------------|
| Order Service | 8001 | Order lifecycle management | Order |
| Waving Service | 8002 | Batch order grouping for picking | Wave |
| Routing Service | 8003 | Pick path optimization | PickRoute |
| Picking Service | 8004 | Warehouse picking operations | PickTask |
| Consolidation Service | 8005 | Multi-item order combining | ConsolidationUnit |
| Packing Service | 8006 | Package preparation and labeling | PackTask |
| Shipping Service | 8007 | Carrier integration and SLAM | Shipment |
| Inventory Service | 8008 | Stock management | InventoryItem |
| Labor Service | 8009 | Workforce management | Worker |

## Documentation Structure

This documentation is organized into the following sections:

### Architecture
- **C4 Diagrams** - System Context, Container, Component, and Code level diagrams
- **System Diagrams** - Infrastructure, deployment, and data flow diagrams
- **Sequence Diagrams** - Workflow interactions and process flows

### Domain-Driven Design
- **Bounded Contexts** - Strategic domain decomposition
- **Context Map** - Relationships between contexts
- **Aggregates** - Tactical patterns and domain models
- **Domain Events** - Event catalog and flows

### Services
Detailed documentation for each of the 10 services including:
- API endpoints
- Domain models
- Events published/consumed
- Configuration options

### API Reference
- REST API specifications
- Event API (AsyncAPI)

### Infrastructure
- MongoDB configuration
- Kafka setup and topics
- Temporal workflows
- Observability stack

## Quick Start

### Prerequisites

- Go 1.24+
- Docker & Docker Compose
- Kubernetes (for production deployment)
- Make

### Running Locally

```bash
# Start infrastructure
make docker-infra

# Create Kafka topics
make kafka-create-topics

# Create Temporal namespace
make temporal-namespace

# Run all services
make run-all
```

### Testing the API

```bash
curl -X POST http://localhost:8001/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customerId": "CUST-001",
    "priority": "standard",
    "items": [
      {"sku": "SKU-001", "quantity": 2, "price": 29.99}
    ],
    "shippingAddress": {
      "street": "123 Main St",
      "city": "New York",
      "state": "NY",
      "zipCode": "10001",
      "country": "US"
    }
  }'
```

## Observability Endpoints

All services expose:
- `GET /health` - Liveness probe
- `GET /ready` - Readiness probe
- `GET /metrics` - Prometheus metrics

### Dashboards

- **Grafana** - Metrics visualization
- **Jaeger/Tempo** - Distributed tracing
- **Temporal UI** - Workflow monitoring
- **Kafka UI** - Event monitoring
