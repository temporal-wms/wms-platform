#!/bin/bash
# WMS Platform - Kong API Gateway Deployment Script
# Deploys Kong with Kubernetes Gateway API support

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOYMENTS_DIR="$(dirname "$SCRIPT_DIR")"
HELM_KONG_DIR="$DEPLOYMENTS_DIR/helm/infrastructure/kong"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Install Gateway API CRDs
install_gateway_api_crds() {
    log_info "Installing Gateway API CRDs (v1.0.0)..."
    kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.0.0/standard-install.yaml
    log_success "Gateway API CRDs installed"
}

# Create kong namespace
create_namespace() {
    log_info "Creating kong namespace..."
    kubectl create namespace kong 2>/dev/null || log_warning "Namespace 'kong' already exists"
}

# Add Kong Helm repository
add_helm_repo() {
    log_info "Adding Kong Helm repository..."
    helm repo add kong https://charts.konghq.com 2>/dev/null || true
    helm repo update
}

# Update Helm dependencies
update_dependencies() {
    log_info "Updating Helm dependencies..."
    cd "$HELM_KONG_DIR"
    helm dependency update
    cd -
}

# Deploy Kong
deploy_kong() {
    local VALUES_FILE="${1:-values-kind.yaml}"

    log_info "Deploying Kong with $VALUES_FILE..."

    helm upgrade --install kong "$HELM_KONG_DIR" \
        -n kong \
        -f "$HELM_KONG_DIR/$VALUES_FILE" \
        --wait --timeout 5m

    log_success "Kong deployed successfully"
}

# Wait for Kong to be ready
wait_for_kong() {
    log_info "Waiting for Kong to be ready..."
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=kong -n kong --timeout=120s
    log_success "Kong is ready"
}

# Show Kong status
show_status() {
    echo ""
    log_info "Kong Status:"
    echo ""
    echo "=== Pods ==="
    kubectl get pods -n kong
    echo ""
    echo "=== Services ==="
    kubectl get svc -n kong
    echo ""
    echo "=== Gateway Resources ==="
    kubectl get gatewayclasses,gateways -A 2>/dev/null || log_warning "No Gateway resources found yet"
}

# Undeploy Kong
undeploy_kong() {
    log_info "Removing Kong..."
    helm uninstall kong -n kong 2>/dev/null || log_warning "Kong release not found"
    kubectl delete namespace kong 2>/dev/null || log_warning "Namespace 'kong' not found"
    log_success "Kong removed"
}

# Print usage info
print_usage() {
    echo ""
    log_success "========================================"
    log_success "Kong API Gateway Deployment Complete!"
    log_success "========================================"
    echo ""
    log_info "Access URLs (Kind development):"
    echo "  - Kong Proxy:  http://localhost:8888"
    echo "  - Kong Admin:  http://localhost:8881"
    echo ""
    log_info "Example API calls through Kong:"
    echo "  - Orders:      curl http://localhost:8888/api/order-service/v1/health"
    echo "  - Inventory:   curl http://localhost:8888/api/inventory-service/v1/health"
    echo "  - Sellers:     curl http://localhost:8888/api/seller-service/v1/health"
    echo "  - Billing:     curl http://localhost:8888/api/billing-service/v1/health"
    echo ""
    log_info "To view HTTPRoutes after deploying WMS:"
    echo "  kubectl get httproutes -n wms-platform-dev"
    echo ""
}

# Main execution
main() {
    local VALUES_FILE="${1:-values-kind.yaml}"

    log_info "========================================"
    log_info "Deploying Kong API Gateway"
    log_info "========================================"
    echo ""

    install_gateway_api_crds
    create_namespace
    add_helm_repo
    update_dependencies
    deploy_kong "$VALUES_FILE"
    wait_for_kong
    show_status
    print_usage
}

# Parse arguments
case "${1:-}" in
    crds)
        install_gateway_api_crds
        ;;
    undeploy)
        undeploy_kong
        ;;
    status)
        show_status
        ;;
    *)
        main "${1:-values-kind.yaml}"
        ;;
esac
