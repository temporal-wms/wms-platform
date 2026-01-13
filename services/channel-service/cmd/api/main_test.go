package main

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wms-platform/shared/pkg/metrics"
)

func TestGetEnv(t *testing.T) {
	key := "CHANNEL_SERVICE_TEST_ENV"
	require.NoError(t, os.Setenv(key, "value"))
	defer os.Unsetenv(key)

	require.Equal(t, "value", getEnv(key, "default"))
	require.Equal(t, "fallback", getEnv("CHANNEL_SERVICE_MISSING", "fallback"))
}

func TestLoadConfig(t *testing.T) {
	require.NoError(t, os.Setenv("SERVER_ADDR", ":9999"))
	require.NoError(t, os.Setenv("MONGODB_URI", "mongodb://example:27017"))
	require.NoError(t, os.Setenv("MONGODB_DATABASE", "channel_test"))
	require.NoError(t, os.Setenv("KAFKA_BROKERS", "broker1:9092,broker2:9092"))
	defer os.Unsetenv("SERVER_ADDR")
	defer os.Unsetenv("MONGODB_URI")
	defer os.Unsetenv("MONGODB_DATABASE")
	defer os.Unsetenv("KAFKA_BROKERS")

	cfg := loadConfig()
	require.Equal(t, ":9999", cfg.ServerAddr)
	require.Equal(t, "mongodb://example:27017", cfg.MongoDB.URI)
	require.Equal(t, "channel_test", cfg.MongoDB.Database)
	require.Equal(t, []string{"broker1:9092", "broker2:9092"}, cfg.Kafka.Brokers)
	require.Equal(t, serviceName, cfg.Kafka.ClientID)
}

func TestChannelMetrics(t *testing.T) {
	m := metrics.New(metrics.DefaultConfig(serviceName))
	cm := NewChannelMetrics(m)

	cm.RecordSyncOperation("ch-1", "orders", "success", time.Millisecond)
	cm.RecordSyncOperation("ch-1", "orders", "error", time.Millisecond)
	cm.RecordOrdersImported("ch-1", 2)
	cm.RecordAPILatency("ch-1", "get_orders", "success", time.Millisecond)
	cm.RecordWebhookReceived("ch-1", "orders/create", "success")
	cm.RecordWebhookReceived("ch-1", "orders/create", "error")
}
