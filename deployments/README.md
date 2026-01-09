# WMS Platform - Kubernetes Deployment

Infrastructure as Code for deploying the WMS Platform on a Kind Kubernetes cluster.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) (v20.10+)
- [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) (v0.20+)
- [kubectl](https://kubernetes.io/docs/tasks/tools/) (v1.28+)
- [Helm](https://helm.sh/docs/intro/install/) (v3.12+)

## Quick Start

```bash
# Complete setup (cluster + infrastructure + observability + WMS)
make all

# Or step by step:
make cluster-create    # Create Kind cluster
make infra-deploy      # Deploy MongoDB, Kafka, Temporal
make observability-deploy  # Deploy Grafana, Loki, Tempo
make temporal-namespace    # Create WMS namespace in Temporal
make wms-build        # Build WMS Docker images
make wms-load         # Load images into Kind
make wms-deploy       # Deploy WMS services
```

## Directory Structure

```
deployments/
├── kind/
│   └── kind-config.yaml          # Kind cluster configuration
├── helm/
│   ├── infrastructure/
│   │   ├── mongodb/
│   │   │   └── values-prod.yaml  # Bitnami MongoDB values
│   │   ├── kafka/
│   │   │   ├── strimzi-operator-values.yaml
│   │   │   ├── kafka-cluster.yaml
│   │   │   └── kafka-topics.yaml
│   │   ├── temporal/
│   │   │   ├── postgresql-values.yaml
│   │   │   └── temporal-values.yaml
│   │   └── observability/
│   │       ├── grafana-values.yaml
│   │       ├── loki-values.yaml
│   │       ├── tempo-values.yaml
│   │       └── promtail-values.yaml
│   └── wms-platform/             # WMS services Helm chart
├── scripts/
│   ├── setup.sh                  # Complete setup script
│   ├── deploy-infrastructure.sh
│   ├── deploy-observability.sh
│   └── create-temporal-namespace.sh
├── Makefile
└── README.md
```

## Components

### Infrastructure

| Component | Chart | Namespace | Description |
|-----------|-------|-----------|-------------|
| MongoDB | bitnami/mongodb | mongodb | Document database with replica set |
| Kafka | strimzi/strimzi-kafka-operator | kafka | Kafka cluster with KRaft (no ZooKeeper) |
| PostgreSQL | bitnami/postgresql | temporal | Database for Temporal |
| Temporal | temporalio/temporal | temporal | Workflow orchestration engine |

### Observability

| Component | Chart | Namespace | Description |
|-----------|-------|-----------|-------------|
| Grafana | grafana/grafana | observability | Dashboards and visualization |
| Loki | grafana/loki | observability | Log aggregation |
| Tempo | grafana/tempo | observability | Distributed tracing |
| Promtail | grafana/promtail | observability | Log collection agent |

## Access URLs

After deployment, the following services are accessible:

| Service | URL | Credentials |
|---------|-----|-------------|
| Grafana | http://localhost:3000 | admin / admin |
| Temporal UI | http://localhost:8080 | - |
| Loki API | http://localhost:30310 | - |
| Tempo API | http://localhost:30311 | - |

### WMS Services (NodePorts)

| Service | URL |
|---------|-----|
| Order Service | http://localhost:30001 |
| Waving Service | http://localhost:30002 |
| Routing Service | http://localhost:30003 |
| Picking Service | http://localhost:30004 |
| Consolidation Service | http://localhost:30005 |
| Packing Service | http://localhost:30006 |
| Shipping Service | http://localhost:30007 |
| Inventory Service | http://localhost:30008 |
| Labor Service | http://localhost:30009 |

## Make Targets

### Cluster Management
```bash
make cluster-create   # Create Kind cluster
make cluster-delete   # Delete Kind cluster
make cluster-info     # Show cluster info
```

### Infrastructure
```bash
make infra-deploy     # Deploy all infrastructure
make infra-mongodb    # Deploy MongoDB only
make infra-kafka      # Deploy Kafka only
make infra-temporal   # Deploy Temporal only
make infra-undeploy   # Remove all infrastructure
```

### Observability
```bash
make observability-deploy    # Deploy full observability stack
make observability-loki      # Deploy Loki only
make observability-tempo     # Deploy Tempo only
make observability-grafana   # Deploy Grafana only
make observability-undeploy  # Remove observability stack
```

### Temporal
```bash
make temporal-namespace  # Create WMS namespace in Temporal
make temporal-ui         # Open Temporal UI in browser
```

### WMS Platform
```bash
make wms-build      # Build all WMS Docker images
make wms-load       # Load images into Kind cluster
make wms-deploy     # Deploy WMS platform
make wms-redeploy   # Rebuild and redeploy
make wms-undeploy   # Remove WMS platform
make wms-restart    # Restart all WMS pods
```

### Utilities
```bash
make status              # Show status of all deployments
make logs SVC=order-service  # Show logs for a service
make port-forward        # Set up port forwarding
make grafana             # Open Grafana in browser
make all                 # Complete setup
make clean               # Delete everything
```

## Configuration

### MongoDB

The MongoDB deployment uses a replica set architecture for transaction support:

```yaml
# helm/infrastructure/mongodb/values-prod.yaml
architecture: replicaset
auth:
  username: wmsuser
  password: wmspassword
  database: wms
```

Connection string: `mongodb://wmsuser:wmspassword@mongodb.mongodb.svc.cluster.local:27017/wms?replicaSet=rs0`

### Kafka

Kafka is deployed using Strimzi with KRaft mode (no ZooKeeper):

- Version: 4.0.0
- 3 broker/controller nodes
- Pre-configured topics for WMS events

Topics:
- `wms.orders.inbound`
- `wms.orders.events`
- `wms.waves.events`
- `wms.picking.events`
- `wms.packing.events`
- `wms.shipping.events`
- `wms.inventory.events`
- `wms.labor.events`

Bootstrap servers: `wms-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092`

### Temporal

Temporal uses PostgreSQL for persistence:

```yaml
# Connection details
host: temporal-postgresql.temporal.svc.cluster.local
port: 5432
database: temporal
user: temporal
password: temporal
```

Frontend address: `temporal-frontend.temporal.svc.cluster.local:7233`

WMS Namespace: `wms`

### Grafana Datasources

Grafana is pre-configured with:
- **Loki** (default): Log aggregation
- **Tempo**: Distributed tracing with trace-to-logs correlation

## Troubleshooting

### Check pod status
```bash
make status
# Or specific namespace:
kubectl get pods -n mongodb
kubectl get pods -n kafka
kubectl get pods -n temporal
kubectl get pods -n observability
kubectl get pods -n wms-platform-dev
```

### View logs
```bash
make logs SVC=order-service
# Or directly:
kubectl logs -f deployment/order-service -n wms-platform-dev
```

### Kafka cluster not ready
```bash
# Check Kafka status
kubectl get kafka -n kafka
kubectl describe kafka wms-kafka -n kafka

# Check Strimzi operator logs
kubectl logs -l name=strimzi-cluster-operator -n kafka
```

### Temporal not starting
```bash
# Check PostgreSQL
kubectl get pods -n temporal -l app.kubernetes.io/name=postgresql

# Check Temporal components
kubectl get pods -n temporal
kubectl logs -l app.kubernetes.io/component=frontend -n temporal
```

### MongoDB connection issues
```bash
# Test connection
kubectl exec -it deployment/mongodb -n mongodb -- mongosh \
  --username wmsuser --password wmspassword --authenticationDatabase wms
```

### Reset everything
```bash
make clean
make all
```

## Resource Requirements

Minimum resources for Kind cluster:

| Component | Memory | CPU |
|-----------|--------|-----|
| MongoDB | 512Mi | 250m |
| Kafka (3 nodes) | 3Gi | 750m |
| PostgreSQL | 512Mi | 500m |
| Temporal | 1Gi | 1000m |
| Observability | 1Gi | 500m |
| WMS Services | 2Gi | 2000m |
| **Total** | **~8Gi** | **~5 cores** |

Recommended: 16GB RAM and 8 CPU cores for comfortable development.
