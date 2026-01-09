// K6 WES (Warehouse Execution System) Helper Library
// Provides functions for WES execution plan orchestration and multi-stage workflows

import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URLS, ENDPOINTS, HTTP_PARAMS, SIGNAL_CONFIG } from './config.js';

// WES-specific configuration
export const WES_CONFIG = {
  simulationDelayMs: parseInt(__ENV.WES_DELAY_MS || '500'),
  stageTransitionDelayMs: parseInt(__ENV.WES_STAGE_TRANSITION_MS || '300'),
  maxExecutionsPerIteration: parseInt(__ENV.MAX_WES_EXECUTIONS || '10'),
};

// Process paths supported by WES
export const PROCESS_PATHS = {
  PICK_PACK: 'pick_pack',                   // 2-stage: Pick → Pack
  PICK_WALL_PACK: 'pick_wall_pack',         // 3-stage: Pick → Wall → Pack
  PICK_CONSOLIDATE_PACK: 'pick_consolidate_pack', // 3-stage: Pick → Consolidate → Pack
};

// Execution stage types
export const STAGE_TYPES = {
  PICK: 'pick',
  WALL: 'wall',
  CONSOLIDATE: 'consolidate',
  PACK: 'pack',
  GIFT_WRAP: 'gift_wrap',
};

// Execution status
export const EXECUTION_STATUS = {
  CREATED: 'created',
  IN_PROGRESS: 'in_progress',
  COMPLETED: 'completed',
  FAILED: 'failed',
  CANCELLED: 'cancelled',
};

// Stage status
export const STAGE_STATUS = {
  PENDING: 'pending',
  ASSIGNED: 'assigned',
  IN_PROGRESS: 'in_progress',
  COMPLETED: 'completed',
  FAILED: 'failed',
  SKIPPED: 'skipped',
};

/**
 * Resolves the optimal process path for an order based on its characteristics
 * @param {Object} order - Order object with items and requirements
 * @returns {string} Process path (pick_pack, pick_wall_pack, or pick_consolidate_pack)
 */
export function resolveProcessPath(order) {
  const items = order.items || [];
  const itemCount = items.length;
  const totalQuantity = items.reduce((sum, item) => sum + (item.quantity || 1), 0);
  const requirements = order.requirements || [];

  // Single-item orders with single quantity: simple pick_pack
  if (itemCount === 1 && totalQuantity === 1) {
    return PROCESS_PATHS.PICK_PACK;
  }

  // Orders requiring walling (many items, specific zones, or explicit requirement)
  const hasWallingRequirement = requirements.includes('walling') ||
    requirements.includes('sortation');
  const isLargeOrder = itemCount > 5 || totalQuantity > 10;

  if (hasWallingRequirement || (isLargeOrder && itemCount > 3)) {
    return PROCESS_PATHS.PICK_WALL_PACK;
  }

  // Multi-item orders: need consolidation
  if (itemCount > 1 || totalQuantity > 1) {
    return PROCESS_PATHS.PICK_CONSOLIDATE_PACK;
  }

  // Default to simple pick_pack
  return PROCESS_PATHS.PICK_PACK;
}

/**
 * Gets the stages for a given process path
 * @param {string} processPath - The process path
 * @returns {Array} Ordered list of stage types
 */
export function getStagesForPath(processPath) {
  switch (processPath) {
    case PROCESS_PATHS.PICK_PACK:
      return [STAGE_TYPES.PICK, STAGE_TYPES.PACK];
    case PROCESS_PATHS.PICK_WALL_PACK:
      return [STAGE_TYPES.PICK, STAGE_TYPES.WALL, STAGE_TYPES.PACK];
    case PROCESS_PATHS.PICK_CONSOLIDATE_PACK:
      return [STAGE_TYPES.PICK, STAGE_TYPES.CONSOLIDATE, STAGE_TYPES.PACK];
    default:
      return [STAGE_TYPES.PICK, STAGE_TYPES.PACK];
  }
}

/**
 * Creates a WES execution plan for an order
 * @param {string} orderId - The order ID
 * @param {Object} orderDetails - Order details for path resolution
 * @returns {Object|null} Created execution plan or null if failed
 */
export function createExecutionPlan(orderId, orderDetails = {}) {
  const processPath = orderDetails.processPath || resolveProcessPath(orderDetails);
  const stages = getStagesForPath(processPath);

  const url = `${BASE_URLS.wes}${ENDPOINTS.wes.resolveExecutionPlan}`;
  const payload = JSON.stringify({
    orderId: orderId,
    processPath: processPath,
    stages: stages.map((stageType, index) => ({
      stageType: stageType,
      sequence: index + 1,
      status: index === 0 ? STAGE_STATUS.PENDING : STAGE_STATUS.PENDING,
    })),
    priority: orderDetails.priority || 'normal',
    requirements: orderDetails.requirements || [],
    giftWrapRequired: orderDetails.giftWrap || false,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'create execution plan status 201': (r) => r.status === 201 || r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to create execution plan for order ${orderId}: ${response.status} - ${response.body}`);
    return null;
  }

  try {
    const result = JSON.parse(response.body);
    console.log(`Created execution plan: ${result.executionId || result.id} with path ${processPath}`);
    return result;
  } catch (e) {
    console.error(`Failed to parse execution plan response: ${e.message}`);
    return null;
  }
}

/**
 * Gets an execution plan by ID
 * @param {string} executionId - The execution ID
 * @returns {Object|null} Execution plan or null if not found
 */
export function getExecutionPlan(executionId) {
  const url = `${BASE_URLS.wes}/api/v1/executions/${executionId}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get execution plan status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to get execution plan ${executionId}: ${response.status}`);
    return null;
  }

  try {
    return JSON.parse(response.body);
  } catch (e) {
    console.error(`Failed to parse execution plan response: ${e.message}`);
    return null;
  }
}

/**
 * Gets execution plan by order ID
 * @param {string} orderId - The order ID
 * @returns {Object|null} Execution plan or null if not found
 */
export function getExecutionPlanByOrder(orderId) {
  const url = `${BASE_URLS.wes}/api/v1/executions/order/${orderId}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get execution by order status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to get execution plan for order ${orderId}: ${response.status}`);
    return null;
  }

  try {
    return JSON.parse(response.body);
  } catch (e) {
    console.error(`Failed to parse execution plan response: ${e.message}`);
    return null;
  }
}

/**
 * Gets the current stage of an execution
 * @param {string} executionId - The execution ID
 * @returns {Object|null} Current stage or null
 */
export function getCurrentStage(executionId) {
  const url = `${BASE_URLS.wes}/api/v1/executions/${executionId}/stages/current`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get current stage status 200': (r) => r.status === 200,
  });

  if (!success) {
    return null;
  }

  try {
    return JSON.parse(response.body);
  } catch (e) {
    console.error(`Failed to parse stage response: ${e.message}`);
    return null;
  }
}

/**
 * Assigns a worker to the current stage
 * @param {string} executionId - The execution ID
 * @param {string} workerId - The worker ID
 * @returns {boolean} True if successful
 */
export function assignWorkerToStage(executionId, workerId = null) {
  const effectiveWorkerId = workerId || `WORKER-${__VU || 1}`;
  const url = `${BASE_URLS.wes}/api/v1/executions/${executionId}/stages/current/assign`;
  const payload = JSON.stringify({
    workerId: effectiveWorkerId,
    assignedAt: new Date().toISOString(),
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'assign worker to stage status 200': (r) => r.status === 200 || r.status === 204,
  });

  if (!success) {
    console.warn(`Failed to assign worker to execution ${executionId}: ${response.status}`);
  }

  return success;
}

/**
 * Starts the current stage of an execution
 * @param {string} executionId - The execution ID
 * @returns {boolean} True if successful
 */
export function startCurrentStage(executionId) {
  const url = `${BASE_URLS.wes}/api/v1/executions/${executionId}/stages/current/start`;
  const payload = JSON.stringify({
    startedAt: new Date().toISOString(),
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'start stage status 200': (r) => r.status === 200 || r.status === 204,
  });

  if (!success) {
    console.warn(`Failed to start stage for execution ${executionId}: ${response.status}`);
  }

  return success;
}

/**
 * Completes the current stage and advances to the next
 * @param {string} executionId - The execution ID
 * @param {Object} completionData - Stage completion data
 * @returns {Object|null} Updated execution or null if failed
 */
export function completeCurrentStage(executionId, completionData = {}) {
  const url = `${BASE_URLS.wes}/api/v1/executions/${executionId}/stages/current/complete`;
  const payload = JSON.stringify({
    completedAt: new Date().toISOString(),
    completedBy: completionData.workerId || `WORKER-${__VU || 1}`,
    result: completionData.result || 'success',
    metrics: completionData.metrics || {},
    ...completionData,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'complete stage status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to complete stage for execution ${executionId}: ${response.status}`);
    return null;
  }

  try {
    return JSON.parse(response.body);
  } catch (e) {
    console.error(`Failed to parse stage completion response: ${e.message}`);
    return null;
  }
}

/**
 * Fails the current stage with an error
 * @param {string} executionId - The execution ID
 * @param {string} reason - Failure reason
 * @returns {boolean} True if successful
 */
export function failCurrentStage(executionId, reason) {
  const url = `${BASE_URLS.wes}/api/v1/executions/${executionId}/stages/current/fail`;
  const payload = JSON.stringify({
    failedAt: new Date().toISOString(),
    reason: reason,
    retryable: true,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'fail stage status 200': (r) => r.status === 200 || r.status === 204,
  });

  if (!success) {
    console.warn(`Failed to fail stage for execution ${executionId}: ${response.status}`);
  }

  return success;
}

/**
 * Gets the execution status
 * @param {string} executionId - The execution ID
 * @returns {string|null} Execution status or null
 */
export function getExecutionStatus(executionId) {
  const execution = getExecutionPlan(executionId);
  return execution?.status || null;
}

/**
 * Checks if execution is complete
 * @param {string} executionId - The execution ID
 * @returns {boolean} True if complete
 */
export function isExecutionComplete(executionId) {
  const status = getExecutionStatus(executionId);
  return status === EXECUTION_STATUS.COMPLETED;
}

/**
 * Advances through all stages of an execution
 * @param {string} executionId - The execution ID
 * @param {Object} options - Execution options
 * @returns {Object} Execution results
 */
export function executeAllStages(executionId, options = {}) {
  const results = {
    executionId: executionId,
    stagesCompleted: 0,
    stagesFailed: 0,
    stages: [],
    success: false,
  };

  let execution = getExecutionPlan(executionId);
  if (!execution) {
    console.warn(`Execution ${executionId} not found`);
    return results;
  }

  const totalStages = execution.stages?.length || 0;
  console.log(`Executing ${totalStages} stages for execution ${executionId}`);

  while (!isExecutionComplete(executionId)) {
    const currentStage = getCurrentStage(executionId);
    if (!currentStage) {
      console.warn(`No current stage found for execution ${executionId}`);
      break;
    }

    const stageResult = {
      stageType: currentStage.stageType,
      sequence: currentStage.sequence,
      success: false,
    };

    // Assign worker
    if (!assignWorkerToStage(executionId)) {
      console.warn(`Failed to assign worker for stage ${currentStage.stageType}`);
      stageResult.error = 'assignment_failed';
      results.stages.push(stageResult);
      results.stagesFailed++;
      break;
    }

    // Start stage
    if (!startCurrentStage(executionId)) {
      console.warn(`Failed to start stage ${currentStage.stageType}`);
      stageResult.error = 'start_failed';
      results.stages.push(stageResult);
      results.stagesFailed++;
      break;
    }

    // Simulate stage work
    const stageDelay = options.stageDelays?.[currentStage.stageType] ||
      WES_CONFIG.simulationDelayMs;
    sleep(stageDelay / 1000);

    // Complete stage
    const completion = completeCurrentStage(executionId, {
      stageType: currentStage.stageType,
    });

    if (completion) {
      stageResult.success = true;
      results.stagesCompleted++;
    } else {
      stageResult.error = 'completion_failed';
      results.stagesFailed++;
    }

    results.stages.push(stageResult);

    // Transition delay
    sleep(WES_CONFIG.stageTransitionDelayMs / 1000);
  }

  results.success = results.stagesFailed === 0 && results.stagesCompleted > 0;
  console.log(`Execution ${executionId}: completed ${results.stagesCompleted}/${totalStages} stages`);

  return results;
}

/**
 * Handles stage transition with appropriate signaling
 * @param {string} orderId - The order ID
 * @param {string} fromStage - The completing stage
 * @param {string} toStage - The next stage
 * @param {Object} transitionData - Data to pass to next stage
 * @returns {boolean} True if successful
 */
export function handleStageTransition(orderId, fromStage, toStage, transitionData = {}) {
  const url = `${BASE_URLS.wes}/api/v1/transitions`;
  const payload = JSON.stringify({
    orderId: orderId,
    fromStage: fromStage,
    toStage: toStage,
    transitionedAt: new Date().toISOString(),
    data: transitionData,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'stage transition status 200': (r) => r.status === 200 || r.status === 201,
  });

  if (!success) {
    console.warn(`Failed stage transition ${fromStage} → ${toStage} for order ${orderId}: ${response.status}`);
  } else {
    console.log(`Stage transition: ${fromStage} → ${toStage} for order ${orderId}`);
  }

  return success;
}

/**
 * Discovers pending WES executions
 * @param {string} status - Execution status to filter by
 * @returns {Array} Array of pending executions
 */
export function discoverPendingExecutions(status = EXECUTION_STATUS.CREATED) {
  const url = `${BASE_URLS.wes}/api/v1/executions?status=${status}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'discover executions status 200': (r) => r.status === 200,
  });

  if (!success || !response.body) {
    console.warn(`Failed to discover executions: ${response.status}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return Array.isArray(data) ? data : (data.executions || data.items || []);
  } catch (e) {
    console.error(`Failed to parse executions response: ${e.message}`);
    return [];
  }
}

/**
 * Processes all pending WES executions
 * @param {number} maxExecutions - Maximum executions to process
 * @returns {Object} Processing results
 */
export function processAllPendingExecutions(maxExecutions = WES_CONFIG.maxExecutionsPerIteration) {
  const results = {
    discovered: 0,
    processed: 0,
    failed: 0,
    executions: [],
  };

  const executions = discoverPendingExecutions();
  results.discovered = executions.length;

  console.log(`Discovered ${executions.length} pending WES executions`);

  const toProcess = executions.slice(0, maxExecutions);

  for (const execution of toProcess) {
    const executionId = execution.executionId || execution.id;
    const result = executeAllStages(executionId);

    results.executions.push({
      executionId: executionId,
      orderId: execution.orderId,
      success: result.success,
      stagesCompleted: result.stagesCompleted,
    });

    if (result.success) {
      results.processed++;
    } else {
      results.failed++;
    }
  }

  console.log(`Processed ${results.processed}/${results.discovered} executions (${results.failed} failed)`);

  return results;
}

/**
 * Creates and executes a WES plan for an order
 * @param {string} orderId - The order ID
 * @param {Object} orderDetails - Order details
 * @returns {Object} Execution results
 */
export function processOrderWithWES(orderId, orderDetails = {}) {
  console.log(`Processing order ${orderId} with WES`);

  // Create execution plan
  const execution = createExecutionPlan(orderId, orderDetails);
  if (!execution) {
    return {
      success: false,
      error: 'failed_to_create_execution',
    };
  }

  const executionId = execution.executionId || execution.id;

  // Execute all stages
  const results = executeAllStages(executionId, {
    stageDelays: {
      [STAGE_TYPES.PICK]: WES_CONFIG.simulationDelayMs,
      [STAGE_TYPES.WALL]: WES_CONFIG.simulationDelayMs * 0.8,
      [STAGE_TYPES.CONSOLIDATE]: WES_CONFIG.simulationDelayMs * 0.6,
      [STAGE_TYPES.PACK]: WES_CONFIG.simulationDelayMs,
    },
  });

  return {
    ...results,
    orderId: orderId,
    processPath: execution.processPath,
  };
}
