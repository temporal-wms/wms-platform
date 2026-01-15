// K6 Consolidation Service Helper Library
// Provides functions for interacting with consolidation-service and orchestrator signal bridge
// Includes multi-route support with tote arrival tracking

import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URLS, ENDPOINTS, HTTP_PARAMS, CONSOLIDATION_CONFIG, MULTI_ROUTE_CONFIG } from './config.js';
import { confirmConsolidationForOrder } from './unit.js';

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

  // 404 is expected when no consolidations exist yet for the order
  if (response.status === 404) {
    return [];
  }

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

  // Step 1b: Confirm unit consolidations for the order
  if (orderId) {
    const destinationBin = `BIN-${consolidationId.slice(-6)}`;
    const workerId = `CONSOL-WORKER-${__VU || 1}`;
    const stationId = CONSOLIDATION_CONFIG.defaultStation;
    const unitResult = confirmConsolidationForOrder(orderId, destinationBin, workerId, stationId);
    if (!unitResult.skipped) {
      console.log(`Unit consolidation confirmations: ${unitResult.success}/${unitResult.total} succeeded`);
    }
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

// ============================================================================
// Multi-Route Consolidation Support
// ============================================================================

/**
 * Sends a tote arrival signal to the orchestrator for multi-route orders
 * @param {string} orderId - The order ID
 * @param {string} toteId - The tote ID that arrived
 * @param {string} routeId - The route ID the tote belongs to
 * @param {number} routeIndex - The route index in the multi-route sequence
 * @returns {boolean} True if successful
 */
export function sendToteArrivedSignal(orderId, toteId, routeId, routeIndex = 0) {
  // Note: This endpoint needs to be added to the orchestrator's signal bridge
  const url = `${BASE_URLS.orchestrator}/api/v1/signals/tote-arrived`;
  const payload = JSON.stringify({
    orderId: orderId,
    toteId: toteId,
    routeId: routeId,
    routeIndex: routeIndex,
    arrivedAt: new Date().toISOString(),
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'signal tote arrived status 200/202': (r) => r.status === 200 || r.status === 202,
  });

  if (!success) {
    console.warn(`Failed to signal tote arrival for order ${orderId}, tote ${toteId}: ${response.status} - ${response.body}`);
  } else {
    console.log(`Tote arrival signal sent for order ${orderId}, tote ${toteId}, route ${routeId}`);
  }

  return success;
}

/**
 * Records tote arrival at consolidation service
 * @param {string} consolidationId - The consolidation ID
 * @param {string} toteId - The tote ID that arrived
 * @param {string} routeId - The route ID the tote belongs to
 * @returns {boolean} True if successful
 */
export function receiveTote(consolidationId, toteId, routeId) {
  const url = `${BASE_URLS.consolidation}/api/v1/consolidations/${consolidationId}/totes`;
  const payload = JSON.stringify({
    toteId: toteId,
    routeId: routeId,
    receivedAt: new Date().toISOString(),
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'receive tote status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to record tote ${toteId} arrival: ${response.status} - ${response.body}`);
  }

  return success;
}

/**
 * Gets tote arrival progress for a multi-route consolidation
 * @param {string} consolidationId - The consolidation ID
 * @returns {Object} Tote progress {received, expected, missing}
 */
export function getToteArrivalProgress(consolidationId) {
  const consolidation = getConsolidation(consolidationId);

  if (!consolidation) {
    return { received: 0, expected: 0, missing: [] };
  }

  const expectedTotes = consolidation.expectedTotes || [];
  const receivedTotes = consolidation.receivedTotes || [];
  const missing = expectedTotes.filter(t => !receivedTotes.includes(t));

  return {
    received: receivedTotes.length,
    expected: expectedTotes.length,
    missing: missing,
    isComplete: missing.length === 0,
    isMultiRoute: consolidation.isMultiRoute || false,
  };
}

/**
 * Creates a multi-route consolidation unit
 * @param {string} orderId - The order ID
 * @param {string} waveId - The wave ID
 * @param {Array} items - Array of expected items
 * @param {number} expectedRouteCount - Number of routes to wait for
 * @param {Array} expectedTotes - Array of expected tote IDs
 * @returns {Object|null} Created consolidation unit or null
 */
export function createMultiRouteConsolidation(orderId, waveId, items, expectedRouteCount, expectedTotes) {
  const url = `${BASE_URLS.consolidation}${ENDPOINTS.consolidation.create}`;
  const payload = JSON.stringify({
    orderId: orderId,
    waveId: waveId,
    expectedItems: items,
    isMultiRoute: true,
    expectedRouteCount: expectedRouteCount,
    expectedTotes: expectedTotes,
    strategy: 'order',
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'create multi-route consolidation status 200/201': (r) => r.status === 200 || r.status === 201,
  });

  if (!success) {
    console.warn(`Failed to create multi-route consolidation for order ${orderId}: ${response.status} - ${response.body}`);
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
 * Waits for all totes to arrive at consolidation (polling)
 * @param {string} consolidationId - The consolidation ID
 * @param {number} timeoutMs - Maximum time to wait in milliseconds
 * @param {number} pollIntervalMs - Polling interval in milliseconds
 * @returns {Object} Result {success, received, expected, timedOut}
 */
export function waitForAllTotes(consolidationId, timeoutMs = 300000, pollIntervalMs = 5000) {
  const startTime = Date.now();

  while (Date.now() - startTime < timeoutMs) {
    const progress = getToteArrivalProgress(consolidationId);

    if (!progress.isMultiRoute) {
      console.log(`Consolidation ${consolidationId} is not multi-route, no tote waiting needed`);
      return { success: true, received: 1, expected: 1, timedOut: false };
    }

    console.log(`Tote progress for ${consolidationId}: ${progress.received}/${progress.expected} received`);

    if (progress.isComplete) {
      console.log(`All ${progress.expected} totes received for consolidation ${consolidationId}`);
      return {
        success: true,
        received: progress.received,
        expected: progress.expected,
        timedOut: false,
      };
    }

    sleep(pollIntervalMs / 1000);
  }

  const finalProgress = getToteArrivalProgress(consolidationId);
  console.warn(`Timeout waiting for totes. Received ${finalProgress.received}/${finalProgress.expected}. Missing: ${finalProgress.missing.join(', ')}`);

  return {
    success: false,
    received: finalProgress.received,
    expected: finalProgress.expected,
    missing: finalProgress.missing,
    timedOut: true,
  };
}

/**
 * Processes a multi-route consolidation with tote waiting
 * @param {Object} unit - The consolidation unit object
 * @param {boolean} waitForTotes - Whether to wait for all totes before processing
 * @returns {boolean} True if fully successful
 */
export function processMultiRouteConsolidation(unit, waitForTotes = true) {
  const consolidationId = unit.consolidationId || unit.id;
  const orderId = unit.orderId;
  const isMultiRoute = unit.isMultiRoute || false;

  console.log(`Processing ${isMultiRoute ? 'multi-route' : 'single-route'} consolidation ${consolidationId} for order ${orderId}`);

  // For multi-route consolidations, wait for all totes
  if (isMultiRoute && waitForTotes) {
    const waitResult = waitForAllTotes(
      consolidationId,
      CONSOLIDATION_CONFIG.toteWaitTimeoutMs || 300000,
      CONSOLIDATION_CONFIG.toteWaitPollIntervalMs || 5000
    );

    if (!waitResult.success) {
      console.warn(`Tote wait failed for consolidation ${consolidationId}. Proceeding with partial consolidation.`);
      // Continue anyway with available totes
    }
  }

  // Process using standard consolidation flow
  return processConsolidation(unit);
}
