#!/bin/bash
# WMS Platform - Kind Cluster Setup Script
# This script creates a kind cluster and deploys the entire WMS platform

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
NAMESPACE="wms-platform"
CLUSTER_NAME="wms"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    if ! command -v kind &> /dev/null; then
        log_error "kind is not installed. Please install it first: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
        exit 1
    fi

    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed. Please install it first."
        exit 1
    fi

    if ! command -v helm &> /dev/null; then
        log_error "helm is not installed. Please install it first: https://helm.sh/docs/intro/install/"
        exit 1
    fi

    if ! command -v docker &> /dev/null; then
        log_error "docker is not installed. Please install it first."
        exit 1
    fi

    log_info "All prerequisites are installed."
}

# Create kind cluster
create_cluster() {
    log_info "Creating kind cluster: $CLUSTER_NAME..."

    # Check if cluster already exists
    if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
        log_warn "Cluster '$CLUSTER_NAME' already exists. Deleting it first..."
        kind delete cluster --name "$CLUSTER_NAME"
    fi

    kind create cluster --config "$SCRIPT_DIR/kind-config.yaml"

    # Wait for cluster to be ready
    log_info "Waiting for cluster to be ready..."
    kubectl wait --for=condition=Ready nodes --all --timeout=120s

    log_info "Kind cluster created successfully."
}

# Load Docker images into kind
load_images() {
    log_info "Loading Docker images into kind cluster..."

    local images=(
        "temporal-wms/order-service:latest"
        "temporal-wms/waving-service:latest"
        "temporal-wms/routing-service:latest"
        "temporal-wms/picking-service:latest"
        "temporal-wms/consolidation-service:latest"
        "temporal-wms/packing-service:latest"
        "temporal-wms/shipping-service:latest"
        "temporal-wms/inventory-service:latest"
        "temporal-wms/labor-service:latest"
        "temporal-wms/orchestrator:latest"
    )

    for image in "${images[@]}"; do
        if docker image inspect "$image" &> /dev/null; then
            log_info "Loading $image..."
            kind load docker-image "$image" --name "$CLUSTER_NAME"
        else
            log_warn "Image $image not found locally. Skipping..."
        fi
    done

    log_info "Docker images loaded successfully."
}

# Add Helm repositories
add_helm_repos() {
    log_info "Adding Helm repositories..."

    helm repo add bitnami https://charts.bitnami.com/bitnami 2>/dev/null || true
    helm repo add temporalio https://go.temporal.io/helm-charts 2>/dev/null || true
    helm repo update

    log_info "Helm repositories added."
}

# Create namespace
create_namespace() {
    log_info "Creating namespace: $NAMESPACE..."
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    log_info "Namespace created."
}

# Deploy MongoDB
deploy_mongodb() {
    log_info "Deploying MongoDB..."

    helm upgrade --install mongodb bitnami/mongodb \
        --namespace "$NAMESPACE" \
        --set architecture=replicaset \
        --set replicaCount=1 \
        --set auth.enabled=true \
        --set auth.rootUser=root \
        --set auth.rootPassword=mongodb123 \
        --set auth.replicaSetKey=wmsreplicaset \
        --set persistence.size=2Gi \
        --set arbiter.enabled=false \
        --wait --timeout 5m

    log_info "MongoDB deployed successfully."
}

# Deploy Kafka
deploy_kafka() {
    log_info "Deploying Kafka..."

    helm upgrade --install kafka bitnami/kafka \
        --namespace "$NAMESPACE" \
        --set controller.replicaCount=1 \
        --set kraft.enabled=true \
        --set listeners.client.protocol=PLAINTEXT \
        --set listeners.controller.protocol=PLAINTEXT \
        --set persistence.size=2Gi \
        --set provisioning.enabled=true \
        --set "provisioning.topics[0].name=wms.orders.events" \
        --set "provisioning.topics[0].partitions=3" \
        --set "provisioning.topics[1].name=wms.waves.events" \
        --set "provisioning.topics[1].partitions=3" \
        --set "provisioning.topics[2].name=wms.routing.events" \
        --set "provisioning.topics[2].partitions=3" \
        --set "provisioning.topics[3].name=wms.picking.events" \
        --set "provisioning.topics[3].partitions=3" \
        --set "provisioning.topics[4].name=wms.consolidation.events" \
        --set "provisioning.topics[4].partitions=3" \
        --set "provisioning.topics[5].name=wms.packing.events" \
        --set "provisioning.topics[5].partitions=3" \
        --set "provisioning.topics[6].name=wms.shipping.events" \
        --set "provisioning.topics[6].partitions=3" \
        --set "provisioning.topics[7].name=wms.inventory.events" \
        --set "provisioning.topics[7].partitions=3" \
        --set "provisioning.topics[8].name=wms.labor.events" \
        --set "provisioning.topics[8].partitions=3" \
        --wait --timeout 5m

    log_info "Kafka deployed successfully."
}

# Deploy Temporal
deploy_temporal() {
    log_info "Deploying Temporal..."

    helm upgrade --install temporal temporalio/temporal \
        --namespace "$NAMESPACE" \
        --set server.replicaCount=1 \
        --set cassandra.enabled=false \
        --set mysql.enabled=false \
        --set postgresql.enabled=true \
        --set postgresql.auth.password=temporal123 \
        --set prometheus.enabled=false \
        --set grafana.enabled=false \
        --set elasticsearch.enabled=false \
        --set web.enabled=true \
        --wait --timeout 10m

    log_info "Temporal deployed successfully."
}

# Create secrets
create_secrets() {
    log_info "Creating secrets..."

    kubectl create secret generic mongodb-credentials \
        --namespace "$NAMESPACE" \
        --from-literal=uri="mongodb://root:mongodb123@mongodb-0.mongodb-headless.$NAMESPACE.svc.cluster.local:27017/?replicaSet=rs0&authSource=admin" \
        --dry-run=client -o yaml | kubectl apply -f -

    log_info "Secrets created."
}

# Deploy WMS Platform
deploy_wms_platform() {
    log_info "Deploying WMS Platform..."

    helm upgrade --install wms-platform "$PROJECT_ROOT/deployments/helm/wms-platform" \
        --namespace "$NAMESPACE" \
        --values "$PROJECT_ROOT/deployments/helm/wms-platform/values-dev.yaml" \
        --set global.imagePullPolicy=Never \
        --set "services.order-service.image.repository=temporal-wms/order-service" \
        --set "services.order-service.image.tag=latest" \
        --set "services.waving-service.image.repository=temporal-wms/waving-service" \
        --set "services.waving-service.image.tag=latest" \
        --set "services.routing-service.image.repository=temporal-wms/routing-service" \
        --set "services.routing-service.image.tag=latest" \
        --set "services.picking-service.image.repository=temporal-wms/picking-service" \
        --set "services.picking-service.image.tag=latest" \
        --set "services.consolidation-service.image.repository=temporal-wms/consolidation-service" \
        --set "services.consolidation-service.image.tag=latest" \
        --set "services.packing-service.image.repository=temporal-wms/packing-service" \
        --set "services.packing-service.image.tag=latest" \
        --set "services.shipping-service.image.repository=temporal-wms/shipping-service" \
        --set "services.shipping-service.image.tag=latest" \
        --set "services.inventory-service.image.repository=temporal-wms/inventory-service" \
        --set "services.inventory-service.image.tag=latest" \
        --set "services.labor-service.image.repository=temporal-wms/labor-service" \
        --set "services.labor-service.image.tag=latest" \
        --set "services.orchestrator.image.repository=temporal-wms/orchestrator" \
        --set "services.orchestrator.image.tag=latest" \
        --wait --timeout 5m

    log_info "WMS Platform deployed successfully."
}

# Patch services to NodePort for local access
patch_services() {
    log_info "Patching services for local access..."

    local services=(
        "order-service:30001"
        "waving-service:30002"
        "routing-service:30003"
        "picking-service:30004"
        "consolidation-service:30005"
        "packing-service:30006"
        "shipping-service:30007"
        "inventory-service:30008"
        "labor-service:30009"
    )

    for svc_port in "${services[@]}"; do
        IFS=':' read -r svc port <<< "$svc_port"
        kubectl patch svc "$svc" -n "$NAMESPACE" -p "{\"spec\":{\"type\":\"NodePort\",\"ports\":[{\"port\":8080,\"nodePort\":$port}]}}" 2>/dev/null || true
    done

    # Patch Temporal Web UI
    kubectl patch svc temporal-web -n "$NAMESPACE" -p '{"spec":{"type":"NodePort","ports":[{"port":8080,"nodePort":30088}]}}' 2>/dev/null || true

    log_info "Services patched."
}

# Print status
print_status() {
    log_info "Deployment Status:"
    echo ""
    echo "Pods:"
    kubectl get pods -n "$NAMESPACE"
    echo ""
    echo "Services:"
    kubectl get svc -n "$NAMESPACE"
    echo ""
    log_info "Access URLs (after port-forwarding or NodePort):"
    echo "  - Order Service:    http://localhost:8001"
    echo "  - Waving Service:   http://localhost:8002"
    echo "  - Routing Service:  http://localhost:8003"
    echo "  - Picking Service:  http://localhost:8004"
    echo "  - Consolidation:    http://localhost:8005"
    echo "  - Packing Service:  http://localhost:8006"
    echo "  - Shipping Service: http://localhost:8007"
    echo "  - Inventory Service: http://localhost:8008"
    echo "  - Labor Service:    http://localhost:8009"
    echo "  - Temporal UI:      http://localhost:8088"
    echo ""
    log_info "To use port-forwarding for a service:"
    echo "  kubectl port-forward svc/order-service 8001:8080 -n $NAMESPACE"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up kind cluster..."
    kind delete cluster --name "$CLUSTER_NAME"
    log_info "Cluster deleted."
}

# Main
main() {
    case "${1:-}" in
        --cleanup)
            cleanup
            ;;
        --status)
            print_status
            ;;
        *)
            check_prerequisites
            create_cluster
            load_images
            add_helm_repos
            create_namespace
            deploy_mongodb
            deploy_kafka
            deploy_temporal
            create_secrets
            deploy_wms_platform
            patch_services
            print_status
            ;;
    esac
}

main "$@"
