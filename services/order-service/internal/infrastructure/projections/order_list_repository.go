package projections

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// OrderListProjectionRepository manages the order list read model
type OrderListProjectionRepository interface {
	// Upsert creates or updates an order projection
	Upsert(ctx context.Context, projection *OrderListProjection) error

	// FindByID retrieves a projection by order ID
	FindByID(ctx context.Context, orderID string) (*OrderListProjection, error)

	// FindWithFilter retrieves projections matching filter criteria with pagination
	FindWithFilter(ctx context.Context, filter OrderListFilter, page Pagination) (*PagedResult[OrderListProjection], error)

	// UpdateFields updates specific fields of a projection
	UpdateFields(ctx context.Context, orderID string, updates map[string]interface{}) error

	// Delete removes a projection
	Delete(ctx context.Context, orderID string) error

	// Count returns the total count matching filter
	Count(ctx context.Context, filter OrderListFilter) (int64, error)
}

// MongoOrderListProjectionRepository is the MongoDB implementation
type MongoOrderListProjectionRepository struct {
	collection *mongo.Collection
}

// NewMongoOrderListProjectionRepository creates a new repository
func NewMongoOrderListProjectionRepository(db *mongo.Database) *MongoOrderListProjectionRepository {
	collection := db.Collection("order_list_projections")
	repo := &MongoOrderListProjectionRepository{
		collection: collection,
	}
	repo.ensureIndexes(context.Background())
	return repo
}

// ensureIndexes creates necessary indexes for efficient queries
func (r *MongoOrderListProjectionRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "orderId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}, {Key: "createdAt", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "priority", Value: 1}, {Key: "receivedAt", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "waveId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "customerId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "assignedPicker", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "isLate", Value: 1}, {Key: "promisedDeliveryAt", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "isPriority", Value: 1}, {Key: "receivedAt", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "shipToState", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "shipToCountry", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "receivedAt", Value: -1}},
		},
	}

	r.collection.Indexes().CreateMany(ctx, indexes)
}

// Upsert creates or updates an order projection
func (r *MongoOrderListProjectionRepository) Upsert(ctx context.Context, projection *OrderListProjection) error {
	projection.UpdatedAt = time.Now()

	filter := bson.M{"orderId": projection.OrderID}
	update := bson.M{"$set": projection}
	opts := options.Update().SetUpsert(true)

	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// FindByID retrieves a projection by order ID
func (r *MongoOrderListProjectionRepository) FindByID(ctx context.Context, orderID string) (*OrderListProjection, error) {
	var projection OrderListProjection
	filter := bson.M{"orderId": orderID}

	err := r.collection.FindOne(ctx, filter).Decode(&projection)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &projection, nil
}

// FindWithFilter retrieves projections matching filter criteria with pagination
func (r *MongoOrderListProjectionRepository) FindWithFilter(ctx context.Context, filter OrderListFilter, page Pagination) (*PagedResult[OrderListProjection], error) {
	// Build filter query
	query := r.buildFilterQuery(filter)

	// Get total count
	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, err
	}

	// Build find options with pagination and sorting
	opts := options.Find()

	// Set limit and offset
	if page.Limit > 0 {
		opts.SetLimit(int64(page.Limit))
	}
	if page.Offset > 0 {
		opts.SetSkip(int64(page.Offset))
	}

	// Set sort order
	sortField := page.SortBy
	if sortField == "" {
		sortField = "createdAt" // Default sort
	}
	sortOrder := -1 // Default descending
	if page.SortOrder == "asc" {
		sortOrder = 1
	}
	opts.SetSort(bson.D{{Key: sortField, Value: sortOrder}})

	// Execute query
	cursor, err := r.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// Decode results
	var projections []OrderListProjection
	if err := cursor.All(ctx, &projections); err != nil {
		return nil, err
	}

	// Build paginated result
	result := &PagedResult[OrderListProjection]{
		Items:  projections,
		Total:  total,
		Limit:  page.Limit,
		Offset: page.Offset,
		HasMore: int64(page.Offset+len(projections)) < total,
	}

	return result, nil
}

// buildFilterQuery builds MongoDB filter from OrderListFilter
func (r *MongoOrderListProjectionRepository) buildFilterQuery(filter OrderListFilter) bson.M {
	query := bson.M{}

	if filter.Status != nil {
		query["status"] = *filter.Status
	}

	if filter.Priority != nil {
		query["priority"] = *filter.Priority
	}

	if filter.WaveID != nil {
		query["waveId"] = *filter.WaveID
	}

	if filter.CustomerID != nil {
		query["customerId"] = *filter.CustomerID
	}

	if filter.AssignedPicker != nil {
		query["assignedPicker"] = *filter.AssignedPicker
	}

	if filter.ShipToState != nil {
		query["shipToState"] = *filter.ShipToState
	}

	if filter.ShipToCountry != nil {
		query["shipToCountry"] = *filter.ShipToCountry
	}

	if filter.IsLate != nil {
		query["isLate"] = *filter.IsLate
	}

	if filter.IsPriority != nil {
		query["isPriority"] = *filter.IsPriority
	}

	// Date range filters
	if filter.ReceivedAfter != nil || filter.ReceivedBefore != nil {
		dateQuery := bson.M{}
		if filter.ReceivedAfter != nil {
			dateQuery["$gte"] = *filter.ReceivedAfter
		}
		if filter.ReceivedBefore != nil {
			dateQuery["$lte"] = *filter.ReceivedBefore
		}
		query["receivedAt"] = dateQuery
	}

	// Text search (search across multiple fields)
	if filter.SearchTerm != "" {
		query["$or"] = []bson.M{
			{"orderId": bson.M{"$regex": filter.SearchTerm, "$options": "i"}},
			{"customerId": bson.M{"$regex": filter.SearchTerm, "$options": "i"}},
			{"customerName": bson.M{"$regex": filter.SearchTerm, "$options": "i"}},
			{"trackingNumber": bson.M{"$regex": filter.SearchTerm, "$options": "i"}},
		}
	}

	return query
}

// UpdateFields updates specific fields of a projection
func (r *MongoOrderListProjectionRepository) UpdateFields(ctx context.Context, orderID string, updates map[string]interface{}) error {
	// Add updatedAt timestamp
	updates["updatedAt"] = time.Now()

	filter := bson.M{"orderId": orderID}
	update := bson.M{"$set": updates}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

// Delete removes a projection
func (r *MongoOrderListProjectionRepository) Delete(ctx context.Context, orderID string) error {
	filter := bson.M{"orderId": orderID}
	_, err := r.collection.DeleteOne(ctx, filter)
	return err
}

// Count returns the total count matching filter
func (r *MongoOrderListProjectionRepository) Count(ctx context.Context, filter OrderListFilter) (int64, error) {
	query := r.buildFilterQuery(filter)
	return r.collection.CountDocuments(ctx, query)
}
