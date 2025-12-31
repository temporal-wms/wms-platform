// Orders API helpers for K6 load testing
import http from 'k6/http';
import { check } from 'k6';
import { BASE_URLS, ENDPOINTS, HTTP_PARAMS, GIFTWRAP_CONFIG } from './config.js';

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
