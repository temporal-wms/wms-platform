import { vi } from 'vitest';

export const createMockEventBus = () => ({
  emit: vi.fn(),
  on: vi.fn(() => vi.fn()), // Returns unsubscribe function
  once: vi.fn(() => vi.fn()),
  filter: vi.fn(() => vi.fn()),
  clear: vi.fn(),
});

export const mockEventBus = createMockEventBus();
