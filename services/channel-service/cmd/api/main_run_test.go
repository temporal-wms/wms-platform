package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wms-platform/services/channel-service/internal/domain"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/metrics"
	"github.com/wms-platform/shared/pkg/mongodb"
	"github.com/wms-platform/shared/pkg/outbox"
	"github.com/wms-platform/shared/pkg/tracing"
	"go.mongodb.org/mongo-driver/mongo"
)

type fakeMongo struct {
	closed bool
	mu     sync.Mutex
}

func (f *fakeMongo) Database() *mongo.Database {
	return nil
}

func (f *fakeMongo) Close(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

func (f *fakeMongo) HealthCheck(context.Context) error {
	return nil
}

type fakePublisher struct {
	started bool
	stopped bool
}

func (f *fakePublisher) Start(context.Context) error {
	f.started = true
	return nil
}

func (f *fakePublisher) Stop() error {
	f.stopped = true
	return nil
}

type fakePublisherStartError struct{}

func (f *fakePublisherStartError) Start(context.Context) error {
	return errors.New("start failed")
}

func (f *fakePublisherStartError) Stop() error {
	return nil
}

type fakeServer struct {
	started  bool
	shutdown bool
}

func (f *fakeServer) ListenAndServe() error {
	f.started = true
	return http.ErrServerClosed
}

func (f *fakeServer) Shutdown(context.Context) error {
	f.shutdown = true
	return nil
}

type fakeTracerProvider struct {
	shutdown bool
}

func (f *fakeTracerProvider) Shutdown(context.Context) error {
	f.shutdown = true
	return nil
}

func stubRunDeps() (func(), *fakeMongo, *fakeServer) {
	origNewMongoClient := newMongoClient
	origNewInstrumentedMongo := newInstrumentedMongo
	origNewChannelRepository := newChannelRepository
	origNewChannelOrderRepository := newChannelOrderRepository
	origNewSyncJobRepository := newSyncJobRepository
	origNewOutboxRepository := newOutboxRepository
	origNewServer := newServer

	fakeMongoClient := &fakeMongo{}
	fakeSrv := &fakeServer{}

	newMongoClient = func(context.Context, *mongodb.Config) (*mongodb.Client, error) {
		return nil, nil
	}
	newInstrumentedMongo = func(*mongodb.Client, *metrics.Metrics, *logging.Logger) mongoClient {
		return fakeMongoClient
	}
	newChannelRepository = func(*mongo.Database) domain.ChannelRepository {
		return &fakeChannelRepo{}
	}
	newChannelOrderRepository = func(*mongo.Database) domain.ChannelOrderRepository {
		return &fakeOrderRepo{}
	}
	newSyncJobRepository = func(*mongo.Database) domain.SyncJobRepository {
		return &fakeSyncJobRepo{}
	}
	newOutboxRepository = func(*mongo.Database) outbox.Repository {
		return &fakeOutboxRepo{}
	}
	newServer = func(string, http.Handler) server {
		return fakeSrv
	}

	return func() {
		newMongoClient = origNewMongoClient
		newInstrumentedMongo = origNewInstrumentedMongo
		newChannelRepository = origNewChannelRepository
		newChannelOrderRepository = origNewChannelOrderRepository
		newSyncJobRepository = origNewSyncJobRepository
		newOutboxRepository = origNewOutboxRepository
		newServer = origNewServer
	}, fakeMongoClient, fakeSrv
}

type fakeChannelRepo struct{}

func (f *fakeChannelRepo) Save(context.Context, *domain.Channel) error                      { return nil }
func (f *fakeChannelRepo) FindByID(context.Context, string) (*domain.Channel, error)      { return nil, nil }
func (f *fakeChannelRepo) FindBySellerID(context.Context, string) ([]*domain.Channel, error) {
	return nil, nil
}
func (f *fakeChannelRepo) FindByType(context.Context, domain.ChannelType) ([]*domain.Channel, error) {
	return nil, nil
}
func (f *fakeChannelRepo) FindActiveChannels(context.Context) ([]*domain.Channel, error) { return nil, nil }
func (f *fakeChannelRepo) FindChannelsNeedingSync(context.Context, domain.SyncType, time.Duration) ([]*domain.Channel, error) {
	return nil, nil
}
func (f *fakeChannelRepo) UpdateStatus(context.Context, string, domain.ChannelStatus) error { return nil }
func (f *fakeChannelRepo) Delete(context.Context, string) error                              { return nil }

type fakeOrderRepo struct{}

func (f *fakeOrderRepo) Save(context.Context, *domain.ChannelOrder) error                          { return nil }
func (f *fakeOrderRepo) SaveAll(context.Context, []*domain.ChannelOrder) error                    { return nil }
func (f *fakeOrderRepo) FindByExternalID(context.Context, string, string) (*domain.ChannelOrder, error) {
	return nil, nil
}
func (f *fakeOrderRepo) FindByChannelID(context.Context, string, domain.Pagination) ([]*domain.ChannelOrder, error) {
	return nil, nil
}
func (f *fakeOrderRepo) FindUnimported(context.Context, string) ([]*domain.ChannelOrder, error) {
	return nil, nil
}
func (f *fakeOrderRepo) FindWithoutTracking(context.Context, string) ([]*domain.ChannelOrder, error) {
	return nil, nil
}
func (f *fakeOrderRepo) MarkImported(context.Context, string, string) error { return nil }
func (f *fakeOrderRepo) MarkTrackingPushed(context.Context, string) error   { return nil }
func (f *fakeOrderRepo) Count(context.Context, string) (int64, error)       { return 0, nil }

type fakeSyncJobRepo struct{}

func (f *fakeSyncJobRepo) Save(context.Context, *domain.SyncJob) error                         { return nil }
func (f *fakeSyncJobRepo) FindByID(context.Context, string) (*domain.SyncJob, error)          { return nil, nil }
func (f *fakeSyncJobRepo) FindByChannelID(context.Context, string, domain.Pagination) ([]*domain.SyncJob, error) {
	return nil, nil
}
func (f *fakeSyncJobRepo) FindRunning(context.Context, string, domain.SyncType) (*domain.SyncJob, error) {
	return nil, nil
}
func (f *fakeSyncJobRepo) FindLatest(context.Context, string, domain.SyncType) (*domain.SyncJob, error) {
	return nil, nil
}

type fakeOutboxRepo struct{}

func (f *fakeOutboxRepo) Save(context.Context, *outbox.OutboxEvent) error                      { return nil }
func (f *fakeOutboxRepo) SaveAll(context.Context, []*outbox.OutboxEvent) error                 { return nil }
func (f *fakeOutboxRepo) FindUnpublished(context.Context, int) ([]*outbox.OutboxEvent, error)  { return nil, nil }
func (f *fakeOutboxRepo) MarkPublished(context.Context, string) error                          { return nil }
func (f *fakeOutboxRepo) IncrementRetry(context.Context, string, string) error                 { return nil }
func (f *fakeOutboxRepo) DeletePublished(context.Context, int64) error                         { return nil }
func (f *fakeOutboxRepo) GetByID(context.Context, string) (*outbox.OutboxEvent, error)         { return nil, nil }
func (f *fakeOutboxRepo) FindByAggregateID(context.Context, string) ([]*outbox.OutboxEvent, error) {
	return nil, nil
}

func TestRunSuccess(t *testing.T) {
	restoreDeps, fakeMongoClient, fakeSrv := stubRunDeps()
	origNewOutboxPublisher := newOutboxPublisher
	origInitializeTracing := initializeTracing

	defer func() {
		restoreDeps()
		newOutboxPublisher = origNewOutboxPublisher
		initializeTracing = origInitializeTracing
	}()

	fakePub := &fakePublisher{}
	newOutboxPublisher = func(outbox.Repository, *kafka.InstrumentedProducer, *logging.Logger, *metrics.Metrics, *outbox.PublisherConfig) outboxPublisher {
		return fakePub
	}

	fakeTracer := &fakeTracerProvider{}
	initializeTracing = func(context.Context, *tracing.Config) (tracerProvider, error) {
		return fakeTracer, nil
	}

	quit := make(chan os.Signal, 1)
	quit <- syscall.SIGTERM

	err := run(context.Background(), quit)
	require.NoError(t, err)
	require.True(t, fakePub.started)
	require.True(t, fakePub.stopped)
	require.True(t, fakeSrv.shutdown)
	require.True(t, fakeMongoClient.closed)
	require.True(t, fakeTracer.shutdown)
}

func TestRunMongoError(t *testing.T) {
	origNewMongoClient := newMongoClient
	defer func() {
		newMongoClient = origNewMongoClient
	}()

	newMongoClient = func(context.Context, *mongodb.Config) (*mongodb.Client, error) {
		return nil, errors.New("boom")
	}

	quit := make(chan os.Signal, 1)
	err := run(context.Background(), quit)
	require.Error(t, err)
}

func TestRunTracingError(t *testing.T) {
	restoreDeps, _, _ := stubRunDeps()
	origInitializeTracing := initializeTracing
	defer func() {
		restoreDeps()
		initializeTracing = origInitializeTracing
	}()

	initializeTracing = func(context.Context, *tracing.Config) (tracerProvider, error) {
		return nil, errors.New("trace init failed")
	}

	quit := make(chan os.Signal, 1)
	quit <- syscall.SIGTERM

	err := run(context.Background(), quit)
	require.NoError(t, err)
}

func TestRunOutboxStartError(t *testing.T) {
	restoreDeps, _, _ := stubRunDeps()
	origNewOutboxPublisher := newOutboxPublisher
	defer func() {
		restoreDeps()
		newOutboxPublisher = origNewOutboxPublisher
	}()

	newOutboxPublisher = func(outbox.Repository, *kafka.InstrumentedProducer, *logging.Logger, *metrics.Metrics, *outbox.PublisherConfig) outboxPublisher {
		return &fakePublisherStartError{}
	}

	quit := make(chan os.Signal, 1)
	quit <- syscall.SIGTERM

	err := run(context.Background(), quit)
	require.NoError(t, err)
}
