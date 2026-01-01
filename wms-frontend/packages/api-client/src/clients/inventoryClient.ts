import { createServiceClient } from '../httpClient';
import type { InventoryItem, Location, PaginatedResponse } from '@wms/types';

const client = createServiceClient('inventory');

export interface InventoryFilters {
  sku?: string;
  zone?: string;
  locationId?: string;
  minQuantity?: number;
  maxQuantity?: number;
  page?: number;
  pageSize?: number;
}

export interface AdjustInventoryRequest {
  sku: string;
  locationId: string;
  quantity: number;
  reason: 'CYCLE_COUNT' | 'DAMAGE' | 'RETURN' | 'RECEIVING' | 'ADJUSTMENT';
  notes?: string;
}

export const inventoryClient = {
  // Get inventory items with filters
  getInventory: async (filters?: InventoryFilters): Promise<PaginatedResponse<InventoryItem>> => {
    const params = new URLSearchParams();
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined) params.append(key, String(value));
      });
    }
    return client.get<PaginatedResponse<InventoryItem>>(`api/v1/inventory?${params.toString()}`);
  },

  // Get inventory by SKU
  getInventoryBySku: async (sku: string): Promise<InventoryItem[]> => {
    return client.get<InventoryItem[]>(`api/v1/inventory/sku/${sku}`);
  },

  // Get inventory at location
  getInventoryByLocation: async (locationId: string): Promise<InventoryItem[]> => {
    return client.get<InventoryItem[]>(`api/v1/inventory/location/${locationId}`);
  },

  // Adjust inventory
  adjustInventory: async (request: AdjustInventoryRequest): Promise<InventoryItem> => {
    return client.post<InventoryItem>('api/v1/inventory/adjust', request);
  },

  // Reserve inventory
  reserveInventory: async (sku: string, quantity: number, orderId: string): Promise<{ success: boolean; reserved: number }> => {
    return client.post('api/v1/inventory/reserve', { sku, quantity, orderId });
  },

  // Release reservation
  releaseReservation: async (sku: string, quantity: number, orderId: string): Promise<void> => {
    return client.post('api/v1/inventory/release', { sku, quantity, orderId });
  },

  // Get all locations
  getLocations: async (zone?: string): Promise<Location[]> => {
    const url = zone ? `api/v1/locations?zone=${zone}` : 'api/v1/locations';
    return client.get<Location[]>(url);
  },

  // Get location by ID
  getLocation: async (locationId: string): Promise<Location> => {
    return client.get<Location>(`api/v1/locations/${locationId}`);
  },

  // Get inventory statistics
  getInventoryStats: async (): Promise<{
    totalItems: number;
    totalSkus: number;
    lowStockItems: number;
    outOfStockItems: number;
    byZone: Record<string, number>;
  }> => {
    return client.get('api/v1/inventory/stats');
  },

  // Get low stock alerts
  getLowStockAlerts: async (): Promise<Array<{ sku: string; productName: string; available: number; reorderPoint: number }>> => {
    return client.get('api/v1/inventory/alerts/low-stock');
  },
};
