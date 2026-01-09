// Gift Wrap Simulator - K6 Gift Wrap Workflow Scenario
// Simulates the gift wrap workflow stage for orders requiring gift wrapping
//
// Usage:
//   k6 run scripts/scenarios/giftwrap-simulator.js
//   k6 run -e GIFTWRAP_DELAY_MS=1500 scripts/scenarios/giftwrap-simulator.js
//
// Environment variables:
//   FACILITY_SERVICE_URL  - Facility service URL (default: http://localhost:8010)
//   ORCHESTRATOR_URL      - Orchestrator URL (default: http://localhost:30010)
//   GIFTWRAP_DELAY_MS     - Simulated gift wrap time in ms (default: 2000)
//   MAX_GIFTWRAP_TASKS    - Max tasks per iteration (default: 5)
//   GIFTWRAP_TYPE         - Default wrap type (default: standard)

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend, Gauge } from 'k6/metrics';
import { SharedArray } from 'k6/data';
import { BASE_URLS, ENDPOINTS, HTTP_PARAMS, GIFTWRAP_CONFIG } from '../lib/config.js';
import {
  findAvailableStation,
  findCapableStations,
  getStationCapacity,
  getGiftWrapStations,
  checkHealth,
} from '../lib/facility.js';

// Load station test data for reference
const stationData = new SharedArray('stations', function () {
  const data = JSON.parse(open('../../data/stations.json'));
  return {
    stations: data.stations.filter((s) => s.capabilities.includes('gift_wrap')),
    wrapTypes: data.giftWrapTypes,
    messages: data.giftMessages,
  };
});

// Custom metrics
const tasksDiscovered = new Counter('giftwrap_tasks_discovered');
const tasksCompleted = new Counter('giftwrap_tasks_completed');
const tasksFailed = new Counter('giftwrap_tasks_failed');
const successRate = new Rate('giftwrap_success_rate');
const processingTime = new Trend('giftwrap_processing_time');
const stationUtilization = new Gauge('giftwrap_station_utilization');
const signalsSent = new Counter('giftwrap_signals_sent');

// Default options
export const options = {
  scenarios: {
    giftwrap_processor: {
      executor: 'constant-vus',
      vus: 1,
      duration: '5m',
    },
  },
  thresholds: {
    giftwrap_success_rate: ['rate>0.9'],
    giftwrap_processing_time: ['p(95)<5000'],
    http_req_duration: ['p(95)<500'],
  },
};

/**
 * Send gift wrap completed signal to orchestrator
 */
function sendGiftWrapCompletedSignal(orderId, stationId, wrapType, giftMessage) {
  const url = `${BASE_URLS.orchestrator}${ENDPOINTS.orchestrator.signalGiftWrapCompleted}`;
  const payload = JSON.stringify({
    orderId,
    stationId,
    wrapType: wrapType || GIFTWRAP_CONFIG.defaultWrapType,
    giftMessage: giftMessage || '',
    completedAt: new Date().toISOString(),
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'gift wrap signal sent': (r) => r.status === 200 || r.status === 202,
  });

  if (success) {
    signalsSent.add(1);
    console.log(`Signal sent for order ${orderId} -> station ${stationId}`);
  } else {
    console.error(`Failed to send signal for order ${orderId}: ${response.status}`);
  }

  return success;
}

/**
 * Discover orders pending gift wrap
 * In a real scenario, this would query the orchestrator or order service
 * For simulation, we generate mock pending tasks
 */
function discoverPendingGiftWrapTasks() {
  // Query orchestrator for orders awaiting gift wrap
  // This endpoint would need to exist in the orchestrator
  const url = `${BASE_URLS.orchestrator}/api/v1/workflows/pending-gift-wrap`;

  try {
    const response = http.get(url, { ...HTTP_PARAMS, timeout: '5s' });

    if (response.status === 200) {
      const data = response.json();
      return data.orders || data || [];
    }
  } catch (e) {
    console.warn(`Could not discover pending tasks: ${e.message}`);
  }

  // Return empty array if endpoint doesn't exist or fails
  // The full-flow-simulator will manage the orders directly
  return [];
}

/**
 * Simulate gift wrap processing for an order
 */
function processGiftWrap(order, station) {
  const startTime = Date.now();

  console.log(`Processing gift wrap for order ${order.orderId}`);
  console.log(`  Station: ${station.stationId}`);
  console.log(`  Wrap type: ${order.giftWrapDetails?.wrapType || 'standard'}`);

  // Simulate gift wrap operation
  const wrapTimeMs = GIFTWRAP_CONFIG.simulationDelayMs;
  const steps = [
    'selecting_materials',
    'wrapping',
    'applying_message',
    'quality_check',
    'complete',
  ];

  for (const step of steps) {
    sleep(wrapTimeMs / (steps.length * 1000));
    console.log(`  Step: ${step}`);
  }

  const endTime = Date.now();
  const duration = endTime - startTime;

  processingTime.add(duration);
  console.log(`  Duration: ${duration}ms`);

  return {
    success: true,
    orderId: order.orderId,
    stationId: station.stationId,
    wrapType: order.giftWrapDetails?.wrapType || 'standard',
    giftMessage: order.giftWrapDetails?.giftMessage || null,
    duration,
  };
}

/**
 * Setup function - verify services and stations
 */
export function setup() {
  console.log('='.repeat(60));
  console.log('Gift Wrap Simulator Starting');
  console.log('='.repeat(60));
  console.log(`Wrap delay: ${GIFTWRAP_CONFIG.simulationDelayMs}ms`);
  console.log(`Max tasks per iteration: ${GIFTWRAP_CONFIG.maxTasksPerIteration}`);
  console.log(`Default wrap type: ${GIFTWRAP_CONFIG.defaultWrapType}`);
  console.log('='.repeat(60));

  // Check facility service
  const facilityHealthy = checkHealth();
  console.log(`Facility service: ${facilityHealthy ? 'OK' : 'FAILED'}`);

  // Check orchestrator
  try {
    const orchResponse = http.get(`${BASE_URLS.orchestrator}/health`, { timeout: '5s' });
    console.log(`Orchestrator: ${orchResponse.status === 200 ? 'OK' : 'FAILED'}`);
  } catch (e) {
    console.log(`Orchestrator: FAILED (${e.message})`);
  }

  // Get available gift wrap stations
  const giftWrapStations = getGiftWrapStations();
  console.log(`Gift wrap stations available: ${giftWrapStations.length}`);

  if (giftWrapStations.length === 0) {
    console.warn('No gift wrap stations available!');
    console.warn('Run facility-simulator.js first to set up stations');
  }

  return {
    startTime: new Date().toISOString(),
    giftWrapStations: giftWrapStations.map((s) => s.stationId),
  };
}

/**
 * Main iteration - discover and process gift wrap tasks
 */
export default function (data) {
  const delaySeconds = GIFTWRAP_CONFIG.simulationDelayMs / 1000;

  console.log('\n' + '-'.repeat(40));
  console.log(`Gift Wrap Iteration - ${new Date().toISOString()}`);
  console.log('-'.repeat(40));

  // Phase 1: Check station availability
  const availableStations = [];
  for (const stationId of data.giftWrapStations) {
    const capacity = getStationCapacity(stationId);
    if (capacity && capacity.isAvailable) {
      availableStations.push({
        stationId,
        ...capacity,
      });
    }
  }

  if (availableStations.length === 0) {
    console.log('No gift wrap stations available with capacity');
    sleep(delaySeconds);
    return;
  }

  // Calculate average utilization
  const totalCapacity = availableStations.reduce((sum, s) => sum + s.maxConcurrentTasks, 0);
  const usedCapacity = availableStations.reduce((sum, s) => sum + s.currentTasks, 0);
  const avgUtilization = totalCapacity > 0 ? (usedCapacity / totalCapacity) * 100 : 0;
  stationUtilization.add(avgUtilization);

  console.log(`Available stations: ${availableStations.length}`);
  console.log(`Station utilization: ${avgUtilization.toFixed(1)}%`);

  // Phase 2: Discover pending gift wrap tasks
  const pendingTasks = discoverPendingGiftWrapTasks();
  tasksDiscovered.add(pendingTasks.length);

  if (pendingTasks.length === 0) {
    console.log('No pending gift wrap tasks');
    sleep(delaySeconds);
    return;
  }

  console.log(`Pending gift wrap tasks: ${pendingTasks.length}`);

  // Phase 3: Process tasks (up to max per iteration)
  const maxTasks = Math.min(pendingTasks.length, GIFTWRAP_CONFIG.maxTasksPerIteration);
  let completed = 0;
  let failed = 0;

  for (let i = 0; i < maxTasks; i++) {
    const order = pendingTasks[i];

    // Find an available station
    const station = findAvailableStation(['gift_wrap'], 'packing', '');
    if (!station) {
      console.warn(`No station available for order ${order.orderId}`);
      failed++;
      tasksFailed.add(1);
      successRate.add(false);
      continue;
    }

    // Process the gift wrap
    const result = processGiftWrap(order, station);

    if (result.success) {
      // Send completion signal to orchestrator
      const signalSuccess = sendGiftWrapCompletedSignal(
        result.orderId,
        result.stationId,
        result.wrapType,
        result.giftMessage
      );

      if (signalSuccess) {
        completed++;
        tasksCompleted.add(1);
        successRate.add(true);
      } else {
        failed++;
        tasksFailed.add(1);
        successRate.add(false);
      }
    } else {
      failed++;
      tasksFailed.add(1);
      successRate.add(false);
    }

    sleep(0.5);  // Brief pause between tasks
  }

  console.log(`\nIteration complete: ${completed} completed, ${failed} failed`);
}

/**
 * Teardown function - summary
 */
export function teardown(data) {
  console.log('\n' + '='.repeat(60));
  console.log('Gift Wrap Simulator Complete');
  console.log('='.repeat(60));
  console.log(`Started: ${data.startTime}`);
  console.log(`Ended: ${new Date().toISOString()}`);
  console.log(`Gift wrap stations used: ${data.giftWrapStations.length}`);
  console.log('='.repeat(60));
}

// Export helper functions for use in full-flow-simulator
export { sendGiftWrapCompletedSignal, processGiftWrap, discoverPendingGiftWrapTasks };
