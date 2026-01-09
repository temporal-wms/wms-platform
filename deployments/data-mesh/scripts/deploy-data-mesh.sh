#!/bin/bash
# Data Mesh Deployment Script for WMS Platform
# Deploys all data mesh components to Kubernetes

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DATA_MESH_DIR="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}   WMS Platform - Data Mesh Deployment${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""

# Check prerequisites
check_prerequisites() {
    echo -e "${YELLOW}Checking prerequisites...${NC}"

    if ! command -v kubectl &> /dev/null; then
        echo -e "${RED}kubectl is not installed${NC}"
        exit 1
    fi

    if ! command -v helm &> /dev/null; then
        echo -e "${RED}helm is not installed${NC}"
        exit 1
    fi

    # Check cluster connectivity
    if ! kubectl cluster-info &> /dev/null; then
        echo -e "${RED}Cannot connect to Kubernetes cluster${NC}"
        exit 1
    fi

    echo -e "${GREEN}Prerequisites check passed${NC}"
}

# Add Helm repositories
add_helm_repos() {
    echo ""
    echo -e "${YELLOW}Adding Helm repositories...${NC}"

    helm repo add minio https://charts.min.io/ 2>/dev/null || true
    helm repo add trino https://trinodb.github.io/charts 2>/dev/null || true
    helm repo add openmetadata https://open-metadata.github.io/openmetadata-helm-charts 2>/dev/null || true
    helm repo add apache-airflow https://airflow.apache.org 2>/dev/null || true
    helm repo add flink-operator https://downloads.apache.org/flink/flink-kubernetes-operator-1.6.0/ 2>/dev/null || true
    helm repo add superset https://apache.github.io/superset 2>/dev/null || true

    helm repo update

    echo -e "${GREEN}Helm repositories added${NC}"
}

# Create namespace
create_namespace() {
    echo ""
    echo -e "${YELLOW}Creating data-mesh namespace...${NC}"

    kubectl apply -f "${DATA_MESH_DIR}/namespace.yaml"

    echo -e "${GREEN}Namespace created${NC}"
}

# Deploy MinIO
deploy_minio() {
    echo ""
    echo -e "${YELLOW}Deploying MinIO object storage...${NC}"

    helm upgrade --install minio minio/minio \
        --namespace data-mesh \
        -f "${DATA_MESH_DIR}/minio/values.yaml" \
        --wait --timeout 5m

    # Create buckets
    kubectl apply -f "${DATA_MESH_DIR}/minio/buckets.yaml" -n data-mesh

    echo -e "${GREEN}MinIO deployed${NC}"
    echo -e "  Console: http://localhost:30900"
    echo -e "  API: http://localhost:30901"
}

# Deploy Hive Metastore
deploy_hive_metastore() {
    echo ""
    echo -e "${YELLOW}Deploying Hive Metastore...${NC}"

    kubectl apply -f "${DATA_MESH_DIR}/trino/hive-metastore.yaml"

    # Wait for deployment
    kubectl rollout status deployment/hive-metastore -n data-mesh --timeout=5m

    echo -e "${GREEN}Hive Metastore deployed${NC}"
}

# Deploy Trino
deploy_trino() {
    echo ""
    echo -e "${YELLOW}Deploying Trino query engine...${NC}"

    helm upgrade --install trino trino/trino \
        --namespace data-mesh \
        -f "${DATA_MESH_DIR}/trino/values.yaml" \
        --wait --timeout 5m

    echo -e "${GREEN}Trino deployed${NC}"
    echo -e "  UI: http://localhost:30808"
}

# Deploy Kafka Connect with Debezium
deploy_debezium() {
    echo ""
    echo -e "${YELLOW}Deploying Debezium CDC connectors...${NC}"

    # Deploy Kafka Connect cluster
    kubectl apply -f "${DATA_MESH_DIR}/debezium/kafka-connect-cluster.yaml"

    # Wait for Kafka Connect to be ready
    echo "Waiting for Kafka Connect cluster..."
    sleep 30

    # Deploy MongoDB connector
    kubectl apply -f "${DATA_MESH_DIR}/debezium/mongodb-connector.yaml"

    echo -e "${GREEN}Debezium deployed${NC}"
}

# Deploy Flink
deploy_flink() {
    echo ""
    echo -e "${YELLOW}Deploying Apache Flink...${NC}"

    # Install Flink Kubernetes Operator if not present
    if ! kubectl get crd flinkdeployments.flink.apache.org &> /dev/null; then
        echo "Installing Flink Kubernetes Operator..."
        helm upgrade --install flink-kubernetes-operator flink-operator/flink-kubernetes-operator \
            --namespace data-mesh \
            --wait --timeout 5m
    fi

    # Deploy Flink cluster
    kubectl apply -f "${DATA_MESH_DIR}/flink/flink-cluster.yaml"

    echo -e "${GREEN}Flink deployed${NC}"
}

# Deploy OpenMetadata
deploy_openmetadata() {
    echo ""
    echo -e "${YELLOW}Deploying OpenMetadata data catalog...${NC}"

    helm upgrade --install openmetadata openmetadata/openmetadata \
        --namespace data-mesh \
        -f "${DATA_MESH_DIR}/openmetadata/values.yaml" \
        --wait --timeout 10m

    echo -e "${GREEN}OpenMetadata deployed${NC}"
    echo -e "  UI: http://localhost:30585"
}

# Deploy Apache Superset
deploy_superset() {
    echo ""
    echo -e "${YELLOW}Deploying Apache Superset BI tool...${NC}"

    helm upgrade --install superset superset/superset \
        --namespace data-mesh \
        -f "${DATA_MESH_DIR}/superset/values.yaml" \
        --wait --timeout 10m

    echo -e "${GREEN}Superset deployed${NC}"
    echo -e "  UI: http://localhost:30089"
    echo -e "  Credentials: admin / admin"
    echo ""
    echo -e "${YELLOW}Post-deployment steps:${NC}"
    echo "  1. Access Superset UI at http://localhost:30089"
    echo "  2. Go to Settings → Database Connections → + Database"
    echo "  3. Select 'Trino' and use URI: trino://trino-coordinator.data-mesh:8080/mongodb"
    echo "  4. Import saved queries from: ${DATA_MESH_DIR}/superset/queries/"
}

# Setup ingestion pipelines
setup_ingestion() {
    echo ""
    echo -e "${YELLOW}Setting up ingestion pipelines...${NC}"

    # Wait for OpenMetadata to be ready
    kubectl rollout status deployment/openmetadata -n data-mesh --timeout=5m

    echo "Ingestion pipelines will be configured via OpenMetadata UI"
    echo "  - Kafka ingestion: ${DATA_MESH_DIR}/openmetadata/ingestion/kafka-ingestion.yaml"
    echo "  - Iceberg ingestion: ${DATA_MESH_DIR}/openmetadata/ingestion/iceberg-ingestion.yaml"
    echo "  - MongoDB ingestion: ${DATA_MESH_DIR}/openmetadata/ingestion/mongodb-ingestion.yaml"

    echo -e "${GREEN}Ingestion setup instructions provided${NC}"
}

# Print summary
print_summary() {
    echo ""
    echo -e "${BLUE}============================================${NC}"
    echo -e "${BLUE}   Data Mesh Deployment Complete!${NC}"
    echo -e "${BLUE}============================================${NC}"
    echo ""
    echo -e "${GREEN}Components Deployed:${NC}"
    echo "  - MinIO (Object Storage)    : http://localhost:30900"
    echo "  - Hive Metastore (Catalog)  : thrift://hive-metastore:9083"
    echo "  - Trino (Query Engine)      : http://localhost:30808"
    echo "  - Debezium (CDC)            : Kafka Connect in kafka namespace"
    echo "  - Flink (Stream Processing) : Flink Dashboard"
    echo "  - OpenMetadata (Catalog)    : http://localhost:30585"
    echo "  - Superset (BI/Search)      : http://localhost:30089 (admin/admin)"
    echo ""
    echo -e "${YELLOW}Data Products:${NC}"
    echo "  - orders-dp      : Order lifecycle data"
    echo "  - inventory-dp   : Stock levels and reservations"
    echo "  - fulfillment-dp : End-to-end fulfillment metrics"
    echo "  - labor-dp       : Worker productivity"
    echo "  - shipping-dp    : Carrier performance"
    echo ""
    echo -e "${YELLOW}Data Layers:${NC}"
    echo "  - Bronze: s3://wms-bronze (Raw CDC events)"
    echo "  - Silver: s3://wms-silver (Cleaned data)"
    echo "  - Gold:   s3://wms-gold (Curated metrics)"
    echo ""
    echo -e "${YELLOW}Next Steps:${NC}"
    echo "  1. Access OpenMetadata UI and configure ingestion pipelines"
    echo "  2. Submit Flink jobs for Bronze/Silver/Gold transformations"
    echo "  3. Query data via Trino: trino --server localhost:30808 --catalog iceberg"
    echo ""
    echo -e "${GREEN}Data Mesh is ready!${NC}"
}

# Main execution
main() {
    local component="${1:-all}"

    check_prerequisites

    case "$component" in
        all)
            add_helm_repos
            create_namespace
            deploy_minio
            deploy_hive_metastore
            deploy_trino
            deploy_debezium
            deploy_flink
            # deploy_openmetadata - Skipped: too resource-intensive for Kind clusters
            deploy_superset
            # setup_ingestion - Skipped: requires OpenMetadata
            print_summary
            ;;
        minio)
            create_namespace
            deploy_minio
            ;;
        trino)
            deploy_hive_metastore
            deploy_trino
            ;;
        debezium)
            deploy_debezium
            ;;
        flink)
            deploy_flink
            ;;
        openmetadata)
            deploy_openmetadata
            setup_ingestion
            ;;
        superset)
            add_helm_repos
            deploy_superset
            ;;
        *)
            echo "Usage: $0 [all|minio|trino|debezium|flink|openmetadata|superset]"
            exit 1
            ;;
    esac
}

main "$@"
