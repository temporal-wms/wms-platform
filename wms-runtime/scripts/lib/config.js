// K6 Configuration for WMS Platform Load Testing

export const BASE_URLS = {
  orders: __ENV.ORDER_SERVICE_URL || 'http://localhost:8001',
  inventory: __ENV.INVENTORY_SERVICE_URL || 'http://localhost:8008',
  labor: __ENV.LABOR_SERVICE_URL || 'http://localhost:8009',
  waving: __ENV.WAVING_SERVICE_URL || 'http://localhost:8002',
  routing: __ENV.ROUTING_SERVICE_URL || 'http://localhost:8003',
  picking: __ENV.PICKING_SERVICE_URL || 'http://localhost:8004',
  consolidation: __ENV.CONSOLIDATION_SERVICE_URL || 'http://localhost:8005',
  packing: __ENV.PACKING_SERVICE_URL || 'http://localhost:8006',
  shipping: __ENV.SHIPPING_SERVICE_URL || 'http://localhost:8007',
  facility: __ENV.FACILITY_SERVICE_URL || 'http://localhost:8010',
  orchestrator: __ENV.ORCHESTRATOR_URL || 'http://localhost:30010',
  unit: __ENV.UNIT_SERVICE_URL || 'http://localhost:8014',
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

// Order generation configuration
export const ORDER_CONFIG = {
  // Item count distribution (must sum to 1.0)
  singleItemRatio: parseFloat(__ENV.SINGLE_ITEM_RATIO || '0.4'),   // 40% single-item orders
  multiItemRatio: parseFloat(__ENV.MULTI_ITEM_RATIO || '0.6'),     // 60% multi-item orders
  maxItemsPerOrder: parseInt(__ENV.MAX_ITEMS_PER_ORDER || '5'),    // Max items in multi-item orders

  // Force specific order types for testing
  forceOrderType: __ENV.FORCE_ORDER_TYPE || null,        // 'single', 'multi', or null
  forceRequirement: __ENV.FORCE_REQUIREMENT || null,     // 'hazmat', 'fragile', 'oversized', 'heavy', 'high_value', or null

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

export const HTTP_PARAMS = {
  headers: {
    'Content-Type': 'application/json',
    'Accept': 'application/json',
  },
  timeout: '30s',
};
