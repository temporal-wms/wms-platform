import axios, { AxiosInstance, AxiosRequestConfig, AxiosError } from 'axios';
import { config } from '@wms/config';
import type { ApiError } from '@wms/types';

const normalizeBaseUrl = (baseUrl?: string): string | undefined => {
  if (!baseUrl) {
    return '/';
  }

  const trimmed = baseUrl.replace(/\/+$/, '');
  return `${trimmed}/`;
};

export class HttpClient {
  private client: AxiosInstance;

  constructor(baseURL?: string) {
    this.client = axios.create({
      baseURL: normalizeBaseUrl(baseURL ?? config.api.baseUrl),
      timeout: config.api.timeout,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    this.setupInterceptors();
  }

  private setupInterceptors(): void {
    // Request interceptor
    this.client.interceptors.request.use(
      (config) => {
        // Add any auth headers here if needed in the future
        return config;
      },
      (error) => Promise.reject(error)
    );

    // Response interceptor
    this.client.interceptors.response.use(
      (response) => response,
      (error: AxiosError<ApiError>) => {
        const apiError: ApiError = {
          code: error.response?.data?.code || error.code || 'UNKNOWN_ERROR',
          message: error.response?.data?.message || error.message || 'An unexpected error occurred',
          details: error.response?.data?.details,
        };

        if (config.features.debugMode) {
          console.error('[API Error]', {
            url: error.config?.url,
            method: error.config?.method,
            status: error.response?.status,
            error: apiError,
          });
        }

        return Promise.reject(apiError);
      }
    );
  }

  async get<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.client.get<T>(url, config);
    return response.data;
  }

  async post<T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.client.post<T>(url, data, config);
    return response.data;
  }

  async put<T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.client.put<T>(url, data, config);
    return response.data;
  }

  async patch<T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.client.patch<T>(url, data, config);
    return response.data;
  }

  async delete<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.client.delete<T>(url, config);
    return response.data;
  }

  // Exposed for testing/advanced scenarios
  get axiosInstance(): AxiosInstance {
    return this.client;
  }
}

// Default HTTP client using API gateway
export const httpClient = new HttpClient();

// Service-specific clients for direct access
export const createServiceClient = (serviceName: keyof typeof config.services): HttpClient => {
  return new HttpClient(config.services[serviceName]);
};

export { normalizeBaseUrl };
