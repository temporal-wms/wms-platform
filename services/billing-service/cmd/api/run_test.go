package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/wms-platform/services/billing-service/internal/domain"
	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/metrics"
	"github.com/wms-platform/shared/pkg/mongodb"
	"github.com/wms-platform/shared/pkg/outbox"
	"github.com/wms-platform/shared/pkg/tracing"
)

type fakeMongo struct{}

func (f *fakeMongo) Database() *mongo.Database { return nil }
func (f *fakeMongo) Close(context.Context) error { return nil }
func (f *fakeMongo) HealthCheck(context.Context) error { return nil }

type fakeProducer struct{}

func (f *fakeProducer) Close() error { return nil }

type fakeOutboxPublisher struct {
	startErr error
	stopErr  error
	started  *bool
	stopped  *bool
}

func (f *fakeOutboxPublisher) Start(context.Context) error {
	if f.started != nil {
		*f.started = true
	}
	return f.startErr
}

func (f *fakeOutboxPublisher) Stop() error {
	if f.stopped != nil {
		*f.stopped = true
	}
	return f.stopErr
}

type fakeOutboxRepo struct{}

func (f *fakeOutboxRepo) Save(context.Context, *outbox.OutboxEvent) error { return nil }
func (f *fakeOutboxRepo) SaveAll(context.Context, []*outbox.OutboxEvent) error { return nil }
func (f *fakeOutboxRepo) FindUnpublished(context.Context, int) ([]*outbox.OutboxEvent, error) {
	return nil, nil
}
func (f *fakeOutboxRepo) MarkPublished(context.Context, string) error { return nil }
func (f *fakeOutboxRepo) IncrementRetry(context.Context, string, string) error { return nil }
func (f *fakeOutboxRepo) DeletePublished(context.Context, int64) error { return nil }
func (f *fakeOutboxRepo) GetByID(context.Context, string) (*outbox.OutboxEvent, error) {
	return nil, nil
}
func (f *fakeOutboxRepo) FindByAggregateID(context.Context, string) ([]*outbox.OutboxEvent, error) {
	return nil, nil
}

type fakeInvoiceRepo struct {
	outboxRepo outbox.Repository
}

func (f *fakeInvoiceRepo) GetOutboxRepository() outbox.Repository { return f.outboxRepo }
func (f *fakeInvoiceRepo) Save(context.Context, *domain.Invoice) error { return nil }
func (f *fakeInvoiceRepo) FindByID(context.Context, string) (*domain.Invoice, error) { return nil, nil }
func (f *fakeInvoiceRepo) FindBySellerID(context.Context, string, domain.Pagination) ([]*domain.Invoice, error) {
	return nil, nil
}
func (f *fakeInvoiceRepo) FindByStatus(context.Context, domain.InvoiceStatus, domain.Pagination) ([]*domain.Invoice, error) {
	return nil, nil
}
func (f *fakeInvoiceRepo) FindOverdue(context.Context) ([]*domain.Invoice, error) { return nil, nil }
func (f *fakeInvoiceRepo) FindByPeriod(context.Context, string, time.Time, time.Time) (*domain.Invoice, error) {
	return nil, nil
}
func (f *fakeInvoiceRepo) UpdateStatus(context.Context, string, domain.InvoiceStatus) error { return nil }
func (f *fakeInvoiceRepo) Count(context.Context, domain.InvoiceFilter) (int64, error) { return 0, nil }

type fakeActivityRepo struct{}

func (f *fakeActivityRepo) Save(context.Context, *domain.BillableActivity) error { return nil }
func (f *fakeActivityRepo) SaveAll(context.Context, []*domain.BillableActivity) error { return nil }
func (f *fakeActivityRepo) FindByID(context.Context, string) (*domain.BillableActivity, error) { return nil, nil }
func (f *fakeActivityRepo) FindBySellerID(context.Context, string, domain.Pagination) ([]*domain.BillableActivity, error) {
	return nil, nil
}
func (f *fakeActivityRepo) FindUninvoiced(context.Context, string, time.Time, time.Time) ([]*domain.BillableActivity, error) {
	return nil, nil
}
func (f *fakeActivityRepo) FindByInvoiceID(context.Context, string) ([]*domain.BillableActivity, error) {
	return nil, nil
}
func (f *fakeActivityRepo) MarkAsInvoiced(context.Context, []string, string) error { return nil }
func (f *fakeActivityRepo) SumBySellerAndType(context.Context, string, time.Time, time.Time) (map[domain.ActivityType]float64, error) {
	return nil, nil
}
func (f *fakeActivityRepo) Count(context.Context, domain.ActivityFilter) (int64, error) { return 0, nil }

type fakeStorageRepo struct{}

func (f *fakeStorageRepo) Save(context.Context, *domain.StorageCalculation) error { return nil }
func (f *fakeStorageRepo) FindBySellerAndDate(context.Context, string, time.Time) (*domain.StorageCalculation, error) {
	return nil, nil
}
func (f *fakeStorageRepo) FindBySellerAndPeriod(context.Context, string, time.Time, time.Time) ([]*domain.StorageCalculation, error) {
	return nil, nil
}
func (f *fakeStorageRepo) SumByPeriod(context.Context, string, time.Time, time.Time) (float64, error) {
	return 0, nil
}

type fakeTracer struct {
	shutdownErr error
}

func (f *fakeTracer) Shutdown(context.Context) error { return f.shutdownErr }

func TestRunSuccess(t *testing.T) {
	oldMongo := newInstrumentedMongoClient
	oldProducer := newInstrumentedKafkaProducer
	oldOutbox := newOutboxPublisher
	oldActivityRepo := newBillableActivityRepository
	oldInvoiceRepo := newInvoiceRepository
	oldStorageRepo := newStorageCalculationRepository
	oldInitTracing := initTracing
	oldStartHTTP := startHTTPServer

	defer func() {
		newInstrumentedMongoClient = oldMongo
		newInstrumentedKafkaProducer = oldProducer
		newOutboxPublisher = oldOutbox
		newBillableActivityRepository = oldActivityRepo
		newInvoiceRepository = oldInvoiceRepo
		newStorageCalculationRepository = oldStorageRepo
		initTracing = oldInitTracing
		startHTTPServer = oldStartHTTP
	}()

	newInstrumentedMongoClient = func(context.Context, *mongodb.Config, *metrics.Metrics, *logging.Logger) (instrumentedMongoClient, error) {
		return &fakeMongo{}, nil
	}
	newInstrumentedKafkaProducer = func(*kafka.Config, *metrics.Metrics, *logging.Logger) kafkaProducer {
		return &fakeProducer{}
	}

	started := false
	stopped := false
	newOutboxPublisher = func(outbox.Repository, kafkaProducer, *logging.Logger, *metrics.Metrics, *outbox.PublisherConfig) outboxPublisher {
		return &fakeOutboxPublisher{
			started: &started,
			stopped: &stopped,
		}
	}

	newBillableActivityRepository = func(*mongo.Database) domain.BillableActivityRepository {
		return &fakeActivityRepo{}
	}
	newInvoiceRepository = func(*mongo.Database, *cloudevents.EventFactory) invoiceRepository {
		return &fakeInvoiceRepo{outboxRepo: &fakeOutboxRepo{}}
	}
	newStorageCalculationRepository = func(*mongo.Database) domain.StorageCalculationRepository {
		return &fakeStorageRepo{}
	}

	initTracing = func(context.Context, *tracing.Config) (*tracing.TracerProvider, error) {
		return &tracing.TracerProvider{}, nil
	}

	startHTTPServer = func(*http.Server) error { return http.ErrServerClosed }

	signalCh := make(chan os.Signal, 1)
	signalCh <- os.Interrupt

	err := run(context.Background(), signalCh)
	require.NoError(t, err)
	assert.True(t, started)
	assert.True(t, stopped)
}

func TestRunTracingError(t *testing.T) {
	oldInitTracing := initTracing
	defer func() { initTracing = oldInitTracing }()

	initTracing = func(context.Context, *tracing.Config) (*tracing.TracerProvider, error) {
		return nil, errors.New("trace init failed")
	}

	oldMongo := newInstrumentedMongoClient
	oldProducer := newInstrumentedKafkaProducer
	oldOutbox := newOutboxPublisher
	oldActivityRepo := newBillableActivityRepository
	oldInvoiceRepo := newInvoiceRepository
	oldStorageRepo := newStorageCalculationRepository
	oldStartHTTP := startHTTPServer

	defer func() {
		newInstrumentedMongoClient = oldMongo
		newInstrumentedKafkaProducer = oldProducer
		newOutboxPublisher = oldOutbox
		newBillableActivityRepository = oldActivityRepo
		newInvoiceRepository = oldInvoiceRepo
		newStorageCalculationRepository = oldStorageRepo
		startHTTPServer = oldStartHTTP
	}()

	newInstrumentedMongoClient = func(context.Context, *mongodb.Config, *metrics.Metrics, *logging.Logger) (instrumentedMongoClient, error) {
		return &fakeMongo{}, nil
	}
	newInstrumentedKafkaProducer = func(*kafka.Config, *metrics.Metrics, *logging.Logger) kafkaProducer {
		return &fakeProducer{}
	}
	newOutboxPublisher = func(outbox.Repository, kafkaProducer, *logging.Logger, *metrics.Metrics, *outbox.PublisherConfig) outboxPublisher {
		return &fakeOutboxPublisher{}
	}
	newBillableActivityRepository = func(*mongo.Database) domain.BillableActivityRepository { return &fakeActivityRepo{} }
	newInvoiceRepository = func(*mongo.Database, *cloudevents.EventFactory) invoiceRepository {
		return &fakeInvoiceRepo{outboxRepo: &fakeOutboxRepo{}}
	}
	newStorageCalculationRepository = func(*mongo.Database) domain.StorageCalculationRepository { return &fakeStorageRepo{} }
	startHTTPServer = func(*http.Server) error { return http.ErrServerClosed }

	signalCh := make(chan os.Signal, 1)
	signalCh <- os.Interrupt

	err := run(context.Background(), signalCh)
	require.NoError(t, err)
}

func TestRunMongoError(t *testing.T) {
	oldMongo := newInstrumentedMongoClient
	defer func() { newInstrumentedMongoClient = oldMongo }()

	newInstrumentedMongoClient = func(context.Context, *mongodb.Config, *metrics.Metrics, *logging.Logger) (instrumentedMongoClient, error) {
		return nil, errors.New("mongo error")
	}

	signalCh := make(chan os.Signal, 1)
	signalCh <- os.Interrupt

	err := run(context.Background(), signalCh)
	assert.Error(t, err)
}

func TestRunOutboxStartError(t *testing.T) {
	oldMongo := newInstrumentedMongoClient
	oldProducer := newInstrumentedKafkaProducer
	oldOutbox := newOutboxPublisher
	oldActivityRepo := newBillableActivityRepository
	oldInvoiceRepo := newInvoiceRepository
	oldStorageRepo := newStorageCalculationRepository
	oldInitTracing := initTracing
	oldStartHTTP := startHTTPServer

	defer func() {
		newInstrumentedMongoClient = oldMongo
		newInstrumentedKafkaProducer = oldProducer
		newOutboxPublisher = oldOutbox
		newBillableActivityRepository = oldActivityRepo
		newInvoiceRepository = oldInvoiceRepo
		newStorageCalculationRepository = oldStorageRepo
		initTracing = oldInitTracing
		startHTTPServer = oldStartHTTP
	}()

	newInstrumentedMongoClient = func(context.Context, *mongodb.Config, *metrics.Metrics, *logging.Logger) (instrumentedMongoClient, error) {
		return &fakeMongo{}, nil
	}
	newInstrumentedKafkaProducer = func(*kafka.Config, *metrics.Metrics, *logging.Logger) kafkaProducer {
		return &fakeProducer{}
	}
	newOutboxPublisher = func(outbox.Repository, kafkaProducer, *logging.Logger, *metrics.Metrics, *outbox.PublisherConfig) outboxPublisher {
		return &fakeOutboxPublisher{startErr: errors.New("start failed")}
	}
	newBillableActivityRepository = func(*mongo.Database) domain.BillableActivityRepository { return &fakeActivityRepo{} }
	newInvoiceRepository = func(*mongo.Database, *cloudevents.EventFactory) invoiceRepository {
		return &fakeInvoiceRepo{outboxRepo: &fakeOutboxRepo{}}
	}
	newStorageCalculationRepository = func(*mongo.Database) domain.StorageCalculationRepository { return &fakeStorageRepo{} }
	initTracing = func(context.Context, *tracing.Config) (*tracing.TracerProvider, error) {
		return &tracing.TracerProvider{}, nil
	}
	startHTTPServer = func(*http.Server) error { return http.ErrServerClosed }

	signalCh := make(chan os.Signal, 1)
	signalCh <- os.Interrupt

	err := run(context.Background(), signalCh)
	assert.Error(t, err)
}

func TestRunServerErrorLogged(t *testing.T) {
	oldMongo := newInstrumentedMongoClient
	oldProducer := newInstrumentedKafkaProducer
	oldOutbox := newOutboxPublisher
	oldActivityRepo := newBillableActivityRepository
	oldInvoiceRepo := newInvoiceRepository
	oldStorageRepo := newStorageCalculationRepository
	oldInitTracing := initTracing
	oldStartHTTP := startHTTPServer

	defer func() {
		newInstrumentedMongoClient = oldMongo
		newInstrumentedKafkaProducer = oldProducer
		newOutboxPublisher = oldOutbox
		newBillableActivityRepository = oldActivityRepo
		newInvoiceRepository = oldInvoiceRepo
		newStorageCalculationRepository = oldStorageRepo
		initTracing = oldInitTracing
		startHTTPServer = oldStartHTTP
	}()

	newInstrumentedMongoClient = func(context.Context, *mongodb.Config, *metrics.Metrics, *logging.Logger) (instrumentedMongoClient, error) {
		return &fakeMongo{}, nil
	}
	newInstrumentedKafkaProducer = func(*kafka.Config, *metrics.Metrics, *logging.Logger) kafkaProducer {
		return &fakeProducer{}
	}
	newOutboxPublisher = func(outbox.Repository, kafkaProducer, *logging.Logger, *metrics.Metrics, *outbox.PublisherConfig) outboxPublisher {
		return &fakeOutboxPublisher{}
	}
	newBillableActivityRepository = func(*mongo.Database) domain.BillableActivityRepository { return &fakeActivityRepo{} }
	newInvoiceRepository = func(*mongo.Database, *cloudevents.EventFactory) invoiceRepository {
		return &fakeInvoiceRepo{outboxRepo: &fakeOutboxRepo{}}
	}
	newStorageCalculationRepository = func(*mongo.Database) domain.StorageCalculationRepository { return &fakeStorageRepo{} }
	initTracing = func(context.Context, *tracing.Config) (*tracing.TracerProvider, error) {
		return &tracing.TracerProvider{}, nil
	}

	serverCalled := make(chan struct{})
	startHTTPServer = func(*http.Server) error {
		close(serverCalled)
		return errors.New("server failed")
	}

	signalCh := make(chan os.Signal, 1)
	go func() {
		<-serverCalled
		signalCh <- os.Interrupt
	}()

	err := run(context.Background(), signalCh)
	assert.NoError(t, err)
}
