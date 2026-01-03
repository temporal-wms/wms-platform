// Walling Simulator - K6 Load Test Script
// Simulates walliner (put-wall worker) sessions to advance WES workflows
//
// Usage:
//   k6 run scripts/scenarios/walling-simulator.js
//   k6 run --vus 5 --duration 5m scripts/scenarios/walling-simulator.js
//
// Environment variables:
//   WALLING_SERVICE_URL  - Walling service URL (default: http://localhost:8017)
//   WES_SERVICE_URL      - WES service URL (default: http://localhost:8016)
//   WALLING_DELAY_MS     - Delay between item sorts in ms (default: 500)
//   MAX_WALLING_TASKS    - Max tasks to process per VU iteration (default: 10)
//   DEFAULT_PUT_WALL     - Default put wall ID (default: PUTWALL-1)

import { sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
import {
  discoverPendingWallingTasks,
  processWallingTask,
  processAllPendingWallingTasks,
} from '../lib/walling.js';
import { WALLING_CONFIG } from '../lib/config.js';

// Custom metrics
const tasksDiscovered = new Counter('walling_tasks_discovered');
const tasksProcessed = new Counter('walling_tasks_processed');
const tasksFailed = new Counter('walling_tasks_failed');
const itemsSorted = new Counter('walling_items_sorted');
const taskSuccessRate = new Rate('walling_task_success_rate');
const taskProcessingTime = new Trend('walling_task_processing_time');

// Default options - can be overridden via CLI
export const options = {
  scenarios: {
    // Continuous walling simulation
    walling_simulation: {
      executor: 'constant-vus',
      vus: 1,
      duration: '2m',
      gracefulStop: '30s',
    },
  },
  thresholds: {
    'walling_task_success_rate': ['rate>0.9'],  // 90% success rate
    'walling_task_processing_time': ['p(95)<30000'],  // 95th percentile under 30s
  },
};

// Setup function - runs once before the test
export function setup() {
  console.log('='.repeat(60));
  console.log('Walling Simulator Starting');
  console.log(`Put Wall ID: ${WALLING_CONFIG.defaultPutWallId}`);
  console.log(`Station: ${WALLING_CONFIG.defaultStation}`);
  console.log('='.repeat(60));

  // Initial discovery to check connectivity
  const tasks = discoverPendingWallingTasks(WALLING_CONFIG.defaultPutWallId);
  console.log(`Initial discovery found ${tasks.length} pending walling tasks`);

  return {
    startTime: new Date().toISOString(),
    initialTaskCount: tasks.length,
    putWallId: WALLING_CONFIG.defaultPutWallId,
  };
}

// Main test function - runs for each VU iteration
export default function (data) {
  const startTime = Date.now();
  const putWallId = data?.putWallId || WALLING_CONFIG.defaultPutWallId;

  // Discover and process pending walling tasks
  const results = processAllPendingWallingTasks(putWallId);

  const processingTime = Date.now() - startTime;

  // Update metrics
  tasksDiscovered.add(results.discovered);
  tasksProcessed.add(results.processed);
  tasksFailed.add(results.failed);

  // Calculate success rate for this iteration
  if (results.discovered > 0) {
    for (const task of results.tasks) {
      taskSuccessRate.add(task.success);
      taskProcessingTime.add(processingTime / results.tasks.length);
    }
  }

  // Log iteration summary
  console.log(`[VU ${__VU}] Walling iteration complete: ${results.processed}/${results.discovered} tasks processed`);

  // Sleep between iterations if no tasks found
  if (results.discovered === 0) {
    console.log(`[VU ${__VU}] No walling tasks found, waiting before retry...`);
    sleep(5);  // Wait 5 seconds before checking again
  } else {
    sleep(1);  // Brief pause between iterations
  }
}

// Teardown function - runs once after all VUs complete
export function teardown(data) {
  console.log('='.repeat(60));
  console.log('Walling Simulator Complete');
  console.log(`Started: ${data.startTime}`);
  console.log(`Initial pending tasks: ${data.initialTaskCount}`);
  console.log(`Put Wall ID: ${data.putWallId}`);
  console.log('='.repeat(60));
}
