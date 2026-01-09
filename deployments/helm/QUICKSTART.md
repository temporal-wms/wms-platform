# WMS Platform - Quick Start Guide

This guide will help you deploy the complete WMS platform on Kubernetes in 15 minutes.

## Prerequisites

- Kubernetes cluster (1.24+)
- kubectl configured
- Helm 3.8+
- 6+ nodes with 4 vCPU, 16Gi RAM each (for production)
  - Or 3 nodes with 4 vCPU, 16Gi RAM each (for development)

## Quick Start (Development)

### Option 1: All-in-One Script

```bash
# Navigate to helm directory
cd /Users/claudioed/development/github/temporal-war/wms-platform/deployments/helm

# Run quick start script
./quickstart-dev.sh
```

### Option 2: Manual Steps

**Step 1: Add Helm Repositories**

```bash
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add temporalio https://go.temporal.io/helm-charts
helm repo update
```

**Step 2: Create Namespace**

```bash
kubectl create namespace wms-platform-dev
```

**Step 3: Deploy MongoDB**

```bash
helm install mongodb bitnami/mongodb \
  --namespace wms-platform-dev \
  --set architecture=replicaset \
  --set replicaCount=3 \
  --set auth.enabled=true \
  --set auth.rootPassword=devRootPass123 \
  --set auth.username=wmsuser \
  --set auth.password=wmsPass123 \
  --set auth.database=wms \
  --set persistence.enabled=true \
  --set persistence.size=10Gi

# Wait for MongoDB
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=mongodb \
  --namespace wms-platform-dev --timeout=300s
```

**Step 4: Deploy Kafka**

```bash
helm install kafka bitnami/kafka \
  --namespace wms-platform-dev \
  --set controller.replicaCount=3 \
  --set broker.replicaCount=3 \
  --set kraft.enabled=true \
  --set persistence.enabled=true \
  --set persistence.size=10Gi

# Wait for Kafka
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=kafka \
  --namespace wms-platform-dev --timeout=300s
```

**Step 5: Create Kafka Topics**

```bash
# Connect to Kafka pod
kubectl exec -it kafka-0 -n wms-platform-dev -- bash

# Create all topics at once
for topic in orders waves routing picking consolidation packing shipping inventory labor; do
  kafka-topics.sh --create \
    --bootstrap-server localhost:9092 \
    --topic wms.${topic}.events \
    --partitions 6 \
    --replication-factor 3 \
    --config retention.ms=604800000
done

# Exit kafka pod
exit
```

**Step 6: Deploy Temporal**

```bash
helm install temporal temporalio/temporal \
  --namespace wms-platform-dev \
  --set server.replicaCount=1 \
  --set cassandra.enabled=false \
  --set postgresql.enabled=true \
  --set elasticsearch.enabled=false \
  --set prometheus.enabled=false \
  --set grafana.enabled=false

# Wait for Temporal
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=temporal-frontend \
  --namespace wms-platform-dev --timeout=600s
```

**Step 7: Create Temporal Namespace**

```bash
# Port forward to Temporal
kubectl port-forward -n wms-platform-dev svc/temporal-frontend 7233:7233 &

# Wait a moment for port-forward to establish
sleep 5

# Create namespace (requires tctl installed)
# If tctl is not installed, skip this - services will auto-register
tctl --namespace wms-platform-dev namespace register || echo "Skipping - tctl not installed"

# Kill port-forward
pkill -f "port-forward.*temporal-frontend"
```

**Step 8: Create MongoDB Secret**

```bash
kubectl create secret generic mongodb-credentials \
  --namespace wms-platform-dev \
  --from-literal=uri="mongodb://wmsuser:wmsPass123@mongodb-0.mongodb-headless.wms-platform-dev.svc.cluster.local:27017,mongodb-1.mongodb-headless.wms-platform-dev.svc.cluster.local:27017,mongodb-2.mongodb-headless.wms-platform-dev.svc.cluster.local:27017/wms?replicaSet=rs0"
```

**Step 9: Deploy WMS Platform**

```bash
cd /Users/claudioed/development/github/temporal-war/wms-platform

helm install wms-platform ./deployments/helm/wms-platform \
  --namespace wms-platform-dev \
  --values ./deployments/helm/wms-platform/values-dev.yaml

# Wait for all services
kubectl wait --for=condition=ready pod \
  -l app.kubernetes.io/instance=wms-platform \
  --namespace wms-platform-dev \
  --timeout=300s
```

**Step 10: Verify Deployment**

```bash
# Check all pods
kubectl get pods -n wms-platform-dev

# Check services
kubectl get svc -n wms-platform-dev

# Test order service
kubectl port-forward -n wms-platform-dev svc/order-service 8001:8001 &
sleep 2
curl http://localhost:8001/health
pkill -f "port-forward.*order-service"
```

## Quick Start (Production)

### Prerequisites

- Production Kubernetes cluster
- MongoDB credentials from your DBA
- External secrets management (e.g., AWS Secrets Manager, HashiCorp Vault)

### Steps

**Step 1: Create Production Namespace**

```bash
kubectl create namespace wms-platform
```

**Step 2: Deploy Infrastructure**

Follow the detailed instructions in [INFRASTRUCTURE.md](./INFRASTRUCTURE.md):

```bash
# Deploy MongoDB
helm install mongodb bitnami/mongodb \
  --namespace wms-platform \
  --values values-mongodb-prod.yaml

# Deploy Kafka
helm install kafka bitnami/kafka \
  --namespace wms-platform \
  --values values-kafka-prod.yaml

# Deploy Temporal
helm install temporal temporalio/temporal \
  --namespace wms-platform \
  --values values-temporal-prod.yaml
```

**Step 3: Create Secrets**

```bash
# Create MongoDB credentials (use your production values)
kubectl create secret generic mongodb-credentials \
  --namespace wms-platform \
  --from-literal=uri="mongodb://user:pass@mongodb-host:27017/wms?replicaSet=rs0"
```

**Step 4: Deploy WMS Platform**

```bash
cd /Users/claudioed/development/github/temporal-war/wms-platform

helm install wms-platform ./deployments/helm/wms-platform \
  --namespace wms-platform \
  --values ./deployments/helm/wms-platform/values-prod.yaml
```

**Step 5: Verify and Monitor**

```bash
# Check deployment
kubectl get pods -n wms-platform
kubectl get hpa -n wms-platform
kubectl get pdb -n wms-platform

# Check Prometheus metrics
kubectl port-forward -n wms-platform svc/order-service 8001:8001
curl http://localhost:8001/metrics
```

## Testing the Platform

### 1. Test Order Service API

```bash
# Port forward
kubectl port-forward -n wms-platform-dev svc/order-service 8001:8001

# In another terminal:

# Health check
curl http://localhost:8001/health

# Create an order
curl -X POST http://localhost:8001/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customerId": "CUST-12345",
    "priority": "same_day",
    "items": [
      {
        "sku": "WIDGET-001",
        "quantity": 2
      }
    ],
    "shippingAddress": {
      "street": "123 Main St",
      "city": "San Francisco",
      "state": "CA",
      "zipCode": "94105",
      "country": "US"
    }
  }'

# Get order
curl http://localhost:8001/api/v1/orders/{orderId}
```

### 2. Test Event Publishing

```bash
# Port forward to Kafka
kubectl port-forward -n wms-platform-dev svc/kafka 9092:9092

# In another terminal, consume events
kubectl exec -it kafka-0 -n wms-platform-dev -- \
  kafka-console-consumer.sh \
    --bootstrap-server localhost:9092 \
    --topic wms.orders.events \
    --from-beginning

# Create an order (in first terminal)
# You should see the event in the consumer
```

### 3. Test Temporal Workflows

```bash
# Port forward to Temporal UI
kubectl port-forward -n wms-platform-dev svc/temporal-web 8080:8080

# Open browser to http://localhost:8080
# You should see the Temporal UI

# View workflows in wms-platform-dev namespace
```

## Monitoring

### Prometheus Metrics

All services expose metrics at `/metrics`:

```bash
# Order service metrics
kubectl port-forward -n wms-platform-dev svc/order-service 8001:8001
curl http://localhost:8001/metrics | grep -E "^(http|kafka|mongodb)_"
```

### Logs

```bash
# Stream logs from order service
kubectl logs -f deployment/order-service -n wms-platform-dev

# Stream logs from all services
kubectl logs -f -l app.kubernetes.io/instance=wms-platform -n wms-platform-dev
```

### Distributed Tracing

If you have Jaeger installed:

```bash
# Port forward to Jaeger UI
kubectl port-forward -n observability svc/jaeger-query 16686:16686

# Open browser to http://localhost:16686
# Select service: order-service
# Click "Find Traces"
```

## Common Operations

### Scale a Service

```bash
# Scale order service to 5 replicas
kubectl scale deployment order-service -n wms-platform-dev --replicas=5

# Or update Helm values
helm upgrade wms-platform ./deployments/helm/wms-platform \
  --namespace wms-platform-dev \
  --values ./deployments/helm/wms-platform/values-dev.yaml \
  --set services.order-service.replicaCount=5
```

### Update Service Image

```bash
# Update order service to new version
helm upgrade wms-platform ./deployments/helm/wms-platform \
  --namespace wms-platform-dev \
  --values ./deployments/helm/wms-platform/values-dev.yaml \
  --set services.order-service.image.tag=v2.0.0
```

### Enable/Disable a Service

```bash
# Disable labor service
helm upgrade wms-platform ./deployments/helm/wms-platform \
  --namespace wms-platform-dev \
  --values ./deployments/helm/wms-platform/values-dev.yaml \
  --set services.labor-service.enabled=false
```

## Cleanup

### Remove WMS Platform Only

```bash
helm uninstall wms-platform --namespace wms-platform-dev
```

### Remove Everything

```bash
# Remove WMS platform
helm uninstall wms-platform --namespace wms-platform-dev

# Remove infrastructure
helm uninstall temporal --namespace wms-platform-dev
helm uninstall kafka --namespace wms-platform-dev
helm uninstall mongodb --namespace wms-platform-dev

# Delete namespace (this deletes all PVCs and data!)
kubectl delete namespace wms-platform-dev
```

## Troubleshooting

### Pods Not Starting

```bash
# Check pod status
kubectl get pods -n wms-platform-dev

# Describe pod
kubectl describe pod <pod-name> -n wms-platform-dev

# Check logs
kubectl logs <pod-name> -n wms-platform-dev

# Common issues:
# 1. ImagePullBackOff - check image exists
# 2. CrashLoopBackOff - check logs
# 3. Pending - check resources/node capacity
```

### Service Not Reachable

```bash
# Check service
kubectl get svc -n wms-platform-dev

# Check endpoints
kubectl get endpoints -n wms-platform-dev

# Check if pods are ready
kubectl get pods -l app=order-service -n wms-platform-dev
```

### MongoDB Connection Issues

```bash
# Test MongoDB connection
kubectl exec -it mongodb-0 -n wms-platform-dev -- \
  mongosh mongodb://wmsuser:wmsPass123@localhost:27017/wms?replicaSet=rs0

# Check replica set status
kubectl exec -it mongodb-0 -n wms-platform-dev -- \
  mongosh --eval "rs.status()"
```

### Kafka Connection Issues

```bash
# Test Kafka connection
kubectl exec -it kafka-0 -n wms-platform-dev -- \
  kafka-broker-api-versions.sh --bootstrap-server localhost:9092

# List topics
kubectl exec -it kafka-0 -n wms-platform-dev -- \
  kafka-topics.sh --list --bootstrap-server localhost:9092
```

## Next Steps

1. **Configure Ingress** - Expose services externally
2. **Set up CI/CD** - Automate deployments
3. **Configure monitoring** - Set up Prometheus + Grafana
4. **Configure alerting** - Set up alert rules
5. **Configure backups** - Set up automated backups
6. **Load testing** - Test platform under load
7. **Security hardening** - Network policies, RBAC, secrets encryption

## Resources

- [Helm Chart README](./wms-platform/README.md)
- [Infrastructure Setup](./INFRASTRUCTURE.md)
- [OpenAPI Documentation](../../services/*/docs/openapi.yaml)
- [AsyncAPI Documentation](../../shared/api/asyncapi/wms-events.asyncapi.yaml)
