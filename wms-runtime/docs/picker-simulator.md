# Picker Simulator

The Picker Simulator is a k6-based tool that simulates warehouse picker work sessions to advance Temporal workflows in the WMS platform.

## Overview

In a real warehouse, pickers use handheld devices to:
1. Receive assigned pick tasks
2. Navigate to locations
3. Pick items and scan them
4. Complete the task

The simulator automates this process for load testing and development, allowing workflows to progress through the picking phase without manual intervention.

## Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  k6 Simulator   │────▶│   Orchestrator   │────▶│    Temporal     │
│                 │     │  (Signal Bridge) │     │                 │
└─────────────────┘     └──────────────────┘     └─────────────────┘
        │                                                 │
        │                                                 │
        ▼                                                 ▼
┌─────────────────┐                              ┌─────────────────┐
│ Picking Service │                              │    Workflow     │
│   (Task API)    │                              │   (Advances)    │
└─────────────────┘                              └─────────────────┘
```

### Components

1. **k6 Simulator** - Discovers tasks, simulates picking, sends completion signals
2. **Picking Service** - Manages pick task state and item confirmations
3. **Orchestrator Signal Bridge** - Bridges HTTP requests to Temporal signals
4. **Temporal** - Receives signals and advances workflow execution

## Usage

### Basic Usage

```bash
cd wms-runtime
k6 run scripts/scenarios/picker-simulator.js
```

### With Custom Options

```bash
k6 run --duration 5m --vus 3 \
  -e PICKING_SERVICE_URL=http://localhost:8004 \
  -e ORCHESTRATOR_URL=http://localhost:8080 \
  -e PICKER_DELAY_MS=500 \
  -e MAX_TASKS_PER_ITERATION=10 \
  scripts/scenarios/picker-simulator.js
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PICKING_SERVICE_URL` | `http://localhost:8004` | Picking service base URL |
| `ORCHESTRATOR_URL` | `http://localhost:8080` | Orchestrator service URL |
| `PICKER_DELAY_MS` | `500` | Delay between picks (simulates walk time) |
| `MAX_TASKS_PER_ITERATION` | `10` | Max tasks to process per VU iteration |

### Kubernetes Port Forwarding

When running against a Kind cluster, you need to set up port forwarding:

```bash
# Terminal 1: Port forward orchestrator
kubectl port-forward svc/orchestrator -n wms-platform 8080:8080

# Terminal 2: Run simulator (picking-service uses NodePort 8004)
k6 run scripts/scenarios/picker-simulator.js
```

## Workflow

The simulator follows this workflow for each task:

```
1. Discover Tasks
   └── GET /api/v1/tasks?status=assigned

2. For each task:
   ├── Start Task
   │   └── POST /api/v1/tasks/{taskId}/start
   │
   ├── For each item:
   │   ├── Simulate delay (walk time)
   │   └── Confirm Pick
   │       └── POST /api/v1/tasks/{taskId}/pick
   │
   ├── Complete Task
   │   └── POST /api/v1/tasks/{taskId}/complete
   │
   └── Signal Workflow
       └── POST /api/v1/signals/pick-completed (orchestrator)
           └── Temporal.SignalWorkflow("picking-{orderId}", "pickCompleted")
```

## Signal Bridge API

The orchestrator exposes an HTTP endpoint that bridges to Temporal signals:

### Endpoint

```
POST /api/v1/signals/pick-completed
Content-Type: application/json
```

### Request Body

```json
{
  "orderId": "ORD-12345",
  "taskId": "PT-abc123",
  "pickedItems": [
    {
      "sku": "SKU-001",
      "quantity": 2,
      "locationId": "LOC-A1",
      "toteId": "TOTE-xyz"
    }
  ]
}
```

### Response

```json
{
  "success": true,
  "workflowId": "picking-ORD-12345",
  "message": "Signal sent successfully"
}
```

## Metrics

The simulator tracks these custom metrics:

| Metric | Description |
|--------|-------------|
| `picker_tasks_discovered` | Total tasks found |
| `picker_tasks_processed` | Successfully processed tasks |
| `picker_tasks_failed` | Failed task count |
| `picker_task_success_rate` | Success rate (threshold: >90%) |
| `picker_task_processing_time` | Time per task (threshold: p95 <30s) |

## Files

| File | Purpose |
|------|---------|
| `scripts/scenarios/picker-simulator.js` | Main simulator script |
| `scripts/lib/picking.js` | Picking helper functions |
| `scripts/lib/config.js` | Configuration and endpoints |

## Helper Functions

The `picking.js` library exports these functions:

```javascript
// Discover assigned tasks
discoverPendingTasks(status = 'assigned')

// Get a specific task
getPickTask(taskId)

// Start a task (required before picking)
startTask(taskId)

// Confirm an item pick
confirmPick(taskId, sku, quantity, locationId, toteId)

// Complete a task
completeTask(taskId)

// Send workflow signal
sendPickCompletedSignal(orderId, taskId, pickedItems)

// Simulate full picking process
simulatePickingTask(task)

// Process a single task end-to-end
processPickTask(task)

// Process all pending tasks
processAllPendingTasks(maxTasks)
```

## Example Test Run

```
$ k6 run --duration 30s scripts/scenarios/picker-simulator.js

  █ THRESHOLDS

    picker_task_processing_time
    ✓ 'p(95)<30000' p(95)=170

    picker_task_success_rate
    ✓ 'rate>0.9' rate=100.00%

  █ TOTAL RESULTS

    ✓ discover tasks status 200
    ✓ start task status 200
    ✓ confirm pick status 200
    ✓ signal pick completed status 200

    picker_tasks_processed.........: 140     4.32147/s
    picker_task_success_rate.......: 100.00% 140 out of 140
```

## Troubleshooting

### Tasks not being discovered

1. Check if pick tasks exist with status `assigned`:
   ```bash
   curl "http://localhost:8004/api/v1/tasks?status=assigned"
   ```

2. Ensure orders have been created and reached the picking phase:
   ```bash
   k6 run scripts/scenarios/load.js
   ```

### Signal failures

1. Verify orchestrator is running and accessible:
   ```bash
   curl http://localhost:8080/health
   ```

2. Check Temporal is running:
   ```bash
   kubectl get pods -n wms-platform | grep temporal
   ```

3. Verify workflow exists:
   - Open Temporal UI at `http://localhost:8088`
   - Search for workflow ID `picking-{orderId}`

### Confirm pick failures

1. Task must be started first (status: `in_progress`)
2. SKU and locationId must match task items exactly
3. Check picking-service logs for detailed errors:
   ```bash
   kubectl logs deployment/picking-service -n wms-platform
   ```

## Integration with Load Testing

For a complete load test cycle:

```bash
# Step 1: Create orders (generates pick tasks)
k6 run --duration 1m scripts/scenarios/load.js

# Step 2: Process pick tasks (advances workflows)
k6 run --duration 2m scripts/scenarios/picker-simulator.js

# Step 3: Verify workflows completed
# Check Temporal UI for workflow status
```
