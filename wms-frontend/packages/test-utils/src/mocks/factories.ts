// Mock data factories for WMS types

export const createMockOrder = (overrides?: Partial<any>): any => ({
  id: 'order-1',
  orderNumber: 'ORD-001',
  customerId: 'cust-1',
  customerName: 'John Doe',
  status: 'PENDING',
  priority: 'NORMAL',
  items: [
    {
      id: 'item-1',
      sku: 'SKU-001',
      productName: 'Product 1',
      quantity: 2,
      pickedQuantity: 0,
      packedQuantity: 0,
    },
  ],
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
  ...overrides,
});

export const createMockWave = (overrides?: Partial<any>): any => ({
  id: 'wave-1',
  waveNumber: 'WAVE-001',
  status: 'PLANNING',
  orderIds: ['order-1', 'order-2'],
  orderCount: 2,
  priority: 'NORMAL',
  createdAt: '2026-01-01T00:00:00Z',
  ...overrides,
});

export const createMockPickTask = (overrides?: Partial<any>): any => ({
  id: 'pick-1',
  orderId: 'order-1',
  waveId: 'wave-1',
  status: 'PENDING',
  items: [
    {
      id: 'item-1',
      sku: 'SKU-001',
      productName: 'Product 1',
      quantity: 2,
      pickedQuantity: 0,
      locationId: 'loc-1',
      locationCode: 'A-01-01',
      sequence: 1,
    },
  ],
  route: {
    id: 'route-1',
    stops: [],
    totalDistance: 100,
    estimatedTime: 300,
  },
  createdAt: '2026-01-01T00:00:00Z',
  ...overrides,
});

export const createMockPackTask = (overrides?: Partial<any>): any => ({
  id: 'pack-1',
  orderId: 'order-1',
  status: 'PENDING',
  items: [],
  createdAt: '2026-01-01T00:00:00Z',
  ...overrides,
});

export const createMockWorker = (overrides?: Partial<any>): any => ({
  id: 'worker-1',
  employeeId: 'EMP-001',
  name: 'Jane Smith',
  role: 'PICKER',
  status: 'AVAILABLE',
  tasksCompleted: 10,
  itemsProcessed: 50,
  ...overrides,
});

export const createMockInventoryItem = (overrides?: Partial<any>): any => ({
  id: 'inv-1',
  sku: 'SKU-001',
  productName: 'Product 1',
  quantity: 100,
  reservedQuantity: 10,
  availableQuantity: 90,
  locationId: 'loc-1',
  locationCode: 'A-01-01',
  zone: 'Zone A',
  updatedAt: '2026-01-01T00:00:00Z',
  ...overrides,
});

export const createMockPaginatedResponse = <T>(
  data: T[],
  options?: { page?: number; pageSize?: number; total?: number }
) => ({
  data,
  page: options?.page || 1,
  pageSize: options?.pageSize || 20,
  total: options?.total || data.length,
  totalPages: Math.ceil((options?.total || data.length) / (options?.pageSize || 20)),
});
