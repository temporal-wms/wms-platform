import { httpClient, createServiceClient } from '../httpClient';
import type {
  SortationBatch,
  CreateBatchRequest,
  SortPackageRequest,
  DispatchBatchRequest,
  PaginatedResponse,
} from '@wms/types';

const client = createServiceClient('sortation');

export interface BatchFilters {
  status?: string;
  carrierId?: string;
  sortationCenter?: string;
  page?: number;
  pageSize?: number;
}

export const sortationClient = {
  getBatches: async (filters?: BatchFilters): Promise<PaginatedResponse<SortationBatch>> => {
    const params = new URLSearchParams();
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined) params.append(key, String(value));
      });
    }
    return client.get<PaginatedResponse<SortationBatch>>(`api/v1/batches?${params.toString()}`);
  },

  getBatch: async (batchId: string): Promise<SortationBatch> => {
    return client.get<SortationBatch>(`api/v1/batches/${batchId}`);
  },

  createBatch: async (request: CreateBatchRequest): Promise<SortationBatch> => {
    return client.post<SortationBatch>('api/v1/batches', request);
  },

  getBatchesByStatus: async (status: string): Promise<SortationBatch[]> => {
    return client.get<SortationBatch[]>(`api/v1/batches/status/${status}`);
  },

  getReadyBatches: async (carrierId?: string, limit = 20): Promise<SortationBatch[]> => {
    const params = carrierId ? `?carrierId=${carrierId}&limit=${limit}` : `?limit=${limit}`;
    return client.get<SortationBatch[]>(`api/v1/batches/ready${params}`);
  },

  addPackage: async (
    batchId: string,
    packageData: SortPackageRequest
  ): Promise<SortationBatch> => {
    return client.post<SortationBatch>(`api/v1/batches/${batchId}/packages`, packageData);
  },

  sortPackage: async (
    batchId: string,
    packageId: string,
    chuteId: string,
    workerId: string
  ): Promise<SortationBatch> => {
    return client.post<SortationBatch>(`api/v1/batches/${batchId}/sort`, {
      packageId,
      chuteId,
      workerId,
    });
  },

  markReady: async (batchId: string): Promise<SortationBatch> => {
    return client.post<SortationBatch>(`api/v1/batches/${batchId}/ready`);
  },

  dispatchBatch: async (
    batchId: string,
    dispatchData: DispatchBatchRequest
  ): Promise<SortationBatch> => {
    return client.post<SortationBatch>(`api/v1/batches/${batchId}/dispatch`, dispatchData);
  },
};

export type { CreateBatchRequest, SortPackageRequest, DispatchBatchRequest };
