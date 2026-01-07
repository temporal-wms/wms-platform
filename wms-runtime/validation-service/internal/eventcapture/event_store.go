package eventcapture

import (
	"sync"
	"time"
)

// CapturedEvent represents an event captured from Kafka
type CapturedEvent struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Source        string                 `json:"source"`
	OrderID       string                 `json:"orderId"`
	Topic         string                 `json:"topic"`
	Partition     int32                  `json:"partition"`
	Offset        int64                  `json:"offset"`
	Timestamp     time.Time              `json:"timestamp"`
	CapturedAt    time.Time              `json:"capturedAt"`
	Data          map[string]interface{} `json:"data"`
	CloudEventsID string                 `json:"cloudEventsId,omitempty"`
}

// EventStore stores captured events with TTL-based expiration
type EventStore struct {
	mu        sync.RWMutex
	events    map[string][]*CapturedEvent // orderId -> events
	tracking  map[string]bool             // orderId -> tracking enabled
	ttl       time.Duration
	createdAt map[string]time.Time // orderId -> tracking start time
}

// NewEventStore creates a new event store with TTL
func NewEventStore(ttl time.Duration) *EventStore {
	store := &EventStore{
		events:    make(map[string][]*CapturedEvent),
		tracking:  make(map[string]bool),
		createdAt: make(map[string]time.Time),
		ttl:       ttl,
	}

	// Start cleanup goroutine
	go store.cleanupExpired()

	return store
}

// StartTracking enables event tracking for an order
func (s *EventStore) StartTracking(orderID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tracking[orderID] = true
	s.createdAt[orderID] = time.Now()
	s.events[orderID] = make([]*CapturedEvent, 0)
}

// IsTracking checks if an order is being tracked
func (s *EventStore) IsTracking(orderID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.tracking[orderID]
}

// StopTracking disables event tracking for an order
func (s *EventStore) StopTracking(orderID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.tracking, orderID)
}

// AddEvent adds an event to the store
func (s *EventStore) AddEvent(event *CapturedEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Only store if tracking is enabled for this order
	if !s.tracking[event.OrderID] {
		return
	}

	if events, exists := s.events[event.OrderID]; exists {
		s.events[event.OrderID] = append(events, event)
	} else {
		s.events[event.OrderID] = []*CapturedEvent{event}
	}
}

// GetEvents retrieves all events for an order
func (s *EventStore) GetEvents(orderID string) []*CapturedEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if events, exists := s.events[orderID]; exists {
		// Return a copy to prevent external modification
		result := make([]*CapturedEvent, len(events))
		copy(result, events)
		return result
	}

	return []*CapturedEvent{}
}

// GetEventsByType retrieves events of a specific type for an order
func (s *EventStore) GetEventsByType(orderID string, eventType string) []*CapturedEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*CapturedEvent, 0)

	if events, exists := s.events[orderID]; exists {
		for _, event := range events {
			if event.Type == eventType {
				result = append(result, event)
			}
		}
	}

	return result
}

// GetEventCount returns the number of events for an order
func (s *EventStore) GetEventCount(orderID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if events, exists := s.events[orderID]; exists {
		return len(events)
	}

	return 0
}

// Clear removes all events for an order
func (s *EventStore) Clear(orderID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.events, orderID)
	delete(s.tracking, orderID)
	delete(s.createdAt, orderID)
}

// GetAllOrderIDs returns all tracked order IDs
func (s *EventStore) GetAllOrderIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.tracking))
	for id := range s.tracking {
		ids = append(ids, id)
	}

	return ids
}

// GetStats returns statistics about the event store
func (s *EventStore) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalEvents := 0
	for _, events := range s.events {
		totalEvents += len(events)
	}

	return map[string]interface{}{
		"trackedOrders": len(s.tracking),
		"totalEvents":   totalEvents,
		"ttlMinutes":    int(s.ttl.Minutes()),
	}
}

// cleanupExpired removes expired tracking data
func (s *EventStore) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()

		for orderID, createdAt := range s.createdAt {
			if now.Sub(createdAt) > s.ttl {
				delete(s.events, orderID)
				delete(s.tracking, orderID)
				delete(s.createdAt, orderID)
			}
		}

		s.mu.Unlock()
	}
}
