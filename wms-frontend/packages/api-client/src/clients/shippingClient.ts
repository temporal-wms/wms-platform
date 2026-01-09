import { createServiceClient } from '../httpClient';
import type { Shipment, PaginatedResponse } from '@wms/types';

const client = createServiceClient('shipping');

export interface ShipmentFilters {
  status?: string;
  carrier?: string;
  fromDate?: string;
  toDate?: string;
  page?: number;
  pageSize?: number;
}

export interface CreateShipmentRequest {
  orderId: string;
  carrier: string;
  serviceLevel: 'GROUND' | 'EXPRESS' | 'OVERNIGHT' | 'FREIGHT';
  packages: string[]; // Package IDs
}

export const shippingClient = {
  // Get shipments with filters
  getShipments: async (filters?: ShipmentFilters): Promise<PaginatedResponse<Shipment>> => {
    const params = new URLSearchParams();
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined) params.append(key, String(value));
      });
    }
    return client.get<PaginatedResponse<Shipment>>(`api/v1/shipments?${params.toString()}`);
  },

  // Get single shipment
  getShipment: async (shipmentId: string): Promise<Shipment> => {
    return client.get<Shipment>(`api/v1/shipments/${shipmentId}`);
  },

  // Create shipment
  createShipment: async (request: CreateShipmentRequest): Promise<Shipment> => {
    return client.post<Shipment>('api/v1/shipments', request);
  },

  // Generate shipping label
  generateLabel: async (shipmentId: string): Promise<{ labelUrl: string; trackingNumber: string }> => {
    return client.post(`api/v1/shipments/${shipmentId}/label`);
  },

  // Mark as shipped
  markShipped: async (shipmentId: string): Promise<Shipment> => {
    return client.post<Shipment>(`api/v1/shipments/${shipmentId}/ship`);
  },

  // Get tracking info
  getTracking: async (shipmentId: string): Promise<{
    status: string;
    events: Array<{ timestamp: string; location: string; status: string; description: string }>;
  }> => {
    return client.get(`api/v1/shipments/${shipmentId}/tracking`);
  },

  // Get available carriers
  getCarriers: async (): Promise<Array<{ id: string; name: string; serviceLevels: string[] }>> => {
    return client.get('api/v1/carriers');
  },

  // Get shipping rates
  getRates: async (orderId: string): Promise<Array<{ carrier: string; service: string; rate: number; estimatedDays: number }>> => {
    return client.get(`api/v1/shipments/rates?orderId=${orderId}`);
  },

  // Get shipping statistics
  getShippingStats: async (): Promise<{
    pending: number;
    shippedToday: number;
    inTransit: number;
    delivered: number;
    byCarrier: Record<string, number>;
  }> => {
    return client.get('api/v1/shipments/stats');
  },
};
