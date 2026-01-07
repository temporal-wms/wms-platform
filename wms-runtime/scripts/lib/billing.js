// K6 Billing Service Helper Library
// Provides functions for recording billable activities and managing invoices

import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URLS, ENDPOINTS, HTTP_PARAMS, BILLING_CONFIG } from './config.js';

// Activity types matching billing-service domain
export const ACTIVITY_TYPES = {
  STORAGE: 'storage',
  PICK: 'pick',
  PACK: 'pack',
  RECEIVING: 'receiving',
  SHIPPING: 'shipping',
  RETURN_PROCESSING: 'return_processing',
  GIFT_WRAP: 'gift_wrap',
  HAZMAT: 'hazmat',
  OVERSIZED: 'oversized',
  COLD_CHAIN: 'cold_chain',
  FRAGILE: 'fragile',
  SPECIAL_HANDLING: 'special_handling',
};

// Invoice status constants
export const INVOICE_STATUS = {
  DRAFT: 'draft',
  FINALIZED: 'finalized',
  PAID: 'paid',
  OVERDUE: 'overdue',
  VOIDED: 'voided',
};

// Reference types for activities
export const REFERENCE_TYPES = {
  ORDER: 'order',
  SHIPMENT: 'shipment',
  INVENTORY: 'inventory',
  RECEIVING: 'receiving',
};

// Payment methods
export const PAYMENT_METHODS = {
  CREDIT_CARD: 'credit_card',
  BANK_TRANSFER: 'bank_transfer',
  ACH: 'ach',
  CHECK: 'check',
};

/**
 * Records a single billable activity
 * @param {Object} activityData - Activity data
 * @returns {Object|null} Created activity or null
 */
export function recordActivity(activityData) {
  const url = `${BASE_URLS.billing}${ENDPOINTS.billing.recordActivity}`;
  const payload = JSON.stringify({
    tenantId: activityData.tenantId,
    sellerId: activityData.sellerId,
    facilityId: activityData.facilityId || BILLING_CONFIG.defaultFacilityId,
    type: activityData.type,
    description: activityData.description || '',
    quantity: activityData.quantity,
    unitPrice: activityData.unitPrice,
    referenceType: activityData.referenceType,
    referenceId: activityData.referenceId,
    metadata: activityData.metadata || {},
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'record activity status 201': (r) => r.status === 201 || r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to record activity: ${response.status} - ${response.body}`);
    return null;
  }

  try {
    const data = JSON.parse(response.body);
    return data.data || data;
  } catch (e) {
    console.error(`Failed to parse activity response: ${e.message}`);
    return null;
  }
}

/**
 * Records multiple activities in batch
 * @param {Array} activities - Array of activity data
 * @returns {Object} Result with counts
 */
export function recordActivitiesBatch(activities) {
  const url = `${BASE_URLS.billing}${ENDPOINTS.billing.recordBatch}`;
  const payload = JSON.stringify({ activities });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'record batch activities status 201': (r) => r.status === 201 || r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to record batch activities: ${response.status} - ${response.body}`);
    return { success: false, recorded: 0, failed: activities.length };
  }

  try {
    const data = JSON.parse(response.body);
    return { success: true, ...data.data };
  } catch (e) {
    return { success: true, recorded: activities.length, failed: 0 };
  }
}

/**
 * Gets an activity by ID
 * @param {string} activityId - The activity ID
 * @returns {Object|null} Activity or null
 */
export function getActivity(activityId) {
  const url = `${BASE_URLS.billing}${ENDPOINTS.billing.getActivity(activityId)}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get activity status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to get activity ${activityId}: ${response.status}`);
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
 * Lists activities for a seller
 * @param {string} sellerId - The seller ID
 * @param {Object} options - Pagination options
 * @returns {Array} Array of activities
 */
export function listActivities(sellerId, options = {}) {
  const params = new URLSearchParams();
  if (options.page) params.append('page', options.page);
  if (options.pageSize) params.append('pageSize', options.pageSize);

  const queryString = params.toString();
  const url = `${BASE_URLS.billing}${ENDPOINTS.billing.listActivities(sellerId)}${queryString ? '?' + queryString : ''}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'list activities status 200': (r) => r.status === 200,
  });

  if (!success || !response.body) {
    console.warn(`Failed to list activities for ${sellerId}: ${response.status}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return data.data || data.activities || [];
  } catch (e) {
    return [];
  }
}

/**
 * Gets activity summary for a seller and period
 * @param {string} sellerId - The seller ID
 * @param {Date|string} periodStart - Period start date
 * @param {Date|string} periodEnd - Period end date
 * @returns {Object|null} Summary or null
 */
export function getActivitySummary(sellerId, periodStart, periodEnd) {
  const startDate = periodStart instanceof Date ? periodStart.toISOString() : periodStart;
  const endDate = periodEnd instanceof Date ? periodEnd.toISOString() : periodEnd;

  const params = new URLSearchParams({
    periodStart: startDate,
    periodEnd: endDate,
  });

  const url = `${BASE_URLS.billing}${ENDPOINTS.billing.activitySummary(sellerId)}?${params.toString()}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get activity summary status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to get activity summary for ${sellerId}: ${response.status}`);
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
 * Creates an invoice for a seller
 * @param {Object} invoiceData - Invoice creation data
 * @returns {Object|null} Created invoice or null
 */
export function createInvoice(invoiceData) {
  const url = `${BASE_URLS.billing}${ENDPOINTS.billing.createInvoice}`;

  const periodStart = invoiceData.periodStart instanceof Date
    ? invoiceData.periodStart.toISOString()
    : invoiceData.periodStart;
  const periodEnd = invoiceData.periodEnd instanceof Date
    ? invoiceData.periodEnd.toISOString()
    : invoiceData.periodEnd;

  const payload = JSON.stringify({
    tenantId: invoiceData.tenantId,
    sellerId: invoiceData.sellerId,
    periodStart: periodStart,
    periodEnd: periodEnd,
    sellerName: invoiceData.sellerName,
    sellerEmail: invoiceData.sellerEmail,
    sellerAddress: invoiceData.sellerAddress || '',
    taxRate: invoiceData.taxRate || 0,
    notes: invoiceData.notes || '',
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'create invoice status 201': (r) => r.status === 201 || r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to create invoice: ${response.status} - ${response.body}`);
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
 * Gets an invoice by ID
 * @param {string} invoiceId - The invoice ID
 * @returns {Object|null} Invoice or null
 */
export function getInvoice(invoiceId) {
  const url = `${BASE_URLS.billing}${ENDPOINTS.billing.getInvoice(invoiceId)}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get invoice status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to get invoice ${invoiceId}: ${response.status}`);
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
 * Lists invoices for a seller
 * @param {string} sellerId - The seller ID
 * @param {Object} options - Filter and pagination options
 * @returns {Array} Array of invoices
 */
export function listInvoices(sellerId, options = {}) {
  const params = new URLSearchParams();
  if (options.page) params.append('page', options.page);
  if (options.pageSize) params.append('pageSize', options.pageSize);
  if (options.status) params.append('status', options.status);

  const queryString = params.toString();
  const url = `${BASE_URLS.billing}${ENDPOINTS.billing.listInvoices(sellerId)}${queryString ? '?' + queryString : ''}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'list invoices status 200': (r) => r.status === 200,
  });

  if (!success || !response.body) {
    console.warn(`Failed to list invoices for ${sellerId}: ${response.status}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return data.data || data.invoices || [];
  } catch (e) {
    return [];
  }
}

/**
 * Finalizes an invoice
 * @param {string} invoiceId - The invoice ID
 * @returns {boolean} True if successful
 */
export function finalizeInvoice(invoiceId) {
  const url = `${BASE_URLS.billing}${ENDPOINTS.billing.finalizeInvoice(invoiceId)}`;
  const response = http.put(url, null, HTTP_PARAMS);

  const success = check(response, {
    'finalize invoice status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to finalize invoice ${invoiceId}: ${response.status}`);
  }

  return success;
}

/**
 * Marks an invoice as paid
 * @param {string} invoiceId - The invoice ID
 * @param {Object} paymentInfo - Payment information
 * @returns {boolean} True if successful
 */
export function payInvoice(invoiceId, paymentInfo = {}) {
  const url = `${BASE_URLS.billing}${ENDPOINTS.billing.payInvoice(invoiceId)}`;
  const payload = JSON.stringify({
    paymentMethod: paymentInfo.paymentMethod || PAYMENT_METHODS.CREDIT_CARD,
    paymentRef: paymentInfo.paymentRef || `PAY-${Date.now()}`,
  });

  const response = http.put(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'pay invoice status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to pay invoice ${invoiceId}: ${response.status}`);
  }

  return success;
}

/**
 * Voids an invoice
 * @param {string} invoiceId - The invoice ID
 * @param {string} reason - Void reason
 * @returns {boolean} True if successful
 */
export function voidInvoice(invoiceId, reason) {
  const url = `${BASE_URLS.billing}${ENDPOINTS.billing.voidInvoice(invoiceId)}`;
  const payload = JSON.stringify({ reason });

  const response = http.put(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'void invoice status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to void invoice ${invoiceId}: ${response.status}`);
  }

  return success;
}

/**
 * Calculates fees preview for activities
 * @param {Object} feeData - Fee calculation data
 * @returns {Object|null} Fee calculation result or null
 */
export function calculateFees(feeData) {
  const url = `${BASE_URLS.billing}${ENDPOINTS.billing.calculateFees}`;
  const payload = JSON.stringify(feeData);

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'calculate fees status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to calculate fees: ${response.status}`);
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
 * Records a storage calculation
 * @param {Object} storageData - Storage calculation data
 * @returns {Object|null} Storage calculation result or null
 */
export function recordStorage(storageData) {
  const url = `${BASE_URLS.billing}${ENDPOINTS.billing.recordStorage}`;

  const calculationDate = storageData.calculationDate instanceof Date
    ? storageData.calculationDate.toISOString().split('T')[0]
    : storageData.calculationDate || new Date().toISOString().split('T')[0];

  const payload = JSON.stringify({
    tenantId: storageData.tenantId,
    sellerId: storageData.sellerId,
    facilityId: storageData.facilityId || BILLING_CONFIG.defaultFacilityId,
    calculationDate: calculationDate,
    totalCubicFeet: storageData.totalCubicFeet,
    ratePerCubicFt: storageData.ratePerCubicFt || 0.05,
    storageBreakdown: storageData.storageBreakdown || null,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'record storage status 201': (r) => r.status === 201 || r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to record storage: ${response.status} - ${response.body}`);
    return null;
  }

  try {
    const data = JSON.parse(response.body);
    return data.data || data;
  } catch (e) {
    return null;
  }
}

// =============================================================================
// Convenience functions for recording activities during WMS operations
// =============================================================================

/**
 * Records a pick activity
 * @param {Object} context - Billing context with sellerId, tenantId, facilityId
 * @param {string} orderId - Order ID
 * @param {number} quantity - Units picked
 * @param {number} unitPrice - Price per unit (default: 0.25)
 * @returns {Object|null} Activity or null
 */
export function recordPickActivity(context, orderId, quantity, unitPrice = 0.25) {
  if (!BILLING_CONFIG.enableBillingTracking) return null;

  return recordActivity({
    ...context,
    type: ACTIVITY_TYPES.PICK,
    description: `Pick operation for order ${orderId}`,
    quantity: quantity,
    unitPrice: unitPrice,
    referenceType: REFERENCE_TYPES.ORDER,
    referenceId: orderId,
  });
}

/**
 * Records a pack activity
 * @param {Object} context - Billing context
 * @param {string} orderId - Order ID
 * @param {number} orderCount - Orders packed (default: 1)
 * @param {number} unitPrice - Price per order (default: 1.50)
 * @returns {Object|null} Activity or null
 */
export function recordPackActivity(context, orderId, orderCount = 1, unitPrice = 1.50) {
  if (!BILLING_CONFIG.enableBillingTracking) return null;

  return recordActivity({
    ...context,
    type: ACTIVITY_TYPES.PACK,
    description: `Pack operation for order ${orderId}`,
    quantity: orderCount,
    unitPrice: unitPrice,
    referenceType: REFERENCE_TYPES.ORDER,
    referenceId: orderId,
  });
}

/**
 * Records a shipping activity
 * @param {Object} context - Billing context
 * @param {string} shipmentId - Shipment ID
 * @param {number} shippingCost - Base shipping cost
 * @param {number} markupPercent - Markup percentage (default: 5)
 * @returns {Object|null} Activity or null
 */
export function recordShippingActivity(context, shipmentId, shippingCost, markupPercent = 5) {
  if (!BILLING_CONFIG.enableBillingTracking) return null;

  const markup = shippingCost * (markupPercent / 100);
  return recordActivity({
    ...context,
    type: ACTIVITY_TYPES.SHIPPING,
    description: `Shipping for shipment ${shipmentId}`,
    quantity: 1,
    unitPrice: shippingCost + markup,
    referenceType: REFERENCE_TYPES.SHIPMENT,
    referenceId: shipmentId,
    metadata: {
      baseCost: shippingCost,
      markupPercent: markupPercent,
      markupAmount: markup,
    },
  });
}

/**
 * Records a receiving activity
 * @param {Object} context - Billing context
 * @param {string} shipmentId - Inbound shipment ID
 * @param {number} quantity - Units received
 * @param {number} unitPrice - Price per unit (default: 0.15)
 * @returns {Object|null} Activity or null
 */
export function recordReceivingActivity(context, shipmentId, quantity, unitPrice = 0.15) {
  if (!BILLING_CONFIG.enableBillingTracking) return null;

  return recordActivity({
    ...context,
    type: ACTIVITY_TYPES.RECEIVING,
    description: `Receiving for shipment ${shipmentId}`,
    quantity: quantity,
    unitPrice: unitPrice,
    referenceType: REFERENCE_TYPES.RECEIVING,
    referenceId: shipmentId,
  });
}

/**
 * Records a gift wrap activity
 * @param {Object} context - Billing context
 * @param {string} orderId - Order ID
 * @param {number} itemCount - Items wrapped (default: 1)
 * @param {number} unitPrice - Price per item (default: 2.50)
 * @returns {Object|null} Activity or null
 */
export function recordGiftWrapActivity(context, orderId, itemCount = 1, unitPrice = 2.50) {
  if (!BILLING_CONFIG.enableBillingTracking) return null;

  return recordActivity({
    ...context,
    type: ACTIVITY_TYPES.GIFT_WRAP,
    description: `Gift wrap for order ${orderId}`,
    quantity: itemCount,
    unitPrice: unitPrice,
    referenceType: REFERENCE_TYPES.ORDER,
    referenceId: orderId,
  });
}

/**
 * Records a return processing activity
 * @param {Object} context - Billing context
 * @param {string} orderId - Original order ID
 * @param {number} returnCount - Returns processed (default: 1)
 * @param {number} unitPrice - Price per return (default: 3.00)
 * @returns {Object|null} Activity or null
 */
export function recordReturnActivity(context, orderId, returnCount = 1, unitPrice = 3.00) {
  if (!BILLING_CONFIG.enableBillingTracking) return null;

  return recordActivity({
    ...context,
    type: ACTIVITY_TYPES.RETURN_PROCESSING,
    description: `Return processing for order ${orderId}`,
    quantity: returnCount,
    unitPrice: unitPrice,
    referenceType: REFERENCE_TYPES.ORDER,
    referenceId: orderId,
  });
}

/**
 * Records special handling activities based on item requirements
 * @param {Object} context - Billing context
 * @param {string} orderId - Order ID
 * @param {Array} requirements - Array of requirements (hazmat, fragile, oversized, cold_chain)
 * @param {Object} feeSchedule - Fee schedule with prices (optional)
 * @returns {Object} Result with recorded count and activities
 */
export function recordSpecialHandlingActivities(context, orderId, requirements, feeSchedule = {}) {
  if (!BILLING_CONFIG.enableBillingTracking || !requirements || requirements.length === 0) {
    return { recorded: 0, activities: [] };
  }

  const activities = [];
  const activityMap = {
    hazmat: { type: ACTIVITY_TYPES.HAZMAT, price: feeSchedule.hazmatHandlingFee || 5.00, desc: 'Hazmat handling' },
    oversized: { type: ACTIVITY_TYPES.OVERSIZED, price: feeSchedule.oversizedItemFee || 10.00, desc: 'Oversized item handling' },
    cold_chain: { type: ACTIVITY_TYPES.COLD_CHAIN, price: feeSchedule.coldChainFeePerUnit || 1.00, desc: 'Cold chain handling' },
    fragile: { type: ACTIVITY_TYPES.FRAGILE, price: feeSchedule.fragileHandlingFee || 1.50, desc: 'Fragile item handling' },
    heavy: { type: ACTIVITY_TYPES.SPECIAL_HANDLING, price: feeSchedule.heavyItemFee || 3.00, desc: 'Heavy item handling' },
    high_value: { type: ACTIVITY_TYPES.SPECIAL_HANDLING, price: feeSchedule.highValueFee || 2.00, desc: 'High value item handling' },
  };

  for (const req of requirements) {
    const config = activityMap[req];
    if (config) {
      const activity = recordActivity({
        ...context,
        type: config.type,
        description: `${config.desc} for order ${orderId}`,
        quantity: 1,
        unitPrice: config.price,
        referenceType: REFERENCE_TYPES.ORDER,
        referenceId: orderId,
        metadata: { requirement: req },
      });
      if (activity) activities.push(activity);
    }
  }

  return { recorded: activities.length, activities };
}

/**
 * Creates a billing context from seller data
 * @param {Object} seller - Seller object with sellerId and tenantId
 * @param {string} facilityId - Optional facility ID override
 * @returns {Object} Billing context
 */
export function createBillingContext(seller, facilityId = null) {
  return {
    tenantId: seller.tenantId,
    sellerId: seller.sellerId,
    facilityId: facilityId || seller.defaultFacilityId || BILLING_CONFIG.defaultFacilityId,
  };
}

/**
 * Records all activities for a completed order
 * @param {Object} context - Billing context
 * @param {Object} orderData - Order data with orderId, items, shipmentId, requirements
 * @param {Object} feeSchedule - Optional fee schedule
 * @returns {Object} Summary of recorded activities
 */
export function recordOrderActivities(context, orderData, feeSchedule = {}) {
  if (!BILLING_CONFIG.enableBillingTracking) {
    return { success: false, activities: [] };
  }

  const activities = [];
  const { orderId, items, shipmentId, requirements, hasGiftWrap, shippingCost } = orderData;
  const itemCount = Array.isArray(items) ? items.length : (items || 1);

  // Pick activity
  const pickActivity = recordPickActivity(
    context,
    orderId,
    itemCount,
    feeSchedule.pickFeePerUnit || 0.25
  );
  if (pickActivity) activities.push(pickActivity);

  // Pack activity
  const packActivity = recordPackActivity(
    context,
    orderId,
    1,
    feeSchedule.packFeePerOrder || 1.50
  );
  if (packActivity) activities.push(packActivity);

  // Gift wrap if applicable
  if (hasGiftWrap) {
    const giftWrapActivity = recordGiftWrapActivity(
      context,
      orderId,
      1,
      feeSchedule.giftWrapFee || 2.50
    );
    if (giftWrapActivity) activities.push(giftWrapActivity);
  }

  // Shipping if shipment info provided
  if (shipmentId && shippingCost !== undefined) {
    const shippingActivity = recordShippingActivity(
      context,
      shipmentId,
      shippingCost,
      feeSchedule.shippingMarkupPercent || 5
    );
    if (shippingActivity) activities.push(shippingActivity);
  }

  // Special handling requirements
  if (requirements && requirements.length > 0) {
    const specialResult = recordSpecialHandlingActivities(context, orderId, requirements, feeSchedule);
    activities.push(...specialResult.activities);
  }

  return {
    success: true,
    orderId,
    totalActivities: activities.length,
    activities,
  };
}

/**
 * Health check for billing service
 * @returns {boolean} True if service is healthy
 */
export function checkHealth() {
  const response = http.get(`${BASE_URLS.billing}/health`);
  return check(response, {
    'billing service healthy': (r) => r.status === 200,
  });
}

/**
 * Readiness check for billing service
 * @returns {boolean} True if service is ready
 */
export function checkReady() {
  const response = http.get(`${BASE_URLS.billing}/ready`);
  return check(response, {
    'billing service ready': (r) => r.status === 200,
  });
}
