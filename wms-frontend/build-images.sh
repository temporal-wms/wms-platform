#!/bin/bash

set -e

REGISTRY="wms-platform"
TAG="latest"

echo "🐳 Building Docker images for WMS Frontend Microfrontends..."
echo "Registry: $REGISTRY"
echo "Tag: $TAG"
echo ""

APPS=(
  "shell"
  "orders"
  "waves"
  "inventory"
  "picking"
  "packing"
  "shipping"
  "labor"
  "dashboard"
  "receiving"
  "stow"
  "routing"
  "walling"
  "consolidation"
  "sortation"
  "facility"
)

TOTAL=${#APPS[@]}
CURRENT=0

for APP in "${APPS[@]}"; do
  CURRENT=$((CURRENT + 1))
  IMAGE_NAME="${REGISTRY}/wms-${APP}:${TAG}"
  
  echo "[$CURRENT/$TOTAL] Building: $IMAGE_NAME"
  echo "  Dockerfile: apps/${APP}/Dockerfile"
  
  docker build \
    -f "apps/${APP}/Dockerfile" \
    --build-arg APP_NAME="${APP}" \
    -t "${IMAGE_NAME}" \
    . 2>&1 | sed 's/^/    /'
  
  if [ $? -eq 0 ]; then
    echo "  ✅ Success: $IMAGE_NAME"
  else
    echo "  ❌ Failed: $IMAGE_NAME"
    exit 1
  fi
  echo ""
done

echo "🎉 All Docker images built successfully!"
echo ""
echo "Built images:"
docker images | grep "wms-platform/wms-" | head -20
