#!/bin/bash

# WMS Platform - Port Forward Script
# This script manages kubectl port-forwards for infrastructure services only
#
# ╔════════════════════════════════════════════════════════════════════════════╗
# ║                    IMPORTANT: Kong API Gateway Integration                 ║
# ╚════════════════════════════════════════════════════════════════════════════╝
#
# WMS SERVICES ARE NOW ACCESSIBLE VIA KONG API GATEWAY:
#   All WMS microservices (order-service, picking-service, etc.) are exposed
#   through Kong Gateway at: http://localhost:8888/{service-name}/api/v1/...
#
#   NO PORT-FORWARDING NEEDED FOR WMS SERVICES!
#
#   Examples:
#     - Order Service:    http://localhost:8888/order-service/api/v1/orders
#     - Picking Service:  http://localhost:8888/picking-service/api/v1/picks
#     - Inventory:        http://localhost:8888/inventory-service/api/v1/stock
#
# INFRASTRUCTURE SERVICES STILL NEED PORT-FORWARDING:
#   This script manages port-forwards for infrastructure UIs and databases:
#     - Grafana (monitoring)
#     - Temporal UI (workflow orchestration)
#     - Kafka UI (message broker management)
#     - Prometheus (metrics)
#     - MongoDB (direct database access)
#     - Trino (data mesh query engine)
#     - Superset (BI dashboard)
#     - MinIO Console (object storage)
#
# Kong Gateway Access:
#   - Port-forward Kong Gateway: kubectl port-forward -n kong svc/kong-gateway 8888:80
#   - Or use the emulator with default Kong settings (no port-forward needed)
#
# For legacy direct service access (debugging/development):
#   Set USE_KONG=false in your environment to use direct port-forward mode
#
# ══════════════════════════════════════════════════════════════════════════════

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# PID file location
PID_FILE="/tmp/wms-port-forwards.pids"

# Service definitions: "name|namespace|service|internal_port|host_port|description"
SERVICES=(
    "Grafana|observability|svc/grafana|3000|3000|Monitoring Dashboard"
    "Temporal|temporal|svc/temporal-web|8080|8080|Workflow Orchestration UI"
    "Kafka UI|kafka|svc/kafka-ui|8080|8081|Kafka Management"
    "Prometheus|observability|svc/prometheus-server|80|9090|Metrics Database"
    "Tempo|observability|svc/tempo|3200|3200|Distributed Tracing"
    "Trino|data-mesh|svc/trino|8080|8082|SQL Query Engine"
    "Superset|data-mesh|svc/superset|8088|8088|BI Dashboard"
    "MinIO Console|data-mesh|svc/minio-console|9001|9001|Object Storage Console"
    "OpenMetadata|data-mesh|svc/openmetadata|8585|8585|Data Catalog"
    "Airflow|data-mesh|svc/airflow-api-server|8080|8083|Workflow Scheduler"
    "MongoDB|mongodb|svc/mongodb-headless|27017|27017|Document Database"
    # Microfrontends
    "WMS Shell|wms-frontend|svc/wms-shell|80|3100|Main Frontend App"
    "Orders MF|wms-frontend|svc/wms-orders-mf|80|3101|Orders Microfrontend"
    "Waves MF|wms-frontend|svc/wms-waves-mf|80|3102|Waves Microfrontend"
    "Inventory MF|wms-frontend|svc/wms-inventory-mf|80|3103|Inventory Microfrontend"
    "Picking MF|wms-frontend|svc/wms-picking-mf|80|3104|Picking Microfrontend"
    "Packing MF|wms-frontend|svc/wms-packing-mf|80|3105|Packing Microfrontend"
    "Shipping MF|wms-frontend|svc/wms-shipping-mf|80|3106|Shipping Microfrontend"
    "Labor MF|wms-frontend|svc/wms-labor-mf|80|3107|Labor Microfrontend"
    "Dashboard MF|wms-frontend|svc/wms-dashboard-mf|80|3108|Dashboard Microfrontend"
)

# Print banner
print_banner() {
    echo -e "${BLUE}"
    echo "╔═══════════════════════════════════════════════════════════╗"
    echo "║           WMS Platform - Port Forward Manager             ║"
    echo "╚═══════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

# Check if a port is in use
is_port_in_use() {
    local port=$1
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Check if kubectl is available
check_kubectl() {
    if ! command -v kubectl &> /dev/null; then
        echo -e "${RED}Error: kubectl is not installed or not in PATH${NC}"
        exit 1
    fi

    if ! kubectl cluster-info &> /dev/null; then
        echo -e "${RED}Error: Cannot connect to Kubernetes cluster${NC}"
        exit 1
    fi
}

# Start all port-forwards
start_forwards() {
    print_banner
    check_kubectl

    echo -e "${YELLOW}Starting port-forwards...${NC}\n"

    # Clear existing PID file
    > "$PID_FILE"

    local started=0
    local skipped=0
    local failed=0

    for service in "${SERVICES[@]}"; do
        IFS='|' read -r name namespace svc internal_port host_port description <<< "$service"

        # Check if port is already in use
        if is_port_in_use "$host_port"; then
            echo -e "${YELLOW}⚠ ${name}${NC} - Port ${host_port} already in use, skipping"
            ((skipped++))
            continue
        fi

        # Check if namespace exists
        if ! kubectl get namespace "$namespace" &> /dev/null; then
            echo -e "${RED}✗ ${name}${NC} - Namespace '${namespace}' not found"
            ((failed++))
            continue
        fi

        # Start port-forward
        kubectl port-forward -n "$namespace" "$svc" "${host_port}:${internal_port}" &> /dev/null &
        local pid=$!

        # Wait a moment and check if process is still running
        sleep 0.5
        if ps -p $pid > /dev/null 2>&1; then
            echo "$pid|$name|$host_port" >> "$PID_FILE"
            echo -e "${GREEN}✓ ${name}${NC} - http://localhost:${host_port} (${description})"
            ((started++))
        else
            echo -e "${RED}✗ ${name}${NC} - Failed to start port-forward"
            ((failed++))
        fi
    done

    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo -e "Started: ${GREEN}${started}${NC} | Skipped: ${YELLOW}${skipped}${NC} | Failed: ${RED}${failed}${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "PIDs saved to: ${PID_FILE}"
    echo -e "Run '${YELLOW}$0 stop${NC}' to stop all port-forwards"
}

# Stop all port-forwards
stop_forwards() {
    print_banner

    if [[ ! -f "$PID_FILE" ]]; then
        echo -e "${YELLOW}No port-forwards are currently running (PID file not found)${NC}"
        return
    fi

    echo -e "${YELLOW}Stopping port-forwards...${NC}\n"

    local stopped=0
    local not_running=0

    while IFS='|' read -r pid name port; do
        if ps -p "$pid" > /dev/null 2>&1; then
            kill "$pid" 2>/dev/null
            echo -e "${GREEN}✓ Stopped ${name}${NC} (PID: ${pid}, Port: ${port})"
            ((stopped++))
        else
            echo -e "${YELLOW}⚠ ${name}${NC} was not running (PID: ${pid})"
            ((not_running++))
        fi
    done < "$PID_FILE"

    rm -f "$PID_FILE"

    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo -e "Stopped: ${GREEN}${stopped}${NC} | Already stopped: ${YELLOW}${not_running}${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
}

# Show status of port-forwards
show_status() {
    print_banner

    if [[ ! -f "$PID_FILE" ]]; then
        echo -e "${YELLOW}No port-forwards are currently managed (PID file not found)${NC}"
        echo ""
        echo "Run '$0 start' to start port-forwards"
        return
    fi

    echo -e "${BLUE}Current Port-Forward Status:${NC}\n"

    printf "%-20s %-10s %-8s %s\n" "SERVICE" "PORT" "STATUS" "PID"
    echo "─────────────────────────────────────────────────────────"

    local running=0
    local stopped=0

    while IFS='|' read -r pid name port; do
        if ps -p "$pid" > /dev/null 2>&1; then
            printf "%-20s %-10s ${GREEN}%-8s${NC} %s\n" "$name" "$port" "RUNNING" "$pid"
            ((running++))
        else
            printf "%-20s %-10s ${RED}%-8s${NC} %s\n" "$name" "$port" "STOPPED" "$pid"
            ((stopped++))
        fi
    done < "$PID_FILE"

    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo -e "Running: ${GREEN}${running}${NC} | Stopped: ${RED}${stopped}${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
}

# Show help
show_help() {
    print_banner
    echo "Usage: $0 {start|stop|status|help}"
    echo ""
    echo "Commands:"
    echo "  start   - Start all port-forwards"
    echo "  stop    - Stop all port-forwards"
    echo "  status  - Show status of port-forwards"
    echo "  help    - Show this help message"
    echo ""
    echo "Services that will be forwarded:"
    echo ""
    printf "%-20s %-12s %s\n" "SERVICE" "HOST PORT" "DESCRIPTION"
    echo "─────────────────────────────────────────────────────────────"
    for service in "${SERVICES[@]}"; do
        IFS='|' read -r name namespace svc internal_port host_port description <<< "$service"
        printf "%-20s %-12s %s\n" "$name" "$host_port" "$description"
    done
    echo ""
}

# Main
case "${1:-}" in
    start)
        start_forwards
        ;;
    stop)
        stop_forwards
        ;;
    status)
        show_status
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        show_help
        exit 1
        ;;
esac
