import { createServiceClient } from '../httpClient';
import type { DashboardMetrics } from '@wms/types';

// Dashboard client uses order service as a proxy for aggregated metrics
const client = createServiceClient('orders');

export const dashboardClient = {
  // Get all dashboard metrics
  getMetrics: async (): Promise<DashboardMetrics> => {
    return client.get<DashboardMetrics>('api/v1/dashboard/metrics');
  },

  // Get order metrics
  getOrderMetrics: async (): Promise<DashboardMetrics['orders']> => {
    return client.get('api/v1/dashboard/metrics/orders');
  },

  // Get wave metrics
  getWaveMetrics: async (): Promise<DashboardMetrics['waves']> => {
    return client.get('api/v1/dashboard/metrics/waves');
  },

  // Get picking metrics
  getPickingMetrics: async (): Promise<DashboardMetrics['picking']> => {
    return client.get('api/v1/dashboard/metrics/picking');
  },

  // Get packing metrics
  getPackingMetrics: async (): Promise<DashboardMetrics['packing']> => {
    return client.get('api/v1/dashboard/metrics/packing');
  },

  // Get shipping metrics
  getShippingMetrics: async (): Promise<DashboardMetrics['shipping']> => {
    return client.get('api/v1/dashboard/metrics/shipping');
  },

  // Get labor metrics
  getLaborMetrics: async (): Promise<DashboardMetrics['labor']> => {
    return client.get('api/v1/dashboard/metrics/labor');
  },

  // Get throughput data (for charts)
  getThroughput: async (period: 'hour' | 'day' | 'week' = 'day'): Promise<{
    labels: string[];
    orders: number[];
    picks: number[];
    shipments: number[];
  }> => {
    return client.get(`api/v1/dashboard/throughput?period=${period}`);
  },

  // Get alerts
  getAlerts: async (): Promise<Array<{
    id: string;
    type: 'warning' | 'error' | 'info';
    title: string;
    message: string;
    timestamp: string;
    resolved: boolean;
  }>> => {
    return client.get('api/v1/dashboard/alerts');
  },

  // Resolve alert
  resolveAlert: async (alertId: string): Promise<void> => {
    return client.post(`api/v1/dashboard/alerts/${alertId}/resolve`);
  },
};
