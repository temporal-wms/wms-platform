#!/bin/bash
# Deploy Grafana Observability Stack (Loki, Tempo, Grafana) to Kind cluster
# Usage: ./scripts/deploy-observability.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HELM_DIR="${SCRIPT_DIR}/../helm/observability"
NAMESPACE="monitoring"

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

    if ! command -v helm &> /dev/null; then
        log_error "helm is not installed. Please install Helm first."
        exit 1
    fi

    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed. Please install kubectl first."
        exit 1
    fi

    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster. Please check your kubeconfig."
        exit 1
    fi

    log_info "Prerequisites check passed."
}

# Add Helm repositories
add_helm_repos() {
    log_info "Adding Helm repositories..."

    helm repo add grafana https://grafana.github.io/helm-charts || true
    helm repo update

    log_info "Helm repositories updated."
}

# Create namespace
create_namespace() {
    log_info "Creating namespace ${NAMESPACE}..."

    kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -

    log_info "Namespace ${NAMESPACE} ready."
}

# Install Loki
install_loki() {
    log_info "Installing Loki..."

    helm upgrade --install loki grafana/loki \
        --namespace ${NAMESPACE} \
        --values "${HELM_DIR}/loki-values.yaml" \
        --wait \
        --timeout 10m

    log_info "Loki installed successfully."
}

# Install Tempo
install_tempo() {
    log_info "Installing Tempo..."

    helm upgrade --install tempo grafana/tempo \
        --namespace ${NAMESPACE} \
        --values "${HELM_DIR}/tempo-values.yaml" \
        --wait \
        --timeout 10m

    log_info "Tempo installed successfully."
}

# Install Grafana
install_grafana() {
    log_info "Installing Grafana..."

    helm upgrade --install grafana grafana/grafana \
        --namespace ${NAMESPACE} \
        --values "${HELM_DIR}/grafana-values.yaml" \
        --wait \
        --timeout 10m

    log_info "Grafana installed successfully."
}

# Verify deployments
verify_deployments() {
    log_info "Verifying deployments..."

    echo ""
    log_info "Pod status in ${NAMESPACE}:"
    kubectl get pods -n ${NAMESPACE} -o wide

    echo ""
    log_info "Services in ${NAMESPACE}:"
    kubectl get svc -n ${NAMESPACE}

    echo ""
    log_info "Checking pod readiness..."

    # Wait for Loki
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=loki -n ${NAMESPACE} --timeout=120s || log_warn "Loki pods not ready"

    # Wait for Tempo
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=tempo -n ${NAMESPACE} --timeout=120s || log_warn "Tempo pods not ready"

    # Wait for Grafana
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=grafana -n ${NAMESPACE} --timeout=120s || log_warn "Grafana pods not ready"

    log_info "Deployment verification complete."
}

# Print access instructions
print_access_info() {
    echo ""
    echo "=========================================="
    echo "      Observability Stack Deployed       "
    echo "=========================================="
    echo ""
    log_info "Access Grafana:"
    echo "  kubectl port-forward svc/grafana -n ${NAMESPACE} 3000:80"
    echo "  Open: http://localhost:3000"
    echo "  Username: admin"
    echo "  Password: wms-admin-2024"
    echo ""
    log_info "Access Loki directly (for debugging):"
    echo "  kubectl port-forward svc/loki-gateway -n ${NAMESPACE} 3100:80"
    echo ""
    log_info "Access Tempo directly (for debugging):"
    echo "  kubectl port-forward svc/tempo -n ${NAMESPACE} 3200:3100"
    echo ""
    log_info "Example Loki query for WMS logs:"
    echo '  {namespace="wms-platform"} | json | wms_correlation_id!=""'
    echo ""
    log_info "To send test logs, update WMS services with new logging configuration."
    echo ""
}

# Main execution
main() {
    log_info "Starting Observability Stack deployment..."
    echo ""

    check_prerequisites
    add_helm_repos
    create_namespace

    # Install components
    install_loki
    install_tempo
    install_grafana

    # Verify and print info
    verify_deployments
    print_access_info

    log_info "Observability Stack deployment complete!"
}

# Run main function
main "$@"
