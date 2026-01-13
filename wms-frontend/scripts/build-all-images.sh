#!/bin/bash

# Build all WMS Frontend Docker images
# This script builds Docker images for all microfrontends and the shell app

set -e

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# Configuration
REGISTRY="${REGISTRY:-wms-platform}"
TAG="${TAG:-latest}"
DOCKER_BUILDKIT="${DOCKER_BUILDKIT:-1}"
COMPOSE_DOCKER_CLI_BUILD="${COMPOSE_DOCKER_CLI_BUILD:-1}"

export DOCKER_BUILDKIT
export COMPOSE_DOCKER_CLI_BUILD

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# List of all apps
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

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}WMS Frontend Docker Image Builder${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Registry: ${REGISTRY}"
echo "Tag: ${TAG}"
echo "Apps: ${#APPS[@]}"
echo ""

# Function to build an image
build_image() {
  local app=$1
  local image_name="${REGISTRY}/wms-${app}:${TAG}"
  local dockerfile="apps/${app}/Dockerfile"

  echo -e "${YELLOW}[${app}]${NC} Building ${image_name}..."

  # Check if Dockerfile exists
  if [ ! -f "$dockerfile" ]; then
    echo -e "${RED}[${app}]${NC} ERROR: Dockerfile not found at ${dockerfile}"
    return 1
  fi

  # Build the image
  docker build -t "${image_name}" -f "${dockerfile}" .

  if [ $? -eq 0 ]; then
    echo -e "${GREEN}[${app}]${NC} ✓ Successfully built ${image_name}"
  else
    echo -e "${RED}[${app}]${NC} ✗ Failed to build ${image_name}"
    return 1
  fi
}

# Function to build specific app
build_specific() {
  local app=$1
  build_image "$app"
}

# Function to build all images
build_all() {
  local failed=0

  echo -e "${GREEN}Starting build of all apps...${NC}"
  echo ""

  for app in "${APPS[@]}"; do
    build_image "$app"
    if [ $? -ne 0 ]; then
      failed=$((failed + 1))
    fi
  done

  echo ""
  echo -e "${GREEN}========================================${NC}"
  echo -e "${GREEN}Build Summary${NC}"
  echo -e "${GREEN}========================================${NC}"
  echo "Total apps: ${#APPS[@]}"
  echo "Successful: $(( ${#APPS[@]} - failed ))"
  echo "Failed: ${failed}"

  if [ $failed -gt 0 ]; then
    echo -e "${RED}${NC}"
    echo -e "${RED}Some builds failed. Please check the output above.${NC}"
    return 1
  else
    echo -e "${GREEN}${NC}"
    echo -e "${GREEN}All builds completed successfully!${NC}"
    return 0
  fi
}

# Function to show usage
usage() {
  echo "Usage: $0 [OPTIONS] [APP_NAME]"
  echo ""
  echo "Options:"
  echo "  -r, --registry REGISTRY    Docker registry (default: wms-platform)"
  echo "  -t, --tag TAG            Image tag (default: latest)"
  echo "  -h, --help              Show this help message"
  echo ""
  echo "Arguments:"
  echo "  APP_NAME                  Build specific app (optional)"
  echo ""
  echo "Available apps:"
  for app in "${APPS[@]}"; do
    echo "  - $app"
  done
  echo ""
  echo "Examples:"
  echo "  # Build all apps with default settings"
  echo "  $0"
  echo ""
  echo "  # Build specific app"
  echo "  $0 orders"
  echo ""
  echo "  # Build all with custom tag"
  echo "  $0 --tag v1.2.3"
  echo ""
  echo "  # Build with custom registry"
  echo "  $0 --registry myregistry.com --tag prod"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    -r|--registry)
      REGISTRY="$2"
      shift 2
      ;;
    -t|--tag)
      TAG="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    -*)
      echo -e "${RED}Unknown option: $1${NC}"
      usage
      exit 1
      ;;
    *)
      # Build specific app
      build_specific "$1"
      exit $?
      ;;
  esac
done

# If no arguments, build all
if [ $# -eq 0 ]; then
  build_all
  exit $?
fi
