// Orders API helpers for K6 load testing
import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URLS, ENDPOINTS, HTTP_PARAMS, GIFTWRAP_CONFIG, ORDER_CONFIG, FLOW_CONFIG } from './config.js';
import { generateOrderWithType, aggregateOrderRequirements } from './data.js';

const baseUrl = BASE_URLS.orders;

export function createOrder(order) {
  const payload = JSON.stringify(order);

  const response = http.post(
    `${baseUrl}${ENDPOINTS.orders.create}`,
    payload,
    HTTP_PARAMS
  );

  const success = check(response, {
    'order created': (r) => r.status === 201 || r.status === 200,
    'order has id': (r) => {
      try {
        const body = r.json();
        return body.order && body.order.orderId;
      } catch {
        return false;
      }
    },
  });

  let body = null;
  try {
    body = response.json();
  } catch {
    body = { error: response.body };
  }

  return {
    success,
    status: response.status,
    body,
    orderId: body?.order?.orderId,
    workflowId: body?.workflowId,
  };
}

export function getOrder(orderId) {
  const response = http.get(
    `${baseUrl}${ENDPOINTS.orders.get(orderId)}`,
    HTTP_PARAMS
  );

  const success = check(response, {
    'order retrieved': (r) => r.status === 200,
  });

  return {
    success,
    status: response.status,
    body: response.json(),
  };
}

/**
 * Wait for an order to reach a specific status
 * @param {string} orderId - Order ID to check
 * @param {string|string[]} expectedStatus - Expected status(es)
 * @param {number} timeoutMs - Timeout in milliseconds
 * @param {number} intervalMs - Polling interval in milliseconds
 * @returns {Object} { success: boolean, order: Object, finalStatus: string }
 */
export function waitForOrderStatus(orderId, expectedStatus, timeoutMs = 120000, intervalMs = 3000) {
  const startTime = Date.now();
  const expectedStatuses = Array.isArray(expectedStatus) ? expectedStatus : [expectedStatus];

  while (Date.now() - startTime < timeoutMs) {
    const result = getOrder(orderId);

    // API returns order directly, not wrapped in { order: {...} }
    if (result.success && result.body?.status) {
      const currentStatus = result.body.status;

      if (expectedStatuses.includes(currentStatus)) {
        return {
          success: true,
          order: result.body,
          finalStatus: currentStatus,
        };
      }

      // Check for terminal failure states
      if (['cancelled', 'failed'].includes(currentStatus)) {
        console.warn(`Order ${orderId} reached terminal state: ${currentStatus}`);
        return {
          success: false,
          order: result.body,
          finalStatus: currentStatus,
        };
      }
    }

    sleep(intervalMs / 1000);
  }

  console.warn(`Timeout waiting for order ${orderId} to reach status: ${expectedStatuses.join(', ')}`);
  return { success: false, order: null, finalStatus: 'timeout' };
}

/**
 * Wait for all orders to reach expected status
 * @param {string[]} orderIds - Array of order IDs
 * @param {string|string[]} expectedStatus - Expected status(es)
 * @param {number} timeoutMs - Timeout per order
 * @param {number} intervalMs - Polling interval
 * @returns {Object} { allSuccess: boolean, results: Object[] }
 */
export function waitForAllOrdersStatus(orderIds, expectedStatus, timeoutMs = 120000, intervalMs = 3000) {
  const results = [];
  let allSuccess = true;

  for (const orderId of orderIds) {
    const result = waitForOrderStatus(orderId, expectedStatus, timeoutMs, intervalMs);
    results.push({ orderId, ...result });

    if (!result.success) {
      allSuccess = false;
    }
  }

  return { allSuccess, results };
}

export function listOrders(limit, offset) {
  let url = `${baseUrl}${ENDPOINTS.orders.list}`;
  const params = [];
  if (limit) params.push(`limit=${limit}`);
  if (offset) params.push(`offset=${offset}`);
  if (params.length > 0) url += `?${params.join('&')}`;

  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'orders listed': (r) => r.status === 200,
  });

  return {
    success,
    status: response.status,
    body: response.json(),
  };
}

export function validateOrder(orderId) {
  const response = http.put(
    `${baseUrl}${ENDPOINTS.orders.validate(orderId)}`,
    '{}',
    HTTP_PARAMS
  );

  const success = check(response, {
    'order validated': (r) => r.status === 200,
  });

  return {
    success,
    status: response.status,
    body: response.json(),
  };
}

export function cancelOrder(orderId) {
  const response = http.put(
    `${baseUrl}${ENDPOINTS.orders.cancel(orderId)}`,
    '{}',
    HTTP_PARAMS
  );

  const success = check(response, {
    'order cancelled': (r) => r.status === 200,
  });

  return {
    success,
    status: response.status,
    body: response.json(),
  };
}

export function checkHealth() {
  const response = http.get(`${baseUrl}/health`);

  return check(response, {
    'order service healthy': (r) => r.status === 200,
  });
}

export function checkReady() {
  const response = http.get(`${baseUrl}/ready`);

  return check(response, {
    'order service ready': (r) => r.status === 200,
  });
}

// ============================================================================
// Gift Wrap Order Generation
// ============================================================================

const GIFT_WRAP_TYPES = ['standard', 'premium', 'holiday', 'birthday', 'wedding'];
const GIFT_MESSAGES = [
  'Happy Birthday!',
  'Congratulations!',
  'With love',
  'Best wishes',
  'Thinking of you',
  'Thank you!',
  'Happy Anniversary!',
  'Happy Holidays!',
  null,  // Some orders without message
];

/**
 * Pick a random element from an array
 */
function randomChoice(arr) {
  return arr[Math.floor(Math.random() * arr.length)];
}

/**
 * Generate random gift wrap details
 */
export function generateGiftWrapDetails() {
  return {
    wrapType: randomChoice(GIFT_WRAP_TYPES),
    giftMessage: randomChoice(GIFT_MESSAGES),
    hidePrice: Math.random() < 0.7,  // 70% want price hidden
    includeReceipt: Math.random() < 0.3,  // 30% want gift receipt
  };
}

/**
 * Add gift wrap details to an existing order object
 */
export function addGiftWrapToOrder(order) {
  return {
    ...order,
    giftWrap: true,
    giftWrapDetails: generateGiftWrapDetails(),
  };
}

/**
 * Create a gift wrap order
 */
export function createGiftWrapOrder(order) {
  const giftWrapOrder = addGiftWrapToOrder(order);
  return createOrder(giftWrapOrder);
}

/**
 * Determine if an order should have gift wrap based on configured ratio
 */
export function shouldHaveGiftWrap() {
  return Math.random() < GIFTWRAP_CONFIG.giftWrapOrderRatio;
}

/**
 * Create order with optional gift wrap based on configured ratio
 */
export function createOrderWithOptionalGiftWrap(order) {
  if (shouldHaveGiftWrap()) {
    console.log('Creating gift wrap order');
    return createGiftWrapOrder(order);
  }
  return createOrder(order);
}

// ============================================================================
// Requirement-Based Order Generation
// ============================================================================

/**
 * Create an order with specific type and requirements
 * @param {string|null} orderType - 'single', 'multi', or null for random
 * @param {string|null} requirement - Force specific requirement (hazmat, fragile, etc.)
 * @param {boolean} includeGiftWrap - Whether to potentially include gift wrap
 * @returns {Object} Order creation result
 */
export function createTypedOrder(orderType = null, requirement = null, includeGiftWrap = true) {
  const order = generateOrderWithType(orderType, requirement);

  // Optionally add gift wrap
  if (includeGiftWrap && shouldHaveGiftWrap()) {
    console.log(`Creating ${order.orderType} order with gift wrap, requirements: [${order.requirements.join(', ')}]`);
    return createGiftWrapOrder(order);
  }

  console.log(`Creating ${order.orderType} order, requirements: [${order.requirements.join(', ')}]`);
  return createOrder(order);
}

/**
 * Create a single-item order
 * @param {string|null} requirement - Optional requirement
 * @returns {Object} Order creation result
 */
export function createSingleItemOrder(requirement = null) {
  return createTypedOrder('single', requirement);
}

/**
 * Create a multi-item order
 * @param {string|null} requirement - Optional requirement
 * @returns {Object} Order creation result
 */
export function createMultiItemOrder(requirement = null) {
  return createTypedOrder('multi', requirement);
}

/**
 * Create an order with hazmat items
 * @param {string|null} orderType - 'single', 'multi', or null
 * @returns {Object} Order creation result
 */
export function createHazmatOrder(orderType = null) {
  return createTypedOrder(orderType, 'hazmat');
}

/**
 * Create an order with fragile items
 * @param {string|null} orderType - 'single', 'multi', or null
 * @returns {Object} Order creation result
 */
export function createFragileOrder(orderType = null) {
  return createTypedOrder(orderType, 'fragile');
}

/**
 * Create an order with oversized items
 * @param {string|null} orderType - 'single', 'multi', or null
 * @returns {Object} Order creation result
 */
export function createOversizedOrder(orderType = null) {
  return createTypedOrder(orderType, 'oversized');
}

/**
 * Create an order with high-value items
 * @param {string|null} orderType - 'single', 'multi', or null
 * @returns {Object} Order creation result
 */
export function createHighValueOrder(orderType = null) {
  return createTypedOrder(orderType, 'high_value');
}

/**
 * Create an order with heavy items
 * @param {string|null} orderType - 'single', 'multi', or null
 * @returns {Object} Order creation result
 */
export function createHeavyOrder(orderType = null) {
  return createTypedOrder(orderType, 'heavy');
}

/**
 * Create orders with a specific distribution of types and requirements
 * Useful for load testing with realistic distribution
 * @param {number} count - Number of orders to create
 * @returns {Array} Array of order creation results
 */
export function createOrderBatch(count) {
  const results = [];

  for (let i = 0; i < count; i++) {
    // Use configured distribution (respects FORCE_ORDER_TYPE and FORCE_REQUIREMENT env vars)
    const result = createTypedOrder(null, null, true);
    results.push(result);
  }

  return results;
}

/**
 * Create orders with explicit distribution
 * @param {Object} distribution - { single: 4, multi: 6, hazmat: 2, fragile: 2, etc. }
 * @returns {Array} Array of order creation results
 */
export function createOrdersWithDistribution(distribution) {
  const results = [];

  // Process single/multi types
  if (distribution.single) {
    for (let i = 0; i < distribution.single; i++) {
      results.push(createSingleItemOrder());
    }
  }

  if (distribution.multi) {
    for (let i = 0; i < distribution.multi; i++) {
      results.push(createMultiItemOrder());
    }
  }

  // Process requirement-based orders
  const requirementTypes = ['hazmat', 'fragile', 'oversized', 'heavy', 'high_value'];
  for (const req of requirementTypes) {
    if (distribution[req]) {
      for (let i = 0; i < distribution[req]; i++) {
        results.push(createTypedOrder(null, req));
      }
    }
  }

  return results;
}
