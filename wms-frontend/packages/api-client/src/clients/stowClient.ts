import { httpClient, createServiceClient } from '../httpClient';
import type {
  PutawayTask,
  CreatePutawayRequest,
  PaginatedResponse,
} from '@wms/types';

const client = createServiceClient('stow');

export interface StowTaskFilters {
  status?: string;
  strategy?: string;
  workerId?: string;
  page?: number;
  pageSize?: number;
}

export const stowClient = {
  getTasks: async (filters?: StowTaskFilters): Promise<PaginatedResponse<PutawayTask>> => {
    const params = new URLSearchParams();
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined) params.append(key, String(value));
      });
    }
    return client.get<PaginatedResponse<PutawayTask>>(`api/v1/tasks?${params.toString()}`);
  },

  getPendingTasks: async (limit = 20): Promise<PutawayTask[]> => {
    return client.get<PutawayTask[]>(`api/v1/tasks/pending?limit=${limit}`);
  },

  getTask: async (taskId: string): Promise<PutawayTask> => {
    return client.get<PutawayTask>(`api/v1/tasks/${taskId}`);
  },

  createTask: async (request: CreatePutawayRequest): Promise<PutawayTask> => {
    return client.post<PutawayTask>('api/v1/tasks', request);
  },

  assignTask: async (taskId: string, workerId: string): Promise<PutawayTask> => {
    return client.post<PutawayTask>(`api/v1/tasks/${taskId}/assign`, { workerId });
  },

  startStow: async (taskId: string): Promise<PutawayTask> => {
    return client.post<PutawayTask>(`api/v1/tasks/${taskId}/start`);
  },

  stowItem: async (
    taskId: string,
    locationId: string,
    quantity: number
  ): Promise<PutawayTask> => {
    return client.post<PutawayTask>(`api/v1/tasks/${taskId}/stow`, { locationId, quantity });
  },

  completeTask: async (taskId: string): Promise<PutawayTask> => {
    return client.post<PutawayTask>(`api/v1/tasks/${taskId}/complete`);
  },

  failTask: async (taskId: string, reason: string): Promise<PutawayTask> => {
    return client.post<PutawayTask>(`api/v1/tasks/${taskId}/fail`, { reason });
  },

  getTasksByWorker: async (workerId: string, status?: string): Promise<PutawayTask[]> => {
    const params = status ? `?status=${status}` : '';
    return client.get<PutawayTask[]>(`api/v1/tasks/worker/${workerId}${params}`);
  },
};

export type { CreatePutawayRequest };
