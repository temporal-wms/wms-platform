// K6 Seller Service Helper Library
// Provides functions for seller account management and setup

import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URLS, ENDPOINTS, HTTP_PARAMS, SELLER_CONFIG } from './config.js';

// Billing cycle constants
export const BILLING_CYCLES = {
  DAILY: 'daily',
  WEEKLY: 'weekly',
  MONTHLY: 'monthly',
};

// Seller status constants
export const SELLER_STATUS = {
  PENDING: 'pending',
  ACTIVE: 'active',
  SUSPENDED: 'suspended',
  CLOSED: 'closed',
};

// Channel types
export const CHANNEL_TYPES = {
  SHOPIFY: 'shopify',
  AMAZON: 'amazon',
  EBAY: 'ebay',
  WOOCOMMERCE: 'woocommerce',
};

/**
 * Creates a new seller account
 * @param {Object} sellerData - Seller creation data
 * @returns {Object} Created seller or null
 */
export function createSeller(sellerData) {
  const url = `${BASE_URLS.sellers}${ENDPOINTS.sellers.create}`;
  const payload = JSON.stringify({
    tenantId: sellerData.tenantId || SELLER_CONFIG.defaultTenantId,
    facilityId: sellerData.facilityId,
    warehouseId: sellerData.warehouseId,
    companyName: sellerData.companyName,
    contactName: sellerData.contactName,
    contactEmail: sellerData.contactEmail,
    contactPhone: sellerData.contactPhone || '',
    billingCycle: sellerData.billingCycle || SELLER_CONFIG.defaultBillingCycle,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'create seller status 201': (r) => r.status === 201 || r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to create seller: ${response.status} - ${response.body}`);
    return null;
  }

  try {
    const data = JSON.parse(response.body);
    return data.data || data;
  } catch (e) {
    console.error(`Failed to parse seller response: ${e.message}`);
    return null;
  }
}

/**
 * Gets a seller by ID
 * @param {string} sellerId - The seller ID
 * @returns {Object|null} The seller or null
 */
export function getSeller(sellerId) {
  const url = `${BASE_URLS.sellers}${ENDPOINTS.sellers.get(sellerId)}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get seller status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to get seller ${sellerId}: ${response.status}`);
    return null;
  }

  try {
    const data = JSON.parse(response.body);
    return data.data || data;
  } catch (e) {
    console.error(`Failed to parse seller response: ${e.message}`);
    return null;
  }
}

/**
 * Lists sellers with optional filters
 * @param {Object} filters - Filter options
 * @returns {Array} Array of sellers
 */
export function listSellers(filters = {}) {
  const params = new URLSearchParams();
  if (filters.page) params.append('page', filters.page);
  if (filters.pageSize) params.append('pageSize', filters.pageSize);
  if (filters.tenantId) params.append('tenantId', filters.tenantId);
  if (filters.status) params.append('status', filters.status);
  if (filters.facilityId) params.append('facilityId', filters.facilityId);

  const queryString = params.toString();
  const url = `${BASE_URLS.sellers}${ENDPOINTS.sellers.list}${queryString ? '?' + queryString : ''}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'list sellers status 200': (r) => r.status === 200,
  });

  if (!success || !response.body) {
    console.warn(`Failed to list sellers: ${response.status}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return data.data || data.sellers || [];
  } catch (e) {
    console.error(`Failed to parse sellers response: ${e.message}`);
    return [];
  }
}

/**
 * Searches sellers by company name or email
 * @param {string} query - Search query
 * @returns {Array} Array of matching sellers
 */
export function searchSellers(query) {
  const url = `${BASE_URLS.sellers}${ENDPOINTS.sellers.search}?q=${encodeURIComponent(query)}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'search sellers status 200': (r) => r.status === 200,
  });

  if (!success || !response.body) {
    console.warn(`Failed to search sellers: ${response.status}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return data.data || data.sellers || [];
  } catch (e) {
    console.error(`Failed to parse search response: ${e.message}`);
    return [];
  }
}

/**
 * Activates a seller account
 * @param {string} sellerId - The seller ID
 * @returns {boolean} True if successful
 */
export function activateSeller(sellerId) {
  const url = `${BASE_URLS.sellers}${ENDPOINTS.sellers.activate(sellerId)}`;
  const response = http.put(url, null, HTTP_PARAMS);

  const success = check(response, {
    'activate seller status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to activate seller ${sellerId}: ${response.status} - ${response.body}`);
  }

  return success;
}

/**
 * Suspends a seller account
 * @param {string} sellerId - The seller ID
 * @param {string} reason - Suspension reason
 * @returns {boolean} True if successful
 */
export function suspendSeller(sellerId, reason) {
  const url = `${BASE_URLS.sellers}${ENDPOINTS.sellers.suspend(sellerId)}`;
  const payload = JSON.stringify({ reason });

  const response = http.put(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'suspend seller status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to suspend seller ${sellerId}: ${response.status}`);
  }

  return success;
}

/**
 * Closes a seller account
 * @param {string} sellerId - The seller ID
 * @param {string} reason - Closure reason
 * @returns {boolean} True if successful
 */
export function closeSeller(sellerId, reason) {
  const url = `${BASE_URLS.sellers}${ENDPOINTS.sellers.close(sellerId)}`;
  const payload = JSON.stringify({ reason });

  const response = http.put(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'close seller status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to close seller ${sellerId}: ${response.status}`);
  }

  return success;
}

/**
 * Assigns a facility to a seller
 * @param {string} sellerId - The seller ID
 * @param {Object} facilityData - Facility assignment data
 * @returns {boolean} True if successful
 */
export function assignFacility(sellerId, facilityData) {
  const url = `${BASE_URLS.sellers}${ENDPOINTS.sellers.assignFacility(sellerId)}`;
  const payload = JSON.stringify({
    facilityId: facilityData.facilityId,
    facilityName: facilityData.facilityName || facilityData.facilityId,
    warehouseIds: facilityData.warehouseIds || [],
    allocatedSpace: facilityData.allocatedSpace || 1000,
    isDefault: facilityData.isDefault !== false,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'assign facility status 201': (r) => r.status === 201 || r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to assign facility to seller ${sellerId}: ${response.status}`);
  }

  return success;
}

/**
 * Removes a facility assignment from a seller
 * @param {string} sellerId - The seller ID
 * @param {string} facilityId - The facility ID to remove
 * @returns {boolean} True if successful
 */
export function removeFacility(sellerId, facilityId) {
  const url = `${BASE_URLS.sellers}${ENDPOINTS.sellers.removeFacility(sellerId, facilityId)}`;
  const response = http.del(url, null, HTTP_PARAMS);

  const success = check(response, {
    'remove facility status 200': (r) => r.status === 200 || r.status === 204,
  });

  if (!success) {
    console.warn(`Failed to remove facility from seller ${sellerId}: ${response.status}`);
  }

  return success;
}

/**
 * Updates a seller's fee schedule
 * @param {string} sellerId - The seller ID
 * @param {Object} feeSchedule - Fee schedule data
 * @returns {boolean} True if successful
 */
export function updateFeeSchedule(sellerId, feeSchedule) {
  const url = `${BASE_URLS.sellers}${ENDPOINTS.sellers.updateFeeSchedule(sellerId)}`;
  const payload = JSON.stringify(feeSchedule);

  const response = http.put(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'update fee schedule status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to update fee schedule for seller ${sellerId}: ${response.status}`);
  }

  return success;
}

/**
 * Connects a sales channel to a seller
 * @param {string} sellerId - The seller ID
 * @param {Object} channelData - Channel connection data
 * @returns {Object|null} Channel info or null
 */
export function connectChannel(sellerId, channelData) {
  const url = `${BASE_URLS.sellers}${ENDPOINTS.sellers.connectChannel(sellerId)}`;
  const payload = JSON.stringify({
    channelType: channelData.channelType,
    storeName: channelData.storeName,
    storeUrl: channelData.storeUrl || '',
    credentials: channelData.credentials || {},
    syncSettings: channelData.syncSettings || {
      autoImportOrders: true,
      autoSyncInventory: true,
      autoPushTracking: true,
      inventorySyncMinutes: 15,
    },
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'connect channel status 201': (r) => r.status === 201 || r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to connect channel for seller ${sellerId}: ${response.status}`);
    return null;
  }

  try {
    const data = JSON.parse(response.body);
    return data.data || data;
  } catch (e) {
    return null;
  }
}

/**
 * Disconnects a sales channel from a seller
 * @param {string} sellerId - The seller ID
 * @param {string} channelId - The channel ID to disconnect
 * @returns {boolean} True if successful
 */
export function disconnectChannel(sellerId, channelId) {
  const url = `${BASE_URLS.sellers}${ENDPOINTS.sellers.disconnectChannel(sellerId, channelId)}`;
  const response = http.del(url, null, HTTP_PARAMS);

  const success = check(response, {
    'disconnect channel status 200': (r) => r.status === 200 || r.status === 204,
  });

  if (!success) {
    console.warn(`Failed to disconnect channel from seller ${sellerId}: ${response.status}`);
  }

  return success;
}

/**
 * Generates an API key for a seller
 * @param {string} sellerId - The seller ID
 * @param {Object} keyData - API key data
 * @returns {Object|null} API key info or null (raw key only returned once)
 */
export function generateApiKey(sellerId, keyData) {
  const url = `${BASE_URLS.sellers}${ENDPOINTS.sellers.generateApiKey(sellerId)}`;
  const payload = JSON.stringify({
    name: keyData.name,
    scopes: keyData.scopes || ['orders:read', 'inventory:read'],
    expiresAt: keyData.expiresAt || null,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'generate api key status 201': (r) => r.status === 201 || r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to generate API key for seller ${sellerId}: ${response.status}`);
    return null;
  }

  try {
    const data = JSON.parse(response.body);
    return data.data || data;
  } catch (e) {
    return null;
  }
}

/**
 * Lists API keys for a seller (without sensitive data)
 * @param {string} sellerId - The seller ID
 * @returns {Array} Array of API key metadata
 */
export function listApiKeys(sellerId) {
  const url = `${BASE_URLS.sellers}${ENDPOINTS.sellers.listApiKeys(sellerId)}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'list api keys status 200': (r) => r.status === 200,
  });

  if (!success || !response.body) {
    console.warn(`Failed to list API keys for seller ${sellerId}: ${response.status}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return data.data || data.apiKeys || [];
  } catch (e) {
    return [];
  }
}

/**
 * Revokes an API key
 * @param {string} sellerId - The seller ID
 * @param {string} keyId - The API key ID to revoke
 * @returns {boolean} True if successful
 */
export function revokeApiKey(sellerId, keyId) {
  const url = `${BASE_URLS.sellers}${ENDPOINTS.sellers.revokeApiKey(sellerId, keyId)}`;
  const response = http.del(url, null, HTTP_PARAMS);

  const success = check(response, {
    'revoke api key status 200': (r) => r.status === 200 || r.status === 204,
  });

  if (!success) {
    console.warn(`Failed to revoke API key for seller ${sellerId}: ${response.status}`);
  }

  return success;
}

/**
 * Returns a default fee schedule with standard rates
 * @returns {Object} Default fee schedule
 */
export function getDefaultFeeSchedule() {
  return {
    storageFeePerCubicFtPerDay: 0.05,
    pickFeePerUnit: 0.25,
    packFeePerOrder: 1.50,
    receivingFeePerUnit: 0.15,
    shippingMarkupPercent: 5.0,
    returnProcessingFee: 3.00,
    giftWrapFee: 2.50,
    hazmatHandlingFee: 5.00,
    oversizedItemFee: 10.00,
    coldChainFeePerUnit: 1.00,
    fragileHandlingFee: 1.50,
    volumeDiscounts: [],
  };
}

/**
 * Returns a premium fee schedule with discounted rates
 * @returns {Object} Premium fee schedule
 */
export function getPremiumFeeSchedule() {
  return {
    storageFeePerCubicFtPerDay: 0.03,
    pickFeePerUnit: 0.20,
    packFeePerOrder: 1.25,
    receivingFeePerUnit: 0.12,
    shippingMarkupPercent: 3.0,
    returnProcessingFee: 2.50,
    giftWrapFee: 2.00,
    hazmatHandlingFee: 4.00,
    oversizedItemFee: 8.00,
    coldChainFeePerUnit: 0.80,
    fragileHandlingFee: 1.25,
    volumeDiscounts: [
      { minUnits: 1000, discountPercent: 5 },
      { minUnits: 5000, discountPercent: 10 },
      { minUnits: 10000, discountPercent: 15 },
    ],
  };
}

/**
 * Creates and fully sets up a seller for testing
 * @param {Object} config - Setup configuration
 * @returns {Object} Setup result with seller and status
 */
export function setupTestSeller(config = {}) {
  const result = {
    success: false,
    seller: null,
    facilityAssigned: false,
    channelConnected: false,
    apiKeyGenerated: false,
    apiKey: null,
  };

  // Create seller
  const seller = createSeller({
    tenantId: config.tenantId || SELLER_CONFIG.defaultTenantId,
    companyName: config.companyName || `Test Seller ${Date.now()}`,
    contactName: config.contactName || 'Test Contact',
    contactEmail: config.contactEmail || `test-${Date.now()}@example.com`,
    billingCycle: config.billingCycle || BILLING_CYCLES.MONTHLY,
  });

  if (!seller) {
    return result;
  }

  result.seller = seller;
  const sellerId = seller.sellerId;

  // Activate seller
  sleep(SELLER_CONFIG.simulationDelayMs / 1000);
  if (!activateSeller(sellerId)) {
    console.warn(`Seller created but activation failed: ${sellerId}`);
  }

  // Assign facility
  if (config.facilityId) {
    sleep(SELLER_CONFIG.simulationDelayMs / 1000);
    result.facilityAssigned = assignFacility(sellerId, {
      facilityId: config.facilityId,
      facilityName: config.facilityName || config.facilityId,
      allocatedSpace: config.allocatedSpace || 1000,
      isDefault: true,
    });
  }

  // Update fee schedule if provided
  if (config.feeSchedule) {
    sleep(SELLER_CONFIG.simulationDelayMs / 1000);
    updateFeeSchedule(sellerId, config.feeSchedule);
  }

  // Connect channel if provided
  if (config.channel) {
    sleep(SELLER_CONFIG.simulationDelayMs / 1000);
    const channel = connectChannel(sellerId, config.channel);
    result.channelConnected = !!channel;
  }

  // Generate API key if requested
  if (config.generateApiKey) {
    sleep(SELLER_CONFIG.simulationDelayMs / 1000);
    const apiKey = generateApiKey(sellerId, {
      name: config.apiKeyName || 'Test API Key',
      scopes: config.apiKeyScopes || ['orders:read', 'inventory:read'],
    });
    result.apiKeyGenerated = !!apiKey;
    result.apiKey = apiKey;
  }

  result.success = true;
  console.log(`Seller setup complete: ${sellerId}`);
  return result;
}

/**
 * Health check for seller service
 * @returns {boolean} True if service is healthy
 */
export function checkHealth() {
  const response = http.get(`${BASE_URLS.sellers}/health`);
  return check(response, {
    'seller service healthy': (r) => r.status === 200,
  });
}

/**
 * Readiness check for seller service
 * @returns {boolean} True if service is ready
 */
export function checkReady() {
  const response = http.get(`${BASE_URLS.sellers}/ready`);
  return check(response, {
    'seller service ready': (r) => r.status === 200,
  });
}
