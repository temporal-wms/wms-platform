package asyncapi

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

// EventValidator validates CloudEvents against AsyncAPI schemas.
type EventValidator struct {
	schemas    map[string]*jsonschema.Schema
	rawSchemas map[string]interface{}
	compiler   *jsonschema.Compiler
}

// CloudEvent represents the CloudEvents specification structure.
type CloudEvent struct {
	SpecVersion     string                 `json:"specversion"`
	Type            string                 `json:"type"`
	Source          string                 `json:"source"`
	Subject         string                 `json:"subject,omitempty"`
	ID              string                 `json:"id"`
	Time            string                 `json:"time,omitempty"`
	DataContentType string                 `json:"datacontenttype,omitempty"`
	Data            interface{}            `json:"data,omitempty"`
	Extensions      map[string]interface{} `json:"-"`
}

// AsyncAPISpec represents the relevant parts of an AsyncAPI specification.
type AsyncAPISpec struct {
	AsyncAPI   string                      `yaml:"asyncapi"`
	Info       AsyncAPIInfo                `yaml:"info"`
	Channels   map[string]AsyncAPIChannel  `yaml:"channels"`
	Components AsyncAPIComponents          `yaml:"components"`
}

// AsyncAPIInfo contains AsyncAPI info section.
type AsyncAPIInfo struct {
	Title   string `yaml:"title"`
	Version string `yaml:"version"`
}

// AsyncAPIChannel represents a channel in AsyncAPI.
type AsyncAPIChannel struct {
	Address  string                 `yaml:"address"`
	Messages map[string]interface{} `yaml:"messages"`
}

// AsyncAPIComponents contains reusable components.
type AsyncAPIComponents struct {
	Schemas  map[string]interface{} `yaml:"schemas"`
	Messages map[string]interface{} `yaml:"messages"`
}

// NewEventValidator creates a new event validator from an AsyncAPI specification file.
func NewEventValidator(asyncAPIPath string) (*EventValidator, error) {
	data, err := os.ReadFile(asyncAPIPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read AsyncAPI spec: %w", err)
	}

	return NewEventValidatorFromBytes(data)
}

// NewEventValidatorFromBytes creates a new event validator from AsyncAPI specification bytes.
func NewEventValidatorFromBytes(specBytes []byte) (*EventValidator, error) {
	var spec AsyncAPISpec
	if err := yaml.Unmarshal(specBytes, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse AsyncAPI spec: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	schemas := make(map[string]*jsonschema.Schema)
	rawSchemas := make(map[string]interface{})

	// Extract schemas from components
	for schemaName, schema := range spec.Components.Schemas {
		schemaMap, ok := schema.(map[string]interface{})
		if !ok {
			continue
		}

		// Determine event type from schema name
		eventType := deriveEventTypeFromSchemaName(schemaName)
		if eventType == "" {
			continue
		}

		// Convert schema to JSON for compilation
		schemaJSON, err := json.Marshal(schemaMap)
		if err != nil {
			continue
		}

		// Add schema to compiler
		schemaURI := fmt.Sprintf("asyncapi://schemas/%s", schemaName)
		if err := compiler.AddResource(schemaURI, strings.NewReader(string(schemaJSON))); err != nil {
			continue
		}

		// Compile schema
		compiled, err := compiler.Compile(schemaURI)
		if err != nil {
			continue
		}

		schemas[eventType] = compiled
		rawSchemas[eventType] = schemaMap
	}

	return &EventValidator{
		schemas:    schemas,
		rawSchemas: rawSchemas,
		compiler:   compiler,
	}, nil
}

// ValidateEvent validates a CloudEvent against its schema.
func (v *EventValidator) ValidateEvent(event CloudEvent) error {
	if event.Type == "" {
		return fmt.Errorf("event type is required")
	}

	schema, ok := v.schemas[event.Type]
	if !ok {
		return fmt.Errorf("no schema found for event type: %s", event.Type)
	}

	// Validate the data payload
	if event.Data == nil {
		return fmt.Errorf("event data is required")
	}

	// Convert data to JSON and back to ensure proper interface{} types
	dataJSON, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	var data interface{}
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		return fmt.Errorf("failed to unmarshal event data: %w", err)
	}

	if err := schema.Validate(data); err != nil {
		return fmt.Errorf("event data validation failed for type %s: %w", event.Type, err)
	}

	return nil
}

// ValidateEventJSON validates a CloudEvent from JSON bytes.
func (v *EventValidator) ValidateEventJSON(eventJSON []byte) error {
	var event CloudEvent
	if err := json.Unmarshal(eventJSON, &event); err != nil {
		return fmt.Errorf("failed to parse CloudEvent: %w", err)
	}
	return v.ValidateEvent(event)
}

// GetSupportedEventTypes returns all event types that have registered schemas.
func (v *EventValidator) GetSupportedEventTypes() []string {
	types := make([]string, 0, len(v.schemas))
	for eventType := range v.schemas {
		types = append(types, eventType)
	}
	return types
}

// HasSchema checks if a schema exists for the given event type.
func (v *EventValidator) HasSchema(eventType string) bool {
	_, ok := v.schemas[eventType]
	return ok
}

// GetSchema returns the raw schema for a given event type.
func (v *EventValidator) GetSchema(eventType string) (interface{}, bool) {
	schema, ok := v.rawSchemas[eventType]
	return schema, ok
}

// deriveEventTypeFromSchemaName converts schema names to event types.
// Examples:
//   - OrderReceivedData -> wms.order.received
//   - WaveCreatedData -> wms.wave.created
//   - ItemPickedData -> wms.picking.item-picked
func deriveEventTypeFromSchemaName(schemaName string) string {
	// Remove "Data" or "Event" suffix
	name := strings.TrimSuffix(schemaName, "Data")
	name = strings.TrimSuffix(name, "Event")

	// Common mappings from schema names to event types
	mappings := map[string]string{
		// Order events
		"OrderReceived":     "wms.order.received",
		"OrderValidated":    "wms.order.validated",
		"OrderWaveAssigned": "wms.order.wave-assigned",
		"OrderShipped":      "wms.order.shipped",
		"OrderCancelled":    "wms.order.cancelled",
		"OrderCompleted":    "wms.order.completed",

		// Wave events
		"WaveCreated":     "wms.wave.created",
		"OrderAddedToWave": "wms.wave.order-added",
		"WaveScheduled":   "wms.wave.scheduled",
		"WaveReleased":    "wms.wave.released",
		"WaveCompleted":   "wms.wave.completed",
		"WaveCancelled":   "wms.wave.cancelled",

		// Routing events
		"RouteCalculated": "wms.routing.route-calculated",
		"RouteStarted":    "wms.routing.route-started",
		"StopCompleted":   "wms.routing.stop-completed",
		"RouteCompleted":  "wms.routing.route-completed",

		// Picking events
		"PickTaskCreated":   "wms.picking.task-created",
		"PickTaskAssigned":  "wms.picking.task-assigned",
		"ItemPicked":        "wms.picking.item-picked",
		"PickException":     "wms.picking.exception",
		"PickTaskCompleted": "wms.picking.task-completed",

		// Consolidation events
		"ConsolidationStarted":   "wms.consolidation.started",
		"ItemConsolidated":       "wms.consolidation.item-consolidated",
		"ConsolidationCompleted": "wms.consolidation.completed",

		// Packing events
		"PackTaskCreated":    "wms.packing.task-created",
		"PackagingSuggested": "wms.packing.packaging-suggested",
		"PackageSealed":      "wms.packing.package-sealed",
		"LabelApplied":       "wms.packing.label-applied",
		"PackTaskCompleted":  "wms.packing.task-completed",

		// Shipping events
		"ShipmentCreated":    "wms.shipping.shipment-created",
		"LabelGenerated":     "wms.shipping.label-generated",
		"ShipmentManifested": "wms.shipping.manifested",
		"ShipConfirmed":      "wms.shipping.confirmed",
		"DeliveryConfirmed":  "wms.shipping.delivery-confirmed",

		// Inventory events
		"InventoryReceived": "wms.inventory.received",
		"InventoryReserved": "wms.inventory.reserved",
		"InventoryPicked":   "wms.inventory.picked",
		"InventoryAdjusted": "wms.inventory.adjusted",
		"LowStockAlert":     "wms.inventory.low-stock-alert",

		// Labor events
		"ShiftStarted":         "wms.labor.shift-started",
		"ShiftEnded":           "wms.labor.shift-ended",
		"TaskAssigned":         "wms.labor.task-assigned",
		"TaskCompleted":        "wms.labor.task-completed",
		"PerformanceRecorded":  "wms.labor.performance-recorded",
	}

	if eventType, ok := mappings[name]; ok {
		return eventType
	}

	return ""
}

// RegisterSchema adds a custom schema for an event type.
func (v *EventValidator) RegisterSchema(eventType string, schemaJSON []byte) error {
	schemaURI := fmt.Sprintf("custom://schemas/%s", eventType)
	if err := v.compiler.AddResource(schemaURI, strings.NewReader(string(schemaJSON))); err != nil {
		return fmt.Errorf("failed to add schema resource: %w", err)
	}

	compiled, err := v.compiler.Compile(schemaURI)
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	v.schemas[eventType] = compiled

	var rawSchema interface{}
	if err := json.Unmarshal(schemaJSON, &rawSchema); err != nil {
		return fmt.Errorf("failed to parse schema JSON: %w", err)
	}
	v.rawSchemas[eventType] = rawSchema

	return nil
}
