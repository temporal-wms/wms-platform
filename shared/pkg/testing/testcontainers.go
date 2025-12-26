package testing

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBContainer wraps a testcontainers MongoDB instance
type MongoDBContainer struct {
	Container *mongodb.MongoDBContainer
	URI       string
}

// NewMongoDBContainer creates a new MongoDB testcontainer
func NewMongoDBContainer(ctx context.Context) (*MongoDBContainer, error) {
	mongoContainer, err := mongodb.Run(ctx,
		"mongo:6",
		mongodb.WithUsername("test"),
		mongodb.WithPassword("test"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start mongodb container: %w", err)
	}

	uri, err := mongoContainer.ConnectionString(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	return &MongoDBContainer{
		Container: mongoContainer,
		URI:       uri,
	}, nil
}

// Close terminates the MongoDB container
func (m *MongoDBContainer) Close(ctx context.Context) error {
	if m.Container != nil {
		return m.Container.Terminate(ctx)
	}
	return nil
}

// GetClient creates a MongoDB client connected to the test container
func (m *MongoDBContainer) GetClient(ctx context.Context) (*mongo.Client, error) {
	clientOptions := options.Client().ApplyURI(m.URI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	// Ping to verify connection
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	return client, nil
}

// KafkaContainer wraps a testcontainers Kafka instance
type KafkaContainer struct {
	Container testcontainers.Container
	Brokers   []string
}

// NewKafkaContainer creates a new Kafka testcontainer
func NewKafkaContainer(ctx context.Context) (*KafkaContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "confluentinc/cp-kafka:7.5.0",
		ExposedPorts: []string{"9093/tcp"},
		Env: map[string]string{
			"KAFKA_BROKER_ID":                        "1",
			"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP":   "PLAINTEXT:PLAINTEXT,PLAINTEXT_HOST:PLAINTEXT",
			"KAFKA_ADVERTISED_LISTENERS":             "PLAINTEXT://kafka:29092,PLAINTEXT_HOST://localhost:9093",
			"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR": "1",
			"KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS": "0",
			"KAFKA_TRANSACTION_STATE_LOG_MIN_ISR":    "1",
			"KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR": "1",
			"KAFKA_ZOOKEEPER_CONNECT":                         "zookeeper:2181",
		},
		WaitingFor: wait.ForLog("started (kafka.server.KafkaServer)").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start kafka container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, "9093")
	if err != nil {
		return nil, fmt.Errorf("failed to get mapped port: %w", err)
	}

	brokers := []string{fmt.Sprintf("%s:%s", host, port.Port())}

	return &KafkaContainer{
		Container: container,
		Brokers:   brokers,
	}, nil
}

// Close terminates the Kafka container
func (k *KafkaContainer) Close(ctx context.Context) error {
	if k.Container != nil {
		return k.Container.Terminate(ctx)
	}
	return nil
}

// TestEnvironment holds all test containers
type TestEnvironment struct {
	MongoDB *MongoDBContainer
	Kafka   *KafkaContainer
}

// NewTestEnvironment creates a complete test environment with all containers
func NewTestEnvironment(ctx context.Context, includeKafka bool) (*TestEnvironment, error) {
	env := &TestEnvironment{}

	// Start MongoDB
	mongoContainer, err := NewMongoDBContainer(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create mongodb container: %w", err)
	}
	env.MongoDB = mongoContainer

	// Optionally start Kafka
	if includeKafka {
		kafkaContainer, err := NewKafkaContainer(ctx)
		if err != nil {
			mongoContainer.Close(ctx)
			return nil, fmt.Errorf("failed to create kafka container: %w", err)
		}
		env.Kafka = kafkaContainer
	}

	return env, nil
}

// Close terminates all containers in the test environment
func (e *TestEnvironment) Close(ctx context.Context) error {
	var errs []error

	if e.MongoDB != nil {
		if err := e.MongoDB.Close(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if e.Kafka != nil {
		if err := e.Kafka.Close(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing test environment: %v", errs)
	}

	return nil
}
