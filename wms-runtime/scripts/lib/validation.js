// K6 Validation Library for WMS Platform
// Integrates with validation-service and temporal-validator

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Configuration
const VALIDATION_SERVICE_URL = __ENV.VALIDATION_SERVICE_URL || 'http://localhost:8080';
const TEMPORAL_VALIDATOR_URL = __ENV.TEMPORAL_VALIDATOR_URL || 'http://localhost:9090';
const VALIDATION_ENABLED = __ENV.ENABLE_EVENT_VALIDATION !== 'false';
const SIGNAL_VALIDATION_ENABLED = __ENV.ENABLE_SIGNAL_VALIDATION !== 'false';

// Custom metrics
const eventValidationRate = new Rate('event_validation_success');
const signalDeliveryRate = new Rate('signal_delivery_success');

// HTTP parameters
const HTTP_PARAMS = {
  headers: {
    'Content-Type': 'application/json',
  },
  timeout: '10s',
};

// =============================================================================
// Event Validation Functions
// =============================================================================

/**
 * Start tracking events for an order
 * @param {string} orderId - The order ID to track
 * @returns {boolean} - Success status
 */
export function startEventTracking(orderId) {
  if (!VALIDATION_ENABLED) {
    return true;
  }

  const url = `${VALIDATION_SERVICE_URL}/api/v1/validation/start-tracking/${orderId}`;
  const response = http.post(url, null, HTTP_PARAMS);

  const success = check(response, {
    'start tracking success': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to start tracking for order ${orderId}: ${response.status}`);
  }

  return success;
}

/**
 * Assert that expected events were received for an order
 * @param {string} orderId - The order ID
 * @param {string[]} expectedEventTypes - Array of expected event types
 * @param {number} retries - Number of retries (default: 3)
 * @param {number} retryDelay - Delay between retries in seconds (default: 2)
 * @returns {boolean} - True if all expected events were received
 */
export function assertEventsReceived(orderId, expectedEventTypes, retries = 3, retryDelay = 2) {
  if (!VALIDATION_ENABLED) {
    return true;
  }

  const url = `${VALIDATION_SERVICE_URL}/api/v1/validation/assert/${orderId}`;
  const payload = JSON.stringify({
    expectedTypes: expectedEventTypes,
  });

  for (let attempt = 0; attempt <= retries; attempt++) {
    const response = http.post(url, payload, HTTP_PARAMS);

    if (response.status === 200) {
      const result = JSON.parse(response.body);

      if (result.success) {
        eventValidationRate.add(true);
        return true;
      }

      // If not successful and we have retries left, wait and retry
      if (attempt < retries) {
        console.log(
          `Attempt ${attempt + 1}/${retries + 1}: Missing events for ${orderId}: ${result.missingEvents.join(', ')}`
        );
        sleep(retryDelay);
        continue;
      }

      // Final attempt failed
      console.error(
        `Event validation failed for ${orderId}. Missing: ${result.missingEvents.join(', ')}`
      );
      eventValidationRate.add(false);
      return false;
    }

    if (attempt < retries) {
      sleep(retryDelay);
    }
  }

  eventValidationRate.add(false);
  return false;
}

/**
 * Validate event sequence for a flow type
 * @param {string} orderId - The order ID
 * @param {string} flowType - The flow type (e.g., 'standard_flow', 'multi_item_flow')
 * @returns {object|null} - Validation result or null on failure
 */
export function validateEventSequence(orderId, flowType) {
  if (!VALIDATION_ENABLED) {
    return { isValid: true };
  }

  const url = `${VALIDATION_SERVICE_URL}/api/v1/validation/sequence/${orderId}`;
  const payload = JSON.stringify({ flowType });

  const response = http.post(url, payload, HTTP_PARAMS);

  if (response.status === 200) {
    return JSON.parse(response.body);
  }

  console.error(`Failed to validate sequence for ${orderId}: ${response.status}`);
  return null;
}

/**
 * Get validation report for an order
 * @param {string} orderId - The order ID
 * @returns {object|null} - Validation report or null on failure
 */
export function getEventValidationReport(orderId) {
  if (!VALIDATION_ENABLED) {
    return null;
  }

  const url = `${VALIDATION_SERVICE_URL}/api/v1/validation/report/${orderId}`;
  const response = http.get(url, HTTP_PARAMS);

  if (response.status === 200) {
    return JSON.parse(response.body);
  }

  return null;
}

/**
 * Get captured events for an order
 * @param {string} orderId - The order ID
 * @returns {array} - Array of captured events
 */
export function getCapturedEvents(orderId) {
  if (!VALIDATION_ENABLED) {
    return [];
  }

  const url = `${VALIDATION_SERVICE_URL}/api/v1/validation/events/${orderId}`;
  const response = http.get(url, HTTP_PARAMS);

  if (response.status === 200) {
    const result = JSON.parse(response.body);
    return result.events || [];
  }

  return [];
}

/**
 * Clear tracking data for an order
 * @param {string} orderId - The order ID
 */
export function clearEventTracking(orderId) {
  if (!VALIDATION_ENABLED) {
    return;
  }

  const url = `${VALIDATION_SERVICE_URL}/api/v1/validation/clear/${orderId}`;
  http.del(url, null, HTTP_PARAMS);
}

// =============================================================================
// Signal Validation Functions
// =============================================================================

/**
 * Validate that a signal was delivered to a workflow
 * @param {string} workflowId - The workflow ID
 * @param {string} signalName - The signal name to validate
 * @param {number} retries - Number of retries (default: 3)
 * @param {number} retryDelay - Delay between retries in seconds (default: 2)
 * @returns {boolean} - True if signal was delivered
 */
export function validateSignalDelivered(workflowId, signalName, retries = 3, retryDelay = 2) {
  if (!SIGNAL_VALIDATION_ENABLED) {
    return true;
  }

  const url = `${TEMPORAL_VALIDATOR_URL}/api/v1/workflow/assert-signal/${workflowId}`;
  const payload = JSON.stringify({ signalName });

  for (let attempt = 0; attempt <= retries; attempt++) {
    const response = http.post(url, payload, HTTP_PARAMS);

    if (response.status === 200) {
      const result = JSON.parse(response.body);

      if (result.delivered) {
        signalDeliveryRate.add(true);
        return true;
      }

      // If not delivered and we have retries left, wait and retry
      if (attempt < retries) {
        console.log(
          `Attempt ${attempt + 1}/${retries + 1}: Signal '${signalName}' not yet delivered to ${workflowId}`
        );
        sleep(retryDelay);
        continue;
      }

      // Final attempt failed
      console.error(`Signal '${signalName}' was not delivered to workflow ${workflowId}`);
      signalDeliveryRate.add(false);
      return false;
    }

    if (attempt < retries) {
      sleep(retryDelay);
    }
  }

  signalDeliveryRate.add(false);
  return false;
}

/**
 * Get workflow status
 * @param {string} workflowId - The workflow ID
 * @returns {object|null} - Workflow status or null on failure
 */
export function getWorkflowStatus(workflowId) {
  if (!SIGNAL_VALIDATION_ENABLED) {
    return null;
  }

  const url = `${TEMPORAL_VALIDATOR_URL}/api/v1/workflow/status/${workflowId}`;
  const response = http.get(url, HTTP_PARAMS);

  if (response.status === 200) {
    return JSON.parse(response.body);
  }

  return null;
}

/**
 * Assert workflow is in expected state
 * @param {string} workflowId - The workflow ID
 * @param {string} expectedStatus - The expected status (e.g., 'Running', 'Completed')
 * @returns {boolean} - True if workflow is in expected state
 */
export function assertWorkflowState(workflowId, expectedStatus) {
  if (!SIGNAL_VALIDATION_ENABLED) {
    return true;
  }

  const status = getWorkflowStatus(workflowId);

  if (!status) {
    console.error(`Failed to get status for workflow ${workflowId}`);
    return false;
  }

  const success = status.status === expectedStatus;

  if (!success) {
    console.error(
      `Workflow ${workflowId} is in state '${status.status}', expected '${expectedStatus}'`
    );
  }

  return success;
}

/**
 * Get all signals received by a workflow
 * @param {string} workflowId - The workflow ID
 * @returns {array} - Array of signal events
 */
export function getWorkflowSignals(workflowId) {
  if (!SIGNAL_VALIDATION_ENABLED) {
    return [];
  }

  const url = `${TEMPORAL_VALIDATOR_URL}/api/v1/signal/list/${workflowId}`;
  const response = http.get(url, HTTP_PARAMS);

  if (response.status === 200) {
    const result = JSON.parse(response.body);
    return result.signals || [];
  }

  return [];
}

/**
 * Get workflow execution progress
 * @param {string} workflowId - The workflow ID
 * @returns {object|null} - Workflow progress information
 */
export function getWorkflowProgress(workflowId) {
  if (!SIGNAL_VALIDATION_ENABLED) {
    return null;
  }

  const url = `${TEMPORAL_VALIDATOR_URL}/api/v1/workflow/describe/${workflowId}`;
  const response = http.get(url, HTTP_PARAMS);

  if (response.status === 200) {
    return JSON.parse(response.body);
  }

  return null;
}

// =============================================================================
// Combined Validation Functions
// =============================================================================

/**
 * Comprehensive validation for a completed order
 * @param {string} orderId - The order ID
 * @param {string} flowType - The flow type
 * @param {string[]} expectedEvents - Expected event types
 * @returns {object} - Validation results
 */
export function validateOrderCompletion(orderId, flowType, expectedEvents) {
  const results = {
    orderId,
    eventsValid: false,
    sequenceValid: false,
    overallValid: false,
    report: null,
  };

  // Validate events were received
  results.eventsValid = assertEventsReceived(orderId, expectedEvents);

  // Validate sequence
  const sequenceResult = validateEventSequence(orderId, flowType);
  results.sequenceValid = sequenceResult ? sequenceResult.isValid : false;

  // Get detailed report
  results.report = getEventValidationReport(orderId);

  // Overall validation
  results.overallValid = results.eventsValid && results.sequenceValid;

  return results;
}

/**
 * Validate workflow progression through all stages
 * @param {string} orderId - The order ID
 * @param {array} stages - Array of stage objects with {workflowId, signalName}
 * @returns {object} - Validation results
 */
export function validateWorkflowProgression(orderId, stages) {
  const results = {
    orderId,
    stagesCompleted: 0,
    stagesFailed: 0,
    details: [],
  };

  for (const stage of stages) {
    const delivered = validateSignalDelivered(stage.workflowId, stage.signalName);

    results.details.push({
      workflowId: stage.workflowId,
      signalName: stage.signalName,
      delivered,
    });

    if (delivered) {
      results.stagesCompleted++;
    } else {
      results.stagesFailed++;
    }
  }

  return results;
}

// Export configuration for external access
export const validationConfig = {
  validationServiceUrl: VALIDATION_SERVICE_URL,
  temporalValidatorUrl: TEMPORAL_VALIDATOR_URL,
  eventValidationEnabled: VALIDATION_ENABLED,
  signalValidationEnabled: SIGNAL_VALIDATION_ENABLED,
};
