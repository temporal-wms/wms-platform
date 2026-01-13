package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) {
	t.Setenv("TEST_ENV", "value")
	assert.Equal(t, "value", getEnv("TEST_ENV", "default"))
	assert.Equal(t, "default", getEnv("MISSING_ENV", "default"))
}

func TestLoadConfig(t *testing.T) {
	t.Setenv("SERVER_ADDR", ":9999")
	t.Setenv("MONGODB_URI", "mongodb://test:27017")
	t.Setenv("MONGODB_DATABASE", "billing_test")
	t.Setenv("KAFKA_BROKERS", "kafka:9092")

	cfg := loadConfig()

	assert.Equal(t, ":9999", cfg.ServerAddr)
	assert.Equal(t, "mongodb://test:27017", cfg.MongoDB.URI)
	assert.Equal(t, "billing_test", cfg.MongoDB.Database)
	assert.Equal(t, "kafka:9092", cfg.Kafka.Brokers[0])
	assert.Equal(t, serviceName, cfg.Kafka.ClientID)
}
