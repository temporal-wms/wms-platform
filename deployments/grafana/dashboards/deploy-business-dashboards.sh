#!/bin/bash

# Deploy Business Metrics Dashboards to Grafana
# This script imports all WMS business dashboards via the Grafana API

set -e

# Configuration
GRAFANA_URL="${GRAFANA_URL:-http://localhost:3000}"
GRAFANA_USER="${GRAFANA_USER:-admin}"
GRAFANA_PASSWORD="${GRAFANA_PASSWORD:-admin}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Dashboard files to deploy
DASHBOARDS=(
  "wms-receiving-business.json"
  "wms-stow-business.json"
  "wms-sortation-business.json"
  "wms-walling-business.json"
  "wms-unit-business.json"
  "wms-process-path-business.json"
  "wms-wes-business.json"
)

echo "=================================================="
echo "Deploying WMS Business Metrics Dashboards"
echo "=================================================="
echo "Grafana URL: $GRAFANA_URL"
echo "Dashboard directory: $SCRIPT_DIR"
echo ""

# Check if Grafana is accessible
echo -n "Checking Grafana connectivity... "
if curl -s -o /dev/null -w "%{http_code}" "$GRAFANA_URL/api/health" | grep -q "200"; then
  echo -e "${GREEN}OK${NC}"
else
  echo -e "${RED}FAILED${NC}"
  echo "Error: Cannot connect to Grafana at $GRAFANA_URL"
  echo "Please ensure Grafana is running and accessible."
  exit 1
fi

# Deploy each dashboard
SUCCESS_COUNT=0
FAILED_COUNT=0

for dashboard_file in "${DASHBOARDS[@]}"; do
  dashboard_path="$SCRIPT_DIR/$dashboard_file"

  echo -n "Deploying $dashboard_file... "

  if [ ! -f "$dashboard_path" ]; then
    echo -e "${RED}FAILED${NC} (file not found)"
    FAILED_COUNT=$((FAILED_COUNT + 1))
    continue
  fi

  # Read dashboard JSON and wrap it in the required format
  dashboard_json=$(cat "$dashboard_path")
  payload=$(jq -n --argjson dashboard "$dashboard_json" '{
    dashboard: $dashboard,
    overwrite: true,
    message: "Deployed via deploy-business-dashboards.sh"
  }')

  # Send to Grafana API
  response=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -u "$GRAFANA_USER:$GRAFANA_PASSWORD" \
    -d "$payload" \
    "$GRAFANA_URL/api/dashboards/db")

  # Check response
  if echo "$response" | jq -e '.status == "success"' > /dev/null 2>&1; then
    dashboard_url=$(echo "$response" | jq -r '.url')
    echo -e "${GREEN}SUCCESS${NC}"
    echo "  → Dashboard URL: $GRAFANA_URL$dashboard_url"
    SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
  else
    echo -e "${RED}FAILED${NC}"
    error_message=$(echo "$response" | jq -r '.message // "Unknown error"')
    echo "  → Error: $error_message"
    FAILED_COUNT=$((FAILED_COUNT + 1))
  fi
done

echo ""
echo "=================================================="
echo "Deployment Summary"
echo "=================================================="
echo -e "Total dashboards: ${#DASHBOARDS[@]}"
echo -e "${GREEN}Successfully deployed: $SUCCESS_COUNT${NC}"
if [ $FAILED_COUNT -gt 0 ]; then
  echo -e "${RED}Failed: $FAILED_COUNT${NC}"
fi
echo ""

if [ $FAILED_COUNT -eq 0 ]; then
  echo -e "${GREEN}All business dashboards deployed successfully!${NC}"
  exit 0
else
  echo -e "${YELLOW}Some dashboards failed to deploy. Please check the errors above.${NC}"
  exit 1
fi
