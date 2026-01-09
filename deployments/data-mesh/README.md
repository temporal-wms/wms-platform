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

### Core Data Products

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

### Extended Data Products (v2.0)

| Data Product | Owner | Source | Freshness SLA | Quality SLA |
|--------------|-------|--------|---------------|-------------|
| `billing-dp` | Finance Team | `wms.billing_activities`, `wms.invoices` | < 5 min | 99.9% completeness |
| `seller-dp` | Business Ops Team | `wms.sellers` | < 15 min | 99.9% completeness |
| `channel-dp` | Integrations Team | `wms.channels` | < 5 min | 99.9% completeness |
| `facility-dp` | Facility Ops Team | `wms.stations` | < 5 min | 99.9% completeness |
| `sortation-dp` | Operations Team | `wms.sortation_batches` | < 5 min | 99.9% completeness |
| `walling-dp` | Operations Team | `wms.wall_assignments` | < 5 min | 99.9% completeness |
| `unit-dp` | Inventory Team | `wms.units` | < 1 min | 99.99% accuracy |
| `process-path-dp` | Engineering Team | `wms.process_paths` | < 15 min | 99.9% completeness |
| `wes-dp` | Engineering Team | `wms.wes_stages` | < 5 min | 99.9% completeness |

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
  - `bronze.billing_activities_raw` (v2.0)
  - `bronze.invoices_raw` (v2.0)
  - `bronze.sellers_raw` (v2.0)
  - `bronze.channels_raw` (v2.0)
  - `bronze.stations_raw` (v2.0)
  - `bronze.sortation_batches_raw` (v2.0)
  - `bronze.wall_assignments_raw` (v2.0)
  - `bronze.units_raw` (v2.0)
  - `bronze.process_paths_raw` (v2.0)
  - `bronze.wes_stages_raw` (v2.0)

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
  - `silver.billing_activities_current` - Billable activities with seller info (v2.0)
  - `silver.invoices_current` - Invoice state with line items (v2.0)
  - `silver.sellers_current` - Seller profiles with contracts (v2.0)
  - `silver.channels_current` - Channel connections with sync status (v2.0)
  - `silver.stations_current` - Station state with worker assignments (v2.0)
  - `silver.sortation_batches_current` - Batch progress with package counts (v2.0)
  - `silver.wall_assignments_current` - Put wall assignments (v2.0)
  - `silver.units_current` - Unit tracking with location (v2.0)
  - `silver.process_paths_current` - Active process paths (v2.0)
  - `silver.wes_stages_current` - WES stage status (v2.0)

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
  - `gold.billing_metrics_daily` - Revenue by activity type and seller (v2.0)
  - `gold.seller_performance_daily` - Seller KPIs and fees (v2.0)
  - `gold.channel_sync_metrics_daily` - Channel health and sync rates (v2.0)
  - `gold.facility_utilization_daily` - Station utilization by zone (v2.0)
  - `gold.sortation_metrics_daily` - Sort rates and dispatch metrics (v2.0)
  - `gold.wms_benchmarks_daily` - Industry benchmark comparisons (v2.0)

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
| `billing_dp_quality` | `expectations/billing_dp.json` | Valid activity types, non-negative amounts, seller reference (v2.0) |
| `seller_dp_quality` | `expectations/seller_dp.json` | Unique seller_id, valid status/tier, contract date ordering (v2.0) |
| `channel_dp_quality` | `expectations/channel_dp.json` | Valid platforms (shopify/amazon/ebay), sync status validation (v2.0) |
| `facility_dp_quality` | `expectations/facility_dp.json` | Valid station types, zone format, capacity constraints (v2.0) |
| `sortation_dp_quality` | `expectations/sortation_dp.json` | Sorted count <= total packages, valid batch status (v2.0) |
| `walling_dp_quality` | `expectations/walling_dp.json` | Valid wall/slot assignments, status validation (v2.0) |
| `unit_dp_quality` | `expectations/unit_dp.json` | Unique unit_id, valid status, license plate format (v2.0) |
| `process_path_dp_quality` | `expectations/process_path_dp.json` | Valid path types, step ordering, handling requirements (v2.0) |
| `wes_dp_quality` | `expectations/wes_dp.json` | Valid stage types, status transitions, route validation (v2.0) |

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

## WMS Industry Benchmarks

The data mesh includes industry-standard WMS benchmarks for performance tracking and comparison.

### Operational KPIs

| Metric | Target | Best-in-Class | Dashboard Color Coding |
|--------|--------|---------------|------------------------|
| **Picks per Hour** | 175 | 250+ | Green: ≥200, Yellow: ≥150, Red: <150 |
| **Order Accuracy** | 99.0% | 99.9% | Green: ≥99.9%, Yellow: ≥99%, Red: <99% |
| **On-time Shipments** | 98% | 99%+ | Green: ≥99%, Yellow: ≥98%, Red: <98% |
| **Dock-to-Stock Time** | <4 hours | <2 hours | Green: ≤2h, Yellow: ≤4h, Red: >4h |
| **Space Utilization** | 80-85% | 90%+ | Green: ≥85%, Yellow: ≥75%, Red: <75% |

### Financial KPIs

| Metric | Target | Best-in-Class | Dashboard Color Coding |
|--------|--------|---------------|------------------------|
| **Cost per Order** | $3.00-$5.00 | <$3.00 | Green: ≤$3, Yellow: ≤$5, Red: >$5 |
| **Pick & Pack Fee** | $1.50-$2.50/order | N/A | Informational |
| **Labor as % of OpEx** | 50-70% | <50% | Green: ≤50%, Yellow: ≤60%, Red: >60% |
| **Fulfillment % of Revenue** | 5-8% | <5% | Green: ≤5%, Yellow: ≤8%, Red: >8% |

### Benchmark Sources
- [Hopstack - 38 Warehouse KPIs](https://www.hopstack.io/blog/warehouse-metrics-kpis)
- [DataDocks - 7 Warehouse KPIs](https://datadocks.com/posts/warehouse-kpis)
- [FCBco - Fulfillment Cost Analysis](https://www.fcbco.com/blog/calculate-fulfillment-cost-per-order)

---

## Superset Dashboards

Apache Superset dashboards provide visual analytics for all data products.

### Dashboard Inventory

| Dashboard | Slug | Description | Key Metrics |
|-----------|------|-------------|-------------|
| **Billing Analytics** | `billing-analytics` | Financial performance with revenue breakdown | Revenue, Cost per Order, Activity Distribution |
| **Seller Performance** | `seller-performance` | Seller KPIs and tier analysis | Orders, Revenue, Storage/Fulfillment Fees |
| **Channel Integration** | `channel-integration` | E-commerce channel health and sync status | Sync Success Rate, Orders Imported, Platform Distribution |
| **Facility Operations** | `facility-operations` | Station utilization and equipment status | Utilization %, Active Stations, Tasks by Zone |
| **Sortation Performance** | `sortation-performance` | Package sorting and dispatch metrics | Packages/Hour, Sort Rate, Carrier Analysis |
| **WMS Benchmarks** | `wms-benchmarks` | Executive dashboard with industry benchmark comparisons | All KPIs vs Targets, Trend Analysis |

### Dashboard Features
- **Native Filters**: Date range, facility, status, and domain-specific filters
- **Conditional Formatting**: Color-coded cells based on benchmark thresholds
- **Reference Lines**: Target and best-in-class benchmarks on trend charts
- **Interactive Drill-down**: Click through from KPIs to detailed tables

### Dataset Configuration

Datasets are configured in `superset/datasets/` with:
- Trino connection to Iceberg Gold layer tables
- Pre-defined metrics with SQL expressions
- Column metadata and formatting

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
│       ├── bronze-ingestion.sql           # CDC → Bronze (all data products)
│       ├── silver-transformation.sql      # Bronze → Silver (core)
│       ├── gold-aggregation.sql           # Silver → Gold (core)
│       └── silver-gold-new-products.sql   # Silver/Gold for v2.0 data products
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
│   │   ├── shipping_dp.json           # Shipping data quality rules
│   │   ├── billing_dp.json            # Billing data quality rules (v2.0)
│   │   ├── seller_dp.json             # Seller data quality rules (v2.0)
│   │   ├── channel_dp.json            # Channel data quality rules (v2.0)
│   │   ├── facility_dp.json           # Facility data quality rules (v2.0)
│   │   ├── sortation_dp.json          # Sortation data quality rules (v2.0)
│   │   ├── walling_dp.json            # Walling data quality rules (v2.0)
│   │   ├── unit_dp.json               # Unit data quality rules (v2.0)
│   │   ├── process_path_dp.json       # Process path data quality rules (v2.0)
│   │   └── wes_dp.json                # WES data quality rules (v2.0)
│   └── checkpoints/
├── superset/
│   ├── datasets/
│   │   ├── billing_metrics.yaml       # Billing metrics dataset (v2.0)
│   │   ├── seller_performance.yaml    # Seller performance dataset (v2.0)
│   │   ├── channel_metrics.yaml       # Channel metrics dataset (v2.0)
│   │   ├── facility_utilization.yaml  # Facility utilization dataset (v2.0)
│   │   ├── sortation_metrics.yaml     # Sortation metrics dataset (v2.0)
│   │   └── wms_benchmarks.yaml        # WMS benchmarks dataset (v2.0)
│   └── dashboards/
│       ├── billing_analytics.yaml     # Billing analytics dashboard (v2.0)
│       ├── seller_performance.yaml    # Seller performance dashboard (v2.0)
│       ├── channel_integration.yaml   # Channel integration dashboard (v2.0)
│       ├── facility_operations.yaml   # Facility operations dashboard (v2.0)
│       ├── sortation_performance.yaml # Sortation performance dashboard (v2.0)
│       └── wms_benchmarks.yaml        # WMS benchmarks dashboard (v2.0)
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
