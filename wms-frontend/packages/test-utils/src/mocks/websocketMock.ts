import { vi } from 'vitest';

export class MockWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  readyState = MockWebSocket.OPEN;
  onopen: ((ev: Event) => void) | null = null;
  onclose: ((ev: CloseEvent) => void) | null = null;
  onerror: ((ev: Event) => void) | null = null;
  onmessage: ((ev: MessageEvent) => void) | null = null;

  constructor(public url: string) {}

  send = vi.fn();
  close = vi.fn();

  // Test helpers
  simulateOpen() {
    this.readyState = MockWebSocket.OPEN;
    this.onopen?.(new Event('open'));
  }

  simulateMessage(data: any) {
    this.onmessage?.(new MessageEvent('message', { data: JSON.stringify(data) }));
  }

  simulateClose() {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.(new CloseEvent('close'));
  }

  simulateError(error: Event) {
    this.onerror?.(error);
  }
}

export const createMockWMSSocket = () => ({
  connect: vi.fn(),
  disconnect: vi.fn(),
  subscribe: vi.fn(() => vi.fn()), // Returns unsubscribe
  send: vi.fn(),
  isConnected: true,
});

export const mockWMSSocket = createMockWMSSocket();
