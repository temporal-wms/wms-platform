#!/bin/bash
# Build all microfrontend Docker images

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

APPS=("shell" "orders" "waves" "inventory" "picking" "packing" "shipping" "labor" "dashboard")

echo -e "${YELLOW}Building WMS Frontend Microfrontends${NC}"
echo ""

# Install dependencies if needed
if [ ! -d "$ROOT_DIR/node_modules" ]; then
    echo -e "${YELLOW}Installing dependencies...${NC}"
    cd "$ROOT_DIR" && npm ci
fi

# Build each app
for app in "${APPS[@]}"; do
    if [ -d "$ROOT_DIR/apps/$app" ]; then
        echo -e "${YELLOW}Building $app...${NC}"

        # Build with Vite
        cd "$ROOT_DIR" && npm run build:$app 2>/dev/null || \
            (cd "$ROOT_DIR/apps/$app" && npm run build)

        # Build Docker image
        docker build \
            --build-arg APP_NAME=$app \
            -t wms-platform/wms-$app:latest \
            -f "$ROOT_DIR/Dockerfile" \
            "$ROOT_DIR"

        echo -e "${GREEN}✓ $app built successfully${NC}"
    else
        echo -e "${YELLOW}⚠ Skipping $app (directory not found)${NC}"
    fi
done

echo ""
echo -e "${GREEN}All microfrontends built successfully!${NC}"
echo ""
echo "To load into Kind cluster:"
echo "  for app in ${APPS[*]}; do kind load docker-image wms-platform/wms-\$app:latest --name wms-platform; done"
