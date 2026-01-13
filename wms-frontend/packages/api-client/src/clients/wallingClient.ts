import { httpClient, createServiceClient } from '../httpClient';
import type {
  WallingTask,
  CreateWallingTaskRequest,
  PaginatedResponse,
} from '@wms/types';

const client = createServiceClient('walling');

export interface WallingTaskFilters {
  status?: string;
  putWallId?: string;
  wallinerId?: string;
  page?: number;
  pageSize?: number;
}

export const wallingClient = {
  getTasks: async (filters?: WallingTaskFilters): Promise<PaginatedResponse<WallingTask>> => {
    const params = new URLSearchParams();
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined) params.append(key, String(value));
      });
    }
    return client.get<PaginatedResponse<WallingTask>>(`api/v1/tasks?${params.toString()}`);
  },

  getPendingTasks: async (putWallId: string, limit = 20): Promise<WallingTask[]> => {
    return client.get<WallingTask[]>(`api/v1/tasks/pending?putWallId=${putWallId}&limit=${limit}`);
  },

  getTask: async (taskId: string): Promise<WallingTask> => {
    return client.get<WallingTask>(`api/v1/tasks/${taskId}`);
  },

  createTask: async (request: CreateWallingTaskRequest): Promise<WallingTask> => {
    return client.post<WallingTask>('api/v1/tasks', request);
  },

  assignWalliner: async (
    taskId: string,
    wallinerId: string,
    station?: string
  ): Promise<WallingTask> => {
    return client.post<WallingTask>(`api/v1/tasks/${taskId}/assign`, {
      wallinerId,
      station,
    });
  },

  sortItem: async (
    taskId: string,
    sku: string,
    quantity: number,
    fromToteId: string
  ): Promise<WallingTask> => {
    return client.post<WallingTask>(`api/v1/tasks/${taskId}/sort`, {
      sku,
      quantity,
      fromToteId,
    });
  },

  completeTask: async (taskId: string): Promise<WallingTask> => {
    return client.post<WallingTask>(`api/v1/tasks/${taskId}/complete`);
  },

  cancelTask: async (taskId: string, reason: string): Promise<WallingTask> => {
    return client.post<WallingTask>(`api/v1/tasks/${taskId}/cancel`, { reason });
  },

  getActiveTask: async (wallinerId: string): Promise<WallingTask> => {
    return client.get<WallingTask>(`api/v1/tasks/active/${wallinerId}`);
  },

  getTasksByStation: async (stationId: string, status?: string): Promise<WallingTask[]> => {
    const params = status ? `?status=${status}` : '';
    return client.get<WallingTask[]>(`api/v1/tasks/station/${stationId}${params}`);
  },
};

export type { CreateWallingTaskRequest };
