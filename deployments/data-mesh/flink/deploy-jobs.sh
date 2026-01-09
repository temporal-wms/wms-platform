#!/bin/bash

# WMS Platform - Flink SQL Jobs Deployment Script
# Deploys Bronze, Silver, and Gold layer jobs to the Flink cluster

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE="data-mesh"
FLINK_CLUSTER="wms-flink-cluster"

print_banner() {
    echo -e "${BLUE}"
    echo "╔═══════════════════════════════════════════════════════════╗"
    echo "║         WMS Platform - Flink SQL Jobs Deployment          ║"
    echo "╚═══════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

check_prerequisites() {
    echo -e "${YELLOW}Checking prerequisites...${NC}"

    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        echo -e "${RED}Error: kubectl not found${NC}"
        exit 1
    fi

    # Check cluster connectivity
    if ! kubectl cluster-info &> /dev/null; then
        echo -e "${RED}Error: Cannot connect to Kubernetes cluster${NC}"
        exit 1
    fi

    # Check if namespace exists
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        echo -e "${RED}Error: Namespace '$NAMESPACE' not found${NC}"
        exit 1
    fi

    echo -e "${GREEN}✓ Prerequisites check passed${NC}"
}

check_flink_cluster() {
    echo -e "\n${YELLOW}Checking Flink cluster status...${NC}"

    # Check if Flink deployment exists
    if ! kubectl get flinkdeployment "$FLINK_CLUSTER" -n "$NAMESPACE" &> /dev/null 2>&1; then
        echo -e "${YELLOW}Flink cluster not deployed. Deploying now...${NC}"
        kubectl apply -f "$SCRIPT_DIR/flink-cluster.yaml"
        echo -e "${GREEN}✓ Flink cluster deployment initiated${NC}"
        echo -e "${YELLOW}Waiting for cluster to be ready (this may take 2-3 minutes)...${NC}"
        sleep 30
    fi

    # Wait for JobManager to be ready
    local retries=0
    local max_retries=20
    while [ $retries -lt $max_retries ]; do
        local status=$(kubectl get flinkdeployment "$FLINK_CLUSTER" -n "$NAMESPACE" -o jsonpath='{.status.jobManagerDeploymentStatus}' 2>/dev/null || echo "UNKNOWN")
        if [ "$status" == "READY" ]; then
            echo -e "${GREEN}✓ Flink cluster is ready${NC}"
            return 0
        fi
        echo -e "  Waiting for Flink cluster... (status: $status)"
        sleep 10
        ((retries++))
    done

    echo -e "${YELLOW}⚠ Flink cluster may not be fully ready, proceeding anyway...${NC}"
}

create_sql_configmaps() {
    echo -e "\n${YELLOW}Creating SQL job ConfigMaps...${NC}"

    # Bronze ingestion SQL
    kubectl create configmap flink-bronze-sql \
        --from-file=bronze-ingestion.sql="$SCRIPT_DIR/jobs/bronze-ingestion.sql" \
        -n "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    echo -e "${GREEN}✓ Created bronze-ingestion ConfigMap${NC}"

    # Silver transformation SQL
    kubectl create configmap flink-silver-sql \
        --from-file=silver-transformation.sql="$SCRIPT_DIR/jobs/silver-transformation.sql" \
        -n "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    echo -e "${GREEN}✓ Created silver-transformation ConfigMap${NC}"

    # Gold aggregation SQL
    kubectl create configmap flink-gold-sql \
        --from-file=gold-aggregation.sql="$SCRIPT_DIR/jobs/gold-aggregation.sql" \
        -n "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    echo -e "${GREEN}✓ Created gold-aggregation ConfigMap${NC}"
}

deploy_flink_sql_jobs() {
    echo -e "\n${YELLOW}Deploying Flink SQL jobs...${NC}"

    # Create FlinkSessionJob resources for each SQL file
    cat <<EOF | kubectl apply -f -
---
apiVersion: flink.apache.org/v1beta1
kind: FlinkSessionJob
metadata:
  name: wms-bronze-ingestion
  namespace: $NAMESPACE
spec:
  deploymentName: $FLINK_CLUSTER
  job:
    jarURI: local:///opt/flink/opt/flink-sql-client.jar
    entryClass: org.apache.flink.table.client.SqlClient
    args:
      - "-f"
      - "/opt/flink/sql/bronze-ingestion.sql"
    parallelism: 2
    upgradeMode: stateless
  flinkConfiguration:
    taskmanager.numberOfTaskSlots: "2"
---
apiVersion: flink.apache.org/v1beta1
kind: FlinkSessionJob
metadata:
  name: wms-silver-transformation
  namespace: $NAMESPACE
spec:
  deploymentName: $FLINK_CLUSTER
  job:
    jarURI: local:///opt/flink/opt/flink-sql-client.jar
    entryClass: org.apache.flink.table.client.SqlClient
    args:
      - "-f"
      - "/opt/flink/sql/silver-transformation.sql"
    parallelism: 2
    upgradeMode: stateless
  flinkConfiguration:
    taskmanager.numberOfTaskSlots: "2"
---
apiVersion: flink.apache.org/v1beta1
kind: FlinkSessionJob
metadata:
  name: wms-gold-aggregation
  namespace: $NAMESPACE
spec:
  deploymentName: $FLINK_CLUSTER
  job:
    jarURI: local:///opt/flink/opt/flink-sql-client.jar
    entryClass: org.apache.flink.table.client.SqlClient
    args:
      - "-f"
      - "/opt/flink/sql/gold-aggregation.sql"
    parallelism: 2
    upgradeMode: stateless
  flinkConfiguration:
    taskmanager.numberOfTaskSlots: "2"
EOF

    echo -e "${GREEN}✓ Flink SQL jobs submitted${NC}"
}

show_status() {
    echo -e "\n${BLUE}═══ Current Status ═══${NC}"

    echo -e "\n${YELLOW}Flink Deployment:${NC}"
    kubectl get flinkdeployment -n "$NAMESPACE" 2>/dev/null || echo "No FlinkDeployment found"

    echo -e "\n${YELLOW}Flink Session Jobs:${NC}"
    kubectl get flinksessionjob -n "$NAMESPACE" 2>/dev/null || echo "No FlinkSessionJobs found"

    echo -e "\n${YELLOW}Flink Pods:${NC}"
    kubectl get pods -n "$NAMESPACE" -l app=flink 2>/dev/null || echo "No Flink pods found"
}

show_help() {
    print_banner
    echo "Usage: $0 {deploy|status|delete|help}"
    echo ""
    echo "Commands:"
    echo "  deploy  - Deploy Flink cluster and SQL jobs"
    echo "  status  - Show status of Flink cluster and jobs"
    echo "  delete  - Delete all Flink jobs (keeps cluster)"
    echo "  help    - Show this help message"
    echo ""
    echo "Jobs to be deployed:"
    echo "  1. bronze-ingestion    - CDC from Kafka to Bronze Iceberg tables"
    echo "  2. silver-transformation - Bronze to Silver (cleaned, enriched)"
    echo "  3. gold-aggregation    - Silver to Gold (KPIs, metrics)"
}

delete_jobs() {
    echo -e "${YELLOW}Deleting Flink SQL jobs...${NC}"
    kubectl delete flinksessionjob wms-bronze-ingestion -n "$NAMESPACE" 2>/dev/null || true
    kubectl delete flinksessionjob wms-silver-transformation -n "$NAMESPACE" 2>/dev/null || true
    kubectl delete flinksessionjob wms-gold-aggregation -n "$NAMESPACE" 2>/dev/null || true
    echo -e "${GREEN}✓ Jobs deleted${NC}"
}

# Main
print_banner

case "${1:-}" in
    deploy)
        check_prerequisites
        check_flink_cluster
        create_sql_configmaps
        deploy_flink_sql_jobs
        echo -e "\n${GREEN}Deployment complete!${NC}"
        echo -e "${YELLOW}Note: Jobs may take a few minutes to start processing data.${NC}"
        echo -e "Run '$0 status' to check job status."
        ;;
    status)
        show_status
        ;;
    delete)
        delete_jobs
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        show_help
        exit 1
        ;;
esac
