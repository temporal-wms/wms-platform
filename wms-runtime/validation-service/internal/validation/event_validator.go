package validation

import (
	"fmt"
	"sync"
)

// EventValidator validates events against AsyncAPI schemas
type EventValidator struct {
	mu      sync.RWMutex
	schemas map[string]interface{} // eventType -> schema
}

// NewEventValidator creates a new event validator
func NewEventValidator() *EventValidator {
	return &EventValidator{
		schemas: make(map[string]interface{}),
	}
}

// LoadSchema loads an AsyncAPI schema for validation
func (v *EventValidator) LoadSchema(eventType string, schema interface{}) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.schemas[eventType] = schema
}

// Validate validates an event against its schema
func (v *EventValidator) Validate(eventType string, event map[string]interface{}) error {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// For now, perform basic validation
	// Full AsyncAPI schema validation will be implemented in Phase 2

	// Check required CloudEvents fields
	if _, ok := event["id"]; !ok {
		return fmt.Errorf("missing required field: id")
	}

	if _, ok := event["type"]; !ok {
		return fmt.Errorf("missing required field: type")
	}

	if _, ok := event["source"]; !ok {
		return fmt.Errorf("missing required field: source")
	}

	if _, ok := event["specversion"]; !ok {
		return fmt.Errorf("missing required field: specversion")
	}

	// Check data field exists
	if _, ok := event["data"]; !ok {
		return fmt.Errorf("missing required field: data")
	}

	return nil
}

// ValidateSequence validates that events appear in expected order
func (v *EventValidator) ValidateSequence(events []string, expectedSequence []string) error {
	eventIndex := 0

	for _, expected := range expectedSequence {
		found := false

		for eventIndex < len(events) {
			if events[eventIndex] == expected {
				found = true
				eventIndex++
				break
			}
			eventIndex++
		}

		if !found {
			return fmt.Errorf("expected event '%s' not found in sequence", expected)
		}
	}

	return nil
}
