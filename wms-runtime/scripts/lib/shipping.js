// K6 Shipping Service Helper Library
// Provides functions for interacting with shipping-service and orchestrator signal bridge

import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URLS, ENDPOINTS, HTTP_PARAMS, SHIPPING_CONFIG } from './config.js';
import { confirmShipsForOrder } from './unit.js';

/**
 * Discovers pending shipments
 * @returns {Array} Array of pending shipments
 */
export function discoverPendingShipments() {
  const url = `${BASE_URLS.shipping}${ENDPOINTS.shipping.pending}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'discover pending shipments status 200': (r) => r.status === 200,
  });

  if (!success || !response.body) {
    console.warn(`Failed to discover pending shipments: ${response.status} - ${response.body}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return Array.isArray(data) ? data : (data.shipments || data.items || []);
  } catch (e) {
    console.error(`Failed to parse shipments response: ${e.message}`);
    return [];
  }
}

/**
 * Gets labeled shipments ready for manifest
 * @returns {Array} Array of labeled shipments
 */
export function discoverLabeledShipments() {
  const url = `${BASE_URLS.shipping}${ENDPOINTS.shipping.labeled}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'discover labeled shipments status 200': (r) => r.status === 200,
  });

  if (!success || !response.body) {
    console.warn(`Failed to discover labeled shipments: ${response.status} - ${response.body}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return Array.isArray(data) ? data : (data.shipments || data.items || []);
  } catch (e) {
    console.error(`Failed to parse shipments response: ${e.message}`);
    return [];
  }
}

/**
 * Gets shipments by status
 * @param {string} status - Shipment status
 * @returns {Array} Array of shipments
 */
export function getShipmentsByStatus(status) {
  const url = `${BASE_URLS.shipping}${ENDPOINTS.shipping.byStatus(status)}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get shipments by status 200': (r) => r.status === 200,
  });

  if (!success || !response.body) {
    console.warn(`Failed to get shipments by status ${status}: ${response.status}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return Array.isArray(data) ? data : (data.shipments || data.items || []);
  } catch (e) {
    console.error(`Failed to parse shipments response: ${e.message}`);
    return [];
  }
}

/**
 * Gets shipment by order ID
 * @param {string} orderId - The order ID
 * @returns {Object|null} The shipment or null
 */
export function getShipmentByOrder(orderId) {
  const url = `${BASE_URLS.shipping}${ENDPOINTS.shipping.byOrder(orderId)}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get shipment by order status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to get shipment for order ${orderId}: ${response.status}`);
    return null;
  }

  try {
    const data = JSON.parse(response.body);
    // May return array or single object
    return Array.isArray(data) ? data[0] : data;
  } catch (e) {
    console.error(`Failed to parse shipment response: ${e.message}`);
    return null;
  }
}

/**
 * Gets a specific shipment by ID
 * @param {string} shipmentId - The shipment ID
 * @returns {Object|null} The shipment or null if not found
 */
export function getShipment(shipmentId) {
  const url = `${BASE_URLS.shipping}${ENDPOINTS.shipping.get(shipmentId)}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get shipment status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to get shipment ${shipmentId}: ${response.status}`);
    return null;
  }

  try {
    return JSON.parse(response.body);
  } catch (e) {
    console.error(`Failed to parse shipment response: ${e.message}`);
    return null;
  }
}

/**
 * Generates a shipping label for a shipment
 * @param {string} shipmentId - The shipment ID
 * @param {Object} labelInfo - Label information {carrier, service, trackingNumber}
 * @returns {Object|null} Label result or null
 */
export function generateLabel(shipmentId, labelInfo = {}) {
  const url = `${BASE_URLS.shipping}${ENDPOINTS.shipping.label(shipmentId)}`;
  const payload = JSON.stringify({
    carrier: labelInfo.carrier || SHIPPING_CONFIG.defaultCarrier,
    service: labelInfo.service || 'ground',
    trackingNumber: labelInfo.trackingNumber,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'generate label status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to generate label for shipment ${shipmentId}: ${response.status} - ${response.body}`);
    return null;
  }

  try {
    return JSON.parse(response.body);
  } catch (e) {
    console.error(`Failed to parse label response: ${e.message}`);
    return null;
  }
}

/**
 * Adds shipment to a manifest
 * @param {string} shipmentId - The shipment ID
 * @param {string} manifestId - The manifest ID (optional, will be assigned if not provided)
 * @returns {boolean} True if successful
 */
export function addToManifest(shipmentId, manifestId = null) {
  const url = `${BASE_URLS.shipping}${ENDPOINTS.shipping.manifest(shipmentId)}`;
  const payload = JSON.stringify({
    manifestId: manifestId,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'add to manifest status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to add shipment ${shipmentId} to manifest: ${response.status} - ${response.body}`);
  }

  return success;
}

/**
 * Confirms shipment has shipped
 * @param {string} shipmentId - The shipment ID
 * @returns {Object|null} The shipped shipment or null
 */
export function confirmShipment(shipmentId) {
  const url = `${BASE_URLS.shipping}${ENDPOINTS.shipping.ship(shipmentId)}`;
  const response = http.post(url, null, HTTP_PARAMS);

  const success = check(response, {
    'confirm shipment status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to confirm shipment ${shipmentId}: ${response.status} - ${response.body}`);
    return null;
  }

  try {
    return JSON.parse(response.body);
  } catch (e) {
    console.error(`Failed to parse ship response: ${e.message}`);
    return null;
  }
}

/**
 * Sends a ship confirmed signal to the orchestrator
 * @param {string} orderId - The order ID
 * @param {string} shipmentId - The shipment ID
 * @param {Object} shipmentInfo - Shipment information {trackingNumber, carrier, shippedAt}
 * @returns {boolean} True if successful
 */
export function sendShipConfirmedSignal(orderId, shipmentId, shipmentInfo = {}) {
  const url = `${BASE_URLS.orchestrator}${ENDPOINTS.orchestrator.signalShipConfirmed}`;
  const payload = JSON.stringify({
    orderId: orderId,
    shipmentId: shipmentId,
    trackingNumber: shipmentInfo.trackingNumber,
    carrier: shipmentInfo.carrier,
    shippedAt: shipmentInfo.shippedAt || new Date().toISOString(),
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'signal ship confirmed status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to signal ship confirmed for order ${orderId}: ${response.status} - ${response.body}`);
  } else {
    try {
      const result = JSON.parse(response.body);
      console.log(`Ship confirmed signal sent for workflow: ${result.workflowId}`);
    } catch (e) {
      // Ignore parse errors for logging
    }
  }

  return success;
}

/**
 * Simulates the full shipping workflow for a shipment
 * @param {Object} shipment - The shipment object
 * @returns {Object} Shipment info for signaling
 */
export function simulateShipping(shipment) {
  const shipmentId = shipment.shipmentId || shipment.id;
  const carrier = SHIPPING_CONFIG.defaultCarrier;

  console.log(`Simulating shipping for shipment ${shipmentId}`);

  const shipmentInfo = {
    shipmentId: shipmentId,
    trackingNumber: null,
    carrier: carrier,
    labeled: false,
    manifested: false,
    shipped: false,
    shippedAt: null,
  };

  // Step 1: Generate label
  sleep(SHIPPING_CONFIG.simulationDelayMs / 1000);
  const trackingNumber = `${carrier}-${Date.now()}-${shipmentId.slice(-4)}`;
  const labelResult = generateLabel(shipmentId, {
    carrier: carrier,
    trackingNumber: trackingNumber,
  });

  if (labelResult) {
    shipmentInfo.labeled = true;
    shipmentInfo.trackingNumber = labelResult.trackingNumber || trackingNumber;
  }

  // Step 2: Add to manifest
  sleep(SHIPPING_CONFIG.simulationDelayMs / 1000);
  shipmentInfo.manifested = addToManifest(shipmentId);

  // Step 3: Confirm shipment
  sleep(SHIPPING_CONFIG.simulationDelayMs / 1000);
  const shipped = confirmShipment(shipmentId);
  if (shipped) {
    shipmentInfo.shipped = true;
    shipmentInfo.shippedAt = shipped.shippedAt || new Date().toISOString();
  }

  return shipmentInfo;
}

/**
 * Processes a single shipment end-to-end
 * @param {Object} shipment - The shipment object
 * @returns {boolean} True if fully successful
 */
export function processShipment(shipment) {
  const shipmentId = shipment.shipmentId || shipment.id;
  const orderId = shipment.orderId;

  console.log(`Processing shipment ${shipmentId} for order ${orderId}`);

  // Step 1: Simulate shipping workflow
  const shipmentInfo = simulateShipping(shipment);

  if (!shipmentInfo.shipped) {
    console.warn(`Failed to complete shipping for ${shipmentId}`);
    return false;
  }

  // Step 1b: Confirm unit ships for the order
  if (orderId) {
    const handlerId = `SHIPPER-SIM-${__VU || 1}`;
    const unitResult = confirmShipsForOrder(
      orderId,
      shipmentId,
      shipmentInfo.trackingNumber,
      handlerId
    );
    if (!unitResult.skipped) {
      console.log(`Unit ship confirmations: ${unitResult.success}/${unitResult.total} succeeded`);
    }
  }

  // Note: No signal needed - workflow activities handle progression automatically
  console.log(`Shipment ${shipmentId} completed successfully with tracking: ${shipmentInfo.trackingNumber}`);
  return true;
}

/**
 * Discovers and processes all pending shipments
 * @param {number} maxShipments - Maximum number of shipments to process
 * @returns {Object} Summary of processing results
 */
export function processAllPendingShipments(maxShipments = SHIPPING_CONFIG.maxShipmentsPerIteration) {
  const results = {
    discovered: 0,
    processed: 0,
    failed: 0,
    shipments: [],
  };

  // Discover pending shipments
  const shipments = discoverPendingShipments();
  results.discovered = shipments.length;

  console.log(`Discovered ${shipments.length} pending shipments`);

  // Process up to maxShipments
  const shipmentsToProcess = shipments.slice(0, maxShipments);

  for (const shipment of shipmentsToProcess) {
    const shipmentId = shipment.shipmentId || shipment.id;
    const success = processShipment(shipment);

    results.shipments.push({
      shipmentId: shipmentId,
      orderId: shipment.orderId,
      success: success,
    });

    if (success) {
      results.processed++;
    } else {
      results.failed++;
    }
  }

  console.log(`Processed ${results.processed}/${results.discovered} shipments (${results.failed} failed)`);

  return results;
}
