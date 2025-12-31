// Consolidation Simulator - K6 Load Test Script
// Simulates consolidation work to advance Temporal workflows
//
// Usage:
//   k6 run scripts/scenarios/consolidation-simulator.js
//   k6 run --vus 3 --duration 5m scripts/scenarios/consolidation-simulator.js
//
// Environment variables:
//   CONSOLIDATION_SERVICE_URL - Consolidation service URL (default: http://localhost:8005)
//   ORCHESTRATOR_URL          - Orchestrator URL (default: http://localhost:30010)
//   CONSOLIDATION_DELAY_MS    - Delay between operations in ms (default: 400)
//   MAX_CONSOLIDATION_TASKS   - Max tasks to process per VU iteration (default: 10)
//   CONSOLIDATION_STATION     - Default consolidation station (default: CONSOL-STATION-1)

import { sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
import {
  discoverPendingConsolidations,
  processAllPendingConsolidations,
} from '../lib/consolidation.js';

// Custom metrics
const consolidationsDiscovered = new Counter('consolidation_tasks_discovered');
const consolidationsProcessed = new Counter('consolidation_tasks_processed');
const consolidationsFailed = new Counter('consolidation_tasks_failed');
const consolidationSuccessRate = new Rate('consolidation_success_rate');
const consolidationProcessingTime = new Trend('consolidation_processing_time');

// Default options - can be overridden via CLI
export const options = {
  scenarios: {
    // Continuous consolidation simulation
    consolidation_simulation: {
      executor: 'constant-vus',
      vus: 1,
      duration: '2m',
      gracefulStop: '30s',
    },
  },
  thresholds: {
    'consolidation_success_rate': ['rate>0.9'],  // 90% success rate
    'consolidation_processing_time': ['p(95)<30000'],  // 95th percentile under 30s
  },
};

// Setup function - runs once before the test
export function setup() {
  console.log('='.repeat(60));
  console.log('Consolidation Simulator Starting');
  console.log('='.repeat(60));

  // Initial discovery to check connectivity
  const consolidations = discoverPendingConsolidations();
  console.log(`Initial discovery found ${consolidations.length} pending consolidations`);

  return {
    startTime: new Date().toISOString(),
    initialConsolidationCount: consolidations.length,
  };
}

// Main test function - runs for each VU iteration
export default function () {
  const startTime = Date.now();

  // Discover and process pending consolidations
  const results = processAllPendingConsolidations();

  const processingTime = Date.now() - startTime;

  // Update metrics
  consolidationsDiscovered.add(results.discovered);
  consolidationsProcessed.add(results.processed);
  consolidationsFailed.add(results.failed);

  // Calculate success rate for this iteration
  if (results.discovered > 0) {
    for (const consolidation of results.consolidations) {
      consolidationSuccessRate.add(consolidation.success);
      consolidationProcessingTime.add(processingTime / results.consolidations.length);
    }
  }

  // Log iteration summary
  console.log(`[VU ${__VU}] Iteration complete: ${results.processed}/${results.discovered} consolidations processed`);

  // Sleep between iterations if no consolidations found
  if (results.discovered === 0) {
    console.log(`[VU ${__VU}] No pending consolidations found, waiting before retry...`);
    sleep(5);  // Wait 5 seconds before checking again
  } else {
    sleep(1);  // Brief pause between iterations
  }
}

// Teardown function - runs once after all VUs complete
export function teardown(data) {
  console.log('='.repeat(60));
  console.log('Consolidation Simulator Complete');
  console.log(`Started: ${data.startTime}`);
  console.log(`Initial pending consolidations: ${data.initialConsolidationCount}`);
  console.log('='.repeat(60));
}
