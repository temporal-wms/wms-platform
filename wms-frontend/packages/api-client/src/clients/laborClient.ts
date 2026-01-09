import { createServiceClient } from '../httpClient';
import type { Worker, Shift, PaginatedResponse } from '@wms/types';

const client = createServiceClient('labor');

export interface WorkerFilters {
  role?: string;
  status?: string;
  zone?: string;
  shiftId?: string;
  page?: number;
  pageSize?: number;
}

export interface CreateWorkerRequest {
  employeeId: string;
  name: string;
  role: 'PICKER' | 'PACKER' | 'SHIPPER' | 'RECEIVER' | 'SUPERVISOR';
}

export interface CreateShiftRequest {
  name: string;
  startTime: string;
  endTime: string;
  workerIds?: string[];
  zone?: string;
}

export const laborClient = {
  // Get workers with filters
  getWorkers: async (filters?: WorkerFilters): Promise<PaginatedResponse<Worker>> => {
    const params = new URLSearchParams();
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined) params.append(key, String(value));
      });
    }
    return client.get<PaginatedResponse<Worker>>(`api/v1/workers?${params.toString()}`);
  },

  // Get single worker
  getWorker: async (workerId: string): Promise<Worker> => {
    return client.get<Worker>(`api/v1/workers/${workerId}`);
  },

  // Create worker
  createWorker: async (request: CreateWorkerRequest): Promise<Worker> => {
    return client.post<Worker>('api/v1/workers', request);
  },

  // Update worker
  updateWorker: async (workerId: string, updates: Partial<Worker>): Promise<Worker> => {
    return client.patch<Worker>(`api/v1/workers/${workerId}`, updates);
  },

  // Clock in worker
  clockIn: async (workerId: string, zone?: string): Promise<Worker> => {
    return client.post<Worker>(`api/v1/workers/${workerId}/clock-in`, { zone });
  },

  // Clock out worker
  clockOut: async (workerId: string): Promise<Worker> => {
    return client.post<Worker>(`api/v1/workers/${workerId}/clock-out`);
  },

  // Start break
  startBreak: async (workerId: string): Promise<Worker> => {
    return client.post<Worker>(`api/v1/workers/${workerId}/break/start`);
  },

  // End break
  endBreak: async (workerId: string): Promise<Worker> => {
    return client.post<Worker>(`api/v1/workers/${workerId}/break/end`);
  },

  // Get shifts
  getShifts: async (): Promise<Shift[]> => {
    return client.get<Shift[]>('api/v1/shifts');
  },

  // Create shift
  createShift: async (request: CreateShiftRequest): Promise<Shift> => {
    return client.post<Shift>('api/v1/shifts', request);
  },

  // Assign workers to shift
  assignToShift: async (shiftId: string, workerIds: string[]): Promise<Shift> => {
    return client.post<Shift>(`api/v1/shifts/${shiftId}/assign`, { workerIds });
  },

  // Get labor statistics
  getLaborStats: async (): Promise<{
    activeWorkers: number;
    totalWorkers: number;
    utilizationRate: number;
    byRole: Record<string, { active: number; total: number }>;
    byZone: Record<string, number>;
  }> => {
    return client.get('api/v1/workers/stats');
  },

  // Get worker productivity
  getWorkerProductivity: async (workerId: string, period: 'day' | 'week' | 'month' = 'day'): Promise<{
    tasksCompleted: number;
    itemsProcessed: number;
    averageTaskTime: number;
    efficiency: number;
  }> => {
    return client.get(`api/v1/workers/${workerId}/productivity?period=${period}`);
  },
};
