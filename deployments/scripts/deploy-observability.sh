#!/bin/bash
# WMS Platform - Observability Stack Deployment Script
# Deploys Loki, Tempo, Promtail, and Grafana

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOYMENTS_DIR="$(dirname "$SCRIPT_DIR")"
HELM_OBS_DIR="$DEPLOYMENTS_DIR/helm/infrastructure/observability"

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

# Create namespace
create_namespace() {
    log_info "Creating observability namespace..."
    kubectl create namespace observability 2>/dev/null || true
}

# Deploy Loki
deploy_loki() {
    log_info "Deploying Loki..."

    helm upgrade --install loki grafana/loki \
        -n observability \
        -f "$HELM_OBS_DIR/loki-values.yaml" \
        --wait --timeout 5m

    log_success "Loki deployed"
}

# Deploy Tempo
deploy_tempo() {
    log_info "Deploying Tempo..."

    helm upgrade --install tempo grafana/tempo \
        -n observability \
        -f "$HELM_OBS_DIR/tempo-values.yaml" \
        --wait --timeout 5m

    log_success "Tempo deployed"
}

# Deploy Promtail
deploy_promtail() {
    log_info "Deploying Promtail..."

    helm upgrade --install promtail grafana/promtail \
        -n observability \
        -f "$HELM_OBS_DIR/promtail-values.yaml" \
        --wait --timeout 5m

    log_success "Promtail deployed"
}

# Deploy Grafana
deploy_grafana() {
    log_info "Deploying Grafana..."

    helm upgrade --install grafana grafana/grafana \
        -n observability \
        -f "$HELM_OBS_DIR/grafana-values.yaml" \
        --wait --timeout 5m

    log_success "Grafana deployed"
}

# Deploy Prometheus
deploy_prometheus() {
    log_info "Deploying Prometheus..."

    helm upgrade --install prometheus prometheus-community/prometheus \
        -n observability \
        -f "$HELM_OBS_DIR/prometheus-values.yaml" \
        --wait --timeout 5m

    log_success "Prometheus deployed"
}

# Wait for all pods
wait_for_observability() {
    log_info "Waiting for observability stack to be ready..."

    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=loki -n observability --timeout=120s || true
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=tempo -n observability --timeout=120s || true
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=promtail -n observability --timeout=120s || true
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=grafana -n observability --timeout=120s || true
    kubectl wait --for=condition=ready pod -l app=prometheus -n observability --timeout=120s || true

    log_success "Observability stack is ready"
}

# Main execution
main() {
    log_info "========================================"
    log_info "Deploying Observability Stack"
    log_info "========================================"
    echo ""

    # Ensure Helm repos are added
    helm repo add grafana https://grafana.github.io/helm-charts 2>/dev/null || true
    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts 2>/dev/null || true
    helm repo update

    create_namespace
    deploy_loki
    deploy_tempo
    deploy_promtail
    deploy_prometheus
    deploy_grafana
    wait_for_observability

    echo ""
    log_success "========================================"
    log_success "Observability Stack Deployment Complete!"
    log_success "========================================"
    echo ""
    log_info "Access URLs:"
    echo "  - Grafana:     http://localhost:3000 (admin/admin)"
    echo "  - Prometheus:  http://localhost:9090"
    echo "  - Loki:        http://localhost:30310"
    echo "  - Tempo:       http://localhost:30311"
    echo ""
    log_info "Grafana pre-configured with:"
    echo "  - Loki datasource (default)"
    echo "  - Tempo datasource"
    echo "  - Prometheus datasource"
    echo "  - WMS Overview dashboard"
    echo ""
}

# Parse arguments
case "${1:-}" in
    loki)
        create_namespace
        deploy_loki
        ;;
    tempo)
        create_namespace
        deploy_tempo
        ;;
    promtail)
        create_namespace
        deploy_promtail
        ;;
    prometheus)
        helm repo add prometheus-community https://prometheus-community.github.io/helm-charts 2>/dev/null || true
        helm repo update
        create_namespace
        deploy_prometheus
        ;;
    grafana)
        create_namespace
        deploy_grafana
        ;;
    *)
        main
        ;;
esac
