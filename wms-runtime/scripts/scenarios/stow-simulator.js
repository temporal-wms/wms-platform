// Stow Simulator
// Simulates putaway/stowage operations for WMS end-to-end testing

import { check, sleep, group } from 'k6';
import { Counter, Trend, Rate } from 'k6/metrics';
import {
  STOW_CONFIG,
  STOW_TASK_STATUS,
  LOCATION_TYPES,
  discoverStowTasks,
  createStowTask,
  assignStowTask,
  startStowTask,
  selectStowLocation,
  confirmStow,
  completeStowTask,
  updateInventoryAfterStow,
  signalStowCompleted,
  processAllPendingStowTasks,
  getStowTasksByShipment,
} from '../lib/stow.js';
import { products, zones } from '../lib/data.js';

// Custom metrics
const tasksDiscovered = new Counter('stow_tasks_discovered');
const tasksCompleted = new Counter('stow_tasks_completed');
const tasksFailed = new Counter('stow_tasks_failed');
const itemsStowed = new Counter('stow_items_stowed');
const stowDuration = new Trend('stow_duration_ms');
const stowSuccessRate = new Rate('stow_success_rate');
const locationSelectionTime = new Trend('stow_location_selection_ms');

// Test configuration
export const options = {
  scenarios: {
    stow_flow: {
      executor: 'ramping-vus',
      startVUs: 1,
      stages: [
        { duration: '30s', target: 3 },   // Ramp up
        { duration: '2m', target: 5 },    // Steady state
        { duration: '30s', target: 0 },   // Ramp down
      ],
      gracefulRampDown: '10s',
    },
  },
  thresholds: {
    'stow_success_rate': ['rate>0.95'],
    'stow_duration_ms': ['p(95)<3000'],
    'stow_location_selection_ms': ['p(95)<500'],
    'http_req_failed': ['rate<0.05'],
  },
};

// Configuration
const CONFIG = {
  maxTasksPerIteration: parseInt(__ENV.MAX_STOW_TASKS || '10'),
  createTestTasks: __ENV.CREATE_TEST_TASKS === 'true',
  testTaskCount: parseInt(__ENV.TEST_TASK_COUNT || '5'),
};

/**
 * Creates test stow tasks for simulation
 */
function createTestStowTasks(count) {
  const createdTasks = [];

  for (let i = 0; i < count; i++) {
    const product = products[Math.floor(Math.random() * products.length)];
    const quantity = Math.floor(Math.random() * 20) + 5;

    const taskData = {
      licensePlate: `LP-TEST-${Date.now()}-${i}`,
      sku: product.sku,
      quantity: quantity,
      sourceLocation: 'RECEIVING-DOCK',
      targetZone: STOW_CONFIG.defaultZone,
      priority: 'normal',
      shipmentId: `SHIP-TEST-${Date.now()}`,
    };

    const task = createStowTask(taskData);
    if (task) {
      createdTasks.push(task);
    }
  }

  console.log(`Created ${createdTasks.length} test stow tasks`);
  return createdTasks;
}

/**
 * Processes a single stow task with full simulation
 */
function processStowTaskFull(task) {
  const taskId = task.taskId || task.id;
  const startTime = Date.now();

  console.log(`Processing stow task: ${taskId} for SKU ${task.sku}`);

  // Assign task to worker
  if (!assignStowTask(taskId)) {
    console.warn(`Failed to assign stow task ${taskId}`);
    return { success: false, error: 'assignment_failed' };
  }

  // Start the task
  if (!startStowTask(taskId)) {
    console.warn(`Failed to start stow task ${taskId}`);
    return { success: false, error: 'start_failed' };
  }

  // Select optimal location
  const locationStartTime = Date.now();
  const location = selectStowLocation(task.sku, task.quantity, task.targetZone);
  locationSelectionTime.add(Date.now() - locationStartTime);

  if (!location) {
    console.warn(`No suitable location found for ${task.sku}`);
    return { success: false, error: 'no_location' };
  }

  console.log(`Selected location: ${location.locationId} for ${task.sku}`);

  // Simulate physical stow operation
  sleep(STOW_CONFIG.simulationDelayMs / 1000);

  // Confirm the stow
  if (!confirmStow(taskId, location.locationId, task.quantity)) {
    console.warn(`Failed to confirm stow for task ${taskId}`);
    return { success: false, error: 'confirm_failed' };
  }

  // Update inventory
  const inventoryUpdated = updateInventoryAfterStow(
    location.locationId,
    task.sku,
    task.quantity
  );

  // Complete the task
  const completed = completeStowTask(taskId, {
    locationId: location.locationId,
    quantity: task.quantity,
  });

  const duration = Date.now() - startTime;
  stowDuration.add(duration);

  if (completed) {
    itemsStowed.add(task.quantity);
    return {
      success: true,
      taskId: taskId,
      sku: task.sku,
      quantity: task.quantity,
      locationId: location.locationId,
      duration: duration,
      inventoryUpdated: inventoryUpdated,
    };
  }

  return { success: false, error: 'completion_failed' };
}

/**
 * Main test function
 */
export default function () {
  const vuId = __VU;
  const iterationId = __ITER;

  console.log(`[VU ${vuId}] Starting stow simulation - iteration ${iterationId}`);

  // Phase 1: Create test tasks if configured
  if (CONFIG.createTestTasks && iterationId === 0) {
    group('Create Test Stow Tasks', function () {
      createTestStowTasks(CONFIG.testTaskCount);
      sleep(1);
    });
  }

  // Phase 2: Discover and process pending stow tasks
  group('Process Pending Stow Tasks', function () {
    const pendingTasks = discoverStowTasks(STOW_TASK_STATUS.PENDING);
    tasksDiscovered.add(pendingTasks.length);

    console.log(`[VU ${vuId}] Found ${pendingTasks.length} pending stow tasks`);

    if (pendingTasks.length === 0) {
      console.log(`[VU ${vuId}] No pending tasks, checking for assigned tasks...`);
      const assignedTasks = discoverStowTasks(STOW_TASK_STATUS.ASSIGNED);
      console.log(`[VU ${vuId}] Found ${assignedTasks.length} assigned tasks`);
    }

    // Process tasks up to the limit
    const tasksToProcess = pendingTasks.slice(0, CONFIG.maxTasksPerIteration);
    const results = {
      processed: 0,
      failed: 0,
      stowedItems: [],
    };

    for (const task of tasksToProcess) {
      const result = processStowTaskFull(task);

      if (result.success) {
        tasksCompleted.add(1);
        stowSuccessRate.add(1);
        results.processed++;
        results.stowedItems.push({
          taskId: result.taskId,
          sku: result.sku,
          quantity: result.quantity,
          locationId: result.locationId,
        });
        console.log(`[VU ${vuId}] Completed stow task ${result.taskId} in ${result.duration}ms`);
      } else {
        tasksFailed.add(1);
        stowSuccessRate.add(0);
        results.failed++;
        console.warn(`[VU ${vuId}] Failed stow task: ${result.error}`);
      }

      // Small delay between tasks
      sleep(0.5);
    }

    // If we stowed items from a shipment, signal completion
    if (results.stowedItems.length > 0) {
      // Group by shipment ID if available
      const shipmentId = tasksToProcess[0]?.shipmentId;
      if (shipmentId) {
        signalStowCompleted(shipmentId, results.stowedItems);
        console.log(`[VU ${vuId}] Signaled stow completion for shipment ${shipmentId}`);
      }
    }

    console.log(`[VU ${vuId}] Stow results: ${results.processed} completed, ${results.failed} failed`);
  });

  // Brief pause between iterations
  sleep(2);
}

/**
 * Setup function - runs once before test
 */
export function setup() {
  console.log('='.repeat(60));
  console.log('Stow Simulator - Setup');
  console.log('='.repeat(60));
  console.log(`Max tasks per iteration: ${CONFIG.maxTasksPerIteration}`);
  console.log(`Create test tasks: ${CONFIG.createTestTasks}`);
  console.log(`Test task count: ${CONFIG.testTaskCount}`);
  console.log('='.repeat(60));

  return {
    startTime: Date.now(),
  };
}

/**
 * Teardown function - runs once after test
 */
export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;

  console.log('='.repeat(60));
  console.log('Stow Simulator - Summary');
  console.log('='.repeat(60));
  console.log(`Total duration: ${duration.toFixed(2)}s`);
  console.log('='.repeat(60));
}

/**
 * Custom summary handler
 */
export function handleSummary(data) {
  const summary = {
    timestamp: new Date().toISOString(),
    simulator: 'stow-simulator',
    metrics: {
      tasks_discovered: data.metrics.stow_tasks_discovered?.values?.count || 0,
      tasks_completed: data.metrics.stow_tasks_completed?.values?.count || 0,
      tasks_failed: data.metrics.stow_tasks_failed?.values?.count || 0,
      items_stowed: data.metrics.stow_items_stowed?.values?.count || 0,
      success_rate: data.metrics.stow_success_rate?.values?.rate || 0,
      avg_duration_ms: data.metrics.stow_duration_ms?.values?.avg || 0,
      p95_duration_ms: data.metrics.stow_duration_ms?.values?.['p(95)'] || 0,
      avg_location_selection_ms: data.metrics.stow_location_selection_ms?.values?.avg || 0,
    },
    thresholds: data.thresholds,
  };

  return {
    'stdout': JSON.stringify(summary, null, 2) + '\n',
    'stow-results.json': JSON.stringify(summary, null, 2),
  };
}
