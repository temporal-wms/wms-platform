// K6 Walling Service Helper Library
// Provides functions for interacting with walling-service and WES signal bridge

import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URLS, ENDPOINTS, HTTP_PARAMS, WALLING_CONFIG } from './config.js';

/**
 * Discovers pending walling tasks from the walling service
 * @param {string} putWallId - The put wall ID to filter tasks (required)
 * @param {number} limit - Maximum number of tasks to return
 * @returns {Array} Array of walling tasks
 */
export function discoverPendingWallingTasks(putWallId = WALLING_CONFIG.defaultPutWallId, limit = WALLING_CONFIG.maxTasksPerIteration) {
  const url = `${BASE_URLS.walling}${ENDPOINTS.walling.pending}?putWallId=${putWallId}&limit=${limit}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'discover walling tasks status 200': (r) => r.status === 200,
  });

  if (!success || !response.body) {
    console.warn(`Failed to discover walling tasks: ${response.status} - ${response.body}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return Array.isArray(data) ? data : (data.tasks || data.items || []);
  } catch (e) {
    console.error(`Failed to parse walling tasks response: ${e.message}`);
    return [];
  }
}

/**
 * Gets a specific walling task by ID
 * @param {string} taskId - The task ID
 * @returns {Object|null} The walling task or null if not found
 */
export function getWallingTask(taskId) {
  const url = `${BASE_URLS.walling}${ENDPOINTS.walling.get(taskId)}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get walling task status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to get walling task ${taskId}: ${response.status}`);
    return null;
  }

  try {
    return JSON.parse(response.body);
  } catch (e) {
    console.error(`Failed to parse walling task response: ${e.message}`);
    return null;
  }
}

/**
 * Assigns a walliner (worker) to a walling task
 * @param {string} taskId - The task ID
 * @param {string} wallinerId - The walliner (worker) ID
 * @param {string} station - The walling station
 * @returns {Object|null} The updated task or null if failed
 */
export function assignWalliner(taskId, wallinerId, station = WALLING_CONFIG.defaultStation) {
  const url = `${BASE_URLS.walling}${ENDPOINTS.walling.assign(taskId)}`;
  const payload = JSON.stringify({
    wallinerId: wallinerId,
    station: station,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'assign walliner status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to assign walliner to task ${taskId}: ${response.status} - ${response.body}`);
    return null;
  }

  try {
    return JSON.parse(response.body);
  } catch (e) {
    console.error(`Failed to parse assign response: ${e.message}`);
    return null;
  }
}

/**
 * Sorts an item from a tote into the put wall slot
 * @param {string} taskId - The task ID
 * @param {string} sku - The SKU being sorted
 * @param {number} quantity - The quantity to sort
 * @param {string} fromToteId - The source tote ID
 * @returns {Object|null} The updated task or null if failed
 */
export function sortItem(taskId, sku, quantity, fromToteId) {
  const url = `${BASE_URLS.walling}${ENDPOINTS.walling.sort(taskId)}`;
  const payload = JSON.stringify({
    sku: sku,
    quantity: quantity,
    fromToteId: fromToteId,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'sort item status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to sort item for task ${taskId}, sku ${sku}: ${response.status} - ${response.body}`);
    return null;
  }

  try {
    return JSON.parse(response.body);
  } catch (e) {
    console.error(`Failed to parse sort response: ${e.message}`);
    return null;
  }
}

/**
 * Completes a walling task
 * @param {string} taskId - The task ID
 * @returns {Object|null} The completed task or null if failed
 */
export function completeWallingTask(taskId) {
  const url = `${BASE_URLS.walling}${ENDPOINTS.walling.complete(taskId)}`;
  const response = http.post(url, null, HTTP_PARAMS);

  const success = check(response, {
    'complete walling task status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to complete walling task ${taskId}: ${response.status} - ${response.body}`);
    return null;
  }

  try {
    return JSON.parse(response.body);
  } catch (e) {
    console.error(`Failed to parse complete response: ${e.message}`);
    return null;
  }
}

/**
 * Sends a walling completed signal to the orchestrator to advance the WES workflow
 * @param {string} orderId - The order ID
 * @param {string} taskId - The walling task ID
 * @param {string} routeId - The WES route ID
 * @param {Array} sortedItems - Array of sorted items with {sku, quantity, slotId}
 * @returns {boolean} True if successful
 */
export function sendWallingCompletedSignal(orderId, taskId, routeId, sortedItems = []) {
  const url = `${BASE_URLS.orchestrator}${ENDPOINTS.orchestrator.signalWallingCompleted}`;
  const payload = JSON.stringify({
    orderId: orderId,
    taskId: taskId,
    routeId: routeId,
    sortedItems: sortedItems,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'signal walling completed status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to signal walling completed for order ${orderId}: ${response.status} - ${response.body}`);
  } else {
    try {
      const result = JSON.parse(response.body);
      console.log(`Walling signal sent successfully for workflow: ${result.workflowId || orderId}`);
    } catch (e) {
      // Ignore parse errors for logging
    }
  }

  return success;
}

/**
 * Simulates sorting all items in a walling task with realistic delays
 * @param {Object} task - The walling task object
 * @returns {Array} Array of sorted items for signaling
 */
export function simulateWallingTask(task) {
  const sortedItems = [];
  const items = task.items || task.expectedItems || [];
  const sourceToteId = task.sourceToteId || task.toteId || `TOTE-${task.taskId}`;

  console.log(`Simulating walling for task ${task.taskId} with ${items.length} items`);

  for (const item of items) {
    // Simulate sorting delay
    sleep(WALLING_CONFIG.simulationDelayMs / 1000);

    // Sort the item
    const result = sortItem(
      task.taskId,
      item.sku,
      item.quantity || item.expectedQuantity || 1,
      sourceToteId
    );

    if (result) {
      sortedItems.push({
        sku: item.sku,
        quantity: item.quantity || item.expectedQuantity || 1,
        slotId: task.slotId || task.putWallSlotId || 'SLOT-DEFAULT',
      });
    }
  }

  return sortedItems;
}

/**
 * Processes a single walling task end-to-end: assign, sort items, complete, signal workflow
 * @param {Object} task - The walling task object
 * @param {string} wallinerId - The walliner ID to assign
 * @returns {boolean} True if fully successful
 */
export function processWallingTask(task, wallinerId = `WALLINER-SIM-${__VU || 1}`) {
  console.log(`Processing walling task ${task.taskId} for order ${task.orderId}`);

  // Step 1: Assign walliner to the task
  const assignedTask = assignWalliner(task.taskId, wallinerId, WALLING_CONFIG.defaultStation);
  if (!assignedTask) {
    console.warn(`Failed to assign walliner to task ${task.taskId}, trying to continue anyway`);
  }

  // Step 2: Simulate sorting all items
  const sortedItems = simulateWallingTask(task);

  if (sortedItems.length === 0) {
    console.warn(`No items sorted for task ${task.taskId}`);
    // Still try to complete
  }

  // Step 3: Complete the task in walling-service
  const completedTask = completeWallingTask(task.taskId);
  if (!completedTask) {
    console.warn(`Failed to complete walling task ${task.taskId}`);
    // Still try to signal workflow
  }

  // Step 4: Signal the WES workflow to advance
  const routeId = task.routeId || task.wesRouteId || '';
  const signalSuccess = sendWallingCompletedSignal(task.orderId, task.taskId, routeId, sortedItems);

  return signalSuccess;
}

/**
 * Discovers and processes all pending walling tasks
 * @param {string} putWallId - The put wall ID to filter tasks
 * @param {number} maxTasks - Maximum number of tasks to process
 * @returns {Object} Summary of processing results
 */
export function processAllPendingWallingTasks(putWallId = WALLING_CONFIG.defaultPutWallId, maxTasks = WALLING_CONFIG.maxTasksPerIteration) {
  const results = {
    discovered: 0,
    processed: 0,
    failed: 0,
    tasks: [],
  };

  // Discover pending tasks
  const tasks = discoverPendingWallingTasks(putWallId, maxTasks);
  results.discovered = tasks.length;

  console.log(`Discovered ${tasks.length} pending walling tasks`);

  // Process tasks
  for (const task of tasks) {
    const success = processWallingTask(task);

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

  console.log(`Processed ${results.processed}/${results.discovered} walling tasks (${results.failed} failed)`);

  return results;
}
