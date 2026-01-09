package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Config holds MongoDB connection configuration
type Config struct {
	URI            string
	Database       string
	ConnectTimeout time.Duration
	MaxPoolSize    uint64
	MinPoolSize    uint64

	// Authentication
	Username string
	Password string
	AuthDB   string

	// TLS
	TLSEnabled bool
	TLSCAFile  string

	// Replica Set
	ReplicaSet string
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		URI:            "mongodb://localhost:27017",
		Database:       "wms",
		ConnectTimeout: 10 * time.Second,
		MaxPoolSize:    100,
		MinPoolSize:    10,
		TLSEnabled:     false,
	}
}

// Client wraps the MongoDB client with WMS-specific functionality
type Client struct {
	client   *mongo.Client
	database *mongo.Database
	config   *Config
}

// NewClient creates a new MongoDB client
func NewClient(ctx context.Context, config *Config) (*Client, error) {
	clientOpts := options.Client().
		ApplyURI(config.URI).
		SetConnectTimeout(config.ConnectTimeout).
		SetMaxPoolSize(config.MaxPoolSize).
		SetMinPoolSize(config.MinPoolSize)

	if config.Username != "" && config.Password != "" {
		credential := options.Credential{
			Username:   config.Username,
			Password:   config.Password,
			AuthSource: config.AuthDB,
		}
		clientOpts.SetAuth(credential)
	}

	if config.ReplicaSet != "" {
		clientOpts.SetReplicaSet(config.ReplicaSet)
	}

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping to verify connection
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return &Client{
		client:   client,
		database: client.Database(config.Database),
		config:   config,
	}, nil
}

// Database returns the database handle
func (c *Client) Database() *mongo.Database {
	return c.database
}

// Collection returns a collection handle
func (c *Client) Collection(name string) *mongo.Collection {
	return c.database.Collection(name)
}

// Client returns the underlying MongoDB client
func (c *Client) Client() *mongo.Client {
	return c.client
}

// Close disconnects the client
func (c *Client) Close(ctx context.Context) error {
	return c.client.Disconnect(ctx)
}

// HealthCheck performs a health check on the MongoDB connection
func (c *Client) HealthCheck(ctx context.Context) error {
	return c.client.Ping(ctx, readpref.Primary())
}

// WithTransaction executes a function within a transaction
func (c *Client) WithTransaction(ctx context.Context, fn func(sessCtx mongo.SessionContext) error) error {
	session, err := c.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		return nil, fn(sessCtx)
	})

	return err
}
