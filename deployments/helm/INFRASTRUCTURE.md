# WMS Platform - Infrastructure Deployment

This document describes how to deploy the infrastructure dependencies (MongoDB, Kafka, Temporal) for the WMS platform using existing Helm charts.

## Overview

The WMS platform requires three main infrastructure components:
1. **MongoDB** - Document database for service persistence
2. **Kafka** - Event streaming platform for async messaging
3. **Temporal** - Workflow orchestration engine

## MongoDB Deployment

We use the Bitnami MongoDB Helm chart with replica set configuration.

### Installation

```bash
# Add Bitnami repository
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# Install MongoDB with replica set
helm install mongodb bitnami/mongodb \
  --namespace wms-platform \
  --create-namespace \
  --set architecture=replicaset \
  --set replicaCount=3 \
  --set auth.enabled=true \
  --set auth.rootPassword=<ROOT_PASSWORD> \
  --set auth.username=wmsuser \
  --set auth.password=<WMS_PASSWORD> \
  --set auth.database=wms \
  --set persistence.enabled=true \
  --set persistence.size=50Gi \
  --set resources.requests.memory=512Mi \
  --set resources.requests.cpu=250m \
  --set resources.limits.memory=1Gi \
  --set resources.limits.cpu=1000m
```

### Values File (values-mongodb.yaml)

```yaml
architecture: replicaset
replicaCount: 3

auth:
  enabled: true
  rootPassword: <ROOT_PASSWORD>
  username: wmsuser
  password: <WMS_PASSWORD>
  database: wms

persistence:
  enabled: true
  size: 50Gi
  storageClass: ""  # Use default storage class

resources:
  requests:
    memory: 512Mi
    cpu: 250m
  limits:
    memory: 1Gi
    cpu: 1000m

arbiter:
  enabled: false

# Metrics for Prometheus
metrics:
  enabled: true
  serviceMonitor:
    enabled: true
    namespace: wms-platform

# Backup configuration
backup:
  enabled: false  # Configure if needed
```

Install with values file:
```bash
helm install mongodb bitnami/mongodb \
  --namespace wms-platform \
  --values values-mongodb.yaml
```

### Connection String

After installation, create the connection secret:

```bash
# Get the MongoDB connection string
export MONGODB_ROOT_PASSWORD=$(kubectl get secret --namespace wms-platform mongodb -o jsonpath="{.data.mongodb-root-password}" | base64 -d)

# Create WMS platform secret
kubectl create secret generic mongodb-credentials \
  --namespace wms-platform \
  --from-literal=uri="mongodb://wmsuser:<WMS_PASSWORD>@mongodb-0.mongodb-headless.wms-platform.svc.cluster.local:27017,mongodb-1.mongodb-headless.wms-platform.svc.cluster.local:27017,mongodb-2.mongodb-headless.wms-platform.svc.cluster.local:27017/wms?replicaSet=rs0"
```

## Kafka Deployment

We use the Bitnami Kafka Helm chart with KRaft mode (no Zookeeper).

### Installation

```bash
# Install Kafka with KRaft
helm install kafka bitnami/kafka \
  --namespace wms-platform \
  --set controller.replicaCount=3 \
  --set broker.replicaCount=3 \
  --set kraft.enabled=true \
  --set persistence.enabled=true \
  --set persistence.size=50Gi \
  --set resources.requests.memory=1Gi \
  --set resources.requests.cpu=500m \
  --set resources.limits.memory=2Gi \
  --set resources.limits.cpu=2000m \
  --set listeners.client.protocol=PLAINTEXT \
  --set listeners.interbroker.protocol=PLAINTEXT
```

### Values File (values-kafka.yaml)

```yaml
# KRaft mode (no Zookeeper)
kraft:
  enabled: true

# Controller configuration
controller:
  replicaCount: 3
  persistence:
    enabled: true
    size: 20Gi

# Broker configuration
broker:
  replicaCount: 3
  persistence:
    enabled: true
    size: 50Gi
  resources:
    requests:
      memory: 1Gi
      cpu: 500m
    limits:
      memory: 2Gi
      cpu: 2000m

# Listeners
listeners:
  client:
    protocol: PLAINTEXT
  controller:
    protocol: PLAINTEXT
  interbroker:
    protocol: PLAINTEXT
  external:
    protocol: PLAINTEXT

# Default number of partitions
defaultReplicationFactor: 3
offsetsTopicReplicationFactor: 3
transactionStateLogReplicationFactor: 3
transactionStateLogMinIsr: 2

# Metrics for Prometheus
metrics:
  kafka:
    enabled: true
  jmx:
    enabled: true
  serviceMonitor:
    enabled: true
    namespace: wms-platform

# Log retention
logRetentionHours: 168  # 7 days
logSegmentBytes: 1073741824  # 1GB
```

Install with values file:
```bash
helm install kafka bitnami/kafka \
  --namespace wms-platform \
  --values values-kafka.yaml
```

### Create Topics

After installation, create the required topics:

```bash
# Connect to Kafka pod
kubectl exec -it kafka-0 -n wms-platform -- bash

# Create topics
kafka-topics.sh --create \
  --bootstrap-server localhost:9092 \
  --topic wms.orders.events \
  --partitions 6 \
  --replication-factor 3 \
  --config retention.ms=604800000 \
  --config compression.type=gzip

kafka-topics.sh --create \
  --bootstrap-server localhost:9092 \
  --topic wms.waves.events \
  --partitions 6 \
  --replication-factor 3

kafka-topics.sh --create \
  --bootstrap-server localhost:9092 \
  --topic wms.routing.events \
  --partitions 6 \
  --replication-factor 3

kafka-topics.sh --create \
  --bootstrap-server localhost:9092 \
  --topic wms.picking.events \
  --partitions 6 \
  --replication-factor 3

kafka-topics.sh --create \
  --bootstrap-server localhost:9092 \
  --topic wms.consolidation.events \
  --partitions 4 \
  --replication-factor 3

kafka-topics.sh --create \
  --bootstrap-server localhost:9092 \
  --topic wms.packing.events \
  --partitions 4 \
  --replication-factor 3

kafka-topics.sh --create \
  --bootstrap-server localhost:9092 \
  --topic wms.shipping.events \
  --partitions 4 \
  --replication-factor 3

kafka-topics.sh --create \
  --bootstrap-server localhost:9092 \
  --topic wms.inventory.events \
  --partitions 6 \
  --replication-factor 3

kafka-topics.sh --create \
  --bootstrap-server localhost:9092 \
  --topic wms.labor.events \
  --partitions 4 \
  --replication-factor 3

# Verify topics
kafka-topics.sh --list --bootstrap-server localhost:9092
```

## Temporal Deployment

We use the official Temporal Helm chart.

### Installation

```bash
# Add Temporal repository
helm repo add temporalio https://go.temporal.io/helm-charts
helm repo update

# Install Temporal
helm install temporal temporalio/temporal \
  --namespace wms-platform \
  --set server.replicaCount=3 \
  --set cassandra.enabled=false \
  --set postgresql.enabled=true \
  --set postgresql.persistence.size=20Gi \
  --set elasticsearch.enabled=true \
  --set prometheus.enabled=true \
  --set grafana.enabled=true
```

### Values File (values-temporal.yaml)

```yaml
# Server configuration
server:
  replicaCount: 3
  resources:
    requests:
      memory: 512Mi
      cpu: 200m
    limits:
      memory: 1Gi
      cpu: 1000m

  config:
    persistence:
      default:
        driver: "sql"
        sql:
          driver: "postgres"
          host: "temporal-postgresql"
          port: 5432
          database: "temporal"
          user: "temporal"
          password: "<POSTGRES_PASSWORD>"
          maxConns: 20
          maxIdleConns: 20

      visibility:
        driver: "sql"
        sql:
          driver: "postgres"
          host: "temporal-postgresql"
          port: 5432
          database: "temporal_visibility"
          user: "temporal"
          password: "<POSTGRES_PASSWORD>"
          maxConns: 20
          maxIdleConns: 20

# PostgreSQL (persistence)
postgresql:
  enabled: true
  persistence:
    enabled: true
    size: 20Gi
  resources:
    requests:
      memory: 512Mi
      cpu: 250m
    limits:
      memory: 1Gi
      cpu: 1000m

# Cassandra (disabled - using PostgreSQL)
cassandra:
  enabled: false

# Elasticsearch (for advanced visibility)
elasticsearch:
  enabled: true
  replicas: 3
  minimumMasterNodes: 2
  persistence:
    enabled: true
    size: 30Gi
  resources:
    requests:
      memory: 1Gi
      cpu: 500m
    limits:
      memory: 2Gi
      cpu: 2000m

# Web UI
web:
  enabled: true
  replicaCount: 1
  service:
    type: ClusterIP
    port: 8080
  ingress:
    enabled: false  # Enable if needed

# Metrics
prometheus:
  enabled: true

grafana:
  enabled: true
```

Install with values file:
```bash
helm install temporal temporalio/temporal \
  --namespace wms-platform \
  --values values-temporal.yaml
```

### Create Namespace

After Temporal is running, create the WMS namespace:

```bash
# Port-forward to Temporal frontend
kubectl port-forward -n wms-platform svc/temporal-frontend 7233:7233

# In another terminal, create namespace using tctl
tctl --namespace wms-platform namespace register

# Verify
tctl --namespace wms-platform namespace describe
```

## Complete Deployment Order

Follow this order to deploy the complete WMS platform:

### 1. Deploy Infrastructure

```bash
# Create namespace
kubectl create namespace wms-platform

# Deploy MongoDB
helm install mongodb bitnami/mongodb \
  --namespace wms-platform \
  --values values-mongodb.yaml

# Wait for MongoDB to be ready
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=mongodb \
  --namespace wms-platform \
  --timeout=300s

# Deploy Kafka
helm install kafka bitnami/kafka \
  --namespace wms-platform \
  --values values-kafka.yaml

# Wait for Kafka to be ready
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=kafka \
  --namespace wms-platform \
  --timeout=300s

# Create Kafka topics (see section above)

# Deploy Temporal
helm install temporal temporalio/temporal \
  --namespace wms-platform \
  --values values-temporal.yaml

# Wait for Temporal to be ready
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=temporal \
  --namespace wms-platform \
  --timeout=600s

# Create Temporal namespace (see section above)
```

### 2. Create Secrets

```bash
# Create MongoDB credentials secret
kubectl create secret generic mongodb-credentials \
  --namespace wms-platform \
  --from-literal=uri="mongodb://wmsuser:<PASSWORD>@mongodb-0.mongodb-headless.wms-platform.svc.cluster.local:27017,mongodb-1.mongodb-headless.wms-platform.svc.cluster.local:27017,mongodb-2.mongodb-headless.wms-platform.svc.cluster.local:27017/wms?replicaSet=rs0"
```

### 3. Deploy WMS Platform

```bash
# Deploy WMS platform services
helm install wms-platform ./deployments/helm/wms-platform \
  --namespace wms-platform \
  --values ./deployments/helm/wms-platform/values-prod.yaml

# Wait for all services to be ready
kubectl wait --for=condition=ready pod -l app.kubernetes.io/instance=wms-platform \
  --namespace wms-platform \
  --timeout=300s
```

### 4. Verify Deployment

```bash
# Check all pods
kubectl get pods -n wms-platform

# Check services
kubectl get svc -n wms-platform

# Check HPA
kubectl get hpa -n wms-platform

# Check PDB
kubectl get pdb -n wms-platform

# Test order service
kubectl port-forward -n wms-platform svc/order-service 8001:8001

# In another terminal
curl http://localhost:8001/health
```

## Resource Requirements

### Minimum Cluster Requirements

| Component | CPU | Memory | Storage |
|-----------|-----|--------|---------|
| MongoDB (3 replicas) | 750m | 1.5Gi | 150Gi |
| Kafka (3 brokers) | 1500m | 3Gi | 150Gi |
| Temporal (3 servers + PostgreSQL) | 1000m | 2Gi | 50Gi |
| WMS Services (10 services, min replicas) | 1500m | 3Gi | - |
| **Total** | **4.75 cores** | **9.5Gi** | **350Gi** |

### Production Cluster Requirements

| Component | CPU | Memory | Storage |
|-----------|-----|--------|---------|
| MongoDB (3 replicas) | 3000m | 3Gi | 150Gi |
| Kafka (3 brokers) | 6000m | 6Gi | 150Gi |
| Temporal (3 servers + dependencies) | 4000m | 6Gi | 90Gi |
| WMS Services (10 services, max replicas) | 8000m | 12Gi | - |
| **Total** | **21 cores** | **27Gi** | **390Gi** |

Recommended Kubernetes cluster:
- **Nodes**: 6-8 nodes
- **Node size**: 4 vCPU, 16Gi RAM each
- **Total cluster**: 24-32 vCPU, 96-128Gi RAM

## Monitoring

### Prometheus

All infrastructure components expose metrics:

```yaml
# MongoDB metrics
mongodb.wms-platform.svc.cluster.local:9216/metrics

# Kafka metrics
kafka-0.wms-platform.svc.cluster.local:9308/metrics

# Temporal metrics
temporal-frontend.wms-platform.svc.cluster.local:9090/metrics
```

### ServiceMonitor

If using Prometheus Operator, ServiceMonitors are created automatically:

```bash
kubectl get servicemonitor -n wms-platform
```

## Backup and Recovery

### MongoDB Backup

```bash
# Manual backup
kubectl exec -it mongodb-0 -n wms-platform -- \
  mongodump --uri="mongodb://wmsuser:<PASSWORD>@localhost:27017/wms?replicaSet=rs0" \
  --out=/tmp/backup

# Copy backup out
kubectl cp wms-platform/mongodb-0:/tmp/backup ./mongodb-backup
```

### Kafka Backup

Kafka topics are configured with retention policies. For critical data:
- Use MirrorMaker 2 for cross-cluster replication
- Enable Kafka Connect for backup to S3/GCS

### Temporal Backup

```bash
# Backup PostgreSQL
kubectl exec -it temporal-postgresql-0 -n wms-platform -- \
  pg_dump -U temporal temporal > temporal-backup.sql
```

## Troubleshooting

### MongoDB Issues

```bash
# Check replica set status
kubectl exec -it mongodb-0 -n wms-platform -- mongosh --eval "rs.status()"

# Check logs
kubectl logs mongodb-0 -n wms-platform
```

### Kafka Issues

```bash
# Check broker status
kubectl exec -it kafka-0 -n wms-platform -- kafka-broker-api-versions.sh \
  --bootstrap-server localhost:9092

# List topics
kubectl exec -it kafka-0 -n wms-platform -- kafka-topics.sh \
  --list --bootstrap-server localhost:9092

# Check consumer groups
kubectl exec -it kafka-0 -n wms-platform -- kafka-consumer-groups.sh \
  --list --bootstrap-server localhost:9092
```

### Temporal Issues

```bash
# Check Temporal health
kubectl port-forward -n wms-platform svc/temporal-frontend 7233:7233

# In another terminal
tctl cluster health

# Check namespace
tctl --namespace wms-platform namespace describe
```

## Cleanup

To remove all infrastructure:

```bash
# Remove WMS platform
helm uninstall wms-platform --namespace wms-platform

# Remove infrastructure
helm uninstall temporal --namespace wms-platform
helm uninstall kafka --namespace wms-platform
helm uninstall mongodb --namespace wms-platform

# Remove namespace (this will delete all PVCs!)
kubectl delete namespace wms-platform
```

⚠️ **Warning**: This will delete all data! Make sure to backup before removing.
