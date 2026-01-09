package kafka

import (
	"time"
)

// Config holds Kafka configuration
type Config struct {
	Brokers       []string
	ConsumerGroup string
	ClientID      string

	// Producer settings
	BatchSize    int
	BatchTimeout time.Duration
	RequiredAcks int // 0: no ack, 1: leader ack, -1: all replicas ack

	// Consumer settings
	MinBytes      int
	MaxBytes      int
	MaxWait       time.Duration
	CommitTimeout time.Duration

	// TLS settings
	TLSEnabled bool
	TLSCert    string
	TLSKey     string
	TLSCA      string

	// SASL settings
	SASLEnabled   bool
	SASLMechanism string
	SASLUsername  string
	SASLPassword  string
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Brokers:       []string{"localhost:9092"},
		ConsumerGroup: "wms-default-group",
		ClientID:      "wms-client",

		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: -1, // All replicas

		MinBytes:      1,
		MaxBytes:      10e6, // 10MB
		MaxWait:       500 * time.Millisecond,
		CommitTimeout: 5 * time.Second,

		TLSEnabled:  false,
		SASLEnabled: false,
	}
}

// Topics contains all WMS Kafka topic names
var Topics = struct {
	// Inbound topics
	OrdersInbound string

	// Domain event topics
	OrdersEvents        string
	WavesEvents         string
	RoutingEvents       string
	PickingEvents       string
	ConsolidationEvents string
	PackingEvents       string
	ShippingEvents      string
	InventoryEvents     string
	LaborEvents         string
	FacilityEvents      string
	WESEvents           string

	// Amazon-aligned process topics
	ReceivingEvents  string
	StowEvents       string
	SLAMEvents       string
	SortationEvents  string

	// Multi-tenant/3PL topics
	SellerEvents  string
	BillingEvents string
	ChannelEvents string

	// Outbound topics
	ShipmentsOutbound string
}{
	OrdersInbound: "wms.orders.inbound",

	OrdersEvents:        "wms.orders.events",
	WavesEvents:         "wms.waves.events",
	RoutingEvents:       "wms.routing.events",
	PickingEvents:       "wms.picking.events",
	ConsolidationEvents: "wms.consolidation.events",
	PackingEvents:       "wms.packing.events",
	ShippingEvents:      "wms.shipping.events",
	InventoryEvents:     "wms.inventory.events",
	LaborEvents:         "wms.labor.events",
	FacilityEvents:      "wms.facility.events",
	WESEvents:           "wms.wes.events",

	// Amazon-aligned process topics
	ReceivingEvents:  "wms.receiving.events",
	StowEvents:       "wms.stow.events",
	SLAMEvents:       "wms.slam.events",
	SortationEvents:  "wms.sortation.events",

	// Multi-tenant/3PL topics
	SellerEvents:  "wms.seller.events",
	BillingEvents: "wms.billing.events",
	ChannelEvents: "wms.channel.events",

	ShipmentsOutbound: "wms.shipments.outbound",
}

// TopicConfig holds configuration for a Kafka topic
type TopicConfig struct {
	Name              string
	Partitions        int
	ReplicationFactor int
	RetentionMs       int64
}

// DefaultTopicConfigs returns default configurations for WMS topics
func DefaultTopicConfigs() []TopicConfig {
	return []TopicConfig{
		{Name: Topics.OrdersInbound, Partitions: 12, ReplicationFactor: 3, RetentionMs: 7 * 24 * 60 * 60 * 1000},   // 7 days
		{Name: Topics.OrdersEvents, Partitions: 6, ReplicationFactor: 3, RetentionMs: 7 * 24 * 60 * 60 * 1000},
		{Name: Topics.WavesEvents, Partitions: 6, ReplicationFactor: 3, RetentionMs: 7 * 24 * 60 * 60 * 1000},
		{Name: Topics.RoutingEvents, Partitions: 6, ReplicationFactor: 3, RetentionMs: 7 * 24 * 60 * 60 * 1000},
		{Name: Topics.PickingEvents, Partitions: 12, ReplicationFactor: 3, RetentionMs: 7 * 24 * 60 * 60 * 1000},
		{Name: Topics.ConsolidationEvents, Partitions: 6, ReplicationFactor: 3, RetentionMs: 7 * 24 * 60 * 60 * 1000},
		{Name: Topics.PackingEvents, Partitions: 6, ReplicationFactor: 3, RetentionMs: 7 * 24 * 60 * 60 * 1000},
		{Name: Topics.ShippingEvents, Partitions: 6, ReplicationFactor: 3, RetentionMs: 7 * 24 * 60 * 60 * 1000},
		{Name: Topics.InventoryEvents, Partitions: 6, ReplicationFactor: 3, RetentionMs: 7 * 24 * 60 * 60 * 1000},
		{Name: Topics.LaborEvents, Partitions: 6, ReplicationFactor: 3, RetentionMs: 7 * 24 * 60 * 60 * 1000},
		{Name: Topics.WESEvents, Partitions: 6, ReplicationFactor: 3, RetentionMs: 7 * 24 * 60 * 60 * 1000},
		// Amazon-aligned process topics
		{Name: Topics.ReceivingEvents, Partitions: 6, ReplicationFactor: 3, RetentionMs: 7 * 24 * 60 * 60 * 1000},
		{Name: Topics.StowEvents, Partitions: 6, ReplicationFactor: 3, RetentionMs: 7 * 24 * 60 * 60 * 1000},
		{Name: Topics.SLAMEvents, Partitions: 6, ReplicationFactor: 3, RetentionMs: 7 * 24 * 60 * 60 * 1000},
		{Name: Topics.SortationEvents, Partitions: 6, ReplicationFactor: 3, RetentionMs: 7 * 24 * 60 * 60 * 1000},
		// Multi-tenant/3PL topics
		{Name: Topics.SellerEvents, Partitions: 6, ReplicationFactor: 3, RetentionMs: 30 * 24 * 60 * 60 * 1000},  // 30 days
		{Name: Topics.BillingEvents, Partitions: 6, ReplicationFactor: 3, RetentionMs: 90 * 24 * 60 * 60 * 1000}, // 90 days for audit
		{Name: Topics.ChannelEvents, Partitions: 6, ReplicationFactor: 3, RetentionMs: 7 * 24 * 60 * 60 * 1000},
		{Name: Topics.ShipmentsOutbound, Partitions: 6, ReplicationFactor: 3, RetentionMs: 30 * 24 * 60 * 60 * 1000}, // 30 days
	}
}
