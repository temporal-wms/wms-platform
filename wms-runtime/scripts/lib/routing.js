// K6 Routing Service Helper Library
// Provides functions for multi-route calculation and route management

import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URLS, ENDPOINTS, HTTP_PARAMS, MULTI_ROUTE_CONFIG } from './config.js';

/**
 * Calculates multi-route for an order (splits by zone and capacity)
 * @param {string} orderId - The order ID
 * @param {Array} items - Array of items with {sku, quantity, locationId}
 * @returns {Object} Multi-route result with routes array
 */
export function calculateMultiRoute(orderId, items) {
  const url = `${BASE_URLS.routing}${ENDPOINTS.routing.calculateMulti}`;
  const payload = JSON.stringify({
    orderId: orderId,
    items: items.map(item => ({
      sku: item.sku,
      quantity: item.quantity || 1,
      locationId: item.locationId || '',
    })),
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'calculate multi-route status 200/201': (r) => r.status === 200 || r.status === 201,
  });

  if (!success) {
    console.warn(`Failed to calculate multi-route for order ${orderId}: ${response.status} - ${response.body}`);
    return {
      success: false,
      routes: [],
      totalRoutes: 0,
      splitReason: 'none',
    };
  }

  try {
    const data = JSON.parse(response.body);
    return {
      success: true,
      orderId: data.orderId,
      routes: data.routes || [],
      totalRoutes: data.totalRoutes || (data.routes?.length || 0),
      splitReason: data.splitReason || 'none',
      zoneBreakdown: data.zoneBreakdown || {},
      totalItems: data.totalItems || 0,
    };
  } catch (e) {
    console.error(`Failed to parse multi-route response: ${e.message}`);
    return {
      success: false,
      routes: [],
      totalRoutes: 0,
      splitReason: 'none',
    };
  }
}

/**
 * Gets route by ID
 * @param {string} routeId - The route ID
 * @returns {Object|null} Route details or null if not found
 */
export function getRoute(routeId) {
  const url = `${BASE_URLS.routing}${ENDPOINTS.routing.get(routeId)}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get route status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to get route ${routeId}: ${response.status}`);
    return null;
  }

  try {
    return JSON.parse(response.body);
  } catch (e) {
    console.error(`Failed to parse route response: ${e.message}`);
    return null;
  }
}

/**
 * Gets all routes for an order
 * @param {string} orderId - The order ID
 * @returns {Array} Array of routes for the order
 */
export function getRoutesByOrder(orderId) {
  const url = `${BASE_URLS.routing}${ENDPOINTS.routing.byOrder(orderId)}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get routes by order status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to get routes for order ${orderId}: ${response.status}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return Array.isArray(data) ? data : (data.routes || []);
  } catch (e) {
    console.error(`Failed to parse routes response: ${e.message}`);
    return [];
  }
}

/**
 * Checks if an order should use multi-route based on item count
 * @param {number} itemCount - Total number of items in the order
 * @returns {boolean} True if order should use multi-route
 */
export function shouldUseMultiRoute(itemCount) {
  if (!MULTI_ROUTE_CONFIG.enableMultiRoute) {
    return false;
  }
  return itemCount > MULTI_ROUTE_CONFIG.maxItemsPerRoute;
}

/**
 * Gets tote IDs from multi-route result for consolidation tracking
 * @param {Object} multiRouteResult - Result from calculateMultiRoute
 * @returns {Array} Array of expected tote IDs
 */
export function getExpectedTotesFromRoutes(multiRouteResult) {
  if (!multiRouteResult.success || !multiRouteResult.routes) {
    return [];
  }

  return multiRouteResult.routes.map((route, index) => {
    return route.sourceToteId || `TOTE-${multiRouteResult.orderId}-R${index}`;
  });
}

/**
 * Generates a summary of multi-route split for logging
 * @param {Object} multiRouteResult - Result from calculateMultiRoute
 * @returns {string} Human-readable summary
 */
export function getMultiRouteSummary(multiRouteResult) {
  if (!multiRouteResult.success) {
    return 'Multi-route calculation failed';
  }

  if (multiRouteResult.totalRoutes <= 1) {
    return 'Single route (no split needed)';
  }

  const zoneInfo = multiRouteResult.zoneBreakdown
    ? Object.entries(multiRouteResult.zoneBreakdown)
        .map(([zone, count]) => `${zone}:${count}`)
        .join(', ')
    : 'N/A';

  return `Split into ${multiRouteResult.totalRoutes} routes (${multiRouteResult.splitReason}), zones: [${zoneInfo}]`;
}

/**
 * Gets pick tasks by route ID
 * @param {string} routeId - The route ID
 * @returns {Array} Array of pick tasks for the route
 */
export function getPickTasksByRoute(routeId) {
  const url = `${BASE_URLS.picking}${ENDPOINTS.picking.byRoute(routeId)}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get pick tasks by route status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to get pick tasks for route ${routeId}: ${response.status}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return Array.isArray(data) ? data : (data.tasks || []);
  } catch (e) {
    console.error(`Failed to parse pick tasks response: ${e.message}`);
    return [];
  }
}
