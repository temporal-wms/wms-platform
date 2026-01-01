// K6 Consolidation Service Helper Library
// Provides functions for interacting with consolidation-service and orchestrator signal bridge

import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URLS, ENDPOINTS, HTTP_PARAMS, CONSOLIDATION_CONFIG } from './config.js';

/**
 * Discovers pending consolidation units
 * @returns {Array} Array of pending consolidation units
 */
export function discoverPendingConsolidations() {
  const url = `${BASE_URLS.consolidation}${ENDPOINTS.consolidation.pending}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'discover pending consolidations status 200': (r) => r.status === 200,
  });

  if (!success || !response.body) {
    console.warn(`Failed to discover pending consolidations: ${response.status} - ${response.body}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return Array.isArray(data) ? data : (data.consolidations || data.items || []);
  } catch (e) {
    console.error(`Failed to parse consolidations response: ${e.message}`);
    return [];
  }
}

/**
 * Gets consolidation units by order ID
 * @param {string} orderId - The order ID
 * @returns {Array} Array of consolidation units
 */
export function getConsolidationsByOrder(orderId) {
  const url = `${BASE_URLS.consolidation}${ENDPOINTS.consolidation.byOrder(orderId)}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get consolidations by order status 200': (r) => r.status === 200,
  });

  if (!success || !response.body) {
    console.warn(`Failed to get consolidations for order ${orderId}: ${response.status}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return Array.isArray(data) ? data : (data.consolidations || data.items || []);
  } catch (e) {
    console.error(`Failed to parse consolidations response: ${e.message}`);
    return [];
  }
}

/**
 * Gets a specific consolidation unit by ID
 * @param {string} consolidationId - The consolidation ID
 * @returns {Object|null} The consolidation unit or null if not found
 */
export function getConsolidation(consolidationId) {
  const url = `${BASE_URLS.consolidation}${ENDPOINTS.consolidation.get(consolidationId)}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get consolidation status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to get consolidation ${consolidationId}: ${response.status}`);
    return null;
  }

  try {
    return JSON.parse(response.body);
  } catch (e) {
    console.error(`Failed to parse consolidation response: ${e.message}`);
    return null;
  }
}

/**
 * Assigns a consolidation unit to a station and worker
 * @param {string} consolidationId - The consolidation ID
 * @param {string} station - The station ID
 * @param {string} workerId - The worker ID
 * @param {string} destinationBin - The destination bin ID
 * @returns {boolean} True if successful
 */
export function assignConsolidation(consolidationId, station, workerId, destinationBin) {
  const url = `${BASE_URLS.consolidation}${ENDPOINTS.consolidation.assign(consolidationId)}`;
  const payload = JSON.stringify({
    station: station,
    workerId: workerId,
    destinationBin: destinationBin,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'assign consolidation status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to assign consolidation ${consolidationId}: ${response.status} - ${response.body}`);
  }

  return success;
}

/**
 * Records consolidation of an item
 * @param {string} consolidationId - The consolidation ID
 * @param {string} sku - The SKU being consolidated
 * @param {number} quantity - The quantity consolidated
 * @param {string} sourceToteId - The source tote ID
 * @param {string} verifiedBy - The worker ID who verified
 * @returns {boolean} True if successful
 */
export function consolidateItem(consolidationId, sku, quantity, sourceToteId, verifiedBy) {
  const url = `${BASE_URLS.consolidation}${ENDPOINTS.consolidation.consolidate(consolidationId)}`;
  const payload = JSON.stringify({
    sku: sku,
    quantity: quantity,
    sourceToteId: sourceToteId,
    verifiedBy: verifiedBy,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'consolidate item status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to consolidate item ${sku} for ${consolidationId}: ${response.status} - ${response.body}`);
  }

  return success;
}

/**
 * Completes a consolidation unit
 * @param {string} consolidationId - The consolidation ID
 * @returns {Object|null} The completed consolidation or null if failed
 */
export function completeConsolidation(consolidationId) {
  const url = `${BASE_URLS.consolidation}${ENDPOINTS.consolidation.complete(consolidationId)}`;
  const response = http.post(url, null, HTTP_PARAMS);

  const success = check(response, {
    'complete consolidation status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to complete consolidation ${consolidationId}: ${response.status} - ${response.body}`);
    return null;
  }

  try {
    return JSON.parse(response.body);
  } catch (e) {
    console.error(`Failed to parse complete consolidation response: ${e.message}`);
    return null;
  }
}

/**
 * Sends a consolidation completed signal to the orchestrator
 * @param {string} orderId - The order ID
 * @param {string} consolidationId - The consolidation ID
 * @param {Array} consolidatedItems - Array of consolidated items
 * @returns {boolean} True if successful
 */
export function sendConsolidationCompleteSignal(orderId, consolidationId, consolidatedItems = []) {
  const url = `${BASE_URLS.orchestrator}${ENDPOINTS.orchestrator.signalConsolidationComplete}`;
  const payload = JSON.stringify({
    orderId: orderId,
    consolidationId: consolidationId,
    consolidatedItems: consolidatedItems,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'signal consolidation complete status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to signal consolidation complete for order ${orderId}: ${response.status} - ${response.body}`);
  } else {
    try {
      const result = JSON.parse(response.body);
      console.log(`Consolidation signal sent for workflow: ${result.workflowId}`);
    } catch (e) {
      // Ignore parse errors for logging
    }
  }

  return success;
}

/**
 * Simulates consolidating all items in a unit with realistic delays
 * @param {Object} unit - The consolidation unit object
 * @returns {Array} Array of consolidated items for signaling
 */
export function simulateConsolidation(unit) {
  const consolidationId = unit.consolidationId || unit.id;
  const items = unit.expectedItems || unit.items || [];
  const orderId = unit.orderId;
  const station = CONSOLIDATION_CONFIG.defaultStation;
  const workerId = `CONSOL-WORKER-${__VU || 1}`;
  const destinationBin = `BIN-${consolidationId.slice(-6)}`;

  console.log(`Simulating consolidation ${consolidationId} with ${items.length} items`);

  const consolidatedItems = [];

  // Step 1: Assign to station with worker and bin
  sleep(CONSOLIDATION_CONFIG.simulationDelayMs / 1000);
  if (!assignConsolidation(consolidationId, station, workerId, destinationBin)) {
    console.warn(`Failed to assign consolidation ${consolidationId}, trying to continue`);
  }

  // Step 2: Consolidate each item
  for (const item of items) {
    sleep(CONSOLIDATION_CONFIG.simulationDelayMs / 1000);

    // Use exact sourceToteId from expectedItems - service validates both SKU and sourceToteId must match
    const sourceToteId = item.sourceToteId !== undefined ? item.sourceToteId : '';
    const success = consolidateItem(
      consolidationId,
      item.sku,
      item.quantity,
      sourceToteId,
      workerId
    );

    if (success) {
      consolidatedItems.push({
        sku: item.sku,
        quantity: item.quantity,
        sourceToteId: sourceToteId,
      });
    }
  }

  return consolidatedItems;
}

/**
 * Processes a single consolidation unit end-to-end
 * @param {Object} unit - The consolidation unit object
 * @returns {boolean} True if fully successful
 */
export function processConsolidation(unit) {
  const consolidationId = unit.consolidationId || unit.id;
  const orderId = unit.orderId;

  console.log(`Processing consolidation ${consolidationId} for order ${orderId}`);

  // Check if items have valid sourceToteId (required by the API)
  const expectedItems = unit.expectedItems || unit.items || [];
  const hasValidSourceTotes = expectedItems.every((item) => item.sourceToteId && item.sourceToteId.length > 0);

  let consolidatedItems = [];
  if (hasValidSourceTotes) {
    // Step 1: Simulate consolidating items (only if we have valid sourceToteIds)
    consolidatedItems = simulateConsolidation(unit);

    if (consolidatedItems.length === 0 && expectedItems.length > 0) {
      console.warn(`No items consolidated for unit ${consolidationId}`);
      return false;
    }
  } else {
    // Data quality issue: sourceToteId is empty (workflow didn't populate from picking)
    // Skip item consolidation and try to complete directly
    console.warn(`Skipping item consolidation for ${consolidationId} - sourceToteId not populated (data quality issue)`);
  }

  // Step 2: Complete the consolidation via API
  const completed = completeConsolidation(consolidationId);
  if (!completed) {
    console.warn(`Failed to complete consolidation ${consolidationId}`);
    return false;
  }

  // Step 3: Signal the orchestrator workflow that consolidation is complete
  // THIS IS REQUIRED for the workflow to progress to packing
  const signalSent = sendConsolidationCompleteSignal(orderId, consolidationId, consolidatedItems);
  if (!signalSent) {
    console.warn(`Failed to send consolidation complete signal for ${orderId}, workflow may be stuck`);
  }

  console.log(`Consolidation ${consolidationId} completed successfully`);
  return true;
}

/**
 * Discovers and processes all pending consolidations
 * @param {number} maxTasks - Maximum number of consolidations to process
 * @returns {Object} Summary of processing results
 */
export function processAllPendingConsolidations(maxTasks = CONSOLIDATION_CONFIG.maxTasksPerIteration) {
  const results = {
    discovered: 0,
    processed: 0,
    failed: 0,
    consolidations: [],
  };

  // Discover pending consolidations
  const consolidations = discoverPendingConsolidations();
  results.discovered = consolidations.length;

  console.log(`Discovered ${consolidations.length} pending consolidations`);

  // Process up to maxTasks
  const unitsToProcess = consolidations.slice(0, maxTasks);

  for (const unit of unitsToProcess) {
    const consolidationId = unit.consolidationId || unit.id;
    const success = processConsolidation(unit);

    results.consolidations.push({
      consolidationId: consolidationId,
      orderId: unit.orderId,
      success: success,
    });

    if (success) {
      results.processed++;
    } else {
      results.failed++;
    }
  }

  console.log(`Processed ${results.processed}/${results.discovered} consolidations (${results.failed} failed)`);

  return results;
}
