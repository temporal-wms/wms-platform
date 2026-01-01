#!/bin/bash
# WMS Platform - Complete Setup Script
# This script sets up the entire WMS platform on a Kind cluster

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOYMENTS_DIR="$(dirname "$SCRIPT_DIR")"
ROOT_DIR="$(dirname "$DEPLOYMENTS_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    local missing=()

    command -v docker >/dev/null 2>&1 || missing+=("docker")
    command -v kind >/dev/null 2>&1 || missing+=("kind")
    command -v kubectl >/dev/null 2>&1 || missing+=("kubectl")
    command -v helm >/dev/null 2>&1 || missing+=("helm")

    if [ ${#missing[@]} -ne 0 ]; then
        log_error "Missing required tools: ${missing[*]}"
        exit 1
    fi

    log_success "All prerequisites are installed"
}

# Create Kind cluster
create_cluster() {
    log_info "Creating Kind cluster..."

    if kind get clusters 2>/dev/null | grep -q "wms"; then
        log_warning "Cluster 'wms' already exists"
        read -p "Do you want to delete and recreate it? (y/n): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            kind delete cluster --name wms
        else
            log_info "Using existing cluster"
            return
        fi
    fi

    kind create cluster --name wms --config "$DEPLOYMENTS_DIR/kind/kind-config.yaml"
    log_success "Kind cluster created"
}

# Add Helm repositories
add_helm_repos() {
    log_info "Adding Helm repositories..."

    helm repo add bitnami https://charts.bitnami.com/bitnami 2>/dev/null || true
    helm repo add strimzi https://strimzi.io/charts/ 2>/dev/null || true
    helm repo add temporalio https://temporalio.github.io/helm-charts 2>/dev/null || true
    helm repo add grafana https://grafana.github.io/helm-charts 2>/dev/null || true

    helm repo update
    log_success "Helm repositories added and updated"
}

# Deploy infrastructure
deploy_infrastructure() {
    log_info "Deploying infrastructure..."
    "$SCRIPT_DIR/deploy-infrastructure.sh"
    log_success "Infrastructure deployed"
}

# Deploy observability
deploy_observability() {
    log_info "Deploying observability stack..."
    "$SCRIPT_DIR/deploy-observability.sh"
    log_success "Observability stack deployed"
}

# Create Temporal namespace
create_temporal_namespace() {
    log_info "Creating Temporal namespace..."
    "$SCRIPT_DIR/create-temporal-namespace.sh"
    log_success "Temporal namespace created"
}

# Build WMS Docker images
build_wms_images() {
    log_info "Building WMS Docker images..."

    local services=(
        "order-service"
        "inventory-service"
        "waving-service"
        "routing-service"
        "picking-service"
        "consolidation-service"
        "packing-service"
        "shipping-service"
        "labor-service"
        "orchestrator"
    )

    for svc in "${services[@]}"; do
        if [ -d "$ROOT_DIR/services/$svc" ]; then
            log_info "Building $svc..."
            docker build -t "wms-platform/$svc:latest" "$ROOT_DIR/services/$svc"
        fi
    done

    log_success "WMS Docker images built"
}

# Load images into Kind
load_images() {
    log_info "Loading Docker images into Kind cluster..."

    local services=(
        "order-service"
        "inventory-service"
        "waving-service"
        "routing-service"
        "picking-service"
        "consolidation-service"
        "packing-service"
        "shipping-service"
        "labor-service"
        "orchestrator"
    )

    for svc in "${services[@]}"; do
        if docker image inspect "wms-platform/$svc:latest" >/dev/null 2>&1; then
            log_info "Loading $svc..."
            kind load docker-image "wms-platform/$svc:latest" --name wms
        fi
    done

    log_success "Docker images loaded into Kind"
}

# Deploy WMS platform
deploy_wms() {
    log_info "Deploying WMS platform..."

    kubectl create namespace wms-platform-dev 2>/dev/null || true

    helm upgrade --install wms-platform "$DEPLOYMENTS_DIR/helm/wms-platform" \
        -n wms-platform-dev \
        -f "$DEPLOYMENTS_DIR/helm/wms-platform/values-kind.yaml" \
        --wait --timeout 5m

    log_success "WMS platform deployed"
}

# Print access information
print_access_info() {
    echo ""
    log_success "========================================"
    log_success "WMS Platform Setup Complete!"
    log_success "========================================"
    echo ""
    log_info "Access URLs:"
    echo "  - Grafana:      http://localhost:3000 (admin/admin)"
    echo "  - Temporal UI:  http://localhost:8080"
    echo "  - Loki:         http://localhost:30310"
    echo "  - Tempo:        http://localhost:30311"
    echo ""
    log_info "WMS Services (NodePorts):"
    echo "  - Order Service:         http://localhost:30001"
    echo "  - Waving Service:        http://localhost:30002"
    echo "  - Routing Service:       http://localhost:30003"
    echo "  - Picking Service:       http://localhost:30004"
    echo "  - Consolidation Service: http://localhost:30005"
    echo "  - Packing Service:       http://localhost:30006"
    echo "  - Shipping Service:      http://localhost:30007"
    echo "  - Inventory Service:     http://localhost:30008"
    echo "  - Labor Service:         http://localhost:30009"
    echo ""
    log_info "Infrastructure:"
    echo "  - MongoDB:  mongodb://wmsuser:wmspassword@localhost:27017/wms"
    echo "  - Kafka:    localhost:9092"
    echo "  - Temporal: localhost:7233"
    echo ""
}

# Main execution
main() {
    echo ""
    log_info "========================================"
    log_info "WMS Platform Setup"
    log_info "========================================"
    echo ""

    check_prerequisites
    create_cluster
    add_helm_repos
    deploy_infrastructure
    deploy_observability
    create_temporal_namespace
    build_wms_images
    load_images
    deploy_wms
    print_access_info
}

# Parse arguments
case "${1:-}" in
    --cluster-only)
        check_prerequisites
        create_cluster
        add_helm_repos
        ;;
    --infra-only)
        add_helm_repos
        deploy_infrastructure
        ;;
    --observability-only)
        add_helm_repos
        deploy_observability
        ;;
    --wms-only)
        build_wms_images
        load_images
        deploy_wms
        ;;
    --help)
        echo "Usage: $0 [OPTIONS]"
        echo ""
        echo "Options:"
        echo "  --cluster-only      Create Kind cluster only"
        echo "  --infra-only        Deploy infrastructure only"
        echo "  --observability-only Deploy observability stack only"
        echo "  --wms-only          Build and deploy WMS services only"
        echo "  --help              Show this help message"
        echo ""
        echo "Without options, runs the complete setup."
        ;;
    *)
        main
        ;;
esac
