// WMS Platform - Smoke Test
// Light load test to validate basic functionality

import { sleep } from 'k6';
import { Counter, Rate } from 'k6/metrics';
import { generateOrder } from '../lib/data.js';
import { createOrder, getOrder, checkHealth } from '../lib/orders.js';
import { checkHealth as checkInventoryHealth } from '../lib/inventory.js';
import { checkHealth as checkLaborHealth } from '../lib/labor.js';

// Custom metrics
const ordersCreated = new Counter('orders_created');
const ordersFailed = new Counter('orders_failed');
const orderSuccessRate = new Rate('order_success_rate');

export const options = {
  vus: 1,
  iterations: 5,
  thresholds: {
    http_req_duration: ['p(95)<1000'],
    http_req_failed: ['rate<0.1'],
    'order_success_rate': ['rate>0.8'],
  },
};

export function setup() {
  console.log('=== Smoke Test ===');
  console.log('');

  // Health checks
  console.log('Checking service health...');
  const ordersHealthy = checkHealth();
  const inventoryHealthy = checkInventoryHealth();
  const laborHealthy = checkLaborHealth();

  console.log(`  Order Service: ${ordersHealthy ? 'OK' : 'FAILED'}`);
  console.log(`  Inventory Service: ${inventoryHealthy ? 'OK' : 'FAILED'}`);
  console.log(`  Labor Service: ${laborHealthy ? 'OK' : 'FAILED'}`);

  if (!ordersHealthy) {
    throw new Error('Order service is not healthy!');
  }

  console.log('');
  console.log('Starting smoke test with 5 orders...');
  console.log('');

  return { startTime: Date.now() };
}

export default function () {
  // Generate and create order
  const order = generateOrder();
  console.log(`Creating order for customer: ${order.customerId}`);
  console.log(`  Items: ${order.items.length}`);
  console.log(`  Priority: ${order.priority}`);

  const result = createOrder(order);

  if (result.success && result.orderId) {
    ordersCreated.add(1);
    orderSuccessRate.add(1);
    console.log(`  SUCCESS: Order ${result.orderId} created`);

    // Verify order can be retrieved
    sleep(0.5);
    const getResult = getOrder(result.orderId);
    if (getResult.success) {
      console.log(`  VERIFIED: Order ${result.orderId} retrieved (status: ${getResult.body?.status})`);
    }

    if (result.workflowId) {
      console.log(`  Workflow: ${result.workflowId}`);
    }
  } else {
    ordersFailed.add(1);
    orderSuccessRate.add(0);
    console.log(`  FAILED: ${result.status}`);
    console.log(`  Error: ${JSON.stringify(result.body)}`);
  }

  console.log('');
  sleep(1);
}

export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;
  console.log('=== Smoke Test Complete ===');
  console.log(`Duration: ${duration.toFixed(2)}s`);
}
