package mongodb

import (
	"context"
	"time"

	"github.com/wms-platform/services/stow-service/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type StorageLocationRepository struct {
	collection *mongo.Collection
}

func NewStorageLocationRepository(db *mongo.Database) *StorageLocationRepository {
	collection := db.Collection("storage_locations")

	repo := &StorageLocationRepository{
		collection: collection,
	}
	repo.ensureIndexes(context.Background())
	return repo
}

func (r *StorageLocationRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "locationId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "zone", Value: 1}}},
		{Keys: bson.D{{Key: "zone", Value: 1}, {Key: "aisle", Value: 1}}},
		{Keys: bson.D{{Key: "capacity", Value: 1}, {Key: "currentQuantity", Value: 1}}},
		{Keys: bson.D{{Key: "allowsHazmat", Value: 1}}},
		{Keys: bson.D{{Key: "allowsColdChain", Value: 1}}},
	}
	r.collection.Indexes().CreateMany(ctx, indexes)
}

func (r *StorageLocationRepository) FindAvailableLocations(ctx context.Context, constraints domain.LocationConstraints, limit int) ([]domain.StorageLocation, error) {
	filter := bson.M{
		"$expr": bson.M{
			"$gte": bson.A{
				bson.M{"$subtract": bson.A{"$capacity", "$currentQuantity"}},
				constraints.MinCapacity,
			},
		},
	}

	// Add weight constraint
	if constraints.MinWeight > 0 {
		filter["$and"] = []bson.M{
			filter,
			{
				"$expr": bson.M{
					"$gte": bson.A{
						bson.M{"$subtract": bson.A{"$maxWeight", "$currentWeight"}},
						constraints.MinWeight,
					},
				},
			},
		}
	}

	// Add special handling constraints
	if constraints.RequiresHazmat {
		filter["allowsHazmat"] = true
	}
	if constraints.RequiresColdChain {
		filter["allowsColdChain"] = true
	}
	if constraints.RequiresOversized {
		filter["allowsOversized"] = true
	}

	// Prefer specific zone if provided
	opts := options.Find().SetLimit(int64(limit))
	if constraints.PreferredZone != "" {
		opts.SetSort(bson.D{
			{Key: "zone", Value: 1},
		})
		// Add zone preference as a soft filter
		filter["zone"] = constraints.PreferredZone
	}

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var locations []domain.StorageLocation
	if err = cursor.All(ctx, &locations); err != nil {
		return nil, err
	}

	// If no locations found with preferred zone, try without zone filter
	if len(locations) == 0 && constraints.PreferredZone != "" {
		delete(filter, "zone")
		cursor, err = r.collection.Find(ctx, filter, opts)
		if err != nil {
			return nil, err
		}
		defer cursor.Close(ctx)
		err = cursor.All(ctx, &locations)
	}

	return locations, err
}

func (r *StorageLocationRepository) FindByID(ctx context.Context, locationID string) (*domain.StorageLocation, error) {
	var location domain.StorageLocation
	err := r.collection.FindOne(ctx, bson.M{"locationId": locationID}).Decode(&location)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &location, err
}

func (r *StorageLocationRepository) FindByZone(ctx context.Context, zone string) ([]domain.StorageLocation, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"zone": zone})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var locations []domain.StorageLocation
	err = cursor.All(ctx, &locations)
	return locations, err
}

func (r *StorageLocationRepository) UpdateCapacity(ctx context.Context, locationID string, quantityChange int, weightChange float64) error {
	update := bson.M{
		"$inc": bson.M{
			"currentQuantity": quantityChange,
			"currentWeight":   weightChange,
		},
		"$set": bson.M{
			"updatedAt": time.Now(),
		},
	}
	_, err := r.collection.UpdateOne(ctx, bson.M{"locationId": locationID}, update)
	return err
}

// Save saves or updates a storage location
func (r *StorageLocationRepository) Save(ctx context.Context, location *domain.StorageLocation) error {
	opts := options.Update().SetUpsert(true)
	filter := bson.M{"locationId": location.LocationID}
	update := bson.M{"$set": location}
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}
