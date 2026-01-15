// K6 Configuration for WMS Platform Load Testing

// ============================================================================
// Kong API Gateway Configuration
// ============================================================================
// All WMS services are accessible through Kong Gateway at http://localhost:8888
// with the pattern: http://localhost:8888/{service-name}/api/v1/...
//
// To use Kong Gateway (default):
//   k6 run scripts/scenarios/your-simulator.js
//
// To use direct port-forward (legacy mode):
//   k6 run -e USE_KONG=false scripts/scenarios/your-simulator.js
//
// To use a custom Kong Gateway URL:
//   k6 run -e KONG_GATEWAY_URL=http://custom-gateway:8888 scripts/scenarios/your-simulator.js
// ============================================================================

const KONG_GATEWAY = __ENV.KONG_GATEWAY_URL || 'http://localhost:8888';
const USE_KONG = __ENV.USE_KONG !== 'false'; // Default to true

/**
 * Build service URL - routes through Kong Gateway or direct port-forward
 * @param {string} serviceName - Service name for Kong routing (e.g., 'order-service')
 * @param {number} directPort - Direct port for legacy port-forward mode
 * @returns {string} Service base URL
 */
function buildServiceUrl(serviceName, directPort) {
  if (!USE_KONG && directPort) {
    return `http://localhost:${directPort}`;
  }
  return `${KONG_GATEWAY}/${serviceName}`;
}

export const BASE_URLS = {
  orders: __ENV.ORDER_SERVICE_URL || buildServiceUrl('order-service', 8001),
  inventory: __ENV.INVENTORY_SERVICE_URL || buildServiceUrl('inventory-service', 8008),
  labor: __ENV.LABOR_SERVICE_URL || buildServiceUrl('labor-service', 8009),
  waving: __ENV.WAVING_SERVICE_URL || buildServiceUrl('waving-service', 8002),
  routing: __ENV.ROUTING_SERVICE_URL || buildServiceUrl('routing-service', 8003),
  picking: __ENV.PICKING_SERVICE_URL || buildServiceUrl('picking-service', 8004),
  consolidation: __ENV.CONSOLIDATION_SERVICE_URL || buildServiceUrl('consolidation-service', 8005),
  packing: __ENV.PACKING_SERVICE_URL || buildServiceUrl('packing-service', 8006),
  shipping: __ENV.SHIPPING_SERVICE_URL || buildServiceUrl('shipping-service', 8007),
  facility: __ENV.FACILITY_SERVICE_URL || buildServiceUrl('facility-service', 8010),
  orchestrator: __ENV.ORCHESTRATOR_URL || buildServiceUrl('orchestrator', 8080),
  unit: __ENV.UNIT_SERVICE_URL || buildServiceUrl('unit-service', 8014),
  walling: __ENV.WALLING_SERVICE_URL || buildServiceUrl('walling-service', 8017),
  wes: __ENV.WES_SERVICE_URL || buildServiceUrl('wes-service', 8016),
  receiving: __ENV.RECEIVING_SERVICE_URL || buildServiceUrl('receiving-service', 8013),
  stow: __ENV.STOW_SERVICE_URL || buildServiceUrl('stow-service', 8011),
  sortation: __ENV.SORTATION_SERVICE_URL || buildServiceUrl('sortation-service', 8012),
  temporalValidator: __ENV.TEMPORAL_VALIDATOR_URL || 'http://localhost:8020', // Not exposed via Kong
  sellers: __ENV.SELLER_SERVICE_URL || buildServiceUrl('seller-service', 8020),
  billing: __ENV.BILLING_SERVICE_URL || buildServiceUrl('billing-service', 8018),
};

// Picker simulator configuration
export const PICKER_CONFIG = {
  simulationDelayMs: parseInt(__ENV.PICKER_DELAY_MS || '500'),
  maxTasksPerIteration: parseInt(__ENV.MAX_TASKS_PER_ITERATION || '10'),
};

// Waving simulator configuration
export const WAVING_CONFIG = {
  simulationDelayMs: parseInt(__ENV.WAVING_DELAY_MS || '300'),
  maxWavesPerIteration: parseInt(__ENV.MAX_WAVES_PER_ITERATION || '5'),
};

// Consolidation simulator configuration
export const CONSOLIDATION_CONFIG = {
  simulationDelayMs: parseInt(__ENV.CONSOLIDATION_DELAY_MS || '400'),
  maxTasksPerIteration: parseInt(__ENV.MAX_CONSOLIDATION_TASKS || '10'),
  defaultStation: __ENV.CONSOLIDATION_STATION || 'CONSOL-STATION-1',
};

// Packing simulator configuration
export const PACKING_CONFIG = {
  simulationDelayMs: parseInt(__ENV.PACKING_DELAY_MS || '600'),
  maxTasksPerIteration: parseInt(__ENV.MAX_PACKING_TASKS || '10'),
  defaultStation: __ENV.PACKING_STATION || 'PACK-STATION-1',
};

// Walling simulator configuration
export const WALLING_CONFIG = {
  simulationDelayMs: parseInt(__ENV.WALLING_DELAY_MS || '500'),
  maxTasksPerIteration: parseInt(__ENV.MAX_WALLING_TASKS || '10'),
  defaultStation: __ENV.WALLING_STATION || 'WALL-STATION-1',
  defaultPutWallId: __ENV.DEFAULT_PUT_WALL || 'PUTWALL-1',
};

// Shipping simulator configuration
export const SHIPPING_CONFIG = {
  simulationDelayMs: parseInt(__ENV.SHIPPING_DELAY_MS || '500'),
  maxShipmentsPerIteration: parseInt(__ENV.MAX_SHIPMENTS || '10'),
  defaultCarrier: __ENV.DEFAULT_CARRIER || 'UPS',
};

// Facility simulator configuration
export const FACILITY_CONFIG = {
  simulationDelayMs: parseInt(__ENV.FACILITY_DELAY_MS || '300'),
  maxStationsPerIteration: parseInt(__ENV.MAX_STATIONS || '20'),
};

// Gift wrap simulator configuration
export const GIFTWRAP_CONFIG = {
  simulationDelayMs: parseInt(__ENV.GIFTWRAP_DELAY_MS || '2000'),
  maxTasksPerIteration: parseInt(__ENV.MAX_GIFTWRAP_TASKS || '5'),
  defaultWrapType: __ENV.GIFTWRAP_TYPE || 'standard',
  giftWrapOrderRatio: parseFloat(__ENV.GIFTWRAP_ORDER_RATIO || '0.2'),
};

// Seller simulator configuration
export const SELLER_CONFIG = {
  simulationDelayMs: parseInt(__ENV.SELLER_DELAY_MS || '300'),
  defaultBillingCycle: __ENV.DEFAULT_BILLING_CYCLE || 'monthly',
  defaultTenantId: __ENV.DEFAULT_TENANT_ID || 'DEFAULT_TENANT',
};

// Billing simulator configuration
export const BILLING_CONFIG = {
  simulationDelayMs: parseInt(__ENV.BILLING_DELAY_MS || '200'),
  enableBillingTracking: __ENV.ENABLE_BILLING_TRACKING !== 'false',
  recordActivitiesInBatch: __ENV.BATCH_BILLING_ACTIVITIES === 'true',
  defaultFacilityId: __ENV.DEFAULT_FACILITY_ID || 'FAC-001',
};

// Full flow orchestrator configuration
export const FLOW_CONFIG = {
  stageDelayMs: parseInt(__ENV.STAGE_DELAY_MS || '5000'),           // 5s between stages
  maxOrdersPerRun: parseInt(__ENV.MAX_ORDERS_PER_RUN || '10'),
  waitForTasksTimeoutMs: parseInt(__ENV.WAIT_TIMEOUT_MS || '300000'), // 5 minutes
  pollIntervalMs: parseInt(__ENV.POLL_INTERVAL_MS || '5000'),         // 5 seconds
  statusCheckTimeoutMs: parseInt(__ENV.STATUS_CHECK_TIMEOUT_MS || '120000'), // 2 min for status
  statusCheckIntervalMs: parseInt(__ENV.STATUS_CHECK_INTERVAL_MS || '3000'), // 3s polling
};

// Unit-level tracking configuration
export const UNIT_CONFIG = {
  enableUnitTracking: __ENV.ENABLE_UNIT_TRACKING !== 'false',  // Default enabled
  createUnitsOnReceive: __ENV.CREATE_UNITS_ON_RECEIVE === 'true',  // Default disabled
};

// Tenant context configuration (required headers for multi-tenant APIs)
export const TENANT_CONFIG = {
  tenantId: __ENV.TENANT_ID || __ENV.DEFAULT_TENANT_ID || 'test-tenant',
  facilityId: __ENV.FACILITY_ID || __ENV.DEFAULT_FACILITY_ID || 'test-facility',
  warehouseId: __ENV.WAREHOUSE_ID || __ENV.DEFAULT_WAREHOUSE_ID || 'test-warehouse',
  sellerId: __ENV.SELLER_ID || '',
  channelId: __ENV.CHANNEL_ID || 'web',
};

// Order generation configuration
export const ORDER_CONFIG = {
  // Item count distribution (must sum to 1.0)
  singleItemRatio: parseFloat(__ENV.SINGLE_ITEM_RATIO || '0.4'),   // 40% single-item orders
  multiItemRatio: parseFloat(__ENV.MULTI_ITEM_RATIO || '0.6'),     // 60% multi-item orders
  maxItemsPerOrder: parseInt(__ENV.MAX_ITEMS_PER_ORDER || '5'),    // Max items in multi-item orders

  // Force specific order types for testing
  forceOrderType: __ENV.FORCE_ORDER_TYPE || null,        // 'single', 'multi', or null
  forceRequirement: __ENV.FORCE_REQUIREMENT || null,     // 'hazmat', 'fragile', 'oversized', 'heavy', 'high_value', or null

  // Multi-tenancy configuration (deprecated - use TENANT_CONFIG)
  defaultTenantId: TENANT_CONFIG.tenantId,
  defaultFacilityId: TENANT_CONFIG.facilityId,
  defaultWarehouseId: TENANT_CONFIG.warehouseId,

  // Available requirements (for reference)
  validRequirements: ['hazmat', 'fragile', 'oversized', 'heavy', 'high_value'],
};

// Multi-route configuration
export const MULTI_ROUTE_CONFIG = {
  enableMultiRoute: __ENV.ENABLE_MULTI_ROUTE !== 'false',  // Default enabled
  maxItemsPerRoute: parseInt(__ENV.MAX_ITEMS_PER_ROUTE || '30'),
  largeOrderItemCount: parseInt(__ENV.LARGE_ORDER_ITEMS || '35'),  // Items to trigger multi-route
  parallelPickingEnabled: __ENV.PARALLEL_PICKING !== 'false',  // Default enabled
};

// Signal retry configuration
export const SIGNAL_CONFIG = {
  maxRetries: parseInt(__ENV.SIGNAL_MAX_RETRIES || '3'),
  retryDelayMs: parseInt(__ENV.SIGNAL_RETRY_DELAY_MS || '1000'),
  timeoutMs: parseInt(__ENV.SIGNAL_TIMEOUT_MS || '5000'),
};

// Workflow ID patterns used by orchestrator
// Reference: orchestrator/cmd/worker/main.go signal handlers
export const WORKFLOW_PATTERNS = {
  planning: (orderId) => `planning-${orderId}`,        // Wave assignment
  picking: (orderId) => `picking-${orderId}`,          // Standalone picking
  consolidation: (orderId) => `consolidation-${orderId}`, // Consolidation
  packing: (orderId) => `packing-${orderId}`,          // Standalone packing
  wes: (orderId) => `wes-${orderId}`,                  // WES orchestration (walling, multi-stage)
  shipping: (orderId) => `shipping-${orderId}`,        // Shipping
  giftWrap: (orderId) => `giftwrap-${orderId}`,        // Gift wrapping
};

// Signal endpoint reference
// All endpoints are relative to ORCHESTRATOR_URL
export const SIGNAL_ENDPOINTS = {
  waveAssigned: '/api/v1/signals/wave-assigned',               // → planning-{orderId}
  pickCompleted: '/api/v1/signals/pick-completed',             // → picking-{orderId}
  toteArrived: '/api/v1/signals/tote-arrived',                 // → consolidation-{orderId}
  consolidationCompleted: '/api/v1/signals/consolidation-completed', // → order-fulfillment-{orderId}
  giftWrapCompleted: '/api/v1/signals/gift-wrap-completed',    // → giftwrap-{orderId}
  wallingCompleted: '/api/v1/signals/walling-completed',       // → wes-execution-{orderId}
  packingCompleted: '/api/v1/signals/packing-completed',       // → packing-{orderId} or wes-execution-{orderId}
  receivingCompleted: '/api/v1/signals/receiving-completed',   // → receiving-{shipmentId}
  stowCompleted: '/api/v1/signals/stow-completed',             // → stow-{shipmentId}
};

export const ENDPOINTS = {
  orders: {
    create: '/api/v1/orders',
    get: (id) => `/api/v1/orders/${id}`,
    list: '/api/v1/orders',
    validate: (id) => `/api/v1/orders/${id}/validate`,
    cancel: (id) => `/api/v1/orders/${id}/cancel`,
  },
  inventory: {
    create: '/api/v1/inventory',
    get: (sku) => `/api/v1/inventory/${sku}`,
    receive: (sku) => `/api/v1/inventory/${sku}/receive`,
    reserve: (sku) => `/api/v1/inventory/${sku}/reserve`,
    pick: (sku) => `/api/v1/inventory/${sku}/pick`,
    release: (sku) => `/api/v1/inventory/${sku}/release`,
    lowStock: '/api/v1/inventory/low-stock',
  },
  labor: {
    createWorker: '/api/v1/workers',
    getWorker: (id) => `/api/v1/workers/${id}`,
    listWorkers: '/api/v1/workers',
    addSkill: (id) => `/api/v1/workers/${id}/skills`,
    startShift: (id) => `/api/v1/workers/${id}/shift/start`,
    endShift: (id) => `/api/v1/workers/${id}/shift/end`,
    startBreak: (id) => `/api/v1/workers/${id}/break/start`,
    endBreak: (id) => `/api/v1/workers/${id}/break/end`,
    assignTask: (id) => `/api/v1/workers/${id}/task/assign`,
    completeTask: (id) => `/api/v1/workers/${id}/task/complete`,
    availableWorkers: '/api/v1/workers/available',
  },
  routing: {
    calculate: '/api/v1/routes',
    calculateMulti: '/api/v1/routes/calculate-multi',
    get: (id) => `/api/v1/routes/${id}`,
    byOrder: (orderId) => `/api/v1/routes/order/${orderId}`,
  },
  picking: {
    pending: '/api/v1/tasks/pending',
    get: (id) => `/api/v1/tasks/${id}`,
    confirmPick: (id) => `/api/v1/tasks/${id}/pick`,
    start: (id) => `/api/v1/tasks/${id}/start`,
    complete: (id) => `/api/v1/tasks/${id}/complete`,
    byRoute: (routeId) => `/api/v1/tasks/route/${routeId}`,
  },
  waving: {
    list: '/api/v1/waves',
    get: (id) => `/api/v1/waves/${id}`,
    create: '/api/v1/waves',
    createFromOrders: '/api/v1/waves/from-orders',
    addOrder: (id) => `/api/v1/waves/${id}/orders`,
    schedule: (id) => `/api/v1/waves/${id}/schedule`,
    release: (id) => `/api/v1/waves/${id}/release`,
    byStatus: (status) => `/api/v1/waves/status/${status}`,
    readyForRelease: '/api/v1/planning/ready-for-release',
  },
  consolidation: {
    pending: '/api/v1/consolidations/pending',
    get: (id) => `/api/v1/consolidations/${id}`,
    create: '/api/v1/consolidations',
    assign: (id) => `/api/v1/consolidations/${id}/assign`,
    consolidate: (id) => `/api/v1/consolidations/${id}/consolidate`,
    complete: (id) => `/api/v1/consolidations/${id}/complete`,
    byOrder: (orderId) => `/api/v1/consolidations/order/${orderId}`,
    byStation: (station) => `/api/v1/consolidations/station/${station}`,
  },
  packing: {
    pending: '/api/v1/tasks/pending',
    get: (id) => `/api/v1/tasks/${id}`,
    create: '/api/v1/tasks',
    assign: (id) => `/api/v1/tasks/${id}/assign`,
    start: (id) => `/api/v1/tasks/${id}/start`,
    verify: (id) => `/api/v1/tasks/${id}/verify`,
    package: (id) => `/api/v1/tasks/${id}/package`,
    seal: (id) => `/api/v1/tasks/${id}/seal`,
    label: (id) => `/api/v1/tasks/${id}/label`,
    complete: (id) => `/api/v1/tasks/${id}/complete`,
    byOrder: (orderId) => `/api/v1/tasks/order/${orderId}`,
  },
  shipping: {
    pending: '/api/v1/shipments/status/pending',
    labeled: '/api/v1/shipments/status/labeled',
    get: (id) => `/api/v1/shipments/${id}`,
    create: '/api/v1/shipments',
    label: (id) => `/api/v1/shipments/${id}/label`,
    manifest: (id) => `/api/v1/shipments/${id}/manifest`,
    ship: (id) => `/api/v1/shipments/${id}/ship`,
    byOrder: (orderId) => `/api/v1/shipments/order/${orderId}`,
    byStatus: (status) => `/api/v1/shipments/status/${status}`,
    pendingManifest: (carrier) => `/api/v1/shipments/carrier/${carrier}/pending`,
  },
  orchestrator: {
    signalPickCompleted: '/api/v1/signals/pick-completed',
    signalWaveAssigned: '/api/v1/signals/wave-assigned',
    signalConsolidationComplete: '/api/v1/signals/consolidation-completed',
    signalPackingComplete: '/api/v1/signals/packing-completed',
    signalShipConfirmed: '/api/v1/signals/ship-confirmed',
    signalGiftWrapCompleted: '/api/v1/signals/gift-wrap-completed',
    signalWallingCompleted: '/api/v1/signals/walling-completed',
    signalReceivingCompleted: '/api/v1/signals/receiving-completed',
    signalStowCompleted: '/api/v1/signals/stow-completed',
  },
  facility: {
    stations: '/api/v1/stations',
    get: (id) => `/api/v1/stations/${id}`,
    update: (id) => `/api/v1/stations/${id}`,
    delete: (id) => `/api/v1/stations/${id}`,
    capabilities: (id) => `/api/v1/stations/${id}/capabilities`,
    addCapability: (id, cap) => `/api/v1/stations/${id}/capabilities/${cap}`,
    removeCapability: (id, cap) => `/api/v1/stations/${id}/capabilities/${cap}`,
    status: (id) => `/api/v1/stations/${id}/status`,
    findCapable: '/api/v1/stations/find-capable',
    byZone: (zone) => `/api/v1/stations/zone/${zone}`,
    byType: (type) => `/api/v1/stations/type/${type}`,
    byStatus: (status) => `/api/v1/stations/status/${status}`,
  },
  unit: {
    create: '/api/v1/units',
    reserve: '/api/v1/units/reserve',
    byOrder: (orderId) => `/api/v1/units/order/${orderId}`,
    get: (unitId) => `/api/v1/units/${unitId}`,
    audit: (unitId) => `/api/v1/units/${unitId}/audit`,
    pick: (unitId) => `/api/v1/units/${unitId}/pick`,
    consolidate: (unitId) => `/api/v1/units/${unitId}/consolidate`,
    pack: (unitId) => `/api/v1/units/${unitId}/pack`,
    ship: (unitId) => `/api/v1/units/${unitId}/ship`,
    exception: (unitId) => `/api/v1/units/${unitId}/exception`,
  },
  walling: {
    pending: '/api/v1/tasks/pending',
    get: (id) => `/api/v1/tasks/${id}`,
    assign: (id) => `/api/v1/tasks/${id}/assign`,
    sort: (id) => `/api/v1/tasks/${id}/sort`,
    complete: (id) => `/api/v1/tasks/${id}/complete`,
    activeByWalliner: (wallinerId) => `/api/v1/tasks/walliner/${wallinerId}/active`,
  },
  wes: {
    resolveExecutionPlan: '/api/v1/execution-plans/resolve',
    routes: '/api/v1/routes',
    getRoute: (id) => `/api/v1/routes/${id}`,
    getRouteByOrder: (orderId) => `/api/v1/routes/order/${orderId}`,
    assignWorker: (routeId) => `/api/v1/routes/${routeId}/stages/current/assign`,
    startStage: (routeId) => `/api/v1/routes/${routeId}/stages/current/start`,
    completeStage: (routeId) => `/api/v1/routes/${routeId}/stages/current/complete`,
    failStage: (routeId) => `/api/v1/routes/${routeId}/stages/current/fail`,
    templates: '/api/v1/templates',
  },
  receiving: {
    shipments: '/api/v1/shipments',
    get: (id) => `/api/v1/shipments/${id}`,
    receive: (id) => `/api/v1/shipments/${id}/receive`,
    confirmItem: (shipmentId, itemId) => `/api/v1/shipments/${shipmentId}/items/${itemId}/confirm`,
    discrepancy: (shipmentId, itemId) => `/api/v1/shipments/${shipmentId}/items/${itemId}/discrepancy`,
    complete: (id) => `/api/v1/shipments/${id}/complete`,
    tasks: (id) => `/api/v1/shipments/${id}/tasks`,
    pendingTasks: '/api/v1/tasks?status=pending',
    byStatus: (status) => `/api/v1/shipments?status=${status}`,
  },
  stow: {
    tasks: '/api/v1/tasks',
    get: (id) => `/api/v1/tasks/${id}`,
    assign: (id) => `/api/v1/tasks/${id}/assign`,
    start: (id) => `/api/v1/tasks/${id}/start`,
    stow: (id) => `/api/v1/tasks/${id}/stow`,
    complete: (id) => `/api/v1/tasks/${id}/complete`,
    byShipment: (shipmentId) => `/api/v1/tasks/shipment/${shipmentId}`,
    suggestLocation: '/api/v1/locations/suggest',
    pending: '/api/v1/tasks?status=pending',
  },
  sortation: {
    batches: '/api/v1/batches',
    get: (id) => `/api/v1/batches/${id}`,
    byStatus: (status) => `/api/v1/batches/status/${status}`,
    ready: '/api/v1/batches/ready',
    addPackage: (id) => `/api/v1/batches/${id}/packages`,
    sort: (id) => `/api/v1/batches/${id}/sort`,
    markReady: (id) => `/api/v1/batches/${id}/ready`,
    dispatch: (id) => `/api/v1/batches/${id}/dispatch`,
  },
  tracking: {
    workflowStatus: (workflowId) => `/workflows/${workflowId}/status`,
    workflowDescribe: (workflowId) => `/workflows/${workflowId}/describe`,
    workflowHistory: (workflowId) => `/workflows/${workflowId}/history`,
    workflowSignals: (workflowId) => `/workflows/${workflowId}/signals`,
    workflowQuery: (workflowId) => `/workflows/${workflowId}/query`,
    assertSignal: (workflowId) => `/workflows/${workflowId}/assert-signal`,
  },
  sellers: {
    create: '/api/v1/sellers',
    list: '/api/v1/sellers',
    get: (sellerId) => `/api/v1/sellers/${sellerId}`,
    search: '/api/v1/sellers/search',
    activate: (sellerId) => `/api/v1/sellers/${sellerId}/activate`,
    suspend: (sellerId) => `/api/v1/sellers/${sellerId}/suspend`,
    close: (sellerId) => `/api/v1/sellers/${sellerId}/close`,
    assignFacility: (sellerId) => `/api/v1/sellers/${sellerId}/facilities`,
    removeFacility: (sellerId, facilityId) => `/api/v1/sellers/${sellerId}/facilities/${facilityId}`,
    updateFeeSchedule: (sellerId) => `/api/v1/sellers/${sellerId}/fee-schedule`,
    connectChannel: (sellerId) => `/api/v1/sellers/${sellerId}/integrations`,
    disconnectChannel: (sellerId, channelId) => `/api/v1/sellers/${sellerId}/integrations/${channelId}`,
    generateApiKey: (sellerId) => `/api/v1/sellers/${sellerId}/api-keys`,
    listApiKeys: (sellerId) => `/api/v1/sellers/${sellerId}/api-keys`,
    revokeApiKey: (sellerId, keyId) => `/api/v1/sellers/${sellerId}/api-keys/${keyId}`,
  },
  billing: {
    recordActivity: '/api/v1/activities',
    recordBatch: '/api/v1/activities/batch',
    getActivity: (activityId) => `/api/v1/activities/${activityId}`,
    listActivities: (sellerId) => `/api/v1/sellers/${sellerId}/activities`,
    activitySummary: (sellerId) => `/api/v1/sellers/${sellerId}/activities/summary`,
    createInvoice: '/api/v1/invoices',
    getInvoice: (invoiceId) => `/api/v1/invoices/${invoiceId}`,
    listInvoices: (sellerId) => `/api/v1/sellers/${sellerId}/invoices`,
    finalizeInvoice: (invoiceId) => `/api/v1/invoices/${invoiceId}/finalize`,
    payInvoice: (invoiceId) => `/api/v1/invoices/${invoiceId}/pay`,
    voidInvoice: (invoiceId) => `/api/v1/invoices/${invoiceId}/void`,
    calculateFees: '/api/v1/fees/calculate',
    recordStorage: '/api/v1/storage/calculate',
  },
};

export const THRESHOLDS = {
  default: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'],
    http_req_failed: ['rate<0.01'],
  },
  strict: {
    http_req_duration: ['p(95)<200', 'p(99)<500'],
    http_req_failed: ['rate<0.001'],
  },
  relaxed: {
    http_req_duration: ['p(95)<2000', 'p(99)<5000'],
    http_req_failed: ['rate<0.05'],
  },
};

// Tenant HTTP headers for multi-tenant API requests
export const TENANT_HEADERS = {
  'X-WMS-Tenant-ID': TENANT_CONFIG.tenantId,
  'X-WMS-Facility-ID': TENANT_CONFIG.facilityId,
  'X-WMS-Warehouse-ID': TENANT_CONFIG.warehouseId,
  'X-WMS-Seller-ID': TENANT_CONFIG.sellerId,
  'X-WMS-Channel-ID': TENANT_CONFIG.channelId,
};

export const HTTP_PARAMS = {
  headers: {
    'Content-Type': 'application/json',
    'Accept': 'application/json',
    // Include tenant headers for multi-tenant APIs
    ...TENANT_HEADERS,
  },
  timeout: '30s',
};

// Helper function to get headers with optional overrides
export function getHeaders(overrides = {}) {
  return {
    ...HTTP_PARAMS.headers,
    ...overrides,
  };
}

// Helper function to get tenant headers only
export function getTenantHeaders(overrides = {}) {
  return {
    ...TENANT_HEADERS,
    ...overrides,
  };
}
