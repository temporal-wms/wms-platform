// K6 Sortation Service Helper Library
// Provides functions for simulating package sortation operations

import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URLS, ENDPOINTS, HTTP_PARAMS } from './config.js';

// Sortation-specific configuration
export const SORTATION_CONFIG = {
  simulationDelayMs: parseInt(__ENV.SORTATION_DELAY_MS || '400'),
  maxBatchesPerIteration: parseInt(__ENV.MAX_SORTATION_BATCHES || '10'),
  sortDelayMs: parseInt(__ENV.SORT_DELAY_MS || '200'),
  defaultCenter: __ENV.DEFAULT_SORTATION_CENTER || 'SORT-CENTER-1',
};

// Sortation batch status constants
export const SORTATION_STATUS = {
  CREATED: 'created',
  RECEIVING: 'receiving',
  SORTING: 'sorting',
  READY: 'ready',
  DISPATCHED: 'dispatched',
};

// Carrier constants
export const CARRIERS = {
  UPS: 'UPS',
  FEDEX: 'FEDEX',
  USPS: 'USPS',
  DHL: 'DHL',
};

/**
 * Lists all sortation batches
 * @param {number} limit - Maximum number of batches to return
 * @returns {Array} Array of sortation batches
 */
export function listBatches(limit = 100) {
  const url = `${BASE_URLS.sortation}/api/v1/batches?limit=${limit}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'list batches status 200': (r) => r.status === 200,
  });

  if (!success || !response.body) {
    console.warn(`Failed to list batches: ${response.status}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return Array.isArray(data) ? data : (data.batches || data.items || []);
  } catch (e) {
    console.error(`Failed to parse batches response: ${e.message}`);
    return [];
  }
}

/**
 * Gets batches by status
 * @param {string} status - Batch status to filter by
 * @returns {Array} Array of sortation batches
 */
export function getBatchesByStatus(status) {
  const url = `${BASE_URLS.sortation}/api/v1/batches/status/${status}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get batches by status 200': (r) => r.status === 200,
  });

  if (!success || !response.body) {
    console.warn(`Failed to get batches by status ${status}: ${response.status}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return Array.isArray(data) ? data : (data.batches || []);
  } catch (e) {
    console.error(`Failed to parse batches response: ${e.message}`);
    return [];
  }
}

/**
 * Gets batches ready for dispatch
 * @returns {Array} Array of ready batches
 */
export function getReadyBatches() {
  const url = `${BASE_URLS.sortation}/api/v1/batches/ready`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get ready batches status 200': (r) => r.status === 200,
  });

  if (!success || !response.body) {
    console.warn(`Failed to get ready batches: ${response.status}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return Array.isArray(data) ? data : (data.batches || []);
  } catch (e) {
    console.error(`Failed to parse ready batches response: ${e.message}`);
    return [];
  }
}

/**
 * Gets a specific sortation batch by ID
 * @param {string} batchId - The batch ID
 * @returns {Object|null} The sortation batch or null if not found
 */
export function getBatch(batchId) {
  const url = `${BASE_URLS.sortation}/api/v1/batches/${batchId}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get batch status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to get batch ${batchId}: ${response.status}`);
    return null;
  }

  try {
    return JSON.parse(response.body);
  } catch (e) {
    console.error(`Failed to parse batch response: ${e.message}`);
    return null;
  }
}

/**
 * Creates a new sortation batch
 * @param {Object} batchData - Batch creation data
 * @returns {Object|null} Created batch or null if failed
 */
export function createBatch(batchData) {
  const url = `${BASE_URLS.sortation}/api/v1/batches`;
  const payload = JSON.stringify({
    sortationCenter: batchData.sortationCenter || SORTATION_CONFIG.defaultCenter,
    destinationGroup: batchData.destinationGroup,
    carrierId: batchData.carrierId || CARRIERS.UPS,
    assignedChute: batchData.assignedChute,
    trailerID: batchData.trailerId,
    dispatchDock: batchData.dispatchDock,
    scheduledDispatch: batchData.scheduledDispatch,
    metadata: batchData.metadata || {},
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'create batch status 201': (r) => r.status === 201 || r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to create batch: ${response.status} - ${response.body}`);
    return null;
  }

  try {
    const result = JSON.parse(response.body);
    console.log(`Created sortation batch: ${result.batchId || result.id}`);
    return result;
  } catch (e) {
    console.error(`Failed to parse batch response: ${e.message}`);
    return null;
  }
}

/**
 * Adds a package to a sortation batch
 * @param {string} batchId - The batch ID
 * @param {Object} packageData - Package data
 * @returns {boolean} True if successful
 */
export function addPackageToBatch(batchId, packageData) {
  const url = `${BASE_URLS.sortation}/api/v1/batches/${batchId}/packages`;
  const payload = JSON.stringify({
    packageId: packageData.packageId,
    trackingNumber: packageData.trackingNumber,
    orderId: packageData.orderId,
    weight: packageData.weight,
    dimensions: packageData.dimensions,
    destination: packageData.destination,
    receivedAt: new Date().toISOString(),
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'add package to batch status 200': (r) => r.status === 200 || r.status === 201,
  });

  if (!success) {
    console.warn(`Failed to add package to batch ${batchId}: ${response.status}`);
  } else {
    console.log(`Added package ${packageData.packageId} to batch ${batchId}`);
  }

  return success;
}

/**
 * Sorts a package to a chute
 * @param {string} batchId - The batch ID
 * @param {Object} sortData - Sort operation data
 * @returns {boolean} True if successful
 */
export function sortPackage(batchId, sortData) {
  const url = `${BASE_URLS.sortation}/api/v1/batches/${batchId}/sort`;
  const payload = JSON.stringify({
    packageId: sortData.packageId,
    chuteId: sortData.chuteId,
    sortedBy: sortData.sortedBy || `SORTER-${__VU || 1}`,
    sortedAt: new Date().toISOString(),
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'sort package status 200': (r) => r.status === 200 || r.status === 204,
  });

  if (!success) {
    console.warn(`Failed to sort package in batch ${batchId}: ${response.status}`);
  } else {
    console.log(`Sorted package ${sortData.packageId} to chute ${sortData.chuteId}`);
  }

  return success;
}

/**
 * Marks a batch as ready for dispatch
 * @param {string} batchId - The batch ID
 * @returns {boolean} True if successful
 */
export function markBatchReady(batchId) {
  const url = `${BASE_URLS.sortation}/api/v1/batches/${batchId}/ready`;
  const payload = JSON.stringify({
    readyAt: new Date().toISOString(),
    markedBy: `SUPERVISOR-${__VU || 1}`,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'mark batch ready status 200': (r) => r.status === 200 || r.status === 204,
  });

  if (!success) {
    console.warn(`Failed to mark batch ${batchId} as ready: ${response.status}`);
  } else {
    console.log(`Marked batch ${batchId} as ready for dispatch`);
  }

  return success;
}

/**
 * Dispatches a batch
 * @param {string} batchId - The batch ID
 * @param {Object} dispatchData - Dispatch details
 * @returns {Object|null} Dispatch result or null if failed
 */
export function dispatchBatch(batchId, dispatchData = {}) {
  const url = `${BASE_URLS.sortation}/api/v1/batches/${batchId}/dispatch`;
  const payload = JSON.stringify({
    dispatchedAt: new Date().toISOString(),
    dispatchedBy: dispatchData.dispatchedBy || `DISPATCHER-${__VU || 1}`,
    trailerId: dispatchData.trailerId,
    dockId: dispatchData.dockId,
    sealNumber: dispatchData.sealNumber,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'dispatch batch status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to dispatch batch ${batchId}: ${response.status}`);
    return null;
  }

  try {
    const result = JSON.parse(response.body);
    console.log(`Dispatched batch ${batchId}`);
    return result;
  } catch (e) {
    console.error(`Failed to parse dispatch response: ${e.message}`);
    return null;
  }
}

/**
 * Simulates sorting all packages in a batch
 * @param {Object} batch - The sortation batch
 * @returns {Object} Sorting result
 */
export function simulateBatchSorting(batch) {
  const batchId = batch.batchId || batch.id;
  const packages = batch.packages || [];

  console.log(`Simulating sorting for batch ${batchId} with ${packages.length} packages`);

  const results = {
    batchId: batchId,
    sorted: 0,
    failed: 0,
  };

  for (const pkg of packages) {
    // Simulate sorting delay
    sleep(SORTATION_CONFIG.sortDelayMs / 1000);

    const success = sortPackage(batchId, {
      packageId: pkg.packageId || pkg.id,
      chuteId: batch.assignedChute || `CHUTE-${Math.floor(Math.random() * 20) + 1}`,
    });

    if (success) {
      results.sorted++;
    } else {
      results.failed++;
    }
  }

  console.log(`Batch ${batchId}: sorted ${results.sorted}/${packages.length} packages`);

  return results;
}

/**
 * Processes a batch through the full sortation workflow
 * @param {Object} batch - The sortation batch
 * @returns {boolean} True if fully successful
 */
export function processBatch(batch) {
  const batchId = batch.batchId || batch.id;
  console.log(`Processing sortation batch ${batchId}`);

  // Step 1: Sort all packages
  const sortResult = simulateBatchSorting(batch);
  if (sortResult.failed > 0) {
    console.warn(`Batch ${batchId} had ${sortResult.failed} sorting failures`);
  }

  // Step 2: Mark as ready
  sleep(SORTATION_CONFIG.simulationDelayMs / 1000);
  const readySuccess = markBatchReady(batchId);
  if (!readySuccess) {
    console.warn(`Failed to mark batch ${batchId} as ready`);
    return false;
  }

  // Step 3: Dispatch
  sleep(SORTATION_CONFIG.simulationDelayMs / 1000);
  const dispatchResult = dispatchBatch(batchId, {
    trailerId: `TRAILER-${Math.floor(Math.random() * 100)}`,
    dockId: `DOCK-${Math.floor(Math.random() * 10) + 1}`,
    sealNumber: `SEAL-${Date.now()}`,
  });

  return dispatchResult !== null;
}

/**
 * Discovers and processes batches that are in sorting status
 * @param {number} maxBatches - Maximum batches to process
 * @returns {Object} Processing summary
 */
export function processAllSortingBatches(maxBatches = SORTATION_CONFIG.maxBatchesPerIteration) {
  const results = {
    discovered: 0,
    processed: 0,
    failed: 0,
    batches: [],
  };

  // Discover batches in sorting status
  const batches = getBatchesByStatus(SORTATION_STATUS.SORTING);
  results.discovered = batches.length;

  console.log(`Discovered ${batches.length} batches in sorting status`);

  // Process up to maxBatches
  const batchesToProcess = batches.slice(0, maxBatches);

  for (const batch of batchesToProcess) {
    const success = processBatch(batch);

    results.batches.push({
      batchId: batch.batchId || batch.id,
      success: success,
    });

    if (success) {
      results.processed++;
    } else {
      results.failed++;
    }
  }

  console.log(`Processed ${results.processed}/${results.discovered} batches (${results.failed} failed)`);

  return results;
}

/**
 * Processes ready batches for dispatch
 * @param {number} maxBatches - Maximum batches to dispatch
 * @returns {Object} Dispatch summary
 */
export function dispatchReadyBatches(maxBatches = SORTATION_CONFIG.maxBatchesPerIteration) {
  const results = {
    discovered: 0,
    dispatched: 0,
    failed: 0,
  };

  const readyBatches = getReadyBatches();
  results.discovered = readyBatches.length;

  console.log(`Found ${readyBatches.length} batches ready for dispatch`);

  const batchesToDispatch = readyBatches.slice(0, maxBatches);

  for (const batch of batchesToDispatch) {
    const dispatchResult = dispatchBatch(batch.batchId || batch.id, {
      trailerId: `TRAILER-${Math.floor(Math.random() * 100)}`,
      dockId: `DOCK-${Math.floor(Math.random() * 10) + 1}`,
      sealNumber: `SEAL-${Date.now()}`,
    });

    if (dispatchResult) {
      results.dispatched++;
    } else {
      results.failed++;
    }

    sleep(SORTATION_CONFIG.simulationDelayMs / 1000);
  }

  console.log(`Dispatched ${results.dispatched}/${results.discovered} batches`);

  return results;
}
