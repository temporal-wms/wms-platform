// Environment configuration
export const config = {
  // API Configuration
  api: {
    baseUrl: import.meta.env.VITE_API_BASE_URL || '',
    timeout: 30000,
  },

  // Service URLs (proxied through nginx in K8s, direct in local dev)
  services: {
    orders: import.meta.env.VITE_ORDER_SERVICE_URL || '/api/orders',
    waves: import.meta.env.VITE_WAVE_SERVICE_URL || '/api/waves',
    routing: import.meta.env.VITE_ROUTING_SERVICE_URL || '/api/routing',
    picking: import.meta.env.VITE_PICKING_SERVICE_URL || '/api/picking',
    consolidation: import.meta.env.VITE_CONSOLIDATION_SERVICE_URL || '/api/consolidation',
    packing: import.meta.env.VITE_PACKING_SERVICE_URL || '/api/packing',
    shipping: import.meta.env.VITE_SHIPPING_SERVICE_URL || '/api/shipping',
    inventory: import.meta.env.VITE_INVENTORY_SERVICE_URL || '/api/inventory',
    labor: import.meta.env.VITE_LABOR_SERVICE_URL || '/api/labor',
  },

  // WebSocket Configuration
  websocket: {
    enabled: import.meta.env.VITE_WS_ENABLED !== 'false',
    baseUrl: import.meta.env.VITE_WS_BASE_URL || 'ws://localhost:8080',
    reconnectAttempts: 5,
    reconnectInterval: 3000,
  },

  // Feature Flags
  features: {
    realTimeUpdates: import.meta.env.VITE_FEATURE_REALTIME !== 'false',
    analytics: import.meta.env.VITE_FEATURE_ANALYTICS === 'true',
    debugMode: import.meta.env.VITE_DEBUG === 'true',
  },

  // Query Configuration
  query: {
    staleTime: 30000, // 30 seconds
    cacheTime: 300000, // 5 minutes
    refetchInterval: 5000, // 5 seconds for real-time data
  },

  // Pagination Defaults
  pagination: {
    defaultPageSize: 20,
    pageSizeOptions: [10, 20, 50, 100],
  },
} as const;

export type Config = typeof config;
