#!/bin/bash
# Deploy WMS Frontend to Kubernetes

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
K8S_DIR="$ROOT_DIR/deploy/k8s"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${YELLOW}Deploying WMS Frontend to Kubernetes${NC}"
echo ""

# Check prerequisites
if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}kubectl is not installed${NC}"
    exit 1
fi

if ! kubectl cluster-info &> /dev/null; then
    echo -e "${RED}Cannot connect to Kubernetes cluster${NC}"
    exit 1
fi

# Deploy using Kustomize
echo -e "${YELLOW}Applying Kubernetes manifests...${NC}"
kubectl apply -k "$K8S_DIR"

# Wait for deployments
echo -e "${YELLOW}Waiting for deployments to be ready...${NC}"
kubectl rollout status deployment/wms-shell -n wms-frontend --timeout=120s
kubectl rollout status deployment/wms-orders-mf -n wms-frontend --timeout=120s
kubectl rollout status deployment/wms-waves-mf -n wms-frontend --timeout=120s

echo ""
echo -e "${GREEN}WMS Frontend deployed successfully!${NC}"
echo ""
echo "Access the application:"
echo "  - Shell (NodePort): http://localhost:30080"
echo "  - Shell (Ingress):  http://wms.local (add to /etc/hosts)"
echo ""
echo "To check status:"
echo "  kubectl get pods -n wms-frontend"
