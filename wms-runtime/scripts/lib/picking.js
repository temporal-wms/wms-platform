// K6 Picking Service Helper Library
// Provides functions for interacting with picking-service and orchestrator signal bridge

import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URLS, ENDPOINTS, HTTP_PARAMS, PICKER_CONFIG, SIGNAL_CONFIG } from './config.js';
import { pickStock, getInventoryItem, reserveStock } from './inventory.js';
import { confirmPicksForOrder } from './unit.js';

/**
 * Discovers pending pick tasks from the picking service
 * @param {string} status - Task status to filter by (default: 'assigned')
 * @returns {Array} Array of pick tasks
 */
export function discoverPendingTasks(status = 'assigned') {
  const url = `${BASE_URLS.picking}/api/v1/tasks?status=${status}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'discover tasks status 200': (r) => r.status === 200,
  });

  if (!success || !response.body) {
    console.warn(`Failed to discover tasks: ${response.status} - ${response.body}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    // Handle both array response and paginated response
    return Array.isArray(data) ? data : (data.tasks || data.items || []);
  } catch (e) {
    console.error(`Failed to parse tasks response: ${e.message}`);
    return [];
  }
}

/**
 * Gets a specific pick task by ID
 * @param {string} taskId - The task ID
 * @returns {Object|null} The pick task or null if not found
 */
export function getPickTask(taskId) {
  const url = `${BASE_URLS.picking}${ENDPOINTS.picking.get(taskId)}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get task status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to get task ${taskId}: ${response.status}`);
    return null;
  }

  try {
    return JSON.parse(response.body);
  } catch (e) {
    console.error(`Failed to parse task response: ${e.message}`);
    return null;
  }
}

/**
 * Confirms a pick for an item in a task
 * @param {string} taskId - The task ID
 * @param {string} sku - The SKU being picked
 * @param {number} quantity - The quantity picked
 * @param {string} locationId - The location ID where item was picked
 * @param {string} toteId - The tote ID for the pick
 * @returns {boolean} True if successful
 */
export function confirmPick(taskId, sku, quantity, locationId, toteId) {
  const url = `${BASE_URLS.picking}${ENDPOINTS.picking.confirmPick(taskId)}`;
  const payload = JSON.stringify({
    sku: sku,
    pickedQty: quantity,
    locationId: locationId,
    toteId: toteId,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'confirm pick status 200': (r) => r.status === 200 || r.status === 204,
  });

  if (!success) {
    console.warn(`Failed to confirm pick for task ${taskId}, sku ${sku}: ${response.status} - ${response.body}`);
  }

  return success;
}

/**
 * Starts a pick task (must be called before confirming picks)
 * @param {string} taskId - The task ID
 * @returns {boolean} True if successful
 */
export function startTask(taskId) {
  const url = `${BASE_URLS.picking}${ENDPOINTS.picking.start(taskId)}`;
  const response = http.post(url, null, HTTP_PARAMS);

  const success = check(response, {
    'start task status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to start task ${taskId}: ${response.status} - ${response.body}`);
  }

  return success;
}

/**
 * Completes a pick task
 * @param {string} taskId - The task ID
 * @returns {Object|null} The completed task or null if failed
 */
export function completeTask(taskId) {
  const url = `${BASE_URLS.picking}${ENDPOINTS.picking.complete(taskId)}`;
  const response = http.post(url, null, HTTP_PARAMS);

  const success = check(response, {
    'complete task status 200': (r) => r.status === 200,
  });

  // Handle "task is already completed" as success (idempotency)
  if (!success && response.status === 400 && response.body) {
    try {
      const errorBody = JSON.parse(response.body);
      if (errorBody.message && errorBody.message.includes('already completed')) {
        console.log(`Task ${taskId} already completed (idempotent success)`);
        return { taskId: taskId, status: 'completed' };
      }
    } catch (e) {
      // Continue to regular error handling
    }
  }

  if (!success) {
    console.warn(`Failed to complete task ${taskId}: ${response.status} - ${response.body}`);
    return null;
  }

  try {
    return JSON.parse(response.body);
  } catch (e) {
    console.error(`Failed to parse complete task response: ${e.message}`);
    return null;
  }
}

/**
 * Sends a pick completed signal to the orchestrator to advance the Temporal workflow
 * @param {string} orderId - The order ID
 * @param {string} taskId - The task ID
 * @param {Array} pickedItems - Array of picked items with {sku, quantity, locationId, toteId}
 * @returns {boolean} True if successful
 */
export function sendPickCompletedSignal(orderId, taskId, pickedItems) {
  // Validate pickedItems is not empty
  if (!pickedItems || pickedItems.length === 0) {
    console.warn(`⚠️  No picked items for order ${orderId}, skipping signal`);
    return false;
  }

  // Validate each picked item has required fields
  const invalidItems = pickedItems.filter(item =>
    !item.sku || !item.quantity || !item.locationId || !item.toteId
  );

  if (invalidItems.length > 0) {
    console.error(`❌ Invalid picked items for order ${orderId}:`, invalidItems);
    return false;
  }

  console.log(`✓ Sending pick completed signal for order ${orderId} with ${pickedItems.length} items`);

  const url = `${BASE_URLS.orchestrator}${ENDPOINTS.orchestrator.signalPickCompleted}`;
  const payload = JSON.stringify({
    orderId: orderId,
    taskId: taskId,
    pickedItems: pickedItems,
  });

  let success = false;
  let lastResponse = null;

  for (let attempt = 1; attempt <= SIGNAL_CONFIG.maxRetries; attempt++) {
    const response = http.post(url, payload, {
      ...HTTP_PARAMS,
      timeout: `${SIGNAL_CONFIG.timeoutMs}ms`,
    });
    lastResponse = response;

    success = check(response, {
      'signal pick completed status 200': (r) => r.status === 200,
    });

    if (success) {
      if (attempt > 1) {
        console.log(`✓ Signal succeeded on attempt ${attempt}/${SIGNAL_CONFIG.maxRetries}`);
      }
      break;
    }

    if (attempt < SIGNAL_CONFIG.maxRetries) {
      console.warn(`⚠️  Signal attempt ${attempt}/${SIGNAL_CONFIG.maxRetries} failed: ${response.status}, retrying...`);
      sleep(SIGNAL_CONFIG.retryDelayMs / 1000);
    }
  }

  const response = lastResponse; // For compatibility with code below

  if (!success) {
    console.warn(`Failed to signal pick completed for order ${orderId}: ${response.status} - ${response.body}`);
  } else {
    try {
      const result = JSON.parse(response.body);
      console.log(`Signal sent successfully for workflow: ${result.workflowId}`);
    } catch (e) {
      // Ignore parse errors for logging
    }
  }

  return success;
}

/**
 * Simulates picking all items in a task with realistic delays
 * @param {Object} task - The pick task object
 * @returns {Array} Array of picked items for signaling
 */
export function simulatePickingTask(task) {
  const pickedItems = [];
  const items = task.items || [];
  const toteId = task.toteId || `TOTE-SIM-${Date.now()}`;

  console.log(`Simulating picking for task ${task.taskId} with ${items.length} items`);

  // Start the task first (required before confirming picks)
  if (!startTask(task.taskId)) {
    console.warn(`Failed to start task ${task.taskId}, trying to continue anyway`);
  }

  for (const item of items) {
    // Simulate picking delay
    sleep(PICKER_CONFIG.simulationDelayMs / 1000);

    // Get locationId from task item, or look up from inventory if empty
    let locationId = item.location?.locationId || item.locationId || '';

    if (!locationId) {
      // Look up location from inventory service
      const invResult = getInventoryItem(item.sku);
      if (invResult.success && invResult.body?.locations?.length > 0) {
        // Use first available location with stock
        const availableLoc = invResult.body.locations.find(loc => loc.available > 0)
          || invResult.body.locations[0];
        locationId = availableLoc.locationId;
      }
    }

    // Confirm the pick (with required toteId)
    const success = confirmPick(
      task.taskId,
      item.sku,
      item.quantity,
      locationId,
      toteId
    );

    if (success) {
      // Reserve inventory (required for staging in orchestrator workflow)
      // Note: We only create reservations here, orchestrator's StageInventory will convert to hard allocations
      const reserveResult = reserveStock(item.sku, task.orderId, locationId, item.quantity);
      if (!reserveResult.success) {
        // 400 is expected when stock is insufficient or SKU doesn't exist at location
        // Only warn on unexpected errors (500, network issues, etc.)
        if (reserveResult.status !== 400) {
          console.warn(`Failed to reserve inventory for ${item.sku}: ${reserveResult.status} (continuing anyway)`);
        }
      } else {
        console.log(`Inventory reserved: ${item.sku} x${item.quantity} at ${locationId} for staging`);
      }

      pickedItems.push({
        sku: item.sku,
        quantity: item.quantity,
        locationId: locationId || 'LOC-DEFAULT',
        toteId: toteId,
      });
    }
  }

  return pickedItems;
}

/**
 * Processes a single pick task end-to-end: pick items, complete task, signal workflow
 * @param {Object} task - The pick task object
 * @returns {boolean} True if fully successful
 */
export function processPickTask(task) {
  console.log(`Processing pick task ${task.taskId} for order ${task.orderId}`);

  // Step 1: Simulate picking all items
  const pickedItems = simulatePickingTask(task);

  if (pickedItems.length === 0) {
    console.warn(`No items picked for task ${task.taskId}`);
    return false;
  }

  // Step 1b: Confirm unit picks for the order
  if (task.orderId && pickedItems.length > 0) {
    const toteId = pickedItems[0]?.toteId || `TOTE-${task.taskId}`;
    const pickerId = `PICKER-SIM-${__VU || 1}`;
    const unitResult = confirmPicksForOrder(task.orderId, toteId, pickerId, '');
    if (!unitResult.skipped) {
      console.log(`Unit pick confirmations: ${unitResult.success}/${unitResult.total} succeeded`);
    }
  }

  // Step 2: Complete the task in picking-service
  const completedTask = completeTask(task.taskId);
  if (!completedTask) {
    console.warn(`Failed to complete task ${task.taskId}`);
    // Still try to signal workflow even if complete fails
  }

  // Step 3: Signal the workflow to advance
  const signalSuccess = sendPickCompletedSignal(task.orderId, task.taskId, pickedItems);

  return signalSuccess;
}

/**
 * Discovers and processes all pending tasks
 * @param {number} maxTasks - Maximum number of tasks to process (default from config)
 * @returns {Object} Summary of processing results
 */
export function processAllPendingTasks(maxTasks = PICKER_CONFIG.maxTasksPerIteration) {
  const results = {
    discovered: 0,
    processed: 0,
    failed: 0,
    tasks: [],
  };

  // Discover pending tasks
  const tasks = discoverPendingTasks('assigned');
  results.discovered = tasks.length;

  console.log(`Discovered ${tasks.length} pending tasks`);

  // Process up to maxTasks
  const tasksToProcess = tasks.slice(0, maxTasks);

  for (const task of tasksToProcess) {
    const success = processPickTask(task);

    results.tasks.push({
      taskId: task.taskId,
      orderId: task.orderId,
      success: success,
    });

    if (success) {
      results.processed++;
    } else {
      results.failed++;
    }
  }

  console.log(`Processed ${results.processed}/${results.discovered} tasks (${results.failed} failed)`);

  return results;
}
