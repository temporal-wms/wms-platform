# WMS Platform Helm Chart

Comprehensive Helm chart for deploying the entire Warehouse Management System platform on Kubernetes.

## Overview

This chart deploys 10 microservices:
- **order-service** - Order management (API + Worker)
- **waving-service** - Wave management (API)
- **routing-service** - Route optimization (API)
- **picking-service** - Pick task management (API + Worker)
- **consolidation-service** - Consolidation operations (API + Worker)
- **packing-service** - Packing operations (API + Worker)
- **shipping-service** - Shipping and carrier integration (API + Worker)
- **inventory-service** - Inventory tracking (API)
- **labor-service** - Labor management (API)
- **orchestrator** - Temporal workflow orchestrator (Worker)

## Prerequisites

- Kubernetes 1.24+
- Helm 3.8+
- MongoDB cluster (external or in-cluster)
- Kafka cluster (external or in-cluster)
- Temporal cluster (external or in-cluster)

## Installation

### Development Environment

```bash
# Install with development values
helm install wms-platform ./deployments/helm/wms-platform \
  --namespace wms-platform-dev \
  --create-namespace \
  --values ./deployments/helm/wms-platform/values-dev.yaml

# Or upgrade
helm upgrade --install wms-platform ./deployments/helm/wms-platform \
  --namespace wms-platform-dev \
  --values ./deployments/helm/wms-platform/values-dev.yaml
```

### Staging Environment

```bash
# Create MongoDB secret first
kubectl create secret generic mongodb-credentials-staging \
  --namespace wms-platform-staging \
  --from-literal=uri='mongodb://user:pass@mongodb:27017/?replicaSet=rs0'

# Install
helm install wms-platform ./deployments/helm/wms-platform \
  --namespace wms-platform-staging \
  --create-namespace \
  --values ./deployments/helm/wms-platform/values-staging.yaml
```

### Production Environment

```bash
# Create MongoDB secret
kubectl create secret generic mongodb-credentials \
  --namespace wms-platform \
  --from-literal=uri='mongodb://user:pass@mongodb:27017/?replicaSet=rs0'

# Install
helm install wms-platform ./deployments/helm/wms-platform \
  --namespace wms-platform \
  --create-namespace \
  --values ./deployments/helm/wms-platform/values-prod.yaml
```

## Configuration

### Global Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `global.namespace` | Kubernetes namespace | `wms-platform` |
| `global.imageRegistry` | Docker image registry | `wms-platform` |
| `global.imagePullPolicy` | Image pull policy | `IfNotPresent` |
| `global.environment` | Environment name | `production` |
| `global.region` | Cloud region | `us-east-1` |

### Service Configuration

Each service supports the following configuration:

```yaml
services:
  <service-name>:
    enabled: true                    # Enable/disable service
    type: api|worker|hybrid          # Service type
    replicaCount: 3                  # Number of replicas
    image:
      repository: <service-name>     # Image repository
      tag: latest                    # Image tag

    ports:
      http: 8001                     # HTTP port
      grpc: 9001                     # gRPC port (optional)
      metrics: 8080                  # Metrics port (worker only)

    service:
      type: ClusterIP               # Service type
      annotations: {}               # Service annotations

    resources:
      requests:
        memory: "256Mi"
        cpu: "100m"
      limits:
        memory: "512Mi"
        cpu: "500m"

    autoscaling:
      enabled: true                 # Enable HPA
      minReplicas: 3
      maxReplicas: 10
      targetCPUUtilizationPercentage: 70
      targetMemoryUtilizationPercentage: 80
      behavior:                     # Scale behavior
        scaleDown:
          stabilizationWindowSeconds: 300
        scaleUp:
          stabilizationWindowSeconds: 0

    podDisruptionBudget:
      enabled: true                 # Enable PDB
      minAvailable: 2              # Minimum available pods

    env:                            # Environment variables
      - name: SERVICE_NAME
        value: "service-name"

    envFrom:                        # ConfigMap references
      - configMapRef:
          name: platform-config

    secrets:                        # Secret references
      - name: MONGODB_URI
        valueFrom:
          secretKeyRef:
            name: mongodb-credentials
            key: uri

    healthCheck:
      liveness:
        path: /health
        port: http
        initialDelaySeconds: 30
      readiness:
        path: /ready
        port: http
        initialDelaySeconds: 5
      startup:
        path: /health
        port: http
        failureThreshold: 30

    podAnnotations:                 # Pod annotations
      prometheus.io/scrape: "true"

    terminationGracePeriodSeconds: 30
```

## Platform ConfigMap

The chart creates a shared `platform-config` ConfigMap with common configuration:

```yaml
platformConfig:
  enabled: true
  data:
    # Kafka Configuration
    KAFKA_BROKERS: "kafka-0.kafka-headless.wms-platform.svc.cluster.local:9092,..."
    KAFKA_CONSUMER_GROUP_PREFIX: "wms-platform"
    KAFKA_PRODUCER_ACKS: "all"
    KAFKA_PRODUCER_RETRIES: "3"

    # Temporal Configuration
    TEMPORAL_HOST: "temporal-frontend.wms-platform.svc.cluster.local:7233"
    TEMPORAL_NAMESPACE: "wms-platform"

    # Observability Configuration
    OTEL_EXPORTER_OTLP_ENDPOINT: "http://jaeger-collector.observability.svc.cluster.local:4317"
    TRACING_ENABLED: "true"
    METRICS_ENABLED: "true"

    # Logging Configuration
    LOG_LEVEL: "info"
    LOG_FORMAT: "json"
```

## Secrets Management

### MongoDB Credentials

The chart expects a secret named `mongodb-credentials` (configurable) with the following key:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: mongodb-credentials
  namespace: wms-platform
type: Opaque
stringData:
  uri: "mongodb://user:pass@host:27017/?replicaSet=rs0"
```

For development, you can create it using the chart:

```yaml
mongodb:
  createSecret: true
  uri: "mongodb://mongodb-0.mongodb-headless.wms-platform-dev.svc.cluster.local:27017/?replicaSet=rs0"
```

## Service Types

### API Services
Services that expose HTTP/gRPC endpoints:
- order-service (hybrid - also runs worker)
- waving-service
- routing-service
- inventory-service
- labor-service

### Hybrid Services
Services that run both API and Worker components:
- picking-service
- consolidation-service
- packing-service
- shipping-service

### Worker Services
Services that only run background workers:
- orchestrator (Temporal workflow orchestrator)

## Observability

### Prometheus Metrics

All services expose Prometheus metrics at `/metrics` endpoint:

```yaml
podAnnotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "8001"
  prometheus.io/path: "/metrics"
```

### Distributed Tracing

All services send traces to Jaeger via OpenTelemetry:

```yaml
OTEL_EXPORTER_OTLP_ENDPOINT: "http://jaeger-collector.observability.svc.cluster.local:4317"
TRACING_ENABLED: "true"
```

## High Availability

### Horizontal Pod Autoscaling (HPA)

HPA is enabled by default for all services in production:
- **CPU threshold**: 70%
- **Memory threshold**: 80%
- **Min replicas**: 2-3 (service-dependent)
- **Max replicas**: 6-10 (service-dependent)

Scale behavior:
- **Scale up**: Immediate (0s stabilization window)
- **Scale down**: Gradual (300s stabilization window)

### Pod Disruption Budgets (PDB)

PDBs ensure minimum availability during voluntary disruptions:
- **API services**: minAvailable: 2
- **Worker services**: minAvailable: 1-2

### Rolling Updates

All deployments use RollingUpdate strategy:
- **maxSurge**: 1 (one extra pod during update)
- **maxUnavailable**: 0 (no downtime)

## Resource Management

### Default Resource Allocations

**API Services**:
- Requests: 256Mi memory, 100m CPU
- Limits: 512Mi memory, 500m CPU

**Worker Services** (orchestrator):
- Requests: 512Mi memory, 200m CPU
- Limits: 1Gi memory, 1000m CPU

### Environment-Specific Resources

**Development**:
- Lower resources (128Mi/50m requests)
- No autoscaling
- No PDB
- Single replica

**Staging**:
- Medium resources (256Mi/100m requests)
- Autoscaling enabled
- Minimal PDB (minAvailable: 1)
- 2 replicas minimum

**Production**:
- Full resources (as per service type)
- Full autoscaling
- Strict PDB (minAvailable: 2 for critical services)
- 3+ replicas minimum

## Health Checks

All services implement three types of health checks:

### Liveness Probe
- **Path**: `/health`
- **Initial delay**: 30s
- **Period**: 10s
- **Failure threshold**: 3

### Readiness Probe
- **Path**: `/ready`
- **Initial delay**: 5s
- **Period**: 5s
- **Failure threshold**: 3

### Startup Probe
- **Path**: `/health`
- **Initial delay**: 0s
- **Period**: 5s
- **Failure threshold**: 30 (allows up to 150s startup time)

## Networking

### Service Ports

| Service | HTTP Port | gRPC Port | Metrics Port |
|---------|-----------|-----------|--------------|
| order-service | 8001 | 9001 | - |
| waving-service | 8002 | 9002 | - |
| routing-service | 8003 | 9003 | - |
| picking-service | 8004 | 9004 | - |
| consolidation-service | 8005 | 9005 | - |
| packing-service | 8006 | 9006 | - |
| shipping-service | 8007 | 9007 | - |
| inventory-service | 8008 | 9008 | - |
| labor-service | 8009 | 9009 | - |
| orchestrator | - | - | 8080 |

### Service Discovery

Services communicate using Kubernetes DNS:

```
http://<service-name>.wms-platform.svc.cluster.local:<port>
```

Example:
```
http://order-service.wms-platform.svc.cluster.local:8001/api/v1/orders
```

## Upgrading

### Upgrade Helm Release

```bash
# Development
helm upgrade wms-platform ./deployments/helm/wms-platform \
  --namespace wms-platform-dev \
  --values ./deployments/helm/wms-platform/values-dev.yaml

# Production
helm upgrade wms-platform ./deployments/helm/wms-platform \
  --namespace wms-platform \
  --values ./deployments/helm/wms-platform/values-prod.yaml
```

### Rollback

```bash
# View release history
helm history wms-platform --namespace wms-platform

# Rollback to previous version
helm rollback wms-platform --namespace wms-platform

# Rollback to specific revision
helm rollback wms-platform 3 --namespace wms-platform
```

## Uninstallation

```bash
helm uninstall wms-platform --namespace wms-platform
```

## Troubleshooting

### Check Pod Status

```bash
kubectl get pods -n wms-platform
kubectl describe pod <pod-name> -n wms-platform
kubectl logs <pod-name> -n wms-platform
```

### Check Service Endpoints

```bash
kubectl get svc -n wms-platform
kubectl get endpoints -n wms-platform
```

### Check HPA Status

```bash
kubectl get hpa -n wms-platform
kubectl describe hpa order-service -n wms-platform
```

### Check PDB Status

```bash
kubectl get pdb -n wms-platform
kubectl describe pdb order-service -n wms-platform
```

### Common Issues

**1. ImagePullBackOff**
- Verify image exists in registry
- Check imagePullPolicy setting
- Verify image tag is correct

**2. CrashLoopBackOff**
- Check pod logs: `kubectl logs <pod> -n wms-platform`
- Verify MongoDB connection
- Verify Kafka connection
- Check environment variables

**3. Service Not Ready**
- Check readiness probe configuration
- Verify health endpoint is responding
- Check resource limits (may be OOMKilled)

**4. HPA Not Scaling**
- Verify metrics-server is installed
- Check HPA describe output for errors
- Verify resource requests are set

## Advanced Configuration

### Custom ConfigMap

```yaml
platformConfig:
  enabled: true
  data:
    CUSTOM_CONFIG: "value"
    FEATURE_FLAG_X: "enabled"
```

### Override Individual Service

```yaml
services:
  order-service:
    enabled: true
    replicaCount: 5
    image:
      tag: v2.0.0
    resources:
      requests:
        memory: "512Mi"
        cpu: "200m"
      limits:
        memory: "1Gi"
        cpu: "1000m"
    env:
      - name: CUSTOM_ENV
        value: "custom-value"
```

### Disable Specific Services

```yaml
services:
  labor-service:
    enabled: false  # Don't deploy labor-service
```

## Support

For issues or questions:
- GitHub: https://github.com/wms-platform/wms-platform
- Email: wms@example.com
