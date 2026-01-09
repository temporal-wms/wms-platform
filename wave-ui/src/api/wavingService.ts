import axios from 'axios';
import type { CreateWaveRequest, CreateWaveResponse } from '../types';

// Use environment variable for dev, or relative path (nginx proxy) for production
const WAVING_SERVICE_URL = import.meta.env.VITE_WAVING_SERVICE_URL || '';

export const createWaveFromOrders = async (request: CreateWaveRequest): Promise<CreateWaveResponse> => {
  const baseUrl = WAVING_SERVICE_URL || '/api/waves';
  const response = await axios.post<CreateWaveResponse>(
    `${baseUrl}/v1/waves/from-orders`,
    request
  );
  return response.data;
};
