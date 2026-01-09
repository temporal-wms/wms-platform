// K6 Tracking Library
// Provides functions for querying Temporal workflows and domain entities in real-time

import http from 'k6/http';
import { check } from 'k6';
import { BASE_URLS, HTTP_PARAMS } from './config.js';

// Tracking-specific configuration
export const TRACKING_CONFIG = {
  enabled: __ENV.ENABLE_TRACKING !== 'false',
  logLevel: __ENV.TRACKING_LOG_LEVEL || 'info', // 'debug', 'info', 'warn', 'error'
  pollIntervalMs: parseInt(__ENV.TRACKING_POLL_INTERVAL_MS || '2000'),
  timeoutMs: parseInt(__ENV.TRACKING_TIMEOUT_MS || '30000'),
};

// Temporal Validator base URL (for workflow queries)
const TEMPORAL_VALIDATOR_URL = __ENV.TEMPORAL_VALIDATOR_URL || 'http://localhost:8020';

// =============================================================================
// WORKFLOW TRACKING FUNCTIONS
// =============================================================================

/**
 * Gets the current status of a Temporal workflow
 * @param {string} workflowId - The workflow ID
 * @returns {Object} Workflow status {workflowId, runId, status, isRunning}
 */
export function getWorkflowStatus(workflowId) {
  if (!TRACKING_CONFIG.enabled) {
    return { workflowId, status: 'tracking_disabled', isRunning: false };
  }

  const url = `${TEMPORAL_VALIDATOR_URL}/workflows/${workflowId}/status`;
  const response = http.get(url, HTTP_PARAMS);

  if (response.status === 200) {
    try {
      return JSON.parse(response.body);
    } catch (e) {
      return { workflowId, status: 'parse_error', isRunning: false };
    }
  } else if (response.status === 404) {
    return { workflowId, status: 'not_found', isRunning: false };
  }

  return { workflowId, status: 'error', isRunning: false, error: response.status };
}

/**
 * Gets detailed description of a Temporal workflow
 * @param {string} workflowId - The workflow ID
 * @param {string} runId - Optional run ID
 * @returns {Object} Workflow description with timing and type info
 */
export function getWorkflowDescription(workflowId, runId = '') {
  if (!TRACKING_CONFIG.enabled) {
    return { workflowId, status: 'tracking_disabled' };
  }

  let url = `${TEMPORAL_VALIDATOR_URL}/workflows/${workflowId}/describe`;
  if (runId) {
    url += `?runId=${runId}`;
  }

  const response = http.get(url, HTTP_PARAMS);

  if (response.status === 200) {
    try {
      return JSON.parse(response.body);
    } catch (e) {
      return { workflowId, status: 'parse_error' };
    }
  }

  return { workflowId, status: 'error', error: response.status };
}

/**
 * Gets signals delivered to a workflow
 * @param {string} workflowId - The workflow ID
 * @returns {Object} Signal information {workflowId, signalCount, signals[]}
 */
export function getWorkflowSignals(workflowId) {
  if (!TRACKING_CONFIG.enabled) {
    return { workflowId, signalCount: 0, signals: [] };
  }

  const url = `${TEMPORAL_VALIDATOR_URL}/workflows/${workflowId}/signals`;
  const response = http.get(url, HTTP_PARAMS);

  if (response.status === 200) {
    try {
      return JSON.parse(response.body);
    } catch (e) {
      return { workflowId, signalCount: 0, signals: [] };
    }
  }

  return { workflowId, signalCount: 0, signals: [], error: response.status };
}

/**
 * Queries a running workflow for its internal state
 * @param {string} workflowId - The workflow ID
 * @param {string} queryType - The query type (e.g., 'getStatus')
 * @param {any} args - Query arguments
 * @returns {Object} Query result
 */
export function queryWorkflow(workflowId, queryType = 'getStatus', args = null) {
  if (!TRACKING_CONFIG.enabled) {
    return { workflowId, queryType, result: null };
  }

  const url = `${TEMPORAL_VALIDATOR_URL}/workflows/${workflowId}/query`;
  const payload = JSON.stringify({
    queryType: queryType,
    args: args,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  if (response.status === 200) {
    try {
      return JSON.parse(response.body);
    } catch (e) {
      return { workflowId, queryType, result: null, error: 'parse_error' };
    }
  }

  return { workflowId, queryType, result: null, error: response.status };
}

/**
 * Queries the order fulfillment workflow state
 * @param {string} orderId - The order ID
 * @returns {Object} Workflow state {currentStage, completionPercent, status}
 */
export function queryOrderFulfillmentState(orderId) {
  const workflowId = `order-fulfillment-${orderId}`;
  const result = queryWorkflow(workflowId, 'getStatus');

  if (result.result) {
    return {
      orderId,
      workflowId,
      currentStage: result.result.CurrentStage || result.result.currentStage,
      completionPercent: result.result.CompletionPercent || result.result.completionPercent || 0,
      status: result.result.Status || result.result.status,
      completedStages: result.result.CompletedStages || result.result.completedStages || 0,
      totalStages: result.result.TotalStages || result.result.totalStages || 5,
    };
  }

  return { orderId, workflowId, currentStage: 'unknown', completionPercent: 0, status: 'unknown' };
}

// =============================================================================
// ORDER TRACKING FUNCTIONS
// =============================================================================

/**
 * Gets detailed order information
 * @param {string} orderId - The order ID
 * @returns {Object} Order details
 */
export function getOrderDetails(orderId) {
  if (!TRACKING_CONFIG.enabled) {
    return { orderId, status: 'tracking_disabled' };
  }

  const url = `${BASE_URLS.orders}/api/v1/orders/${orderId}`;
  const response = http.get(url, HTTP_PARAMS);

  if (response.status === 200) {
    try {
      const order = JSON.parse(response.body);
      return {
        orderId: order.orderId || order.id,
        status: order.status,
        customerId: order.customerId,
        itemCount: order.items?.length || 0,
        items: order.items || [],
        priority: order.priority,
        waveId: order.waveId,
        trackingNumber: order.trackingNumber,
        requirements: order.requirements || [],
        createdAt: order.createdAt,
        updatedAt: order.updatedAt,
      };
    } catch (e) {
      return { orderId, status: 'parse_error' };
    }
  }

  return { orderId, status: 'not_found' };
}

/**
 * Gets order timeline with stage transitions
 * @param {string} orderId - The order ID
 * @returns {Object} Order timeline
 */
export function getOrderTimeline(orderId) {
  const order = getOrderDetails(orderId);
  const workflow = queryOrderFulfillmentState(orderId);

  return {
    orderId,
    orderStatus: order.status,
    workflowStage: workflow.currentStage,
    workflowProgress: workflow.completionPercent,
    timestamps: {
      created: order.createdAt,
      updated: order.updatedAt,
    },
  };
}

// =============================================================================
// UNIT TRACKING FUNCTIONS
// =============================================================================

/**
 * Gets units associated with an order
 * @param {string} orderId - The order ID
 * @returns {Array} Units for the order
 */
export function getUnitsForOrder(orderId) {
  if (!TRACKING_CONFIG.enabled) {
    return [];
  }

  const url = `${BASE_URLS.unit}/api/v1/units/order/${orderId}`;
  const response = http.get(url, HTTP_PARAMS);

  if (response.status === 200) {
    try {
      const data = JSON.parse(response.body);
      const units = Array.isArray(data) ? data : (data.units || []);

      return units.map(unit => ({
        unitId: unit.unitId || unit.id,
        sku: unit.sku,
        status: unit.status,
        currentLocation: unit.currentLocationId,
        toteId: unit.toteId,
        routeId: unit.routeId,
        movementCount: unit.movements?.length || 0,
        movements: unit.movements || [],
        timestamps: {
          received: unit.receivedAt,
          reserved: unit.reservedAt,
          staged: unit.stagedAt,
          picked: unit.pickedAt,
          consolidated: unit.consolidatedAt,
          packed: unit.packedAt,
          shipped: unit.shippedAt,
        },
      }));
    } catch (e) {
      return [];
    }
  }

  return [];
}

/**
 * Gets movement history for a specific unit
 * @param {string} unitId - The unit ID
 * @returns {Object} Unit with movement history
 */
export function getUnitMovementHistory(unitId) {
  if (!TRACKING_CONFIG.enabled) {
    return { unitId, movements: [] };
  }

  const url = `${BASE_URLS.unit}/api/v1/units/${unitId}`;
  const response = http.get(url, HTTP_PARAMS);

  if (response.status === 200) {
    try {
      const unit = JSON.parse(response.body);
      return {
        unitId: unit.unitId || unit.id,
        sku: unit.sku,
        status: unit.status,
        movements: (unit.movements || []).map(m => ({
          movementId: m.movementId,
          fromLocation: m.fromLocationId,
          toLocation: m.toLocationId,
          fromStatus: m.fromStatus,
          toStatus: m.toStatus,
          stationId: m.stationId,
          handlerId: m.handlerId,
          timestamp: m.timestamp,
          notes: m.notes,
        })),
      };
    } catch (e) {
      return { unitId, movements: [] };
    }
  }

  return { unitId, movements: [] };
}

/**
 * Summarizes unit statuses for an order
 * @param {string} orderId - The order ID
 * @returns {Object} Unit status summary
 */
export function getUnitStatusSummary(orderId) {
  const units = getUnitsForOrder(orderId);

  const statusCounts = {};
  for (const unit of units) {
    statusCounts[unit.status] = (statusCounts[unit.status] || 0) + 1;
  }

  return {
    orderId,
    totalUnits: units.length,
    statusCounts,
    statuses: units.map(u => ({ unitId: u.unitId, status: u.status })),
  };
}

// =============================================================================
// INVENTORY TRACKING FUNCTIONS
// =============================================================================

/**
 * Gets inventory state for a SKU
 * @param {string} sku - The SKU
 * @returns {Object} Inventory state
 */
export function getInventoryState(sku) {
  if (!TRACKING_CONFIG.enabled) {
    return { sku, available: 0, reserved: 0 };
  }

  const url = `${BASE_URLS.inventory}/api/v1/inventory/${sku}`;
  const response = http.get(url, HTTP_PARAMS);

  if (response.status === 200) {
    try {
      const inv = JSON.parse(response.body);
      return {
        sku: inv.sku,
        totalQuantity: inv.totalQuantity || 0,
        reservedQuantity: inv.reservedQuantity || 0,
        hardAllocatedQuantity: inv.hardAllocatedQuantity || 0,
        availableQuantity: inv.availableQuantity || 0,
        locations: (inv.locations || []).map(loc => ({
          locationId: loc.locationId,
          zone: loc.zone,
          quantity: loc.quantity,
          reserved: loc.reserved,
          available: loc.available,
        })),
        reservationCount: inv.reservations?.length || 0,
        allocationCount: inv.hardAllocations?.length || 0,
      };
    } catch (e) {
      return { sku, available: 0, reserved: 0, error: 'parse_error' };
    }
  }

  return { sku, available: 0, reserved: 0, notFound: true };
}

/**
 * Gets inventory reservations for an order
 * @param {string} orderId - The order ID
 * @returns {Array} Reservations for the order
 */
export function getInventoryReservationsForOrder(orderId) {
  if (!TRACKING_CONFIG.enabled) {
    return [];
  }

  const url = `${BASE_URLS.inventory}/api/v1/reservations/order/${orderId}`;
  const response = http.get(url, HTTP_PARAMS);

  if (response.status === 200) {
    try {
      const data = JSON.parse(response.body);
      const reservations = Array.isArray(data) ? data : (data.reservations || []);
      return reservations.map(r => ({
        reservationId: r.reservationId,
        sku: r.sku,
        quantity: r.quantity,
        locationId: r.locationId,
        status: r.status,
        createdAt: r.createdAt,
      }));
    } catch (e) {
      return [];
    }
  }

  return [];
}

/**
 * Gets inventory state for all SKUs in an order
 * @param {string} orderId - The order ID
 * @returns {Object} Inventory state for order items
 */
export function getInventoryForOrder(orderId) {
  const order = getOrderDetails(orderId);
  if (!order.items || order.items.length === 0) {
    return { orderId, items: [], reservations: [] };
  }

  const inventoryStates = [];
  for (const item of order.items) {
    const state = getInventoryState(item.sku);
    inventoryStates.push({
      sku: item.sku,
      orderedQuantity: item.quantity,
      ...state,
    });
  }

  const reservations = getInventoryReservationsForOrder(orderId);

  return {
    orderId,
    items: inventoryStates,
    reservations,
    totalReserved: reservations.reduce((sum, r) => sum + r.quantity, 0),
  };
}

// =============================================================================
// SHIPMENT TRACKING FUNCTIONS
// =============================================================================

/**
 * Gets shipment for an order
 * @param {string} orderId - The order ID
 * @returns {Object} Shipment details
 */
export function getShipmentForOrder(orderId) {
  if (!TRACKING_CONFIG.enabled) {
    return { orderId, status: 'tracking_disabled' };
  }

  const url = `${BASE_URLS.shipping}/api/v1/shipments/order/${orderId}`;
  const response = http.get(url, HTTP_PARAMS);

  if (response.status === 200) {
    try {
      const data = JSON.parse(response.body);
      const shipment = Array.isArray(data) ? data[0] : data;

      if (!shipment) {
        return { orderId, status: 'no_shipment' };
      }

      return {
        orderId,
        shipmentId: shipment.shipmentId || shipment.id,
        status: shipment.status,
        carrier: shipment.carrier?.code || shipment.carrierCode,
        trackingNumber: shipment.label?.trackingNumber || shipment.trackingNumber,
        packageWeight: shipment.package?.weight,
        timestamps: {
          created: shipment.createdAt,
          labeled: shipment.labeledAt,
          manifested: shipment.manifestedAt,
          shipped: shipment.shippedAt,
          estimatedDelivery: shipment.estimatedDelivery,
        },
      };
    } catch (e) {
      return { orderId, status: 'parse_error' };
    }
  }

  return { orderId, status: 'not_found' };
}

// =============================================================================
// COMPREHENSIVE STATE CAPTURE
// =============================================================================

/**
 * Captures complete state for an order (all entities)
 * @param {string} orderId - The order ID
 * @returns {Object} Complete entity state
 */
export function captureOrderState(orderId) {
  if (!TRACKING_CONFIG.enabled) {
    return { orderId, trackingEnabled: false };
  }

  const workflow = queryOrderFulfillmentState(orderId);
  const order = getOrderDetails(orderId);
  const units = getUnitsForOrder(orderId);
  const inventory = getInventoryForOrder(orderId);
  const shipment = getShipmentForOrder(orderId);

  return {
    orderId,
    capturedAt: new Date().toISOString(),
    workflow: {
      workflowId: workflow.workflowId,
      currentStage: workflow.currentStage,
      completionPercent: workflow.completionPercent,
      status: workflow.status,
    },
    order: {
      status: order.status,
      itemCount: order.itemCount,
      priority: order.priority,
      waveId: order.waveId,
    },
    units: {
      total: units.length,
      byStatus: summarizeByStatus(units),
      totalMovements: units.reduce((sum, u) => sum + u.movementCount, 0),
    },
    inventory: {
      itemsTracked: inventory.items?.length || 0,
      totalReserved: inventory.totalReserved || 0,
      reservationCount: inventory.reservations?.length || 0,
    },
    shipment: {
      status: shipment.status,
      carrier: shipment.carrier,
      trackingNumber: shipment.trackingNumber,
    },
  };
}

/**
 * Helper: Summarize items by status
 */
function summarizeByStatus(items) {
  const summary = {};
  for (const item of items) {
    const status = item.status || 'unknown';
    summary[status] = (summary[status] || 0) + 1;
  }
  return summary;
}

// =============================================================================
// LOGGING UTILITIES
// =============================================================================

/**
 * Logs tracking state with formatted output
 * @param {string} stageName - Current stage name
 * @param {Object} state - State object from captureOrderState
 */
export function logTrackingState(stageName, state) {
  if (!TRACKING_CONFIG.enabled) return;

  const level = TRACKING_CONFIG.logLevel;
  if (level === 'error') return;

  console.log(`[TRACKING] Stage: ${stageName}`);
  console.log(`  Workflow: ${state.workflow.workflowId} | Status: ${state.workflow.status} | Progress: ${state.workflow.completionPercent}%`);
  console.log(`  Order: ${state.orderId} | Status: ${state.order.status} | Items: ${state.order.itemCount}`);
  console.log(`  Units: ${state.units.total} units | Statuses: ${JSON.stringify(state.units.byStatus)}`);
  console.log(`  Inventory: ${state.inventory.itemsTracked} SKUs tracked | Reserved: ${state.inventory.totalReserved}`);
  console.log(`  Shipment: ${state.shipment.status} | Carrier: ${state.shipment.carrier || 'N/A'}`);
}

/**
 * Logs state changes between two snapshots
 * @param {string} fromStage - Previous stage
 * @param {string} toStage - Current stage
 * @param {Object} before - Previous state
 * @param {Object} after - Current state
 */
export function logStateChange(fromStage, toStage, before, after) {
  if (!TRACKING_CONFIG.enabled) return;

  console.log(`[TRACKING] Stage Change: ${fromStage} → ${toStage}`);

  // Workflow progress
  const progressDelta = (after.workflow.completionPercent || 0) - (before.workflow.completionPercent || 0);
  if (progressDelta > 0) {
    console.log(`  Workflow Progress: ${before.workflow.completionPercent}% → ${after.workflow.completionPercent}%`);
  }

  // Order status
  if (before.order.status !== after.order.status) {
    console.log(`  Order Status: ${before.order.status} → ${after.order.status}`);
  }

  // Unit movements
  const movementDelta = (after.units.totalMovements || 0) - (before.units.totalMovements || 0);
  if (movementDelta > 0) {
    console.log(`  Unit Movements: +${movementDelta} new movements`);
  }

  // Inventory changes
  const reservationDelta = (after.inventory.reservationCount || 0) - (before.inventory.reservationCount || 0);
  if (reservationDelta !== 0) {
    console.log(`  Inventory Reservations: ${reservationDelta > 0 ? '+' : ''}${reservationDelta}`);
  }

  // Shipment status
  if (before.shipment.status !== after.shipment.status) {
    console.log(`  Shipment: ${before.shipment.status} → ${after.shipment.status}`);
  }
}
