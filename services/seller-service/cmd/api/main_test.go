package main

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	kafkaConfig "github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/mongodb"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		setEnv      func()
		wantServer  string
		wantMongoDB *mongodb.Config
		wantKafka   *kafkaConfig.Config
	}{
		{
			name: "Default configuration",
			setEnv: func() {
				os.Unsetenv("SERVER_ADDR")
				os.Unsetenv("MONGODB_URI")
				os.Unsetenv("MONGODB_DATABASE")
				os.Unsetenv("KAFKA_BROKERS")
			},
			wantServer: ":8010",
			wantMongoDB: &mongodb.Config{
				URI:            "mongodb://localhost:27017",
				Database:       "sellers_db",
				ConnectTimeout: 10 * time.Second,
				MaxPoolSize:    100,
				MinPoolSize:    10,
			},
			wantKafka: &kafkaConfig.Config{
				Brokers:       []string{"localhost:9092"},
				ConsumerGroup: serviceName,
				ClientID:      serviceName,
				BatchSize:     100,
				BatchTimeout:  10 * time.Millisecond,
				RequiredAcks:  -1,
			},
		},
		{
			name: "Custom configuration",
			setEnv: func() {
				os.Setenv("SERVER_ADDR", ":9000")
				os.Setenv("MONGODB_URI", "mongodb://custom:27017")
				os.Setenv("MONGODB_DATABASE", "custom_db")
				os.Setenv("KAFKA_BROKERS", "custom:9092")
			},
			wantServer: ":9000",
			wantMongoDB: &mongodb.Config{
				URI:            "mongodb://custom:27017",
				Database:       "custom_db",
				ConnectTimeout: 10 * time.Second,
				MaxPoolSize:    100,
				MinPoolSize:    10,
			},
			wantKafka: &kafkaConfig.Config{
				Brokers:       []string{"custom:9092"},
				ConsumerGroup: serviceName,
				ClientID:      serviceName,
				BatchSize:     100,
				BatchTimeout:  10 * time.Millisecond,
				RequiredAcks:  -1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setEnv()
			config := loadConfig()

			assert.NotNil(t, config)
			assert.Equal(t, tt.wantServer, config.ServerAddr)
			assert.Equal(t, tt.wantMongoDB.URI, config.MongoDB.URI)
			assert.Equal(t, tt.wantMongoDB.Database, config.MongoDB.Database)
			assert.Equal(t, tt.wantKafka.Brokers, config.Kafka.Brokers)
		})
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		setEnv       func()
		want         string
	}{
		{
			name:         "Environment variable set",
			key:          "TEST_VAR",
			defaultValue: "default",
			setEnv: func() {
				os.Setenv("TEST_VAR", "value")
			},
			want: "value",
		},
		{
			name:         "Environment variable not set",
			key:          "TEST_VAR_NOT_SET",
			defaultValue: "default",
			setEnv: func() {
				os.Unsetenv("TEST_VAR_NOT_SET")
			},
			want: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setEnv()
			result := getEnv(tt.key, tt.defaultValue)
			assert.Equal(t, tt.want, result)
		})
	}
}
