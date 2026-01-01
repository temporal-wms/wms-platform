#!/bin/bash
# WMS Platform - Infrastructure Deployment Script
# Deploys MongoDB, Kafka (Strimzi), and Temporal

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOYMENTS_DIR="$(dirname "$SCRIPT_DIR")"
HELM_INFRA_DIR="$DEPLOYMENTS_DIR/helm/infrastructure"

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

# Wait for pod to be ready
wait_for_pods() {
    local namespace=$1
    local label=$2
    local timeout=${3:-300}

    log_info "Waiting for pods with label '$label' in namespace '$namespace'..."
    kubectl wait --for=condition=ready pod -l "$label" -n "$namespace" --timeout="${timeout}s" || {
        log_error "Timeout waiting for pods"
        kubectl get pods -n "$namespace" -l "$label"
        return 1
    }
}

# Deploy MongoDB
deploy_mongodb() {
    log_info "Deploying MongoDB..."

    kubectl create namespace mongodb 2>/dev/null || true

    helm upgrade --install mongodb oci://registry-1.docker.io/bitnamicharts/mongodb \
        -n mongodb \
        -f "$HELM_INFRA_DIR/mongodb/values-prod.yaml" \
        --wait --timeout 5m

    log_success "MongoDB deployed"

    # Wait for replica set to be ready
    log_info "Waiting for MongoDB replica set..."
    sleep 10
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=mongodb -n mongodb --timeout=300s
}

# Deploy Strimzi Kafka Operator
deploy_strimzi_operator() {
    log_info "Deploying Strimzi Kafka Operator..."

    kubectl create namespace kafka 2>/dev/null || true

    helm upgrade --install strimzi-kafka-operator strimzi/strimzi-kafka-operator \
        -n kafka \
        -f "$HELM_INFRA_DIR/kafka/strimzi-operator-values.yaml" \
        --wait --timeout 5m

    log_success "Strimzi operator deployed"

    # Wait for operator to be ready
    wait_for_pods "kafka" "name=strimzi-cluster-operator"
}

# Deploy Kafka Cluster
deploy_kafka_cluster() {
    log_info "Deploying Kafka cluster (KRaft mode)..."

    # Apply Kafka cluster manifests
    kubectl apply -f "$HELM_INFRA_DIR/kafka/kafka-cluster.yaml"

    log_info "Waiting for Kafka cluster to be ready..."
    sleep 30

    # Wait for Kafka pods
    local retries=30
    local count=0
    while [ $count -lt $retries ]; do
        if kubectl get kafka wms-kafka -n kafka -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null | grep -q "True"; then
            log_success "Kafka cluster is ready"
            break
        fi
        log_info "Waiting for Kafka cluster... ($((count+1))/$retries)"
        sleep 10
        count=$((count+1))
    done

    if [ $count -eq $retries ]; then
        log_warning "Kafka cluster may not be fully ready, checking pods..."
        kubectl get pods -n kafka
    fi

    # Deploy Kafka topics
    log_info "Creating Kafka topics..."
    kubectl apply -f "$HELM_INFRA_DIR/kafka/kafka-topics.yaml"

    log_success "Kafka cluster and topics deployed"
}

# Deploy PostgreSQL for Temporal
deploy_temporal_postgresql() {
    log_info "Deploying PostgreSQL for Temporal..."

    kubectl create namespace temporal 2>/dev/null || true

    helm upgrade --install temporal-postgresql oci://registry-1.docker.io/bitnamicharts/postgresql \
        -n temporal \
        -f "$HELM_INFRA_DIR/temporal/postgresql-values.yaml" \
        --wait --timeout 5m

    log_success "PostgreSQL deployed"

    # Wait for PostgreSQL to be ready
    wait_for_pods "temporal" "app.kubernetes.io/name=postgresql"

    # Give PostgreSQL a moment to fully initialize
    sleep 10
}

# Deploy Temporal
deploy_temporal() {
    log_info "Deploying Temporal..."

    helm upgrade --install temporal temporalio/temporal \
        -n temporal \
        -f "$HELM_INFRA_DIR/temporal/temporal-values.yaml" \
        --wait --timeout 10m

    log_success "Temporal deployed"

    # Wait for Temporal frontend to be ready
    log_info "Waiting for Temporal services..."
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/component=frontend -n temporal --timeout=300s || true
}

# Main execution
main() {
    log_info "========================================"
    log_info "Deploying WMS Infrastructure"
    log_info "========================================"
    echo ""

    # Ensure Helm repos are added
    helm repo add bitnami https://charts.bitnami.com/bitnami 2>/dev/null || true
    helm repo add strimzi https://strimzi.io/charts/ 2>/dev/null || true
    helm repo add temporalio https://temporalio.github.io/helm-charts 2>/dev/null || true
    helm repo update

    deploy_mongodb
    deploy_strimzi_operator
    deploy_kafka_cluster
    deploy_temporal_postgresql
    deploy_temporal

    echo ""
    log_success "========================================"
    log_success "Infrastructure Deployment Complete!"
    log_success "========================================"
    echo ""
    log_info "Deployed components:"
    echo "  - MongoDB (mongodb namespace)"
    echo "  - Strimzi Kafka Operator (kafka namespace)"
    echo "  - Kafka Cluster with KRaft (kafka namespace)"
    echo "  - PostgreSQL for Temporal (temporal namespace)"
    echo "  - Temporal (temporal namespace)"
    echo ""
}

# Parse arguments
case "${1:-}" in
    mongodb)
        deploy_mongodb
        ;;
    kafka)
        deploy_strimzi_operator
        deploy_kafka_cluster
        ;;
    temporal)
        deploy_temporal_postgresql
        deploy_temporal
        ;;
    *)
        main
        ;;
esac
