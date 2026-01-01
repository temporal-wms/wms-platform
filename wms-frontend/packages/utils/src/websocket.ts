import type { WMSEvent } from '@wms/types';
import { eventBus } from './eventBus';

export interface WMSSocketOptions {
  reconnectAttempts?: number;
  reconnectInterval?: number;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: Event) => void;
}

export class WMSSocket {
  private ws: WebSocket | null = null;
  private url: string;
  private options: Required<WMSSocketOptions>;
  private reconnectCount = 0;
  private isManualClose = false;
  private eventHandlers: Map<string, Set<(data: unknown) => void>> = new Map();

  constructor(url: string, options: WMSSocketOptions = {}) {
    this.url = url;
    this.options = {
      reconnectAttempts: options.reconnectAttempts ?? 5,
      reconnectInterval: options.reconnectInterval ?? 3000,
      onConnect: options.onConnect ?? (() => {}),
      onDisconnect: options.onDisconnect ?? (() => {}),
      onError: options.onError ?? (() => {}),
    };
  }

  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN) return;

    try {
      this.ws = new WebSocket(this.url);

      this.ws.onopen = () => {
        console.log('[WMS Socket] Connected to', this.url);
        this.reconnectCount = 0;
        this.options.onConnect();
      };

      this.ws.onclose = () => {
        console.log('[WMS Socket] Disconnected');
        this.options.onDisconnect();

        if (!this.isManualClose && this.reconnectCount < this.options.reconnectAttempts) {
          this.reconnectCount++;
          console.log(`[WMS Socket] Reconnecting... attempt ${this.reconnectCount}`);
          setTimeout(() => this.connect(), this.options.reconnectInterval);
        }
      };

      this.ws.onerror = (error) => {
        console.error('[WMS Socket] Error:', error);
        this.options.onError(error);
      };

      this.ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data) as WMSEvent;

          // Emit to global event bus
          eventBus.emit(message);

          // Emit to local handlers
          const handlers = this.eventHandlers.get(message.type);
          if (handlers) {
            handlers.forEach((handler) => handler(message.payload));
          }
        } catch (error) {
          console.error('[WMS Socket] Failed to parse message:', error);
        }
      };
    } catch (error) {
      console.error('[WMS Socket] Failed to connect:', error);
    }
  }

  disconnect(): void {
    this.isManualClose = true;
    this.ws?.close();
    this.ws = null;
  }

  subscribe<T = unknown>(eventType: string, handler: (data: T) => void): () => void {
    if (!this.eventHandlers.has(eventType)) {
      this.eventHandlers.set(eventType, new Set());
    }

    const handlers = this.eventHandlers.get(eventType)!;
    handlers.add(handler as (data: unknown) => void);

    return () => {
      handlers.delete(handler as (data: unknown) => void);
      if (handlers.size === 0) {
        this.eventHandlers.delete(eventType);
      }
    };
  }

  send(message: unknown): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    } else {
      console.warn('[WMS Socket] Not connected, cannot send message');
    }
  }

  get isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }
}

// Factory function for creating service-specific sockets
export function createWMSSocket(
  serviceName: string,
  options?: WMSSocketOptions
): WMSSocket {
  const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const baseUrl = import.meta.env.VITE_WS_BASE_URL || `${wsProtocol}//${window.location.host}`;
  const url = `${baseUrl}/api/${serviceName}/ws`;

  return new WMSSocket(url, options);
}

// Pre-configured sockets for each service
export const orderSocket = () => createWMSSocket('orders');
export const waveSocket = () => createWMSSocket('waves');
export const pickingSocket = () => createWMSSocket('picking');
export const packingSocket = () => createWMSSocket('packing');
export const shippingSocket = () => createWMSSocket('shipping');
export const inventorySocket = () => createWMSSocket('inventory');
export const laborSocket = () => createWMSSocket('labor');
