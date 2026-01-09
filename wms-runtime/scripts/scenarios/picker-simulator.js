// Picker Simulator - K6 Load Test Script
// Simulates picker work sessions to advance Temporal workflows
//
// Usage:
//   k6 run scripts/scenarios/picker-simulator.js
//   k6 run --vus 5 --duration 5m scripts/scenarios/picker-simulator.js
//
// Environment variables:
//   PICKING_SERVICE_URL  - Picking service URL (default: http://localhost:8004)
//   ORCHESTRATOR_URL     - Orchestrator URL (default: http://localhost:8080)
//   PICKER_DELAY_MS      - Delay between picks in ms (default: 500)
//   MAX_TASKS_PER_ITERATION - Max tasks to process per VU iteration (default: 10)

import { sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
import {
  discoverPendingTasks,
  processPickTask,
  processAllPendingTasks,
} from '../lib/picking.js';

// Custom metrics
const tasksDiscovered = new Counter('picker_tasks_discovered');
const tasksProcessed = new Counter('picker_tasks_processed');
const tasksFailed = new Counter('picker_tasks_failed');
const taskSuccessRate = new Rate('picker_task_success_rate');
const taskProcessingTime = new Trend('picker_task_processing_time');

// Default options - can be overridden via CLI
export const options = {
  scenarios: {
    // Continuous picker simulation
    picker_simulation: {
      executor: 'constant-vus',
      vus: 1,
      duration: '2m',
      gracefulStop: '30s',
    },
  },
  thresholds: {
    'picker_task_success_rate': ['rate>0.9'],  // 90% success rate
    'picker_task_processing_time': ['p(95)<30000'],  // 95th percentile under 30s
  },
};

// Setup function - runs once before the test
export function setup() {
  console.log('='.repeat(60));
  console.log('Picker Simulator Starting');
  console.log('='.repeat(60));

  // Initial discovery to check connectivity
  const tasks = discoverPendingTasks('assigned');
  console.log(`Initial discovery found ${tasks.length} pending tasks`);

  return {
    startTime: new Date().toISOString(),
    initialTaskCount: tasks.length,
  };
}

// Main test function - runs for each VU iteration
export default function () {
  const startTime = Date.now();

  // Discover and process pending tasks
  const results = processAllPendingTasks();

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
  console.log(`[VU ${__VU}] Iteration complete: ${results.processed}/${results.discovered} tasks processed`);

  // Sleep between iterations if no tasks found
  if (results.discovered === 0) {
    console.log(`[VU ${__VU}] No tasks found, waiting before retry...`);
    sleep(5);  // Wait 5 seconds before checking again
  } else {
    sleep(1);  // Brief pause between iterations
  }
}

// Teardown function - runs once after all VUs complete
export function teardown(data) {
  console.log('='.repeat(60));
  console.log('Picker Simulator Complete');
  console.log(`Started: ${data.startTime}`);
  console.log(`Initial pending tasks: ${data.initialTaskCount}`);
  console.log('='.repeat(60));
}
