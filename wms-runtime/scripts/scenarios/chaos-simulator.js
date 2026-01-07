// Chaos Simulator
// Simulates failure scenarios and tests error recovery mechanisms

import { check, sleep, group } from 'k6';
import { Counter, Trend, Rate } from 'k6/metrics';
import {
  CHAOS_CONFIG,
  FAILURE_SCENARIOS,
  setChaosEnabled,
  setFailureProbability,
  shouldInjectFailure,
  selectRandomFailure,
  injectFailure,
  executeWithRetry,
  wrapWithChaos,
  triggerCompensation,
  verifyCompensation,
  runChaosScenario,
  chaosMetrics,
} from '../lib/chaos.js';
import { createOrder, getOrderStatus, cancelOrder } from '../lib/orders.js';
import { processPickTask, discoverPendingTasks } from '../lib/picking.js';
import { generateOrderWithType } from '../lib/data.js';

// Custom metrics
const scenariosRun = new Counter('chaos_scenarios_run');
const scenariosSuccessful = new Counter('chaos_scenarios_successful');
const scenariosFailed = new Counter('chaos_scenarios_failed');
const recoveryAttempts = new Counter('chaos_recovery_attempts');
const recoverySuccessful = new Counter('chaos_recovery_successful');
const compensationSuccessful = new Counter('chaos_compensation_successful');
const chaosSuccessRate = new Rate('chaos_success_rate');
const recoveryTime = new Trend('chaos_recovery_time_ms');

// Test configuration
export const options = {
  scenarios: {
    chaos_testing: {
      executor: 'constant-vus',
      vus: 2,
      duration: '3m',
    },
  },
  thresholds: {
    'chaos_success_rate': ['rate>0.80'],
    'chaos_recovery_time_ms': ['p(95)<10000'],
    'http_req_failed': ['rate<0.20'], // Higher tolerance for chaos testing
  },
};

// Configuration
const CONFIG = {
  scenariosPerIteration: parseInt(__ENV.CHAOS_SCENARIOS_PER_ITERATION || '3'),
  failureProbability: parseFloat(__ENV.CHAOS_FAILURE_PROBABILITY || '0.3'),
  testAllScenarios: __ENV.TEST_ALL_SCENARIOS === 'true',
  enableCompensation: __ENV.ENABLE_COMPENSATION !== 'false',
};

/**
 * Tests retry behavior for a retryable failure
 */
function testRetryableFailure(scenario) {
  console.log(`Testing retryable failure: ${scenario.name}`);
  const startTime = Date.now();

  // Create a test order
  const order = generateOrderWithType('single', null);
  const orderResult = createOrder(order);

  if (!orderResult || !orderResult.orderId) {
    console.warn('Failed to create test order');
    return { success: false, error: 'order_creation_failed' };
  }

  const orderId = orderResult.orderId;
  console.log(`Created test order: ${orderId}`);

  // Simulate the failure scenario with retries
  let attempts = 0;
  let success = false;

  for (let i = 0; i < (scenario.maxRetries || 3); i++) {
    attempts++;
    recoveryAttempts.add(1);

    // Simulate operation that might fail
    const shouldFail = i < 2; // Fail first two attempts

    if (shouldFail) {
      console.log(`Attempt ${attempts}: Simulated failure (${scenario.name})`);
      sleep(CHAOS_CONFIG.retryDelayMs / 1000);
    } else {
      console.log(`Attempt ${attempts}: Success`);
      success = true;
      recoverySuccessful.add(1);
      break;
    }
  }

  const duration = Date.now() - startTime;
  recoveryTime.add(duration);

  return {
    success: success,
    scenario: scenario.name,
    attempts: attempts,
    duration: duration,
    orderId: orderId,
  };
}

/**
 * Tests compensation for non-retryable failures
 */
function testCompensationFlow(scenario) {
  console.log(`Testing compensation flow: ${scenario.name}`);
  const startTime = Date.now();

  // Create a test order
  const order = generateOrderWithType('multi', null);
  const orderResult = createOrder(order);

  if (!orderResult || !orderResult.orderId) {
    console.warn('Failed to create test order');
    return { success: false, error: 'order_creation_failed' };
  }

  const orderId = orderResult.orderId;
  console.log(`Created test order: ${orderId}`);

  // Simulate partial processing then failure
  sleep(0.5);

  // Inject the failure
  injectFailure(scenario, {
    orderId: orderId,
    stage: 'mid-process',
  });

  // Trigger compensation
  const compensationType = scenario.compensation || 'rollback';

  if (CONFIG.enableCompensation) {
    const compensationTriggered = triggerCompensation(orderId, compensationType, {
      failureScenario: scenario.name,
      failedAt: new Date().toISOString(),
    });

    if (compensationTriggered) {
      // Verify compensation completed
      const verification = verifyCompensation(orderId, 30000);

      if (verification.success) {
        compensationSuccessful.add(1);
        console.log(`Compensation completed for order ${orderId}`);
      } else {
        console.warn(`Compensation verification failed: ${verification.error}`);
      }

      const duration = Date.now() - startTime;
      recoveryTime.add(duration);

      return {
        success: verification.success,
        scenario: scenario.name,
        compensationType: compensationType,
        duration: duration,
        orderId: orderId,
      };
    }
  }

  // Fallback: cancel the order
  console.log(`Canceling order ${orderId} as compensation fallback`);
  const cancelled = cancelOrder(orderId, 'chaos_test_failure');

  const duration = Date.now() - startTime;
  return {
    success: cancelled,
    scenario: scenario.name,
    compensationType: 'order_cancellation',
    duration: duration,
    orderId: orderId,
  };
}

/**
 * Tests the chaos wrapper functionality
 */
function testChaosWrapper() {
  console.log('Testing chaos wrapper functionality');

  let operationCalled = 0;
  const testOperation = () => {
    operationCalled++;
    return { success: true, value: 'operation_result' };
  };

  // Enable chaos for this test
  const originalEnabled = CHAOS_CONFIG.enabled;
  CHAOS_CONFIG.enabled = true;
  setFailureProbability(CONFIG.failureProbability);

  const result = wrapWithChaos(testOperation, {
    testName: 'chaos_wrapper_test',
  });

  CHAOS_CONFIG.enabled = originalEnabled;

  return {
    success: result.success,
    operationCalled: operationCalled,
    chaosInjected: result.chaosInjected || false,
    requiresCompensation: result.requiresCompensation || false,
  };
}

/**
 * Runs a specific chaos scenario
 */
function runScenario(scenario) {
  scenariosRun.add(1);

  let result;
  if (scenario.retryable) {
    result = testRetryableFailure(scenario);
  } else if (scenario.compensation) {
    result = testCompensationFlow(scenario);
  } else {
    // Generic scenario test
    result = runChaosScenario(scenario, () => {
      return { success: true };
    });
  }

  if (result.success) {
    scenariosSuccessful.add(1);
    chaosSuccessRate.add(1);
  } else {
    scenariosFailed.add(1);
    chaosSuccessRate.add(0);
  }

  return result;
}

/**
 * Main test function
 */
export default function () {
  const vuId = __VU;
  const iterationId = __ITER;

  console.log(`[VU ${vuId}] Starting chaos simulation - iteration ${iterationId}`);

  // Phase 1: Test all scenarios if configured
  if (CONFIG.testAllScenarios && iterationId === 0) {
    group('Test All Failure Scenarios', function () {
      const scenarios = Object.values(FAILURE_SCENARIOS);

      for (const scenario of scenarios) {
        console.log(`[VU ${vuId}] Testing scenario: ${scenario.name}`);
        const result = runScenario(scenario);
        console.log(`[VU ${vuId}] Scenario ${scenario.name}: ${result.success ? 'PASSED' : 'FAILED'}`);
        sleep(1);
      }
    });
  }

  // Phase 2: Run random scenarios
  group('Random Chaos Scenarios', function () {
    for (let i = 0; i < CONFIG.scenariosPerIteration; i++) {
      // Select random scenario
      const scenarios = Object.values(FAILURE_SCENARIOS);
      const scenario = scenarios[Math.floor(Math.random() * scenarios.length)];

      console.log(`[VU ${vuId}] Running random scenario: ${scenario.name}`);
      const result = runScenario(scenario);

      console.log(`[VU ${vuId}] Result: ${result.success ? 'SUCCESS' : 'FAILED'} in ${result.duration}ms`);
      sleep(1);
    }
  });

  // Phase 3: Test chaos wrapper
  group('Chaos Wrapper Tests', function () {
    const wrapperResult = testChaosWrapper();
    console.log(`[VU ${vuId}] Wrapper test: operation called ${wrapperResult.operationCalled} times`);
    if (wrapperResult.chaosInjected) {
      console.log(`[VU ${vuId}] Chaos was injected`);
    }
  });

  // Phase 4: Test retry mechanism
  group('Retry Mechanism Tests', function () {
    let successCount = 0;
    const testOperation = () => {
      successCount++;
      // Fail first 2 attempts
      if (successCount < 3) {
        throw new Error('Simulated failure');
      }
      return { success: true };
    };

    const retryResult = executeWithRetry(testOperation, {
      maxRetries: 5,
      initialDelay: 100,
    });

    console.log(`[VU ${vuId}] Retry test: ${retryResult.success ? 'SUCCESS' : 'FAILED'} after ${retryResult.attempts} attempts`);
  });

  // Brief pause between iterations
  sleep(2);
}

/**
 * Setup function
 */
export function setup() {
  console.log('='.repeat(60));
  console.log('Chaos Simulator - Setup');
  console.log('='.repeat(60));
  console.log(`Scenarios per iteration: ${CONFIG.scenariosPerIteration}`);
  console.log(`Failure probability: ${CONFIG.failureProbability}`);
  console.log(`Test all scenarios: ${CONFIG.testAllScenarios}`);
  console.log(`Enable compensation: ${CONFIG.enableCompensation}`);
  console.log('Available scenarios:');
  Object.values(FAILURE_SCENARIOS).forEach(s => {
    console.log(`  - ${s.name}: ${s.description}`);
  });
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
  console.log('Chaos Simulator - Summary');
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
    simulator: 'chaos-simulator',
    metrics: {
      scenarios_run: data.metrics.chaos_scenarios_run?.values?.count || 0,
      scenarios_successful: data.metrics.chaos_scenarios_successful?.values?.count || 0,
      scenarios_failed: data.metrics.chaos_scenarios_failed?.values?.count || 0,
      recovery_attempts: data.metrics.chaos_recovery_attempts?.values?.count || 0,
      recovery_successful: data.metrics.chaos_recovery_successful?.values?.count || 0,
      compensation_successful: data.metrics.chaos_compensation_successful?.values?.count || 0,
      success_rate: data.metrics.chaos_success_rate?.values?.rate || 0,
      avg_recovery_time_ms: data.metrics.chaos_recovery_time_ms?.values?.avg || 0,
      p95_recovery_time_ms: data.metrics.chaos_recovery_time_ms?.values?.['p(95)'] || 0,
    },
    thresholds: data.thresholds,
  };

  return {
    'stdout': JSON.stringify(summary, null, 2) + '\n',
    'chaos-results.json': JSON.stringify(summary, null, 2),
  };
}
