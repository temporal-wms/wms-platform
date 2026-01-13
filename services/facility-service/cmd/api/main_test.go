package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/idempotency"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/metrics"
	"github.com/wms-platform/shared/pkg/middleware"
	"github.com/wms-platform/shared/pkg/mongodb"
	"github.com/wms-platform/shared/pkg/outbox"
	"github.com/wms-platform/shared/pkg/tracing"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/wms-platform/facility-service/internal/api/handlers"
	"github.com/wms-platform/facility-service/internal/application"
	"github.com/wms-platform/facility-service/internal/domain"
)

func TestGetEnv(t *testing.T) {
	t.Setenv("FACILITY_TEST_ENV", "value")

	if got := getEnv("FACILITY_TEST_ENV", "default"); got != "value" {
		t.Fatalf("getEnv returned %q", got)
	}
	if got := getEnv("FACILITY_MISSING_ENV", "default"); got != "default" {
		t.Fatalf("getEnv default returned %q", got)
	}
}

func TestLoadConfig(t *testing.T) {
	t.Setenv("SERVER_ADDR", ":9000")
	t.Setenv("MONGODB_URI", "mongodb://example:27017")
	t.Setenv("MONGODB_DATABASE", "facility_test")
	t.Setenv("KAFKA_BROKERS", "kafka:9092")

	cfg := loadConfig()

	if cfg.ServerAddr != ":9000" {
		t.Fatalf("ServerAddr = %q", cfg.ServerAddr)
	}
	if cfg.MongoDB.URI != "mongodb://example:27017" || cfg.MongoDB.Database != "facility_test" {
		t.Fatalf("MongoDB config = %#v", cfg.MongoDB)
	}
	if cfg.MongoDB.ConnectTimeout != 10*time.Second || cfg.MongoDB.MaxPoolSize != 100 || cfg.MongoDB.MinPoolSize != 10 {
		t.Fatalf("MongoDB defaults unexpected: %#v", cfg.MongoDB)
	}
	if len(cfg.Kafka.Brokers) != 1 || cfg.Kafka.Brokers[0] != "kafka:9092" {
		t.Fatalf("Kafka brokers = %#v", cfg.Kafka.Brokers)
	}
}

type fakeTracerProvider struct {
	shutdownCalls int
}

func (f *fakeTracerProvider) Shutdown(ctx context.Context) error {
	f.shutdownCalls++
	return nil
}

type fakeInstrumentedMongo struct {
	closeCalls  int
	healthCalls int
}

func (f *fakeInstrumentedMongo) Database() *mongo.Database {
	return nil
}

func (f *fakeInstrumentedMongo) Close(ctx context.Context) error {
	f.closeCalls++
	return nil
}

func (f *fakeInstrumentedMongo) HealthCheck(ctx context.Context) error {
	f.healthCalls++
	return nil
}

type fakeOutboxPublisher struct {
	startCalls int
	stopCalls  int
	startErr   error
}

func (f *fakeOutboxPublisher) Start(ctx context.Context) error {
	f.startCalls++
	return f.startErr
}

func (f *fakeOutboxPublisher) Stop() error {
	f.stopCalls++
	return nil
}

type fakeServer struct {
	listenCalls   int
	shutdownCalls int
	listenErr     error
}

func (f *fakeServer) ListenAndServe() error {
	f.listenCalls++
	if f.listenErr != nil {
		return f.listenErr
	}
	return http.ErrServerClosed
}

func (f *fakeServer) Shutdown(ctx context.Context) error {
	f.shutdownCalls++
	return nil
}

type fakeOutboxRepo struct{}

func (f *fakeOutboxRepo) Save(ctx context.Context, event *outbox.OutboxEvent) error {
	return nil
}

func (f *fakeOutboxRepo) SaveAll(ctx context.Context, events []*outbox.OutboxEvent) error {
	return nil
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

type fakeStationRepo struct {
	outboxRepo outbox.Repository
}

func (f *fakeStationRepo) Save(ctx context.Context, station *domain.Station) error {
	return nil
}

func (f *fakeStationRepo) FindByID(ctx context.Context, stationID string) (*domain.Station, error) {
	return nil, nil
}

func (f *fakeStationRepo) FindByZone(ctx context.Context, zone string) ([]*domain.Station, error) {
	return nil, nil
}

func (f *fakeStationRepo) FindByType(ctx context.Context, stationType domain.StationType) ([]*domain.Station, error) {
	return nil, nil
}

func (f *fakeStationRepo) FindByStatus(ctx context.Context, status domain.StationStatus) ([]*domain.Station, error) {
	return nil, nil
}

func (f *fakeStationRepo) FindCapableStations(ctx context.Context, requirements []domain.StationCapability, stationType domain.StationType, zone string) ([]*domain.Station, error) {
	return nil, nil
}

func (f *fakeStationRepo) FindByCapability(ctx context.Context, capability domain.StationCapability) ([]*domain.Station, error) {
	return nil, nil
}

func (f *fakeStationRepo) FindByWorkerID(ctx context.Context, workerID string) (*domain.Station, error) {
	return nil, nil
}

func (f *fakeStationRepo) FindAll(ctx context.Context, limit, offset int) ([]*domain.Station, error) {
	return nil, nil
}

func (f *fakeStationRepo) Delete(ctx context.Context, stationID string) error {
	return nil
}

func (f *fakeStationRepo) GetOutboxRepository() outbox.Repository {
	return f.outboxRepo
}

type fakeKeyRepo struct{}

func (f fakeKeyRepo) AcquireLock(ctx context.Context, key *idempotency.IdempotencyKey) (*idempotency.IdempotencyKey, bool, error) {
	return nil, false, nil
}

func (f fakeKeyRepo) ReleaseLock(ctx context.Context, keyID string) error {
	return nil
}

func (f fakeKeyRepo) StoreResponse(ctx context.Context, keyID string, responseCode int, responseBody []byte, headers map[string]string) error {
	return nil
}

func (f fakeKeyRepo) UpdateRecoveryPoint(ctx context.Context, keyID string, phase string) error {
	return nil
}

func (f fakeKeyRepo) Get(ctx context.Context, key, serviceID string) (*idempotency.IdempotencyKey, error) {
	return nil, nil
}

func (f fakeKeyRepo) GetByID(ctx context.Context, keyID string) (*idempotency.IdempotencyKey, error) {
	return nil, nil
}

func (f fakeKeyRepo) Clean(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

func (f fakeKeyRepo) EnsureIndexes(ctx context.Context) error {
	return nil
}

type fakeStationService struct{}

func (f fakeStationService) CreateStation(ctx context.Context, cmd application.CreateStationCommand) (*application.StationDTO, error) {
	return &application.StationDTO{}, nil
}

func (f fakeStationService) GetStation(ctx context.Context, query application.GetStationQuery) (*application.StationDTO, error) {
	return &application.StationDTO{}, nil
}

func (f fakeStationService) UpdateStation(ctx context.Context, cmd application.UpdateStationCommand) (*application.StationDTO, error) {
	return &application.StationDTO{}, nil
}

func (f fakeStationService) DeleteStation(ctx context.Context, cmd application.DeleteStationCommand) error {
	return nil
}

func (f fakeStationService) SetCapabilities(ctx context.Context, cmd application.SetCapabilitiesCommand) (*application.StationDTO, error) {
	return &application.StationDTO{}, nil
}

func (f fakeStationService) AddCapability(ctx context.Context, cmd application.AddCapabilityCommand) (*application.StationDTO, error) {
	return &application.StationDTO{}, nil
}

func (f fakeStationService) RemoveCapability(ctx context.Context, cmd application.RemoveCapabilityCommand) (*application.StationDTO, error) {
	return &application.StationDTO{}, nil
}

func (f fakeStationService) SetStatus(ctx context.Context, cmd application.SetStationStatusCommand) (*application.StationDTO, error) {
	return &application.StationDTO{}, nil
}

func (f fakeStationService) FindCapableStations(ctx context.Context, query application.FindCapableStationsQuery) ([]application.StationDTO, error) {
	return nil, nil
}

func (f fakeStationService) ListStations(ctx context.Context, query application.ListStationsQuery) ([]application.StationDTO, error) {
	return nil, nil
}

func (f fakeStationService) GetByZone(ctx context.Context, query application.GetStationsByZoneQuery) ([]application.StationDTO, error) {
	return nil, nil
}

func (f fakeStationService) GetByType(ctx context.Context, query application.GetStationsByTypeQuery) ([]application.StationDTO, error) {
	return nil, nil
}

func (f fakeStationService) GetByStatus(ctx context.Context, query application.GetStationsByStatusQuery) ([]application.StationDTO, error) {
	return nil, nil
}

func TestRunSuccess(t *testing.T) {
	tracer := &fakeTracerProvider{}
	fakeMongo := &fakeInstrumentedMongo{}
	publisher := &fakeOutboxPublisher{}
	server := &fakeServer{}
	repo := &fakeStationRepo{outboxRepo: &fakeOutboxRepo{}}

	var idempotencyCalls int
	var producerCloseCalls int

	deps := appDependencies{
		initTracing: func(ctx context.Context, cfg *tracing.Config) (tracerProvider, error) {
			return tracer, nil
		},
		newMetrics: func(cfg *metrics.Config) *metrics.Metrics {
			return metrics.New(cfg)
		},
		newBusinessMetrics: func(m *metrics.Metrics) *middleware.BusinessMetrics {
			return middleware.NewBusinessMetrics(m)
		},
		newMongoClient: func(ctx context.Context, cfg *mongodb.Config) (*mongodb.Client, error) {
			return nil, nil
		},
		newInstrumentedMongo: func(client *mongodb.Client, m *metrics.Metrics, logger *logging.Logger) instrumentedMongo {
			return fakeMongo
		},
		initIdempotencyIndexes: func(ctx context.Context, db *mongo.Database) error {
			idempotencyCalls++
			return nil
		},
		newKafkaProducer: func(cfg *kafka.Config) *kafka.Producer {
			return kafka.NewProducer(cfg)
		},
		newInstrumentedProducer: func(p *kafka.Producer, m *metrics.Metrics, logger *logging.Logger) *kafka.InstrumentedProducer {
			return kafka.NewInstrumentedProducer(p, m, logger)
		},
		closeInstrumentedProd: func(p *kafka.InstrumentedProducer) error {
			producerCloseCalls++
			return nil
		},
		newEventFactory: func(source string) *cloudevents.EventFactory {
			return cloudevents.NewEventFactory(source)
		},
		newStationRepository: func(db *mongo.Database, factory *cloudevents.EventFactory) stationRepository {
			return repo
		},
		newIdempotencyKeyRepo: func(db *mongo.Database) idempotency.KeyRepository {
			return fakeKeyRepo{}
		},
		newOutboxPublisher: func(repo outbox.Repository, producer *kafka.InstrumentedProducer, logger *logging.Logger, m *metrics.Metrics, cfg *outbox.PublisherConfig) outboxPublisher {
			return publisher
		},
		newStationService: func(repo application.StationRepository, producer *kafka.InstrumentedProducer, factory *cloudevents.EventFactory, logger *logging.Logger) handlers.StationService {
			return fakeStationService{}
		},
		newHTTPServer: func(addr string, handler http.Handler) httpServer {
			return server
		},
	}

	signalCh := make(chan os.Signal, 1)
	signalCh <- syscall.SIGTERM

	if err := run(context.Background(), loadConfig(), deps, signalCh); err != nil {
		t.Fatalf("run error: %v", err)
	}

	if tracer.shutdownCalls != 1 {
		t.Fatalf("tracer shutdown calls = %d", tracer.shutdownCalls)
	}
	if fakeMongo.closeCalls != 1 {
		t.Fatalf("mongo close calls = %d", fakeMongo.closeCalls)
	}
	if idempotencyCalls != 1 {
		t.Fatalf("idempotency index calls = %d", idempotencyCalls)
	}
	if publisher.startCalls != 1 || publisher.stopCalls != 1 {
		t.Fatalf("outbox start/stop calls = %d/%d", publisher.startCalls, publisher.stopCalls)
	}
	if server.listenCalls != 1 || server.shutdownCalls != 1 {
		t.Fatalf("server listen/shutdown calls = %d/%d", server.listenCalls, server.shutdownCalls)
	}
	if producerCloseCalls != 1 {
		t.Fatalf("producer close calls = %d", producerCloseCalls)
	}
}

func TestRunOutboxStartError(t *testing.T) {
	deps := appDependencies{
		initTracing: func(ctx context.Context, cfg *tracing.Config) (tracerProvider, error) {
			return &fakeTracerProvider{}, nil
		},
		newMetrics:         metrics.New,
		newBusinessMetrics: middleware.NewBusinessMetrics,
		newMongoClient: func(ctx context.Context, cfg *mongodb.Config) (*mongodb.Client, error) {
			return nil, nil
		},
		newInstrumentedMongo: func(client *mongodb.Client, m *metrics.Metrics, logger *logging.Logger) instrumentedMongo {
			return &fakeInstrumentedMongo{}
		},
		initIdempotencyIndexes: func(ctx context.Context, db *mongo.Database) error {
			return nil
		},
		newKafkaProducer: func(cfg *kafka.Config) *kafka.Producer {
			return nil
		},
		newInstrumentedProducer: func(p *kafka.Producer, m *metrics.Metrics, logger *logging.Logger) *kafka.InstrumentedProducer {
			return nil
		},
		closeInstrumentedProd: func(p *kafka.InstrumentedProducer) error {
			return nil
		},
		newEventFactory: func(source string) *cloudevents.EventFactory {
			return cloudevents.NewEventFactory(source)
		},
		newStationRepository: func(db *mongo.Database, factory *cloudevents.EventFactory) stationRepository {
			return &fakeStationRepo{outboxRepo: &fakeOutboxRepo{}}
		},
		newIdempotencyKeyRepo: func(db *mongo.Database) idempotency.KeyRepository {
			return fakeKeyRepo{}
		},
		newOutboxPublisher: func(repo outbox.Repository, producer *kafka.InstrumentedProducer, logger *logging.Logger, m *metrics.Metrics, cfg *outbox.PublisherConfig) outboxPublisher {
			return &fakeOutboxPublisher{startErr: errors.New("start failed")}
		},
		newStationService: func(repo application.StationRepository, producer *kafka.InstrumentedProducer, factory *cloudevents.EventFactory, logger *logging.Logger) handlers.StationService {
			return fakeStationService{}
		},
		newHTTPServer: func(addr string, handler http.Handler) httpServer {
			return &fakeServer{}
		},
	}

	if err := run(context.Background(), loadConfig(), deps, make(chan os.Signal, 1)); err == nil {
		t.Fatalf("expected outbox start error")
	}
}

func TestRunMongoClientError(t *testing.T) {
	deps := appDependencies{
		initTracing: func(ctx context.Context, cfg *tracing.Config) (tracerProvider, error) {
			return &fakeTracerProvider{}, nil
		},
		newMetrics:         metrics.New,
		newBusinessMetrics: middleware.NewBusinessMetrics,
		newMongoClient: func(ctx context.Context, cfg *mongodb.Config) (*mongodb.Client, error) {
			return nil, errors.New("mongo failed")
		},
	}

	if err := run(context.Background(), loadConfig(), deps, make(chan os.Signal, 1)); err == nil {
		t.Fatalf("expected mongo client error")
	}
}
