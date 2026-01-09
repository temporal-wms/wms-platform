// WMS Platform - Load Test
// Normal load test to validate system under expected traffic

import { sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
import { generateOrder } from '../lib/data.js';
import { createOrder, checkHealth } from '../lib/orders.js';
import { THRESHOLDS } from '../lib/config.js';

// Custom metrics
const ordersCreated = new Counter('orders_created');
const ordersFailed = new Counter('orders_failed');
const orderSuccessRate = new Rate('order_success_rate');
const orderDuration = new Trend('order_duration');

// Priority distribution tracking
const prioritySameDay = new Counter('priority_same_day');
const priorityNextDay = new Counter('priority_next_day');
const priorityStandard = new Counter('priority_standard');

export const options = {
  scenarios: {
    load_test: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '1m', target: 10 },   // Ramp up to 10 VUs over 1 minute
        { duration: '5m', target: 10 },   // Stay at 10 VUs for 5 minutes
        { duration: '1m', target: 0 },    // Ramp down over 1 minute
      ],
      gracefulRampDown: '30s',
    },
  },
  thresholds: {
    ...THRESHOLDS.default,
    'order_success_rate': ['rate>0.95'],
    'order_duration': ['p(95)<500', 'p(99)<1000'],
    'http_reqs': ['rate>5'],  // At least 5 requests per second
  },
};

export function setup() {
  console.log('=== Load Test ===');
  console.log('');
  console.log('Configuration:');
  console.log('  Ramp up: 1 minute to 10 VUs');
  console.log('  Sustain: 5 minutes at 10 VUs');
  console.log('  Ramp down: 1 minute');
  console.log('');

  // Health check
  const healthy = checkHealth();
  if (!healthy) {
    throw new Error('Order service is not healthy!');
  }
  console.log('Order service is healthy, starting load test...');
  console.log('');

  return { startTime: Date.now() };
}

export default function () {
  // Generate a random order
  const order = generateOrder();

  // Track priority distribution
  switch (order.priority) {
    case 'same_day':
      prioritySameDay.add(1);
      break;
    case 'next_day':
      priorityNextDay.add(1);
      break;
    case 'standard':
      priorityStandard.add(1);
      break;
  }

  // Create the order
  const startTime = Date.now();
  const result = createOrder(order);
  const duration = Date.now() - startTime;

  // Record metrics
  orderDuration.add(duration);

  if (result.success && result.orderId) {
    ordersCreated.add(1);
    orderSuccessRate.add(1);
  } else {
    ordersFailed.add(1);
    orderSuccessRate.add(0);
    console.log(`Order failed: ${result.status}`);
  }

  // Realistic delay between orders (300-700ms)
  sleep(0.3 + Math.random() * 0.4);
}

export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;
  console.log('');
  console.log('=== Load Test Complete ===');
  console.log(`Total duration: ${(duration / 60).toFixed(2)} minutes`);
}
