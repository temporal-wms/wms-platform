// K6 Picking Service Helper Library
// Provides functions for interacting with picking-service and orchestrator signal bridge

import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URLS, ENDPOINTS, HTTP_PARAMS, PICKER_CONFIG } from './config.js';

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
  const url = `${BASE_URLS.orchestrator}${ENDPOINTS.orchestrator.signalPickCompleted}`;
  const payload = JSON.stringify({
    orderId: orderId,
    taskId: taskId,
    pickedItems: pickedItems,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'signal pick completed status 200': (r) => r.status === 200,
  });

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

    // Use the exact locationId from the task item (may be empty)
    const locationId = item.location?.locationId || item.locationId || '';

    // Confirm the pick (with required toteId)
    const success = confirmPick(
      task.taskId,
      item.sku,
      item.quantity,
      locationId,
      toteId
    );

    if (success) {
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
