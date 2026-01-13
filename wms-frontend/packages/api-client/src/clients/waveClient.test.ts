import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => {
  const client = {
    get: vi.fn(),
    post: vi.fn(),
    delete: vi.fn(),
  };
  return { client };
});

vi.mock('../httpClient', () => ({
  createServiceClient: () => mocks.client,
}));

import { waveClient } from './waveClient';

describe('waveClient', () => {
  beforeEach(() => {
    mocks.client.get.mockReset();
    mocks.client.post.mockReset();
    mocks.client.delete.mockReset();
  });

  it('builds query params when requesting waves', async () => {
    mocks.client.get.mockResolvedValueOnce({ data: [] });

    await waveClient.getWaves({ status: 'READY', page: 2, pageSize: 25 });

    expect(mocks.client.get).toHaveBeenCalledWith('api/v1/waves?status=READY&page=2&pageSize=25');
  });

  it('releases a wave by id', async () => {
    mocks.client.post.mockResolvedValueOnce({});

    await waveClient.releaseWave('wave-123');

    expect(mocks.client.post).toHaveBeenCalledWith('api/v1/waves/wave-123/release');
  });

  it('removes orders using DELETE payload', async () => {
    mocks.client.delete.mockResolvedValueOnce({});

    await waveClient.removeOrdersFromWave('wave-1', ['order-1', 'order-2']);

    expect(mocks.client.delete).toHaveBeenCalledWith('api/v1/waves/wave-1/orders', {
      data: { orderIds: ['order-1', 'order-2'] },
    });
  });
});
