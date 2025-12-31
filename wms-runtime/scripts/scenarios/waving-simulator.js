// Waving Simulator - K6 Load Test Script
// Simulates wave processing to advance Temporal workflows
//
// Usage:
//   k6 run scripts/scenarios/waving-simulator.js
//   k6 run --vus 2 --duration 5m scripts/scenarios/waving-simulator.js
//
// Environment variables:
//   WAVING_SERVICE_URL  - Waving service URL (default: http://localhost:8002)
//   ORCHESTRATOR_URL    - Orchestrator URL (default: http://localhost:30010)
//   WAVING_DELAY_MS     - Delay between operations in ms (default: 300)
//   MAX_WAVES_PER_ITERATION - Max waves to process per VU iteration (default: 5)

import { sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
import {
  discoverReadyWaves,
  processAllReadyWaves,
} from '../lib/waving.js';

// Custom metrics
const wavesDiscovered = new Counter('waving_waves_discovered');
const wavesProcessed = new Counter('waving_waves_processed');
const wavesFailed = new Counter('waving_waves_failed');
const ordersSignaled = new Counter('waving_orders_signaled');
const waveSuccessRate = new Rate('waving_success_rate');
const waveProcessingTime = new Trend('waving_processing_time');

// Default options - can be overridden via CLI
export const options = {
  scenarios: {
    // Continuous waving simulation
    waving_simulation: {
      executor: 'constant-vus',
      vus: 1,
      duration: '2m',
      gracefulStop: '30s',
    },
  },
  thresholds: {
    'waving_success_rate': ['rate>0.9'],  // 90% success rate
    'waving_processing_time': ['p(95)<30000'],  // 95th percentile under 30s
  },
};

// Setup function - runs once before the test
export function setup() {
  console.log('='.repeat(60));
  console.log('Waving Simulator Starting');
  console.log('='.repeat(60));

  // Initial discovery to check connectivity
  const waves = discoverReadyWaves();
  console.log(`Initial discovery found ${waves.length} ready waves`);

  return {
    startTime: new Date().toISOString(),
    initialWaveCount: waves.length,
  };
}

// Main test function - runs for each VU iteration
export default function () {
  const startTime = Date.now();

  // Discover and process ready waves
  const results = processAllReadyWaves();

  const processingTime = Date.now() - startTime;

  // Update metrics
  wavesDiscovered.add(results.discovered);
  wavesProcessed.add(results.processed);
  wavesFailed.add(results.failed);
  ordersSignaled.add(results.signaledOrders);

  // Calculate success rate for this iteration
  if (results.discovered > 0) {
    for (const wave of results.waves) {
      waveSuccessRate.add(wave.success);
      waveProcessingTime.add(processingTime / results.waves.length);
    }
  }

  // Log iteration summary
  console.log(`[VU ${__VU}] Iteration complete: ${results.processed}/${results.discovered} waves processed, ${results.signaledOrders} orders signaled`);

  // Sleep between iterations if no waves found
  if (results.discovered === 0) {
    console.log(`[VU ${__VU}] No ready waves found, waiting before retry...`);
    sleep(5);  // Wait 5 seconds before checking again
  } else {
    sleep(1);  // Brief pause between iterations
  }
}

// Teardown function - runs once after all VUs complete
export function teardown(data) {
  console.log('='.repeat(60));
  console.log('Waving Simulator Complete');
  console.log(`Started: ${data.startTime}`);
  console.log(`Initial ready waves: ${data.initialWaveCount}`);
  console.log('='.repeat(60));
}
