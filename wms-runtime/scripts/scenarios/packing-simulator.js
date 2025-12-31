// Packing Simulator - K6 Load Test Script
// Simulates packing work to advance Temporal workflows
//
// Usage:
//   k6 run scripts/scenarios/packing-simulator.js
//   k6 run --vus 3 --duration 5m scripts/scenarios/packing-simulator.js
//
// Environment variables:
//   PACKING_SERVICE_URL  - Packing service URL (default: http://localhost:8006)
//   ORCHESTRATOR_URL     - Orchestrator URL (default: http://localhost:30010)
//   PACKING_DELAY_MS     - Delay between operations in ms (default: 600)
//   MAX_PACKING_TASKS    - Max tasks to process per VU iteration (default: 10)
//   PACKING_STATION      - Default packing station (default: PACK-STATION-1)

import { sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
import {
  discoverPendingPackTasks,
  processAllPendingPackTasks,
} from '../lib/packing.js';

// Custom metrics
const packTasksDiscovered = new Counter('packing_tasks_discovered');
const packTasksProcessed = new Counter('packing_tasks_processed');
const packTasksFailed = new Counter('packing_tasks_failed');
const packSuccessRate = new Rate('packing_success_rate');
const packProcessingTime = new Trend('packing_processing_time');

// Default options - can be overridden via CLI
export const options = {
  scenarios: {
    // Continuous packing simulation
    packing_simulation: {
      executor: 'constant-vus',
      vus: 1,
      duration: '2m',
      gracefulStop: '30s',
    },
  },
  thresholds: {
    'packing_success_rate': ['rate>0.9'],  // 90% success rate
    'packing_processing_time': ['p(95)<30000'],  // 95th percentile under 30s
  },
};

// Setup function - runs once before the test
export function setup() {
  console.log('='.repeat(60));
  console.log('Packing Simulator Starting');
  console.log('='.repeat(60));

  // Initial discovery to check connectivity
  const tasks = discoverPendingPackTasks();
  console.log(`Initial discovery found ${tasks.length} pending pack tasks`);

  return {
    startTime: new Date().toISOString(),
    initialTaskCount: tasks.length,
  };
}

// Main test function - runs for each VU iteration
export default function () {
  const startTime = Date.now();

  // Discover and process pending pack tasks
  const results = processAllPendingPackTasks();

  const processingTime = Date.now() - startTime;

  // Update metrics
  packTasksDiscovered.add(results.discovered);
  packTasksProcessed.add(results.processed);
  packTasksFailed.add(results.failed);

  // Calculate success rate for this iteration
  if (results.discovered > 0) {
    for (const task of results.tasks) {
      packSuccessRate.add(task.success);
      packProcessingTime.add(processingTime / results.tasks.length);
    }
  }

  // Log iteration summary
  console.log(`[VU ${__VU}] Iteration complete: ${results.processed}/${results.discovered} pack tasks processed`);

  // Sleep between iterations if no tasks found
  if (results.discovered === 0) {
    console.log(`[VU ${__VU}] No pending pack tasks found, waiting before retry...`);
    sleep(5);  // Wait 5 seconds before checking again
  } else {
    sleep(1);  // Brief pause between iterations
  }
}

// Teardown function - runs once after all VUs complete
export function teardown(data) {
  console.log('='.repeat(60));
  console.log('Packing Simulator Complete');
  console.log(`Started: ${data.startTime}`);
  console.log(`Initial pending pack tasks: ${data.initialTaskCount}`);
  console.log('='.repeat(60));
}
