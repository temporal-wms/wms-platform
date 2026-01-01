# WMS Platform - Network Utilities

This folder contains scripts to manage network access to WMS platform services running in Kubernetes.

## Port Forward Script

The `port-forward.sh` script creates kubectl port-forwards for all WMS platform services that have web UIs, plus the MongoDB database.

### Prerequisites

- `kubectl` installed and configured
- Access to the WMS Kubernetes cluster
- `lsof` command available (for port checking)

### Usage

```bash
# Start all port-forwards
./port-forward.sh start

# Stop all port-forwards
./port-forward.sh stop

# Check status of port-forwards
./port-forward.sh status

# Show help
./port-forward.sh help
```

## Services & Ports

| Service | Host Port | URL | Description |
|---------|-----------|-----|-------------|
| Grafana | 3000 | http://localhost:3000 | Monitoring Dashboard (Logs, Metrics, Traces) |
| Temporal | 8080 | http://localhost:8080 | Workflow Orchestration UI |
| Kafka UI | 8081 | http://localhost:8081 | Kafka Topic & Consumer Management |
| Trino | 8082 | http://localhost:8082 | Distributed SQL Query Engine |
| Airflow | 8083 | http://localhost:8083 | Workflow Scheduler |
| Superset | 8088 | http://localhost:8088 | Business Intelligence Dashboard |
| Tempo | 3200 | http://localhost:3200 | Distributed Tracing Backend |
| OpenMetadata | 8585 | http://localhost:8585 | Data Catalog & Governance |
| Prometheus | 9090 | http://localhost:9090 | Metrics Database |
| MinIO Console | 9001 | http://localhost:9001 | S3-Compatible Object Storage |
| MongoDB | 27017 | mongodb://localhost:27017 | Document Database |

## Credentials

| Service | Username | Password | Notes |
|---------|----------|----------|-------|
| Grafana | admin | *(see secret)* | `kubectl get secret grafana -n observability -o jsonpath='{.data.admin-password}' \| base64 -d` |
| Superset | admin | admin | Default credentials |
| MinIO | admin | minio123456 | Console access |
| MongoDB (root) | root | rootpassword | Admin access |
| MongoDB (app) | wms | wmspassword | Application access |

### MongoDB Connection Strings

```bash
# Root user
mongodb://root:rootpassword@localhost:27017/?authSource=admin

# Application user
mongodb://wms:wmspassword@localhost:27017/?authSource=admin
```

## Port Conflict Resolution

The script avoids port conflicts by mapping services to different host ports:

| Internal Port | Services | Host Port Mapping |
|---------------|----------|-------------------|
| 8080 | Temporal, Kafka UI, Trino, Airflow | 8080, 8081, 8082, 8083 |
| All others | Various | Same as internal port |

If a port is already in use, the script will skip that service and continue with the others.

## Troubleshooting

### Port already in use

If you see "Port X already in use, skipping", another process is using that port:

```bash
# Find what's using a port
lsof -i :8080

# Kill the process if needed
kill -9 <PID>
```

### Namespace not found

If a service fails with "Namespace not found", the required infrastructure may not be deployed:

```bash
# Check available namespaces
kubectl get namespaces

# Deploy missing infrastructure
cd deployments/helm/infrastructure
./deploy.sh
```

### Service not found

If port-forward fails for a specific service:

```bash
# Check if service exists
kubectl get svc -n <namespace>

# Check pod status
kubectl get pods -n <namespace>
```

### Connection refused

If you can't connect to a forwarded port:

```bash
# Check if port-forward is running
./port-forward.sh status

# Check logs for errors
kubectl logs -n <namespace> <pod-name>
```

## File Locations

- **PID File**: `/tmp/wms-port-forwards.pids` - Stores PIDs of running port-forwards
- **Script**: `./port-forward.sh` - Main port-forward manager

## Quick Access Links

After running `./port-forward.sh start`:

- **Monitoring**: http://localhost:3000 (Grafana)
- **Workflows**: http://localhost:8080 (Temporal)
- **Messaging**: http://localhost:8081 (Kafka UI)
- **SQL Queries**: http://localhost:8082 (Trino)
- **Analytics**: http://localhost:8088 (Superset)
- **Data Catalog**: http://localhost:8585 (OpenMetadata)
- **Object Storage**: http://localhost:9001 (MinIO)
