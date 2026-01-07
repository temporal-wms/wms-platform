// Billing Simulator
// Simulates billing operations for WMS testing
// Tests: activity recording, invoicing, fee calculations, storage tracking

import { check, sleep, group } from 'k6';
import { Counter, Trend, Rate } from 'k6/metrics';
import {
  recordActivity,
  recordActivitiesBatch,
  getActivity,
  listActivities,
  getActivitySummary,
  createInvoice,
  getInvoice,
  listInvoices,
  finalizeInvoice,
  payInvoice,
  voidInvoice,
  calculateFees,
  recordStorage,
  recordPickActivity,
  recordPackActivity,
  recordShippingActivity,
  recordReceivingActivity,
  recordGiftWrapActivity,
  recordSpecialHandlingActivities,
  createBillingContext,
  checkHealth,
  ACTIVITY_TYPES,
  INVOICE_STATUS,
  REFERENCE_TYPES,
  PAYMENT_METHODS,
} from '../lib/billing.js';
import { BILLING_CONFIG, SELLER_CONFIG } from '../lib/config.js';

// Custom metrics
const activitiesRecorded = new Counter('billing_activities_recorded');
const invoicesCreated = new Counter('billing_invoices_created');
const invoicesFinalized = new Counter('billing_invoices_finalized');
const invoicesPaid = new Counter('billing_invoices_paid');
const storageRecorded = new Counter('billing_storage_recorded');
const billingDuration = new Trend('billing_operation_duration_ms');
const billingSuccessRate = new Rate('billing_success_rate');

// Test configuration
export const options = {
  scenarios: {
    billing_flow: {
      executor: 'ramping-vus',
      startVUs: 1,
      stages: [
        { duration: '30s', target: 2 },   // Ramp up
        { duration: '2m', target: 3 },    // Steady state
        { duration: '30s', target: 0 },   // Ramp down
      ],
      gracefulRampDown: '10s',
    },
  },
  thresholds: {
    'billing_success_rate': ['rate>0.95'],
    'billing_operation_duration_ms': ['p(95)<3000'],
    'http_req_failed': ['rate<0.05'],
  },
};

// Configuration from environment variables
const CONFIG = {
  activitiesPerIteration: parseInt(__ENV.ACTIVITIES_PER_ITERATION || '10'),
  enableInvoiceFlow: __ENV.ENABLE_INVOICE_FLOW !== 'false',
  enableBatchRecording: __ENV.ENABLE_BATCH_RECORDING === 'true',
  billingPeriodDays: parseInt(__ENV.BILLING_PERIOD_DAYS || '30'),
  testSellerId: __ENV.TEST_SELLER_ID || `SLR-TEST-${Date.now()}`,
  testTenantId: __ENV.TEST_TENANT_ID || SELLER_CONFIG.defaultTenantId,
  testFacilityId: __ENV.TEST_FACILITY_ID || BILLING_CONFIG.defaultFacilityId,
};

// Test billing context
const TEST_CONTEXT = {
  tenantId: CONFIG.testTenantId,
  sellerId: CONFIG.testSellerId,
  facilityId: CONFIG.testFacilityId,
};

// Activity types for simulation
const ACTIVITY_CONFIGS = [
  { type: ACTIVITY_TYPES.PICK, unitPrice: 0.25, minQty: 1, maxQty: 20, refType: REFERENCE_TYPES.ORDER },
  { type: ACTIVITY_TYPES.PACK, unitPrice: 1.50, minQty: 1, maxQty: 5, refType: REFERENCE_TYPES.ORDER },
  { type: ACTIVITY_TYPES.RECEIVING, unitPrice: 0.15, minQty: 10, maxQty: 100, refType: REFERENCE_TYPES.RECEIVING },
  { type: ACTIVITY_TYPES.SHIPPING, unitPrice: 5.00, minQty: 1, maxQty: 3, refType: REFERENCE_TYPES.SHIPMENT },
  { type: ACTIVITY_TYPES.GIFT_WRAP, unitPrice: 2.50, minQty: 1, maxQty: 3, refType: REFERENCE_TYPES.ORDER },
  { type: ACTIVITY_TYPES.HAZMAT, unitPrice: 5.00, minQty: 1, maxQty: 2, refType: REFERENCE_TYPES.ORDER },
  { type: ACTIVITY_TYPES.FRAGILE, unitPrice: 1.50, minQty: 1, maxQty: 5, refType: REFERENCE_TYPES.ORDER },
  { type: ACTIVITY_TYPES.OVERSIZED, unitPrice: 10.00, minQty: 1, maxQty: 2, refType: REFERENCE_TYPES.ORDER },
];

/**
 * Generates a random reference ID
 */
function generateRefId(refType) {
  const timestamp = Date.now();
  const rand = Math.floor(Math.random() * 10000);
  switch (refType) {
    case REFERENCE_TYPES.ORDER:
      return `ORD-${timestamp}-${rand}`;
    case REFERENCE_TYPES.SHIPMENT:
      return `SHP-${timestamp}-${rand}`;
    case REFERENCE_TYPES.RECEIVING:
      return `RCV-${timestamp}-${rand}`;
    case REFERENCE_TYPES.INVENTORY:
      return `INV-${timestamp}-${rand}`;
    default:
      return `REF-${timestamp}-${rand}`;
  }
}

/**
 * Generates a random activity
 */
function generateActivity() {
  const config = ACTIVITY_CONFIGS[Math.floor(Math.random() * ACTIVITY_CONFIGS.length)];
  const quantity = config.minQty + Math.floor(Math.random() * (config.maxQty - config.minQty + 1));

  return {
    ...TEST_CONTEXT,
    type: config.type,
    description: `Test ${config.type} activity`,
    quantity: quantity,
    unitPrice: config.unitPrice,
    referenceType: config.refType,
    referenceId: generateRefId(config.refType),
    metadata: {
      simulatedAt: new Date().toISOString(),
      vuId: __VU,
    },
  };
}

/**
 * Generates batch of activities
 */
function generateActivityBatch(count) {
  const activities = [];
  for (let i = 0; i < count; i++) {
    activities.push(generateActivity());
  }
  return activities;
}

/**
 * Main test function - Billing operations simulation
 */
export default function () {
  const vuId = __VU;
  const iterationId = __ITER;

  console.log(`[VU ${vuId}] Starting billing simulation - iteration ${iterationId}`);

  // Health check
  group('Health Check', function () {
    const healthy = checkHealth();
    check(healthy, {
      'billing service is healthy': (h) => h === true,
    });
    if (!healthy) {
      console.error('Billing service health check failed');
      return;
    }
  });

  // Track recorded activities for verification
  const recordedActivities = [];

  // Phase 1: Record individual activities
  group('Record Individual Activities', function () {
    const activityCount = Math.ceil(CONFIG.activitiesPerIteration / 2);

    for (let i = 0; i < activityCount; i++) {
      const startTime = Date.now();
      const activityData = generateActivity();

      const activity = recordActivity(activityData);

      if (activity && activity.activityId) {
        activitiesRecorded.add(1);
        billingSuccessRate.add(true);
        recordedActivities.push(activity);
        console.log(`[VU ${vuId}] Recorded activity: ${activity.activityId} (${activityData.type})`);
      } else {
        billingSuccessRate.add(false);
        console.warn(`[VU ${vuId}] Failed to record activity: ${activityData.type}`);
      }

      billingDuration.add(Date.now() - startTime);
      sleep(BILLING_CONFIG.simulationDelayMs / 1000);
    }
  });

  // Phase 2: Record batch activities
  if (CONFIG.enableBatchRecording) {
    group('Record Batch Activities', function () {
      const batchSize = Math.ceil(CONFIG.activitiesPerIteration / 2);
      const startTime = Date.now();

      const activities = generateActivityBatch(batchSize);
      const result = recordActivitiesBatch(activities);

      if (result.success) {
        activitiesRecorded.add(result.recorded || batchSize);
        billingSuccessRate.add(true);
        console.log(`[VU ${vuId}] Recorded batch: ${result.recorded || batchSize} activities`);
      } else {
        billingSuccessRate.add(false);
        console.warn(`[VU ${vuId}] Failed to record batch activities`);
      }

      billingDuration.add(Date.now() - startTime);
      sleep(BILLING_CONFIG.simulationDelayMs / 1000);
    });
  }

  // Phase 3: Record storage calculation
  group('Record Storage Calculation', function () {
    const startTime = Date.now();

    const storageData = {
      ...TEST_CONTEXT,
      calculationDate: new Date(),
      totalCubicFeet: 100 + Math.floor(Math.random() * 900),
      ratePerCubicFt: 0.05,
      storageBreakdown: {
        standard: 80 + Math.floor(Math.random() * 100),
        oversized: Math.floor(Math.random() * 50),
        hazmat: Math.floor(Math.random() * 20),
        coldChain: Math.floor(Math.random() * 30),
      },
    };

    const storage = recordStorage(storageData);

    if (storage) {
      storageRecorded.add(1);
      billingSuccessRate.add(true);
      console.log(`[VU ${vuId}] Recorded storage: ${storageData.totalCubicFeet} cubic feet`);
    } else {
      billingSuccessRate.add(false);
      console.warn(`[VU ${vuId}] Failed to record storage calculation`);
    }

    billingDuration.add(Date.now() - startTime);
    sleep(BILLING_CONFIG.simulationDelayMs / 1000);
  });

  // Phase 4: Use convenience functions
  group('Convenience Functions', function () {
    const orderId = generateRefId(REFERENCE_TYPES.ORDER);
    const shipmentId = generateRefId(REFERENCE_TYPES.SHIPMENT);

    // Pick activity
    const pickResult = recordPickActivity(TEST_CONTEXT, orderId, 5, 0.25);
    if (pickResult) {
      activitiesRecorded.add(1);
      billingSuccessRate.add(true);
    }
    sleep(BILLING_CONFIG.simulationDelayMs / 1000 / 2);

    // Pack activity
    const packResult = recordPackActivity(TEST_CONTEXT, orderId, 1, 1.50);
    if (packResult) {
      activitiesRecorded.add(1);
      billingSuccessRate.add(true);
    }
    sleep(BILLING_CONFIG.simulationDelayMs / 1000 / 2);

    // Shipping activity
    const shippingResult = recordShippingActivity(TEST_CONTEXT, shipmentId, 8.50, 5);
    if (shippingResult) {
      activitiesRecorded.add(1);
      billingSuccessRate.add(true);
    }
    sleep(BILLING_CONFIG.simulationDelayMs / 1000 / 2);

    // Special handling
    const specialResult = recordSpecialHandlingActivities(TEST_CONTEXT, orderId, ['fragile', 'high_value']);
    if (specialResult.recorded > 0) {
      activitiesRecorded.add(specialResult.recorded);
      billingSuccessRate.add(true);
    }
  });

  // Phase 5: Query activities
  group('Query Activities', function () {
    // List activities
    const activities = listActivities(CONFIG.testSellerId, { page: 1, pageSize: 20 });
    check(activities, {
      'activities list returned': (a) => Array.isArray(a),
    });

    // Get activity summary
    const periodStart = new Date();
    periodStart.setDate(periodStart.getDate() - CONFIG.billingPeriodDays);
    const periodEnd = new Date();

    const summary = getActivitySummary(CONFIG.testSellerId, periodStart, periodEnd);
    check(summary, {
      'activity summary returned': (s) => s !== null,
    });

    // Verify individual activity
    if (recordedActivities.length > 0) {
      const activity = getActivity(recordedActivities[0].activityId);
      check(activity, {
        'activity retrieved': (a) => a !== null,
        'activity has correct ID': (a) => a && a.activityId === recordedActivities[0].activityId,
      });
    }

    sleep(BILLING_CONFIG.simulationDelayMs / 1000);
  });

  // Phase 6: Invoice lifecycle
  if (CONFIG.enableInvoiceFlow) {
    group('Invoice Lifecycle', function () {
      const periodStart = new Date();
      periodStart.setDate(periodStart.getDate() - CONFIG.billingPeriodDays);
      const periodEnd = new Date();

      // Create invoice
      const startTime = Date.now();
      const invoice = createInvoice({
        tenantId: CONFIG.testTenantId,
        sellerId: CONFIG.testSellerId,
        periodStart: periodStart,
        periodEnd: periodEnd,
        sellerName: 'Test Seller Company',
        sellerEmail: 'billing@testseller.com',
        taxRate: 8.25,
        notes: `Test invoice created by VU ${vuId}`,
      });

      if (invoice && invoice.invoiceId) {
        invoicesCreated.add(1);
        billingSuccessRate.add(true);
        console.log(`[VU ${vuId}] Created invoice: ${invoice.invoiceId}`);

        billingDuration.add(Date.now() - startTime);
        sleep(BILLING_CONFIG.simulationDelayMs / 1000);

        // Get invoice
        const fetchedInvoice = getInvoice(invoice.invoiceId);
        check(fetchedInvoice, {
          'invoice retrieved': (i) => i !== null,
          'invoice is draft': (i) => i && i.status === INVOICE_STATUS.DRAFT,
        });

        // Finalize invoice
        const finalized = finalizeInvoice(invoice.invoiceId);
        if (finalized) {
          invoicesFinalized.add(1);
          billingSuccessRate.add(true);
          console.log(`[VU ${vuId}] Finalized invoice: ${invoice.invoiceId}`);

          sleep(BILLING_CONFIG.simulationDelayMs / 1000);

          // Pay invoice
          const paid = payInvoice(invoice.invoiceId, {
            paymentMethod: PAYMENT_METHODS.CREDIT_CARD,
            paymentRef: `PAY-TEST-${Date.now()}`,
          });

          if (paid) {
            invoicesPaid.add(1);
            billingSuccessRate.add(true);
            console.log(`[VU ${vuId}] Paid invoice: ${invoice.invoiceId}`);
          } else {
            billingSuccessRate.add(false);
          }
        } else {
          billingSuccessRate.add(false);
        }
      } else {
        billingSuccessRate.add(false);
        console.warn(`[VU ${vuId}] Failed to create invoice`);
      }

      // List invoices
      const invoices = listInvoices(CONFIG.testSellerId, { status: INVOICE_STATUS.PAID });
      check(invoices, {
        'invoice list returned': (i) => Array.isArray(i),
      });
    });
  }

  // Phase 7: Fee calculation preview
  group('Fee Calculation Preview', function () {
    const feeData = {
      sellerId: CONFIG.testSellerId,
      activities: [
        { type: ACTIVITY_TYPES.PICK, quantity: 100 },
        { type: ACTIVITY_TYPES.PACK, quantity: 20 },
        { type: ACTIVITY_TYPES.STORAGE, quantity: 500 },
      ],
      feeSchedule: {
        pickFeePerUnit: 0.25,
        packFeePerOrder: 1.50,
        storageFeePerCubicFtPerDay: 0.05,
      },
    };

    const fees = calculateFees(feeData);
    check(fees, {
      'fee calculation returned': (f) => f !== null,
    });

    if (fees) {
      console.log(`[VU ${vuId}] Fee preview calculated: $${fees.total || 'N/A'}`);
    }
  });

  console.log(`[VU ${vuId}] Completed billing simulation - recorded ${recordedActivities.length} activities`);
}

/**
 * Setup function - runs once before all VUs
 */
export function setup() {
  console.log('Starting billing simulator...');
  console.log(`Configuration:`);
  console.log(`  - Activities per iteration: ${CONFIG.activitiesPerIteration}`);
  console.log(`  - Invoice flow enabled: ${CONFIG.enableInvoiceFlow}`);
  console.log(`  - Batch recording enabled: ${CONFIG.enableBatchRecording}`);
  console.log(`  - Billing period days: ${CONFIG.billingPeriodDays}`);
  console.log(`  - Test seller ID: ${CONFIG.testSellerId}`);
  console.log(`  - Test tenant ID: ${CONFIG.testTenantId}`);

  // Health check
  const healthy = checkHealth();
  if (!healthy) {
    console.error('Billing service is not healthy - tests may fail');
  }

  return { startTime: Date.now() };
}

/**
 * Teardown function - runs once after all VUs complete
 */
export function teardown(data) {
  const duration = Date.now() - data.startTime;
  console.log(`Billing simulator completed in ${duration}ms`);
}
