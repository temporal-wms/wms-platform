import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => {
  const client = {
    get: vi.fn(),
    post: vi.fn(),
  };
  return { client };
});

vi.mock('../httpClient', () => ({
  createServiceClient: () => mocks.client,
}));

import { inventoryClient } from './inventoryClient';

describe('inventoryClient', () => {
  beforeEach(() => {
    mocks.client.get.mockReset();
    mocks.client.post.mockReset();
  });

  it('serializes filters for inventory queries', async () => {
    mocks.client.get.mockResolvedValueOnce({ data: [] });

    await inventoryClient.getInventory({ sku: 'SKU-1', page: 3, pageSize: 10 });

    expect(mocks.client.get).toHaveBeenCalledWith('api/v1/inventory?sku=SKU-1&page=3&pageSize=10');
  });

  it('sends reserve payloads to the service', async () => {
    mocks.client.post.mockResolvedValueOnce({ success: true });

    await inventoryClient.reserveInventory('SKU-1', 5, 'order-9');

    expect(mocks.client.post).toHaveBeenCalledWith('api/v1/inventory/reserve', {
      sku: 'SKU-1',
      quantity: 5,
      orderId: 'order-9',
    });
  });

  it('fetches locations with optional zone filter', async () => {
    mocks.client.get.mockResolvedValueOnce([]);

    await inventoryClient.getLocations('A');

    expect(mocks.client.get).toHaveBeenCalledWith('api/v1/locations?zone=A');
  });
});
