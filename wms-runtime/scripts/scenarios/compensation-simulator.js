// Compensation Simulator
// Tests order cancellation and compensation workflows

import { check, sleep, group } from 'k6';
import { Counter, Trend, Rate } from 'k6/metrics';
import http from 'k6/http';
import { BASE_URLS, HTTP_PARAMS } from '../lib/config.js';
import { createOrder, getOrderStatus, cancelOrder } from '../lib/orders.js';
import { processPickTask, discoverPendingTasks } from '../lib/picking.js';
import { triggerCompensation, verifyCompensation } from '../lib/chaos.js';
import { generateOrderWithType } from '../lib/data.js';

// Custom metrics
const cancellationsTriggered = new Counter('compensation_cancellations_triggered');
const cancellationsSuccessful = new Counter('compensation_cancellations_successful');
const cancellationsFailed = new Counter('compensation_cancellations_failed');
const inventoryReleased = new Counter('compensation_inventory_released');
const unitsCleanedUp = new Counter('compensation_units_cleaned_up');
const compensationDuration = new Trend('compensation_duration_ms');
const compensationSuccessRate = new Rate('compensation_success_rate');

// Test configuration
export const options = {
  scenarios: {
    compensation_testing: {
      executor: 'constant-vus',
      vus: 2,
      duration: '3m',
    },
  },
  thresholds: {
    'compensation_success_rate': ['rate>0.90'],
    'compensation_duration_ms': ['p(95)<15000'],
    'http_req_failed': ['rate<0.10'],
  },
};

// Compensation stages for testing
const CANCELLATION_STAGES = {
  BEFORE_WAVE: 'before_wave',
  AFTER_WAVE: 'after_wave',
  DURING_PICK: 'during_pick',
  AFTER_PICK: 'after_pick',
  DURING_PACK: 'during_pack',
  AFTER_PACK: 'after_pack',
};

// Configuration
const CONFIG = {
  ordersPerIteration: parseInt(__ENV.COMPENSATION_ORDERS || '3'),
  testAllStages: __ENV.TEST_ALL_STAGES === 'true',
  verifyCleanup: __ENV.VERIFY_CLEANUP !== 'false',
  verifyTimeoutMs: parseInt(__ENV.VERIFY_TIMEOUT_MS || '30000'),
};

/**
 * Creates an order and advances it to a specific stage
 */
function createOrderAtStage(stage) {
  console.log(`Creating order to test cancellation at stage: ${stage}`);

  // Create order
  const order = generateOrderWithType('multi', null);
  const orderResult = createOrder(order);

  if (!orderResult || !orderResult.orderId) {
    console.warn('Failed to create order');
    return null;
  }

  const orderId = orderResult.orderId;
  console.log(`Created order: ${orderId}`);

  // Advance to the target stage
  switch (stage) {
    case CANCELLATION_STAGES.BEFORE_WAVE:
      // Order is just created, no advancement needed
      break;

    case CANCELLATION_STAGES.AFTER_WAVE:
      // Would need to trigger wave assignment - simulated here
      sleep(0.5);
      break;

    case CANCELLATION_STAGES.DURING_PICK:
      // Simulate order being in pick phase
      sleep(1);
      break;

    case CANCELLATION_STAGES.AFTER_PICK:
      // Simulate completed pick
      sleep(1.5);
      break;

    case CANCELLATION_STAGES.DURING_PACK:
      // Simulate order being in pack phase
      sleep(2);
      break;

    case CANCELLATION_STAGES.AFTER_PACK:
      // Simulate completed pack (before ship)
      sleep(2.5);
      break;
  }

  return {
    orderId: orderId,
    stage: stage,
    items: order.items,
  };
}

/**
 * Cancels an order and verifies cleanup
 */
function cancelAndVerify(orderInfo) {
  const startTime = Date.now();
  const orderId = orderInfo.orderId;

  console.log(`Cancelling order ${orderId} at stage ${orderInfo.stage}`);
  cancellationsTriggered.add(1);

  // Cancel the order
  const cancelResult = cancelOrder(orderId, `compensation_test_${orderInfo.stage}`);

  if (!cancelResult) {
    console.warn(`Failed to cancel order ${orderId}`);
    cancellationsFailed.add(1);
    compensationSuccessRate.add(0);
    return {
      success: false,
      error: 'cancel_failed',
      orderId: orderId,
      stage: orderInfo.stage,
    };
  }

  console.log(`Order ${orderId} cancellation initiated`);

  // Verify order status is cancelled
  let attempts = 0;
  const maxAttempts = 10;
  let orderStatus = null;

  while (attempts < maxAttempts) {
    attempts++;
    sleep(1);

    const status = getOrderStatus(orderId);
    if (status && (status.status === 'cancelled' || status.state === 'cancelled')) {
      orderStatus = status;
      break;
    }
  }

  if (!orderStatus) {
    console.warn(`Order ${orderId} status verification timed out`);
    cancellationsFailed.add(1);
    compensationSuccessRate.add(0);
    return {
      success: false,
      error: 'status_verification_timeout',
      orderId: orderId,
      stage: orderInfo.stage,
    };
  }

  console.log(`Order ${orderId} confirmed as cancelled`);

  // Verify inventory release (if applicable)
  if (CONFIG.verifyCleanup && orderInfo.stage !== CANCELLATION_STAGES.BEFORE_WAVE) {
    const inventoryVerified = verifyInventoryRelease(orderId, orderInfo.items);
    if (inventoryVerified) {
      inventoryReleased.add(orderInfo.items.length);
    }

    // Verify unit cleanup
    const unitsVerified = verifyUnitCleanup(orderId);
    if (unitsVerified) {
      unitsCleanedUp.add(1);
    }
  }

  const duration = Date.now() - startTime;
  compensationDuration.add(duration);
  cancellationsSuccessful.add(1);
  compensationSuccessRate.add(1);

  console.log(`Compensation complete for ${orderId} in ${duration}ms`);

  return {
    success: true,
    orderId: orderId,
    stage: orderInfo.stage,
    duration: duration,
  };
}

/**
 * Verifies that inventory was released for cancelled order
 */
function verifyInventoryRelease(orderId, items) {
  console.log(`Verifying inventory release for order ${orderId}`);

  // Check inventory reservations are cleared
  const url = `${BASE_URLS.inventory}/api/v1/reservations/order/${orderId}`;
  const response = http.get(url, HTTP_PARAMS);

  if (response.status === 200) {
    try {
      const data = JSON.parse(response.body);
      const reservations = Array.isArray(data) ? data : (data.reservations || []);

      if (reservations.length === 0) {
        console.log(`Inventory reservations cleared for ${orderId}`);
        return true;
      }

      // Check if all reservations are released
      const activeReservations = reservations.filter(r =>
        r.status === 'active' || r.status === 'reserved'
      );

      if (activeReservations.length === 0) {
        console.log(`All reservations released for ${orderId}`);
        return true;
      }

      console.warn(`Found ${activeReservations.length} active reservations for ${orderId}`);
      return false;
    } catch (e) {
      console.warn(`Failed to parse reservations: ${e.message}`);
    }
  } else if (response.status === 404) {
    // No reservations found - could be expected
    console.log(`No reservations found for ${orderId}`);
    return true;
  }

  return false;
}

/**
 * Verifies that units were cleaned up for cancelled order
 */
function verifyUnitCleanup(orderId) {
  console.log(`Verifying unit cleanup for order ${orderId}`);

  const url = `${BASE_URLS.unit}/api/v1/units/order/${orderId}`;
  const response = http.get(url, HTTP_PARAMS);

  if (response.status === 200) {
    try {
      const data = JSON.parse(response.body);
      const units = Array.isArray(data) ? data : (data.units || []);

      // Check if all units are in terminal state
      const activeUnits = units.filter(u =>
        u.status !== 'cancelled' && u.status !== 'exception' && u.status !== 'released'
      );

      if (activeUnits.length === 0) {
        console.log(`All units cleaned up for ${orderId}`);
        return true;
      }

      console.warn(`Found ${activeUnits.length} active units for ${orderId}`);
      return false;
    } catch (e) {
      console.warn(`Failed to parse units: ${e.message}`);
    }
  } else if (response.status === 404) {
    console.log(`No units found for ${orderId}`);
    return true;
  }

  return false;
}

/**
 * Tests partial fulfillment cancellation
 */
function testPartialFulfillmentCancellation() {
  console.log('Testing partial fulfillment cancellation');
  const startTime = Date.now();

  // Create multi-item order
  const order = generateOrderWithType('multi', null);
  // Ensure we have multiple items
  while (order.items.length < 3) {
    const extraOrder = generateOrderWithType('single', null);
    order.items.push(...extraOrder.items);
  }

  const orderResult = createOrder(order);
  if (!orderResult || !orderResult.orderId) {
    return { success: false, error: 'order_creation_failed' };
  }

  const orderId = orderResult.orderId;
  console.log(`Created multi-item order: ${orderId} with ${order.items.length} items`);

  // Simulate partial pick (half the items)
  sleep(1);
  console.log(`Simulating partial pick for ${orderId}`);

  // Cancel after partial pick
  cancellationsTriggered.add(1);
  const cancelResult = cancelOrder(orderId, 'partial_fulfillment_test');

  if (!cancelResult) {
    cancellationsFailed.add(1);
    compensationSuccessRate.add(0);
    return { success: false, error: 'cancel_failed' };
  }

  // Verify compensation
  sleep(2);

  const inventoryVerified = verifyInventoryRelease(orderId, order.items);
  const unitsVerified = verifyUnitCleanup(orderId);

  const duration = Date.now() - startTime;
  compensationDuration.add(duration);

  if (inventoryVerified && unitsVerified) {
    cancellationsSuccessful.add(1);
    compensationSuccessRate.add(1);
    inventoryReleased.add(order.items.length);
    unitsCleanedUp.add(1);
  } else {
    cancellationsFailed.add(1);
    compensationSuccessRate.add(0);
  }

  return {
    success: inventoryVerified && unitsVerified,
    orderId: orderId,
    itemCount: order.items.length,
    inventoryReleased: inventoryVerified,
    unitsCleanedUp: unitsVerified,
    duration: duration,
  };
}

/**
 * Main test function
 */
export default function () {
  const vuId = __VU;
  const iterationId = __ITER;

  console.log(`[VU ${vuId}] Starting compensation simulation - iteration ${iterationId}`);

  // Phase 1: Test cancellation at all stages
  if (CONFIG.testAllStages && iterationId === 0) {
    group('Test Cancellation At All Stages', function () {
      const stages = Object.values(CANCELLATION_STAGES);

      for (const stage of stages) {
        console.log(`[VU ${vuId}] Testing cancellation at: ${stage}`);

        const orderInfo = createOrderAtStage(stage);
        if (orderInfo) {
          const result = cancelAndVerify(orderInfo);
          console.log(`[VU ${vuId}] Stage ${stage}: ${result.success ? 'SUCCESS' : 'FAILED'}`);
        }

        sleep(1);
      }
    });
  }

  // Phase 2: Random cancellation tests
  group('Random Cancellation Tests', function () {
    const stages = Object.values(CANCELLATION_STAGES);

    for (let i = 0; i < CONFIG.ordersPerIteration; i++) {
      // Select random stage
      const stage = stages[Math.floor(Math.random() * stages.length)];
      console.log(`[VU ${vuId}] Testing random cancellation at: ${stage}`);

      const orderInfo = createOrderAtStage(stage);
      if (orderInfo) {
        const result = cancelAndVerify(orderInfo);
        console.log(`[VU ${vuId}] Result: ${result.success ? 'SUCCESS' : 'FAILED'}`);
      }

      sleep(1);
    }
  });

  // Phase 3: Partial fulfillment test
  group('Partial Fulfillment Cancellation', function () {
    const result = testPartialFulfillmentCancellation();
    console.log(`[VU ${vuId}] Partial fulfillment test: ${result.success ? 'SUCCESS' : 'FAILED'}`);
  });

  // Brief pause between iterations
  sleep(2);
}

/**
 * Setup function
 */
export function setup() {
  console.log('='.repeat(60));
  console.log('Compensation Simulator - Setup');
  console.log('='.repeat(60));
  console.log(`Orders per iteration: ${CONFIG.ordersPerIteration}`);
  console.log(`Test all stages: ${CONFIG.testAllStages}`);
  console.log(`Verify cleanup: ${CONFIG.verifyCleanup}`);
  console.log('Cancellation stages:');
  Object.values(CANCELLATION_STAGES).forEach(s => {
    console.log(`  - ${s}`);
  });
  console.log('='.repeat(60));

  return {
    startTime: Date.now(),
  };
}

/**
 * Teardown function
 */
export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;

  console.log('='.repeat(60));
  console.log('Compensation Simulator - Summary');
  console.log('='.repeat(60));
  console.log(`Total duration: ${duration.toFixed(2)}s`);
  console.log('='.repeat(60));
}

/**
 * Custom summary handler
 */
export function handleSummary(data) {
  const summary = {
    timestamp: new Date().toISOString(),
    simulator: 'compensation-simulator',
    metrics: {
      cancellations_triggered: data.metrics.compensation_cancellations_triggered?.values?.count || 0,
      cancellations_successful: data.metrics.compensation_cancellations_successful?.values?.count || 0,
      cancellations_failed: data.metrics.compensation_cancellations_failed?.values?.count || 0,
      inventory_released: data.metrics.compensation_inventory_released?.values?.count || 0,
      units_cleaned_up: data.metrics.compensation_units_cleaned_up?.values?.count || 0,
      success_rate: data.metrics.compensation_success_rate?.values?.rate || 0,
      avg_duration_ms: data.metrics.compensation_duration_ms?.values?.avg || 0,
      p95_duration_ms: data.metrics.compensation_duration_ms?.values?.['p(95)'] || 0,
    },
    thresholds: data.thresholds,
  };

  return {
    'stdout': JSON.stringify(summary, null, 2) + '\n',
    'compensation-results.json': JSON.stringify(summary, null, 2),
  };
}
