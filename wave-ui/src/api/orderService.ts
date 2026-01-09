import axios from 'axios';
import type { PagedOrdersResult } from '../types';

// Use environment variable for dev, or relative path (nginx proxy) for production
const ORDER_SERVICE_URL = import.meta.env.VITE_ORDER_SERVICE_URL || '';

export const fetchValidatedOrders = async (): Promise<PagedOrdersResult> => {
  const baseUrl = ORDER_SERVICE_URL || '/api/orders';
  const response = await axios.get<PagedOrdersResult>(
    `${baseUrl}/v1/orders/status/validated`,
    {
      params: {
        limit: 100,
        sortBy: 'promisedDeliveryAt',
        sortOrder: 'asc'
      }
    }
  );
  return response.data;
};
