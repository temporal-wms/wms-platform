// K6 Chaos Engineering Helper Library
// Provides functions for failure injection, error recovery testing, and chaos scenarios

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import { BASE_URLS, ENDPOINTS, HTTP_PARAMS } from './config.js';

// Custom metrics for chaos testing
export const chaosMetrics = {
  failuresInjected: new Counter('chaos_failures_injected'),
  retriesAttempted: new Counter('chaos_retries_attempted'),
  recoverySuccesses: new Counter('chaos_recovery_successes'),
  recoveryFailures: new Counter('chaos_recovery_failures'),
  compensationTriggered: new Counter('chaos_compensation_triggered'),
  retryLatency: new Trend('chaos_retry_latency_ms'),
};

// Chaos configuration
export const CHAOS_CONFIG = {
  enabled: __ENV.ENABLE_CHAOS === 'true',
  failureProbability: parseFloat(__ENV.CHAOS_FAILURE_PROBABILITY || '0.05'),
  maxRetries: parseInt(__ENV.CHAOS_MAX_RETRIES || '3'),
  retryDelayMs: parseInt(__ENV.CHAOS_RETRY_DELAY_MS || '1000'),
  retryBackoffMultiplier: parseFloat(__ENV.CHAOS_BACKOFF_MULTIPLIER || '2.0'),
  maxRetryDelayMs: parseInt(__ENV.CHAOS_MAX_RETRY_DELAY_MS || '10000'),
};

// Failure scenarios with their probabilities and behaviors
export const FAILURE_SCENARIOS = {
  TASK_FAILURE: {
    name: 'task_failure',
    probability: 0.05,
    retryable: true,
    maxRetries: 3,
    description: 'Task execution fails mid-process',
  },
  INVENTORY_SHORT: {
    name: 'inventory_short',
    probability: 0.03,
    retryable: false,
    compensation: 'reallocate',
    description: 'Inventory not available at expected location',
  },
  SIGNAL_TIMEOUT: {
    name: 'signal_timeout',
    probability: 0.02,
    retryable: true,
    maxRetries: 3,
    description: 'Signal to orchestrator times out',
  },
  SERVICE_UNAVAILABLE: {
    name: 'service_unavailable',
    probability: 0.01,
    retryable: true,
    fallback: 'circuit_breaker',
    description: 'Service temporarily unavailable',
  },
  PARTIAL_COMPLETION: {
    name: 'partial_completion',
    probability: 0.02,
    retryable: false,
    compensation: 'rollback',
    description: 'Task partially completed before failure',
  },
  DATA_CORRUPTION: {
    name: 'data_corruption',
    probability: 0.005,
    retryable: false,
    compensation: 'manual_intervention',
    description: 'Data integrity issue detected',
  },
  NETWORK_PARTITION: {
    name: 'network_partition',
    probability: 0.01,
    retryable: true,
    maxRetries: 5,
    description: 'Network partition between services',
  },
};

/**
 * Determines if a failure should be injected based on probability
 * @param {number} probability - Failure probability (0-1)
 * @returns {boolean} True if failure should be injected
 */
export function shouldInjectFailure(probability = CHAOS_CONFIG.failureProbability) {
  if (!CHAOS_CONFIG.enabled) {
    return false;
  }
  return Math.random() < probability;
}

/**
 * Selects a random failure scenario based on weighted probabilities
 * @returns {Object|null} Selected failure scenario or null
 */
export function selectRandomFailure() {
  if (!CHAOS_CONFIG.enabled) {
    return null;
  }

  const scenarios = Object.values(FAILURE_SCENARIOS);
  const totalProbability = scenarios.reduce((sum, s) => sum + s.probability, 0);

  let random = Math.random() * totalProbability;
  for (const scenario of scenarios) {
    random -= scenario.probability;
    if (random <= 0) {
      return scenario;
    }
  }

  return null;
}

/**
 * Injects a failure and records metrics
 * @param {Object} scenario - The failure scenario to inject
 * @param {Object} context - Context about the operation
 * @returns {Object} Failure details
 */
export function injectFailure(scenario, context = {}) {
  chaosMetrics.failuresInjected.add(1);

  const failure = {
    scenario: scenario.name,
    description: scenario.description,
    injectedAt: new Date().toISOString(),
    context: context,
    retryable: scenario.retryable,
    compensation: scenario.compensation || null,
  };

  console.log(`CHAOS: Injected failure - ${scenario.name}: ${scenario.description}`);
  console.log(`CHAOS: Context - ${JSON.stringify(context)}`);

  return failure;
}

/**
 * Executes an operation with retry logic
 * @param {Function} operation - The operation to execute
 * @param {Object} options - Retry options
 * @returns {Object} Result with success status and attempts
 */
export function executeWithRetry(operation, options = {}) {
  const maxRetries = options.maxRetries || CHAOS_CONFIG.maxRetries;
  let delay = options.initialDelay || CHAOS_CONFIG.retryDelayMs;
  const backoffMultiplier = options.backoffMultiplier || CHAOS_CONFIG.retryBackoffMultiplier;
  const maxDelay = options.maxDelay || CHAOS_CONFIG.maxRetryDelayMs;

  let lastError = null;
  let attempts = 0;

  for (let attempt = 1; attempt <= maxRetries + 1; attempt++) {
    attempts = attempt;

    try {
      const startTime = Date.now();
      const result = operation();

      if (result && (result.success || result.status === 200 || result.status === 201)) {
        if (attempt > 1) {
          console.log(`CHAOS: Operation succeeded on attempt ${attempt}/${maxRetries + 1}`);
          chaosMetrics.recoverySuccesses.add(1);
        }
        return {
          success: true,
          result: result,
          attempts: attempts,
          retriedCount: attempts - 1,
        };
      }

      lastError = result?.error || 'Operation returned failure status';
    } catch (e) {
      lastError = e.message || 'Unknown error';
    }

    if (attempt <= maxRetries) {
      chaosMetrics.retriesAttempted.add(1);
      console.log(`CHAOS: Attempt ${attempt}/${maxRetries + 1} failed, retrying in ${delay}ms...`);

      const retryStart = Date.now();
      sleep(delay / 1000);
      chaosMetrics.retryLatency.add(Date.now() - retryStart);

      // Apply exponential backoff
      delay = Math.min(delay * backoffMultiplier, maxDelay);
    }
  }

  console.log(`CHAOS: All ${maxRetries + 1} attempts failed`);
  chaosMetrics.recoveryFailures.add(1);

  return {
    success: false,
    error: lastError,
    attempts: attempts,
    retriedCount: attempts - 1,
  };
}

/**
 * Wraps an operation with chaos injection
 * @param {Function} operation - The operation to wrap
 * @param {Object} context - Context for failure tracking
 * @returns {Object} Result with potential failure injection
 */
export function wrapWithChaos(operation, context = {}) {
  // Check if we should inject a failure
  const failureScenario = selectRandomFailure();

  if (failureScenario && shouldInjectFailure(failureScenario.probability)) {
    const failure = injectFailure(failureScenario, context);

    // If retryable, attempt with retries
    if (failureScenario.retryable) {
      return executeWithRetry(operation, {
        maxRetries: failureScenario.maxRetries || CHAOS_CONFIG.maxRetries,
      });
    }

    // Non-retryable failure
    return {
      success: false,
      failure: failure,
      requiresCompensation: !!failureScenario.compensation,
      compensationType: failureScenario.compensation,
    };
  }

  // No chaos injection, execute normally
  try {
    const result = operation();
    return {
      success: true,
      result: result,
      chaosInjected: false,
    };
  } catch (e) {
    return {
      success: false,
      error: e.message,
      chaosInjected: false,
    };
  }
}

/**
 * Triggers compensation workflow for an order
 * @param {string} orderId - The order ID
 * @param {string} compensationType - Type of compensation needed
 * @param {Object} context - Failure context
 * @returns {boolean} True if compensation triggered successfully
 */
export function triggerCompensation(orderId, compensationType, context = {}) {
  chaosMetrics.compensationTriggered.add(1);

  console.log(`CHAOS: Triggering compensation '${compensationType}' for order ${orderId}`);

  const url = `${BASE_URLS.orchestrator}/api/v1/compensation`;
  const payload = JSON.stringify({
    orderId: orderId,
    compensationType: compensationType,
    context: context,
    triggeredAt: new Date().toISOString(),
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'trigger compensation status 200': (r) => r.status === 200 || r.status === 202,
  });

  if (!success) {
    console.warn(`CHAOS: Failed to trigger compensation for ${orderId}: ${response.status}`);
  } else {
    console.log(`CHAOS: Compensation triggered successfully for ${orderId}`);
  }

  return success;
}

/**
 * Verifies that compensation completed successfully
 * @param {string} orderId - The order ID
 * @param {number} timeoutMs - Timeout for verification
 * @returns {Object} Verification result
 */
export function verifyCompensation(orderId, timeoutMs = 30000) {
  const startTime = Date.now();
  const pollIntervalMs = 2000;

  while (Date.now() - startTime < timeoutMs) {
    const url = `${BASE_URLS.orchestrator}/api/v1/compensation/order/${orderId}`;
    const response = http.get(url, HTTP_PARAMS);

    if (response.status === 200) {
      try {
        const data = JSON.parse(response.body);
        if (data.status === 'completed') {
          console.log(`CHAOS: Compensation verified for order ${orderId}`);
          return {
            success: true,
            status: data.status,
            completedAt: data.completedAt,
          };
        }
        if (data.status === 'failed') {
          console.warn(`CHAOS: Compensation failed for order ${orderId}`);
          return {
            success: false,
            status: data.status,
            error: data.error,
          };
        }
      } catch (e) {
        // Continue polling
      }
    }

    sleep(pollIntervalMs / 1000);
  }

  console.warn(`CHAOS: Compensation verification timed out for order ${orderId}`);
  return {
    success: false,
    status: 'timeout',
    error: 'Verification timed out',
  };
}

/**
 * Simulates a network partition between services
 * @param {string} sourceService - Source service
 * @param {string} targetService - Target service
 * @param {number} durationMs - Partition duration
 * @returns {Object} Partition simulation result
 */
export function simulateNetworkPartition(sourceService, targetService, durationMs = 5000) {
  console.log(`CHAOS: Simulating network partition ${sourceService} -> ${targetService} for ${durationMs}ms`);

  // In a real implementation, this would interact with a chaos proxy or service mesh
  // For simulation purposes, we just track the partition
  const partition = {
    source: sourceService,
    target: targetService,
    startedAt: new Date().toISOString(),
    durationMs: durationMs,
  };

  sleep(durationMs / 1000);

  partition.endedAt = new Date().toISOString();
  console.log(`CHAOS: Network partition ended ${sourceService} -> ${targetService}`);

  return partition;
}

/**
 * Simulates a service outage
 * @param {string} serviceName - The service to simulate as down
 * @param {number} durationMs - Outage duration
 * @returns {Object} Outage simulation result
 */
export function simulateServiceOutage(serviceName, durationMs = 10000) {
  console.log(`CHAOS: Simulating service outage for ${serviceName} for ${durationMs}ms`);

  const outage = {
    service: serviceName,
    startedAt: new Date().toISOString(),
    durationMs: durationMs,
  };

  sleep(durationMs / 1000);

  outage.endedAt = new Date().toISOString();
  console.log(`CHAOS: Service outage ended for ${serviceName}`);

  return outage;
}

/**
 * Runs a chaos scenario with controlled failure injection
 * @param {Object} scenario - The chaos scenario configuration
 * @param {Function} operation - The operation to run
 * @returns {Object} Scenario execution results
 */
export function runChaosScenario(scenario, operation) {
  const results = {
    scenario: scenario.name,
    startedAt: new Date().toISOString(),
    failuresInjected: 0,
    retriesAttempted: 0,
    compensationsTriggered: 0,
    success: false,
  };

  console.log(`CHAOS: Starting scenario '${scenario.name}'`);

  try {
    // Force failure injection for this scenario
    const originalProbability = CHAOS_CONFIG.failureProbability;
    CHAOS_CONFIG.failureProbability = 1.0; // Force failure

    const injectedFailure = injectFailure(scenario, { scenario: scenario.name });
    results.failuresInjected++;

    CHAOS_CONFIG.failureProbability = originalProbability;

    // Execute with retry if applicable
    if (scenario.retryable) {
      const retryResult = executeWithRetry(operation, {
        maxRetries: scenario.maxRetries || 3,
      });
      results.retriesAttempted = retryResult.retriedCount;
      results.success = retryResult.success;
    } else if (scenario.compensation) {
      results.compensationsTriggered++;
      results.requiresCompensation = true;
      results.compensationType = scenario.compensation;
    }

  } catch (e) {
    results.error = e.message;
  }

  results.endedAt = new Date().toISOString();
  console.log(`CHAOS: Scenario '${scenario.name}' completed - success: ${results.success}`);

  return results;
}

/**
 * Gets a summary of chaos metrics
 * @returns {Object} Metrics summary
 */
export function getChaosMetricsSummary() {
  return {
    enabled: CHAOS_CONFIG.enabled,
    failureProbability: CHAOS_CONFIG.failureProbability,
    // Note: In K6, metric values are not directly accessible during execution
    // This returns the configuration for reference
    config: CHAOS_CONFIG,
  };
}

/**
 * Enables or disables chaos injection
 * @param {boolean} enabled - Whether to enable chaos
 */
export function setChaosEnabled(enabled) {
  CHAOS_CONFIG.enabled = enabled;
  console.log(`CHAOS: ${enabled ? 'Enabled' : 'Disabled'} chaos injection`);
}

/**
 * Sets the failure probability
 * @param {number} probability - Probability (0-1)
 */
export function setFailureProbability(probability) {
  CHAOS_CONFIG.failureProbability = Math.max(0, Math.min(1, probability));
  console.log(`CHAOS: Set failure probability to ${CHAOS_CONFIG.failureProbability}`);
}
