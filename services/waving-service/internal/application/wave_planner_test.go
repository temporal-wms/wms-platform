package application

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wms-platform/waving-service/internal/domain"
)

type MockWavePlannerRepo struct {
	mock.Mock
}

func (m *MockWavePlannerRepo) Save(ctx context.Context, wave *domain.Wave) error {
	args := m.Called(ctx, wave)
	return args.Error(0)
}

func (m *MockWavePlannerRepo) FindByID(ctx context.Context, waveID string) (*domain.Wave, error) {
	args := m.Called(ctx, waveID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Wave), args.Error(1)
}

func (m *MockWavePlannerRepo) FindByStatus(ctx context.Context, status domain.WaveStatus) ([]*domain.Wave, error) {
	args := m.Called(ctx, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Wave), args.Error(1)
}

func (m *MockWavePlannerRepo) FindByType(ctx context.Context, waveType domain.WaveType) ([]*domain.Wave, error) {
	args := m.Called(ctx, waveType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Wave), args.Error(1)
}

func (m *MockWavePlannerRepo) FindByZone(ctx context.Context, zone string) ([]*domain.Wave, error) {
	args := m.Called(ctx, zone)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Wave), args.Error(1)
}

func (m *MockWavePlannerRepo) FindScheduledBefore(ctx context.Context, before time.Time) ([]*domain.Wave, error) {
	args := m.Called(ctx, before)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Wave), args.Error(1)
}

func (m *MockWavePlannerRepo) FindReadyForRelease(ctx context.Context) ([]*domain.Wave, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Wave), args.Error(1)
}

func (m *MockWavePlannerRepo) FindByOrderID(ctx context.Context, orderID string) (*domain.Wave, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Wave), args.Error(1)
}

func (m *MockWavePlannerRepo) FindActive(ctx context.Context) ([]*domain.Wave, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Wave), args.Error(1)
}

func (m *MockWavePlannerRepo) FindByDateRange(ctx context.Context, start, end time.Time) ([]*domain.Wave, error) {
	args := m.Called(ctx, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Wave), args.Error(1)
}

func (m *MockWavePlannerRepo) Delete(ctx context.Context, waveID string) error {
	args := m.Called(ctx, waveID)
	return args.Error(0)
}

func (m *MockWavePlannerRepo) Count(ctx context.Context, status domain.WaveStatus) (int64, error) {
	args := m.Called(ctx, status)
	return args.Get(0).(int64), args.Error(1)
}

type MockOrderServicePlanner struct {
	mock.Mock
}

func (m *MockOrderServicePlanner) GetOrdersReadyForWaving(ctx context.Context, filter domain.OrderFilter) ([]domain.WaveOrder, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.WaveOrder), args.Error(1)
}

func (m *MockOrderServicePlanner) NotifyWaveAssignment(ctx context.Context, orderID, waveID string, scheduledStart time.Time) error {
	args := m.Called(ctx, orderID, waveID, scheduledStart)
	return args.Error(0)
}

func createTestOrdersForPlanner() []domain.WaveOrder {
	now := time.Now()
	return []domain.WaveOrder{
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
		{
			OrderID:            "ORD-003",
			CustomerID:         "CUST-003",
			Priority:           "standard",
			ItemCount:          8,
			TotalWeight:        15.0,
			PromisedDeliveryAt: now.Add(72 * time.Hour),
			CarrierCutoff:      now.Add(24 * time.Hour),
			Zone:               "ZONE-A",
			Status:             "pending",
		},
	}
}

func TestWavePlanner_PlanWave(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*MockWavePlannerRepo, *MockOrderServicePlanner)
		config      domain.WavePlanningConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "Successfully plan wave with orders",
			setup: func(repo *MockWavePlannerRepo, orderService *MockOrderServicePlanner) {
				orders := createTestOrdersForPlanner()
				orderService.On("GetOrdersReadyForWaving", mock.Anything, mock.Anything).Return(orders, nil)
			},
			config: domain.WavePlanningConfig{
				WaveType:        domain.WaveTypeDigital,
				FulfillmentMode: domain.FulfillmentModeWave,
				MaxOrders:       10,
				MaxItems:        100,
				MaxWeight:       500.0,
				Zone:            "ZONE-A",
				CutoffTime:      time.Now().Add(24 * time.Hour),
			},
			wantErr: false,
		},
		{
			name: "No orders available for waving",
			setup: func(repo *MockWavePlannerRepo, orderService *MockOrderServicePlanner) {
				orderService.On("GetOrdersReadyForWaving", mock.Anything, mock.Anything).Return([]domain.WaveOrder{}, nil)
			},
			config: domain.WavePlanningConfig{
				WaveType:        domain.WaveTypeDigital,
				FulfillmentMode: domain.FulfillmentModeWave,
				MaxOrders:       10,
				CutoffTime:      time.Now().Add(24 * time.Hour),
			},
			wantErr:     true,
			errContains: "no orders available",
		},
		{
			name: "Failed to get orders",
			setup: func(repo *MockWavePlannerRepo, orderService *MockOrderServicePlanner) {
				orderService.On("GetOrdersReadyForWaving", mock.Anything, mock.Anything).Return(nil, errors.New("service error"))
			},
			config: domain.WavePlanningConfig{
				WaveType:   domain.WaveTypeDigital,
				MaxOrders:  10,
				CutoffTime: time.Now().Add(24 * time.Hour),
			},
			wantErr:     true,
			errContains: "failed to get orders",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockWavePlannerRepo)
			mockOrderService := new(MockOrderServicePlanner)

			tt.setup(mockRepo, mockOrderService)

			planner := NewWavePlanner(mockRepo, mockOrderService)
			result, err := planner.PlanWave(context.Background(), tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.NotEmpty(t, result.WaveID)
				assert.Equal(t, tt.config.WaveType, result.WaveType)
			}

			mockOrderService.AssertExpectations(t)
		})
	}
}

func TestWavePlanner_OptimizeWave(t *testing.T) {
	tests := []struct {
		name        string
		wave        *domain.Wave
		wantErr     bool
		errContains string
	}{
		{
			name: "Optimize scheduled wave",
			wave: func() *domain.Wave {
				config := domain.WaveConfiguration{MaxOrders: 10}
				wave, _ := domain.NewWave("WAVE-001", domain.WaveTypeDigital, domain.FulfillmentModeWave, config)
				wave.AddOrder(domain.WaveOrder{OrderID: "ORD-001", Priority: "standard", ItemCount: 8, Zone: "ZONE-B", Status: "pending"})
				wave.AddOrder(domain.WaveOrder{OrderID: "ORD-002", Priority: "same_day", ItemCount: 3, Zone: "ZONE-A", Status: "pending"})
				wave.Schedule(time.Now().Add(1*time.Hour), time.Now().Add(3*time.Hour))
				return wave
			}(),
			wantErr: false,
		},
		{
			name: "Optimize planning wave",
			wave: func() *domain.Wave {
				config := domain.WaveConfiguration{MaxOrders: 10}
				wave, _ := domain.NewWave("WAVE-001", domain.WaveTypeDigital, domain.FulfillmentModeWave, config)
				wave.AddOrder(domain.WaveOrder{OrderID: "ORD-001", Priority: "standard", ItemCount: 8, Zone: "ZONE-B", Status: "pending"})
				return wave
			}(),
			wantErr: false,
		},
		{
			name: "Cannot optimize released wave",
			wave: func() *domain.Wave {
				config := domain.WaveConfiguration{MaxOrders: 10}
				wave, _ := domain.NewWave("WAVE-001", domain.WaveTypeDigital, domain.FulfillmentModeWave, config)
				wave.AddOrder(domain.WaveOrder{OrderID: "ORD-001", Priority: "standard", ItemCount: 8, Zone: "ZONE-B", Status: "pending"})
				wave.Schedule(time.Now().Add(1*time.Hour), time.Now().Add(3*time.Hour))
				wave.Release()
				return wave
			}(),
			wantErr:     true,
			errContains: "can only optimize waves in planning or scheduled status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockWavePlannerRepo)
			mockOrderService := new(MockOrderServicePlanner)

			planner := NewWavePlanner(mockRepo, mockOrderService)
			result, err := planner.OptimizeWave(context.Background(), tt.wave)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.wave.WaveID, result.WaveID)
			}
		})
	}
}

func TestWavePlanner_SuggestOrders(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*MockWavePlannerRepo, *MockOrderServicePlanner)
		wave      *domain.Wave
		limit     int
		wantErr   bool
		wantCount int
	}{
		{
			name: "Successfully suggest orders",
			setup: func(repo *MockWavePlannerRepo, orderService *MockOrderServicePlanner) {
				orders := createTestOrdersForPlanner()
				orderService.On("GetOrdersReadyForWaving", mock.Anything, mock.Anything).Return(orders, nil)
			},
			wave: func() *domain.Wave {
				config := domain.WaveConfiguration{MaxOrders: 10}
				wave, _ := domain.NewWave("WAVE-001", domain.WaveTypeDigital, domain.FulfillmentModeWave, config)
				wave.AddOrder(domain.WaveOrder{OrderID: "ORD-001", Priority: "same_day", Zone: "ZONE-A", Status: "pending"})
				wave.Configuration.ZoneFilter = []string{"ZONE-A"}
				wave.Configuration.PriorityFilter = []string{"same_day"}
				return wave
			}(),
			limit:     5,
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "No orders to suggest",
			setup: func(repo *MockWavePlannerRepo, orderService *MockOrderServicePlanner) {
				orderService.On("GetOrdersReadyForWaving", mock.Anything, mock.Anything).Return([]domain.WaveOrder{}, nil)
			},
			wave: func() *domain.Wave {
				config := domain.WaveConfiguration{MaxOrders: 10}
				wave, _ := domain.NewWave("WAVE-001", domain.WaveTypeDigital, domain.FulfillmentModeWave, config)
				wave.AddOrder(domain.WaveOrder{OrderID: "ORD-001", Priority: "same_day", Zone: "ZONE-A", Status: "pending"})
				return wave
			}(),
			limit:     5,
			wantErr:   false,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockWavePlannerRepo)
			mockOrderService := new(MockOrderServicePlanner)

			tt.setup(mockRepo, mockOrderService)

			planner := NewWavePlanner(mockRepo, mockOrderService)
			result, err := planner.SuggestOrders(context.Background(), tt.wave, tt.limit)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantCount, len(result))
			}

			mockOrderService.AssertExpectations(t)
		})
	}
}

func TestWavePlanner_SortOrdersForWave(t *testing.T) {
	orders := createTestOrdersForPlanner()

	sorted := sortOrdersForWave(orders, domain.WavePlanningConfig{})

	assert.Equal(t, 3, len(sorted))
	assert.Equal(t, "same_day", sorted[0].Priority)
	assert.Equal(t, "next_day", sorted[1].Priority)
	assert.Equal(t, "standard", sorted[2].Priority)
}

func TestWavePlanner_OptimizeOrderSequence(t *testing.T) {
	tests := []struct {
		name  string
		input []domain.WaveOrder
	}{
		{
			name: "Optimize orders by zone",
			input: []domain.WaveOrder{
				{OrderID: "ORD-001", Zone: "ZONE-B", ItemCount: 8},
				{OrderID: "ORD-002", Zone: "ZONE-A", ItemCount: 3},
				{OrderID: "ORD-003", Zone: "ZONE-A", ItemCount: 5},
			},
		},
		{
			name:  "Empty orders",
			input: []domain.WaveOrder{},
		},
		{
			name:  "Single order",
			input: []domain.WaveOrder{{OrderID: "ORD-001", Zone: "ZONE-A", ItemCount: 3}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := optimizeOrderSequence(tt.input)
			if len(tt.input) > 1 {
				assert.Equal(t, tt.input[1].Zone, result[0].Zone)
				assert.Equal(t, tt.input[2].Zone, result[1].Zone)
				assert.Equal(t, tt.input[0].Zone, result[2].Zone)
			} else {
				assert.Equal(t, len(tt.input), len(result))
			}
		})
	}
}

func TestWavePlanner_CalculateLaborRequirements(t *testing.T) {
	tests := []struct {
		name        string
		orderCount  int
		totalItems  int
		wantPickers int
		wantPackers int
	}{
		{
			name:        "Small wave",
			orderCount:  5,
			totalItems:  50,
			wantPickers: 1,
			wantPackers: 1,
		},
		{
			name:        "Medium wave",
			orderCount:  60,
			totalItems:  300,
			wantPickers: 4,
			wantPackers: 2,
		},
		{
			name:        "Large wave",
			orderCount:  150,
			totalItems:  1000,
			wantPickers: 10,
			wantPackers: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := domain.WaveConfiguration{MaxOrders: 200}
			wave, _ := domain.NewWave("WAVE-001", domain.WaveTypeDigital, domain.FulfillmentModeWave, config)

			for i := 0; i < tt.orderCount; i++ {
				itemsPerOrder := tt.totalItems / tt.orderCount
				wave.AddOrder(domain.WaveOrder{
					OrderID:     fmt.Sprintf("ORD-%03d", i+1),
					Priority:    "standard",
					ItemCount:   itemsPerOrder,
					TotalWeight: 5.0,
					Zone:        "ZONE-A",
					Status:      "pending",
				})
			}

			labor := calculateLaborRequirements(wave)
			assert.Equal(t, tt.wantPickers, labor.PickersRequired)
			assert.Equal(t, tt.wantPackers, labor.PackersRequired)
		})
	}
}

func TestWavePlanner_CalculateWavePriority(t *testing.T) {
	tests := []struct {
		name       string
		priorities []string
		wantPrio   int
	}{
		{
			name:       "Same day priority",
			priorities: []string{"same_day"},
			wantPrio:   1,
		},
		{
			name:       "Next day priority",
			priorities: []string{"next_day"},
			wantPrio:   2,
		},
		{
			name:       "Standard priority",
			priorities: []string{"standard"},
			wantPrio:   3,
		},
		{
			name:       "Mixed with same day",
			priorities: []string{"standard", "same_day"},
			wantPrio:   1,
		},
		{
			name:       "Mixed with next day",
			priorities: []string{"standard", "next_day"},
			wantPrio:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := domain.WaveConfiguration{MaxOrders: len(tt.priorities)}
			wave, _ := domain.NewWave("WAVE-001", domain.WaveTypeDigital, domain.FulfillmentModeWave, config)

			for i, prio := range tt.priorities {
				wave.AddOrder(domain.WaveOrder{
					OrderID:     fmt.Sprintf("ORD-%03d", i+1),
					Priority:    prio,
					ItemCount:   5,
					TotalWeight: 10.0,
					Zone:        "ZONE-A",
					Status:      "pending",
				})
			}

			prio := calculateWavePriority(wave)
			assert.Equal(t, tt.wantPrio, prio)
		})
	}
}
