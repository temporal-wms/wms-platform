// K6 Receiving Service Helper Library
// Provides functions for simulating inbound goods receipt operations

import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URLS, ENDPOINTS, HTTP_PARAMS, SIGNAL_CONFIG } from './config.js';

// Receiving-specific configuration
export const RECEIVING_CONFIG = {
  simulationDelayMs: parseInt(__ENV.RECEIVING_DELAY_MS || '800'),
  maxShipmentsPerIteration: parseInt(__ENV.MAX_RECEIVING_SHIPMENTS || '5'),
  defaultDockDoor: __ENV.DEFAULT_DOCK_DOOR || 'DOCK-1',
  receiptConfirmationDelayMs: parseInt(__ENV.RECEIPT_CONFIRM_DELAY_MS || '500'),
};

// Shipment status constants
export const SHIPMENT_STATUS = {
  EXPECTED: 'expected',
  ARRIVED: 'arrived',
  RECEIVING: 'receiving',
  RECEIVED: 'received',
  STOW_PENDING: 'stow_pending',
  COMPLETED: 'completed',
};

// ASN (Advance Shipment Notice) types
export const ASN_TYPES = {
  PURCHASE_ORDER: 'purchase_order',
  TRANSFER: 'transfer',
  RETURN: 'customer_return',
  REPLENISHMENT: 'replenishment',
};

/**
 * Creates an inbound shipment (ASN - Advance Shipment Notice)
 * @param {Object} shipmentData - Shipment details
 * @returns {Object} Created shipment or null if failed
 */
export function createInboundShipment(shipmentData) {
  const url = `${BASE_URLS.receiving}/api/v1/shipments`;

  const shipmentId = shipmentData.shipmentId || `SHIP-${Date.now()}-${Math.floor(Math.random() * 10000)}`;
  const asnId = shipmentData.asnNumber || `ASN-${Date.now()}-${Math.floor(Math.random() * 10000)}`;

  // Transform items to expectedItems format
  const expectedItems = (shipmentData.items || []).map(item => ({
    sku: item.sku || `SKU-${Math.floor(Math.random() * 1000)}`,
    productName: item.productName || item.sku || 'Unknown Product',
    expectedQuantity: item.expectedQuantity || item.quantity || 10,
    unitCost: item.unitCost || 9.99,
    weight: item.weight || 0.5,
    isHazmat: item.isHazmat || false,
    requiresColdChain: item.requiresColdChain || false,
  }));

  // Ensure at least one item
  if (expectedItems.length === 0) {
    expectedItems.push({
      sku: `SKU-DEFAULT-${Math.floor(Math.random() * 1000)}`,
      productName: 'Default Test Product',
      expectedQuantity: 10,
      unitCost: 9.99,
      weight: 0.5,
      isHazmat: false,
      requiresColdChain: false,
    });
  }

  const payload = JSON.stringify({
    shipmentId: shipmentId,
    purchaseOrderId: shipmentData.purchaseOrderId || `PO-${Date.now()}`,
    asn: {
      asnId: asnId,
      carrierName: shipmentData.carrierName || 'UPS',
      trackingNumber: shipmentData.trackingNumber || `TRK-${Date.now()}`,
      expectedArrival: shipmentData.expectedArrival || new Date(Date.now() + 86400000).toISOString(),
      containerCount: shipmentData.containerCount || 1,
      totalWeight: shipmentData.totalWeight || expectedItems.reduce((sum, i) => sum + (i.weight * i.expectedQuantity), 0),
      specialHandling: shipmentData.specialHandling || [],
    },
    supplier: {
      supplierId: shipmentData.vendorId || `SUP-${Math.floor(Math.random() * 100)}`,
      supplierName: shipmentData.vendorName || shipmentData.vendorId || 'Test Supplier Inc.',
      contactName: shipmentData.contactName || 'John Doe',
      contactPhone: shipmentData.contactPhone || '+1-555-0100',
      contactEmail: shipmentData.contactEmail || 'supplier@example.com',
    },
    expectedItems: expectedItems,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'create shipment status 201': (r) => r.status === 201 || r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to create inbound shipment: ${response.status} - ${response.body}`);
    return null;
  }

  try {
    const result = JSON.parse(response.body);
    console.log(`Created inbound shipment: ${result.shipmentId || result.id}`);
    return result;
  } catch (e) {
    console.error(`Failed to parse shipment response: ${e.message}`);
    return null;
  }
}

/**
 * Discovers pending inbound shipments ready for receiving
 * @param {string} status - Shipment status to filter by
 * @returns {Array} Array of pending shipments
 */
export function discoverPendingShipments(status = SHIPMENT_STATUS.ARRIVED) {
  const url = `${BASE_URLS.receiving}/api/v1/shipments?status=${status}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'discover shipments status 200': (r) => r.status === 200,
  });

  if (!success || !response.body) {
    console.warn(`Failed to discover shipments: ${response.status}`);
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
 * Gets a specific shipment by ID
 * @param {string} shipmentId - The shipment ID
 * @returns {Object|null} The shipment or null if not found
 */
export function getShipment(shipmentId) {
  const url = `${BASE_URLS.receiving}/api/v1/shipments/${shipmentId}`;
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
 * Marks a shipment as arrived at dock
 * @param {string} shipmentId - The shipment ID
 * @param {string} dockId - The dock where truck arrived
 * @returns {boolean} True if successful
 */
export function markShipmentArrived(shipmentId, dockId = RECEIVING_CONFIG.defaultDockDoor) {
  const url = `${BASE_URLS.receiving}/api/v1/shipments/${shipmentId}/arrive`;
  const payload = JSON.stringify({
    dockId: dockId,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'mark arrived status 200': (r) => r.status === 200 || r.status === 204,
  });

  if (!success) {
    console.warn(`Failed to mark shipment ${shipmentId} as arrived: ${response.status} - ${response.body}`);
  } else {
    console.log(`Shipment ${shipmentId} arrived at dock ${dockId}`);
  }

  return success;
}

/**
 * Starts receiving process for a shipment
 * @param {string} shipmentId - The shipment ID
 * @param {string} workerId - The worker ID
 * @returns {boolean} True if successful
 */
export function startReceiving(shipmentId, workerId = null) {
  const effectiveWorkerId = workerId || `RECEIVER-${__VU || 1}`;
  const url = `${BASE_URLS.receiving}/api/v1/shipments/${shipmentId}/start`;
  const payload = JSON.stringify({
    workerId: effectiveWorkerId,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'start receiving status 200': (r) => r.status === 200 || r.status === 204,
  });

  if (!success) {
    console.warn(`Failed to start receiving for ${shipmentId}: ${response.status} - ${response.body}`);
  } else {
    console.log(`Started receiving for shipment ${shipmentId} with worker ${effectiveWorkerId}`);
  }

  return success;
}

/**
 * Receives an item in a shipment
 * @param {string} shipmentId - The shipment ID
 * @param {Object} itemDetails - Item receipt details
 * @returns {boolean} True if successful
 */
export function receiveItem(shipmentId, itemDetails) {
  const url = `${BASE_URLS.receiving}/api/v1/shipments/${shipmentId}/receive`;
  const payload = JSON.stringify({
    sku: itemDetails.sku,
    quantity: itemDetails.quantity || itemDetails.expectedQuantity || 1,
    condition: itemDetails.condition || 'good',
    toteId: itemDetails.toteId || `TOTE-${Date.now()}`,
    workerId: itemDetails.workerId || `RECEIVER-${__VU || 1}`,
    notes: itemDetails.notes || '',
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'receive item status 200': (r) => r.status === 200 || r.status === 204,
  });

  if (!success) {
    console.warn(`Failed to receive item ${itemDetails.sku}: ${response.status} - ${response.body}`);
  } else {
    console.log(`Received ${itemDetails.quantity || 1} x ${itemDetails.sku}`);
  }

  return success;
}

/**
 * Confirms receipt of an individual item in a shipment
 * @param {string} shipmentId - The shipment ID
 * @param {string} itemId - The item ID within the shipment
 * @param {Object} receiptDetails - Receipt details (quantity, condition, etc.)
 * @returns {boolean} True if successful
 */
export function confirmItemReceipt(shipmentId, itemId, receiptDetails) {
  const url = `${BASE_URLS.receiving}/api/v1/shipments/${shipmentId}/items/${itemId}/confirm`;
  const payload = JSON.stringify({
    receivedQuantity: receiptDetails.quantity,
    condition: receiptDetails.condition || 'good',
    licensePlate: receiptDetails.licensePlate || `LP-${Date.now()}`,
    expirationDate: receiptDetails.expirationDate || null,
    lotNumber: receiptDetails.lotNumber || null,
    serialNumbers: receiptDetails.serialNumbers || [],
    notes: receiptDetails.notes || '',
    confirmedAt: new Date().toISOString(),
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'confirm item receipt status 200': (r) => r.status === 200 || r.status === 204,
  });

  if (!success) {
    console.warn(`Failed to confirm receipt for item ${itemId}: ${response.status}`);
  }

  return success;
}

/**
 * Reports a discrepancy during receiving (short ship, damage, etc.)
 * @param {string} shipmentId - The shipment ID
 * @param {string} itemId - The item ID
 * @param {Object} discrepancy - Discrepancy details
 * @returns {boolean} True if successful
 */
export function reportDiscrepancy(shipmentId, itemId, discrepancy) {
  const url = `${BASE_URLS.receiving}/api/v1/shipments/${shipmentId}/items/${itemId}/discrepancy`;
  const payload = JSON.stringify({
    type: discrepancy.type, // 'short', 'over', 'damaged', 'wrong_item'
    expectedQuantity: discrepancy.expectedQuantity,
    actualQuantity: discrepancy.actualQuantity,
    description: discrepancy.description || '',
    photos: discrepancy.photos || [],
    reportedAt: new Date().toISOString(),
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'report discrepancy status 200': (r) => r.status === 200 || r.status === 201,
  });

  if (!success) {
    console.warn(`Failed to report discrepancy for item ${itemId}: ${response.status}`);
  }

  return success;
}

/**
 * Completes receiving for a shipment
 * @param {string} shipmentId - The shipment ID
 * @returns {Object|null} Completed shipment or null if failed
 */
export function completeReceiving(shipmentId) {
  const url = `${BASE_URLS.receiving}/api/v1/shipments/${shipmentId}/complete`;
  const payload = JSON.stringify({
    completedAt: new Date().toISOString(),
    completedBy: `RECEIVER-${__VU || 1}`,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'complete receiving status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to complete receiving for ${shipmentId}: ${response.status}`);
    return null;
  }

  try {
    const result = JSON.parse(response.body);
    console.log(`Completed receiving for shipment ${shipmentId}`);
    return result;
  } catch (e) {
    console.error(`Failed to parse complete response: ${e.message}`);
    return null;
  }
}

/**
 * Sends receiving completed signal to the orchestrator
 * @param {string} shipmentId - The shipment ID
 * @param {Array} receivedItems - Array of received items
 * @returns {boolean} True if successful
 */
export function signalReceivingCompleted(shipmentId, receivedItems) {
  const url = `${BASE_URLS.orchestrator}/api/v1/signals/receiving-completed`;
  const payload = JSON.stringify({
    shipmentId: shipmentId,
    receivedItems: receivedItems,
    completedAt: new Date().toISOString(),
  });

  let success = false;
  let lastResponse = null;

  for (let attempt = 1; attempt <= SIGNAL_CONFIG.maxRetries; attempt++) {
    const response = http.post(url, payload, {
      ...HTTP_PARAMS,
      timeout: `${SIGNAL_CONFIG.timeoutMs}ms`,
    });
    lastResponse = response;

    success = check(response, {
      'signal receiving completed status 200': (r) => r.status === 200,
    });

    if (success) {
      if (attempt > 1) {
        console.log(`Signal succeeded on attempt ${attempt}/${SIGNAL_CONFIG.maxRetries}`);
      }
      break;
    }

    if (attempt < SIGNAL_CONFIG.maxRetries) {
      console.warn(`Signal attempt ${attempt}/${SIGNAL_CONFIG.maxRetries} failed: ${lastResponse.status}, retrying...`);
      sleep(SIGNAL_CONFIG.retryDelayMs / 1000);
    }
  }

  if (!success) {
    console.warn(`Failed to signal receiving completed for ${shipmentId}: ${lastResponse.status}`);
  } else {
    console.log(`Signaled receiving completed for shipment ${shipmentId}`);
  }

  return success;
}

/**
 * Simulates receiving all items in a shipment
 * @param {Object} shipment - The shipment object
 * @returns {Array} Array of received items for signaling
 */
export function simulateReceivingShipment(shipment) {
  const receivedItems = [];
  const shipmentId = shipment.shipmentId || shipment.id;
  // API returns expectedItems, but we may also get items from creation
  const items = shipment.expectedItems || shipment.items || [];

  console.log(`Simulating receiving for shipment ${shipmentId} with ${items.length} items`);

  // Step 1: Mark shipment as arrived at dock
  if (!markShipmentArrived(shipmentId)) {
    console.warn(`Failed to mark shipment as arrived, trying to continue anyway`);
  }

  // Step 2: Start receiving process
  if (!startReceiving(shipmentId)) {
    console.warn(`Failed to start receiving, trying to continue anyway`);
  }

  // Step 3: Receive each item
  for (const item of items) {
    // Simulate receiving delay
    sleep(RECEIVING_CONFIG.receiptConfirmationDelayMs / 1000);

    // Receive the item using the correct API
    const quantity = item.expectedQuantity || item.quantity || 1;
    const success = receiveItem(shipmentId, {
      sku: item.sku,
      quantity: quantity,
      condition: 'good',
      toteId: `TOTE-${Date.now()}-${item.sku}`,
    });

    if (success) {
      receivedItems.push({
        sku: item.sku,
        productName: item.productName,
        quantity: quantity,
        toteId: `TOTE-${Date.now()}-${item.sku}`,
        condition: 'good',
      });
    }
  }

  return receivedItems;
}

/**
 * Processes a single shipment end-to-end: receive items, complete, signal
 * @param {Object} shipment - The shipment object (may be partial from create response)
 * @param {Array} originalItems - Optional: original items if not in shipment object
 * @returns {boolean} True if fully successful
 */
export function processInboundShipment(shipment, originalItems = null) {
  const shipmentId = shipment.shipmentId || shipment.id;
  console.log(`Processing inbound shipment ${shipmentId}`);

  // Get full shipment details if items not present
  let fullShipment = shipment;
  if (!shipment.expectedItems && !shipment.items) {
    console.log(`Fetching full shipment details for ${shipmentId}`);
    fullShipment = getShipment(shipmentId);
    if (!fullShipment) {
      console.warn(`Failed to fetch shipment ${shipmentId}`);
      // Use original items if provided
      if (originalItems) {
        fullShipment = { ...shipment, expectedItems: originalItems };
      } else {
        return false;
      }
    }
  }

  // If still no items but originalItems provided, use them
  if (!fullShipment.expectedItems && !fullShipment.items && originalItems) {
    fullShipment = { ...fullShipment, expectedItems: originalItems };
  }

  // Step 1: Simulate receiving all items
  const receivedItems = simulateReceivingShipment(fullShipment);

  if (receivedItems.length === 0) {
    console.warn(`No items received for shipment ${shipmentId}`);
    return false;
  }

  // Step 2: Complete the receiving
  const completedShipment = completeReceiving(shipmentId);
  if (!completedShipment) {
    console.warn(`Failed to complete shipment ${shipmentId}`);
    // Still try to signal workflow
  }

  // Step 3: Signal the workflow to advance
  const signalSuccess = signalReceivingCompleted(shipmentId, receivedItems);

  return signalSuccess;
}

/**
 * Discovers and processes all pending inbound shipments
 * @param {number} maxShipments - Maximum number of shipments to process
 * @returns {Object} Summary of processing results
 */
export function processAllPendingShipments(maxShipments = RECEIVING_CONFIG.maxShipmentsPerIteration) {
  const results = {
    discovered: 0,
    processed: 0,
    failed: 0,
    shipments: [],
  };

  // Discover pending shipments
  const shipments = discoverPendingShipments(SHIPMENT_STATUS.ARRIVED);
  results.discovered = shipments.length;

  console.log(`Discovered ${shipments.length} pending inbound shipments`);

  // Process up to maxShipments
  const shipmentsToProcess = shipments.slice(0, maxShipments);

  for (const shipment of shipmentsToProcess) {
    const success = processInboundShipment(shipment);

    results.shipments.push({
      shipmentId: shipment.shipmentId || shipment.id,
      success: success,
    });

    if (success) {
      results.processed++;
    } else {
      results.failed++;
    }

    // Delay between shipments
    sleep(RECEIVING_CONFIG.simulationDelayMs / 1000);
  }

  console.log(`Processed ${results.processed}/${results.discovered} shipments (${results.failed} failed)`);

  return results;
}

/**
 * Creates receiving tasks for shipment items (for labor assignment)
 * @param {string} shipmentId - The shipment ID
 * @returns {Array} Created receiving tasks
 */
export function createReceivingTasks(shipmentId) {
  const url = `${BASE_URLS.receiving}/api/v1/shipments/${shipmentId}/tasks`;
  const response = http.post(url, null, HTTP_PARAMS);

  const success = check(response, {
    'create receiving tasks status 200': (r) => r.status === 200 || r.status === 201,
  });

  if (!success) {
    console.warn(`Failed to create receiving tasks for ${shipmentId}: ${response.status}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return Array.isArray(data) ? data : (data.tasks || []);
  } catch (e) {
    console.error(`Failed to parse receiving tasks response: ${e.message}`);
    return [];
  }
}

/**
 * Discovers receiving tasks ready for workers
 * @param {string} status - Task status to filter by
 * @returns {Array} Array of receiving tasks
 */
export function discoverReceivingTasks(status = 'pending') {
  const url = `${BASE_URLS.receiving}/api/v1/tasks?status=${status}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'discover receiving tasks status 200': (r) => r.status === 200,
  });

  if (!success || !response.body) {
    console.warn(`Failed to discover receiving tasks: ${response.status}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return Array.isArray(data) ? data : (data.tasks || data.items || []);
  } catch (e) {
    console.error(`Failed to parse receiving tasks response: ${e.message}`);
    return [];
  }
}
