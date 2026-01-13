package mongodb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/outbox"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/wms-platform/facility-service/internal/domain"
)

type fakeOutboxRepo struct {
	saveAllCalls int
	lastEvents   []*outbox.OutboxEvent
	saveAllErr   error
}

func (f *fakeOutboxRepo) Save(ctx context.Context, event *outbox.OutboxEvent) error {
	return nil
}

func (f *fakeOutboxRepo) SaveAll(ctx context.Context, events []*outbox.OutboxEvent) error {
	f.saveAllCalls++
	f.lastEvents = events
	return f.saveAllErr
}

func (f *fakeOutboxRepo) FindUnpublished(ctx context.Context, limit int) ([]*outbox.OutboxEvent, error) {
	return nil, nil
}

func (f *fakeOutboxRepo) MarkPublished(ctx context.Context, eventID string) error {
	return nil
}

func (f *fakeOutboxRepo) IncrementRetry(ctx context.Context, eventID string, errorMsg string) error {
	return nil
}

func (f *fakeOutboxRepo) DeletePublished(ctx context.Context, olderThan int64) error {
	return nil
}

func (f *fakeOutboxRepo) GetByID(ctx context.Context, eventID string) (*outbox.OutboxEvent, error) {
	return nil, nil
}

func (f *fakeOutboxRepo) FindByAggregateID(ctx context.Context, aggregateID string) ([]*outbox.OutboxEvent, error) {
	return nil, nil
}

type fakeIndexView struct{}

func (f fakeIndexView) CreateMany(ctx context.Context, models []mongo.IndexModel, opts ...*options.CreateIndexesOptions) ([]string, error) {
	return nil, nil
}

type fakeSingleResult struct {
	station   *domain.Station
	decodeErr error
}

func (f fakeSingleResult) Decode(v interface{}) error {
	if f.decodeErr != nil {
		return f.decodeErr
	}
	switch target := v.(type) {
	case *domain.Station:
		*target = *f.station
		return nil
	default:
		return fmt.Errorf("unexpected decode target %T", v)
	}
}

type fakeCursor struct {
	stations []*domain.Station
	allErr   error
	closed   bool
}

func (f *fakeCursor) All(ctx context.Context, results interface{}) error {
	if f.allErr != nil {
		return f.allErr
	}
	switch target := results.(type) {
	case *[]*domain.Station:
		*target = f.stations
		return nil
	default:
		return fmt.Errorf("unexpected results target %T", results)
	}
}

func (f *fakeCursor) Close(ctx context.Context) error {
	f.closed = true
	return nil
}

type fakeCollection struct {
	updateFilter interface{}
	updateDoc    interface{}
	updateErr    error
	updateOpts   []*options.UpdateOptions

	findOneFilter interface{}
	findOneResult mongoSingleResult

	findFilter interface{}
	findOpts   []*options.FindOptions
	findCursor mongoCursor
	findErr    error

	deleteFilter interface{}
	deleteErr    error
}

func (f *fakeCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	f.updateFilter = filter
	f.updateDoc = update
	f.updateOpts = opts
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	return &mongo.UpdateResult{}, nil
}

func (f *fakeCollection) FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResult {
	f.findOneFilter = filter
	return f.findOneResult
}

func (f *fakeCollection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursor, error) {
	f.findFilter = filter
	f.findOpts = opts
	if f.findErr != nil {
		return nil, f.findErr
	}
	return f.findCursor, nil
}

func (f *fakeCollection) DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	f.deleteFilter = filter
	if f.deleteErr != nil {
		return nil, f.deleteErr
	}
	return &mongo.DeleteResult{}, nil
}

func (f *fakeCollection) Indexes() mongoIndexView {
	return fakeIndexView{}
}

type fakeSession struct {
	transactionErr error
	endCalled      bool
}

func (f *fakeSession) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if err := fn(ctx); err != nil {
		return err
	}
	if f.transactionErr != nil {
		return f.transactionErr
	}
	return nil
}

func (f *fakeSession) EndSession(ctx context.Context) {
	f.endCalled = true
}

type fakeSessionClient struct {
	startErr error
	session  *fakeSession
}

func (f *fakeSessionClient) StartSession(opts ...*options.SessionOptions) (mongoSession, error) {
	if f.startErr != nil {
		return nil, f.startErr
	}
	if f.session == nil {
		f.session = &fakeSession{}
	}
	return f.session, nil
}

type fakeDatabase struct {
	collection mongoCollection
	client     *fakeSessionClient
}

func (f *fakeDatabase) Collection(name string, opts ...*options.CollectionOptions) mongoCollection {
	return f.collection
}

func (f *fakeDatabase) Client() mongoSessionClient {
	return f.client
}

func TestStationRepository_Save(t *testing.T) {
	t.Run("saves station and outbox events", func(t *testing.T) {
		station, err := domain.NewStation("STN-1", "Station 1", "A", domain.StationTypePacking, 2)
		if err != nil {
			t.Fatalf("NewStation error: %v", err)
		}

		collection := &fakeCollection{}
		outboxRepo := &fakeOutboxRepo{}
		db := &fakeDatabase{collection: collection, client: &fakeSessionClient{}}
		repo := newStationRepository(db, outboxRepo, cloudevents.NewEventFactory("/facility-service"))

		if err := repo.Save(context.Background(), station); err != nil {
			t.Fatalf("Save error: %v", err)
		}

		filter, ok := collection.updateFilter.(bson.M)
		if !ok || filter["stationId"] != "STN-1" {
			t.Fatalf("unexpected filter: %#v", collection.updateFilter)
		}
		if len(outboxRepo.lastEvents) != 1 || outboxRepo.saveAllCalls != 1 {
			t.Fatalf("expected 1 outbox event, got %d", len(outboxRepo.lastEvents))
		}
		if len(station.GetDomainEvents()) != 0 {
			t.Fatalf("expected domain events cleared")
		}
	})

	t.Run("no events skips outbox save", func(t *testing.T) {
		station, err := domain.NewStation("STN-2", "Station 2", "B", domain.StationTypePacking, 1)
		if err != nil {
			t.Fatalf("NewStation error: %v", err)
		}
		station.ClearDomainEvents()

		outboxRepo := &fakeOutboxRepo{}
		db := &fakeDatabase{collection: &fakeCollection{}, client: &fakeSessionClient{}}
		repo := newStationRepository(db, outboxRepo, cloudevents.NewEventFactory("/facility-service"))

		if err := repo.Save(context.Background(), station); err != nil {
			t.Fatalf("Save error: %v", err)
		}
		if outboxRepo.saveAllCalls != 0 {
			t.Fatalf("expected no outbox SaveAll calls, got %d", outboxRepo.saveAllCalls)
		}
	})

	t.Run("update error fails transaction", func(t *testing.T) {
		station, _ := domain.NewStation("STN-3", "Station 3", "C", domain.StationTypePacking, 1)
		collection := &fakeCollection{updateErr: errors.New("update failed")}
		db := &fakeDatabase{collection: collection, client: &fakeSessionClient{}}
		repo := newStationRepository(db, &fakeOutboxRepo{}, cloudevents.NewEventFactory("/facility-service"))

		err := repo.Save(context.Background(), station)
		if err == nil || !strings.Contains(err.Error(), "failed to save station") {
			t.Fatalf("expected save error, got %v", err)
		}
	})

	t.Run("outbox error fails transaction", func(t *testing.T) {
		station, _ := domain.NewStation("STN-4", "Station 4", "D", domain.StationTypePacking, 1)
		outboxRepo := &fakeOutboxRepo{saveAllErr: errors.New("outbox failed")}
		db := &fakeDatabase{collection: &fakeCollection{}, client: &fakeSessionClient{}}
		repo := newStationRepository(db, outboxRepo, cloudevents.NewEventFactory("/facility-service"))

		err := repo.Save(context.Background(), station)
		if err == nil || !strings.Contains(err.Error(), "failed to save outbox events") {
			t.Fatalf("expected outbox error, got %v", err)
		}
	})

	t.Run("start session error", func(t *testing.T) {
		station, _ := domain.NewStation("STN-5", "Station 5", "E", domain.StationTypePacking, 1)
		db := &fakeDatabase{
			collection: &fakeCollection{},
			client:     &fakeSessionClient{startErr: errors.New("session failed")},
		}
		repo := newStationRepository(db, &fakeOutboxRepo{}, cloudevents.NewEventFactory("/facility-service"))

		err := repo.Save(context.Background(), station)
		if err == nil || !strings.Contains(err.Error(), "failed to start session") {
			t.Fatalf("expected start session error, got %v", err)
		}
	})
}

func TestStationRepository_FindMethods(t *testing.T) {
	station := &domain.Station{StationID: "STN-1"}
	collection := &fakeCollection{
		findOneResult: fakeSingleResult{station: station},
	}
	db := &fakeDatabase{collection: collection, client: &fakeSessionClient{}}
	repo := newStationRepository(db, &fakeOutboxRepo{}, cloudevents.NewEventFactory("/facility-service"))

	found, err := repo.FindByID(context.Background(), "STN-1")
	if err != nil || found == nil || found.StationID != "STN-1" {
		t.Fatalf("FindByID failed: %v", err)
	}

	collection.findOneResult = fakeSingleResult{decodeErr: mongo.ErrNoDocuments}
	found, err = repo.FindByID(context.Background(), "missing")
	if err != nil || found != nil {
		t.Fatalf("FindByID missing expected nil, err=%v", err)
	}
}

func TestStationRepository_FindCapableStations(t *testing.T) {
	activeOK := &domain.Station{StationID: "A", Status: domain.StationStatusActive, MaxConcurrentTasks: 2, CurrentTasks: 1}
	activeFull := &domain.Station{StationID: "B", Status: domain.StationStatusActive, MaxConcurrentTasks: 1, CurrentTasks: 1}
	inactive := &domain.Station{StationID: "C", Status: domain.StationStatusInactive, MaxConcurrentTasks: 2, CurrentTasks: 0}
	cursor := &fakeCursor{stations: []*domain.Station{activeOK, activeFull, inactive}}

	collection := &fakeCollection{findCursor: cursor}
	db := &fakeDatabase{collection: collection, client: &fakeSessionClient{}}
	repo := newStationRepository(db, &fakeOutboxRepo{}, cloudevents.NewEventFactory("/facility-service"))

	stations, err := repo.FindCapableStations(context.Background(), []domain.StationCapability{"c1"}, domain.StationTypePacking, "Z1")
	if err != nil {
		t.Fatalf("FindCapableStations error: %v", err)
	}
	if len(stations) != 1 || stations[0].StationID != "A" {
		t.Fatalf("unexpected stations: %#v", stations)
	}
	filter, ok := collection.findFilter.(bson.M)
	if !ok || filter["status"] != domain.StationStatusActive {
		t.Fatalf("unexpected filter: %#v", collection.findFilter)
	}
}

func TestStationRepository_FindLists(t *testing.T) {
	cursor := &fakeCursor{stations: []*domain.Station{{StationID: "S1"}}}
	collection := &fakeCollection{findCursor: cursor}
	db := &fakeDatabase{collection: collection, client: &fakeSessionClient{}}
	repo := newStationRepository(db, &fakeOutboxRepo{}, cloudevents.NewEventFactory("/facility-service"))

	_, _ = repo.FindByZone(context.Background(), "Z")
	filter, _ := collection.findFilter.(bson.M)
	if filter["zone"] != "Z" {
		t.Fatalf("FindByZone filter: %#v", filter)
	}

	_, _ = repo.FindByType(context.Background(), domain.StationTypePacking)
	filter, _ = collection.findFilter.(bson.M)
	if filter["stationType"] != domain.StationTypePacking {
		t.Fatalf("FindByType filter: %#v", filter)
	}

	_, _ = repo.FindByStatus(context.Background(), domain.StationStatusActive)
	filter, _ = collection.findFilter.(bson.M)
	if filter["status"] != domain.StationStatusActive {
		t.Fatalf("FindByStatus filter: %#v", filter)
	}
}

func TestStationRepository_FindAllDeleteWorker(t *testing.T) {
	cursor := &fakeCursor{stations: []*domain.Station{{StationID: "S2"}}}
	collection := &fakeCollection{findCursor: cursor}
	db := &fakeDatabase{collection: collection, client: &fakeSessionClient{}}
	repo := newStationRepository(db, &fakeOutboxRepo{}, cloudevents.NewEventFactory("/facility-service"))

	_, _ = repo.FindAll(context.Background(), 10, 5)
	if len(collection.findOpts) == 0 || collection.findOpts[0].Limit == nil || collection.findOpts[0].Skip == nil {
		t.Fatalf("expected find options for limit/skip")
	}
	if *collection.findOpts[0].Limit != 10 || *collection.findOpts[0].Skip != 5 {
		t.Fatalf("unexpected find options: %#v", collection.findOpts[0])
	}

	collection.findOneResult = fakeSingleResult{decodeErr: mongo.ErrNoDocuments}
	station, err := repo.FindByWorkerID(context.Background(), "W1")
	if err != nil || station != nil {
		t.Fatalf("FindByWorkerID expected nil, err=%v", err)
	}

	if err := repo.Delete(context.Background(), "S2"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	filter, _ := collection.deleteFilter.(bson.M)
	if filter["stationId"] != "S2" {
		t.Fatalf("Delete filter: %#v", filter)
	}
}

func TestStationRepository_GetOutboxRepository(t *testing.T) {
	outboxRepo := &fakeOutboxRepo{}
	db := &fakeDatabase{collection: &fakeCollection{}, client: &fakeSessionClient{}}
	repo := newStationRepository(db, outboxRepo, cloudevents.NewEventFactory("/facility-service"))

	if repo.GetOutboxRepository() != outboxRepo {
		t.Fatalf("expected outbox repository passthrough")
	}
}
