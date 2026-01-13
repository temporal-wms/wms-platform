# WMS Frontend Helm Chart

## Overview

This Helm chart deploys the WMS Frontend microfrontends architecture including:
- **Shell app** (host application - port 3000)
- **15 Microfrontend apps** (remote applications - ports 3001-3015)

## Prerequisites

- Kubernetes 1.23+
- Helm 3.0+
- Docker registry access

## Architecture

```
┌─────────────────────────────────────────┐
│  Ingress (nginx)                   │
│  wms.local → Shell (port 80)     │
└─────────────────────────────────────────┘
                │
                ├── /remotes/{app}/ (Module Federation proxy)
                │
                ├── /mf/{app}/ (direct access)
                │
                ├── /health (health check)
                └── /api/* (Kong - backend routing)
```

## Applications

### Shell App
- **Name**: `wms-shell`
- **Type**: Host application
- **Port**: 80
- **Routes**: `/` (catch-all), proxies to microfrontends
- **Resources**: 64-128Mi, 50-100m CPU

### Microfrontends
| App | Image | Service | Port | Resources |
|------|-------|---------|------|-----------|
| orders | `wms-platform/wms-orders` | `wms-orders-mf` | 80 | 32-64Mi, 25-50m |
| waves | `wms-platform/wms-waves` | `wms-waves-mf` | 80 | 32-64Mi, 25-50m |
| inventory | `wms-platform/wms-inventory` | `wms-inventory-mf` | 80 | 32-64Mi, 25-50m |
| picking | `wms-platform/wms-picking` | `wms-picking-mf` | 80 | 32-64Mi, 25-50m |
| packing | `wms-platform/wms-packing` | `wms-packing-mf` | 80 | 32-64Mi, 25-50m |
| shipping | `wms-platform/wms-shipping` | `wms-shipping-mf` | 80 | 32-64Mi, 25-50m |
| labor | `wms-platform/wms-labor` | `wms-labor-mf` | 80 | 32-64Mi, 25-50m |
| dashboard | `wms-platform/wms-dashboard` | `wms-dashboard-mf` | 80 | 32-64Mi, 25-50m |
| receiving | `wms-platform/wms-receiving` | `wms-receiving-mf` | 80 | 32-64Mi, 25-50m |
| stow | `wms-platform/wms-stow` | `wms-stow-mf` | 80 | 32-64Mi, 25-50m |
| routing | `wms-platform/wms-routing` | `wms-routing-mf` | 80 | 32-64Mi, 25-50m |
| walling | `wms-platform/wms-walling` | `wms-walling-mf` | 80 | 32-64Mi, 25-50m |
| consolidation | `wms-platform/wms-consolidation` | `wms-consolidation-mf` | 80 | 32-64Mi, 25-50m |
| sortation | `wms-platform/wms-sortation` | `wms-sortation-mf` | 80 | 32-64Mi, 25-50m |
| facility | `wms-platform/wms-facility` | `wms-facility-mf` | 80 | 32-64Mi, 25-50m |

## Installation

### Install to Kubernetes

```bash
# Install with default values
helm install wms-frontend ./deploy/helm/wms-frontend \
  --namespace wms-frontend \
  --create-namespace

# Install with development values
helm install wms-frontend ./deploy/helm/wms-frontend \
  --namespace wms-frontend \
  --create-namespace \
  --values ./deploy/helm/wms-frontend/values-dev.yaml

# Install with production values
helm install wms-frontend ./deploy/helm/wms-frontend \
  --namespace wms-frontend \
  --create-namespace \
  --values ./deploy/helm/wms-frontend/values-prod.yaml
```

### Upgrade Existing Installation

```bash
# Upgrade with production values
helm upgrade wms-frontend ./deploy/helm/wms-frontend \
  --namespace wms-frontend \
  --values ./deploy/helm/wms-frontend/values-prod.yaml

# Scale specific app
helm upgrade wms-frontend ./deploy/helm/wms-frontend \
  --namespace wms-frontend \
  --set microfrontends.orders.replicaCount=5

# Enable/disable specific app
helm upgrade wms-frontend ./deploy/helm/wms-frontend \
  --namespace wms-frontend \
  --set microfrontends.receiving.enabled=false
```

### Uninstall

```bash
# Uninstall from namespace
helm uninstall wms-frontend --namespace wms-frontend

# Delete namespace
kubectl delete namespace wms-frontend
```

## Configuration

### Key Parameters

| Parameter | Description | Default |
|-----------|-------------|----------|
| `global.namespace` | Kubernetes namespace | `wms-frontend` |
| `global.imageRegistry` | Docker registry | `wms-platform` |
| `global.imagePullPolicy` | Image pull policy | `IfNotPresent` |
| `shell.enabled` | Enable shell app | `true` |
| `shell.replicaCount` | Shell replica count | `2` |
| `microfrontends.{app}.enabled` | Enable microfrontend | `true` |
| `microfrontends.{app}.replicaCount` | Replica count | `2` |
| `configMap.enabled` | Enable ConfigMap | `true` |
| `ingress.enabled` | Enable Ingress | `true` |
| `ingress.className` | Ingress class | `nginx` |
| `ingress.hosts[*].host` | Host name | `wms.local` |

### Environment Variables

| Variable | Description | Default |
|----------|-------------|----------|
| `VITE_K8S_DEPLOY` | Kubernetes deployment flag | `true` |
| `VITE_API_BASE_URL` | API Gateway URL | `http://api-gateway.wms-platform.svc.cluster.local:8080` |
| `VITE_WS_ENABLED` | WebSocket enabled | `true` |
| `VITE_WS_BASE_URL` | WebSocket URL | `ws://api-gateway.wms-platform.svc.cluster.local:8080` |
| `VITE_{APP}_URL` | Microfrontend URL | `/remotes/{app}` |

## Accessing Applications

### Via Shell (Recommended)

Access through shell app at `http://wms.local/` and navigate to routes:
- `/orders` - Orders management
- `/waves` - Wave management
- `/inventory` - Inventory management
- `/picking` - Picking operations
- `/packing` - Packing operations
- `/shipping` - Shipping operations
- `/labor` - Labor management
- `/dashboard` - Dashboard & metrics
- `/receiving` - Receiving operations
- `/stow` - Stow operations
- `/routing` - Routing operations
- `/walling` - Walling operations
- `/consolidation` - Consolidation operations
- `/sortation` - Sortation operations
- `/facility` - Facility management

### Direct Access

Each microfrontend can be accessed directly via `/mf/{app}/`:
- `http://wms.local/mf/orders/` - Orders microfrontend
- `http://wms.local/mf/waves/` - Waves microfrontend
- ... (all 15 microfrontends)

### Health Checks

All applications expose `/health` endpoint:
- `http://wms.local/health` - Shell health
- `http://wms.local/mf/orders/health` - Orders health
- ... (all 15 microfrontends)

## Building Docker Images

Use the provided build script:

```bash
# Build all images
cd /path/to/wms-frontend
./scripts/build-all-images.sh

# Build specific app
./scripts/build-all-images.sh orders

# Build with custom tag
./scripts/build-all-images.sh --tag v1.2.3

# Build with custom registry
./scripts/build-all-images.sh --registry myregistry.com --tag prod
```

## Troubleshooting

### Check Pod Status

```bash
# All pods in namespace
kubectl get pods -n wms-frontend

# Shell pod
kubectl get pods -n wms-frontend -l app=wms-shell

# Specific microfrontend
kubectl get pods -n wms-frontend -l app=wms-orders-mf
```

### Check Pod Logs

```bash
# Shell logs
kubectl logs -n wms-frontend -l app=wms-shell -f

# Microfrontend logs
kubectl logs -n wms-frontend -l app=wms-orders-mf -f
```

### Check Services

```bash
# All services
kubectl get svc -n wms-frontend

# Shell service
kubectl get svc -n wms-frontend wms-shell

# Microfrontend service
kubectl get svc -n wms-frontend wms-orders-mf
```

### Check Ingress

```bash
# Ingress status
kubectl get ingress -n wms-frontend wms-frontend-ingress

# Ingress details
kubectl describe ingress -n wms-frontend wms-frontend-ingress
```

### Check ConfigMap

```bash
# View ConfigMap
kubectl get configmap -n wms-frontend wms-frontend-config -o yaml

# Edit ConfigMap
kubectl edit configmap -n wms-frontend wms-frontend-config
```

### Common Issues

**Images not pulling**:
- Check image registry access
- Verify image names match `wms-platform/wms-{app}`
- Check `imagePullPolicy` setting

**Ingress not working**:
- Verify nginx ingress controller is installed
- Check `ingressClassName` matches your ingress controller
- Check DNS resolution for `wms.local`

**Module Federation errors**:
- Verify all microfrontends are running and healthy
- Check `VITE_K8S_DEPLOY=true` is set
- Verify `/remotes/{app}/` proxy paths in shell nginx config

**Health checks failing**:
- Check pod status with `kubectl describe pod`
- Check pod logs for errors
- Verify `/health` endpoint exists and returns 200

## Development

For local development, use NodePort instead of Ingress:

```bash
# Install with development values
helm install wms-frontend ./deploy/helm/wms-frontend \
  --namespace wms-frontend \
  --create-namespace \
  --values ./deploy/helm/wms-frontend/values-dev.yaml

# Access shell at NodePort
kubectl get svc -n wms-frontend wms-shell

# Access specific app at NodePort
kubectl get svc -n wms-frontend wms-orders-mf
```

## Support

For issues or questions:
1. Check pod logs
2. Check ConfigMap values
3. Check ingress configuration
4. Review this README for common issues
