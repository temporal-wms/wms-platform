#!/bin/bash
# WMS Platform - Create Temporal Namespace Script
# Creates the 'wms' namespace in Temporal

set -e

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

NAMESPACE="wms"
TEMPORAL_NAMESPACE="temporal"
RETENTION="168h"  # 7 days

# Wait for Temporal to be ready
wait_for_temporal() {
    log_info "Waiting for Temporal frontend to be ready..."

    local retries=30
    local count=0
    while [ $count -lt $retries ]; do
        if kubectl get pods -n "$TEMPORAL_NAMESPACE" -l app.kubernetes.io/component=frontend -o jsonpath='{.items[0].status.phase}' 2>/dev/null | grep -q "Running"; then
            log_success "Temporal frontend is running"
            return 0
        fi
        log_info "Waiting for Temporal frontend... ($((count+1))/$retries)"
        sleep 10
        count=$((count+1))
    done

    log_error "Temporal frontend is not ready after $retries attempts"
    return 1
}

# Create namespace using tctl
create_namespace() {
    log_info "Creating Temporal namespace '$NAMESPACE'..."

    # Get the Temporal admin-tools pod
    local admin_pod
    admin_pod=$(kubectl get pods -n "$TEMPORAL_NAMESPACE" -l app.kubernetes.io/component=admintools -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

    if [ -z "$admin_pod" ]; then
        log_error "Could not find Temporal admin-tools pod"
        log_info "Attempting to create namespace via port-forward..."
        create_namespace_via_port_forward
        return
    fi

    # Check if namespace already exists
    if kubectl exec -n "$TEMPORAL_NAMESPACE" "$admin_pod" -- tctl --namespace "$NAMESPACE" namespace describe >/dev/null 2>&1; then
        log_warning "Namespace '$NAMESPACE' already exists"
        return 0
    fi

    # Create the namespace
    kubectl exec -n "$TEMPORAL_NAMESPACE" "$admin_pod" -- tctl --namespace "$NAMESPACE" namespace register \
        --retention "$RETENTION" \
        --description "WMS Platform namespace"

    log_success "Namespace '$NAMESPACE' created with retention $RETENTION"
}

# Fallback: create namespace via port-forward
create_namespace_via_port_forward() {
    log_info "Setting up port-forward to Temporal frontend..."

    # Start port-forward in background
    kubectl port-forward -n "$TEMPORAL_NAMESPACE" svc/temporal-frontend 7233:7233 &
    local pf_pid=$!
    sleep 5

    # Check if tctl is available locally
    if ! command -v tctl >/dev/null 2>&1; then
        log_warning "tctl not found locally. Please install the Temporal CLI."
        log_info "You can create the namespace manually with:"
        echo "  tctl --namespace $NAMESPACE namespace register --retention $RETENTION"
        kill $pf_pid 2>/dev/null || true
        return 1
    fi

    # Create namespace
    tctl --address localhost:7233 --namespace "$NAMESPACE" namespace register \
        --retention "$RETENTION" \
        --description "WMS Platform namespace" || true

    # Cleanup
    kill $pf_pid 2>/dev/null || true

    log_success "Namespace '$NAMESPACE' created"
}

# Main execution
main() {
    log_info "========================================"
    log_info "Creating Temporal Namespace"
    log_info "========================================"
    echo ""

    wait_for_temporal
    create_namespace

    echo ""
    log_success "========================================"
    log_success "Temporal Namespace Setup Complete!"
    log_success "========================================"
    echo ""
    log_info "Namespace details:"
    echo "  - Name: $NAMESPACE"
    echo "  - Retention: $RETENTION"
    echo ""
}

main "$@"
