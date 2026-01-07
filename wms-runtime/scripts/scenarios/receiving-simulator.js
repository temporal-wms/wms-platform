// Receiving Simulator
// Simulates inbound goods receipt operations for WMS end-to-end testing

import { check, sleep, group } from 'k6';
import { Counter, Trend, Rate } from 'k6/metrics';
import {
  RECEIVING_CONFIG,
  SHIPMENT_STATUS,
  ASN_TYPES,
  createInboundShipment,
  discoverPendingShipments,
  startReceiving,
  confirmItemReceipt,
  completeReceiving,
  signalReceivingCompleted,
  processAllPendingShipments,
  simulateReceivingShipment,
} from '../lib/receiving.js';
import { products, generateUUID } from '../lib/data.js';

// Custom metrics
const shipmentsCreated = new Counter('receiving_shipments_created');
const shipmentsReceived = new Counter('receiving_shipments_received');
const itemsReceived = new Counter('receiving_items_received');
const receivingDuration = new Trend('receiving_duration_ms');
const receivingSuccessRate = new Rate('receiving_success_rate');

// Test configuration
export const options = {
  scenarios: {
    receiving_flow: {
      executor: 'ramping-vus',
      startVUs: 1,
      stages: [
        { duration: '30s', target: 3 },   // Ramp up
        { duration: '2m', target: 5 },    // Steady state
        { duration: '30s', target: 0 },   // Ramp down
      ],
      gracefulRampDown: '10s',
    },
  },
  thresholds: {
    'receiving_success_rate': ['rate>0.95'],
    'receiving_duration_ms': ['p(95)<5000'],
    'http_req_failed': ['rate<0.05'],
  },
};

// Configuration
const CONFIG = {
  shipmentCount: parseInt(__ENV.SHIPMENT_COUNT || '5'),
  itemsPerShipment: parseInt(__ENV.ITEMS_PER_SHIPMENT || '10'),
  processExisting: __ENV.PROCESS_EXISTING === 'true',
  skipCreation: __ENV.SKIP_CREATION === 'true',
};

/**
 * Generates a random inbound shipment with items
 */
function generateShipment() {
  const itemCount = Math.floor(Math.random() * CONFIG.itemsPerShipment) + 1;
  const selectedProducts = [];

  for (let i = 0; i < itemCount; i++) {
    const product = products[Math.floor(Math.random() * products.length)];
    selectedProducts.push({
      itemId: `ITEM-${Date.now()}-${i}`,
      sku: product.sku,
      productName: product.productName,
      expectedQuantity: Math.floor(Math.random() * 50) + 10,
      weight: product.weight,
    });
  }

  return {
    asnNumber: `ASN-${Date.now()}-${Math.floor(Math.random() * 10000)}`,
    type: ASN_TYPES.PURCHASE_ORDER,
    vendorId: `VENDOR-${Math.floor(Math.random() * 100) + 1}`,
    expectedArrival: new Date(Date.now() + 86400000).toISOString(),
    dockDoor: `DOCK-${Math.floor(Math.random() * 5) + 1}`,
    items: selectedProducts,
  };
}

/**
 * Main test function
 */
export default function () {
  const vuId = __VU;
  const iterationId = __ITER;

  console.log(`[VU ${vuId}] Starting receiving simulation - iteration ${iterationId}`);

  // Phase 1: Create new shipments (unless skipped)
  if (!CONFIG.skipCreation) {
    group('Create Inbound Shipments', function () {
      const shipmentsToCreate = Math.ceil(CONFIG.shipmentCount / (__VU || 1));

      for (let i = 0; i < shipmentsToCreate; i++) {
        const shipmentData = generateShipment();
        const startTime = Date.now();

        const shipment = createInboundShipment(shipmentData);

        if (shipment) {
          shipmentsCreated.add(1);
          console.log(`[VU ${vuId}] Created shipment: ${shipment.shipmentId || shipment.id}`);

          // Simulate the shipment arriving
          sleep(1);

          // Mark as arrived (in real scenario, this would be truck arrival)
          console.log(`[VU ${vuId}] Shipment arrived at dock`);
        } else {
          console.warn(`[VU ${vuId}] Failed to create shipment`);
        }

        sleep(0.5);
      }
    });
  }

  // Phase 2: Process pending/arrived shipments
  group('Process Arrived Shipments', function () {
    // Discover arrived shipments
    const pendingShipments = discoverPendingShipments(SHIPMENT_STATUS.ARRIVED);
    console.log(`[VU ${vuId}] Found ${pendingShipments.length} shipments ready for receiving`);

    if (pendingShipments.length === 0 && CONFIG.processExisting) {
      // Try to find shipments in other statuses
      const expectedShipments = discoverPendingShipments(SHIPMENT_STATUS.EXPECTED);
      console.log(`[VU ${vuId}] Found ${expectedShipments.length} expected shipments`);
    }

    // Process each arrived shipment
    for (const shipment of pendingShipments.slice(0, 3)) {
      const shipmentId = shipment.shipmentId || shipment.id;
      const startTime = Date.now();

      console.log(`[VU ${vuId}] Starting to receive shipment: ${shipmentId}`);

      // Start receiving
      const started = startReceiving(shipmentId, shipment.dockDoor);
      if (!started) {
        console.warn(`[VU ${vuId}] Failed to start receiving for ${shipmentId}`);
        receivingSuccessRate.add(0);
        continue;
      }

      // Receive each item
      const items = shipment.items || [];
      let itemsReceivedCount = 0;

      for (const item of items) {
        sleep(RECEIVING_CONFIG.receiptConfirmationDelayMs / 1000);

        const confirmed = confirmItemReceipt(shipmentId, item.itemId || item.id, {
          quantity: item.expectedQuantity || item.quantity,
          condition: 'good',
          licensePlate: `LP-${Date.now()}-${item.sku}`,
        });

        if (confirmed) {
          itemsReceivedCount++;
          itemsReceived.add(1);
        }
      }

      console.log(`[VU ${vuId}] Received ${itemsReceivedCount}/${items.length} items for ${shipmentId}`);

      // Complete receiving
      const completed = completeReceiving(shipmentId);
      const duration = Date.now() - startTime;
      receivingDuration.add(duration);

      if (completed) {
        shipmentsReceived.add(1);
        receivingSuccessRate.add(1);

        // Signal orchestrator
        const receivedItems = items.map(item => ({
          itemId: item.itemId || item.id,
          sku: item.sku,
          quantity: item.expectedQuantity || item.quantity,
          licensePlate: `LP-${shipmentId}-${item.sku}`,
        }));

        signalReceivingCompleted(shipmentId, receivedItems);

        console.log(`[VU ${vuId}] Completed receiving for ${shipmentId} in ${duration}ms`);
      } else {
        receivingSuccessRate.add(0);
        console.warn(`[VU ${vuId}] Failed to complete receiving for ${shipmentId}`);
      }

      sleep(1);
    }
  });

  // Brief pause between iterations
  sleep(2);
}

/**
 * Setup function - runs once before test
 */
export function setup() {
  console.log('='.repeat(60));
  console.log('Receiving Simulator - Setup');
  console.log('='.repeat(60));
  console.log(`Shipments per VU: ${CONFIG.shipmentCount}`);
  console.log(`Items per shipment: ${CONFIG.itemsPerShipment}`);
  console.log(`Skip creation: ${CONFIG.skipCreation}`);
  console.log(`Process existing: ${CONFIG.processExisting}`);
  console.log('='.repeat(60));

  return {
    startTime: Date.now(),
  };
}

/**
 * Teardown function - runs once after test
 */
export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;

  console.log('='.repeat(60));
  console.log('Receiving Simulator - Summary');
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
    simulator: 'receiving-simulator',
    metrics: {
      shipments_created: data.metrics.receiving_shipments_created?.values?.count || 0,
      shipments_received: data.metrics.receiving_shipments_received?.values?.count || 0,
      items_received: data.metrics.receiving_items_received?.values?.count || 0,
      success_rate: data.metrics.receiving_success_rate?.values?.rate || 0,
      avg_duration_ms: data.metrics.receiving_duration_ms?.values?.avg || 0,
      p95_duration_ms: data.metrics.receiving_duration_ms?.values?.['p(95)'] || 0,
    },
    thresholds: data.thresholds,
  };

  return {
    'stdout': JSON.stringify(summary, null, 2) + '\n',
    'receiving-results.json': JSON.stringify(summary, null, 2),
  };
}
