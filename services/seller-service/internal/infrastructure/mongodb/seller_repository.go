package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/wms-platform/services/seller-service/internal/domain"
	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/outbox"
	outboxMongo "github.com/wms-platform/shared/pkg/outbox/mongodb"
	"github.com/wms-platform/shared/pkg/tenant"
)

// SellerRepository implements domain.SellerRepository using MongoDB
type SellerRepository struct {
	collection   *mongo.Collection
	db           *mongo.Database
	outboxRepo   *outboxMongo.OutboxRepository
	eventFactory *cloudevents.EventFactory
	tenantHelper *tenant.RepositoryHelper
}

// NewSellerRepository creates a new SellerRepository
func NewSellerRepository(db *mongo.Database, eventFactory *cloudevents.EventFactory) *SellerRepository {
	collection := db.Collection("sellers")
	outboxRepo := outboxMongo.NewOutboxRepository(db)

	// Create indexes
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		// Unique seller ID within tenant scope
		{
			Keys: bson.D{
				{Key: "tenantId", Value: 1},
				{Key: "sellerId", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		// Backward compatibility: unique sellerId for existing data
		{
			Keys:    bson.D{{Key: "sellerId", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
		// Tenant + status index
		{
			Keys: bson.D{
				{Key: "tenantId", Value: 1},
				{Key: "status", Value: 1},
				{Key: "createdAt", Value: -1},
			},
		},
		// Email index (for lookup)
		{
			Keys:    bson.D{{Key: "contactEmail", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
		// API key lookup index
		{
			Keys: bson.D{{Key: "apiKeys.hashedKey", Value: 1}},
		},
		// Text index for search
		{
			Keys: bson.D{
				{Key: "companyName", Value: "text"},
				{Key: "contactEmail", Value: "text"},
				{Key: "contactName", Value: "text"},
			},
		},
		// Channel integration index
		{
			Keys: bson.D{
				{Key: "tenantId", Value: 1},
				{Key: "integrations.channelType", Value: 1},
			},
		},
		// Facility assignment index
		{
			Keys: bson.D{
				{Key: "tenantId", Value: 1},
				{Key: "assignedFacilities.facilityId", Value: 1},
			},
		},
	}

	_, _ = collection.Indexes().CreateMany(ctx, indexes)

	// Create outbox indexes
	_ = outboxRepo.EnsureIndexes(ctx)

	return &SellerRepository{
		collection:   collection,
		db:           db,
		outboxRepo:   outboxRepo,
		eventFactory: eventFactory,
		tenantHelper: tenant.NewRepositoryHelper(false), // Don't enforce tenant initially for backward compatibility
	}
}

// Save persists a seller with its domain events in a single transaction
func (r *SellerRepository) Save(ctx context.Context, seller *domain.Seller) error {
	seller.UpdatedAt = time.Now().UTC()

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
		filter := bson.M{"sellerId": seller.SellerID}
		update := bson.M{"$set": seller}

		if _, err := r.collection.UpdateOne(sessCtx, filter, update, opts); err != nil {
			return nil, fmt.Errorf("failed to save seller: %w", err)
		}

		// 2. Save domain events to outbox
		domainEvents := seller.DomainEvents()
		if len(domainEvents) > 0 {
			outboxEvents := make([]*outbox.OutboxEvent, 0, len(domainEvents))

			for _, event := range domainEvents {
				// Convert domain event to CloudEvent
				var cloudEvent *cloudevents.WMSCloudEvent
				switch e := event.(type) {
				case *domain.SellerCreatedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "seller/"+e.SellerID, e)
				case *domain.SellerActivatedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "seller/"+e.SellerID, e)
				case *domain.SellerSuspendedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "seller/"+e.SellerID, e)
				case *domain.SellerClosedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "seller/"+e.SellerID, e)
				case *domain.FacilityAssignedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "seller/"+e.SellerID, e)
				case *domain.ChannelConnectedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "seller/"+e.SellerID, e)
				default:
					continue
				}

				// Create outbox event from CloudEvent
				outboxEvent, err := outbox.NewOutboxEventFromCloudEvent(
					seller.SellerID,
					"Seller",
					kafka.Topics.SellerEvents,
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
		seller.ClearDomainEvents()

		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

// FindByID retrieves a seller by its SellerID with tenant scoping
func (r *SellerRepository) FindByID(ctx context.Context, sellerID string) (*domain.Seller, error) {
	var seller domain.Seller
	filter := bson.M{"sellerId": sellerID}

	// Apply tenant filtering
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	err := r.collection.FindOne(ctx, filter).Decode(&seller)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return &seller, nil
}

// FindByTenantID retrieves all sellers for a tenant
func (r *SellerRepository) FindByTenantID(ctx context.Context, tenantID string, pagination domain.Pagination) ([]*domain.Seller, error) {
	filter := bson.M{"tenantId": tenantID}

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(pagination.Skip()).
		SetLimit(pagination.Limit())

	return r.findMany(ctx, filter, opts)
}

// FindByStatus retrieves sellers by status with tenant scoping
func (r *SellerRepository) FindByStatus(ctx context.Context, status domain.SellerStatus, pagination domain.Pagination) ([]*domain.Seller, error) {
	filter := bson.M{"status": status}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(pagination.Skip()).
		SetLimit(pagination.Limit())

	return r.findMany(ctx, filter, opts)
}

// FindByAPIKey finds a seller by API key (hashed)
func (r *SellerRepository) FindByAPIKey(ctx context.Context, hashedKey string) (*domain.Seller, error) {
	var seller domain.Seller
	filter := bson.M{
		"apiKeys.hashedKey": hashedKey,
		"apiKeys.revokedAt": bson.M{"$exists": false},
	}

	err := r.collection.FindOne(ctx, filter).Decode(&seller)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return &seller, nil
}

// FindByEmail finds a seller by contact email
func (r *SellerRepository) FindByEmail(ctx context.Context, email string) (*domain.Seller, error) {
	var seller domain.Seller
	filter := bson.M{"contactEmail": email}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	err := r.collection.FindOne(ctx, filter).Decode(&seller)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return &seller, nil
}

// UpdateStatus updates the seller status with tenant scoping
func (r *SellerRepository) UpdateStatus(ctx context.Context, sellerID string, status domain.SellerStatus) error {
	filter := bson.M{"sellerId": sellerID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	update := bson.M{
		"$set": bson.M{
			"status":    status,
			"updatedAt": time.Now().UTC(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("seller not found")
	}

	return nil
}

// Delete deletes a seller (soft delete by setting status to closed)
func (r *SellerRepository) Delete(ctx context.Context, sellerID string) error {
	return r.UpdateStatus(ctx, sellerID, domain.SellerStatusClosed)
}

// Count returns the total number of sellers matching the filter
func (r *SellerRepository) Count(ctx context.Context, filter domain.SellerFilter) (int64, error) {
	mongoFilter := r.buildFilter(ctx, filter)
	return r.collection.CountDocuments(ctx, mongoFilter)
}

// Search searches sellers by company name or email
func (r *SellerRepository) Search(ctx context.Context, query string, pagination domain.Pagination) ([]*domain.Seller, error) {
	filter := bson.M{
		"$text": bson.M{"$search": query},
	}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	opts := options.Find().
		SetSort(bson.D{{Key: "score", Value: bson.M{"$meta": "textScore"}}}).
		SetSkip(pagination.Skip()).
		SetLimit(pagination.Limit())

	return r.findMany(ctx, filter, opts)
}

// findMany is a helper for finding multiple sellers
func (r *SellerRepository) findMany(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]*domain.Seller, error) {
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var sellers []*domain.Seller
	if err := cursor.All(ctx, &sellers); err != nil {
		return nil, err
	}

	return sellers, nil
}

// buildFilter builds a MongoDB filter from a SellerFilter
func (r *SellerRepository) buildFilter(ctx context.Context, filter domain.SellerFilter) bson.M {
	mongoFilter := bson.M{}

	if filter.TenantID != nil {
		mongoFilter["tenantId"] = *filter.TenantID
	}

	if filter.Status != nil {
		mongoFilter["status"] = *filter.Status
	}

	if filter.FacilityID != nil {
		mongoFilter["assignedFacilities.facilityId"] = *filter.FacilityID
	}

	if filter.HasChannel != nil {
		mongoFilter["integrations.channelType"] = *filter.HasChannel
	}

	// Apply tenant filtering
	return r.tenantHelper.WithTenantFilterOptional(ctx, mongoFilter)
}

// GetOutboxRepository returns the outbox repository for this service
func (r *SellerRepository) GetOutboxRepository() outbox.Repository {
	return r.outboxRepo
}
