// K6 Waving Service Helper Library
// Provides functions for interacting with waving-service and orchestrator signal bridge

import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URLS, ENDPOINTS, HTTP_PARAMS, WAVING_CONFIG } from './config.js';

/**
 * Discovers waves that are ready for release
 * @returns {Array} Array of waves ready for release
 */
export function discoverReadyWaves() {
  const url = `${BASE_URLS.waving}${ENDPOINTS.waving.readyForRelease}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'discover ready waves status 200': (r) => r.status === 200,
  });

  if (!success || !response.body) {
    console.warn(`Failed to discover ready waves: ${response.status} - ${response.body}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return Array.isArray(data) ? data : (data.waves || data.items || []);
  } catch (e) {
    console.error(`Failed to parse waves response: ${e.message}`);
    return [];
  }
}

/**
 * Gets waves by status
 * @param {string} status - Wave status (planned, scheduled, released, completed)
 * @returns {Array} Array of waves
 */
export function getWavesByStatus(status) {
  const url = `${BASE_URLS.waving}${ENDPOINTS.waving.byStatus(status)}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get waves by status 200': (r) => r.status === 200,
  });

  if (!success || !response.body) {
    console.warn(`Failed to get waves by status ${status}: ${response.status}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return Array.isArray(data) ? data : (data.waves || data.items || []);
  } catch (e) {
    console.error(`Failed to parse waves response: ${e.message}`);
    return [];
  }
}

/**
 * Gets a specific wave by ID
 * @param {string} waveId - The wave ID
 * @returns {Object|null} The wave or null if not found
 */
export function getWave(waveId) {
  const url = `${BASE_URLS.waving}${ENDPOINTS.waving.get(waveId)}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get wave status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to get wave ${waveId}: ${response.status}`);
    return null;
  }

  try {
    return JSON.parse(response.body);
  } catch (e) {
    console.error(`Failed to parse wave response: ${e.message}`);
    return null;
  }
}

/**
 * Creates a new wave from a list of order IDs
 * @param {Array} orderIds - Array of order IDs to include in the wave
 * @param {string} waveType - Type of wave (physical, digital, mixed)
 * @returns {Object|null} The created wave or null if failed
 */
export function createWaveFromOrders(orderIds, waveType = 'digital') {
  const url = `${BASE_URLS.waving}${ENDPOINTS.waving.createFromOrders}`;
  const payload = JSON.stringify({
    orderIds: orderIds,
    waveType: waveType,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'create wave status 200/201': (r) => r.status === 200 || r.status === 201,
  });

  if (!success) {
    console.warn(`Failed to create wave from orders: ${response.status} - ${response.body}`);
    return null;
  }

  try {
    const data = JSON.parse(response.body);
    // Response is wrapped in "wave" object: { wave: {...} }
    return data.wave || data;
  } catch (e) {
    console.error(`Failed to parse create wave response: ${e.message}`);
    return null;
  }
}

/**
 * Schedules a wave for execution
 * @param {string} waveId - The wave ID
 * @returns {boolean} True if successful
 */
export function scheduleWave(waveId) {
  const url = `${BASE_URLS.waving}${ENDPOINTS.waving.schedule(waveId)}`;
  const response = http.post(url, null, HTTP_PARAMS);

  const success = check(response, {
    'schedule wave status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to schedule wave ${waveId}: ${response.status} - ${response.body}`);
  }

  return success;
}

/**
 * Releases a wave to start picking
 * @param {string} waveId - The wave ID
 * @returns {boolean} True if successful
 */
export function releaseWave(waveId) {
  const url = `${BASE_URLS.waving}${ENDPOINTS.waving.release(waveId)}`;
  const response = http.post(url, null, HTTP_PARAMS);

  const success = check(response, {
    'release wave status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to release wave ${waveId}: ${response.status} - ${response.body}`);
  }

  return success;
}

/**
 * Sends a wave assigned signal to the orchestrator to advance the Temporal workflow
 * @param {string} orderId - The order ID
 * @param {string} waveId - The wave ID
 * @returns {boolean} True if successful
 */
export function sendWaveAssignedSignal(orderId, waveId) {
  const url = `${BASE_URLS.orchestrator}${ENDPOINTS.orchestrator.signalWaveAssigned}`;
  const payload = JSON.stringify({
    orderId: orderId,
    waveId: waveId,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'signal wave assigned status 200': (r) => r.status === 200,
  });

  // Handle "workflow execution already completed" as success (idempotent behavior)
  if (!success && response.status === 500) {
    try {
      const errorBody = JSON.parse(response.body);
      if (errorBody.error && errorBody.error.includes('already completed')) {
        console.log(`Wave signal for order ${orderId}: workflow already completed (idempotent success)`);
        return true;
      }
    } catch (e) {
      // Continue to regular error handling
    }
  }

  if (!success) {
    console.warn(`Failed to signal wave assigned for order ${orderId}: ${response.status} - ${response.body}`);
  } else {
    try {
      const result = JSON.parse(response.body);
      console.log(`Wave assigned signal sent for workflow: ${result.workflowId}`);
    } catch (e) {
      // Ignore parse errors for logging
    }
  }

  return success;
}

/**
 * Simulates processing a wave: schedule, release, and signal orders
 * @param {Object} wave - The wave object
 * @returns {Object} Results with success count
 */
export function simulateWaveProcessing(wave) {
  const waveId = wave.waveId || wave.id;
  const orderIds = wave.orderIds || wave.orders || [];

  console.log(`Simulating wave processing for wave ${waveId} with ${orderIds.length} orders`);

  const results = {
    waveId: waveId,
    scheduled: false,
    released: false,
    signaled: 0,
    failed: 0,
  };

  // Step 1: Schedule the wave
  sleep(WAVING_CONFIG.simulationDelayMs / 1000);
  results.scheduled = scheduleWave(waveId);

  if (!results.scheduled) {
    console.warn(`Failed to schedule wave ${waveId}, trying to release anyway`);
  }

  // Step 2: Release the wave
  sleep(WAVING_CONFIG.simulationDelayMs / 1000);
  results.released = releaseWave(waveId);

  if (!results.released) {
    console.warn(`Failed to release wave ${waveId}`);
    return results;
  }

  // Step 3: Signal each order in the wave
  for (const orderId of orderIds) {
    sleep(WAVING_CONFIG.simulationDelayMs / 2000);
    const success = sendWaveAssignedSignal(orderId, waveId);
    if (success) {
      results.signaled++;
    } else {
      results.failed++;
    }
  }

  return results;
}

/**
 * Processes a single wave end-to-end
 * @param {Object} wave - The wave object
 * @returns {boolean} True if fully successful
 */
export function processWave(wave) {
  const waveId = wave.waveId || wave.id;
  console.log(`Processing wave ${waveId}`);

  const results = simulateWaveProcessing(wave);
  const orderCount = (wave.orderIds || wave.orders || []).length;

  return results.released && results.signaled === orderCount;
}

/**
 * Discovers and processes all ready waves
 * @param {number} maxWaves - Maximum number of waves to process
 * @returns {Object} Summary of processing results
 */
export function processAllReadyWaves(maxWaves = WAVING_CONFIG.maxWavesPerIteration) {
  const results = {
    discovered: 0,
    processed: 0,
    failed: 0,
    totalOrders: 0,
    signaledOrders: 0,
    waves: [],
  };

  // Discover ready waves
  const waves = discoverReadyWaves();
  results.discovered = waves.length;

  console.log(`Discovered ${waves.length} ready waves`);

  // Process up to maxWaves
  const wavesToProcess = waves.slice(0, maxWaves);

  for (const wave of wavesToProcess) {
    const waveId = wave.waveId || wave.id;
    const orderCount = (wave.orderIds || wave.orders || []).length;
    results.totalOrders += orderCount;

    const waveResults = simulateWaveProcessing(wave);

    results.waves.push({
      waveId: waveId,
      success: waveResults.released,
      signaled: waveResults.signaled,
    });

    results.signaledOrders += waveResults.signaled;

    if (waveResults.released) {
      results.processed++;
    } else {
      results.failed++;
    }
  }

  console.log(`Processed ${results.processed}/${results.discovered} waves (${results.failed} failed)`);
  console.log(`Signaled ${results.signaledOrders}/${results.totalOrders} orders`);

  return results;
}
