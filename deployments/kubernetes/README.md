# WMS Platform Kubernetes Deployment

This directory contains Kubernetes manifests for deploying the WMS Platform to a Kubernetes cluster.

## Prerequisites

- Kubernetes cluster 1.27+
- kubectl configured with cluster access
- Docker registry access (for pulling images)
- Helm 3.x (for infrastructure components)

## Directory Structure

```
kubernetes/
├── base/                       # Base resources
│   ├── namespace.yaml         # wms-platform namespace
│   ├── serviceaccount.yaml    # Service account and RBAC
│   ├── configmaps/            # Shared configuration
│   └── secrets/               # Secret templates
├── services/                   # Microservice deployments
│   ├── order-service/
│   ├── waving-service/
│   ├── routing-service/
│   ├── picking-service/
│   ├── consolidation-service/
│   ├── packing-service/
│   ├── shipping-service/
│   ├── inventory-service/
│   ├── labor-service/
│   └── orchestrator/
└── infrastructure/             # Infrastructure components
    ├── mongodb/
    ├── kafka/
    └── temporal/
```

## Quick Start

### 1. Create Namespace and Base Resources

```bash
kubectl apply -f base/namespace.yaml
kubectl apply -f base/serviceaccount.yaml
kubectl apply -f base/configmaps/
```

### 2. Configure Secrets

**IMPORTANT**: Update secrets with production values before applying!

```bash
# Edit MongoDB credentials
vi base/secrets/mongodb-credentials.yaml

# Apply secrets
kubectl apply -f base/secrets/
```

### 3. Deploy Infrastructure (MongoDB, Kafka, Temporal)

```bash
# Deploy MongoDB (StatefulSet with 3 replicas)
kubectl apply -f infrastructure/mongodb/

# Deploy Kafka (3-node cluster)
kubectl apply -f infrastructure/kafka/

# Deploy Temporal
kubectl apply -f infrastructure/temporal/
```

Wait for infrastructure to be ready:

```bash
kubectl wait --for=condition=ready pod -l app=mongodb -n wms-platform --timeout=5m
kubectl wait --for=condition=ready pod -l app=kafka -n wms-platform --timeout=5m
kubectl wait --for=condition=ready pod -l app=temporal -n wms-platform --timeout=5m
```

### 4. Deploy Microservices

```bash
# Deploy all services
kubectl apply -f services/

# Or deploy individually
kubectl apply -f services/order-service/
kubectl apply -f services/waving-service/
# ... etc
```

### 5. Verify Deployment

```bash
# Check all pods
kubectl get pods -n wms-platform

# Check services
kubectl get svc -n wms-platform

# Check HPA status
kubectl get hpa -n wms-platform

# Check PDB status
kubectl get pdb -n wms-platform
```

## Resource Configuration

### Per-Service Resources

Each microservice deployment includes:

**Deployment** (`deployment.yaml`):
- Replicas: 3 (default)
- Resource requests: 256Mi memory, 100m CPU
- Resource limits: 512Mi memory, 500m CPU
- Health probes: liveness, readiness, startup
- Rolling update strategy

**Service** (`service.yaml`):
- Type: ClusterIP
- HTTP port: 800X
- gRPC port: 900X

**HorizontalPodAutoscaler** (`hpa.yaml`):
- Min replicas: 3
- Max replicas: 10
- CPU target: 70%
- Memory target: 80%

**PodDisruptionBudget** (`pdb.yaml`):
- Min available: 2

### Orchestrator (Temporal Worker)

The orchestrator has higher resource limits:
- Resource requests: 512Mi memory, 200m CPU
- Resource limits: 1Gi memory, 1000m CPU
- Max replicas: 15

## Environment Variables

All services receive:

- `MONGODB_URI`: MongoDB connection string (from secret)
- `KAFKA_BROKERS`: Kafka broker list
- `TEMPORAL_HOST`: Temporal frontend address
- `OTEL_EXPORTER_OTLP_ENDPOINT`: OpenTelemetry collector endpoint
- `TRACING_ENABLED`: Enable distributed tracing
- `LOG_LEVEL`: Logging level (info, debug, warn, error)

## Monitoring

### Prometheus Metrics

All services expose metrics at `/metrics` on their HTTP port.

Prometheus scraping is enabled via pod annotations:
```yaml
prometheus.io/scrape: "true"
prometheus.io/port: "800X"
prometheus.io/path: "/metrics"
```

### Health Checks

- `/health`: Liveness probe
- `/ready`: Readiness probe

## Scaling

### Manual Scaling

```bash
# Scale a specific service
kubectl scale deployment order-service --replicas=5 -n wms-platform
```

### Autoscaling

HPA is configured for all services. Monitor with:

```bash
kubectl get hpa -n wms-platform -w
```

## Rolling Updates

Updates are performed with zero downtime:

```bash
# Update image
kubectl set image deployment/order-service order-service=wms-platform/order-service:v1.2.0 -n wms-platform

# Monitor rollout
kubectl rollout status deployment/order-service -n wms-platform

# Rollback if needed
kubectl rollout undo deployment/order-service -n wms-platform
```

## Troubleshooting

### View Logs

```bash
# Recent logs
kubectl logs -l app=order-service -n wms-platform --tail=100

# Stream logs
kubectl logs -f deployment/order-service -n wms-platform

# Logs from specific pod
kubectl logs order-service-abc123-xyz -n wms-platform
```

### Debug Pods

```bash
# Describe pod
kubectl describe pod order-service-abc123-xyz -n wms-platform

# Execute commands in pod
kubectl exec -it order-service-abc123-xyz -n wms-platform -- /bin/sh

# Port forward for local access
kubectl port-forward svc/order-service 8001:8001 -n wms-platform
```

### Check Events

```bash
kubectl get events -n wms-platform --sort-by='.lastTimestamp'
```

## Security

### Secrets Management

For production, use external secrets management:

**Option 1: Sealed Secrets**
```bash
kubeseal --format=yaml < base/secrets/mongodb-credentials.yaml > base/secrets/mongodb-credentials-sealed.yaml
```

**Option 2: External Secrets Operator**
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: mongodb-credentials
  namespace: wms-platform
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault
    kind: SecretStore
  target:
    name: mongodb-credentials
  data:
  - secretKey: uri
    remoteRef:
      key: wms-platform/mongodb
      property: uri
```

### Network Policies

Apply network policies to restrict pod-to-pod communication:

```bash
kubectl apply -f base/network-policies/
```

## Backup and Recovery

### MongoDB Backup

```bash
# Create backup job
kubectl create job --from=cronjob/mongodb-backup mongodb-backup-manual -n wms-platform

# Restore from backup
kubectl apply -f infrastructure/mongodb/restore-job.yaml
```

## Production Checklist

Before deploying to production:

- [ ] Update all secrets with production values
- [ ] Configure resource limits based on load testing
- [ ] Set up monitoring dashboards
- [ ] Configure alerting rules
- [ ] Test disaster recovery procedures
- [ ] Review and apply network policies
- [ ] Enable pod security policies
- [ ] Configure backup schedules
- [ ] Document runbook procedures
- [ ] Perform load testing
- [ ] Set up log aggregation
- [ ] Configure distributed tracing

## CI/CD Integration

The platform includes GitHub Actions workflows for:

1. **Continuous Integration** (`.github/workflows/ci.yaml`):
   - Unit tests
   - Integration tests
   - Linting
   - Docker image builds

2. **Continuous Deployment**:
   - Staging deployment (develop branch)
   - Production deployment (main branch, manual approval)

See `.github/workflows/` for details.

## Support

For issues or questions:
- Check logs: `kubectl logs -l app=<service-name> -n wms-platform`
- Review events: `kubectl get events -n wms-platform`
- Consult runbooks in `/docs/runbooks/`
