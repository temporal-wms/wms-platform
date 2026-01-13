import { httpClient, createServiceClient } from '../httpClient';
import type {
  ReceivingShipment,
  CreateShipmentRequest,
  ExpectedItem,
  ReceiveItemRequest,
  PaginatedResponse,
} from '@wms/types';

const client = createServiceClient('receiving');

export interface ReceivingShipmentFilters {
  status?: string;
  supplierId?: string;
  fromDate?: string;
  toDate?: string;
  search?: string;
  page?: number;
  pageSize?: number;
}

export const receivingClient = {
  getShipments: async (filters?: ReceivingShipmentFilters): Promise<PaginatedResponse<ReceivingShipment>> => {
    const params = new URLSearchParams();
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined) params.append(key, String(value));
      });
    }
    return client.get<PaginatedResponse<ReceivingShipment>>(`api/v1/shipments?${params.toString()}`);
  },

  getShipment: async (shipmentId: string): Promise<ReceivingShipment> => {
    return client.get<ReceivingShipment>(`api/v1/shipments/${shipmentId}`);
  },

  createShipment: async (request: CreateShipmentRequest): Promise<ReceivingShipment> => {
    return client.post<ReceivingShipment>('api/v1/shipments', request);
  },

  markArrived: async (shipmentId: string, dockId: string): Promise<ReceivingShipment> => {
    return client.post<ReceivingShipment>(`api/v1/shipments/${shipmentId}/arrive`, { dockId });
  },

  startReceiving: async (shipmentId: string, workerId: string): Promise<ReceivingShipment> => {
    return client.post<ReceivingShipment>(`api/v1/shipments/${shipmentId}/start`, { workerId });
  },

  receiveItem: async (
    shipmentId: string,
    item: ReceiveItemRequest
  ): Promise<ReceivingShipment> => {
    return client.post<ReceivingShipment>(`api/v1/shipments/${shipmentId}/receive`, item);
  },

  completeReceiving: async (shipmentId: string): Promise<ReceivingShipment> => {
    return client.post<ReceivingShipment>(`api/v1/shipments/${shipmentId}/complete`, {});
  },

  getExpectedArrivals: async (date?: string): Promise<ReceivingShipment[]> => {
    const params = date ? `?date=${date}` : '';
    return client.get<ReceivingShipment[]>(`api/v1/shipments/expected${params}`);
  },

  getShipmentsByStatus: async (status: string): Promise<ReceivingShipment[]> => {
    return client.get<ReceivingShipment[]>(`api/v1/shipments/status/${status}`);
  },
};

export type { CreateShipmentRequest, ReceiveItemRequest, ReceivingShipmentFilters as ShipmentFilters };
