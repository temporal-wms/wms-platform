// Shipping Simulator - K6 Load Test Script
// Simulates shipping work to advance Temporal workflows
//
// Usage:
//   k6 run scripts/scenarios/shipping-simulator.js
//   k6 run --vus 2 --duration 5m scripts/scenarios/shipping-simulator.js
//
// Environment variables:
//   SHIPPING_SERVICE_URL - Shipping service URL (default: http://localhost:8007)
//   ORCHESTRATOR_URL     - Orchestrator URL (default: http://localhost:30010)
//   SHIPPING_DELAY_MS    - Delay between operations in ms (default: 500)
//   MAX_SHIPMENTS        - Max shipments to process per VU iteration (default: 10)
//   DEFAULT_CARRIER      - Default shipping carrier (default: UPS)

import { sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
import {
  discoverPendingShipments,
  processAllPendingShipments,
} from '../lib/shipping.js';

// Custom metrics
const shipmentsDiscovered = new Counter('shipping_shipments_discovered');
const shipmentsProcessed = new Counter('shipping_shipments_processed');
const shipmentsFailed = new Counter('shipping_shipments_failed');
const shipSuccessRate = new Rate('shipping_success_rate');
const shipProcessingTime = new Trend('shipping_processing_time');

// Default options - can be overridden via CLI
export const options = {
  scenarios: {
    // Continuous shipping simulation
    shipping_simulation: {
      executor: 'constant-vus',
      vus: 1,
      duration: '2m',
      gracefulStop: '30s',
    },
  },
  thresholds: {
    'shipping_success_rate': ['rate>0.9'],  // 90% success rate
    'shipping_processing_time': ['p(95)<30000'],  // 95th percentile under 30s
  },
};

// Setup function - runs once before the test
export function setup() {
  console.log('='.repeat(60));
  console.log('Shipping Simulator Starting');
  console.log('='.repeat(60));

  // Initial discovery to check connectivity
  const shipments = discoverPendingShipments();
  console.log(`Initial discovery found ${shipments.length} pending shipments`);

  return {
    startTime: new Date().toISOString(),
    initialShipmentCount: shipments.length,
  };
}

// Main test function - runs for each VU iteration
export default function () {
  const startTime = Date.now();

  // Discover and process pending shipments
  const results = processAllPendingShipments();

  const processingTime = Date.now() - startTime;

  // Update metrics
  shipmentsDiscovered.add(results.discovered);
  shipmentsProcessed.add(results.processed);
  shipmentsFailed.add(results.failed);

  // Calculate success rate for this iteration
  if (results.discovered > 0) {
    for (const shipment of results.shipments) {
      shipSuccessRate.add(shipment.success);
      shipProcessingTime.add(processingTime / results.shipments.length);
    }
  }

  // Log iteration summary
  console.log(`[VU ${__VU}] Iteration complete: ${results.processed}/${results.discovered} shipments processed`);

  // Sleep between iterations if no shipments found
  if (results.discovered === 0) {
    console.log(`[VU ${__VU}] No pending shipments found, waiting before retry...`);
    sleep(5);  // Wait 5 seconds before checking again
  } else {
    sleep(1);  // Brief pause between iterations
  }
}

// Teardown function - runs once after all VUs complete
export function teardown(data) {
  console.log('='.repeat(60));
  console.log('Shipping Simulator Complete');
  console.log(`Started: ${data.startTime}`);
  console.log(`Initial pending shipments: ${data.initialShipmentCount}`);
  console.log('='.repeat(60));
}
