package projections

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InventoryListProjectionRepository manages the inventory list read model
type InventoryListProjectionRepository interface {
	// Upsert creates or updates an inventory projection
	Upsert(ctx context.Context, projection *InventoryListProjection) error

	// FindBySKU retrieves a projection by SKU
	FindBySKU(ctx context.Context, sku string) (*InventoryListProjection, error)

	// FindWithFilter retrieves projections matching filter criteria with pagination
	FindWithFilter(ctx context.Context, filter InventoryListFilter, page Pagination) (*PagedResult[InventoryListProjection], error)

	// UpdateFields updates specific fields of a projection
	UpdateFields(ctx context.Context, sku string, updates map[string]interface{}) error

	// Delete removes a projection
	Delete(ctx context.Context, sku string) error

	// Count returns the total count matching filter
	Count(ctx context.Context, filter InventoryListFilter) (int64, error)
}

// MongoInventoryListProjectionRepository is the MongoDB implementation
type MongoInventoryListProjectionRepository struct {
	collection *mongo.Collection
}

// NewMongoInventoryListProjectionRepository creates a new repository
func NewMongoInventoryListProjectionRepository(db *mongo.Database) *MongoInventoryListProjectionRepository {
	collection := db.Collection("inventory_list_projections")
	repo := &MongoInventoryListProjectionRepository{
		collection: collection,
	}
	repo.ensureIndexes(context.Background())
	return repo
}

// ensureIndexes creates necessary indexes for efficient queries
func (r *MongoInventoryListProjectionRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "sku", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "productName", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "isLowStock", Value: 1}, {Key: "availableQuantity", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "isOutOfStock", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "availableLocations", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "lastReceived", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "lastPicked", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "availableQuantity", Value: 1}},
		},
	}

	r.collection.Indexes().CreateMany(ctx, indexes)
}

// Upsert creates or updates an inventory projection
func (r *MongoInventoryListProjectionRepository) Upsert(ctx context.Context, projection *InventoryListProjection) error {
	projection.UpdatedAt = time.Now()

	filter := bson.M{"_id": projection.SKU}
	update := bson.M{"$set": projection}
	opts := options.Update().SetUpsert(true)

	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// FindBySKU retrieves a projection by SKU
func (r *MongoInventoryListProjectionRepository) FindBySKU(ctx context.Context, sku string) (*InventoryListProjection, error) {
	var projection InventoryListProjection
	filter := bson.M{"_id": sku}

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
func (r *MongoInventoryListProjectionRepository) FindWithFilter(ctx context.Context, filter InventoryListFilter, page Pagination) (*PagedResult[InventoryListProjection], error) {
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
		sortField = "updatedAt" // Default sort
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
	var projections []InventoryListProjection
	if err := cursor.All(ctx, &projections); err != nil {
		return nil, err
	}

	// Build paginated result
	result := &PagedResult[InventoryListProjection]{
		Items:   projections,
		Total:   total,
		Limit:   page.Limit,
		Offset:  page.Offset,
		HasMore: int64(page.Offset+len(projections)) < total,
	}

	return result, nil
}

// buildFilterQuery builds MongoDB filter from InventoryListFilter
func (r *MongoInventoryListProjectionRepository) buildFilterQuery(filter InventoryListFilter) bson.M {
	query := bson.M{}

	if filter.SKU != nil {
		query["_id"] = *filter.SKU
	}

	if filter.ProductName != nil {
		query["productName"] = bson.M{"$regex": *filter.ProductName, "$options": "i"}
	}

	if filter.IsLowStock != nil {
		query["isLowStock"] = *filter.IsLowStock
	}

	if filter.IsOutOfStock != nil {
		query["isOutOfStock"] = *filter.IsOutOfStock
	}

	if filter.HasReservations != nil {
		if *filter.HasReservations {
			query["activeReservations"] = bson.M{"$gt": 0}
		} else {
			query["activeReservations"] = 0
		}
	}

	if filter.MinQuantity != nil {
		query["availableQuantity"] = bson.M{"$gte": *filter.MinQuantity}
	}

	if filter.MaxQuantity != nil {
		if query["availableQuantity"] != nil {
			query["availableQuantity"].(bson.M)["$lte"] = *filter.MaxQuantity
		} else {
			query["availableQuantity"] = bson.M{"$lte": *filter.MaxQuantity}
		}
	}

	if filter.LocationID != nil {
		query["availableLocations"] = *filter.LocationID
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
		query["lastReceived"] = dateQuery
	}

	// Text search (search across multiple fields)
	if filter.SearchTerm != "" {
		query["$or"] = []bson.M{
			{"_id": bson.M{"$regex": filter.SearchTerm, "$options": "i"}},
			{"productName": bson.M{"$regex": filter.SearchTerm, "$options": "i"}},
		}
	}

	return query
}

// UpdateFields updates specific fields of a projection
func (r *MongoInventoryListProjectionRepository) UpdateFields(ctx context.Context, sku string, updates map[string]interface{}) error {
	// Add updatedAt timestamp
	updates["updatedAt"] = time.Now()

	filter := bson.M{"_id": sku}
	update := bson.M{"$set": updates}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

// Delete removes a projection
func (r *MongoInventoryListProjectionRepository) Delete(ctx context.Context, sku string) error {
	filter := bson.M{"_id": sku}
	_, err := r.collection.DeleteOne(ctx, filter)
	return err
}

// Count returns the total count matching filter
func (r *MongoInventoryListProjectionRepository) Count(ctx context.Context, filter InventoryListFilter) (int64, error) {
	query := r.buildFilterQuery(filter)
	return r.collection.CountDocuments(ctx, query)
}
