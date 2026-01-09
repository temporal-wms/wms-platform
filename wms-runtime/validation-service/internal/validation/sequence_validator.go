package validation

import (
	"fmt"
)

// SequenceValidator validates event sequences for different flow types
type SequenceValidator struct {
	expectedSequences map[string][]string
}

// NewSequenceValidator creates a new sequence validator
func NewSequenceValidator() *SequenceValidator {
	sv := &SequenceValidator{
		expectedSequences: make(map[string][]string),
	}

	// Load default expected sequences
	sv.loadDefaultSequences()

	return sv
}

// loadDefaultSequences loads expected event sequences for standard flows
func (sv *SequenceValidator) loadDefaultSequences() {
	// Standard order fulfillment flow
	sv.expectedSequences["standard_flow"] = []string{
		"wms.order.received",
		"wms.order.validated",
		"wms.wave.released",
		"wms.picking.task-created",
		"wms.picking.task-completed",
		"wms.packing.task-created",
		"wms.packing.task-completed",
		"wms.shipping.shipment-created",
		"wms.order.shipped",
	}

	// Multi-item flow with consolidation
	sv.expectedSequences["multi_item_flow"] = []string{
		"wms.order.received",
		"wms.order.validated",
		"wms.wave.released",
		"wms.picking.task-created",
		"wms.picking.task-completed",
		"wms.consolidation.consolidation-created",
		"wms.consolidation.consolidation-completed",
		"wms.packing.task-created",
		"wms.packing.task-completed",
		"wms.shipping.shipment-created",
		"wms.order.shipped",
	}

	// Pick-wall-pack flow
	sv.expectedSequences["pick_wall_pack_flow"] = []string{
		"wms.order.received",
		"wms.order.validated",
		"wms.wave.released",
		"wms.picking.task-created",
		"wms.picking.task-completed",
		"wms.walling.task-created",
		"wms.walling.task-completed",
		"wms.packing.task-created",
		"wms.packing.task-completed",
		"wms.shipping.shipment-created",
		"wms.order.shipped",
	}

	// Cancellation flow
	sv.expectedSequences["cancellation_flow"] = []string{
		"wms.order.received",
		"wms.order.validated",
		"wms.order.cancellation-requested",
		"wms.inventory.reservation-released",
		"wms.order.cancelled",
	}

	// Gift wrap flow
	sv.expectedSequences["gift_wrap_flow"] = []string{
		"wms.order.received",
		"wms.order.validated",
		"wms.wave.released",
		"wms.picking.task-created",
		"wms.picking.task-completed",
		"wms.giftwrap.task-created",
		"wms.giftwrap.task-completed",
		"wms.packing.task-created",
		"wms.packing.task-completed",
		"wms.shipping.shipment-created",
		"wms.order.shipped",
	}
}

// ValidateSequence validates that events follow the expected sequence for a flow type
func (sv *SequenceValidator) ValidateSequence(flowType string, actualEvents []string) (*SequenceValidationResult, error) {
	expectedSequence, exists := sv.expectedSequences[flowType]
	if !exists {
		return nil, fmt.Errorf("unknown flow type: %s", flowType)
	}

	result := &SequenceValidationResult{
		FlowType:         flowType,
		ExpectedSequence: expectedSequence,
		ActualEvents:     actualEvents,
		IsValid:          true,
		MissingEvents:    []string{},
		UnexpectedEvents: []string{},
		OutOfOrderEvents: []string{},
	}

	// Check for missing events
	actualEventSet := make(map[string]bool)
	for _, event := range actualEvents {
		actualEventSet[event] = true
	}

	for _, expected := range expectedSequence {
		if !actualEventSet[expected] {
			result.MissingEvents = append(result.MissingEvents, expected)
			result.IsValid = false
		}
	}

	// Check for unexpected events
	expectedEventSet := make(map[string]bool)
	for _, expected := range expectedSequence {
		expectedEventSet[expected] = true
	}

	for _, actual := range actualEvents {
		if !expectedEventSet[actual] {
			result.UnexpectedEvents = append(result.UnexpectedEvents, actual)
		}
	}

	// Check event ordering
	expectedIndex := 0
	for _, actual := range actualEvents {
		if expectedIndex < len(expectedSequence) && actual == expectedSequence[expectedIndex] {
			expectedIndex++
		} else if expectedEventSet[actual] {
			// Event is expected but out of order
			result.OutOfOrderEvents = append(result.OutOfOrderEvents, actual)
			result.IsValid = false
		}
	}

	return result, nil
}

// AddExpectedSequence adds a custom expected sequence
func (sv *SequenceValidator) AddExpectedSequence(flowType string, sequence []string) {
	sv.expectedSequences[flowType] = sequence
}

// GetExpectedSequence returns the expected sequence for a flow type
func (sv *SequenceValidator) GetExpectedSequence(flowType string) ([]string, error) {
	sequence, exists := sv.expectedSequences[flowType]
	if !exists {
		return nil, fmt.Errorf("unknown flow type: %s", flowType)
	}
	return sequence, nil
}

// SequenceValidationResult represents the result of sequence validation
type SequenceValidationResult struct {
	FlowType         string   `json:"flowType"`
	ExpectedSequence []string `json:"expectedSequence"`
	ActualEvents     []string `json:"actualEvents"`
	IsValid          bool     `json:"isValid"`
	MissingEvents    []string `json:"missingEvents"`
	UnexpectedEvents []string `json:"unexpectedEvents,omitempty"`
	OutOfOrderEvents []string `json:"outOfOrderEvents,omitempty"`
}
