package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/outbox"
	outboxMongo "github.com/wms-platform/shared/pkg/outbox/mongodb"
	"github.com/wms-platform/shipping-service/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ManifestRepository implements the repository for OutboundManifest aggregate
type ManifestRepository struct {
	collection   *mongo.Collection
	db           *mongo.Database
	outboxRepo   *outboxMongo.OutboxRepository
	eventFactory *cloudevents.EventFactory
}

// NewManifestRepository creates a new ManifestRepository
func NewManifestRepository(db *mongo.Database, eventFactory *cloudevents.EventFactory) *ManifestRepository {
	collection := db.Collection("manifests")
	outboxRepo := outboxMongo.NewOutboxRepository(db)

	repo := &ManifestRepository{
		collection:   collection,
		db:           db,
		outboxRepo:   outboxRepo,
		eventFactory: eventFactory,
	}
	repo.ensureIndexes(context.Background())
	return repo
}

func (r *ManifestRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "manifestId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "carrierId", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "dispatchDock", Value: 1}}},
		{Keys: bson.D{{Key: "trailerId", Value: 1}}},
		{Keys: bson.D{{Key: "createdAt", Value: -1}}},
		{Keys: bson.D{{Key: "closedAt", Value: -1}}},
		{Keys: bson.D{{Key: "dispatchedAt", Value: -1}}},
		{Keys: bson.D{
			{Key: "carrierId", Value: 1},
			{Key: "status", Value: 1},
		}},
	}
	r.collection.Indexes().CreateMany(ctx, indexes)

	// Ensure outbox indexes
	_ = r.outboxRepo.EnsureIndexes(ctx)
}

// Save saves an OutboundManifest with transactional outbox pattern
func (r *ManifestRepository) Save(ctx context.Context, manifest *domain.OutboundManifest) error {
	manifest.UpdatedAt = time.Now().UTC()

	// Start a MongoDB session for transaction
	session, err := r.db.Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	// Execute transaction
	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		// 1. Save the aggregate
		opts := options.Update().SetUpsert(true)
		filter := bson.M{"manifestId": manifest.ManifestID}
		update := bson.M{"$set": manifest}

		if _, err := r.collection.UpdateOne(sessCtx, filter, update, opts); err != nil {
			return nil, fmt.Errorf("failed to save manifest: %w", err)
		}

		// 2. Save domain events to outbox
		domainEvents := manifest.GetManifestDomainEvents()
		if len(domainEvents) > 0 {
			outboxEvents := make([]*outbox.OutboxEvent, 0, len(domainEvents))

			for _, event := range domainEvents {
				var cloudEvent *cloudevents.WMSCloudEvent
				switch e := event.(type) {
				case *domain.ManifestClosedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "manifest/"+e.ManifestID, e)
				case *domain.ManifestDispatchedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "manifest/"+e.ManifestID, e)
				default:
					continue
				}

				// Create outbox event from CloudEvent
				outboxEvent, err := outbox.NewOutboxEventFromCloudEvent(
					manifest.ManifestID,
					"OutboundManifest",
					kafka.Topics.ShippingEvents,
					cloudEvent,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create outbox event: %w", err)
				}

				outboxEvents = append(outboxEvents, outboxEvent)
			}

			// Save all outbox events in the same transaction
			if len(outboxEvents) > 0 {
				if err := r.outboxRepo.SaveAll(sessCtx, outboxEvents); err != nil {
					return nil, fmt.Errorf("failed to save outbox events: %w", err)
				}
			}
		}

		// 3. Clear domain events from the aggregate
		manifest.ClearManifestDomainEvents()

		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

// FindByID finds a manifest by its ID
func (r *ManifestRepository) FindByID(ctx context.Context, manifestID string) (*domain.OutboundManifest, error) {
	var manifest domain.OutboundManifest
	err := r.collection.FindOne(ctx, bson.M{"manifestId": manifestID}).Decode(&manifest)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find manifest: %w", err)
	}
	return &manifest, nil
}

// FindByCarrierID finds all manifests for a carrier
func (r *ManifestRepository) FindByCarrierID(ctx context.Context, carrierID string) ([]*domain.OutboundManifest, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"carrierId": carrierID})
	if err != nil {
		return nil, fmt.Errorf("failed to find manifests: %w", err)
	}
	defer cursor.Close(ctx)

	var manifests []*domain.OutboundManifest
	if err := cursor.All(ctx, &manifests); err != nil {
		return nil, fmt.Errorf("failed to decode manifests: %w", err)
	}
	return manifests, nil
}

// FindByStatus finds all manifests with a specific status
func (r *ManifestRepository) FindByStatus(ctx context.Context, status domain.ManifestStatus) ([]*domain.OutboundManifest, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"status": status})
	if err != nil {
		return nil, fmt.Errorf("failed to find manifests: %w", err)
	}
	defer cursor.Close(ctx)

	var manifests []*domain.OutboundManifest
	if err := cursor.All(ctx, &manifests); err != nil {
		return nil, fmt.Errorf("failed to decode manifests: %w", err)
	}
	return manifests, nil
}

// FindOpenByCarrier finds open manifests for a specific carrier
func (r *ManifestRepository) FindOpenByCarrier(ctx context.Context, carrierID string) (*domain.OutboundManifest, error) {
	var manifest domain.OutboundManifest
	err := r.collection.FindOne(ctx, bson.M{
		"carrierId": carrierID,
		"status":    domain.ManifestStatusOpen,
	}).Decode(&manifest)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find open manifest: %w", err)
	}
	return &manifest, nil
}

// FindByDispatchDock finds manifests assigned to a dispatch dock
func (r *ManifestRepository) FindByDispatchDock(ctx context.Context, dispatchDock string) ([]*domain.OutboundManifest, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"dispatchDock": dispatchDock})
	if err != nil {
		return nil, fmt.Errorf("failed to find manifests: %w", err)
	}
	defer cursor.Close(ctx)

	var manifests []*domain.OutboundManifest
	if err := cursor.All(ctx, &manifests); err != nil {
		return nil, fmt.Errorf("failed to decode manifests: %w", err)
	}
	return manifests, nil
}

// FindByTrailerID finds manifests assigned to a trailer
func (r *ManifestRepository) FindByTrailerID(ctx context.Context, trailerID string) ([]*domain.OutboundManifest, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"trailerId": trailerID})
	if err != nil {
		return nil, fmt.Errorf("failed to find manifests: %w", err)
	}
	defer cursor.Close(ctx)

	var manifests []*domain.OutboundManifest
	if err := cursor.All(ctx, &manifests); err != nil {
		return nil, fmt.Errorf("failed to decode manifests: %w", err)
	}
	return manifests, nil
}

// FindByDateRange finds manifests created within a date range
func (r *ManifestRepository) FindByDateRange(ctx context.Context, start, end time.Time) ([]*domain.OutboundManifest, error) {
	cursor, err := r.collection.Find(ctx, bson.M{
		"createdAt": bson.M{
			"$gte": start,
			"$lte": end,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find manifests: %w", err)
	}
	defer cursor.Close(ctx)

	var manifests []*domain.OutboundManifest
	if err := cursor.All(ctx, &manifests); err != nil {
		return nil, fmt.Errorf("failed to decode manifests: %w", err)
	}
	return manifests, nil
}

// FindClosedByCarrier finds closed manifests for a carrier ready for dispatch
func (r *ManifestRepository) FindClosedByCarrier(ctx context.Context, carrierID string) ([]*domain.OutboundManifest, error) {
	cursor, err := r.collection.Find(ctx, bson.M{
		"carrierId": carrierID,
		"status":    domain.ManifestStatusClosed,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find manifests: %w", err)
	}
	defer cursor.Close(ctx)

	var manifests []*domain.OutboundManifest
	if err := cursor.All(ctx, &manifests); err != nil {
		return nil, fmt.Errorf("failed to decode manifests: %w", err)
	}
	return manifests, nil
}

// FindDispatchedToday finds all manifests dispatched today
func (r *ManifestRepository) FindDispatchedToday(ctx context.Context) ([]*domain.OutboundManifest, error) {
	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	cursor, err := r.collection.Find(ctx, bson.M{
		"status": domain.ManifestStatusDispatched,
		"dispatchedAt": bson.M{
			"$gte": startOfDay,
			"$lt":  endOfDay,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find manifests: %w", err)
	}
	defer cursor.Close(ctx)

	var manifests []*domain.OutboundManifest
	if err := cursor.All(ctx, &manifests); err != nil {
		return nil, fmt.Errorf("failed to decode manifests: %w", err)
	}
	return manifests, nil
}

// CountByStatus counts manifests by status
func (r *ManifestRepository) CountByStatus(ctx context.Context, status domain.ManifestStatus) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{"status": status})
}

// Delete deletes a manifest by ID
func (r *ManifestRepository) Delete(ctx context.Context, manifestID string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"manifestId": manifestID})
	if err != nil {
		return fmt.Errorf("failed to delete manifest: %w", err)
	}
	return nil
}

// GetOutboxRepository returns the outbox repository for this service
func (r *ManifestRepository) GetOutboxRepository() outbox.Repository {
	return r.outboxRepo
}
