// Seller Simulator
// Simulates seller account lifecycle operations for WMS testing
// Tests: creation, activation, facility assignment, channel connection, API key generation

import { check, sleep, group } from 'k6';
import { Counter, Trend, Rate } from 'k6/metrics';
import {
  createSeller,
  getSeller,
  listSellers,
  searchSellers,
  activateSeller,
  suspendSeller,
  assignFacility,
  removeFacility,
  updateFeeSchedule,
  connectChannel,
  disconnectChannel,
  generateApiKey,
  listApiKeys,
  revokeApiKey,
  setupTestSeller,
  getDefaultFeeSchedule,
  getPremiumFeeSchedule,
  checkHealth,
  BILLING_CYCLES,
  SELLER_STATUS,
  CHANNEL_TYPES,
} from '../lib/sellers.js';
import { SELLER_CONFIG } from '../lib/config.js';

// Custom metrics
const sellersCreated = new Counter('sellers_created');
const sellersActivated = new Counter('sellers_activated');
const sellersSuspended = new Counter('sellers_suspended');
const facilitiesAssigned = new Counter('facilities_assigned');
const channelsConnected = new Counter('channels_connected');
const apiKeysGenerated = new Counter('api_keys_generated');
const sellerSetupDuration = new Trend('seller_setup_duration_ms');
const sellerOperationSuccessRate = new Rate('seller_operation_success_rate');

// Test configuration
export const options = {
  scenarios: {
    seller_lifecycle: {
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
    'seller_operation_success_rate': ['rate>0.95'],
    'seller_setup_duration_ms': ['p(95)<5000'],
    'http_req_failed': ['rate<0.05'],
  },
};

// Configuration from environment variables
const CONFIG = {
  sellerCount: parseInt(__ENV.SELLER_COUNT || '3'),
  enableChannelConnect: __ENV.ENABLE_CHANNEL_CONNECT !== 'false',
  enableApiKeyGeneration: __ENV.ENABLE_API_KEY_GENERATION !== 'false',
  enableFeeScheduleUpdate: __ENV.ENABLE_FEE_SCHEDULE !== 'false',
  enableSuspendTest: __ENV.ENABLE_SUSPEND_TEST === 'true',
  facilityId: __ENV.TEST_FACILITY_ID || 'FAC-001',
  tenantId: __ENV.TEST_TENANT_ID || SELLER_CONFIG.defaultTenantId,
};

// Test data
const TEST_CHANNELS = [
  { channelType: CHANNEL_TYPES.SHOPIFY, storeName: 'Test Shopify Store', storeUrl: 'https://test.myshopify.com' },
  { channelType: CHANNEL_TYPES.AMAZON, storeName: 'Test Amazon FBA', storeUrl: '' },
  { channelType: CHANNEL_TYPES.WOOCOMMERCE, storeName: 'Test WooCommerce', storeUrl: 'https://test-woo.example.com' },
  { channelType: CHANNEL_TYPES.EBAY, storeName: 'Test eBay Store', storeUrl: 'https://ebay.com/usr/teststore' },
];

/**
 * Generates unique test seller data
 */
function generateSellerData(index) {
  const timestamp = Date.now();
  const vuId = __VU || 1;
  const billingCycles = [BILLING_CYCLES.DAILY, BILLING_CYCLES.WEEKLY, BILLING_CYCLES.MONTHLY];

  return {
    tenantId: CONFIG.tenantId,
    companyName: `Test Company ${vuId}-${index}-${timestamp}`,
    contactName: `Contact Person ${index}`,
    contactEmail: `test-${vuId}-${index}-${timestamp}@example.com`,
    contactPhone: `+1-555-${String(index).padStart(4, '0')}`,
    billingCycle: billingCycles[index % billingCycles.length],
  };
}

/**
 * Gets a random channel for testing
 */
function getRandomChannel() {
  return TEST_CHANNELS[Math.floor(Math.random() * TEST_CHANNELS.length)];
}

/**
 * Main test function - Seller lifecycle simulation
 */
export default function () {
  const vuId = __VU;
  const iterationId = __ITER;

  console.log(`[VU ${vuId}] Starting seller simulation - iteration ${iterationId}`);

  // Health check
  group('Health Check', function () {
    const healthy = checkHealth();
    check(healthy, {
      'seller service is healthy': (h) => h === true,
    });
    if (!healthy) {
      console.error('Seller service health check failed');
      return;
    }
  });

  // Track created sellers for cleanup/further testing
  const createdSellers = [];

  // Phase 1: Create sellers
  group('Create Sellers', function () {
    for (let i = 0; i < CONFIG.sellerCount; i++) {
      const startTime = Date.now();
      const sellerData = generateSellerData(i);

      const seller = createSeller(sellerData);

      if (seller && seller.sellerId) {
        sellersCreated.add(1);
        sellerOperationSuccessRate.add(true);
        createdSellers.push(seller);
        console.log(`[VU ${vuId}] Created seller: ${seller.sellerId} - ${sellerData.companyName}`);
      } else {
        sellerOperationSuccessRate.add(false);
        console.warn(`[VU ${vuId}] Failed to create seller: ${sellerData.companyName}`);
      }

      sellerSetupDuration.add(Date.now() - startTime);
      sleep(SELLER_CONFIG.simulationDelayMs / 1000);
    }
  });

  // Phase 2: Activate sellers
  group('Activate Sellers', function () {
    for (const seller of createdSellers) {
      const success = activateSeller(seller.sellerId);

      if (success) {
        sellersActivated.add(1);
        sellerOperationSuccessRate.add(true);
        console.log(`[VU ${vuId}] Activated seller: ${seller.sellerId}`);
      } else {
        sellerOperationSuccessRate.add(false);
        console.warn(`[VU ${vuId}] Failed to activate seller: ${seller.sellerId}`);
      }

      sleep(SELLER_CONFIG.simulationDelayMs / 1000);
    }
  });

  // Phase 3: Assign facilities
  group('Assign Facilities', function () {
    for (const seller of createdSellers) {
      const success = assignFacility(seller.sellerId, {
        facilityId: CONFIG.facilityId,
        facilityName: 'Test Fulfillment Center',
        warehouseIds: ['WH-001'],
        allocatedSpace: 1000 + Math.floor(Math.random() * 4000),
        isDefault: true,
      });

      if (success) {
        facilitiesAssigned.add(1);
        sellerOperationSuccessRate.add(true);
        console.log(`[VU ${vuId}] Assigned facility to seller: ${seller.sellerId}`);
      } else {
        sellerOperationSuccessRate.add(false);
        console.warn(`[VU ${vuId}] Failed to assign facility to seller: ${seller.sellerId}`);
      }

      sleep(SELLER_CONFIG.simulationDelayMs / 1000);
    }
  });

  // Phase 4: Update fee schedules
  if (CONFIG.enableFeeScheduleUpdate) {
    group('Update Fee Schedules', function () {
      for (let i = 0; i < createdSellers.length; i++) {
        const seller = createdSellers[i];
        // Alternate between default and premium
        const feeSchedule = i % 2 === 0 ? getDefaultFeeSchedule() : getPremiumFeeSchedule();

        const success = updateFeeSchedule(seller.sellerId, feeSchedule);

        if (success) {
          sellerOperationSuccessRate.add(true);
          console.log(`[VU ${vuId}] Updated fee schedule for seller: ${seller.sellerId}`);
        } else {
          sellerOperationSuccessRate.add(false);
        }

        sleep(SELLER_CONFIG.simulationDelayMs / 1000);
      }
    });
  }

  // Phase 5: Connect channels
  if (CONFIG.enableChannelConnect) {
    group('Connect Channels', function () {
      for (const seller of createdSellers) {
        const channelData = getRandomChannel();
        const channel = connectChannel(seller.sellerId, channelData);

        if (channel) {
          channelsConnected.add(1);
          sellerOperationSuccessRate.add(true);
          console.log(`[VU ${vuId}] Connected ${channelData.channelType} channel to seller: ${seller.sellerId}`);
        } else {
          sellerOperationSuccessRate.add(false);
          console.warn(`[VU ${vuId}] Failed to connect channel to seller: ${seller.sellerId}`);
        }

        sleep(SELLER_CONFIG.simulationDelayMs / 1000);
      }
    });
  }

  // Phase 6: Generate API keys
  if (CONFIG.enableApiKeyGeneration) {
    group('Generate API Keys', function () {
      for (const seller of createdSellers) {
        const apiKey = generateApiKey(seller.sellerId, {
          name: `Test Key ${Date.now()}`,
          scopes: ['orders:read', 'inventory:read', 'shipments:read'],
        });

        if (apiKey) {
          apiKeysGenerated.add(1);
          sellerOperationSuccessRate.add(true);
          console.log(`[VU ${vuId}] Generated API key for seller: ${seller.sellerId} (prefix: ${apiKey.prefix || 'N/A'})`);
        } else {
          sellerOperationSuccessRate.add(false);
          console.warn(`[VU ${vuId}] Failed to generate API key for seller: ${seller.sellerId}`);
        }

        sleep(SELLER_CONFIG.simulationDelayMs / 1000);
      }
    });
  }

  // Phase 7: Verify sellers (read operations)
  group('Verify Sellers', function () {
    // Get individual sellers
    for (const seller of createdSellers) {
      const fetchedSeller = getSeller(seller.sellerId);

      const verified = check(fetchedSeller, {
        'seller exists': (s) => s !== null,
        'seller has correct ID': (s) => s && s.sellerId === seller.sellerId,
        'seller is active': (s) => s && s.status === SELLER_STATUS.ACTIVE,
      });

      sellerOperationSuccessRate.add(verified);
      sleep(SELLER_CONFIG.simulationDelayMs / 1000 / 2);
    }

    // List sellers
    const sellerList = listSellers({ tenantId: CONFIG.tenantId, status: SELLER_STATUS.ACTIVE });
    check(sellerList, {
      'seller list not empty': (list) => list.length > 0,
    });

    // Search sellers
    if (createdSellers.length > 0) {
      const searchResults = searchSellers(createdSellers[0].companyName || 'Test');
      check(searchResults, {
        'search returns results': (results) => Array.isArray(results),
      });
    }
  });

  // Phase 8: Test suspend/reactivate (optional)
  if (CONFIG.enableSuspendTest && createdSellers.length > 0) {
    group('Suspend and Reactivate', function () {
      const testSeller = createdSellers[0];

      // Suspend
      const suspended = suspendSeller(testSeller.sellerId, 'Testing suspension flow');
      if (suspended) {
        sellersSuspended.add(1);
        sellerOperationSuccessRate.add(true);
        console.log(`[VU ${vuId}] Suspended seller: ${testSeller.sellerId}`);

        // Verify suspended status
        sleep(SELLER_CONFIG.simulationDelayMs / 1000);
        const suspendedSeller = getSeller(testSeller.sellerId);
        check(suspendedSeller, {
          'seller is suspended': (s) => s && s.status === SELLER_STATUS.SUSPENDED,
        });

        // Reactivate
        sleep(SELLER_CONFIG.simulationDelayMs / 1000);
        const reactivated = activateSeller(testSeller.sellerId);
        if (reactivated) {
          sellerOperationSuccessRate.add(true);
          console.log(`[VU ${vuId}] Reactivated seller: ${testSeller.sellerId}`);
        }
      } else {
        sellerOperationSuccessRate.add(false);
      }
    });
  }

  console.log(`[VU ${vuId}] Completed seller simulation - created ${createdSellers.length} sellers`);
}

/**
 * Setup function - runs once before all VUs
 */
export function setup() {
  console.log('Starting seller simulator...');
  console.log(`Configuration:`);
  console.log(`  - Seller count per VU: ${CONFIG.sellerCount}`);
  console.log(`  - Channel connect: ${CONFIG.enableChannelConnect}`);
  console.log(`  - API key generation: ${CONFIG.enableApiKeyGeneration}`);
  console.log(`  - Fee schedule update: ${CONFIG.enableFeeScheduleUpdate}`);
  console.log(`  - Facility ID: ${CONFIG.facilityId}`);
  console.log(`  - Tenant ID: ${CONFIG.tenantId}`);

  // Health check
  const healthy = checkHealth();
  if (!healthy) {
    console.error('Seller service is not healthy - tests may fail');
  }

  return { startTime: Date.now() };
}

/**
 * Teardown function - runs once after all VUs complete
 */
export function teardown(data) {
  const duration = Date.now() - data.startTime;
  console.log(`Seller simulator completed in ${duration}ms`);
}
