// K6 Unit Service Helper Library
// Provides functions for interacting with unit-service for item-level tracking

import http from 'k6/http';
import { check } from 'k6';
import { BASE_URLS, ENDPOINTS, HTTP_PARAMS, UNIT_CONFIG } from './config.js';

/**
 * Creates units for a given SKU at receiving
 * @param {string} sku - The SKU
 * @param {string} shipmentId - The receiving shipment ID
 * @param {string} locationId - The initial location ID
 * @param {number} quantity - Number of units to create
 * @param {string} createdBy - Worker ID who received
 * @returns {Object} { success, unitIds, count }
 */
export function createUnits(sku, shipmentId, locationId, quantity, createdBy) {
  if (!UNIT_CONFIG.enableUnitTracking) {
    return { success: true, unitIds: [], count: 0, skipped: true };
  }

  const url = `${BASE_URLS.unit}${ENDPOINTS.unit.create}`;
  const payload = JSON.stringify({
    sku: sku,
    shipmentId: shipmentId,
    locationId: locationId,
    quantity: quantity,
    createdBy: createdBy,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'create units status 201': (r) => r.status === 201,
  });

  if (!success) {
    console.warn(`Failed to create units for SKU ${sku}: ${response.status} - ${response.body}`);
    return { success: false, unitIds: [], count: 0 };
  }

  try {
    const data = JSON.parse(response.body);
    return {
      success: true,
      unitIds: data.unitIds || [],
      count: data.count || 0,
      sku: data.sku,
    };
  } catch (e) {
    console.error(`Failed to parse create units response: ${e.message}`);
    return { success: false, unitIds: [], count: 0 };
  }
}

/**
 * Reserves units for an order
 * @param {string} orderId - The order ID
 * @param {string} pathId - The pick path ID
 * @param {Array} items - Array of { sku, quantity }
 * @param {string} handlerId - Worker ID handling reservation
 * @returns {Object} { success, reservedUnits, failedItems }
 */
export function reserveUnits(orderId, pathId, items, handlerId) {
  if (!UNIT_CONFIG.enableUnitTracking) {
    return { success: true, reservedUnits: [], failedItems: [], skipped: true };
  }

  const url = `${BASE_URLS.unit}${ENDPOINTS.unit.reserve}`;
  const payload = JSON.stringify({
    orderId: orderId,
    pathId: pathId,
    items: items,
    handlerId: handlerId,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'reserve units status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to reserve units for order ${orderId}: ${response.status} - ${response.body}`);
    return { success: false, reservedUnits: [], failedItems: items };
  }

  try {
    const data = JSON.parse(response.body);
    return {
      success: true,
      reservedUnits: data.reservedUnits || [],
      failedItems: data.failedItems || [],
    };
  } catch (e) {
    console.error(`Failed to parse reserve units response: ${e.message}`);
    return { success: false, reservedUnits: [], failedItems: items };
  }
}

/**
 * Gets all units for an order
 * @param {string} orderId - The order ID
 * @returns {Array} Array of unit summaries
 */
export function getUnitsForOrder(orderId) {
  if (!UNIT_CONFIG.enableUnitTracking) {
    return [];
  }

  const url = `${BASE_URLS.unit}${ENDPOINTS.unit.byOrder(orderId)}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get units for order status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to get units for order ${orderId}: ${response.status}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return data.units || [];
  } catch (e) {
    console.error(`Failed to parse units response: ${e.message}`);
    return [];
  }
}

/**
 * Confirms pick for a unit
 * @param {string} unitId - The unit ID
 * @param {string} toteId - The tote ID where unit was placed
 * @param {string} pickerId - The picker worker ID
 * @param {string} stationId - Optional station ID
 * @returns {boolean} True if successful
 */
export function confirmUnitPick(unitId, toteId, pickerId, stationId = '') {
  if (!UNIT_CONFIG.enableUnitTracking) {
    return true;
  }

  const url = `${BASE_URLS.unit}${ENDPOINTS.unit.pick(unitId)}`;
  const payload = JSON.stringify({
    toteId: toteId,
    pickerId: pickerId,
    stationId: stationId,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'confirm unit pick status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to confirm pick for unit ${unitId}: ${response.status} - ${response.body}`);
  }

  return success;
}

/**
 * Confirms consolidation for a unit
 * @param {string} unitId - The unit ID
 * @param {string} destinationBin - The destination bin ID
 * @param {string} workerId - The worker ID
 * @param {string} stationId - Optional station ID
 * @returns {boolean} True if successful
 */
export function confirmUnitConsolidation(unitId, destinationBin, workerId, stationId = '') {
  if (!UNIT_CONFIG.enableUnitTracking) {
    return true;
  }

  const url = `${BASE_URLS.unit}${ENDPOINTS.unit.consolidate(unitId)}`;
  const payload = JSON.stringify({
    destinationBin: destinationBin,
    workerId: workerId,
    stationId: stationId,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'confirm unit consolidation status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to confirm consolidation for unit ${unitId}: ${response.status} - ${response.body}`);
  }

  return success;
}

/**
 * Confirms packing for a unit
 * @param {string} unitId - The unit ID
 * @param {string} packageId - The package ID
 * @param {string} packerId - The packer worker ID
 * @param {string} stationId - Optional station ID
 * @returns {boolean} True if successful
 */
export function confirmUnitPack(unitId, packageId, packerId, stationId = '') {
  if (!UNIT_CONFIG.enableUnitTracking) {
    return true;
  }

  const url = `${BASE_URLS.unit}${ENDPOINTS.unit.pack(unitId)}`;
  const payload = JSON.stringify({
    packageId: packageId,
    packerId: packerId,
    stationId: stationId,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'confirm unit pack status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to confirm pack for unit ${unitId}: ${response.status} - ${response.body}`);
  }

  return success;
}

/**
 * Confirms shipping for a unit
 * @param {string} unitId - The unit ID
 * @param {string} shipmentId - The outbound shipment ID
 * @param {string} trackingNumber - The tracking number
 * @param {string} handlerId - The handler worker ID
 * @returns {boolean} True if successful
 */
export function confirmUnitShip(unitId, shipmentId, trackingNumber, handlerId) {
  if (!UNIT_CONFIG.enableUnitTracking) {
    return true;
  }

  const url = `${BASE_URLS.unit}${ENDPOINTS.unit.ship(unitId)}`;
  const payload = JSON.stringify({
    shipmentId: shipmentId,
    trackingNumber: trackingNumber,
    handlerId: handlerId,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'confirm unit ship status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to confirm ship for unit ${unitId}: ${response.status} - ${response.body}`);
  }

  return success;
}

/**
 * Reports an exception for a unit
 * @param {string} unitId - The unit ID
 * @param {string} exceptionType - Type: damaged, missing, wrong_item, quantity_mismatch
 * @param {string} stage - Stage: picking, consolidation, packing, shipping
 * @param {string} description - Description of the exception
 * @param {string} reportedBy - Worker ID who reported
 * @param {string} stationId - Optional station ID
 * @returns {Object|null} Exception response or null
 */
export function reportUnitException(unitId, exceptionType, stage, description, reportedBy, stationId = '') {
  if (!UNIT_CONFIG.enableUnitTracking) {
    return null;
  }

  const url = `${BASE_URLS.unit}${ENDPOINTS.unit.exception(unitId)}`;
  const payload = JSON.stringify({
    exceptionType: exceptionType,
    stage: stage,
    description: description,
    stationId: stationId,
    reportedBy: reportedBy,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'report unit exception status 201': (r) => r.status === 201,
  });

  if (!success) {
    console.warn(`Failed to report exception for unit ${unitId}: ${response.status} - ${response.body}`);
    return null;
  }

  try {
    return JSON.parse(response.body);
  } catch (e) {
    console.error(`Failed to parse exception response: ${e.message}`);
    return null;
  }
}

/**
 * Batch confirm picks for all units in an order
 * @param {string} orderId - The order ID
 * @param {string} toteId - The tote ID
 * @param {string} pickerId - The picker ID
 * @param {string} stationId - Optional station ID
 * @returns {Object} { success: number, failed: number }
 */
export function confirmPicksForOrder(orderId, toteId, pickerId, stationId = '') {
  if (!UNIT_CONFIG.enableUnitTracking) {
    return { success: 0, failed: 0, total: 0, skipped: true };
  }

  const units = getUnitsForOrder(orderId);
  let successCount = 0;
  let failedCount = 0;

  for (const unit of units) {
    if (unit.status === 'reserved' || unit.status === 'staged') {
      if (confirmUnitPick(unit.unitId, toteId, pickerId, stationId)) {
        successCount++;
      } else {
        failedCount++;
      }
    }
  }

  return { success: successCount, failed: failedCount, total: units.length };
}

/**
 * Batch confirm consolidation for all units in an order
 * @param {string} orderId - The order ID
 * @param {string} destinationBin - The destination bin
 * @param {string} workerId - The worker ID
 * @param {string} stationId - Optional station ID
 * @returns {Object} { success: number, failed: number }
 */
export function confirmConsolidationForOrder(orderId, destinationBin, workerId, stationId = '') {
  if (!UNIT_CONFIG.enableUnitTracking) {
    return { success: 0, failed: 0, total: 0, skipped: true };
  }

  const units = getUnitsForOrder(orderId);
  let successCount = 0;
  let failedCount = 0;

  for (const unit of units) {
    if (unit.status === 'picked') {
      if (confirmUnitConsolidation(unit.unitId, destinationBin, workerId, stationId)) {
        successCount++;
      } else {
        failedCount++;
      }
    }
  }

  return { success: successCount, failed: failedCount, total: units.length };
}

/**
 * Batch confirm packing for all units in an order
 * @param {string} orderId - The order ID
 * @param {string} packageId - The package ID
 * @param {string} packerId - The packer ID
 * @param {string} stationId - Optional station ID
 * @returns {Object} { success: number, failed: number }
 */
export function confirmPacksForOrder(orderId, packageId, packerId, stationId = '') {
  if (!UNIT_CONFIG.enableUnitTracking) {
    return { success: 0, failed: 0, total: 0, skipped: true };
  }

  const units = getUnitsForOrder(orderId);
  let successCount = 0;
  let failedCount = 0;

  for (const unit of units) {
    if (unit.status === 'picked' || unit.status === 'consolidated') {
      if (confirmUnitPack(unit.unitId, packageId, packerId, stationId)) {
        successCount++;
      } else {
        failedCount++;
      }
    }
  }

  return { success: successCount, failed: failedCount, total: units.length };
}

/**
 * Batch confirm shipping for all units in an order
 * @param {string} orderId - The order ID
 * @param {string} shipmentId - The shipment ID
 * @param {string} trackingNumber - The tracking number
 * @param {string} handlerId - The handler ID
 * @returns {Object} { success: number, failed: number }
 */
export function confirmShipsForOrder(orderId, shipmentId, trackingNumber, handlerId) {
  if (!UNIT_CONFIG.enableUnitTracking) {
    return { success: 0, failed: 0, total: 0, skipped: true };
  }

  const units = getUnitsForOrder(orderId);
  let successCount = 0;
  let failedCount = 0;

  for (const unit of units) {
    if (unit.status === 'packed') {
      if (confirmUnitShip(unit.unitId, shipmentId, trackingNumber, handlerId)) {
        successCount++;
      } else {
        failedCount++;
      }
    }
  }

  return { success: successCount, failed: failedCount, total: units.length };
}

/**
 * Health check for unit service
 * @returns {boolean} True if healthy
 */
export function checkHealth() {
  if (!UNIT_CONFIG.enableUnitTracking) {
    return true;
  }

  const response = http.get(`${BASE_URLS.unit}/health`);

  return check(response, {
    'unit service healthy': (r) => r.status === 200,
  });
}
