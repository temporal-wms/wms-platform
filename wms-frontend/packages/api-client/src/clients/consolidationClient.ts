import { httpClient, createServiceClient } from '../httpClient';
import type {
  Consolidation,
  CreateConsolidationRequest,
  ConsolidateItemRequest,
  PaginatedResponse,
} from '@wms/types';

const client = createServiceClient('consolidation');

export interface ConsolidationFilters {
  status?: string;
  stationId?: string;
  waveId?: string;
  page?: number;
  pageSize?: number;
}

export const consolidationClient = {
  getConsolidations: async (filters?: ConsolidationFilters): Promise<PaginatedResponse<Consolidation>> => {
    const params = new URLSearchParams();
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined) params.append(key, String(value));
      });
    }
    return client.get<PaginatedResponse<Consolidation>>(`api/v1/consolidations?${params.toString()}`);
  },

  getConsolidation: async (consolidationId: string): Promise<Consolidation> => {
    return client.get<Consolidation>(`api/v1/consolidations/${consolidationId}`);
  },

  createConsolidation: async (request: CreateConsolidationRequest): Promise<Consolidation> => {
    return client.post<Consolidation>('api/v1/consolidations', request);
  },

  consolidateItem: async (
    consolidationId: string,
    item: ConsolidateItemRequest
  ): Promise<Consolidation> => {
    return client.post<Consolidation>(`api/v1/consolidations/${consolidationId}/consolidate`, item);
  },

  completeConsolidation: async (
    consolidationId: string,
    notes?: string,
    forceComplete?: boolean
  ): Promise<Consolidation> => {
    return client.post<Consolidation>(`api/v1/consolidations/${consolidationId}/complete`, {
      notes,
      forceComplete,
    });
  },

  cancelConsolidation: async (consolidationId: string, reason: string): Promise<Consolidation> => {
    return client.post<Consolidation>(`api/v1/consolidations/${consolidationId}/cancel`, { reason });
  },

  getByOrderId: async (orderId: string): Promise<Consolidation> => {
    return client.get<Consolidation>(`api/v1/consolidations/order/${orderId}`);
  },

  getByStation: async (stationId: string, status?: string): Promise<Consolidation[]> => {
    const params = status ? `?status=${status}` : '';
    return client.get<Consolidation[]>(`api/v1/consolidations/station/${stationId}${params}`);
  },

  getByWave: async (waveId: string): Promise<Consolidation[]> => {
    return client.get<Consolidation[]>(`api/v1/consolidations/wave/${waveId}`);
  },

  getByStatus: async (status: string, limit = 50): Promise<Consolidation[]> => {
    return client.get<Consolidation[]>(`api/v1/consolidations/status/${status}?limit=${limit}`);
  },
};

export type { CreateConsolidationRequest, ConsolidateItemRequest };
