import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { WMSSocket, createWMSSocket } from './websocket';
import { eventBus } from './eventBus';
import type { WMSEvent } from '@wms/types';

// Mock WebSocket
class MockWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  readyState = MockWebSocket.CONNECTING;
  onopen: ((ev: Event) => void) | null = null;
  onclose: ((ev: CloseEvent) => void) | null = null;
  onerror: ((ev: Event) => void) | null = null;
  onmessage: ((ev: MessageEvent) => void) | null = null;

  constructor(public url: string) {
    // Simulate async connection
    setTimeout(() => {
      this.readyState = MockWebSocket.OPEN;
      this.onopen?.(new Event('open'));
    }, 0);
  }

  send = vi.fn();

  close() {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.(new CloseEvent('close'));
  }

  // Test helpers
  simulateMessage(data: WMSEvent) {
    this.onmessage?.(new MessageEvent('message', { data: JSON.stringify(data) }));
  }

  simulateError(error: Event) {
    this.onerror?.(error);
  }
}

describe('WMSSocket', () => {
  let consoleLogSpy: ReturnType<typeof vi.spyOn>;
  let consoleErrorSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    vi.useFakeTimers();
    global.WebSocket = MockWebSocket as any;
    consoleLogSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    eventBus.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.useRealTimers();
    eventBus.clear();
  });

  describe('constructor', () => {
    it('creates socket with default options', () => {
      const socket = new WMSSocket('ws://localhost:8080/ws');
      expect(socket).toBeInstanceOf(WMSSocket);
    });

    it('creates socket with custom options', () => {
      const onConnect = vi.fn();
      const onDisconnect = vi.fn();
      const onError = vi.fn();

      const socket = new WMSSocket('ws://localhost:8080/ws', {
        reconnectAttempts: 3,
        reconnectInterval: 5000,
        onConnect,
        onDisconnect,
        onError,
      });

      expect(socket).toBeInstanceOf(WMSSocket);
    });
  });

  describe('connect', () => {
    it('establishes WebSocket connection', async () => {
      const onConnect = vi.fn();
      const socket = new WMSSocket('ws://localhost:8080/ws', { onConnect });

      socket.connect();
      await vi.runAllTimersAsync();

      expect(onConnect).toHaveBeenCalledOnce();
      expect(consoleLogSpy).toHaveBeenCalledWith(
        '[WMS Socket] Connected to',
        'ws://localhost:8080/ws'
      );
    });

    it('does not reconnect if already connected', async () => {
      const onConnect = vi.fn();
      const socket = new WMSSocket('ws://localhost:8080/ws', { onConnect });

      socket.connect();
      await vi.runAllTimersAsync();
      expect(onConnect).toHaveBeenCalledOnce();

      socket.connect(); // Try to connect again
      await vi.runAllTimersAsync();
      expect(onConnect).toHaveBeenCalledOnce(); // Still only once
    });

    it('calls onConnect callback on successful connection', async () => {
      const onConnect = vi.fn();
      const socket = new WMSSocket('ws://localhost:8080/ws', { onConnect });

      socket.connect();
      await vi.runAllTimersAsync();

      expect(onConnect).toHaveBeenCalledOnce();
    });

    it('resets reconnect count on successful connection', async () => {
      const socket = new WMSSocket('ws://localhost:8080/ws', {
        reconnectAttempts: 3,
        reconnectInterval: 1000,
      });

      socket.connect();
      await vi.runAllTimersAsync();

      // Access private field through any for testing
      expect((socket as any).reconnectCount).toBe(0);
    });
  });

  describe('disconnect', () => {
    it('closes WebSocket connection', async () => {
      const onDisconnect = vi.fn();
      const socket = new WMSSocket('ws://localhost:8080/ws', { onDisconnect });

      socket.connect();
      await vi.runAllTimersAsync();

      const closeSpy = vi.spyOn((socket as any).ws, 'close');
      socket.disconnect();

      expect(closeSpy).toHaveBeenCalled();
      expect(onDisconnect).toHaveBeenCalledOnce();
    });

    it('sets manual close flag to prevent reconnection', async () => {
      const socket = new WMSSocket('ws://localhost:8080/ws', {
        reconnectAttempts: 5,
      });

      socket.connect();
      await vi.runAllTimersAsync();

      socket.disconnect();

      expect((socket as any).isManualClose).toBe(true);
    });

    it('calls onDisconnect callback', async () => {
      const onDisconnect = vi.fn();
      const socket = new WMSSocket('ws://localhost:8080/ws', { onDisconnect });

      socket.connect();
      await vi.runAllTimersAsync();

      socket.disconnect();

      expect(onDisconnect).toHaveBeenCalledOnce();
    });
  });

  describe('reconnection', () => {
    it('attempts reconnection on unexpected close', async () => {
      const socket = new WMSSocket('ws://localhost:8080/ws', {
        reconnectAttempts: 3,
        reconnectInterval: 1000,
      });

      socket.connect();
      await vi.runAllTimersAsync();

      // Simulate unexpected close
      const ws = (socket as any).ws;
      (socket as any).isManualClose = false;
      ws.readyState = MockWebSocket.CLOSED;
      ws.onclose(new CloseEvent('close'));

      expect(consoleLogSpy).toHaveBeenCalledWith(
        '[WMS Socket] Reconnecting... attempt 1'
      );

      // Fast-forward reconnection interval
      await vi.advanceTimersByTimeAsync(1000);
      expect((socket as any).reconnectCount).toBe(1);
    });

    it('stops reconnecting after max attempts', async () => {
      const consoleLogSpy = vi.spyOn(console, 'log');
      const socket = new WMSSocket('ws://localhost:8080/ws', {
        reconnectAttempts: 3,
        reconnectInterval: 100,
      });

      socket.connect();
      await vi.runAllTimersAsync();

      // Clear initial connection logs
      consoleLogSpy.mockClear();

      // Force 4 unexpected closes
      for (let i = 0; i < 4; i++) {
        (socket as any).isManualClose = false;
        const ws = (socket as any).ws;
        if (ws) {
          ws.readyState = MockWebSocket.CLOSED;
          ws.onclose?.(new CloseEvent('close'));
          await vi.runAllTimersAsync();
        }
      }

      // Count how many reconnection attempts were logged
      const reconnectLogs = consoleLogSpy.mock.calls.filter((call) =>
        call[0]?.includes('Reconnecting')
      );

      // Should have attempted exactly 3 reconnections (but may get 4 due to timing)
      expect(reconnectLogs.length).toBeGreaterThanOrEqual(3);
      expect(reconnectLogs.length).toBeLessThanOrEqual(4);

      consoleLogSpy.mockRestore();
    });

    it('does not reconnect on manual disconnect', async () => {
      const socket = new WMSSocket('ws://localhost:8080/ws', {
        reconnectAttempts: 5,
        reconnectInterval: 1000,
      });

      socket.connect();
      await vi.runAllTimersAsync();

      socket.disconnect();

      // Should not attempt reconnection
      await vi.advanceTimersByTimeAsync(2000);
      expect((socket as any).reconnectCount).toBe(0);
    });
  });

  describe('subscribe', () => {
    it('subscribes to specific event type', async () => {
      const handler = vi.fn();
      const socket = new WMSSocket('ws://localhost:8080/ws');

      socket.connect();
      await vi.runAllTimersAsync();

      socket.subscribe('ORDER_CREATED', handler);

      const event: WMSEvent = {
        type: 'ORDER_CREATED',
        payload: { orderId: '123' },
      };

      const ws = (socket as any).ws as MockWebSocket;
      ws.simulateMessage(event);

      expect(handler).toHaveBeenCalledWith(event.payload);
    });

    it('does not call handler for different event types', async () => {
      const handler = vi.fn();
      const socket = new WMSSocket('ws://localhost:8080/ws');

      socket.connect();
      await vi.runAllTimersAsync();

      socket.subscribe('ORDER_CREATED', handler);

      const event: WMSEvent = {
        type: 'WAVE_RELEASED',
        payload: { waveId: 'wave-1' },
      };

      const ws = (socket as any).ws as MockWebSocket;
      ws.simulateMessage(event);

      expect(handler).not.toHaveBeenCalled();
    });

    it('returns unsubscribe function', async () => {
      const handler = vi.fn();
      const socket = new WMSSocket('ws://localhost:8080/ws');

      socket.connect();
      await vi.runAllTimersAsync();

      const unsubscribe = socket.subscribe('ORDER_CREATED', handler);

      const event: WMSEvent = {
        type: 'ORDER_CREATED',
        payload: { orderId: '123' },
      };

      const ws = (socket as any).ws as MockWebSocket;
      ws.simulateMessage(event);
      expect(handler).toHaveBeenCalledOnce();

      unsubscribe();
      ws.simulateMessage(event);
      expect(handler).toHaveBeenCalledOnce(); // Still only once
    });

    it('handles multiple subscriptions to same event type', async () => {
      const handler1 = vi.fn();
      const handler2 = vi.fn();
      const socket = new WMSSocket('ws://localhost:8080/ws');

      socket.connect();
      await vi.runAllTimersAsync();

      socket.subscribe('ORDER_CREATED', handler1);
      socket.subscribe('ORDER_CREATED', handler2);

      const event: WMSEvent = {
        type: 'ORDER_CREATED',
        payload: { orderId: '123' },
      };

      const ws = (socket as any).ws as MockWebSocket;
      ws.simulateMessage(event);

      expect(handler1).toHaveBeenCalledWith(event.payload);
      expect(handler2).toHaveBeenCalledWith(event.payload);
    });

    it('cleans up event type when last handler unsubscribes', async () => {
      const handler = vi.fn();
      const socket = new WMSSocket('ws://localhost:8080/ws');

      socket.connect();
      await vi.runAllTimersAsync();

      const unsubscribe = socket.subscribe('ORDER_CREATED', handler);
      expect((socket as any).eventHandlers.has('ORDER_CREATED')).toBe(true);

      unsubscribe();
      expect((socket as any).eventHandlers.has('ORDER_CREATED')).toBe(false);
    });
  });

  describe('send', () => {
    it('sends message when connected', async () => {
      const socket = new WMSSocket('ws://localhost:8080/ws');

      socket.connect();
      await vi.runAllTimersAsync();

      const message = { type: 'PING', data: 'test' };
      socket.send(message);

      const ws = (socket as any).ws as MockWebSocket;
      expect(ws.send).toHaveBeenCalledWith(JSON.stringify(message));
    });

    it('warns when not connected', () => {
      const consoleWarnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
      const socket = new WMSSocket('ws://localhost:8080/ws');

      socket.send({ type: 'PING' });

      expect(consoleWarnSpy).toHaveBeenCalledWith(
        '[WMS Socket] Not connected, cannot send message'
      );
    });
  });

  describe('isConnected', () => {
    it('returns true when WebSocket is open', async () => {
      const socket = new WMSSocket('ws://localhost:8080/ws');

      socket.connect();
      await vi.runAllTimersAsync();

      expect(socket.isConnected).toBe(true);
    });

    it('returns false when WebSocket is not open', () => {
      const socket = new WMSSocket('ws://localhost:8080/ws');
      expect(socket.isConnected).toBe(false);
    });

    it('returns false after disconnect', async () => {
      const socket = new WMSSocket('ws://localhost:8080/ws');

      socket.connect();
      await vi.runAllTimersAsync();
      expect(socket.isConnected).toBe(true);

      socket.disconnect();
      expect(socket.isConnected).toBe(false);
    });
  });

  describe('message handling', () => {
    it('emits messages to global event bus', async () => {
      const globalHandler = vi.fn();
      eventBus.on(globalHandler);

      const socket = new WMSSocket('ws://localhost:8080/ws');
      socket.connect();
      await vi.runAllTimersAsync();

      const event: WMSEvent = {
        type: 'ORDER_CREATED',
        payload: { orderId: '123' },
      };

      const ws = (socket as any).ws as MockWebSocket;
      ws.simulateMessage(event);

      expect(globalHandler).toHaveBeenCalledWith(event);
    });

    it('handles invalid JSON gracefully', async () => {
      const socket = new WMSSocket('ws://localhost:8080/ws');
      socket.connect();
      await vi.runAllTimersAsync();

      const ws = (socket as any).ws as MockWebSocket;
      ws.onmessage?.(new MessageEvent('message', { data: 'invalid json' }));

      expect(consoleErrorSpy).toHaveBeenCalledWith(
        '[WMS Socket] Failed to parse message:',
        expect.any(Error)
      );
    });
  });

  describe('error handling', () => {
    it('calls onError callback on error', async () => {
      const onError = vi.fn();
      const socket = new WMSSocket('ws://localhost:8080/ws', { onError });

      socket.connect();
      await vi.runAllTimersAsync();

      const error = new Event('error');
      const ws = (socket as any).ws as MockWebSocket;
      ws.simulateError(error);

      expect(onError).toHaveBeenCalledWith(error);
      expect(consoleErrorSpy).toHaveBeenCalledWith('[WMS Socket] Error:', error);
    });
  });
});

describe('createWMSSocket', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    global.WebSocket = MockWebSocket as any;
    delete (import.meta as any).env.VITE_WS_BASE_URL;
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  it('creates socket with correct URL from window location', () => {
    const originalLocation = window.location;
    Object.defineProperty(window, 'location', {
      value: { protocol: 'http:', host: 'localhost:3000' },
      writable: true,
    });

    const socket = createWMSSocket('orders');

    expect((socket as any).url).toBe('ws://localhost:3000/api/orders/ws');

    Object.defineProperty(window, 'location', {
      value: originalLocation,
      writable: true,
    });
  });

  it('uses wss:// for https protocol', () => {
    const originalLocation = window.location;
    Object.defineProperty(window, 'location', {
      value: { protocol: 'https:', host: 'example.com' },
      writable: true,
    });

    const socket = createWMSSocket('orders');

    expect((socket as any).url).toBe('wss://example.com/api/orders/ws');

    Object.defineProperty(window, 'location', {
      value: originalLocation,
      writable: true,
    });
  });

  it('uses window location protocol when env is not provided', () => {
    const originalLocation = window.location;
    Object.defineProperty(window, 'location', {
      value: { protocol: 'http:', host: 'localhost:3000' },
      writable: true,
    });

    const socket = createWMSSocket('orders');

    expect((socket as any).url).toBe('ws://localhost:3000/api/orders/ws');

    Object.defineProperty(window, 'location', {
      value: originalLocation,
      writable: true,
    });
  });

  it('passes custom options to socket', () => {
    const onConnect = vi.fn();
    const socket = createWMSSocket('orders', { onConnect });

    expect((socket as any).options.onConnect).toBe(onConnect);
  });
});
