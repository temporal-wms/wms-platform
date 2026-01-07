// K6 Stow Service Helper Library
// Provides functions for simulating putaway/stowage operations

import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URLS, ENDPOINTS, HTTP_PARAMS, SIGNAL_CONFIG } from './config.js';

// Stow-specific configuration
export const STOW_CONFIG = {
  simulationDelayMs: parseInt(__ENV.STOW_DELAY_MS || '600'),
  maxTasksPerIteration: parseInt(__ENV.MAX_STOW_TASKS || '10'),
  locationSelectionDelayMs: parseInt(__ENV.STOW_LOCATION_DELAY_MS || '300'),
  defaultZone: __ENV.DEFAULT_STOW_ZONE || 'RESERVE',
};

// Stow task status constants
export const STOW_TASK_STATUS = {
  PENDING: 'pending',
  ASSIGNED: 'assigned',
  IN_PROGRESS: 'in_progress',
  COMPLETED: 'completed',
  CANCELLED: 'cancelled',
};

// Location types for stow
export const LOCATION_TYPES = {
  RESERVE: 'reserve',
  FORWARD_PICK: 'forward_pick',
  OVERFLOW: 'overflow',
  BULK: 'bulk',
};

/**
 * Discovers pending stow tasks
 * @param {string} status - Task status to filter by
 * @returns {Array} Array of stow tasks
 */
export function discoverStowTasks(status = STOW_TASK_STATUS.PENDING) {
  const url = `${BASE_URLS.stow}/api/v1/tasks?status=${status}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'discover stow tasks status 200': (r) => r.status === 200,
  });

  if (!success || !response.body) {
    console.warn(`Failed to discover stow tasks: ${response.status}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return Array.isArray(data) ? data : (data.tasks || data.items || []);
  } catch (e) {
    console.error(`Failed to parse stow tasks response: ${e.message}`);
    return [];
  }
}

/**
 * Gets a specific stow task by ID
 * @param {string} taskId - The task ID
 * @returns {Object|null} The stow task or null if not found
 */
export function getStowTask(taskId) {
  const url = `${BASE_URLS.stow}/api/v1/tasks/${taskId}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get stow task status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to get stow task ${taskId}: ${response.status}`);
    return null;
  }

  try {
    return JSON.parse(response.body);
  } catch (e) {
    console.error(`Failed to parse stow task response: ${e.message}`);
    return null;
  }
}

/**
 * Creates a stow task for received items
 * @param {Object} taskData - Stow task details
 * @returns {Object|null} Created stow task or null if failed
 */
export function createStowTask(taskData) {
  const url = `${BASE_URLS.stow}/api/v1/tasks`;

  // Map priority string to int (1=low, 2=normal, 3=high, 4=urgent)
  let priorityInt = 2; // default normal
  if (typeof taskData.priority === 'string') {
    const priorityMap = { 'low': 1, 'normal': 2, 'high': 3, 'urgent': 4 };
    priorityInt = priorityMap[taskData.priority] || 2;
  } else if (typeof taskData.priority === 'number') {
    priorityInt = taskData.priority;
  }

  const payload = JSON.stringify({
    shipmentId: taskData.shipmentId,
    sku: taskData.sku,
    productName: taskData.productName || taskData.sku || 'Unknown Product',
    quantity: taskData.quantity,
    sourceToteId: taskData.sourceToteId || taskData.licensePlate || `TOTE-RCV-${Date.now()}`,
    sourceLocationId: taskData.sourceLocationId || taskData.sourceLocation || 'RECEIVING-DOCK',
    isHazmat: taskData.isHazmat || false,
    requiresColdChain: taskData.requiresColdChain || false,
    isOversized: taskData.isOversized || false,
    isFragile: taskData.isFragile || false,
    weight: taskData.weight || 0,
    priority: priorityInt,
    strategy: taskData.strategy || 'chaotic',
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'create stow task status 201': (r) => r.status === 201 || r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to create stow task: ${response.status} - ${response.body}`);
    return null;
  }

  try {
    const result = JSON.parse(response.body);
    console.log(`Created stow task: ${result.taskId || result.id}`);
    return result;
  } catch (e) {
    console.error(`Failed to parse stow task response: ${e.message}`);
    return null;
  }
}

/**
 * Assigns a stow task to a worker
 * @param {string} taskId - The task ID
 * @param {string} workerId - The worker ID
 * @returns {boolean} True if successful
 */
export function assignStowTask(taskId, workerId = null) {
  const effectiveWorkerId = workerId || `STOWER-${__VU || 1}`;
  const url = `${BASE_URLS.stow}/api/v1/tasks/${taskId}/assign`;
  const payload = JSON.stringify({
    workerId: effectiveWorkerId,
    assignedAt: new Date().toISOString(),
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'assign stow task status 200': (r) => r.status === 200 || r.status === 204,
  });

  if (!success) {
    console.warn(`Failed to assign stow task ${taskId}: ${response.status}`);
  } else {
    console.log(`Assigned stow task ${taskId} to worker ${effectiveWorkerId}`);
  }

  return success;
}

/**
 * Starts a stow task
 * @param {string} taskId - The task ID
 * @returns {boolean} True if successful
 */
export function startStowTask(taskId) {
  const url = `${BASE_URLS.stow}/api/v1/tasks/${taskId}/start`;
  const payload = JSON.stringify({
    startedAt: new Date().toISOString(),
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'start stow task status 200': (r) => r.status === 200 || r.status === 204,
  });

  if (!success) {
    console.warn(`Failed to start stow task ${taskId}: ${response.status}`);
  }

  return success;
}

/**
 * Selects an optimal stow location for an item
 * @param {string} sku - The SKU to stow
 * @param {number} quantity - Quantity to stow
 * @param {string} zone - Target zone preference
 * @returns {Object|null} Selected location or null if none available
 */
export function selectStowLocation(sku, quantity, zone = null) {
  const effectiveZone = zone || STOW_CONFIG.defaultZone;
  const url = `${BASE_URLS.stow}/api/v1/locations/suggest`;
  const payload = JSON.stringify({
    sku: sku,
    quantity: quantity,
    zone: effectiveZone,
    locationType: LOCATION_TYPES.RESERVE,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'select stow location status 200': (r) => r.status === 200,
  });

  if (!success) {
    // Try to get any available location
    console.warn(`Location suggestion failed, using fallback for ${sku}`);
    return {
      locationId: `LOC-${effectiveZone}-${Math.floor(Math.random() * 100)}`,
      zone: effectiveZone,
      type: LOCATION_TYPES.RESERVE,
    };
  }

  try {
    return JSON.parse(response.body);
  } catch (e) {
    console.error(`Failed to parse location suggestion: ${e.message}`);
    return null;
  }
}

/**
 * Records stow quantity for a task
 * @param {string} taskId - The task ID
 * @param {number} quantity - Quantity stowed
 * @returns {boolean} True if successful
 */
export function recordStow(taskId, quantity) {
  const url = `${BASE_URLS.stow}/api/v1/tasks/${taskId}/stow`;
  const payload = JSON.stringify({
    quantity: quantity,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'record stow status 200': (r) => r.status === 200 || r.status === 204,
  });

  if (!success) {
    console.warn(`Failed to record stow for task ${taskId}: ${response.status} - ${response.body}`);
  } else {
    console.log(`Recorded stow for task ${taskId}: ${quantity} units`);
  }

  return success;
}

/**
 * Confirms stow to a location (alias for recordStow for backward compatibility)
 * @param {string} taskId - The task ID
 * @param {string} locationId - The target location ID (ignored - API doesn't use it)
 * @param {number} quantity - Quantity stowed
 * @returns {boolean} True if successful
 */
export function confirmStow(taskId, locationId, quantity) {
  return recordStow(taskId, quantity);
}

/**
 * Completes a stow task
 * @param {string} taskId - The task ID
 * @param {Object} completionDetails - Completion details
 * @returns {Object|null} Completed task or null if failed
 */
export function completeStowTask(taskId, completionDetails = {}) {
  const url = `${BASE_URLS.stow}/api/v1/tasks/${taskId}/complete`;
  const payload = JSON.stringify({
    completedAt: new Date().toISOString(),
    completedBy: `STOWER-${__VU || 1}`,
    locationId: completionDetails.locationId,
    quantity: completionDetails.quantity,
    ...completionDetails,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'complete stow task status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to complete stow task ${taskId}: ${response.status}`);
    return null;
  }

  try {
    const result = JSON.parse(response.body);
    console.log(`Completed stow task ${taskId}`);
    return result;
  } catch (e) {
    console.error(`Failed to parse complete response: ${e.message}`);
    return null;
  }
}

/**
 * Updates inventory after stow
 * @param {string} locationId - The location where items were stowed
 * @param {string} sku - The SKU stowed
 * @param {number} quantity - Quantity stowed
 * @returns {boolean} True if successful
 */
export function updateInventoryAfterStow(locationId, sku, quantity) {
  const url = `${BASE_URLS.inventory}/api/v1/locations/${locationId}/stock`;
  const payload = JSON.stringify({
    sku: sku,
    quantity: quantity,
    action: 'add',
    source: 'stow',
    timestamp: new Date().toISOString(),
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'update inventory after stow status 200': (r) => r.status === 200 || r.status === 204,
  });

  if (!success) {
    console.warn(`Failed to update inventory for ${sku} at ${locationId}: ${response.status}`);
  }

  return success;
}

/**
 * Sends stow completed signal to the orchestrator
 * @param {string} shipmentId - The shipment ID
 * @param {Array} stowedItems - Array of stowed items
 * @returns {boolean} True if successful
 */
export function signalStowCompleted(shipmentId, stowedItems) {
  const url = `${BASE_URLS.orchestrator}/api/v1/signals/stow-completed`;
  const payload = JSON.stringify({
    shipmentId: shipmentId,
    stowedItems: stowedItems,
    completedAt: new Date().toISOString(),
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
      'signal stow completed status 200': (r) => r.status === 200,
    });

    if (success) {
      if (attempt > 1) {
        console.log(`Signal succeeded on attempt ${attempt}/${SIGNAL_CONFIG.maxRetries}`);
      }
      break;
    }

    if (attempt < SIGNAL_CONFIG.maxRetries) {
      console.warn(`Signal attempt ${attempt}/${SIGNAL_CONFIG.maxRetries} failed: ${lastResponse.status}, retrying...`);
      sleep(SIGNAL_CONFIG.retryDelayMs / 1000);
    }
  }

  if (!success) {
    console.warn(`Failed to signal stow completed for ${shipmentId}: ${lastResponse.status}`);
  } else {
    console.log(`Signaled stow completed for shipment ${shipmentId}`);
  }

  return success;
}

/**
 * Simulates stowing all items in a task
 * @param {Object} task - The stow task object
 * @returns {Object|null} Stow result with location info
 */
export function simulateStowingTask(task) {
  const taskId = task.taskId || task.id;
  console.log(`Simulating stow for task ${taskId}`);

  // Step 1: Assign the task to a worker
  const assignSuccess = assignStowTask(taskId);
  if (!assignSuccess) {
    console.warn(`Failed to assign stow task ${taskId}, trying direct complete`);
  }

  // Try the full flow first (assign -> start -> stow -> complete)
  let fullFlowSuccess = false;

  if (assignSuccess) {
    // Step 2: Start the task (may fail if no location was assigned)
    const startSuccess = startStowTask(taskId);

    if (startSuccess) {
      // Simulate physical stow delay
      sleep(STOW_CONFIG.simulationDelayMs / 1000);

      // Step 3: Record the stow
      const stowSuccess = recordStow(taskId, task.quantity);

      if (stowSuccess) {
        fullFlowSuccess = true;
      }
    }
  }

  // If full flow failed, try direct complete
  // CompleteTask in the service auto-stows remaining items
  if (!fullFlowSuccess) {
    console.log(`Full stow flow failed for ${taskId}, trying direct complete`);
    sleep(STOW_CONFIG.simulationDelayMs / 1000);
  }

  // Step 4: Complete the task (also stows remaining items)
  const completedTask = completeStowTask(taskId, {
    quantity: task.quantity,
  });

  const locationId = `LOC-RESERVE-${Math.floor(Math.random() * 100)}`;

  if (completedTask) {
    // Update inventory (optional, may fail)
    updateInventoryAfterStow(locationId, task.sku, task.quantity);

    return {
      taskId: taskId,
      sku: task.sku,
      quantity: task.quantity,
      locationId: completedTask.targetLocationId || locationId,
      zone: STOW_CONFIG.defaultZone,
    };
  }

  console.warn(`Failed to complete stow task ${taskId}`);
  return null;
}

/**
 * Processes a single stow task end-to-end
 * @param {Object} task - The stow task object
 * @returns {boolean} True if fully successful
 */
export function processStowTask(task) {
  const taskId = task.taskId || task.id;
  console.log(`Processing stow task ${taskId}`);

  // Step 1: Simulate stowing
  const stowResult = simulateStowingTask(task);
  if (!stowResult) {
    console.warn(`Failed to stow task ${taskId}`);
    return false;
  }

  // Step 2: Complete the task
  const completedTask = completeStowTask(taskId, {
    locationId: stowResult.locationId,
    quantity: stowResult.quantity,
  });

  return completedTask !== null;
}

/**
 * Processes stow tasks for a shipment and signals completion
 * @param {string} shipmentId - The shipment ID
 * @param {Array} tasks - Stow tasks for the shipment
 * @returns {Object} Processing results
 */
export function processShipmentStowTasks(shipmentId, tasks) {
  const results = {
    shipmentId: shipmentId,
    processed: 0,
    failed: 0,
    stowedItems: [],
  };

  for (const task of tasks) {
    const stowResult = simulateStowingTask(task);
    if (stowResult) {
      results.stowedItems.push(stowResult);
      results.processed++;
      // Note: Task already completed in simulateStowingTask
    } else {
      results.failed++;
    }
  }

  // Signal stow completion for the shipment
  if (results.stowedItems.length > 0) {
    signalStowCompleted(shipmentId, results.stowedItems);
  }

  console.log(`Shipment ${shipmentId}: stowed ${results.processed}/${tasks.length} items`);

  return results;
}

/**
 * Discovers and processes all pending stow tasks
 * @param {number} maxTasks - Maximum number of tasks to process
 * @returns {Object} Summary of processing results
 */
export function processAllPendingStowTasks(maxTasks = STOW_CONFIG.maxTasksPerIteration) {
  const results = {
    discovered: 0,
    processed: 0,
    failed: 0,
    tasks: [],
  };

  // Discover pending stow tasks
  const tasks = discoverStowTasks(STOW_TASK_STATUS.PENDING);
  results.discovered = tasks.length;

  console.log(`Discovered ${tasks.length} pending stow tasks`);

  // Process up to maxTasks
  const tasksToProcess = tasks.slice(0, maxTasks);

  for (const task of tasksToProcess) {
    const success = processStowTask(task);

    results.tasks.push({
      taskId: task.taskId || task.id,
      sku: task.sku,
      success: success,
    });

    if (success) {
      results.processed++;
    } else {
      results.failed++;
    }
  }

  console.log(`Processed ${results.processed}/${results.discovered} stow tasks (${results.failed} failed)`);

  return results;
}

/**
 * Gets stow tasks by shipment ID
 * @param {string} shipmentId - The shipment ID
 * @returns {Array} Stow tasks for the shipment
 */
export function getStowTasksByShipment(shipmentId) {
  const url = `${BASE_URLS.stow}/api/v1/tasks/shipment/${shipmentId}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get shipment stow tasks status 200': (r) => r.status === 200,
  });

  if (!success || !response.body) {
    console.warn(`Failed to get stow tasks for shipment ${shipmentId}: ${response.status}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return Array.isArray(data) ? data : (data.tasks || []);
  } catch (e) {
    console.error(`Failed to parse stow tasks response: ${e.message}`);
    return [];
  }
}
