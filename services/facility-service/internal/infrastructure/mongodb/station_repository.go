package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/facility-service/internal/domain"
	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/outbox"
	outboxMongo "github.com/wms-platform/shared/pkg/outbox/mongodb"
	"github.com/wms-platform/shared/pkg/tenant"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoCollection interface {
	UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error)
	FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResult
	Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursor, error)
	DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error)
	Indexes() mongoIndexView
}

type mongoSingleResult interface {
	Decode(v interface{}) error
}

type mongoCursor interface {
	All(ctx context.Context, results interface{}) error
	Close(ctx context.Context) error
}

type mongoIndexView interface {
	CreateMany(ctx context.Context, models []mongo.IndexModel, opts ...*options.CreateIndexesOptions) ([]string, error)
}

type mongoDatabase interface {
	Collection(name string, opts ...*options.CollectionOptions) mongoCollection
	Client() mongoSessionClient
}

type mongoSessionClient interface {
	StartSession(opts ...*options.SessionOptions) (mongoSession, error)
}

type mongoSession interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	EndSession(ctx context.Context)
}

type mongoDatabaseWrapper struct {
	db *mongo.Database
}

func (w mongoDatabaseWrapper) Collection(name string, opts ...*options.CollectionOptions) mongoCollection {
	return mongoCollectionWrapper{collection: w.db.Collection(name, opts...)}
}

func (w mongoDatabaseWrapper) Client() mongoSessionClient {
	return mongoClientWrapper{client: w.db.Client()}
}

type mongoCollectionWrapper struct {
	collection *mongo.Collection
}

func (w mongoCollectionWrapper) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	return w.collection.UpdateOne(ctx, filter, update, opts...)
}

func (w mongoCollectionWrapper) FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResult {
	return w.collection.FindOne(ctx, filter, opts...)
}

func (w mongoCollectionWrapper) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursor, error) {
	cursor, err := w.collection.Find(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}
	return mongoCursorWrapper{cursor: cursor}, nil
}

func (w mongoCollectionWrapper) DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	return w.collection.DeleteOne(ctx, filter, opts...)
}

func (w mongoCollectionWrapper) Indexes() mongoIndexView {
	return w.collection.Indexes()
}

type mongoCursorWrapper struct {
	cursor *mongo.Cursor
}

func (w mongoCursorWrapper) All(ctx context.Context, results interface{}) error {
	return w.cursor.All(ctx, results)
}

func (w mongoCursorWrapper) Close(ctx context.Context) error {
	return w.cursor.Close(ctx)
}

type mongoClientWrapper struct {
	client *mongo.Client
}

func (w mongoClientWrapper) StartSession(opts ...*options.SessionOptions) (mongoSession, error) {
	session, err := w.client.StartSession(opts...)
	if err != nil {
		return nil, err
	}
	return mongoSessionWrapper{session: session}, nil
}

type mongoSessionWrapper struct {
	session mongo.Session
}

func (w mongoSessionWrapper) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	_, err := w.session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		return nil, fn(sessCtx)
	})
	return err
}

func (w mongoSessionWrapper) EndSession(ctx context.Context) {
	w.session.EndSession(ctx)
}

type StationRepository struct {
	collection   mongoCollection
	db           mongoDatabase
	outboxRepo   outbox.Repository
	eventFactory *cloudevents.EventFactory
	tenantHelper *tenant.RepositoryHelper
}

func NewStationRepository(db *mongo.Database, eventFactory *cloudevents.EventFactory) *StationRepository {
	outboxRepo := outboxMongo.NewOutboxRepository(db)

	repo := newStationRepository(
		mongoDatabaseWrapper{db: db},
		outboxRepo,
		eventFactory,
	)
	repo.ensureIndexes(context.Background())

	// Create outbox indexes
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = outboxRepo.EnsureIndexes(ctx)

	return repo
}

func newStationRepository(db mongoDatabase, outboxRepo outbox.Repository, eventFactory *cloudevents.EventFactory) *StationRepository {
	return &StationRepository{
		collection:   db.Collection("stations"),
		db:           db,
		outboxRepo:   outboxRepo,
		eventFactory: eventFactory,
		tenantHelper: tenant.NewRepositoryHelper(false),
	}
}

func (r *StationRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "stationId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "zone", Value: 1}}},
		{Keys: bson.D{{Key: "stationType", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "capabilities", Value: 1}}},
		{Keys: bson.D{
			{Key: "capabilities", Value: 1},
			{Key: "status", Value: 1},
			{Key: "stationType", Value: 1},
		}},
		{Keys: bson.D{{Key: "assignedWorkerId", Value: 1}}},
	}
	r.collection.Indexes().CreateMany(ctx, indexes)
}

func (r *StationRepository) Save(ctx context.Context, station *domain.Station) error {
	station.UpdatedAt = time.Now()

	// Start a MongoDB session for transaction
	session, err := r.db.Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	// Execute transaction
	err = session.WithTransaction(ctx, func(sessCtx context.Context) error {
		// 1. Save the aggregate
		opts := options.Update().SetUpsert(true)
		filter := bson.M{"stationId": station.StationID}
		update := bson.M{"$set": station}

		if _, err := r.collection.UpdateOne(sessCtx, filter, update, opts); err != nil {
			return fmt.Errorf("failed to save station: %w", err)
		}

		// 2. Save domain events to outbox
		domainEvents := station.GetDomainEvents()
		if len(domainEvents) > 0 {
			outboxEvents := make([]*outbox.OutboxEvent, 0, len(domainEvents))

			for _, event := range domainEvents {
				// Convert domain event to CloudEvent
				var cloudEvent *cloudevents.WMSCloudEvent
				switch e := event.(type) {
				case *domain.StationCreatedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "station/"+e.StationID, e)
				case *domain.StationCapabilityAddedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "station/"+e.StationID, e)
				case *domain.StationCapabilityRemovedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "station/"+e.StationID, e)
				case *domain.StationCapabilitiesUpdatedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "station/"+e.StationID, e)
				case *domain.StationStatusChangedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "station/"+e.StationID, e)
				case *domain.WorkerAssignedToStationEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "station/"+e.StationID, e)
				default:
					continue
				}

				// Create outbox event from CloudEvent - publish to FacilityEvents topic
				outboxEvent, err := outbox.NewOutboxEventFromCloudEvent(
					station.StationID,
					"Station",
					kafka.Topics.FacilityEvents,
					cloudEvent,
				)
				if err != nil {
					return fmt.Errorf("failed to create outbox event: %w", err)
				}

				outboxEvents = append(outboxEvents, outboxEvent)
			}

			// Save all outbox events in the same transaction
			if len(outboxEvents) > 0 {
				if err := r.outboxRepo.SaveAll(sessCtx, outboxEvents); err != nil {
					return fmt.Errorf("failed to save outbox events: %w", err)
				}
			}
		}

		// 3. Clear domain events from the aggregate
		station.ClearDomainEvents()

		return nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

func (r *StationRepository) FindByID(ctx context.Context, stationID string) (*domain.Station, error) {
	filter := bson.M{"stationId": stationID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	var station domain.Station
	err := r.collection.FindOne(ctx, filter).Decode(&station)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &station, err
}

func (r *StationRepository) FindByZone(ctx context.Context, zone string) ([]*domain.Station, error) {
	filter := bson.M{"zone": zone}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var stations []*domain.Station
	err = cursor.All(ctx, &stations)
	return stations, err
}

func (r *StationRepository) FindByType(ctx context.Context, stationType domain.StationType) ([]*domain.Station, error) {
	filter := bson.M{"stationType": stationType}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var stations []*domain.Station
	err = cursor.All(ctx, &stations)
	return stations, err
}

func (r *StationRepository) FindByStatus(ctx context.Context, status domain.StationStatus) ([]*domain.Station, error) {
	filter := bson.M{"status": status}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var stations []*domain.Station
	err = cursor.All(ctx, &stations)
	return stations, err
}

// FindCapableStations finds active stations that have ALL required capabilities
// This is the key query for process path routing
func (r *StationRepository) FindCapableStations(ctx context.Context, requirements []domain.StationCapability, stationType domain.StationType, zone string) ([]*domain.Station, error) {
	filter := bson.M{
		"status": domain.StationStatusActive,
	}

	// Filter by station type if specified
	if stationType != "" {
		filter["stationType"] = stationType
	}

	// Filter by zone if specified
	if zone != "" {
		filter["zone"] = zone
	}

	// Filter by capabilities - station must have ALL required capabilities
	if len(requirements) > 0 {
		filter["capabilities"] = bson.M{
			"$all": requirements,
		}
	}

	// Add tenant filtering
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	// Sort by available capacity (stations with more capacity first)
	opts := options.Find().SetSort(bson.D{
		{Key: "currentTasks", Value: 1}, // Least loaded first
	})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var stations []*domain.Station
	err = cursor.All(ctx, &stations)
	if err != nil {
		return nil, err
	}

	// Additional filter: only return stations that can accept tasks
	capable := make([]*domain.Station, 0)
	for _, station := range stations {
		if station.CanAcceptTask() {
			capable = append(capable, station)
		}
	}

	return capable, nil
}

// FindByCapability finds stations that have a specific capability
func (r *StationRepository) FindByCapability(ctx context.Context, capability domain.StationCapability) ([]*domain.Station, error) {
	filter := bson.M{
		"capabilities": capability,
		"status":       domain.StationStatusActive,
	}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var stations []*domain.Station
	err = cursor.All(ctx, &stations)
	return stations, err
}

// FindByWorkerID finds the station assigned to a specific worker
func (r *StationRepository) FindByWorkerID(ctx context.Context, workerID string) (*domain.Station, error) {
	filter := bson.M{"assignedWorkerId": workerID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	var station domain.Station
	err := r.collection.FindOne(ctx, filter).Decode(&station)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &station, err
}

func (r *StationRepository) FindAll(ctx context.Context, limit, offset int) ([]*domain.Station, error) {
	filter := bson.M{}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset))
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var stations []*domain.Station
	err = cursor.All(ctx, &stations)
	return stations, err
}

func (r *StationRepository) Delete(ctx context.Context, stationID string) error {
	filter := bson.M{"stationId": stationID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	_, err := r.collection.DeleteOne(ctx, filter)
	return err
}

// GetOutboxRepository returns the outbox repository for this service
func (r *StationRepository) GetOutboxRepository() outbox.Repository {
	return r.outboxRepo
}
