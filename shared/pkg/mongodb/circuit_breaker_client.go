package mongodb

import (
	"context"
	"log/slog"

	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/metrics"
	"github.com/wms-platform/shared/pkg/resilience"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CircuitBreakerClient wraps InstrumentedClient with circuit breaker protection
type CircuitBreakerClient struct {
	client         *InstrumentedClient
	circuitBreaker *resilience.CircuitBreaker
	logger         *logging.Logger
}

// NewCircuitBreakerClient creates a new circuit breaker protected MongoDB client
func NewCircuitBreakerClient(client *InstrumentedClient, logger *logging.Logger) *CircuitBreakerClient {
	// Create circuit breaker config for MongoDB
	config := &resilience.CircuitBreakerConfig{
		Name:                  "mongodb",
		MaxRequests:           5,
		Interval:              60, // 1 minute
		Timeout:               30, // 30 seconds
		FailureThreshold:      5,
		SuccessThreshold:      2,
		FailureRatioThreshold: 0.5,
		MinRequestsToTrip:     10,
	}

	var slogLogger *slog.Logger
	if logger != nil && logger.Logger != nil {
		slogLogger = logger.Logger
	} else {
		slogLogger = slog.Default()
	}

	cb := resilience.NewCircuitBreaker(config, slogLogger)

	return &CircuitBreakerClient{
		client:         client,
		circuitBreaker: cb,
		logger:         logger,
	}
}

// Collection returns a circuit breaker protected collection
func (c *CircuitBreakerClient) Collection(name string) *CircuitBreakerCollection {
	return &CircuitBreakerCollection{
		collection:     c.client.Collection(name),
		circuitBreaker: c.circuitBreaker,
		logger:         c.logger,
	}
}

// Database returns the underlying database handle
func (c *CircuitBreakerClient) Database() *mongo.Database {
	return c.client.Database()
}

// Client returns the underlying MongoDB client
func (c *CircuitBreakerClient) Client() *mongo.Client {
	return c.client.Client()
}

// Close disconnects the client
func (c *CircuitBreakerClient) Close(ctx context.Context) error {
	return c.client.Close(ctx)
}

// HealthCheck performs a health check with circuit breaker protection
func (c *CircuitBreakerClient) HealthCheck(ctx context.Context) error {
	_, err := c.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return nil, c.client.HealthCheck(ctx)
	})
	return err
}

// WithTransaction executes a function within a transaction with circuit breaker protection
func (c *CircuitBreakerClient) WithTransaction(ctx context.Context, fn func(sessCtx mongo.SessionContext) error) error {
	_, err := c.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return nil, c.client.WithTransaction(ctx, fn)
	})
	return err
}

// RawClient returns the underlying InstrumentedClient
func (c *CircuitBreakerClient) RawClient() *InstrumentedClient {
	return c.client
}

// CircuitBreakerCollection wraps InstrumentedCollection with circuit breaker protection
type CircuitBreakerCollection struct {
	collection     *InstrumentedCollection
	circuitBreaker *resilience.CircuitBreaker
	logger         *logging.Logger
}

// InsertOne inserts a single document with circuit breaker protection
func (c *CircuitBreakerCollection) InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	result, err := c.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return c.collection.InsertOne(ctx, document, opts...)
	})
	if err != nil {
		return nil, err
	}
	return result.(*mongo.InsertOneResult), nil
}

// InsertMany inserts multiple documents with circuit breaker protection
func (c *CircuitBreakerCollection) InsertMany(ctx context.Context, documents []interface{}, opts ...*options.InsertManyOptions) (*mongo.InsertManyResult, error) {
	result, err := c.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return c.collection.InsertMany(ctx, documents, opts...)
	})
	if err != nil {
		return nil, err
	}
	return result.(*mongo.InsertManyResult), nil
}

// FindOne finds a single document with circuit breaker protection
func (c *CircuitBreakerCollection) FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult {
	// Note: SingleResult doesn't return error immediately, so we execute but don't check breaker error here
	result, err := c.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return c.collection.FindOne(ctx, filter, opts...), nil
	})
	if err != nil {
		// Return an error result
		return &mongo.SingleResult{}
	}
	return result.(*mongo.SingleResult)
}

// Find finds multiple documents with circuit breaker protection
func (c *CircuitBreakerCollection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	result, err := c.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return c.collection.Find(ctx, filter, opts...)
	})
	if err != nil {
		return nil, err
	}
	return result.(*mongo.Cursor), nil
}

// UpdateOne updates a single document with circuit breaker protection
func (c *CircuitBreakerCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	result, err := c.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return c.collection.UpdateOne(ctx, filter, update, opts...)
	})
	if err != nil {
		return nil, err
	}
	return result.(*mongo.UpdateResult), nil
}

// UpdateMany updates multiple documents with circuit breaker protection
func (c *CircuitBreakerCollection) UpdateMany(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	result, err := c.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return c.collection.UpdateMany(ctx, filter, update, opts...)
	})
	if err != nil {
		return nil, err
	}
	return result.(*mongo.UpdateResult), nil
}

// DeleteOne deletes a single document with circuit breaker protection
func (c *CircuitBreakerCollection) DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	result, err := c.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return c.collection.DeleteOne(ctx, filter, opts...)
	})
	if err != nil {
		return nil, err
	}
	return result.(*mongo.DeleteResult), nil
}

// DeleteMany deletes multiple documents with circuit breaker protection
func (c *CircuitBreakerCollection) DeleteMany(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	result, err := c.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return c.collection.DeleteMany(ctx, filter, opts...)
	})
	if err != nil {
		return nil, err
	}
	return result.(*mongo.DeleteResult), nil
}

// CountDocuments counts documents with circuit breaker protection
func (c *CircuitBreakerCollection) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	result, err := c.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		count, err := c.collection.CountDocuments(ctx, filter, opts...)
		return count, err
	})
	if err != nil {
		return 0, err
	}
	return result.(int64), nil
}

// Aggregate runs an aggregation pipeline with circuit breaker protection
func (c *CircuitBreakerCollection) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (*mongo.Cursor, error) {
	result, err := c.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return c.collection.Aggregate(ctx, pipeline, opts...)
	})
	if err != nil {
		return nil, err
	}
	return result.(*mongo.Cursor), nil
}

// FindOneAndUpdate finds and updates a document with circuit breaker protection
func (c *CircuitBreakerCollection) FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) *mongo.SingleResult {
	result, err := c.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return c.collection.FindOneAndUpdate(ctx, filter, update, opts...), nil
	})
	if err != nil {
		return &mongo.SingleResult{}
	}
	return result.(*mongo.SingleResult)
}

// FindOneAndDelete finds and deletes a document with circuit breaker protection
func (c *CircuitBreakerCollection) FindOneAndDelete(ctx context.Context, filter interface{}, opts ...*options.FindOneAndDeleteOptions) *mongo.SingleResult {
	result, err := c.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return c.collection.FindOneAndDelete(ctx, filter, opts...), nil
	})
	if err != nil {
		return &mongo.SingleResult{}
	}
	return result.(*mongo.SingleResult)
}

// BulkWrite performs bulk write operations with circuit breaker protection
func (c *CircuitBreakerCollection) BulkWrite(ctx context.Context, models []mongo.WriteModel, opts ...*options.BulkWriteOptions) (*mongo.BulkWriteResult, error) {
	result, err := c.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return c.collection.BulkWrite(ctx, models, opts...)
	})
	if err != nil {
		return nil, err
	}
	return result.(*mongo.BulkWriteResult), nil
}

// CreateIndex creates an index with circuit breaker protection
func (c *CircuitBreakerCollection) CreateIndex(ctx context.Context, model mongo.IndexModel, opts ...*options.CreateIndexesOptions) (string, error) {
	result, err := c.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return c.collection.CreateIndex(ctx, model, opts...)
	})
	if err != nil {
		return "", err
	}
	return result.(string), nil
}

// Watch creates a change stream with circuit breaker protection
func (c *CircuitBreakerCollection) Watch(ctx context.Context, pipeline interface{}, opts ...*options.ChangeStreamOptions) (*mongo.ChangeStream, error) {
	result, err := c.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return c.collection.Watch(ctx, pipeline, opts...)
	})
	if err != nil {
		return nil, err
	}
	return result.(*mongo.ChangeStream), nil
}

// Underlying returns the underlying InstrumentedCollection
func (c *CircuitBreakerCollection) Underlying() *InstrumentedCollection {
	return c.collection
}

// Name returns the collection name
func (c *CircuitBreakerCollection) Name() string {
	return c.collection.Name()
}

// NewProductionClient creates a fully configured MongoDB client with instrumentation and circuit breaker
func NewProductionClient(ctx context.Context, config *Config, m *metrics.Metrics, logger *logging.Logger) (*CircuitBreakerClient, error) {
	// Create base client
	baseClient, err := NewClient(ctx, config)
	if err != nil {
		return nil, err
	}

	// Wrap with instrumentation
	instrumentedClient := NewInstrumentedClient(baseClient, m, logger)

	// Wrap with circuit breaker
	cbClient := NewCircuitBreakerClient(instrumentedClient, logger)

	return cbClient, nil
}
