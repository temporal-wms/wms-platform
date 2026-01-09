#!/bin/bash

# WMS Platform - Development Quick Start Script
# This script deploys the complete WMS platform in development mode

set -e

NAMESPACE="wms-platform-dev"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=========================================="
echo "WMS Platform - Development Quick Start"
echo "=========================================="
echo ""

# Check prerequisites
echo "Checking prerequisites..."

if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl not found. Please install kubectl first."
    exit 1
fi

if ! command -v helm &> /dev/null; then
    echo "❌ helm not found. Please install Helm 3.8+ first."
    exit 1
fi

echo "✓ kubectl found"
echo "✓ helm found"
echo ""

# Check cluster connectivity
echo "Checking Kubernetes cluster connectivity..."
if ! kubectl cluster-info &> /dev/null; then
    echo "❌ Cannot connect to Kubernetes cluster. Please configure kubectl first."
    exit 1
fi
echo "✓ Connected to Kubernetes cluster"
echo ""

# Add Helm repositories
echo "Adding Helm repositories..."
helm repo add bitnami https://charts.bitnami.com/bitnami &> /dev/null || true
helm repo add temporalio https://go.temporal.io/helm-charts &> /dev/null || true
helm repo update
echo "✓ Helm repositories added"
echo ""

# Create namespace
echo "Creating namespace: $NAMESPACE..."
kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
echo "✓ Namespace created"
echo ""

# Deploy MongoDB
echo "=========================================="
echo "Deploying MongoDB (3 replicas)..."
echo "=========================================="
helm upgrade --install mongodb bitnami/mongodb \
  --namespace $NAMESPACE \
  --set architecture=replicaset \
  --set replicaCount=3 \
  --set auth.enabled=true \
  --set auth.rootPassword=devRootPass123 \
  --set auth.username=wmsuser \
  --set auth.password=wmsPass123 \
  --set auth.database=wms \
  --set persistence.enabled=true \
  --set persistence.size=10Gi \
  --set resources.requests.memory=256Mi \
  --set resources.requests.cpu=100m \
  --set resources.limits.memory=512Mi \
  --set resources.limits.cpu=500m \
  --wait --timeout=10m

echo "✓ MongoDB deployed"
echo ""

# Deploy Kafka
echo "=========================================="
echo "Deploying Kafka (3 brokers)..."
echo "=========================================="
helm upgrade --install kafka bitnami/kafka \
  --namespace $NAMESPACE \
  --set controller.replicaCount=3 \
  --set broker.replicaCount=3 \
  --set kraft.enabled=true \
  --set persistence.enabled=true \
  --set persistence.size=10Gi \
  --set resources.requests.memory=512Mi \
  --set resources.requests.cpu=250m \
  --set resources.limits.memory=1Gi \
  --set resources.limits.cpu=1000m \
  --wait --timeout=10m

echo "✓ Kafka deployed"
echo ""

# Create Kafka topics
echo "Creating Kafka topics..."
kubectl exec -it kafka-0 -n $NAMESPACE -- bash -c '
for topic in orders waves routing picking consolidation packing shipping inventory labor; do
  kafka-topics.sh --create \
    --bootstrap-server localhost:9092 \
    --topic wms.${topic}.events \
    --partitions 6 \
    --replication-factor 3 \
    --config retention.ms=604800000 \
    --if-not-exists 2>/dev/null || true
done
echo "Topics created:"
kafka-topics.sh --list --bootstrap-server localhost:9092
'
echo "✓ Kafka topics created"
echo ""

# Deploy Temporal
echo "=========================================="
echo "Deploying Temporal..."
echo "=========================================="
helm upgrade --install temporal temporalio/temporal \
  --namespace $NAMESPACE \
  --set server.replicaCount=1 \
  --set cassandra.enabled=false \
  --set postgresql.enabled=true \
  --set postgresql.persistence.size=10Gi \
  --set elasticsearch.enabled=false \
  --set prometheus.enabled=false \
  --set grafana.enabled=false \
  --set web.enabled=true \
  --wait --timeout=15m

echo "✓ Temporal deployed"
echo ""

# Create MongoDB secret
echo "Creating MongoDB credentials secret..."
kubectl create secret generic mongodb-credentials \
  --namespace $NAMESPACE \
  --from-literal=uri="mongodb://wmsuser:wmsPass123@mongodb-0.mongodb-headless.$NAMESPACE.svc.cluster.local:27017,mongodb-1.mongodb-headless.$NAMESPACE.svc.cluster.local:27017,mongodb-2.mongodb-headless.$NAMESPACE.svc.cluster.local:27017/wms?replicaSet=rs0" \
  --dry-run=client -o yaml | kubectl apply -f -

echo "✓ MongoDB secret created"
echo ""

# Deploy WMS Platform
echo "=========================================="
echo "Deploying WMS Platform (10 services)..."
echo "=========================================="
cd "$SCRIPT_DIR/../.."
helm upgrade --install wms-platform ./deployments/helm/wms-platform \
  --namespace $NAMESPACE \
  --values ./deployments/helm/wms-platform/values-dev.yaml \
  --wait --timeout=10m

echo "✓ WMS Platform deployed"
echo ""

# Verify deployment
echo "=========================================="
echo "Verifying deployment..."
echo "=========================================="
echo ""

echo "Pods:"
kubectl get pods -n $NAMESPACE
echo ""

echo "Services:"
kubectl get svc -n $NAMESPACE
echo ""

# Test order service
echo "Testing Order Service..."
kubectl wait --for=condition=ready pod -l app=order-service -n $NAMESPACE --timeout=60s

echo ""
echo "=========================================="
echo "✓ Deployment Complete!"
echo "=========================================="
echo ""
echo "Next steps:"
echo ""
echo "1. Check all pods are running:"
echo "   kubectl get pods -n $NAMESPACE"
echo ""
echo "2. Test Order Service API:"
echo "   kubectl port-forward -n $NAMESPACE svc/order-service 8001:8001"
echo "   curl http://localhost:8001/health"
echo ""
echo "3. Access Temporal Web UI:"
echo "   kubectl port-forward -n $NAMESPACE svc/temporal-web 8080:8080"
echo "   Open http://localhost:8080 in your browser"
echo ""
echo "4. View logs:"
echo "   kubectl logs -f deployment/order-service -n $NAMESPACE"
echo ""
echo "5. Cleanup (when done):"
echo "   helm uninstall wms-platform --namespace $NAMESPACE"
echo "   helm uninstall temporal --namespace $NAMESPACE"
echo "   helm uninstall kafka --namespace $NAMESPACE"
echo "   helm uninstall mongodb --namespace $NAMESPACE"
echo "   kubectl delete namespace $NAMESPACE"
echo ""
echo "For more information, see:"
echo "  - deployments/helm/wms-platform/README.md"
echo "  - deployments/helm/QUICKSTART.md"
echo "  - deployments/helm/INFRASTRUCTURE.md"
echo ""
