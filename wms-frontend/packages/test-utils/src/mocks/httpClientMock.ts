import { vi } from 'vitest';

export const createMockHttpClient = () => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  patch: vi.fn(),
  delete: vi.fn(),
  axiosInstance: {
    defaults: { baseURL: '/' },
    getUri: vi.fn((config: any) => `${config.url}`),
  },
});

export const mockHttpClient = createMockHttpClient();
