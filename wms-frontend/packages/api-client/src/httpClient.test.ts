import { describe, expect, it, vi, beforeEach } from 'vitest';
import { config } from '@wms/config';

const mocks = vi.hoisted(() => {
  const mockAxiosInstance = {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
    interceptors: {
      request: { use: vi.fn() },
      response: { use: vi.fn() },
    },
    defaults: {
      baseURL: '/',
    },
    getUri: vi.fn((config: any) => `${config.url}`),
  };

  return {
    mockAxiosInstance,
    mockAxiosCreate: vi.fn(() => mockAxiosInstance),
  };
});

vi.mock('axios', () => ({
  default: {
    create: mocks.mockAxiosCreate,
  },
}));

import { HttpClient, normalizeBaseUrl, createServiceClient } from './httpClient';

describe('normalizeBaseUrl', () => {
  it('returns root when base url is missing', () => {
    expect(normalizeBaseUrl(undefined)).toBe('/');
    expect(normalizeBaseUrl('')).toBe('/');
  });

  it('appends a trailing slash and removes duplicates', () => {
    expect(normalizeBaseUrl('/api/orders')).toBe('/api/orders/');
    expect(normalizeBaseUrl('/api/orders///')).toBe('/api/orders/');
  });

  it('keeps absolute URLs intact with trailing slash', () => {
    expect(normalizeBaseUrl('https://example.com/api/orders')).toBe('https://example.com/api/orders/');
  });
});

describe('HttpClient', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('constructor', () => {
    it('normalizes the provided base url', () => {
      const client = new HttpClient('/api/orders');

      expect(mocks.mockAxiosCreate).toHaveBeenCalledWith(
        expect.objectContaining({
          baseURL: '/api/orders/',
        })
      );
    });

    it('falls back to config.api.baseUrl when no base url is supplied', () => {
      const client = new HttpClient();

      expect(mocks.mockAxiosCreate).toHaveBeenCalledWith(
        expect.objectContaining({
          baseURL: normalizeBaseUrl(config.api.baseUrl),
        })
      );
    });

    it('sets timeout from config', () => {
      const client = new HttpClient();

      expect(mocks.mockAxiosCreate).toHaveBeenCalledWith(
        expect.objectContaining({
          timeout: config.api.timeout,
        })
      );
    });

    it('sets default Content-Type header', () => {
      const client = new HttpClient();

      expect(mocks.mockAxiosCreate).toHaveBeenCalledWith(
        expect.objectContaining({
          headers: {
            'Content-Type': 'application/json',
          },
        })
      );
    });

    it('sets up interceptors', () => {
      const client = new HttpClient();

      expect(mocks.mockAxiosInstance.interceptors.request.use).toHaveBeenCalled();
      expect(mocks.mockAxiosInstance.interceptors.response.use).toHaveBeenCalled();
    });
  });

  describe('HTTP methods', () => {
    let client: HttpClient;

    beforeEach(() => {
      client = new HttpClient('/api/test');
    });

    describe('get', () => {
      it('makes GET request and returns data', async () => {
        const mockData = { id: 1, name: 'Test' };
        mocks.mockAxiosInstance.get.mockResolvedValue({ data: mockData });

        const result = await client.get('/items');

        expect(mocks.mockAxiosInstance.get).toHaveBeenCalledWith('/items', undefined);
        expect(result).toEqual(mockData);
      });

      it('passes config to axios', async () => {
        mocks.mockAxiosInstance.get.mockResolvedValue({ data: {} });
        const config = { params: { page: 1 } };

        await client.get('/items', config);

        expect(mocks.mockAxiosInstance.get).toHaveBeenCalledWith('/items', config);
      });
    });

    describe('post', () => {
      it('makes POST request and returns data', async () => {
        const mockData = { id: 1, created: true };
        mocks.mockAxiosInstance.post.mockResolvedValue({ data: mockData });

        const payload = { name: 'New Item' };
        const result = await client.post('/items', payload);

        expect(mocks.mockAxiosInstance.post).toHaveBeenCalledWith('/items', payload, undefined);
        expect(result).toEqual(mockData);
      });

      it('passes config to axios', async () => {
        mocks.mockAxiosInstance.post.mockResolvedValue({ data: {} });
        const payload = { name: 'New Item' };
        const config = { headers: { 'X-Custom': 'value' } };

        await client.post('/items', payload, config);

        expect(mocks.mockAxiosInstance.post).toHaveBeenCalledWith('/items', payload, config);
      });
    });

    describe('put', () => {
      it('makes PUT request and returns data', async () => {
        const mockData = { id: 1, updated: true };
        mocks.mockAxiosInstance.put.mockResolvedValue({ data: mockData });

        const payload = { name: 'Updated Item' };
        const result = await client.put('/items/1', payload);

        expect(mocks.mockAxiosInstance.put).toHaveBeenCalledWith('/items/1', payload, undefined);
        expect(result).toEqual(mockData);
      });
    });

    describe('patch', () => {
      it('makes PATCH request and returns data', async () => {
        const mockData = { id: 1, patched: true };
        mocks.mockAxiosInstance.patch.mockResolvedValue({ data: mockData });

        const payload = { status: 'ACTIVE' };
        const result = await client.patch('/items/1', payload);

        expect(mocks.mockAxiosInstance.patch).toHaveBeenCalledWith('/items/1', payload, undefined);
        expect(result).toEqual(mockData);
      });
    });

    describe('delete', () => {
      it('makes DELETE request and returns data', async () => {
        const mockData = { deleted: true };
        mocks.mockAxiosInstance.delete.mockResolvedValue({ data: mockData });

        const result = await client.delete('/items/1');

        expect(mocks.mockAxiosInstance.delete).toHaveBeenCalledWith('/items/1', undefined);
        expect(result).toEqual(mockData);
      });
    });
  });

  describe('interceptors', () => {
    let requestInterceptor: any;
    let responseInterceptor: any;

    beforeEach(() => {
      mocks.mockAxiosInstance.interceptors.request.use.mockImplementation((onFulfilled: any) => {
        requestInterceptor = onFulfilled;
      });

      mocks.mockAxiosInstance.interceptors.response.use.mockImplementation(
        (onFulfilled: any, onRejected: any) => {
          responseInterceptor = { onFulfilled, onRejected };
        }
      );

      new HttpClient('/api/test');
    });

    describe('request interceptor', () => {
      it('passes through request config', () => {
        const config = { url: '/test', headers: {} };
        const result = requestInterceptor(config);

        expect(result).toEqual(config);
      });
    });

    describe('response interceptor', () => {
      it('passes through successful responses', () => {
        const response = { data: { success: true } };
        const result = responseInterceptor.onFulfilled(response);

        expect(result).toEqual(response);
      });

      it('transforms axios errors to ApiError format', async () => {
        const axiosError = {
          response: {
            status: 404,
            data: {
              code: 'NOT_FOUND',
              message: 'Resource not found',
              details: { resource: 'Order', id: '123' },
            },
          },
          config: {
            url: '/orders/123',
            method: 'get',
          },
          code: 'ERR_BAD_REQUEST',
          message: 'Request failed',
        };

        await expect(responseInterceptor.onRejected(axiosError)).rejects.toEqual({
          code: 'NOT_FOUND',
          message: 'Resource not found',
          details: { resource: 'Order', id: '123' },
        });
      });

      it('handles errors without response data', async () => {
        const axiosError = {
          code: 'NETWORK_ERROR',
          message: 'Network Error',
          config: { url: '/test' },
        };

        await expect(responseInterceptor.onRejected(axiosError)).rejects.toEqual({
          code: 'NETWORK_ERROR',
          message: 'Network Error',
          details: undefined,
        });
      });

      it('handles errors without code or message', async () => {
        const axiosError = {
          config: { url: '/test' },
        };

        await expect(responseInterceptor.onRejected(axiosError)).rejects.toEqual({
          code: 'UNKNOWN_ERROR',
          message: 'An unexpected error occurred',
          details: undefined,
        });
      });

      it('logs error in debug mode', async () => {
        const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
        const originalDebugMode = config.features.debugMode;
        config.features.debugMode = true;

        const axiosError = {
          response: {
            status: 500,
            data: { code: 'SERVER_ERROR', message: 'Internal error' },
          },
          config: { url: '/test', method: 'post' },
        };

        await expect(responseInterceptor.onRejected(axiosError)).rejects.toEqual({
          code: 'SERVER_ERROR',
          message: 'Internal error',
          details: undefined,
        });

        expect(consoleErrorSpy).toHaveBeenCalledWith(
          '[API Error]',
          expect.objectContaining({
            url: '/test',
            method: 'post',
            status: 500,
          })
        );

        config.features.debugMode = originalDebugMode;
        consoleErrorSpy.mockRestore();
      });

      it('does not log in non-debug mode', async () => {
        const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
        const originalDebugMode = config.features.debugMode;
        config.features.debugMode = false;

        const axiosError = {
          response: { data: { code: 'ERROR', message: 'Error' } },
          config: { url: '/test' },
        };

        await expect(responseInterceptor.onRejected(axiosError)).rejects.toEqual({
          code: 'ERROR',
          message: 'Error',
          details: undefined,
        });

        expect(consoleErrorSpy).not.toHaveBeenCalled();

        config.features.debugMode = originalDebugMode;
        consoleErrorSpy.mockRestore();
      });
    });
  });

  describe('axiosInstance getter', () => {
    it('exposes the axios instance', () => {
      const client = new HttpClient('/api/test');

      expect(client.axiosInstance).toBe(mocks.mockAxiosInstance);
    });
  });
});

describe('createServiceClient', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('creates client with service URL from config', () => {
    const client = createServiceClient('orderService');

    expect(mocks.mockAxiosCreate).toHaveBeenCalledWith(
      expect.objectContaining({
        baseURL: normalizeBaseUrl(config.services.orderService),
      })
    );
  });

  it('works with different service names', () => {
    createServiceClient('waveService');

    expect(mocks.mockAxiosCreate).toHaveBeenCalledWith(
      expect.objectContaining({
        baseURL: normalizeBaseUrl(config.services.waveService),
      })
    );
  });
});
