// Full Flow Simulator - K6 Master Orchestrator
// Simulates the complete order fulfillment flow:
//   Facility Setup → Order Creation → Waving → Picking → Consolidation → Gift Wrap → Packing → Shipping
//
// Usage:
//   k6 run scripts/scenarios/full-flow-simulator.js
//   k6 run -e MAX_ORDERS_PER_RUN=20 scripts/scenarios/full-flow-simulator.js
//   k6 run -e FORCE_ORDER_TYPE=single scripts/scenarios/full-flow-simulator.js  # All single-item orders
//   k6 run -e FORCE_REQUIREMENT=hazmat scripts/scenarios/full-flow-simulator.js # All hazmat orders
//
// Environment variables:
//   ORDER_SERVICE_URL     - Order service URL (default: http://localhost:8001)
//   WAVING_SERVICE_URL    - Waving service URL (default: http://localhost:8002)
//   PICKING_SERVICE_URL   - Picking service URL (default: http://localhost:8004)
//   CONSOLIDATION_SERVICE_URL - Consolidation service URL (default: http://localhost:8005)
//   PACKING_SERVICE_URL   - Packing service URL (default: http://localhost:8006)
//   SHIPPING_SERVICE_URL  - Shipping service URL (default: http://localhost:8007)
//   FACILITY_SERVICE_URL  - Facility service URL (default: http://localhost:8010)
//   ORCHESTRATOR_URL      - Orchestrator URL (default: http://localhost:30010)
//   STAGE_DELAY_MS        - Delay between stages in ms (default: 2000)
//   MAX_ORDERS_PER_RUN    - Number of orders to create per run (default: 10)
//   WAIT_TIMEOUT_MS       - Timeout waiting for tasks in ms (default: 60000)
//   POLL_INTERVAL_MS      - Polling interval in ms (default: 3000)
//   GIFTWRAP_ORDER_RATIO  - Ratio of orders with gift wrap (default: 0.2)
//   SKIP_FACILITY_SETUP   - Set to 'true' to skip facility setup phase
//
// Order Type Configuration:
//   SINGLE_ITEM_RATIO     - Ratio of single-item orders (default: 0.4 = 40%)
//   MULTI_ITEM_RATIO      - Ratio of multi-item orders (default: 0.6 = 60%)
//   MAX_ITEMS_PER_ORDER   - Max items in multi-item orders (default: 5)
//   FORCE_ORDER_TYPE      - Force all orders to be 'single' or 'multi' (default: null)
//   FORCE_REQUIREMENT     - Force all orders to include a requirement: hazmat, fragile, oversized, heavy, high_value (default: null)

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend, Gauge } from 'k6/metrics';
import { SharedArray } from 'k6/data';
import { BASE_URLS, ENDPOINTS, HTTP_PARAMS, FLOW_CONFIG, GIFTWRAP_CONFIG, ORDER_CONFIG } from '../lib/config.js';
import {
  discoverReadyWaves,
  processWave,
  createWaveFromOrders,
  scheduleWave,
  releaseWave,
  sendWaveAssignedSignal,
} from '../lib/waving.js';
import {
  discoverPendingTasks,
  processPickTask,
} from '../lib/picking.js';
import {
  discoverPendingConsolidations,
  processConsolidation,
} from '../lib/consolidation.js';
import {
  discoverPendingPackTasks,
  processPackTask,
} from '../lib/packing.js';
import {
  discoverPendingShipments,
  processShipment,
} from '../lib/shipping.js';
import {
  createStation,
  activateStation,
  findAvailableStation,
  listStations,
  checkHealth as checkFacilityHealth,
} from '../lib/facility.js';
import {
  addGiftWrapToOrder,
  shouldHaveGiftWrap,
  generateGiftWrapDetails,
  getOrder,
  waitForOrderStatus,
  waitForAllOrdersStatus,
} from '../lib/orders.js';
import { generateOrder, generateOrderWithType, getProductCountByRequirement } from '../lib/data.js';

// Load station test data for facility setup
const stationData = new SharedArray('stations', function () {
  try {
    const data = JSON.parse(open('../../data/stations.json'));
    return data.stations || [];
  } catch (e) {
    console.warn('Could not load stations.json, using default stations');
    return [];
  }
});

// Flow-level metrics
const flowOrdersCreated = new Counter('flow_orders_created');
const flowOrdersCompleted = new Counter('flow_orders_completed');
const flowOrdersFailed = new Counter('flow_orders_failed');
const flowSuccessRate = new Rate('flow_success_rate');
const flowE2ELatency = new Trend('flow_e2e_latency');
const flowCurrentStage = new Gauge('flow_current_stage');

// Per-stage metrics
const stageWavingProcessed = new Counter('flow_stage_waving_processed');
const stagePickingProcessed = new Counter('flow_stage_picking_processed');
const stageConsolidationProcessed = new Counter('flow_stage_consolidation_processed');
const stageGiftWrapProcessed = new Counter('flow_stage_giftwrap_processed');
const stagePackingProcessed = new Counter('flow_stage_packing_processed');
const stageShippingProcessed = new Counter('flow_stage_shipping_processed');

// Facility and gift wrap metrics
const facilityStationsCreated = new Counter('flow_facility_stations_created');
const giftWrapOrdersCount = new Counter('flow_giftwrap_orders');

// Order type and requirement metrics
const singleItemOrders = new Counter('flow_single_item_orders');
const multiItemOrders = new Counter('flow_multi_item_orders');
const hazmatOrders = new Counter('flow_hazmat_orders');
const fragileOrders = new Counter('flow_fragile_orders');
const oversizedOrders = new Counter('flow_oversized_orders');
const heavyOrders = new Counter('flow_heavy_orders');
const highValueOrders = new Counter('flow_high_value_orders');

// Stage constants for gauge
const STAGE = {
  FACILITY_SETUP: 0,
  ORDER_CREATION: 1,
  WAVING: 2,
  PICKING: 3,
  CONSOLIDATION: 4,
  GIFT_WRAP: 5,
  PACKING: 6,
  SHIPPING: 7,
  COMPLETE: 8,
};

/**
 * Verify all orders have reached expected status before proceeding
 * @param {Object[]} orders - Array of order objects with orderId
 * @param {string|string[]} expectedStatus - Expected status(es)
 * @param {string} stageName - Name of current stage for logging
 * @returns {number} Number of orders that reached expected status
 */
function verifyOrdersReachedStatus(orders, expectedStatus, stageName) {
  console.log(`\n[${stageName}] Verifying ${orders.length} orders reached status: ${expectedStatus}`);

  const orderIds = orders.map(o => o.orderId);
  const { allSuccess, results } = waitForAllOrdersStatus(
    orderIds,
    expectedStatus,
    FLOW_CONFIG.statusCheckTimeoutMs || 120000,
    FLOW_CONFIG.statusCheckIntervalMs || 3000
  );

  const successCount = results.filter(r => r.success).length;
  console.log(`[${stageName}] ${successCount}/${orders.length} orders reached expected status`);

  if (!allSuccess) {
    const failed = results.filter(r => !r.success);
    for (const f of failed) {
      console.warn(`[${stageName}] Order ${f.orderId} stuck at status: ${f.finalStatus}`);
    }
  }

  return successCount;
}

// Default options
export const options = {
  scenarios: {
    full_flow: {
      executor: 'per-vu-iterations',
      vus: 1,
      iterations: 1,
      maxDuration: '10m',
    },
  },
  thresholds: {
    'flow_success_rate': ['rate>0.8'],
    'flow_e2e_latency': ['p(95)<300000'],  // 5 minutes
  },
};

/**
 * Set up facility stations
 */
function setupFacilityStations(existingStationIds) {
  console.log('\n' + '='.repeat(50));
  console.log('Phase 0: Setting up Facility Stations');
  console.log('='.repeat(50));

  if (__ENV.SKIP_FACILITY_SETUP === 'true') {
    console.log('Skipping facility setup (SKIP_FACILITY_SETUP=true)');
    return { created: 0, skipped: stationData.length };
  }

  let created = 0;
  let skipped = 0;

  for (const station of stationData) {
    // Skip if already exists
    if (existingStationIds.includes(station.stationId)) {
      console.log(`Station ${station.stationId} already exists, skipping`);
      skipped++;
      continue;
    }

    const result = createStation(station);
    if (result.success) {
      created++;
      facilityStationsCreated.add(1);

      // Activate the station
      activateStation(station.stationId);
      console.log(`Created and activated: ${station.stationId}`);
    } else {
      console.warn(`Failed to create station ${station.stationId}`);
    }

    sleep(0.1);
  }

  console.log(`Facility setup complete: ${created} created, ${skipped} skipped`);
  return { created, skipped };
}

/**
 * Creates test orders for the flow (with typed orders and requirements)
 */
function createTestOrders(count) {
  const orders = [];
  let giftWrapCount = 0;
  let singleItemCount = 0;
  let multiItemCount = 0;
  const requirementCounts = {
    hazmat: 0,
    fragile: 0,
    oversized: 0,
    heavy: 0,
    high_value: 0,
  };

  for (let i = 0; i < count; i++) {
    // Use typed order generation (respects FORCE_ORDER_TYPE and FORCE_REQUIREMENT)
    let order = generateOrderWithType();

    // Randomly add gift wrap based on configured ratio
    const isGiftWrap = shouldHaveGiftWrap();
    if (isGiftWrap) {
      order = addGiftWrapToOrder(order);
      giftWrapCount++;
    }

    // Track order type
    if (order.orderType === 'single_item') {
      singleItemCount++;
      singleItemOrders.add(1);
    } else {
      multiItemCount++;
      multiItemOrders.add(1);
    }

    // Track requirements
    if (order.requirements) {
      for (const req of order.requirements) {
        if (requirementCounts.hasOwnProperty(req)) {
          requirementCounts[req]++;
        }
      }
      if (order.requirements.includes('hazmat')) hazmatOrders.add(1);
      if (order.requirements.includes('fragile')) fragileOrders.add(1);
      if (order.requirements.includes('oversized')) oversizedOrders.add(1);
      if (order.requirements.includes('heavy')) heavyOrders.add(1);
      if (order.requirements.includes('high_value')) highValueOrders.add(1);
    }

    const orderPayload = JSON.stringify(order);

    const url = `${BASE_URLS.orders}${ENDPOINTS.orders.create}`;
    const response = http.post(url, orderPayload, HTTP_PARAMS);

    const success = check(response, {
      'create order status 200/201': (r) => r.status === 200 || r.status === 201,
    });

    if (success) {
      try {
        const responseData = JSON.parse(response.body);
        // Response is wrapped in "order" object: { order: {...}, workflowId: "..." }
        const orderData = responseData.order || responseData;
        const orderId = orderData.orderId || orderData.id;
        orders.push({
          orderId: orderId,
          customerId: orderData.customerId,
          workflowId: responseData.workflowId,
          giftWrap: isGiftWrap,
          giftWrapDetails: isGiftWrap ? order.giftWrapDetails : null,
          orderType: order.orderType,
          requirements: order.requirements || [],
        });
        flowOrdersCreated.add(1);
        if (isGiftWrap) {
          giftWrapOrdersCount.add(1);
        }
        const reqStr = order.requirements?.length > 0 ? ` [${order.requirements.join(', ')}]` : '';
        console.log(`Created ${order.orderType} order: ${orderId}${reqStr}${isGiftWrap ? ' (gift wrap)' : ''}`);
      } catch (e) {
        console.error(`Failed to parse order response: ${e.message}`);
      }
    } else {
      console.error(`Failed to create order: ${response.status} - ${response.body}`);
    }

    sleep(0.2);  // Brief pause between order creation
  }

  console.log(`Created ${orders.length} orders:`);
  console.log(`  - Single-item: ${singleItemCount}`);
  console.log(`  - Multi-item: ${multiItemCount}`);
  console.log(`  - Gift wrap: ${giftWrapCount}`);
  console.log(`  - Requirements: hazmat=${requirementCounts.hazmat}, fragile=${requirementCounts.fragile}, ` +
              `oversized=${requirementCounts.oversized}, heavy=${requirementCounts.heavy}, high_value=${requirementCounts.high_value}`);
  return orders;
}

/**
 * Send gift wrap completed signal to orchestrator
 */
function sendGiftWrapCompletedSignal(orderId, stationId, wrapType, giftMessage) {
  const url = `${BASE_URLS.orchestrator}${ENDPOINTS.orchestrator.signalGiftWrapCompleted}`;
  const payload = JSON.stringify({
    orderId,
    stationId,
    wrapType: wrapType || 'standard',
    giftMessage: giftMessage || '',
    completedAt: new Date().toISOString(),
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  return check(response, {
    'gift wrap signal sent': (r) => r.status === 200 || r.status === 202,
  });
}

/**
 * Process gift wrap orders
 */
function processGiftWrapOrders(orders) {
  const giftWrapOrders = orders.filter((o) => o.giftWrap);

  if (giftWrapOrders.length === 0) {
    console.log('No gift wrap orders to process');
    return { processed: 0, failed: 0 };
  }

  console.log(`Processing ${giftWrapOrders.length} gift wrap orders`);

  let processed = 0;
  let failed = 0;

  for (const order of giftWrapOrders) {
    // Find an available gift wrap station
    const station = findAvailableStation(['gift_wrap'], 'packing', '');

    if (!station) {
      console.warn(`No gift wrap station available for order ${order.orderId}`);
      failed++;
      continue;
    }

    // Simulate gift wrap processing
    console.log(`Gift wrapping order ${order.orderId} at station ${station.stationId}`);
    sleep(GIFTWRAP_CONFIG.simulationDelayMs / 1000);

    // Send completion signal
    const signalSuccess = sendGiftWrapCompletedSignal(
      order.orderId,
      station.stationId,
      order.giftWrapDetails?.wrapType,
      order.giftWrapDetails?.giftMessage
    );

    if (signalSuccess) {
      processed++;
      stageGiftWrapProcessed.add(1);
      console.log(`Gift wrap completed for order ${order.orderId}`);
    } else {
      failed++;
      console.warn(`Failed to signal gift wrap completion for ${order.orderId}`);
    }
  }

  return { processed, failed };
}

/**
 * Polls for tasks/items with timeout
 */
function pollForItems(discoverFn, itemName, orderIds, timeoutMs = FLOW_CONFIG.waitForTasksTimeoutMs) {
  const startTime = Date.now();
  const pollInterval = FLOW_CONFIG.pollIntervalMs;

  while (Date.now() - startTime < timeoutMs) {
    const items = discoverFn();

    // Filter to only items for our orders if possible
    const relevantItems = items.filter((item) => {
      const itemOrderId = item.orderId;
      return !orderIds || orderIds.length === 0 || orderIds.includes(itemOrderId);
    });

    if (relevantItems.length > 0) {
      console.log(`Found ${relevantItems.length} ${itemName} items`);
      return relevantItems;
    }

    console.log(`Waiting for ${itemName} items... (${Math.floor((Date.now() - startTime) / 1000)}s elapsed)`);
    sleep(pollInterval / 1000);
  }

  console.warn(`Timeout waiting for ${itemName} items after ${timeoutMs}ms`);
  return [];
}

/**
 * Processes a stage with polling
 */
function processStage(stageName, discoverFn, processFn, stageCounter, orderIds = []) {
  console.log(`\n${'='.repeat(50)}`);
  console.log(`Stage: ${stageName}`);
  console.log('='.repeat(50));

  const items = pollForItems(discoverFn, stageName, orderIds);

  if (items.length === 0) {
    console.warn(`No ${stageName} items found to process`);
    return { processed: 0, failed: 0 };
  }

  let processed = 0;
  let failed = 0;

  for (const item of items) {
    try {
      const success = processFn(item);
      if (success) {
        processed++;
        stageCounter.add(1);
      } else {
        failed++;
      }
    } catch (e) {
      console.error(`Error processing ${stageName} item: ${e.message}`);
      failed++;
    }

    sleep(FLOW_CONFIG.stageDelayMs / 2000);  // Pause between items
  }

  console.log(`${stageName} complete: ${processed} processed, ${failed} failed`);
  return { processed, failed };
}

// Setup function - health checks
export function setup() {
  console.log('='.repeat(60));
  console.log('Full Flow Simulator Starting');
  console.log('='.repeat(60));
  console.log(`Max orders per run: ${FLOW_CONFIG.maxOrdersPerRun}`);
  console.log(`Stage delay: ${FLOW_CONFIG.stageDelayMs}ms`);
  console.log(`Wait timeout: ${FLOW_CONFIG.waitForTasksTimeoutMs}ms`);
  console.log(`Gift wrap ratio: ${GIFTWRAP_CONFIG.giftWrapOrderRatio * 100}%`);
  console.log(`Skip facility setup: ${__ENV.SKIP_FACILITY_SETUP === 'true'}`);
  console.log('--- Order Configuration ---');
  console.log(`Single-item ratio: ${ORDER_CONFIG.singleItemRatio * 100}%`);
  console.log(`Multi-item ratio: ${ORDER_CONFIG.multiItemRatio * 100}%`);
  console.log(`Max items per order: ${ORDER_CONFIG.maxItemsPerOrder}`);
  console.log(`Force order type: ${ORDER_CONFIG.forceOrderType || 'none'}`);
  console.log(`Force requirement: ${ORDER_CONFIG.forceRequirement || 'none'}`);
  console.log('='.repeat(60));

  // Health check all services
  const services = [
    { name: 'Orders', url: `${BASE_URLS.orders}/health` },
    { name: 'Waving', url: `${BASE_URLS.waving}/health` },
    { name: 'Picking', url: `${BASE_URLS.picking}/health` },
    { name: 'Consolidation', url: `${BASE_URLS.consolidation}/health` },
    { name: 'Packing', url: `${BASE_URLS.packing}/health` },
    { name: 'Shipping', url: `${BASE_URLS.shipping}/health` },
    { name: 'Facility', url: `${BASE_URLS.facility}/health` },
    { name: 'Orchestrator', url: `${BASE_URLS.orchestrator}/health` },
  ];

  const healthStatus = {};
  for (const service of services) {
    try {
      const response = http.get(service.url, { timeout: '5s' });
      healthStatus[service.name] = response.status === 200;
      console.log(`${service.name}: ${response.status === 200 ? 'OK' : 'FAILED'}`);
    } catch (e) {
      healthStatus[service.name] = false;
      console.log(`${service.name}: FAILED (${e.message})`);
    }
  }

  // Get existing stations for facility setup phase
  let existingStationIds = [];
  try {
    const stationsResult = listStations(100, 0);
    if (stationsResult.success && stationsResult.stations) {
      existingStationIds = stationsResult.stations.map((s) => s.stationId);
      console.log(`Existing stations: ${existingStationIds.length}`);
    }
  } catch (e) {
    console.warn(`Could not list existing stations: ${e.message}`);
  }

  return {
    startTime: new Date().toISOString(),
    healthStatus: healthStatus,
    existingStationIds: existingStationIds,
  };
}

// Main flow execution
export default function (data) {
  const flowStartTime = Date.now();

  console.log('\n' + '='.repeat(60));
  console.log('Starting Full Order Fulfillment Flow');
  console.log('='.repeat(60));

  // Phase 0: Facility Setup
  flowCurrentStage.add(STAGE.FACILITY_SETUP);
  const facilitySetup = setupFacilityStations(data.existingStationIds || []);
  sleep(FLOW_CONFIG.stageDelayMs / 1000);

  // Phase 1: Order Creation
  flowCurrentStage.add(STAGE.ORDER_CREATION);
  console.log('\nPhase 1: Creating Orders');
  const orders = createTestOrders(FLOW_CONFIG.maxOrdersPerRun);
  const orderIds = orders.map((o) => o.orderId);
  const giftWrapOrders = orders.filter((o) => o.giftWrap);

  if (orders.length === 0) {
    console.error('No orders created, aborting flow');
    flowOrdersFailed.add(FLOW_CONFIG.maxOrdersPerRun);
    return;
  }

  console.log(`Created ${orders.length} orders`);
  sleep(FLOW_CONFIG.stageDelayMs / 1000);

  // Phase 2: Waving - Create wave, release it, and signal workflows
  flowCurrentStage.add(STAGE.WAVING);
  console.log('\nPhase 2: Creating and Releasing Wave');
  const wavingResults = { processed: 0, failed: 0 };

  // Create a wave from our orders
  const wave = createWaveFromOrders(orderIds);
  if (wave) {
    const waveId = wave.waveId || wave.id;
    console.log(`Created wave: ${waveId} with ${orderIds.length} orders`);

    // Schedule and release the wave
    sleep(0.5);
    scheduleWave(waveId);
    sleep(0.5);
    if (releaseWave(waveId)) {
      console.log(`Wave ${waveId} released`);

      // Send waveAssigned signals to progress workflows
      for (const orderId of orderIds) {
        if (sendWaveAssignedSignal(orderId, waveId)) {
          wavingResults.processed++;
        } else {
          wavingResults.failed++;
        }
        sleep(0.1);  // Brief pause between signals
      }
      stageWavingProcessed.add(wavingResults.processed);
      console.log(`Sent ${wavingResults.processed} wave assigned signals`);
    } else {
      console.warn('Failed to release wave');
      wavingResults.failed = orderIds.length;
    }
  } else {
    console.warn('Failed to create wave from orders');
    wavingResults.failed = orderIds.length;
  }
  sleep(FLOW_CONFIG.stageDelayMs / 1000);

  // Verify orders have transitioned to wave_assigned status
  verifyOrdersReachedStatus(orders, ['wave_assigned', 'picking', 'consolidated', 'packed', 'shipped'], 'WAVING');

  // Phase 3: Picking
  flowCurrentStage.add(STAGE.PICKING);
  console.log('\nPhase 3: Processing Pick Tasks');
  const pickingResults = processStage(
    'Picking',
    () => discoverPendingTasks('assigned'),
    processPickTask,
    stagePickingProcessed,
    orderIds
  );
  sleep(FLOW_CONFIG.stageDelayMs / 1000);

  // Verify orders have progressed past picking
  verifyOrdersReachedStatus(orders, ['picking', 'consolidated', 'packed', 'shipped'], 'PICKING');

  // Phase 4: Consolidation
  flowCurrentStage.add(STAGE.CONSOLIDATION);
  console.log('\nPhase 4: Processing Consolidations');
  const consolidationResults = processStage(
    'Consolidation',
    discoverPendingConsolidations,
    processConsolidation,
    stageConsolidationProcessed,
    orderIds
  );
  sleep(FLOW_CONFIG.stageDelayMs / 1000);

  // Verify multi-item orders have consolidated
  verifyOrdersReachedStatus(orders, ['consolidated', 'packed', 'shipped'], 'CONSOLIDATION');

  // Phase 5: Gift Wrap (for orders with giftWrap=true)
  flowCurrentStage.add(STAGE.GIFT_WRAP);
  console.log('\nPhase 5: Processing Gift Wrap');
  const giftWrapResults = processGiftWrapOrders(orders);
  sleep(FLOW_CONFIG.stageDelayMs / 1000);

  // Phase 6: Packing
  flowCurrentStage.add(STAGE.PACKING);
  console.log('\nPhase 6: Processing Pack Tasks');
  const packingResults = processStage(
    'Packing',
    discoverPendingPackTasks,
    processPackTask,
    stagePackingProcessed,
    orderIds
  );
  sleep(FLOW_CONFIG.stageDelayMs / 1000);

  // Verify orders have been packed
  verifyOrdersReachedStatus(orders, ['packed', 'shipped'], 'PACKING');

  // Phase 7: Shipping
  flowCurrentStage.add(STAGE.SHIPPING);
  console.log('\nPhase 7: Processing Shipments');
  const shippingResults = processStage(
    'Shipping',
    discoverPendingShipments,
    processShipment,
    stageShippingProcessed,
    orderIds
  );

  // Calculate final metrics
  flowCurrentStage.add(STAGE.COMPLETE);
  const flowEndTime = Date.now();
  const flowDuration = flowEndTime - flowStartTime;

  // Count completed orders (shipped successfully)
  const completedOrders = shippingResults.processed;
  const failedOrders = orders.length - completedOrders;

  flowOrdersCompleted.add(completedOrders);
  flowOrdersFailed.add(failedOrders);
  flowE2ELatency.add(flowDuration);

  for (let i = 0; i < completedOrders; i++) {
    flowSuccessRate.add(true);
  }
  for (let i = 0; i < failedOrders; i++) {
    flowSuccessRate.add(false);
  }

  // Calculate order type stats
  const singleItems = orders.filter(o => o.orderType === 'single_item').length;
  const multiItems = orders.filter(o => o.orderType === 'multi_item').length;
  const reqStats = {
    hazmat: orders.filter(o => o.requirements?.includes('hazmat')).length,
    fragile: orders.filter(o => o.requirements?.includes('fragile')).length,
    oversized: orders.filter(o => o.requirements?.includes('oversized')).length,
    heavy: orders.filter(o => o.requirements?.includes('heavy')).length,
    high_value: orders.filter(o => o.requirements?.includes('high_value')).length,
  };

  // Summary
  console.log('\n' + '='.repeat(60));
  console.log('Flow Complete - Summary');
  console.log('='.repeat(60));
  console.log(`Facility Stations Created: ${facilitySetup.created}`);
  console.log(`Orders Created: ${orders.length}`);
  console.log(`  - Single-item: ${singleItems}`);
  console.log(`  - Multi-item: ${multiItems}`);
  console.log(`  - Gift Wrap: ${giftWrapOrders.length}`);
  console.log(`  - Requirements:`);
  console.log(`    - Hazmat: ${reqStats.hazmat}`);
  console.log(`    - Fragile: ${reqStats.fragile}`);
  console.log(`    - Oversized: ${reqStats.oversized}`);
  console.log(`    - Heavy: ${reqStats.heavy}`);
  console.log(`    - High Value: ${reqStats.high_value}`);
  console.log(`Waves Processed: ${wavingResults.processed}`);
  console.log(`Pick Tasks Processed: ${pickingResults.processed}`);
  console.log(`Consolidations Processed: ${consolidationResults.processed}`);
  console.log(`Gift Wrap Processed: ${giftWrapResults.processed}`);
  console.log(`Pack Tasks Processed: ${packingResults.processed}`);
  console.log(`Shipments Processed: ${shippingResults.processed}`);
  console.log(`Total Duration: ${flowDuration}ms (${(flowDuration / 1000).toFixed(1)}s)`);
  console.log(`Success Rate: ${((completedOrders / orders.length) * 100).toFixed(1)}%`);
  console.log('='.repeat(60));
}

// Teardown
export function teardown(data) {
  console.log('\n' + '='.repeat(60));
  console.log('Full Flow Simulator Complete');
  console.log(`Started: ${data.startTime}`);
  console.log(`Ended: ${new Date().toISOString()}`);
  console.log('='.repeat(60));
}
