import { createServiceClient } from '../httpClient';
import type { Wave, PaginatedResponse } from '@wms/types';

const client = createServiceClient('waves');

export interface CreateWaveRequest {
  orderIds: string[];
  priority?: 'LOW' | 'NORMAL' | 'HIGH' | 'RUSH';
  scheduledAt?: string;
}

export interface WaveFilters {
  status?: string;
  priority?: string;
  fromDate?: string;
  toDate?: string;
  page?: number;
  pageSize?: number;
}

export const waveClient = {
  // Get all waves with optional filters
  getWaves: async (filters?: WaveFilters): Promise<PaginatedResponse<Wave>> => {
    const params = new URLSearchParams();
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined) params.append(key, String(value));
      });
    }
    return client.get<PaginatedResponse<Wave>>(`api/v1/waves?${params.toString()}`);
  },

  // Get single wave by ID
  getWave: async (waveId: string): Promise<Wave> => {
    return client.get<Wave>(`api/v1/waves/${waveId}`);
  },

  // Create new wave
  createWave: async (request: CreateWaveRequest): Promise<Wave> => {
    return client.post<Wave>('api/v1/waves', request);
  },

  // Release wave for processing
  releaseWave: async (waveId: string): Promise<Wave> => {
    return client.post<Wave>(`api/v1/waves/${waveId}/release`);
  },

  // Cancel wave
  cancelWave: async (waveId: string): Promise<void> => {
    return client.post<void>(`api/v1/waves/${waveId}/cancel`);
  },

  // Add orders to wave
  addOrdersToWave: async (waveId: string, orderIds: string[]): Promise<Wave> => {
    return client.post<Wave>(`api/v1/waves/${waveId}/orders`, { orderIds });
  },

  // Remove orders from wave
  removeOrdersFromWave: async (waveId: string, orderIds: string[]): Promise<Wave> => {
    return client.delete<Wave>(`api/v1/waves/${waveId}/orders`, { data: { orderIds } });
  },

  // Get available orders for waving
  getAvailableOrders: async (page = 1, pageSize = 50): Promise<PaginatedResponse<{ id: string; orderNumber: string; priority: string; itemCount: number }>> => {
    return client.get(`api/v1/waves/available-orders?page=${page}&pageSize=${pageSize}`);
  },

  // Get wave statistics
  getWaveStats: async (): Promise<{
    total: number;
    active: number;
    completed: number;
    ordersInWaves: number;
  }> => {
    return client.get('api/v1/waves/stats');
  },
};
