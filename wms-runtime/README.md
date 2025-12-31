# WMS Runtime - K6 Load Testing

Load testing suite for the WMS Platform using [k6](https://k6.io/).

## Prerequisites

- [k6](https://k6.io/docs/get-started/installation/) installed
- WMS Platform running (all services accessible)

## Quick Start

### 1. Run Setup (Required First)

The setup script creates inventory items, receives stock, creates workers, and starts shifts:

```bash
k6 run scripts/setup.js
```

### 2. Run Smoke Test

Validate basic functionality with 5 orders:

```bash
k6 run scripts/scenarios/smoke.js
```

### 3. Run Load Test

Normal load with 10 concurrent users for 7 minutes:

```bash
k6 run scripts/scenarios/load.js
```

### 4. Run Stress Test

Find system limits with up to 100 concurrent users:

```bash
k6 run scripts/scenarios/stress.js
```

## Project Structure

```
wms-runtime/
├── package.json              # Project configuration
├── README.md                 # This file
├── data/
│   ├── skus.json            # Product catalog (110 products across 17 categories)
│   ├── locations.json       # Warehouse locations (8 zones, 80 locations, 110 stock entries)
│   ├── workers.json         # Worker definitions (56 workers across all zones and shifts)
│   └── stations.json        # Station definitions (18 stations across zones)
├── scripts/
│   ├── setup.js             # One-time setup script
│   ├── orders.js            # Main order injection script
│   ├── lib/
│   │   ├── config.js        # Base URLs and thresholds
│   │   ├── data.js          # Data generators
│   │   ├── inventory.js     # Inventory API helpers
│   │   ├── labor.js         # Labor API helpers
│   │   ├── orders.js        # Order API helpers (with gift wrap support)
│   │   └── facility.js      # Facility API helpers (station management)
│   └── scenarios/
│       ├── smoke.js                 # Light validation (5 orders)
│       ├── load.js                  # Normal load (10 VUs, 7 min)
│       ├── stress.js                # Stress test (up to 100 VUs)
│       ├── full-flow-simulator.js   # Complete order fulfillment flow
│       ├── facility-simulator.js    # Station setup and management
│       └── giftwrap-simulator.js    # Gift wrap workflow simulation
```

## Service Endpoints

| Service | Default URL | Purpose |
|---------|-------------|---------|
| Order Service | http://localhost:8001 | Create and manage orders |
| Waving Service | http://localhost:8002 | Wave planning and scheduling |
| Routing Service | http://localhost:8003 | Pick route optimization |
| Picking Service | http://localhost:8004 | Pick task management |
| Consolidation Service | http://localhost:8005 | Order consolidation |
| Packing Service | http://localhost:8006 | Packing task management |
| Shipping Service | http://localhost:8007 | Shipment processing |
| Inventory Service | http://localhost:8008 | Manage inventory and stock |
| Labor Service | http://localhost:8009 | Manage workers and shifts |
| Facility Service | http://localhost:8010 | Station and capability management |
| Orchestrator | http://localhost:30010 | Temporal workflow orchestration |

## Configuration

Override service URLs with environment variables:

```bash
k6 run -e ORDER_SERVICE_URL=http://myhost:8001 scripts/setup.js
k6 run -e INVENTORY_SERVICE_URL=http://myhost:8008 scripts/setup.js
k6 run -e LABOR_SERVICE_URL=http://myhost:8009 scripts/setup.js
```

## Test Scenarios

### Smoke Test
- **Purpose**: Validate basic functionality
- **VUs**: 1
- **Iterations**: 5
- **Duration**: ~30 seconds

### Load Test
- **Purpose**: Normal traffic simulation
- **VUs**: 0 → 10 → 0
- **Duration**: 7 minutes
- **Expected Orders**: 500-1000

### Stress Test
- **Purpose**: Find breaking points
- **VUs**: 0 → 100 → 0
- **Duration**: ~19 minutes
- **Stages**: Gradual ramp to 100 VUs

### Full Flow Simulator
- **Purpose**: End-to-end order fulfillment simulation
- **VUs**: 1
- **Phases**: Facility Setup → Order Creation → Waving → Picking → Consolidation → Gift Wrap → Packing → Shipping
- **Gift Wrap**: 20% of orders include gift wrap (configurable)

```bash
k6 run scripts/scenarios/full-flow-simulator.js

# With custom order count
k6 run -e MAX_ORDERS_PER_RUN=20 scripts/scenarios/full-flow-simulator.js

# Skip facility setup (if stations already exist)
k6 run -e SKIP_FACILITY_SETUP=true scripts/scenarios/full-flow-simulator.js

# Increase gift wrap ratio to 50%
k6 run -e GIFTWRAP_ORDER_RATIO=0.5 scripts/scenarios/full-flow-simulator.js
```

### Facility Simulator
- **Purpose**: Set up and manage warehouse stations
- **VUs**: 1
- **Creates**: Packing, consolidation, shipping, and picking stations
- **Capabilities**: Gift wrap, hazmat, oversized, fragile handling

```bash
k6 run scripts/scenarios/facility-simulator.js

# Cleanup stations after run
k6 run -e CLEANUP_STATIONS=true scripts/scenarios/facility-simulator.js
```

### Gift Wrap Simulator
- **Purpose**: Simulate gift wrap workflow processing
- **VUs**: 1
- **Duration**: 5 minutes (continuous processing)
- **Finds**: Available gift wrap stations and processes pending orders

```bash
k6 run scripts/scenarios/giftwrap-simulator.js

# Adjust processing delay
k6 run -e GIFTWRAP_DELAY_MS=1500 scripts/scenarios/giftwrap-simulator.js
```

## Test Data

### Products (110 SKUs across 17 categories)
- **Electronics**: Laptops (8), Phones (8), Tablets (6), Monitors (6), Cameras (6)
- **Peripherals**: Keyboards (6), Mice (6), Speakers (6)
- **Audio/Wearables**: Headphones (8), Watches (6)
- **Accessories**: Chargers (8), Cables (8), Cases (8), Power Banks (6)
- **Storage**: SSDs, HDDs, Memory Cards (8)
- **Gaming**: Controllers, Headsets, Chairs (8)
- **Smart Home/Network**: Routers, Smart Devices (14)

### Warehouse Locations (8 Zones, 80 Locations)
- **ZONE-A**: Electronics - High Value (Laptops, Monitors) - 10 locations
- **ZONE-B**: Electronics - Mobile (Phones, Tablets) - 10 locations
- **ZONE-C**: Audio & Wearables (Headphones, Watches) - 10 locations
- **ZONE-D**: Peripherals (Keyboards, Mice) - 8 locations
- **ZONE-E**: Accessories (Chargers, Power Banks) - 10 locations
- **ZONE-F**: Small Items (Cables, Cases) - 12 locations
- **ZONE-G**: Storage & Gaming - 10 locations
- **ZONE-H**: Smart Home & Network - 10 locations

### Workers (56 across all zones and shifts)
- **7 workers per zone** (8 zones = 56 total)
- **Morning shift**: 5 workers per zone (pickers, packers, receivers, team lead)
- **Afternoon shift**: 1 worker per zone
- **Night shift**: 1 worker per zone
- **Skills distribution**:
  - Pickers: 32 workers
  - Packers: 16 workers
  - Receivers/Replenishment: 16 workers
  - Multi-skilled: 8 team leads (one per zone)
  - All skill levels (1-5) represented

### Stations (18 across 6 zones)
- **Packing Stations (8)**:
  - Gift Wrap: 3 stations (zone-a, zone-b, zone-a) with gift_wrap capability
  - Standard: 3 stations (zone-a, zone-b, zone-c)
  - Hazmat: 1 station (zone-d)
  - Oversized: 1 station (zone-d)
- **Consolidation Stations (3)**:
  - Multi-item handling in zones a, b, c
- **Shipping Stations (4)**:
  - Standard: 2 stations (zone-e)
  - Oversized: 1 station (zone-e)
  - Hazmat: 1 station (zone-f)
- **Picking Stations (3)**:
  - Standard: 2 stations (zone-a, zone-b)
  - Hazmat: 1 station (zone-d)
- **Capabilities**: gift_wrap, hazmat, oversized, heavy, fragile, single_item, multi_item, high_value, premium_wrap

## Metrics

### Custom Metrics
- `orders_created`: Counter of successful orders
- `orders_failed`: Counter of failed orders
- `order_success_rate`: Success rate percentage
- `order_duration`: Response time trend

### Full Flow Metrics
- `flow_orders_created`: Orders created in flow
- `flow_orders_completed`: Successfully shipped orders
- `flow_e2e_latency`: End-to-end latency
- `flow_stage_*_processed`: Items processed per stage
- `flow_giftwrap_orders`: Orders with gift wrap
- `flow_facility_stations_created`: Stations created

### Facility Metrics
- `facility_stations_created`: Stations created
- `facility_stations_active`: Currently active stations
- `facility_capability_changes`: Capability modifications
- `facility_api_success_rate`: API success rate

### Gift Wrap Metrics
- `giftwrap_tasks_completed`: Gift wrap tasks completed
- `giftwrap_success_rate`: Processing success rate
- `giftwrap_processing_time`: Time to complete gift wrap
- `giftwrap_station_utilization`: Station utilization percentage

### Thresholds
- Default: 95th percentile < 500ms, error rate < 1%
- Strict: 95th percentile < 200ms, error rate < 0.1%
- Relaxed: 95th percentile < 2000ms, error rate < 5%

## Order Generation

Orders are generated with:
- **Priority Distribution**: 70% standard, 25% next_day, 5% same_day
- **Items**: 1-3 random products per order
- **Addresses**: Random US addresses from 15 major cities
- **Delivery Dates**: Based on priority (same day to 7 days)

## Troubleshooting

### Setup Fails
1. Ensure all services are running
2. Check service health endpoints:
   ```bash
   curl http://localhost:8001/health
   curl http://localhost:8008/health
   curl http://localhost:8009/health
   ```

### Orders Failing
1. Run setup again to ensure inventory exists
2. Check worker shifts are active
3. Verify Temporal is running for workflow processing

### High Error Rates
1. Check service logs for errors
2. Verify MongoDB and Kafka connectivity
3. Review Temporal workflow status at http://localhost:8088

## Running with Docker

If k6 is not installed locally:

```bash
docker run --rm -i --network=host \
  -v $(pwd):/scripts \
  grafana/k6 run /scripts/scripts/setup.js
```

## Output Formats

### JSON Output
```bash
k6 run --out json=results.json scripts/scenarios/load.js
```

### InfluxDB Output
```bash
k6 run --out influxdb=http://localhost:8086/k6 scripts/scenarios/load.js
```

### Cloud Output (k6 Cloud)
```bash
k6 cloud scripts/scenarios/load.js
```
