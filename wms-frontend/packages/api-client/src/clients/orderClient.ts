import { httpClient, createServiceClient } from '../httpClient';
import type { Order, OrderItem, PaginatedResponse, ApiResponse } from '@wms/types';

const client = createServiceClient('orders');

export interface CreateOrderRequest {
  customerId: string;
  customerName: string;
  items: Array<{
    sku: string;
    productName: string;
    quantity: number;
  }>;
  priority?: 'LOW' | 'NORMAL' | 'HIGH' | 'RUSH';
}

export interface OrderFilters {
  status?: string;
  priority?: string;
  customerId?: string;
  fromDate?: string;
  toDate?: string;
  search?: string;
  page?: number;
  pageSize?: number;
}

export const orderClient = {
  // Get all orders with optional filters
  getOrders: async (filters?: OrderFilters): Promise<PaginatedResponse<Order>> => {
    const params = new URLSearchParams();
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined) params.append(key, String(value));
      });
    }
    return client.get<PaginatedResponse<Order>>(`api/v1/orders?${params.toString()}`);
  },

  // Get single order by ID
  getOrder: async (orderId: string): Promise<Order> => {
    return client.get<Order>(`api/v1/orders/${orderId}`);
  },

  // Create new order
  createOrder: async (request: CreateOrderRequest): Promise<Order> => {
    return client.post<Order>('api/v1/orders', request);
  },

  // Update order
  updateOrder: async (orderId: string, updates: Partial<Order>): Promise<Order> => {
    return client.patch<Order>(`api/v1/orders/${orderId}`, updates);
  },

  // Cancel order
  cancelOrder: async (orderId: string): Promise<void> => {
    return client.post<void>(`api/v1/orders/${orderId}/cancel`);
  },

  // Get orders in DLQ
  getDLQOrders: async (page = 1, pageSize = 20): Promise<PaginatedResponse<Order>> => {
    return client.get<PaginatedResponse<Order>>(`api/v1/orders/dlq?page=${page}&pageSize=${pageSize}`);
  },

  // Retry DLQ order
  retryDLQOrder: async (orderId: string): Promise<Order> => {
    return client.post<Order>(`api/v1/orders/${orderId}/retry`);
  },

  // Get order statistics
  getOrderStats: async (): Promise<{
    total: number;
    byStatus: Record<string, number>;
    byPriority: Record<string, number>;
    todayCount: number;
  }> => {
    return client.get('api/v1/orders/stats');
  },
};
