// WMS Platform - Order Injection Script
// Main script for injecting orders into the system

import { sleep, check } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
import { generateOrder } from './lib/data.js';
import { createOrder, checkHealth } from './lib/orders.js';
import { THRESHOLDS } from './lib/config.js';

// Custom metrics
const ordersCreated = new Counter('orders_created');
const ordersFailed = new Counter('orders_failed');
const orderSuccessRate = new Rate('order_success_rate');
const orderDuration = new Trend('order_duration');

export const options = {
  scenarios: {
    orders: {
      executor: 'ramping-vus',
      startVUs: 1,
      stages: [
        { duration: '30s', target: 5 },   // Ramp up to 5 VUs
        { duration: '1m', target: 5 },    // Stay at 5 VUs
        { duration: '30s', target: 0 },   // Ramp down
      ],
      gracefulRampDown: '10s',
    },
  },
  thresholds: {
    ...THRESHOLDS.default,
    'order_success_rate': ['rate>0.95'],
    'order_duration': ['p(95)<1000'],
  },
};

export function setup() {
  // Verify service is healthy before starting
  const healthy = checkHealth();
  if (!healthy) {
    throw new Error('Order service is not healthy!');
  }
  console.log('Order service is healthy, starting load test...');
  return {};
}

export default function () {
  // Generate a random order
  const order = generateOrder();

  // Create the order
  const startTime = Date.now();
  const result = createOrder(order);
  const duration = Date.now() - startTime;

  // Record metrics
  orderDuration.add(duration);

  if (result.success && result.orderId) {
    ordersCreated.add(1);
    orderSuccessRate.add(1);
    console.log(`Order created: ${result.orderId} (${duration}ms)`);
  } else {
    ordersFailed.add(1);
    orderSuccessRate.add(0);
    console.log(`Order failed: ${result.status} - ${JSON.stringify(result.body)}`);
  }

  // Small delay between orders
  sleep(0.5);
}

export function teardown(data) {
  console.log('Load test completed!');
}
