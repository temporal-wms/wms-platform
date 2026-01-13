import { httpClient, createServiceClient } from '../httpClient';
import type {
  Route,
  CreateRouteRequest,
  RouteAnalysis,
  RouteItem,
  RouteStop,
  PaginatedResponse,
  SkipReason,
} from '@wms/types';

const client = createServiceClient('routing');

export interface RouteFilters {
  status?: string;
  strategy?: string;
  pickerId?: string;
  waveId?: string;
  page?: number;
  pageSize?: number;
}

export const routingClient = {
  getRoutes: async (filters?: RouteFilters): Promise<PaginatedResponse<Route>> => {
    const params = new URLSearchParams();
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined) params.append(key, String(value));
      });
    }
    return client.get<PaginatedResponse<Route>>(`api/v1/routes?${params.toString()}`);
  },

  getPendingRoutes: async (limit = 20): Promise<Route[]> => {
    return client.get<Route[]>(`api/v1/routes/pending?limit=${limit}`);
  },

  getRoute: async (routeId: string): Promise<Route> => {
    return client.get<Route>(`api/v1/routes/${routeId}`);
  },

  createRoute: async (request: CreateRouteRequest): Promise<Route> => {
    return client.post<Route>('api/v1/routes', request);
  },

  startRoute: async (routeId: string): Promise<Route> => {
    return client.post<Route>(`api/v1/routes/${routeId}/start`);
  },

  pauseRoute: async (routeId: string, reason?: string): Promise<Route> => {
    return client.post<Route>(`api/v1/routes/${routeId}/pause`, reason ? { reason } : {});
  },

  resumeRoute: async (routeId: string): Promise<Route> => {
    return client.post<Route>(`api/v1/routes/${routeId}/resume`);
  },

  completeRoute: async (routeId: string): Promise<Route> => {
    return client.post<Route>(`api/v1/routes/${routeId}/complete`);
  },

  cancelRoute: async (routeId: string, reason?: string): Promise<Route> => {
    return client.post<Route>(`api/v1/routes/${routeId}/cancel`, reason ? { reason } : {});
  },

  completeStop: async (
    routeId: string,
    stopNumber: number,
    actualQuantity: number,
    notes?: string
  ): Promise<Route> => {
    return client.post<Route>(`api/v1/routes/${routeId}/stops/${stopNumber}/complete`, {
      actualQuantity,
      notes,
    });
  },

  skipStop: async (
    routeId: string,
    stopNumber: number,
    reason: SkipReason,
    notes?: string
  ): Promise<Route> => {
    return client.post<Route>(`api/v1/routes/${routeId}/stops/${stopNumber}/skip`, {
      reason,
      notes,
    });
  },

  getRouteAnalysis: async (routeId: string): Promise<RouteAnalysis> => {
    return client.get<RouteAnalysis>(`api/v1/analysis/route/${routeId}`);
  },

  suggestStrategy: async (items: RouteItem[]): Promise<{
    recommendedStrategy: string;
    confidence: number;
    estimatedDistance: number;
    estimatedTimeMinutes: number;
    alternatives: Array<{
      strategy: string;
      estimatedDistance: number;
      estimatedTimeMinutes: number;
    }>;
    reasoning: string;
  }> => {
    return client.post('api/v1/analysis/suggest-strategy', { items });
  },

  getRoutesByOrder: async (orderId: string): Promise<Route[]> => {
    return client.get<Route[]>(`api/v1/routes/order/${orderId}`);
  },

  getRoutesByWave: async (waveId: string): Promise<Route[]> => {
    return client.get<Route[]>(`api/v1/routes/wave/${waveId}`);
  },

  getRoutesByPicker: async (pickerId: string, status?: string): Promise<Route[]> => {
    const params = status ? `?status=${status}` : '';
    return client.get<Route[]>(`api/v1/routes/picker/${pickerId}${params}`);
  },

  getActiveRouteForPicker: async (pickerId: string): Promise<Route> => {
    return client.get<Route>(`api/v1/routes/picker/${pickerId}/active`);
  },

  getRoutesByStatus: async (status: string, limit = 50): Promise<Route[]> => {
    return client.get<Route[]>(`api/v1/routes/status/${status}?limit=${limit}`);
  },
};

export type { CreateRouteRequest, RouteItem, SkipReason };
