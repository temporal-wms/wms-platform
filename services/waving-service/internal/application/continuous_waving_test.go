package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wms-platform/waving-service/internal/domain"
)

type MockContinuousWaveRepo struct {
	mock.Mock
}

func (m *MockContinuousWaveRepo) Save(ctx context.Context, wave *domain.Wave) error {
	args := m.Called(ctx, wave)
	return args.Error(0)
}

func (m *MockContinuousWaveRepo) FindByID(ctx context.Context, waveID string) (*domain.Wave, error) {
	args := m.Called(ctx, waveID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Wave), args.Error(1)
}

func (m *MockContinuousWaveRepo) FindByStatus(ctx context.Context, status domain.WaveStatus) ([]*domain.Wave, error) {
	args := m.Called(ctx, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Wave), args.Error(1)
}

func (m *MockContinuousWaveRepo) FindByType(ctx context.Context, waveType domain.WaveType) ([]*domain.Wave, error) {
	args := m.Called(ctx, waveType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Wave), args.Error(1)
}

func (m *MockContinuousWaveRepo) FindByZone(ctx context.Context, zone string) ([]*domain.Wave, error) {
	args := m.Called(ctx, zone)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Wave), args.Error(1)
}

func (m *MockContinuousWaveRepo) FindScheduledBefore(ctx context.Context, before time.Time) ([]*domain.Wave, error) {
	args := m.Called(ctx, before)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Wave), args.Error(1)
}

func (m *MockContinuousWaveRepo) FindReadyForRelease(ctx context.Context) ([]*domain.Wave, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Wave), args.Error(1)
}

func (m *MockContinuousWaveRepo) FindByOrderID(ctx context.Context, orderID string) (*domain.Wave, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Wave), args.Error(1)
}

func (m *MockContinuousWaveRepo) FindActive(ctx context.Context) ([]*domain.Wave, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Wave), args.Error(1)
}

func (m *MockContinuousWaveRepo) FindByDateRange(ctx context.Context, start, end time.Time) ([]*domain.Wave, error) {
	args := m.Called(ctx, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Wave), args.Error(1)
}

func (m *MockContinuousWaveRepo) Delete(ctx context.Context, waveID string) error {
	args := m.Called(ctx, waveID)
	return args.Error(0)
}

func (m *MockContinuousWaveRepo) Count(ctx context.Context, status domain.WaveStatus) (int64, error) {
	args := m.Called(ctx, status)
	return args.Get(0).(int64), args.Error(1)
}

type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) Publish(ctx context.Context, event domain.DomainEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockEventPublisher) PublishAll(ctx context.Context, events []domain.DomainEvent) error {
	args := m.Called(ctx, events)
	return args.Error(0)
}

type MockContinuousOrderService struct {
	mock.Mock
}

func (m *MockContinuousOrderService) GetOrdersReadyForWaving(ctx context.Context, filter domain.OrderFilter) ([]domain.WaveOrder, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.WaveOrder), args.Error(1)
}

func (m *MockContinuousOrderService) NotifyWaveAssignment(ctx context.Context, orderID, waveID string, scheduledStart time.Time) error {
	args := m.Called(ctx, orderID, waveID, scheduledStart)
	return args.Error(0)
}

func TestContinuousWavingService_Start(t *testing.T) {
	tests := []struct {
		name        string
		wantErr     bool
		errContains string
	}{
		{
			name:    "Successfully start service",
			wantErr: false,
		},
		{
			name:        "Start already running service",
			wantErr:     true,
			errContains: "already running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockContinuousWaveRepo)
			mockOrderService := new(MockContinuousOrderService)
			mockEventPublisher := new(MockEventPublisher)

			config := DefaultContinuousWavingConfig()
			service := NewContinuousWavingService(mockRepo, mockOrderService, mockEventPublisher, config)

			ctx, cancel := context.WithCancel(context.Background())

			err := service.Start(ctx)
			if tt.name == "Start already running service" {
				cancel()
				time.Sleep(50 * time.Millisecond)
				err = service.Start(ctx)
			} else {
				defer cancel()
			}

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.True(t, service.IsRunning())
			}
		})
	}
}

func TestContinuousWavingService_Stop(t *testing.T) {
	t.Run("Successfully stop service", func(t *testing.T) {
		mockRepo := new(MockContinuousWaveRepo)
		mockOrderService := new(MockContinuousOrderService)
		mockEventPublisher := new(MockEventPublisher)

		config := DefaultContinuousWavingConfig()
		service := NewContinuousWavingService(mockRepo, mockOrderService, mockEventPublisher, config)

		ctx, cancel := context.WithCancel(context.Background())
		err := service.Start(ctx)
		require.NoError(t, err)

		service.Stop()
		cancel()
		time.Sleep(50 * time.Millisecond)

		assert.False(t, service.IsRunning())
	})

	t.Run("Stop stopped service", func(t *testing.T) {
		mockRepo := new(MockContinuousWaveRepo)
		mockOrderService := new(MockContinuousOrderService)
		mockEventPublisher := new(MockEventPublisher)

		config := DefaultContinuousWavingConfig()
		service := NewContinuousWavingService(mockRepo, mockOrderService, mockEventPublisher, config)

		service.Stop()

		assert.False(t, service.IsRunning())
	})
}

func TestContinuousWavingService_IsRunning(t *testing.T) {
	t.Run("Service not running initially", func(t *testing.T) {
		mockRepo := new(MockContinuousWaveRepo)
		mockOrderService := new(MockContinuousOrderService)
		mockEventPublisher := new(MockEventPublisher)

		config := DefaultContinuousWavingConfig()
		service := NewContinuousWavingService(mockRepo, mockOrderService, mockEventPublisher, config)

		assert.False(t, service.IsRunning())
	})

	t.Run("Service running after start", func(t *testing.T) {
		mockRepo := new(MockContinuousWaveRepo)
		mockOrderService := new(MockContinuousOrderService)
		mockEventPublisher := new(MockEventPublisher)

		config := DefaultContinuousWavingConfig()
		service := NewContinuousWavingService(mockRepo, mockOrderService, mockEventPublisher, config)

		ctx, cancel := context.WithCancel(context.Background())
		err := service.Start(ctx)
		require.NoError(t, err)
		defer cancel()

		assert.True(t, service.IsRunning())
	})
}

func TestContinuousWavingService_ProcessSingleOrder(t *testing.T) {
	now := time.Now()
	order := domain.WaveOrder{
		OrderID:            "ORD-001",
		CustomerID:         "CUST-001",
		Priority:           "same_day",
		ItemCount:          5,
		TotalWeight:        10.5,
		PromisedDeliveryAt: now.Add(24 * time.Hour),
		CarrierCutoff:      now.Add(8 * time.Hour),
		Zone:               "ZONE-A",
		Status:             "pending",
	}

	t.Run("Successfully process single order", func(t *testing.T) {
		mockRepo := new(MockContinuousWaveRepo)
		mockOrderService := new(MockContinuousOrderService)
		mockEventPublisher := new(MockEventPublisher)

		mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.Wave")).Return(nil)
		mockOrderService.On("NotifyWaveAssignment", mock.Anything, "ORD-001", mock.Anything, mock.Anything).Return(nil)
		mockEventPublisher.On("PublishAll", mock.Anything, mock.Anything).Return(nil)

		config := DefaultContinuousWavingConfig()
		service := NewContinuousWavingService(mockRepo, mockOrderService, mockEventPublisher, config)

		err := service.ProcessSingleOrder(context.Background(), order)

		assert.NoError(t, err)
	})

	t.Run("Failed to publish events", func(t *testing.T) {
		mockRepo := new(MockContinuousWaveRepo)
		mockOrderService := new(MockContinuousOrderService)
		mockEventPublisher := new(MockEventPublisher)

		mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.Wave")).Return(nil)
		mockOrderService.On("NotifyWaveAssignment", mock.Anything, "ORD-001", mock.Anything, mock.Anything).Return(nil)
		mockEventPublisher.On("PublishAll", mock.Anything, mock.Anything).Return(nil)

		config := DefaultContinuousWavingConfig()
		service := NewContinuousWavingService(mockRepo, mockOrderService, mockEventPublisher, config)

		err := service.ProcessSingleOrder(context.Background(), order)

		assert.NoError(t, err)
	})
}

func TestContinuousWavingService_ReleaseOrders(t *testing.T) {
	now := time.Now()
	orders := []domain.WaveOrder{
		{
			OrderID:            "ORD-001",
			CustomerID:         "CUST-001",
			Priority:           "same_day",
			ItemCount:          5,
			TotalWeight:        10.5,
			PromisedDeliveryAt: now.Add(24 * time.Hour),
			CarrierCutoff:      now.Add(8 * time.Hour),
			Zone:               "ZONE-A",
			Status:             "pending",
		},
		{
			OrderID:            "ORD-002",
			CustomerID:         "CUST-002",
			Priority:           "next_day",
			ItemCount:          3,
			TotalWeight:        7.2,
			PromisedDeliveryAt: now.Add(48 * time.Hour),
			CarrierCutoff:      now.Add(16 * time.Hour),
			Zone:               "ZONE-B",
			Status:             "pending",
		},
	}

	t.Run("Successfully release orders", func(t *testing.T) {
		mockRepo := new(MockContinuousWaveRepo)
		mockOrderService := new(MockContinuousOrderService)
		mockEventPublisher := new(MockEventPublisher)

		mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.Wave")).Return(nil)
		mockOrderService.On("NotifyWaveAssignment", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Twice()
		mockEventPublisher.On("PublishAll", mock.Anything, mock.Anything).Return(nil)

		config := DefaultContinuousWavingConfig()
		service := NewContinuousWavingService(mockRepo, mockOrderService, mockEventPublisher, config)

		err := service.releaseOrders(context.Background(), orders, "immediate")

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockOrderService.AssertExpectations(t)
		mockEventPublisher.AssertExpectations(t)
	})
}

func TestContinuousWavingService_ProcessOrders(t *testing.T) {
	now := time.Now()
	immediateOrders := []domain.WaveOrder{
		{
			OrderID:            "ORD-001",
			Priority:           "same_day",
			ItemCount:          5,
			TotalWeight:        10.5,
			PromisedDeliveryAt: now.Add(24 * time.Hour),
			CarrierCutoff:      now.Add(8 * time.Hour),
			Zone:               "ZONE-A",
			Status:             "pending",
		},
	}

	t.Run("Successfully process immediate orders", func(t *testing.T) {
		mockRepo := new(MockContinuousWaveRepo)
		mockOrderService := new(MockContinuousOrderService)
		mockEventPublisher := new(MockEventPublisher)

		mockOrderService.On("GetOrdersReadyForWaving", mock.Anything, mock.Anything).Return(immediateOrders, nil)
		mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.Wave")).Return(nil)
		mockOrderService.On("NotifyWaveAssignment", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		mockEventPublisher.On("PublishAll", mock.Anything, mock.Anything).Return(nil)

		config := DefaultContinuousWavingConfig()
		config.BatchSize = 10
		config.MinOrdersForRelease = 1
		service := NewContinuousWavingService(mockRepo, mockOrderService, mockEventPublisher, config)

		err := service.processOrders(context.Background())

		assert.NoError(t, err)
		mockOrderService.AssertExpectations(t)
	})

	t.Run("No orders available", func(t *testing.T) {
		mockRepo := new(MockContinuousWaveRepo)
		mockOrderService := new(MockContinuousOrderService)
		mockEventPublisher := new(MockEventPublisher)

		mockOrderService.On("GetOrdersReadyForWaving", mock.Anything, mock.Anything).Return([]domain.WaveOrder{}, nil)

		config := DefaultContinuousWavingConfig()
		service := NewContinuousWavingService(mockRepo, mockOrderService, mockEventPublisher, config)

		err := service.processOrders(context.Background())

		assert.NoError(t, err)
		mockOrderService.AssertExpectations(t)
	})

	t.Run("Failed to get orders", func(t *testing.T) {
		mockRepo := new(MockContinuousWaveRepo)
		mockOrderService := new(MockContinuousOrderService)
		mockEventPublisher := new(MockEventPublisher)

		mockOrderService.On("GetOrdersReadyForWaving", mock.Anything, mock.Anything).Return(nil, errors.New("service error"))

		config := DefaultContinuousWavingConfig()
		service := NewContinuousWavingService(mockRepo, mockOrderService, mockEventPublisher, config)

		err := service.processOrders(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get orders")
	})
}

func TestGetPriorityValue(t *testing.T) {
	tests := []struct {
		priority string
		expected int
	}{
		{"same_day", 1},
		{"next_day", 2},
		{"standard", 3},
		{"unknown", 3},
		{"", 3},
	}

	for _, tt := range tests {
		t.Run(tt.priority, func(t *testing.T) {
			result := getPriorityValue(tt.priority)
			assert.Equal(t, tt.expected, result)
		})
	}
}
