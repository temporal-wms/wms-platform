import type { WMSEvent } from '@wms/types';

type EventHandler = (event: WMSEvent) => void;

const WMS_EVENT_KEY = 'wms:event';

class EventBus {
  private handlers: Set<EventHandler> = new Set();

  emit(event: WMSEvent): void {
    window.dispatchEvent(
      new CustomEvent(WMS_EVENT_KEY, { detail: event })
    );
  }

  on(handler: EventHandler): () => void {
    const listener = (e: Event) => {
      const customEvent = e as CustomEvent<WMSEvent>;
      handler(customEvent.detail);
    };

    window.addEventListener(WMS_EVENT_KEY, listener);
    this.handlers.add(handler);

    return () => {
      window.removeEventListener(WMS_EVENT_KEY, listener);
      this.handlers.delete(handler);
    };
  }

  once(eventType: WMSEvent['type'], handler: EventHandler): () => void {
    const wrappedHandler: EventHandler = (event) => {
      if (event.type === eventType) {
        handler(event);
        unsubscribe();
      }
    };

    const unsubscribe = this.on(wrappedHandler);
    return unsubscribe;
  }

  filter(eventType: WMSEvent['type'], handler: EventHandler): () => void {
    return this.on((event) => {
      if (event.type === eventType) {
        handler(event);
      }
    });
  }

  clear(): void {
    this.handlers.clear();
  }
}

export const eventBus = new EventBus();

// React hook for event bus
export function useEventBus(
  eventType: WMSEvent['type'] | null,
  handler: EventHandler
): void {
  if (typeof window === 'undefined') return;

  const unsubscribe = eventType
    ? eventBus.filter(eventType, handler)
    : eventBus.on(handler);

  // Note: This should be wrapped in useEffect in the consuming component
  return unsubscribe as unknown as void;
}
