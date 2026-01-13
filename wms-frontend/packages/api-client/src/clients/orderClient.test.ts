import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => {
  const client = {
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
  };

  return { client };
});

vi.mock('../httpClient', () => ({
  createServiceClient: () => mocks.client,
}));

import { orderClient } from './orderClient';

describe('orderClient', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('builds a query string for filters', async () => {
    mocks.client.get.mockResolvedValueOnce({ data: [] });

    await orderClient.getOrders({
      status: 'PENDING',
      page: 2,
      pageSize: 50,
    });

    expect(mocks.client.get).toHaveBeenCalledWith('api/v1/orders?status=PENDING&page=2&pageSize=50');
  });

  it('requests a specific order by id', async () => {
    mocks.client.get.mockResolvedValueOnce({ id: 'order-1' });

    await orderClient.getOrder('order-1');

    expect(mocks.client.get).toHaveBeenCalledWith('api/v1/orders/order-1');
  });

  it('posts a new order payload', async () => {
    mocks.client.post.mockResolvedValueOnce({ id: 'order-2' });

    await orderClient.createOrder({
      customerId: 'cust-1',
      customerName: 'Jane Doe',
      items: [{ sku: 'SKU-1', productName: 'Product', quantity: 1 }],
    });

    expect(mocks.client.post).toHaveBeenCalledWith(
      'api/v1/orders',
      expect.objectContaining({ customerId: 'cust-1' })
    );
  });
});
