import { createServiceClient } from '../httpClient';
import type { PickTask, PickRoute, PaginatedResponse } from '@wms/types';

const client = createServiceClient('picking');

export interface PickTaskFilters {
  status?: string;
  workerId?: string;
  waveId?: string;
  zone?: string;
  page?: number;
  pageSize?: number;
}

export interface ConfirmPickRequest {
  taskId: string;
  itemId: string;
  pickedQuantity: number;
  locationId: string;
}

export const pickingClient = {
  // Get pick tasks with filters
  getPickTasks: async (filters?: PickTaskFilters): Promise<PaginatedResponse<PickTask>> => {
    const params = new URLSearchParams();
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined) params.append(key, String(value));
      });
    }
    return client.get<PaginatedResponse<PickTask>>(`api/v1/pick-tasks?${params.toString()}`);
  },

  // Get single pick task
  getPickTask: async (taskId: string): Promise<PickTask> => {
    return client.get<PickTask>(`api/v1/pick-tasks/${taskId}`);
  },

  // Assign task to worker
  assignTask: async (taskId: string, workerId: string): Promise<PickTask> => {
    return client.post<PickTask>(`api/v1/pick-tasks/${taskId}/assign`, { workerId });
  },

  // Start picking
  startPicking: async (taskId: string): Promise<PickTask> => {
    return client.post<PickTask>(`api/v1/pick-tasks/${taskId}/start`);
  },

  // Confirm item pick
  confirmPick: async (request: ConfirmPickRequest): Promise<PickTask> => {
    return client.post<PickTask>(`api/v1/pick-tasks/${request.taskId}/pick`, request);
  },

  // Complete pick task
  completeTask: async (taskId: string): Promise<PickTask> => {
    return client.post<PickTask>(`api/v1/pick-tasks/${taskId}/complete`);
  },

  // Get route for task
  getRoute: async (taskId: string): Promise<PickRoute> => {
    return client.get<PickRoute>(`api/v1/pick-tasks/${taskId}/route`);
  },

  // Optimize route
  optimizeRoute: async (taskId: string): Promise<PickRoute> => {
    return client.post<PickRoute>(`api/v1/pick-tasks/${taskId}/route/optimize`);
  },

  // Get picking statistics
  getPickingStats: async (): Promise<{
    activeTasks: number;
    completedToday: number;
    itemsPicked: number;
    averageTime: number;
    byWorker: Array<{ workerId: string; workerName: string; completed: number }>;
  }> => {
    return client.get('api/v1/pick-tasks/stats');
  },
};
