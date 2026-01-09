import { createServiceClient } from '../httpClient';
import type { PackTask, Package, PaginatedResponse } from '@wms/types';

const client = createServiceClient('packing');

export interface PackTaskFilters {
  status?: string;
  workerId?: string;
  orderId?: string;
  page?: number;
  pageSize?: number;
}

export interface CreatePackageRequest {
  taskId: string;
  items: Array<{ itemId: string; quantity: number }>;
  weight: number;
  dimensions: {
    length: number;
    width: number;
    height: number;
  };
}

export const packingClient = {
  // Get pack tasks with filters
  getPackTasks: async (filters?: PackTaskFilters): Promise<PaginatedResponse<PackTask>> => {
    const params = new URLSearchParams();
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined) params.append(key, String(value));
      });
    }
    return client.get<PaginatedResponse<PackTask>>(`api/v1/pack-tasks?${params.toString()}`);
  },

  // Get single pack task
  getPackTask: async (taskId: string): Promise<PackTask> => {
    return client.get<PackTask>(`api/v1/pack-tasks/${taskId}`);
  },

  // Start packing
  startPacking: async (taskId: string, workerId: string): Promise<PackTask> => {
    return client.post<PackTask>(`api/v1/pack-tasks/${taskId}/start`, { workerId });
  },

  // Create package
  createPackage: async (request: CreatePackageRequest): Promise<Package> => {
    return client.post<Package>(`api/v1/pack-tasks/${request.taskId}/packages`, request);
  },

  // Verify item
  verifyItem: async (taskId: string, itemId: string, scannedSku: string): Promise<{ verified: boolean; message?: string }> => {
    return client.post(`api/v1/pack-tasks/${taskId}/verify`, { itemId, scannedSku });
  },

  // Complete packing
  completePacking: async (taskId: string): Promise<PackTask> => {
    return client.post<PackTask>(`api/v1/pack-tasks/${taskId}/complete`);
  },

  // Get packing statistics
  getPackingStats: async (): Promise<{
    activeTasks: number;
    completedToday: number;
    packagesCreated: number;
    averageTime: number;
  }> => {
    return client.get('api/v1/pack-tasks/stats');
  },
};
