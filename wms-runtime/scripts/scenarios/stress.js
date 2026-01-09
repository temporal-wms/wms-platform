// WMS Platform - Stress Test
// Heavy load test to find system breaking points

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

// Error tracking
const errorsByStatus = new Counter('errors_by_status');

export const options = {
  scenarios: {
    stress_test: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '2m', target: 10 },   // Warm up
        { duration: '3m', target: 25 },   // Ramp to moderate load
        { duration: '3m', target: 50 },   // Ramp to high load
        { duration: '3m', target: 75 },   // Ramp to very high load
        { duration: '2m', target: 100 },  // Peak load
        { duration: '2m', target: 100 },  // Sustain peak
        { duration: '2m', target: 50 },   // Scale down
        { duration: '2m', target: 0 },    // Recovery
      ],
      gracefulRampDown: '30s',
    },
  },
  thresholds: {
    // More relaxed thresholds for stress testing
    ...THRESHOLDS.relaxed,
    'order_success_rate': ['rate>0.80'],  // Allow up to 20% failure under stress
    'order_duration': ['p(95)<3000'],     // Allow up to 3s response time
  },
};

export function setup() {
  console.log('=== Stress Test ===');
  console.log('');
  console.log('WARNING: This test will push the system to its limits!');
  console.log('');
  console.log('Load Profile:');
  console.log('  Stage 1 (0-2m):   Warm up to 10 VUs');
  console.log('  Stage 2 (2-5m):   Ramp to 25 VUs');
  console.log('  Stage 3 (5-8m):   Ramp to 50 VUs');
  console.log('  Stage 4 (8-11m):  Ramp to 75 VUs');
  console.log('  Stage 5 (11-13m): Peak at 100 VUs');
  console.log('  Stage 6 (13-15m): Sustain peak');
  console.log('  Stage 7 (15-17m): Scale down to 50 VUs');
  console.log('  Stage 8 (17-19m): Recovery');
  console.log('');
  console.log('Total duration: ~19 minutes');
  console.log('');

  // Health check
  const healthy = checkHealth();
  if (!healthy) {
    throw new Error('Order service is not healthy!');
  }
  console.log('Order service is healthy, starting stress test...');
  console.log('');

  return { startTime: Date.now() };
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
  } else {
    ordersFailed.add(1);
    orderSuccessRate.add(0);

    // Track error status codes
    errorsByStatus.add(1, { status: result.status.toString() });

    // Log errors (but not too many)
    if (Math.random() < 0.1) {
      console.log(`Error: ${result.status} - VU ${__VU}`);
    }
  }

  // Minimal delay for maximum stress
  sleep(0.1 + Math.random() * 0.2);
}

export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;
  console.log('');
  console.log('=== Stress Test Complete ===');
  console.log(`Total duration: ${(duration / 60).toFixed(2)} minutes`);
  console.log('');
  console.log('Review the metrics to identify:');
  console.log('  - Maximum sustainable throughput');
  console.log('  - Response time degradation patterns');
  console.log('  - Error rate thresholds');
  console.log('  - Resource bottlenecks');
}
