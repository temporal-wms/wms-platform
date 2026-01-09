#!/bin/bash
# Setup Debezium connectors and verify data flow

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DATA_MESH_DIR="$(dirname "$SCRIPT_DIR")"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}   Data Mesh Connector Setup${NC}"
echo -e "${BLUE}============================================${NC}"

# Wait for Kafka Connect
wait_for_connect() {
    echo -e "${YELLOW}Waiting for Kafka Connect...${NC}"

    local max_attempts=30
    local attempt=0

    while [ $attempt -lt $max_attempts ]; do
        if kubectl exec -n kafka wms-kafka-connect-0 -- curl -s http://localhost:8083/connectors > /dev/null 2>&1; then
            echo -e "${GREEN}Kafka Connect is ready${NC}"
            return 0
        fi
        attempt=$((attempt + 1))
        echo "  Waiting... ($attempt/$max_attempts)"
        sleep 10
    done

    echo "Kafka Connect not ready after $max_attempts attempts"
    return 1
}

# Check connector status
check_connector_status() {
    local connector_name="$1"

    echo -e "${YELLOW}Checking connector: $connector_name${NC}"

    local status=$(kubectl exec -n kafka wms-kafka-connect-0 -- \
        curl -s http://localhost:8083/connectors/$connector_name/status 2>/dev/null)

    if echo "$status" | grep -q '"state":"RUNNING"'; then
        echo -e "${GREEN}  Connector $connector_name is RUNNING${NC}"
    else
        echo -e "${YELLOW}  Connector $connector_name status: $status${NC}"
    fi
}

# Verify CDC topics
verify_cdc_topics() {
    echo ""
    echo -e "${YELLOW}Verifying CDC topics...${NC}"

    local topics=$(kubectl exec -n kafka wms-kafka-kafka-0 -- \
        /opt/kafka/bin/kafka-topics.sh \
        --bootstrap-server localhost:9092 \
        --list 2>/dev/null | grep "cdc.wms" || true)

    if [ -n "$topics" ]; then
        echo -e "${GREEN}CDC Topics created:${NC}"
        echo "$topics" | while read topic; do
            echo "  - $topic"
        done
    else
        echo "No CDC topics found yet. Debezium may still be initializing."
    fi
}

# Sample CDC data
sample_cdc_data() {
    local topic="$1"

    echo ""
    echo -e "${YELLOW}Sampling data from $topic...${NC}"

    kubectl exec -n kafka wms-kafka-kafka-0 -- \
        /opt/kafka/bin/kafka-console-consumer.sh \
        --bootstrap-server localhost:9092 \
        --topic "$topic" \
        --from-beginning \
        --max-messages 1 \
        --timeout-ms 5000 2>/dev/null || echo "No messages available yet"
}

# Verify MinIO buckets
verify_minio_buckets() {
    echo ""
    echo -e "${YELLOW}Verifying MinIO buckets...${NC}"

    kubectl exec -n data-mesh deployment/minio -- \
        mc alias set myminio http://localhost:9000 admin minio123456 2>/dev/null

    kubectl exec -n data-mesh deployment/minio -- \
        mc ls myminio 2>/dev/null | while read line; do
            echo "  $line"
        done
}

# Check Flink jobs
check_flink_jobs() {
    echo ""
    echo -e "${YELLOW}Checking Flink jobs...${NC}"

    local jobs=$(kubectl get flinkdeployment -n data-mesh -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)

    if [ -n "$jobs" ]; then
        echo -e "${GREEN}Flink deployments:${NC}"
        for job in $jobs; do
            local status=$(kubectl get flinkdeployment -n data-mesh "$job" -o jsonpath='{.status.jobStatus.state}' 2>/dev/null)
            echo "  - $job: $status"
        done
    else
        echo "No Flink jobs deployed yet"
    fi
}

# Check Trino connectivity
check_trino() {
    echo ""
    echo -e "${YELLOW}Checking Trino...${NC}"

    local trino_pod=$(kubectl get pods -n data-mesh -l app=trino,component=coordinator -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

    if [ -n "$trino_pod" ]; then
        echo -e "${GREEN}Trino coordinator: $trino_pod${NC}"

        # List catalogs
        echo "  Catalogs:"
        kubectl exec -n data-mesh "$trino_pod" -- \
            trino --execute "SHOW CATALOGS" 2>/dev/null | while read catalog; do
                echo "    - $catalog"
            done
    else
        echo "Trino not deployed yet"
    fi
}

# Check OpenMetadata
check_openmetadata() {
    echo ""
    echo -e "${YELLOW}Checking OpenMetadata...${NC}"

    local om_pod=$(kubectl get pods -n data-mesh -l app=openmetadata -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

    if [ -n "$om_pod" ]; then
        echo -e "${GREEN}OpenMetadata pod: $om_pod${NC}"

        # Check health
        local health=$(kubectl exec -n data-mesh "$om_pod" -- \
            curl -s http://localhost:8585/api/v1/system/health 2>/dev/null)

        if echo "$health" | grep -q '"status":"UP"'; then
            echo "  Status: UP"
        else
            echo "  Status: Starting..."
        fi
    else
        echo "OpenMetadata not deployed yet"
    fi
}

# Print summary
print_summary() {
    echo ""
    echo -e "${BLUE}============================================${NC}"
    echo -e "${BLUE}   Data Mesh Status Summary${NC}"
    echo -e "${BLUE}============================================${NC}"
    echo ""
    echo -e "${GREEN}Access Points:${NC}"
    echo "  - MinIO Console  : http://localhost:30900 (admin/minio123456)"
    echo "  - Trino UI       : http://localhost:30808"
    echo "  - OpenMetadata   : http://localhost:30585"
    echo ""
    echo -e "${YELLOW}Trino Query Example:${NC}"
    echo '  kubectl exec -it -n data-mesh trino-coordinator-0 -- trino'
    echo '  trino> SHOW SCHEMAS FROM iceberg;'
    echo '  trino> SELECT * FROM iceberg.gold.order_fulfillment_daily LIMIT 10;'
    echo ""
}

# Main
main() {
    wait_for_connect
    check_connector_status "mongodb-cdc-connector"
    verify_cdc_topics
    sample_cdc_data "cdc.wms.orders"
    verify_minio_buckets
    check_flink_jobs
    check_trino
    check_openmetadata
    print_summary
}

main "$@"
