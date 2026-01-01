# WMS Platform - Data Mesh Architecture

A complete Data Mesh implementation for the WMS (Warehouse Management System) ecosystem using 100% open-source tools. This architecture enables self-serve analytics, data discovery, and cross-domain insights **without modifying existing WMS services**.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Open-Source Tool Stack](#open-source-tool-stack)
- [Data Products](#data-products)
- [Data Layers (Medallion Architecture)](#data-layers-medallion-architecture)
- [Deployment](#deployment)
- [Usage](#usage)
- [Data Quality](#data-quality)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)

---

## Overview

### What is Data Mesh?

Data Mesh is a decentralized data architecture that treats data as a product, with domain teams owning their data end-to-end. This implementation applies Data Mesh principles to the WMS platform:

| Principle | Implementation |
|-----------|---------------|
| **Domain Ownership** | Each WMS service team owns their data product |
| **Data as a Product** | Iceberg tables with defined schemas, quality SLAs, and documentation |
| **Self-Serve Platform** | Trino for SQL queries, OpenMetadata for discovery |
| **Federated Governance** | Great Expectations for contracts, OpenMetadata for policies |

### Key Features

- **Zero Code Changes** - Uses CDC (Debezium) to capture MongoDB changes without modifying services
- **Real-time Data Pipeline** - Kafka + Flink for streaming transformations
- **Unified Query Layer** - Trino enables SQL across all data sources
- **Data Catalog** - OpenMetadata provides discovery, lineage, and governance
- **Data Quality** - Great Expectations validates data contracts

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                              DATA MESH PLATFORM                                      │
├─────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                      │
│  ┌─────────────────────────────────────────────────────────────────────────────┐    │
│  │                        DATA CATALOG & DISCOVERY                              │    │
│  │                           (OpenMetadata)                                     │    │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐          │    │
│  │  │ Orders   │ │Inventory │ │  Waves   │ │ Picking  │ │ Shipping │ ...      │    │
│  │  │ Product  │ │ Product  │ │ Product  │ │ Product  │ │ Product  │          │    │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘          │    │
│  └─────────────────────────────────────────────────────────────────────────────┘    │
│                                       │                                              │
│  ┌────────────────────────────────────┼────────────────────────────────────────┐    │
│  │                     DATA QUALITY & GOVERNANCE                                │    │
│  │                    (Great Expectations + OpenMetadata)                       │    │
│  └────────────────────────────────────┼────────────────────────────────────────┘    │
│                                       │                                              │
│  ┌────────────────────────────────────┼────────────────────────────────────────┐    │
│  │                      SELF-SERVE DATA PLATFORM                                │    │
│  │                                                                              │    │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐              │    │
│  │  │   CDC Layer     │  │  Stream Layer   │  │   Query Layer   │              │    │
│  │  │   (Debezium)    │  │ (Kafka+Flink)   │  │    (Trino)      │              │    │
│  │  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘              │    │
│  │           │                    │                    │                        │    │
│  │  ┌────────┴────────────────────┴────────────────────┴────────┐              │    │
│  │  │                    DATA LAKEHOUSE                          │              │    │
│  │  │              (Apache Iceberg + MinIO)                      │              │    │
│  │  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐          │              │    │
│  │  │  │ Bronze  │→│ Silver  │→│  Gold   │→│Semantic │          │              │    │
│  │  │  │ (Raw)   │ │(Cleaned)│ │(Curated)│ │ (Views) │          │              │    │
│  │  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘          │              │    │
│  │  └────────────────────────────────────────────────────────────┘              │    │
│  └──────────────────────────────────────────────────────────────────────────────┘    │
│                                                                                      │
│  ┌──────────────────────────────────────────────────────────────────────────────┐   │
│  │                         EXISTING WMS PLATFORM                                 │   │
│  │  ┌────────────────────────────────────────────────────────────────────────┐  │   │
│  │  │                     Kafka (Event Streams)                               │  │   │
│  │  │  wms.orders.events │ wms.inventory.events │ wms.picking.events │ ...   │  │   │
│  │  └────────────────────────────────────────────────────────────────────────┘  │   │
│  │  ┌────────────────────────────────────────────────────────────────────────┐  │   │
│  │  │                     MongoDB (Operational Data)                          │  │   │
│  │  │  orders │ inventory │ waves │ pick_tasks │ shipments │ workers │ ...   │  │   │
│  │  └────────────────────────────────────────────────────────────────────────┘  │   │
│  └──────────────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

### Data Flow

```
MongoDB Collections ──┐
                      ├──→ Debezium CDC ──→ cdc.wms.* topics ──┐
Kafka Event Topics ───┘                                         │
                                                                ▼
                                                         Apache Flink
                                                                │
                    ┌───────────────────────────────────────────┼───────────────────┐
                    ▼                                           ▼                   ▼
              Bronze Layer                               Silver Layer          Gold Layer
            (Raw CDC Events)                           (Cleaned Data)      (Business Metrics)
                    │                                           │                   │
                    └───────────────────────────────────────────┴───────────────────┘
                                                                │
                                                                ▼
                                                    ┌───────────────────┐
                                                    │       Trino       │
                                                    │  (Federated SQL)  │
                                                    └───────────────────┘
                                                                │
                                    ┌───────────────────────────┼───────────────────┐
                                    ▼                           ▼                   ▼
                               Dashboards                  ML Models            Ad-hoc Queries
```

---

## Open-Source Tool Stack

| Component | Tool | Version | Port | Purpose |
|-----------|------|---------|------|---------|
| **Data Catalog** | [OpenMetadata](https://open-metadata.org) | 1.3.1 | 30585 | Discovery, lineage, governance |
| **CDC** | [Debezium](https://debezium.io) | 2.5.0 | - | MongoDB change capture |
| **Stream Processing** | [Apache Flink](https://flink.apache.org) | 1.18 | - | Real-time transformations |
| **Object Storage** | [MinIO](https://min.io) | Latest | 30900/30901 | S3-compatible lakehouse storage |
| **Table Format** | [Apache Iceberg](https://iceberg.apache.org) | Latest | - | ACID transactions, time travel |
| **Query Engine** | [Trino](https://trino.io) | 436 | 30808 | Federated SQL queries |
| **Metastore** | [Hive Metastore](https://hive.apache.org) | 4.0.0 | 9083 | Iceberg catalog backend |
| **Data Quality** | [Great Expectations](https://greatexpectations.io) | 0.18.0 | - | Data contract validation |

---

## Data Products

Each WMS domain is exposed as a Data Product with defined ownership, SLAs, and quality expectations.

| Data Product | Owner | Source | Freshness SLA | Quality SLA |
|--------------|-------|--------|---------------|-------------|
| `orders-dp` | Order Team | `wms.orders.events`, MongoDB | < 5 min | 99.9% completeness |
| `inventory-dp` | Inventory Team | `wms.inventory.events`, MongoDB | < 1 min | 99.99% accuracy |
| `waves-dp` | Operations | `wms.waves.events`, MongoDB | < 5 min | 99.9% completeness |
| `picking-dp` | Operations | `wms.picking.events`, MongoDB | < 5 min | 99.9% completeness |
| `packing-dp` | Operations | `wms.packing.events`, MongoDB | < 5 min | 99.9% completeness |
| `shipping-dp` | Logistics | `wms.shipping.events`, MongoDB | < 5 min | 99.9% completeness |
| `labor-dp` | HR/Operations | `wms.labor.events`, MongoDB | < 5 min | 99.9% completeness |
| `fulfillment-dp` | Operations | Multiple domains (aggregated) | < 15 min | N/A |

### Data Product Schema

Each data product includes:
- **Schema**: Defined in Iceberg tables with versioning
- **Documentation**: Registered in OpenMetadata
- **Quality Rules**: Defined in Great Expectations
- **Lineage**: Automatically tracked from source to consumption

---

## Data Layers (Medallion Architecture)

### Bronze Layer (Raw)
- **Location**: `s3://wms-bronze/`
- **Content**: Raw CDC events from Debezium
- **Retention**: 90 days
- **Tables**:
  - `bronze.orders_raw`
  - `bronze.inventory_raw`
  - `bronze.waves_raw`
  - `bronze.pick_tasks_raw`
  - `bronze.pack_tasks_raw`
  - `bronze.shipments_raw`
  - `bronze.workers_raw`

### Silver Layer (Cleaned)
- **Location**: `s3://wms-silver/`
- **Content**: Deduplicated, validated, enriched data
- **Retention**: 365 days
- **Tables**:
  - `silver.orders_current` - Latest state per order
  - `silver.inventory_current` - Current stock levels
  - `silver.pick_tasks_enriched` - With order and worker data
  - `silver.shipments_current` - With carrier details
  - `silver.workers_current` - With performance metrics

### Gold Layer (Curated)
- **Location**: `s3://wms-gold/`
- **Content**: Business metrics and KPIs
- **Retention**: Indefinite (archived after 1 year)
- **Tables**:
  - `gold.order_fulfillment_daily` - Daily fulfillment metrics
  - `gold.inventory_metrics_daily` - Stock turnover
  - `gold.labor_productivity_daily` - Worker performance
  - `gold.shipping_performance_daily` - Carrier metrics
  - `gold.wave_performance_daily` - Wave efficiency
  - `gold.active_operations_snapshot` - Real-time dashboard

---

## Deployment

### Prerequisites

- Kubernetes cluster (Kind, Minikube, or production cluster)
- kubectl configured
- Helm 3.x installed
- Existing WMS platform deployed (Kafka, MongoDB)

### Quick Start

```bash
# Navigate to data-mesh directory
cd wms-platform/deployments/data-mesh

# Deploy all components
./scripts/deploy-data-mesh.sh

# Verify deployment
./scripts/setup-connectors.sh
```

### Component-by-Component Deployment

```bash
# Deploy only MinIO
./scripts/deploy-data-mesh.sh minio

# Deploy only Trino
./scripts/deploy-data-mesh.sh trino

# Deploy only Debezium
./scripts/deploy-data-mesh.sh debezium

# Deploy only Flink
./scripts/deploy-data-mesh.sh flink

# Deploy only OpenMetadata
./scripts/deploy-data-mesh.sh openmetadata
```

### Access Points

| Component | URL | Credentials |
|-----------|-----|-------------|
| MinIO Console | http://localhost:30900 | admin / minio123456 |
| MinIO API | http://localhost:30901 | datamesh / datamesh123456 |
| Trino UI | http://localhost:30808 | - |
| OpenMetadata | http://localhost:30585 | admin / admin |

### Port Forwarding (if NodePort not available)

```bash
# MinIO
kubectl port-forward -n data-mesh svc/minio 9000:9000 9001:9001

# Trino
kubectl port-forward -n data-mesh svc/trino 8080:8080

# OpenMetadata
kubectl port-forward -n data-mesh svc/openmetadata 8585:8585
```

---

## Usage

### Querying Data with Trino

```bash
# Connect to Trino
kubectl exec -it -n data-mesh deploy/trino-coordinator -- trino

# Or via CLI
trino --server localhost:30808 --catalog iceberg
```

#### Example Queries

```sql
-- List available schemas
SHOW SCHEMAS FROM iceberg;

-- View order fulfillment metrics
SELECT
    date,
    priority,
    total_orders,
    completed_orders,
    completion_rate,
    avg_fulfillment_time_hours
FROM iceberg.gold.order_fulfillment_daily
WHERE date >= CURRENT_DATE - INTERVAL '7' DAY
ORDER BY date DESC;

-- Cross-domain query: Orders with inventory and shipping
SELECT
    o.order_id,
    o.customer_id,
    o.priority,
    o.status,
    s.tracking_number,
    s.carrier_code,
    s.shipped_at
FROM iceberg.silver.orders_current o
LEFT JOIN iceberg.silver.shipments_current s ON o.order_id = s.order_id
WHERE o.created_at >= CURRENT_DATE - INTERVAL '1' DAY
LIMIT 100;

-- Real-time operational metrics
SELECT * FROM iceberg.gold.active_operations_snapshot
ORDER BY snapshot_time DESC
LIMIT 1;

-- Query operational MongoDB directly
SELECT order_id, status, priority
FROM mongodb.wms.orders
WHERE status = 'picking'
LIMIT 10;

-- Query Kafka events
SELECT
    _message,
    _timestamp
FROM kafka.default."wms.orders.events"
LIMIT 10;
```

### Data Discovery with OpenMetadata

1. Access OpenMetadata UI: http://localhost:30585
2. Browse Data Products under "Explore"
3. View data lineage for any table
4. Check data quality results
5. Search for datasets using keywords

### Submitting Flink Jobs

```bash
# Submit Bronze ingestion job
kubectl exec -n data-mesh flink-jobmanager-0 -- \
    flink run -d /opt/flink/jobs/bronze-ingestion.jar

# Submit Silver transformation job
kubectl exec -n data-mesh flink-jobmanager-0 -- \
    flink run -d /opt/flink/jobs/silver-transformation.jar

# Submit Gold aggregation job
kubectl exec -n data-mesh flink-jobmanager-0 -- \
    flink run -d /opt/flink/jobs/gold-aggregation.jar
```

---

## Data Quality

### Great Expectations Suites

Data quality is enforced using Great Expectations with suites defined per data product:

| Suite | File | Key Validations |
|-------|------|-----------------|
| `orders_dp_quality` | `expectations/orders_dp.json` | Unique order_id, valid status/priority, timestamp ordering |
| `inventory_dp_quality` | `expectations/inventory_dp.json` | Non-negative quantities, unique SKU, location format |
| `shipping_dp_quality` | `expectations/shipping_dp.json` | Valid carrier codes, tracking number format, date ordering |

### Running Quality Checks

```bash
# Run quality checks via Airflow DAG
kubectl exec -n data-mesh airflow-scheduler-0 -- \
    airflow dags trigger data_quality_checks

# Or run manually
great_expectations checkpoint run orders_dp_checkpoint
```

### Quality Metrics in OpenMetadata

Data quality results are automatically pushed to OpenMetadata:
1. Navigate to a table in OpenMetadata
2. Click "Quality" tab
3. View test results and trends

---

## Monitoring

### Grafana Dashboards

The data mesh integrates with existing Prometheus/Grafana stack:

- **Debezium Metrics**: CDC lag, events captured, errors
- **Flink Metrics**: Job status, throughput, checkpoints
- **Trino Metrics**: Query latency, active queries, memory usage
- **MinIO Metrics**: Storage usage, request rates

### Key Metrics to Monitor

| Metric | Description | Alert Threshold |
|--------|-------------|-----------------|
| `debezium_mongodb_streaming_seconds_behind_source` | CDC lag | > 60s |
| `flink_jobmanager_job_uptime` | Job health | Job down |
| `trino_query_execution_time_seconds` | Query performance | p99 > 30s |
| `minio_bucket_usage_total_bytes` | Storage usage | > 80% capacity |

---

## Troubleshooting

### Debezium Not Capturing Changes

```bash
# Check connector status
kubectl exec -n kafka wms-kafka-connect-0 -- \
    curl -s http://localhost:8083/connectors/mongodb-cdc-connector/status

# View connector logs
kubectl logs -n kafka wms-kafka-connect-0 -f

# Restart connector
kubectl exec -n kafka wms-kafka-connect-0 -- \
    curl -X POST http://localhost:8083/connectors/mongodb-cdc-connector/restart
```

### Flink Job Failures

```bash
# Check Flink job status
kubectl get flinkdeployment -n data-mesh

# View job manager logs
kubectl logs -n data-mesh -l component=jobmanager -f

# Check savepoints
kubectl exec -n data-mesh flink-jobmanager-0 -- \
    flink list -a
```

### Trino Query Errors

```bash
# Check Trino coordinator logs
kubectl logs -n data-mesh -l app=trino,component=coordinator -f

# Verify catalog connectivity
trino --execute "SELECT * FROM system.runtime.nodes"

# Test Iceberg catalog
trino --execute "SHOW SCHEMAS FROM iceberg"
```

### OpenMetadata Ingestion Issues

```bash
# Check OpenMetadata logs
kubectl logs -n data-mesh -l app=openmetadata -f

# Verify Elasticsearch
kubectl exec -n data-mesh openmetadata-elasticsearch-0 -- \
    curl -s http://localhost:9200/_cluster/health

# Re-run ingestion
# Use OpenMetadata UI > Settings > Services > Run Ingestion
```

---

## Directory Structure

```
deployments/data-mesh/
├── namespace.yaml                     # Kubernetes namespace + quotas
├── README.md                          # This file
├── debezium/
│   ├── kafka-connect-cluster.yaml     # Kafka Connect cluster
│   └── mongodb-connector.yaml         # MongoDB CDC connector + topics
├── minio/
│   ├── values.yaml                    # MinIO Helm values
│   └── buckets.yaml                   # Bucket lifecycle policies
├── flink/
│   ├── flink-cluster.yaml             # Flink Kubernetes deployment
│   └── jobs/
│       ├── bronze-ingestion.sql       # CDC → Bronze
│       ├── silver-transformation.sql  # Bronze → Silver
│       └── gold-aggregation.sql       # Silver → Gold
├── trino/
│   ├── values.yaml                    # Trino Helm values
│   ├── hive-metastore.yaml            # Hive Metastore for Iceberg
│   └── catalogs/
│       ├── iceberg.properties         # Iceberg catalog config
│       ├── mongodb.properties         # MongoDB catalog config
│       └── kafka.properties           # Kafka catalog config
├── openmetadata/
│   ├── values.yaml                    # OpenMetadata Helm values
│   └── ingestion/
│       ├── kafka-ingestion.yaml       # Kafka topic discovery
│       ├── iceberg-ingestion.yaml     # Lakehouse table discovery
│       └── mongodb-ingestion.yaml     # Operational DB discovery
├── great-expectations/
│   ├── expectations/
│   │   ├── orders_dp.json             # Order data quality rules
│   │   ├── inventory_dp.json          # Inventory data quality rules
│   │   └── shipping_dp.json           # Shipping data quality rules
│   └── checkpoints/
├── airflow/
│   ├── values.yaml
│   └── dags/
│       ├── data_quality_checks.py
│       └── catalog_refresh.py
└── scripts/
    ├── deploy-data-mesh.sh            # Full deployment script
    └── setup-connectors.sh            # Connector verification
```

---

## References

- [Data Mesh Principles](https://martinfowler.com/articles/data-mesh-principles.html) - Zhamak Dehghani
- [Apache Iceberg Documentation](https://iceberg.apache.org/docs/latest/)
- [Trino Documentation](https://trino.io/docs/current/)
- [OpenMetadata Documentation](https://docs.open-metadata.org/)
- [Debezium MongoDB Connector](https://debezium.io/documentation/reference/connectors/mongodb.html)
- [Apache Flink Documentation](https://nightlies.apache.org/flink/flink-docs-stable/)
- [Great Expectations Documentation](https://docs.greatexpectations.io/)

---

## License

This data mesh implementation is part of the WMS Platform project.
