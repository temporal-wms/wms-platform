// WES Execution Simulator
// Simulates WES multi-stage workflow execution with dynamic process path resolution

import { check, sleep, group } from 'k6';
import { Counter, Trend, Rate } from 'k6/metrics';
import {
  WES_CONFIG,
  PROCESS_PATHS,
  STAGE_TYPES,
  EXECUTION_STATUS,
  STAGE_STATUS,
  resolveProcessPath,
  getStagesForPath,
  createExecutionPlan,
  getExecutionPlan,
  getExecutionPlanByOrder,
  getCurrentStage,
  assignWorkerToStage,
  startCurrentStage,
  completeCurrentStage,
  failCurrentStage,
  executeAllStages,
  processOrderWithWES,
  discoverPendingExecutions,
} from '../lib/wes.js';
import { createOrder, getOrderStatus } from '../lib/orders.js';
import { generateOrderWithType } from '../lib/data.js';

// Custom metrics
const executionsCreated = new Counter('wes_executions_created');
const executionsCompleted = new Counter('wes_executions_completed');
const executionsFailed = new Counter('wes_executions_failed');
const stagesCompleted = new Counter('wes_stages_completed');
const stagesFailed = new Counter('wes_stages_failed');
const executionDuration = new Trend('wes_execution_duration_ms');
const stageDuration = new Trend('wes_stage_duration_ms');
const pathResolutionTime = new Trend('wes_path_resolution_ms');
const successRate = new Rate('wes_success_rate');

// Path distribution counters
const pathPickPack = new Counter('wes_path_pick_pack');
const pathPickWallPack = new Counter('wes_path_pick_wall_pack');
const pathPickConsolidatePack = new Counter('wes_path_pick_consolidate_pack');

// Test configuration
export const options = {
  scenarios: {
    wes_execution: {
      executor: 'ramping-vus',
      startVUs: 1,
      stages: [
        { duration: '30s', target: 2 },   // Ramp up
        { duration: '2m', target: 3 },    // Steady state
        { duration: '30s', target: 0 },   // Ramp down
      ],
      gracefulRampDown: '10s',
    },
  },
  thresholds: {
    'wes_success_rate': ['rate>0.90'],
    'wes_execution_duration_ms': ['p(95)<15000'],
    'wes_stage_duration_ms': ['p(95)<5000'],
    'http_req_failed': ['rate<0.05'],
  },
};

// Configuration
const CONFIG = {
  ordersPerIteration: parseInt(__ENV.ORDERS_PER_ITERATION || '3'),
  forcePath: __ENV.FORCE_PROCESS_PATH || null, // 'pick_pack', 'pick_wall_pack', 'pick_consolidate_pack'
  testAllPaths: __ENV.TEST_ALL_PATHS === 'true',
  processExisting: __ENV.PROCESS_EXISTING === 'true',
};

/**
 * Generates an order that will trigger a specific process path
 */
function generateOrderForPath(processPath) {
  switch (processPath) {
    case PROCESS_PATHS.PICK_PACK:
      // Single item order
      return generateOrderWithType('single', null);

    case PROCESS_PATHS.PICK_WALL_PACK:
      // Large order with many items
      const largeOrder = generateOrderWithType('multi', null);
      // Add more items to trigger walling
      while (largeOrder.items.length < 6) {
        const extraOrder = generateOrderWithType('single', null);
        largeOrder.items.push(...extraOrder.items);
      }
      return largeOrder;

    case PROCESS_PATHS.PICK_CONSOLIDATE_PACK:
      // Multi-item order (2-5 items)
      return generateOrderWithType('multi', null);

    default:
      return generateOrderWithType(null, null);
  }
}

/**
 * Executes a single stage with timing
 */
function executeStage(executionId, stageType) {
  const startTime = Date.now();

  // Assign worker
  if (!assignWorkerToStage(executionId)) {
    return { success: false, error: 'assignment_failed' };
  }

  // Start stage
  if (!startCurrentStage(executionId)) {
    return { success: false, error: 'start_failed' };
  }

  // Simulate stage work based on type
  const stageDelays = {
    [STAGE_TYPES.PICK]: WES_CONFIG.simulationDelayMs,
    [STAGE_TYPES.WALL]: WES_CONFIG.simulationDelayMs * 0.8,
    [STAGE_TYPES.CONSOLIDATE]: WES_CONFIG.simulationDelayMs * 0.6,
    [STAGE_TYPES.PACK]: WES_CONFIG.simulationDelayMs,
    [STAGE_TYPES.GIFT_WRAP]: WES_CONFIG.simulationDelayMs * 1.5,
  };

  sleep((stageDelays[stageType] || WES_CONFIG.simulationDelayMs) / 1000);

  // Complete stage
  const completion = completeCurrentStage(executionId, {
    stageType: stageType,
  });

  const duration = Date.now() - startTime;
  stageDuration.add(duration);

  if (completion) {
    stagesCompleted.add(1);
    return { success: true, duration: duration };
  }

  stagesFailed.add(1);
  return { success: false, error: 'completion_failed' };
}

/**
 * Processes a complete WES execution for an order
 */
function processWESExecution(order) {
  const orderId = order.orderId || order.id;
  const startTime = Date.now();

  console.log(`Processing WES execution for order ${orderId}`);

  // Resolve process path
  const pathStartTime = Date.now();
  const resolvedPath = resolveProcessPath(order);
  pathResolutionTime.add(Date.now() - pathStartTime);

  console.log(`Resolved process path: ${resolvedPath} for order ${orderId}`);

  // Track path distribution
  switch (resolvedPath) {
    case PROCESS_PATHS.PICK_PACK:
      pathPickPack.add(1);
      break;
    case PROCESS_PATHS.PICK_WALL_PACK:
      pathPickWallPack.add(1);
      break;
    case PROCESS_PATHS.PICK_CONSOLIDATE_PACK:
      pathPickConsolidatePack.add(1);
      break;
  }

  // Create execution plan
  const execution = createExecutionPlan(orderId, {
    ...order,
    processPath: resolvedPath,
  });

  if (!execution) {
    console.warn(`Failed to create execution plan for order ${orderId}`);
    return { success: false, error: 'creation_failed' };
  }

  executionsCreated.add(1);
  const executionId = execution.executionId || execution.id;
  console.log(`Created execution: ${executionId} with path ${resolvedPath}`);

  // Get stages for the path
  const stages = getStagesForPath(resolvedPath);
  console.log(`Executing ${stages.length} stages: ${stages.join(' → ')}`);

  // Execute each stage
  const stageResults = [];
  for (const stageType of stages) {
    console.log(`Executing stage: ${stageType}`);
    const stageResult = executeStage(executionId, stageType);
    stageResults.push({
      stageType: stageType,
      ...stageResult,
    });

    if (!stageResult.success) {
      console.warn(`Stage ${stageType} failed: ${stageResult.error}`);
      executionsFailed.add(1);
      successRate.add(0);

      return {
        success: false,
        executionId: executionId,
        orderId: orderId,
        processPath: resolvedPath,
        failedStage: stageType,
        stageResults: stageResults,
        duration: Date.now() - startTime,
      };
    }

    // Small delay between stages
    sleep(WES_CONFIG.stageTransitionDelayMs / 1000);
  }

  const totalDuration = Date.now() - startTime;
  executionDuration.add(totalDuration);
  executionsCompleted.add(1);
  successRate.add(1);

  console.log(`Completed execution ${executionId} in ${totalDuration}ms`);

  return {
    success: true,
    executionId: executionId,
    orderId: orderId,
    processPath: resolvedPath,
    stagesCompleted: stages.length,
    stageResults: stageResults,
    duration: totalDuration,
  };
}

/**
 * Main test function
 */
export default function () {
  const vuId = __VU;
  const iterationId = __ITER;

  console.log(`[VU ${vuId}] Starting WES execution simulation - iteration ${iterationId}`);

  // Phase 1: Test all process paths if configured
  if (CONFIG.testAllPaths && iterationId === 0) {
    group('Test All Process Paths', function () {
      const paths = [
        PROCESS_PATHS.PICK_PACK,
        PROCESS_PATHS.PICK_WALL_PACK,
        PROCESS_PATHS.PICK_CONSOLIDATE_PACK,
      ];

      for (const path of paths) {
        console.log(`[VU ${vuId}] Testing path: ${path}`);

        const order = generateOrderForPath(path);
        const orderResult = createOrder(order);

        if (orderResult && orderResult.orderId) {
          const result = processWESExecution({
            ...order,
            orderId: orderResult.orderId,
          });

          console.log(`[VU ${vuId}] Path ${path}: ${result.success ? 'SUCCESS' : 'FAILED'}`);
        }

        sleep(1);
      }
    });
  }

  // Phase 2: Create and process orders with WES
  group('Process Orders with WES', function () {
    for (let i = 0; i < CONFIG.ordersPerIteration; i++) {
      // Generate order (optionally forcing a specific path)
      let order;
      if (CONFIG.forcePath) {
        order = generateOrderForPath(CONFIG.forcePath);
      } else {
        // Random order type
        const orderTypes = ['single', 'multi', null];
        const randomType = orderTypes[Math.floor(Math.random() * orderTypes.length)];
        order = generateOrderWithType(randomType, null);
      }

      // Create the order
      const orderResult = createOrder(order);
      if (!orderResult || !orderResult.orderId) {
        console.warn(`[VU ${vuId}] Failed to create order`);
        continue;
      }

      const orderId = orderResult.orderId;
      console.log(`[VU ${vuId}] Created order ${orderId} with ${order.items.length} items`);

      // Process through WES
      const result = processWESExecution({
        ...order,
        orderId: orderId,
      });

      if (result.success) {
        console.log(`[VU ${vuId}] Order ${orderId} completed via ${result.processPath} in ${result.duration}ms`);
      } else {
        console.warn(`[VU ${vuId}] Order ${orderId} failed at stage ${result.failedStage}`);
      }

      sleep(1);
    }
  });

  // Phase 3: Process any existing pending executions
  if (CONFIG.processExisting) {
    group('Process Pending Executions', function () {
      const pendingExecutions = discoverPendingExecutions();
      console.log(`[VU ${vuId}] Found ${pendingExecutions.length} pending executions`);

      for (const execution of pendingExecutions.slice(0, 3)) {
        const executionId = execution.executionId || execution.id;
        const result = executeAllStages(executionId);

        if (result.success) {
          executionsCompleted.add(1);
          successRate.add(1);
          console.log(`[VU ${vuId}] Completed pending execution ${executionId}`);
        } else {
          executionsFailed.add(1);
          successRate.add(0);
          console.warn(`[VU ${vuId}] Failed pending execution ${executionId}`);
        }
      }
    });
  }

  // Brief pause between iterations
  sleep(2);
}

/**
 * Setup function
 */
export function setup() {
  console.log('='.repeat(60));
  console.log('WES Execution Simulator - Setup');
  console.log('='.repeat(60));
  console.log(`Orders per iteration: ${CONFIG.ordersPerIteration}`);
  console.log(`Force path: ${CONFIG.forcePath || 'auto'}`);
  console.log(`Test all paths: ${CONFIG.testAllPaths}`);
  console.log(`Process existing: ${CONFIG.processExisting}`);
  console.log('='.repeat(60));

  return {
    startTime: Date.now(),
  };
}

/**
 * Teardown function
 */
export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;

  console.log('='.repeat(60));
  console.log('WES Execution Simulator - Summary');
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
    simulator: 'wes-execution-simulator',
    metrics: {
      executions_created: data.metrics.wes_executions_created?.values?.count || 0,
      executions_completed: data.metrics.wes_executions_completed?.values?.count || 0,
      executions_failed: data.metrics.wes_executions_failed?.values?.count || 0,
      stages_completed: data.metrics.wes_stages_completed?.values?.count || 0,
      stages_failed: data.metrics.wes_stages_failed?.values?.count || 0,
      success_rate: data.metrics.wes_success_rate?.values?.rate || 0,
      avg_execution_duration_ms: data.metrics.wes_execution_duration_ms?.values?.avg || 0,
      p95_execution_duration_ms: data.metrics.wes_execution_duration_ms?.values?.['p(95)'] || 0,
      avg_stage_duration_ms: data.metrics.wes_stage_duration_ms?.values?.avg || 0,
      path_distribution: {
        pick_pack: data.metrics.wes_path_pick_pack?.values?.count || 0,
        pick_wall_pack: data.metrics.wes_path_pick_wall_pack?.values?.count || 0,
        pick_consolidate_pack: data.metrics.wes_path_pick_consolidate_pack?.values?.count || 0,
      },
    },
    thresholds: data.thresholds,
  };

  return {
    'stdout': JSON.stringify(summary, null, 2) + '\n',
    'wes-execution-results.json': JSON.stringify(summary, null, 2),
  };
}
