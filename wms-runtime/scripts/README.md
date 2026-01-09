# WMS Platform - Load Testing Scripts

k6 load testing scripts for the WMS (Warehouse Management System) platform.

## Prerequisites

- [k6](https://k6.io/docs/getting-started/installation/) installed
- WMS services running (order, inventory, labor services)

## Quick Start

```bash
# Run smoke test (quick validation)
k6 run scenarios/smoke.js

# Run load test (normal load)
k6 run scenarios/load.js

# Run stress test (find breaking points)
k6 run scenarios/stress.js

# Run endurance test (1 hour comprehensive test)
k6 run scenarios/endurance.js
```

## Test Scenarios

### Smoke Test (`scenarios/smoke.js`)

Quick validation test to ensure the system is working correctly.

| Parameter | Value |
|-----------|-------|
| Duration | ~1 minute |
| VUs | 1 |
| Iterations | 5 |
| Purpose | Basic functionality validation |

**Use when:** Deploying new changes, quick health check, CI/CD pipelines.

```bash
k6 run scenarios/smoke.js
```

---

### Load Test (`scenarios/load.js`)

Normal load test to validate system under expected traffic.

| Parameter | Value |
|-----------|-------|
| Duration | 7 minutes |
| Max VUs | 10 |
| Stages | Ramp up → Sustain → Ramp down |
| Purpose | Validate normal operation |

**Stages:**
1. Ramp up to 10 VUs over 1 minute
2. Sustain 10 VUs for 5 minutes
3. Ramp down over 1 minute

**Use when:** Regular performance validation, pre-release testing.

```bash
k6 run scenarios/load.js
```

---

### Stress Test (`scenarios/stress.js`)

Heavy load test to find system breaking points.

| Parameter | Value |
|-----------|-------|
| Duration | ~19 minutes |
| Max VUs | 100 |
| Stages | Progressive ramp to peak |
| Purpose | Find system limits |

**Stages:**
1. Warm up to 10 VUs (2 min)
2. Ramp to 25 VUs (3 min)
3. Ramp to 50 VUs (3 min)
4. Ramp to 75 VUs (3 min)
5. Peak at 100 VUs (2 min)
6. Sustain peak (2 min)
7. Scale down (2 min)
8. Recovery (2 min)

**Use when:** Capacity planning, finding bottlenecks, pre-launch validation.

```bash
k6 run scenarios/stress.js
```

---

### Endurance Test (`scenarios/endurance.js`)

Comprehensive 1-hour test with varying load patterns to understand system behavior under realistic conditions.

| Parameter | Value |
|-----------|-------|
| Duration | 60 minutes |
| Max VUs | 80 |
| Stages | 8 distinct phases |
| Purpose | Long-term behavior analysis |

**Phases:**

| Phase | Time | VUs | Description |
|-------|------|-----|-------------|
| 1. Warm-up | 0-5 min | 5→10 | Gentle system warm-up |
| 2. Normal Load | 5-15 min | 15 | Baseline sustained load |
| 3. First Peak | 15-20 min | 15→40→15 | Initial spike test |
| 4. Gradual Ramp | 20-28 min | 20-30 | Progressive load changes |
| 5. Major Peak | 28-34 min | 60→80→50 | High stress period |
| 6. Chaotic Load | 34-44 min | 10-55 | Unpredictable patterns |
| 7. Sustained High | 44-52 min | 45 | Extended high load |
| 8. Final Stress | 52-60 min | 70→0 | Final push & cooldown |

**What it tests:**
- System behavior under normal sustained load
- Response to sudden traffic spikes
- Recovery time after peak loads
- Performance under chaotic/unpredictable load patterns
- Sustained high-load endurance
- Resource exhaustion and memory leak detection
- Long-running stability

**Use when:** Pre-production validation, capacity planning, understanding system behavior, detecting memory leaks.

```bash
k6 run scenarios/endurance.js
```

---

### Picker Simulator (`scenarios/picker-simulator.js`)

Simulates warehouse picker workers processing tasks.

**Use when:** Testing the picking workflow, worker assignment logic.

```bash
k6 run scenarios/picker-simulator.js
```

---

## Configuration

### Environment Variables

Override default service URLs:

```bash
# Order service
export ORDER_SERVICE_URL=http://localhost:8001

# Inventory service
export INVENTORY_SERVICE_URL=http://localhost:8008

# Labor service
export LABOR_SERVICE_URL=http://localhost:8009

# Run with custom URLs
k6 run scenarios/load.js
```

### Thresholds

Three threshold profiles are available in `lib/config.js`:

| Profile | p95 Duration | p99 Duration | Error Rate |
|---------|--------------|--------------|------------|
| `strict` | < 200ms | < 500ms | < 0.1% |
| `default` | < 500ms | < 1000ms | < 1% |
| `relaxed` | < 2000ms | < 5000ms | < 5% |

---

## Output & Metrics

### Custom Metrics

All scenarios track:

| Metric | Type | Description |
|--------|------|-------------|
| `orders_created` | Counter | Successfully created orders |
| `orders_failed` | Counter | Failed order attempts |
| `order_success_rate` | Rate | Success percentage |
| `order_duration` | Trend | Order creation time |
| `priority_same_day` | Counter | Same-day priority orders |
| `priority_next_day` | Counter | Next-day priority orders |
| `priority_standard` | Counter | Standard priority orders |

### Export Results

```bash
# JSON output
k6 run --out json=results.json scenarios/load.js

# CSV output
k6 run --out csv=results.csv scenarios/load.js

# InfluxDB (for Grafana dashboards)
k6 run --out influxdb=http://localhost:8086/k6 scenarios/load.js
```

---

## Project Structure

```
scripts/
├── README.md              # This file
├── scenarios/
│   ├── smoke.js           # Quick validation test
│   ├── load.js            # Normal load test
│   ├── stress.js          # Stress/breaking point test
│   ├── endurance.js       # 1-hour comprehensive test
│   └── picker-simulator.js # Picker workflow simulation
├── lib/
│   ├── config.js          # Configuration and thresholds
│   ├── data.js            # Data generators
│   ├── orders.js          # Order API helpers
│   ├── inventory.js       # Inventory API helpers
│   ├── labor.js           # Labor API helpers
│   └── picking.js         # Picking API helpers
├── data/
│   ├── skus.json          # Product catalog
│   ├── locations.json     # Warehouse locations
│   └── workers.json       # Worker definitions
├── orders.js              # Standalone order test
└── setup.js               # Initial data setup
```

---

## Best Practices

1. **Always run smoke test first** - Validate the system is healthy before longer tests.

2. **Monitor infrastructure** - Run alongside Grafana/Prometheus to correlate k6 metrics with system resources.

3. **Test in isolation** - Run load tests in environments without other traffic.

4. **Gradual increases** - Start with load test before stress test to establish baselines.

5. **Run endurance test for production validation** - The 1-hour test helps identify issues that only appear under sustained load.

---

## Analyzing Results

### Key Questions to Answer

After running tests, analyze:

1. **Response Time**
   - What is the p95/p99 response time?
   - Does it degrade under load?

2. **Error Rate**
   - What percentage of requests fail?
   - What types of errors occur?

3. **Throughput**
   - How many requests/second can the system handle?
   - Where does throughput plateau?

4. **Recovery**
   - How quickly does the system recover after spikes?
   - Does it return to baseline performance?

5. **Resource Usage**
   - CPU/memory correlation with load
   - Database connection pools
   - Network saturation
