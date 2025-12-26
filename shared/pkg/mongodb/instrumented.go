package mongodb

import (
	"context"
	"time"

	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/metrics"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedClient wraps a MongoDB Client with metrics and tracing
type InstrumentedClient struct {
	client  *Client
	metrics *metrics.Metrics
	logger  *logging.Logger
	tracer  trace.Tracer
}

// NewInstrumentedClient creates a new instrumented MongoDB client
func NewInstrumentedClient(client *Client, m *metrics.Metrics, logger *logging.Logger) *InstrumentedClient {
	return &InstrumentedClient{
		client:  client,
		metrics: m,
		logger:  logger,
		tracer:  otel.Tracer("mongodb"),
	}
}

// Collection returns an instrumented collection
func (c *InstrumentedClient) Collection(name string) *InstrumentedCollection {
	return &InstrumentedCollection{
		collection: c.client.Collection(name),
		name:       name,
		database:   c.client.config.Database,
		metrics:    c.metrics,
		logger:     c.logger,
		tracer:     c.tracer,
	}
}

// Database returns the underlying database handle
func (c *InstrumentedClient) Database() *mongo.Database {
	return c.client.Database()
}

// Client returns the underlying MongoDB client
func (c *InstrumentedClient) Client() *mongo.Client {
	return c.client.Client()
}

// Close disconnects the client
func (c *InstrumentedClient) Close(ctx context.Context) error {
	return c.client.Close(ctx)
}

// HealthCheck performs a health check with tracing
func (c *InstrumentedClient) HealthCheck(ctx context.Context) error {
	ctx, span := c.tracer.Start(ctx, "mongodb.ping",
		trace.WithAttributes(
			semconv.DBSystemMongoDB,
			semconv.DBNameKey.String(c.client.config.Database),
		),
	)
	defer span.End()

	err := c.client.HealthCheck(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
	return err
}

// WithTransaction executes a function within a transaction with tracing
func (c *InstrumentedClient) WithTransaction(ctx context.Context, fn func(sessCtx mongo.SessionContext) error) error {
	ctx, span := c.tracer.Start(ctx, "mongodb.transaction",
		trace.WithAttributes(
			semconv.DBSystemMongoDB,
			semconv.DBNameKey.String(c.client.config.Database),
		),
	)
	defer span.End()

	err := c.client.WithTransaction(ctx, fn)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
	return err
}

// RawClient returns the underlying Client for advanced operations
func (c *InstrumentedClient) RawClient() *Client {
	return c.client
}

// InstrumentedCollection wraps a MongoDB Collection with metrics and tracing
type InstrumentedCollection struct {
	collection *mongo.Collection
	name       string
	database   string
	metrics    *metrics.Metrics
	logger     *logging.Logger
	tracer     trace.Tracer
}

// startSpan starts a new span for a database operation
func (c *InstrumentedCollection) startSpan(ctx context.Context, operation string) (context.Context, trace.Span) {
	return c.tracer.Start(ctx, "mongodb."+operation,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.DBSystemMongoDB,
			semconv.DBNameKey.String(c.database),
			semconv.DBOperationKey.String(operation),
			attribute.String("db.collection", c.name),
		),
	)
}

// recordMetrics records operation metrics
func (c *InstrumentedCollection) recordMetrics(ctx context.Context, operation string, success bool, duration time.Duration, rowsAffected int64) {
	if c.metrics != nil {
		c.metrics.RecordMongoDBOperation(c.name, operation, success, duration)
	}
	if c.logger != nil {
		c.logger.DatabaseQuery(ctx, c.name, operation, duration, success, rowsAffected)
	}
}

// InsertOne inserts a single document with instrumentation
func (c *InstrumentedCollection) InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	start := time.Now()
	ctx, span := c.startSpan(ctx, "insertOne")
	defer span.End()

	result, err := c.collection.InsertOne(ctx, document, opts...)
	duration := time.Since(start)

	success := err == nil
	var rowsAffected int64 = 0
	if success {
		rowsAffected = 1
	}

	c.recordMetrics(ctx, "insertOne", success, duration, rowsAffected)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(attribute.Int64("db.rows_affected", rowsAffected))
	}

	return result, err
}

// InsertMany inserts multiple documents with instrumentation
func (c *InstrumentedCollection) InsertMany(ctx context.Context, documents []interface{}, opts ...*options.InsertManyOptions) (*mongo.InsertManyResult, error) {
	start := time.Now()
	ctx, span := c.startSpan(ctx, "insertMany")
	defer span.End()

	span.SetAttributes(attribute.Int("db.batch_size", len(documents)))

	result, err := c.collection.InsertMany(ctx, documents, opts...)
	duration := time.Since(start)

	success := err == nil
	var rowsAffected int64 = 0
	if success && result != nil {
		rowsAffected = int64(len(result.InsertedIDs))
	}

	c.recordMetrics(ctx, "insertMany", success, duration, rowsAffected)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(attribute.Int64("db.rows_affected", rowsAffected))
	}

	return result, err
}

// FindOne finds a single document with instrumentation
func (c *InstrumentedCollection) FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult {
	start := time.Now()
	ctx, span := c.startSpan(ctx, "findOne")
	defer span.End()

	result := c.collection.FindOne(ctx, filter, opts...)
	duration := time.Since(start)

	success := result.Err() == nil || result.Err() == mongo.ErrNoDocuments
	var rowsAffected int64 = 0
	if result.Err() == nil {
		rowsAffected = 1
	}

	c.recordMetrics(ctx, "findOne", success, duration, rowsAffected)

	if result.Err() != nil && result.Err() != mongo.ErrNoDocuments {
		span.RecordError(result.Err())
		span.SetStatus(codes.Error, result.Err().Error())
	} else {
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(attribute.Int64("db.rows_affected", rowsAffected))
	}

	return result
}

// Find finds multiple documents with instrumentation
func (c *InstrumentedCollection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	start := time.Now()
	ctx, span := c.startSpan(ctx, "find")
	defer span.End()

	cursor, err := c.collection.Find(ctx, filter, opts...)
	duration := time.Since(start)

	success := err == nil
	c.recordMetrics(ctx, "find", success, duration, 0) // Count not available until cursor iteration

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return cursor, err
}

// UpdateOne updates a single document with instrumentation
func (c *InstrumentedCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	start := time.Now()
	ctx, span := c.startSpan(ctx, "updateOne")
	defer span.End()

	result, err := c.collection.UpdateOne(ctx, filter, update, opts...)
	duration := time.Since(start)

	success := err == nil
	var rowsAffected int64 = 0
	if success && result != nil {
		rowsAffected = result.ModifiedCount
	}

	c.recordMetrics(ctx, "updateOne", success, duration, rowsAffected)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(
			attribute.Int64("db.rows_affected", rowsAffected),
			attribute.Int64("db.matched_count", result.MatchedCount),
			attribute.Int64("db.upserted_count", result.UpsertedCount),
		)
	}

	return result, err
}

// UpdateMany updates multiple documents with instrumentation
func (c *InstrumentedCollection) UpdateMany(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	start := time.Now()
	ctx, span := c.startSpan(ctx, "updateMany")
	defer span.End()

	result, err := c.collection.UpdateMany(ctx, filter, update, opts...)
	duration := time.Since(start)

	success := err == nil
	var rowsAffected int64 = 0
	if success && result != nil {
		rowsAffected = result.ModifiedCount
	}

	c.recordMetrics(ctx, "updateMany", success, duration, rowsAffected)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(
			attribute.Int64("db.rows_affected", rowsAffected),
			attribute.Int64("db.matched_count", result.MatchedCount),
		)
	}

	return result, err
}

// DeleteOne deletes a single document with instrumentation
func (c *InstrumentedCollection) DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	start := time.Now()
	ctx, span := c.startSpan(ctx, "deleteOne")
	defer span.End()

	result, err := c.collection.DeleteOne(ctx, filter, opts...)
	duration := time.Since(start)

	success := err == nil
	var rowsAffected int64 = 0
	if success && result != nil {
		rowsAffected = result.DeletedCount
	}

	c.recordMetrics(ctx, "deleteOne", success, duration, rowsAffected)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(attribute.Int64("db.rows_affected", rowsAffected))
	}

	return result, err
}

// DeleteMany deletes multiple documents with instrumentation
func (c *InstrumentedCollection) DeleteMany(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	start := time.Now()
	ctx, span := c.startSpan(ctx, "deleteMany")
	defer span.End()

	result, err := c.collection.DeleteMany(ctx, filter, opts...)
	duration := time.Since(start)

	success := err == nil
	var rowsAffected int64 = 0
	if success && result != nil {
		rowsAffected = result.DeletedCount
	}

	c.recordMetrics(ctx, "deleteMany", success, duration, rowsAffected)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(attribute.Int64("db.rows_affected", rowsAffected))
	}

	return result, err
}

// CountDocuments counts documents with instrumentation
func (c *InstrumentedCollection) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	start := time.Now()
	ctx, span := c.startSpan(ctx, "countDocuments")
	defer span.End()

	count, err := c.collection.CountDocuments(ctx, filter, opts...)
	duration := time.Since(start)

	success := err == nil
	c.recordMetrics(ctx, "countDocuments", success, duration, count)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(attribute.Int64("db.count", count))
	}

	return count, err
}

// Aggregate runs an aggregation pipeline with instrumentation
func (c *InstrumentedCollection) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (*mongo.Cursor, error) {
	start := time.Now()
	ctx, span := c.startSpan(ctx, "aggregate")
	defer span.End()

	cursor, err := c.collection.Aggregate(ctx, pipeline, opts...)
	duration := time.Since(start)

	success := err == nil
	c.recordMetrics(ctx, "aggregate", success, duration, 0)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return cursor, err
}

// FindOneAndUpdate finds and updates a document with instrumentation
func (c *InstrumentedCollection) FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) *mongo.SingleResult {
	start := time.Now()
	ctx, span := c.startSpan(ctx, "findOneAndUpdate")
	defer span.End()

	result := c.collection.FindOneAndUpdate(ctx, filter, update, opts...)
	duration := time.Since(start)

	success := result.Err() == nil || result.Err() == mongo.ErrNoDocuments
	var rowsAffected int64 = 0
	if result.Err() == nil {
		rowsAffected = 1
	}

	c.recordMetrics(ctx, "findOneAndUpdate", success, duration, rowsAffected)

	if result.Err() != nil && result.Err() != mongo.ErrNoDocuments {
		span.RecordError(result.Err())
		span.SetStatus(codes.Error, result.Err().Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return result
}

// FindOneAndDelete finds and deletes a document with instrumentation
func (c *InstrumentedCollection) FindOneAndDelete(ctx context.Context, filter interface{}, opts ...*options.FindOneAndDeleteOptions) *mongo.SingleResult {
	start := time.Now()
	ctx, span := c.startSpan(ctx, "findOneAndDelete")
	defer span.End()

	result := c.collection.FindOneAndDelete(ctx, filter, opts...)
	duration := time.Since(start)

	success := result.Err() == nil || result.Err() == mongo.ErrNoDocuments
	var rowsAffected int64 = 0
	if result.Err() == nil {
		rowsAffected = 1
	}

	c.recordMetrics(ctx, "findOneAndDelete", success, duration, rowsAffected)

	if result.Err() != nil && result.Err() != mongo.ErrNoDocuments {
		span.RecordError(result.Err())
		span.SetStatus(codes.Error, result.Err().Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return result
}

// BulkWrite performs bulk write operations with instrumentation
func (c *InstrumentedCollection) BulkWrite(ctx context.Context, models []mongo.WriteModel, opts ...*options.BulkWriteOptions) (*mongo.BulkWriteResult, error) {
	start := time.Now()
	ctx, span := c.startSpan(ctx, "bulkWrite")
	defer span.End()

	span.SetAttributes(attribute.Int("db.bulk_operations", len(models)))

	result, err := c.collection.BulkWrite(ctx, models, opts...)
	duration := time.Since(start)

	success := err == nil
	var rowsAffected int64 = 0
	if success && result != nil {
		rowsAffected = result.InsertedCount + result.ModifiedCount + result.DeletedCount
	}

	c.recordMetrics(ctx, "bulkWrite", success, duration, rowsAffected)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(
			attribute.Int64("db.rows_affected", rowsAffected),
			attribute.Int64("db.inserted_count", result.InsertedCount),
			attribute.Int64("db.modified_count", result.ModifiedCount),
			attribute.Int64("db.deleted_count", result.DeletedCount),
		)
	}

	return result, err
}

// CreateIndex creates an index with instrumentation
func (c *InstrumentedCollection) CreateIndex(ctx context.Context, model mongo.IndexModel, opts ...*options.CreateIndexesOptions) (string, error) {
	start := time.Now()
	ctx, span := c.startSpan(ctx, "createIndex")
	defer span.End()

	name, err := c.collection.Indexes().CreateOne(ctx, model, opts...)
	duration := time.Since(start)

	success := err == nil
	c.recordMetrics(ctx, "createIndex", success, duration, 0)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(attribute.String("db.index_name", name))
	}

	return name, err
}

// Watch creates a change stream with instrumentation
func (c *InstrumentedCollection) Watch(ctx context.Context, pipeline interface{}, opts ...*options.ChangeStreamOptions) (*mongo.ChangeStream, error) {
	start := time.Now()
	ctx, span := c.startSpan(ctx, "watch")
	defer span.End()

	stream, err := c.collection.Watch(ctx, pipeline, opts...)
	duration := time.Since(start)

	success := err == nil
	c.recordMetrics(ctx, "watch", success, duration, 0)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return stream, err
}

// Underlying returns the underlying mongo.Collection
func (c *InstrumentedCollection) Underlying() *mongo.Collection {
	return c.collection
}

// Name returns the collection name
func (c *InstrumentedCollection) Name() string {
	return c.name
}

// ConnectionPoolMonitor monitors MongoDB connection pool metrics
type ConnectionPoolMonitor struct {
	metrics *metrics.Metrics
}

// NewConnectionPoolMonitor creates a new connection pool monitor
func NewConnectionPoolMonitor(m *metrics.Metrics) *ConnectionPoolMonitor {
	return &ConnectionPoolMonitor{metrics: m}
}

// UpdateConnectionCount updates the connection count metric
func (m *ConnectionPoolMonitor) UpdateConnectionCount(count int) {
	if m.metrics != nil {
		m.metrics.SetMongoDBConnections(count)
	}
}

// Helper function to convert filter to string for logging (limited info)
func filterToString(filter interface{}) string {
	if filter == nil {
		return "{}"
	}
	if f, ok := filter.(bson.M); ok {
		keys := make([]string, 0, len(f))
		for k := range f {
			keys = append(keys, k)
		}
		return "{" + joinStrings(keys, ", ") + "}"
	}
	return "{...}"
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
