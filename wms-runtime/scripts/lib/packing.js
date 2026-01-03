// K6 Packing Service Helper Library
// Provides functions for interacting with packing-service and orchestrator signal bridge

import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URLS, ENDPOINTS, HTTP_PARAMS, PACKING_CONFIG } from './config.js';
import { confirmPacksForOrder } from './unit.js';

/**
 * Discovers pending pack tasks
 * @returns {Array} Array of pending pack tasks
 */
export function discoverPendingPackTasks() {
  const url = `${BASE_URLS.packing}${ENDPOINTS.packing.pending}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'discover pending pack tasks status 200': (r) => r.status === 200,
  });

  if (!success || !response.body) {
    console.warn(`Failed to discover pending pack tasks: ${response.status} - ${response.body}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return Array.isArray(data) ? data : (data.tasks || data.items || []);
  } catch (e) {
    console.error(`Failed to parse pack tasks response: ${e.message}`);
    return [];
  }
}

/**
 * Gets pack tasks by order ID
 * @param {string} orderId - The order ID
 * @returns {Array} Array of pack tasks
 */
export function getPackTasksByOrder(orderId) {
  const url = `${BASE_URLS.packing}${ENDPOINTS.packing.byOrder(orderId)}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get pack tasks by order status 200': (r) => r.status === 200,
  });

  if (!success || !response.body) {
    console.warn(`Failed to get pack tasks for order ${orderId}: ${response.status}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return Array.isArray(data) ? data : (data.tasks || data.items || []);
  } catch (e) {
    console.error(`Failed to parse pack tasks response: ${e.message}`);
    return [];
  }
}

/**
 * Gets a specific pack task by ID
 * @param {string} taskId - The task ID
 * @returns {Object|null} The pack task or null if not found
 */
export function getPackTask(taskId) {
  const url = `${BASE_URLS.packing}${ENDPOINTS.packing.get(taskId)}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get pack task status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to get pack task ${taskId}: ${response.status}`);
    return null;
  }

  try {
    return JSON.parse(response.body);
  } catch (e) {
    console.error(`Failed to parse pack task response: ${e.message}`);
    return null;
  }
}

/**
 * Assigns a pack task to a packer and station
 * @param {string} taskId - The task ID
 * @param {string} packerId - The packer worker ID
 * @param {string} station - The packing station
 * @returns {boolean} True if successful
 */
export function assignPackTask(taskId, packerId, station) {
  const url = `${BASE_URLS.packing}${ENDPOINTS.packing.assign(taskId)}`;
  const payload = JSON.stringify({
    packerId: packerId,
    station: station,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'assign pack task status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to assign pack task ${taskId}: ${response.status} - ${response.body}`);
  }

  return success;
}

/**
 * Starts a pack task
 * @param {string} taskId - The task ID
 * @returns {boolean} True if successful
 */
export function startPackTask(taskId) {
  const url = `${BASE_URLS.packing}${ENDPOINTS.packing.start(taskId)}`;
  const response = http.post(url, null, HTTP_PARAMS);

  const success = check(response, {
    'start pack task status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to start pack task ${taskId}: ${response.status} - ${response.body}`);
  }

  return success;
}

/**
 * Verifies an item in the pack task
 * @param {string} taskId - The task ID
 * @param {string} sku - The SKU to verify
 * @param {number} quantity - The quantity verified (optional)
 * @returns {boolean} True if successful
 */
export function verifyItem(taskId, sku, quantity = null) {
  const url = `${BASE_URLS.packing}${ENDPOINTS.packing.verify(taskId)}`;
  const payload = JSON.stringify({
    sku: sku,
    quantity: quantity,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'verify item status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to verify item ${sku} in task ${taskId}: ${response.status} - ${response.body}`);
  }

  return success;
}

/**
 * Selects packaging for the task
 * @param {string} taskId - The task ID
 * @param {string} packagingType - Type of packaging (box, envelope, etc.)
 * @param {Object} dimensions - Package dimensions {length, width, height, weight}
 * @returns {boolean} True if successful
 */
export function selectPackaging(taskId, packagingType, dimensions = {}) {
  const url = `${BASE_URLS.packing}${ENDPOINTS.packing.package(taskId)}`;
  const payload = JSON.stringify({
    packagingType: packagingType,
    ...dimensions,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'select packaging status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to select packaging for task ${taskId}: ${response.status} - ${response.body}`);
  }

  return success;
}

/**
 * Seals the package
 * @param {string} taskId - The task ID
 * @returns {boolean} True if successful
 */
export function sealPackage(taskId) {
  const url = `${BASE_URLS.packing}${ENDPOINTS.packing.seal(taskId)}`;
  const response = http.post(url, null, HTTP_PARAMS);

  const success = check(response, {
    'seal package status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to seal package for task ${taskId}: ${response.status} - ${response.body}`);
  }

  return success;
}

/**
 * Applies shipping label to the package
 * @param {string} taskId - The task ID
 * @param {string} labelData - Label data or tracking number
 * @returns {boolean} True if successful
 */
export function applyLabel(taskId, labelData = null) {
  const url = `${BASE_URLS.packing}${ENDPOINTS.packing.label(taskId)}`;
  const payload = JSON.stringify({
    labelData: labelData,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'apply label status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to apply label for task ${taskId}: ${response.status} - ${response.body}`);
  }

  return success;
}

/**
 * Completes a pack task
 * @param {string} taskId - The task ID
 * @returns {Object|null} The completed task or null if failed
 */
export function completePackTask(taskId) {
  const url = `${BASE_URLS.packing}${ENDPOINTS.packing.complete(taskId)}`;
  const response = http.post(url, null, HTTP_PARAMS);

  const success = check(response, {
    'complete pack task status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to complete pack task ${taskId}: ${response.status} - ${response.body}`);
    return null;
  }

  try {
    return JSON.parse(response.body);
  } catch (e) {
    console.error(`Failed to parse complete pack task response: ${e.message}`);
    return null;
  }
}

/**
 * Sends a packing completed signal to the orchestrator
 * @param {string} orderId - The order ID
 * @param {string} taskId - The task ID
 * @param {Object} packageInfo - Package information {trackingNumber, weight, dimensions}
 * @returns {boolean} True if successful
 */
export function sendPackingCompleteSignal(orderId, taskId, packageInfo = {}) {
  const url = `${BASE_URLS.orchestrator}${ENDPOINTS.orchestrator.signalPackingComplete}`;
  const payload = JSON.stringify({
    orderId: orderId,
    taskId: taskId,
    packageInfo: packageInfo,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'signal packing complete status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to signal packing complete for order ${orderId}: ${response.status} - ${response.body}`);
  } else {
    try {
      const result = JSON.parse(response.body);
      console.log(`Packing signal sent for workflow: ${result.workflowId}`);
    } catch (e) {
      // Ignore parse errors for logging
    }
  }

  return success;
}

/**
 * Simulates the full packing workflow for a task
 * @param {Object} task - The pack task object
 * @returns {Object} Package info for signaling
 */
export function simulatePackingTask(task) {
  const taskId = task.taskId || task.id;
  const items = task.items || [];
  const station = PACKING_CONFIG.defaultStation;
  const packerId = `PACKER-SIM-${__VU || 1}`;

  console.log(`Simulating packing task ${taskId} with ${items.length} items`);

  const packageInfo = {
    taskId: taskId,
    trackingNumber: null,
    weight: 0,
    dimensions: {},
    itemsVerified: 0,
  };

  // Step 1: Assign task
  sleep(PACKING_CONFIG.simulationDelayMs / 1000);
  if (!assignPackTask(taskId, packerId, station)) {
    console.warn(`Failed to assign pack task ${taskId}, trying to continue`);
  }

  // Step 2: Start task
  sleep(PACKING_CONFIG.simulationDelayMs / 2000);
  if (!startPackTask(taskId)) {
    console.warn(`Failed to start pack task ${taskId}, trying to continue`);
  }

  // Step 3: Verify each item
  for (const item of items) {
    sleep(PACKING_CONFIG.simulationDelayMs / 2000);
    if (verifyItem(taskId, item.sku, item.quantity)) {
      packageInfo.itemsVerified++;
      packageInfo.weight += (item.weight || 0.5) * (item.quantity || 1);
    }
  }

  // Step 4: Select packaging
  sleep(PACKING_CONFIG.simulationDelayMs / 2000);
  const dimensions = {
    length: 12,
    width: 8,
    height: 6,
    weight: packageInfo.weight,
  };
  selectPackaging(taskId, 'box', dimensions);
  packageInfo.dimensions = dimensions;

  // Step 5: Seal package
  sleep(PACKING_CONFIG.simulationDelayMs / 2000);
  sealPackage(taskId);

  // Step 6: Apply label
  sleep(PACKING_CONFIG.simulationDelayMs / 2000);
  packageInfo.trackingNumber = `TRK-${Date.now()}-${taskId.slice(-4)}`;
  applyLabel(taskId, packageInfo.trackingNumber);

  return packageInfo;
}

/**
 * Processes a single pack task end-to-end
 * @param {Object} task - The pack task object
 * @returns {boolean} True if fully successful
 */
export function processPackTask(task) {
  const taskId = task.taskId || task.id;
  const orderId = task.orderId;

  console.log(`Processing pack task ${taskId} for order ${orderId}`);

  // Step 1: Simulate packing workflow
  const packageInfo = simulatePackingTask(task);

  // Step 1b: Confirm unit packs for the order
  if (orderId) {
    const packageId = `PKG-${taskId.slice(-8)}`;
    const packerId = `PACKER-SIM-${__VU || 1}`;
    const stationId = PACKING_CONFIG.defaultStation;
    const unitResult = confirmPacksForOrder(orderId, packageId, packerId, stationId);
    if (!unitResult.skipped) {
      console.log(`Unit pack confirmations: ${unitResult.success}/${unitResult.total} succeeded`);
    }
  }

  // Step 2: Complete the task via API
  const completed = completePackTask(taskId);
  if (!completed) {
    console.warn(`Failed to complete pack task ${taskId}`);
    return false;
  }

  // Step 3: Signal the orchestrator workflow that packing is complete
  // THIS IS REQUIRED for the workflow to progress to shipping
  const signalSent = sendPackingCompleteSignal(orderId, taskId, packageInfo);
  if (!signalSent) {
    console.warn(`Failed to send packing complete signal for ${orderId}, workflow may be stuck`);
  }

  console.log(`Pack task ${taskId} completed successfully`);
  return true;
}

/**
 * Discovers and processes all pending pack tasks
 * @param {number} maxTasks - Maximum number of tasks to process
 * @returns {Object} Summary of processing results
 */
export function processAllPendingPackTasks(maxTasks = PACKING_CONFIG.maxTasksPerIteration) {
  const results = {
    discovered: 0,
    processed: 0,
    failed: 0,
    tasks: [],
  };

  // Discover pending pack tasks
  const tasks = discoverPendingPackTasks();
  results.discovered = tasks.length;

  console.log(`Discovered ${tasks.length} pending pack tasks`);

  // Process up to maxTasks
  const tasksToProcess = tasks.slice(0, maxTasks);

  for (const task of tasksToProcess) {
    const taskId = task.taskId || task.id;
    const success = processPackTask(task);

    results.tasks.push({
      taskId: taskId,
      orderId: task.orderId,
      success: success,
    });

    if (success) {
      results.processed++;
    } else {
      results.failed++;
    }
  }

  console.log(`Processed ${results.processed}/${results.discovered} pack tasks (${results.failed} failed)`);

  return results;
}
