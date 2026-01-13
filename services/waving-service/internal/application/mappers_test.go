package application

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wms-platform/waving-service/internal/domain"
)

func createTestWaveForMappers(waveID string) *domain.Wave {
	config := domain.WaveConfiguration{
		MaxOrders:           50,
		MaxItems:            500,
		MaxWeight:           1000.0,
		CutoffTime:          time.Now().Add(8 * time.Hour),
		ReleaseDelay:        30 * time.Minute,
		AutoRelease:         false,
		OptimizeForCarrier:  true,
		OptimizeForZone:     true,
		OptimizeForPriority: true,
	}

	wave, err := domain.NewWave(waveID, domain.WaveTypeDigital, domain.FulfillmentModeWave, config)
	if err != nil {
		panic(err)
	}

	order1 := domain.WaveOrder{
		OrderID:            "ORD-001",
		CustomerID:         "CUST-001",
		Priority:           "same_day",
		ItemCount:          5,
		TotalWeight:        10.5,
		PromisedDeliveryAt: time.Now().Add(24 * time.Hour),
		CarrierCutoff:      time.Now().Add(8 * time.Hour),
		Zone:               "ZONE-A",
		Status:             "pending",
		AddedAt:            time.Now(),
	}
	wave.AddOrder(order1)

	order2 := domain.WaveOrder{
		OrderID:            "ORD-002",
		CustomerID:         "CUST-002",
		Priority:           "next_day",
		ItemCount:          3,
		TotalWeight:        7.2,
		PromisedDeliveryAt: time.Now().Add(48 * time.Hour),
		CarrierCutoff:      time.Now().Add(16 * time.Hour),
		Zone:               "ZONE-B",
		Status:             "pending",
		AddedAt:            time.Now(),
	}
	wave.AddOrder(order2)

	wave.SetPriority(1)
	wave.SetZone("ZONE-A")

	now := time.Now()
	wave.Schedule(now.Add(1*time.Hour), now.Add(3*time.Hour))
	wave.Release()

	return wave
}

func TestToWaveDTO(t *testing.T) {
	tests := []struct {
		name     string
		wave     *domain.Wave
		wantNil  bool
		wantID   string
		wantType string
	}{
		{
			name:     "Convert full wave to DTO",
			wave:     createTestWaveForMappers("WAVE-001"),
			wantNil:  false,
			wantID:   "WAVE-001",
			wantType: "digital",
		},
		{
			name:    "Convert nil wave",
			wave:    nil,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToWaveDTO(tt.wave)

			if tt.wantNil {
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			assert.Equal(t, tt.wantID, result.WaveID)
			assert.Equal(t, tt.wantType, result.WaveType)
			assert.Equal(t, 2, len(result.Orders))
			assert.Equal(t, 8, result.TotalItems)
			assert.Equal(t, 17.7, result.TotalWeight)
			assert.Equal(t, 1, result.Priority)
			assert.Equal(t, "ZONE-A", result.Zone)
			assert.Equal(t, "released", result.Status)
		})
	}
}

func TestToWaveListDTO(t *testing.T) {
	tests := []struct {
		name    string
		wave    *domain.Wave
		wantNil bool
		wantID  string
	}{
		{
			name:    "Convert wave to list DTO",
			wave:    createTestWaveForMappers("WAVE-001"),
			wantNil: false,
			wantID:  "WAVE-001",
		},
		{
			name:    "Convert nil wave",
			wave:    nil,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToWaveListDTO(tt.wave)

			if tt.wantNil {
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			assert.Equal(t, tt.wantID, result.WaveID)
			assert.Equal(t, "digital", result.WaveType)
			assert.Equal(t, 2, result.OrderCount)
			assert.NotZero(t, result.CreatedAt)
		})
	}
}

func TestToWaveDTOs(t *testing.T) {
	tests := []struct {
		name    string
		waves   []*domain.Wave
		wantLen int
	}{
		{
			name:    "Convert multiple waves",
			waves:   []*domain.Wave{createTestWaveForMappers("WAVE-001"), createTestWaveForMappers("WAVE-002")},
			wantLen: 2,
		},
		{
			name:    "Convert empty slice",
			waves:   []*domain.Wave{},
			wantLen: 0,
		},
		{
			name:    "Convert nil slice",
			waves:   nil,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToWaveDTOs(tt.waves)
			assert.Equal(t, tt.wantLen, len(result))
		})
	}
}

func TestToWaveListDTOs(t *testing.T) {
	tests := []struct {
		name    string
		waves   []*domain.Wave
		wantLen int
	}{
		{
			name:    "Convert multiple waves",
			waves:   []*domain.Wave{createTestWaveForMappers("WAVE-001"), createTestWaveForMappers("WAVE-002")},
			wantLen: 2,
		},
		{
			name:    "Convert empty slice",
			waves:   []*domain.Wave{},
			wantLen: 0,
		},
		{
			name:    "Convert nil slice",
			waves:   nil,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToWaveListDTOs(tt.waves)
			assert.Equal(t, tt.wantLen, len(result))
		})
	}
}

func TestToWaveDTO_Configuration(t *testing.T) {
	wave := createTestWaveForMappers("WAVE-001")

	dto := ToWaveDTO(wave)
	require.NotNil(t, dto)

	assert.Equal(t, 50, dto.Configuration.MaxOrders)
	assert.Equal(t, 500, dto.Configuration.MaxItems)
	assert.Equal(t, 1000.0, dto.Configuration.MaxWeight)
	assert.True(t, dto.Configuration.OptimizeForCarrier)
	assert.True(t, dto.Configuration.OptimizeForZone)
	assert.True(t, dto.Configuration.OptimizeForPriority)
	assert.Equal(t, "30m0s", dto.Configuration.ReleaseDelay)
	assert.False(t, dto.Configuration.AutoRelease)
}

func TestToWaveDTO_LaborAllocation(t *testing.T) {
	wave := createTestWaveForMappers("WAVE-001")
	wave.AllocateLabor(domain.LaborAllocation{
		PickersRequired:   2,
		PackersRequired:   1,
		PickersAssigned:   2,
		PackersAssigned:   1,
		AssignedWorkerIDs: []string{"worker-001", "worker-002", "worker-003"},
	})

	dto := ToWaveDTO(wave)
	require.NotNil(t, dto)

	assert.Equal(t, 2, dto.LaborAllocation.PickersRequired)
	assert.Equal(t, 1, dto.LaborAllocation.PackersRequired)
	assert.Equal(t, 2, dto.LaborAllocation.PickersAssigned)
	assert.Equal(t, 1, dto.LaborAllocation.PackersAssigned)
	assert.Equal(t, 3, len(dto.LaborAllocation.AssignedWorkerIDs))
}

func TestToWaveDTO_Orders(t *testing.T) {
	wave := createTestWaveForMappers("WAVE-001")

	dto := ToWaveDTO(wave)
	require.NotNil(t, dto)

	require.Len(t, dto.Orders, 2)

	assert.Equal(t, "ORD-001", dto.Orders[0].OrderID)
	assert.Equal(t, "CUST-001", dto.Orders[0].CustomerID)
	assert.Equal(t, "same_day", dto.Orders[0].Priority)
	assert.Equal(t, 5, dto.Orders[0].ItemCount)
	assert.Equal(t, 10.5, dto.Orders[0].TotalWeight)
	assert.Equal(t, "ZONE-A", dto.Orders[0].Zone)
	assert.Equal(t, "picking", dto.Orders[0].Status)

	assert.Equal(t, "ORD-002", dto.Orders[1].OrderID)
	assert.Equal(t, "next_day", dto.Orders[1].Priority)
	assert.Equal(t, 3, dto.Orders[1].ItemCount)
}

func TestToWaveDTO_Timestamps(t *testing.T) {
	wave := createTestWaveForMappers("WAVE-001")

	dto := ToWaveDTO(wave)
	require.NotNil(t, dto)

	assert.NotZero(t, dto.ScheduledStart)
	assert.NotZero(t, dto.ScheduledEnd)
	assert.NotZero(t, dto.ActualStart)
	assert.NotZero(t, dto.ReleasedAt)
	assert.NotZero(t, dto.CreatedAt)
	assert.NotZero(t, dto.UpdatedAt)
}
