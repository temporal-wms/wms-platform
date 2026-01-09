package application

import (
	"context"
	"errors"
	"testing"

	"github.com/wms-platform/facility-service/internal/domain"
	sharedErrors "github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/logging"
)

// MockStationRepository is a mock implementation of StationRepository for testing
type MockStationRepository struct {
	stations map[string]*domain.Station
	saveErr  error
	findErr  error
}

func NewMockStationRepository() *MockStationRepository {
	return &MockStationRepository{
		stations: make(map[string]*domain.Station),
	}
}

func (m *MockStationRepository) Save(ctx context.Context, station *domain.Station) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.stations[station.StationID] = station
	return nil
}

func (m *MockStationRepository) FindByID(ctx context.Context, stationID string) (*domain.Station, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.stations[stationID], nil
}

func (m *MockStationRepository) FindByZone(ctx context.Context, zone string) ([]*domain.Station, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	var result []*domain.Station
	for _, station := range m.stations {
		if station.Zone == zone {
			result = append(result, station)
		}
	}
	return result, nil
}

func (m *MockStationRepository) FindByType(ctx context.Context, stationType domain.StationType) ([]*domain.Station, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	var result []*domain.Station
	for _, station := range m.stations {
		if station.StationType == stationType {
			result = append(result, station)
		}
	}
	return result, nil
}

func (m *MockStationRepository) FindByStatus(ctx context.Context, status domain.StationStatus) ([]*domain.Station, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	var result []*domain.Station
	for _, station := range m.stations {
		if station.Status == status {
			result = append(result, station)
		}
	}
	return result, nil
}

func (m *MockStationRepository) FindCapableStations(ctx context.Context, requirements []domain.StationCapability, stationType domain.StationType, zone string) ([]*domain.Station, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	var result []*domain.Station
	for _, station := range m.stations {
		if station.Status != domain.StationStatusActive {
			continue
		}
		if stationType != "" && station.StationType != stationType {
			continue
		}
		if zone != "" && station.Zone != zone {
			continue
		}
		if station.HasAllCapabilities(requirements) {
			result = append(result, station)
		}
	}
	return result, nil
}

func (m *MockStationRepository) FindByCapability(ctx context.Context, capability domain.StationCapability) ([]*domain.Station, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	var result []*domain.Station
	for _, station := range m.stations {
		if station.HasCapability(capability) {
			result = append(result, station)
		}
	}
	return result, nil
}

func (m *MockStationRepository) FindByWorkerID(ctx context.Context, workerID string) (*domain.Station, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	for _, station := range m.stations {
		if station.AssignedWorkerID == workerID {
			return station, nil
		}
	}
	return nil, nil
}

func (m *MockStationRepository) FindAll(ctx context.Context, limit, offset int) ([]*domain.Station, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	var result []*domain.Station
	for _, station := range m.stations {
		result = append(result, station)
	}
	// Apply offset and limit
	if offset >= len(result) {
		return []*domain.Station{}, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

func (m *MockStationRepository) Delete(ctx context.Context, stationID string) error {
	if m.findErr != nil {
		return m.findErr
	}
	delete(m.stations, stationID)
	return nil
}

// SetError sets an error for save or find operations
func (m *MockStationRepository) SetSaveError(err error) {
	m.saveErr = err
}

func (m *MockStationRepository) SetFindError(err error) {
	m.findErr = err
}

// AddStation adds a station directly to the mock (for test setup)
func (m *MockStationRepository) AddStation(station *domain.Station) {
	m.stations[station.StationID] = station
}

// createTestService creates a service with a mock repository for testing
func createTestService() (*StationApplicationService, *MockStationRepository) {
	repo := NewMockStationRepository()
	logger := logging.New(logging.DefaultConfig("test"))
	service := NewStationApplicationService(repo, nil, nil, logger)
	return service, repo
}

// =============================================================================
// CreateStation Tests
// =============================================================================

func TestStationApplicationService_CreateStation(t *testing.T) {
	t.Run("creates station successfully", func(t *testing.T) {
		service, _ := createTestService()
		ctx := context.Background()

		cmd := CreateStationCommand{
			StationID:          "STN-001",
			Name:               "Packing Station 1",
			Zone:               "zone-a",
			StationType:        "packing",
			Capabilities:       []string{"single_item", "gift_wrap"},
			MaxConcurrentTasks: 5,
		}

		dto, err := service.CreateStation(ctx, cmd)

		if err != nil {
			t.Fatalf("CreateStation() error = %v", err)
		}
		if dto.StationID != "STN-001" {
			t.Errorf("StationID = %v, want STN-001", dto.StationID)
		}
		if dto.Name != "Packing Station 1" {
			t.Errorf("Name = %v, want Packing Station 1", dto.Name)
		}
		if len(dto.Capabilities) != 2 {
			t.Errorf("Capabilities length = %v, want 2", len(dto.Capabilities))
		}
	})

	t.Run("returns error for invalid station type", func(t *testing.T) {
		service, _ := createTestService()
		ctx := context.Background()

		cmd := CreateStationCommand{
			StationID:   "STN-001",
			Name:        "Station",
			Zone:        "zone-a",
			StationType: "invalid_type",
		}

		_, err := service.CreateStation(ctx, cmd)

		if err == nil {
			t.Fatal("CreateStation() should return error for invalid station type")
		}
	})

	t.Run("returns error for invalid capability", func(t *testing.T) {
		service, _ := createTestService()
		ctx := context.Background()

		cmd := CreateStationCommand{
			StationID:    "STN-001",
			Name:         "Station",
			Zone:         "zone-a",
			StationType:  "packing",
			Capabilities: []string{"invalid_capability"},
		}

		_, err := service.CreateStation(ctx, cmd)

		if err == nil {
			t.Fatal("CreateStation() should return error for invalid capability")
		}
	})

	t.Run("returns error when save fails", func(t *testing.T) {
		service, repo := createTestService()
		repo.SetSaveError(errors.New("database error"))
		ctx := context.Background()

		cmd := CreateStationCommand{
			StationID:   "STN-001",
			Name:        "Station",
			Zone:        "zone-a",
			StationType: "packing",
		}

		_, err := service.CreateStation(ctx, cmd)

		if err == nil {
			t.Fatal("CreateStation() should return error when save fails")
		}
	})
}

// =============================================================================
// GetStation Tests
// =============================================================================

func TestStationApplicationService_GetStation(t *testing.T) {
	t.Run("returns station when found", func(t *testing.T) {
		service, repo := createTestService()
		ctx := context.Background()

		station, _ := domain.NewStation("STN-001", "Station 1", "zone-a", domain.StationTypePacking, 5)
		repo.AddStation(station)

		query := GetStationQuery{StationID: "STN-001"}
		dto, err := service.GetStation(ctx, query)

		if err != nil {
			t.Fatalf("GetStation() error = %v", err)
		}
		if dto.StationID != "STN-001" {
			t.Errorf("StationID = %v, want STN-001", dto.StationID)
		}
	})

	t.Run("returns not found error when station doesn't exist", func(t *testing.T) {
		service, _ := createTestService()
		ctx := context.Background()

		query := GetStationQuery{StationID: "nonexistent"}
		_, err := service.GetStation(ctx, query)

		if err == nil {
			t.Fatal("GetStation() should return error for nonexistent station")
		}
		appErr, ok := err.(*sharedErrors.AppError)
		if !ok || appErr.Code != sharedErrors.CodeNotFound {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

// =============================================================================
// UpdateStation Tests
// =============================================================================

func TestStationApplicationService_UpdateStation(t *testing.T) {
	t.Run("updates station fields", func(t *testing.T) {
		service, repo := createTestService()
		ctx := context.Background()

		station, _ := domain.NewStation("STN-001", "Original Name", "zone-a", domain.StationTypePacking, 3)
		repo.AddStation(station)

		cmd := UpdateStationCommand{
			StationID:          "STN-001",
			Name:               "Updated Name",
			Zone:               "zone-b",
			MaxConcurrentTasks: 10,
		}

		dto, err := service.UpdateStation(ctx, cmd)

		if err != nil {
			t.Fatalf("UpdateStation() error = %v", err)
		}
		if dto.Name != "Updated Name" {
			t.Errorf("Name = %v, want Updated Name", dto.Name)
		}
		if dto.Zone != "zone-b" {
			t.Errorf("Zone = %v, want zone-b", dto.Zone)
		}
		if dto.MaxConcurrentTasks != 10 {
			t.Errorf("MaxConcurrentTasks = %v, want 10", dto.MaxConcurrentTasks)
		}
	})

	t.Run("returns not found error for nonexistent station", func(t *testing.T) {
		service, _ := createTestService()
		ctx := context.Background()

		cmd := UpdateStationCommand{
			StationID: "nonexistent",
			Name:      "New Name",
		}

		_, err := service.UpdateStation(ctx, cmd)

		if err == nil {
			t.Fatal("UpdateStation() should return error for nonexistent station")
		}
	})
}

// =============================================================================
// Capability Tests
// =============================================================================

func TestStationApplicationService_AddCapability(t *testing.T) {
	t.Run("adds capability successfully", func(t *testing.T) {
		service, repo := createTestService()
		ctx := context.Background()

		station, _ := domain.NewStation("STN-001", "Station 1", "zone-a", domain.StationTypePacking, 5)
		repo.AddStation(station)

		cmd := AddCapabilityCommand{
			StationID:  "STN-001",
			Capability: "hazmat",
		}

		dto, err := service.AddCapability(ctx, cmd)

		if err != nil {
			t.Fatalf("AddCapability() error = %v", err)
		}
		found := false
		for _, cap := range dto.Capabilities {
			if cap == "hazmat" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Capability 'hazmat' should be in the list")
		}
	})

	t.Run("returns conflict error for duplicate capability", func(t *testing.T) {
		service, repo := createTestService()
		ctx := context.Background()

		station, _ := domain.NewStation("STN-001", "Station 1", "zone-a", domain.StationTypePacking, 5)
		station.AddCapability(domain.CapabilityHazmat)
		repo.AddStation(station)

		cmd := AddCapabilityCommand{
			StationID:  "STN-001",
			Capability: "hazmat",
		}

		_, err := service.AddCapability(ctx, cmd)

		if err == nil {
			t.Fatal("AddCapability() should return error for duplicate capability")
		}
	})
}

func TestStationApplicationService_RemoveCapability(t *testing.T) {
	t.Run("removes capability successfully", func(t *testing.T) {
		service, repo := createTestService()
		ctx := context.Background()

		station, _ := domain.NewStation("STN-001", "Station 1", "zone-a", domain.StationTypePacking, 5)
		station.AddCapability(domain.CapabilityHazmat)
		repo.AddStation(station)

		cmd := RemoveCapabilityCommand{
			StationID:  "STN-001",
			Capability: "hazmat",
		}

		dto, err := service.RemoveCapability(ctx, cmd)

		if err != nil {
			t.Fatalf("RemoveCapability() error = %v", err)
		}
		for _, cap := range dto.Capabilities {
			if cap == "hazmat" {
				t.Error("Capability 'hazmat' should have been removed")
			}
		}
	})

	t.Run("returns not found error for nonexistent capability", func(t *testing.T) {
		service, repo := createTestService()
		ctx := context.Background()

		station, _ := domain.NewStation("STN-001", "Station 1", "zone-a", domain.StationTypePacking, 5)
		repo.AddStation(station)

		cmd := RemoveCapabilityCommand{
			StationID:  "STN-001",
			Capability: "hazmat",
		}

		_, err := service.RemoveCapability(ctx, cmd)

		if err == nil {
			t.Fatal("RemoveCapability() should return error for nonexistent capability")
		}
	})
}

func TestStationApplicationService_SetCapabilities(t *testing.T) {
	t.Run("replaces all capabilities", func(t *testing.T) {
		service, repo := createTestService()
		ctx := context.Background()

		station, _ := domain.NewStation("STN-001", "Station 1", "zone-a", domain.StationTypePacking, 5)
		station.AddCapability(domain.CapabilityHazmat)
		repo.AddStation(station)

		cmd := SetCapabilitiesCommand{
			StationID:    "STN-001",
			Capabilities: []string{"single_item", "fragile"},
		}

		dto, err := service.SetCapabilities(ctx, cmd)

		if err != nil {
			t.Fatalf("SetCapabilities() error = %v", err)
		}
		if len(dto.Capabilities) != 2 {
			t.Errorf("Capabilities length = %v, want 2", len(dto.Capabilities))
		}
	})
}

// =============================================================================
// Status Tests
// =============================================================================

func TestStationApplicationService_SetStatus(t *testing.T) {
	t.Run("sets status successfully", func(t *testing.T) {
		service, repo := createTestService()
		ctx := context.Background()

		station, _ := domain.NewStation("STN-001", "Station 1", "zone-a", domain.StationTypePacking, 5)
		repo.AddStation(station)

		cmd := SetStationStatusCommand{
			StationID: "STN-001",
			Status:    "maintenance",
		}

		dto, err := service.SetStatus(ctx, cmd)

		if err != nil {
			t.Fatalf("SetStatus() error = %v", err)
		}
		if dto.Status != "maintenance" {
			t.Errorf("Status = %v, want maintenance", dto.Status)
		}
	})

	t.Run("returns error for invalid status", func(t *testing.T) {
		service, repo := createTestService()
		ctx := context.Background()

		station, _ := domain.NewStation("STN-001", "Station 1", "zone-a", domain.StationTypePacking, 5)
		repo.AddStation(station)

		cmd := SetStationStatusCommand{
			StationID: "STN-001",
			Status:    "invalid_status",
		}

		_, err := service.SetStatus(ctx, cmd)

		if err == nil {
			t.Fatal("SetStatus() should return error for invalid status")
		}
	})
}

// =============================================================================
// Query Tests
// =============================================================================

func TestStationApplicationService_FindCapableStations(t *testing.T) {
	t.Run("finds stations with required capabilities", func(t *testing.T) {
		service, repo := createTestService()
		ctx := context.Background()

		// Create stations with different capabilities
		station1, _ := domain.NewStation("STN-001", "Station 1", "zone-a", domain.StationTypePacking, 5)
		station1.AddCapability(domain.CapabilitySingleItem)
		station1.AddCapability(domain.CapabilityGiftWrap)
		repo.AddStation(station1)

		station2, _ := domain.NewStation("STN-002", "Station 2", "zone-a", domain.StationTypePacking, 5)
		station2.AddCapability(domain.CapabilitySingleItem)
		repo.AddStation(station2)

		query := FindCapableStationsQuery{
			Requirements: []string{"single_item", "gift_wrap"},
			StationType:  "packing",
		}

		dtos, err := service.FindCapableStations(ctx, query)

		if err != nil {
			t.Fatalf("FindCapableStations() error = %v", err)
		}
		if len(dtos) != 1 {
			t.Errorf("Found %d stations, want 1", len(dtos))
		}
		if len(dtos) > 0 && dtos[0].StationID != "STN-001" {
			t.Errorf("StationID = %v, want STN-001", dtos[0].StationID)
		}
	})
}

func TestStationApplicationService_ListStations(t *testing.T) {
	t.Run("lists all stations", func(t *testing.T) {
		service, repo := createTestService()
		ctx := context.Background()

		station1, _ := domain.NewStation("STN-001", "Station 1", "zone-a", domain.StationTypePacking, 5)
		station2, _ := domain.NewStation("STN-002", "Station 2", "zone-b", domain.StationTypeShipping, 5)
		repo.AddStation(station1)
		repo.AddStation(station2)

		query := ListStationsQuery{Limit: 10, Offset: 0}
		dtos, err := service.ListStations(ctx, query)

		if err != nil {
			t.Fatalf("ListStations() error = %v", err)
		}
		if len(dtos) != 2 {
			t.Errorf("Found %d stations, want 2", len(dtos))
		}
	})

	t.Run("applies default limit", func(t *testing.T) {
		service, repo := createTestService()
		ctx := context.Background()

		station, _ := domain.NewStation("STN-001", "Station 1", "zone-a", domain.StationTypePacking, 5)
		repo.AddStation(station)

		query := ListStationsQuery{Limit: 0, Offset: 0}
		_, err := service.ListStations(ctx, query)

		if err != nil {
			t.Fatalf("ListStations() error = %v", err)
		}
	})
}

func TestStationApplicationService_GetByZone(t *testing.T) {
	t.Run("returns stations in zone", func(t *testing.T) {
		service, repo := createTestService()
		ctx := context.Background()

		station1, _ := domain.NewStation("STN-001", "Station 1", "zone-a", domain.StationTypePacking, 5)
		station2, _ := domain.NewStation("STN-002", "Station 2", "zone-b", domain.StationTypePacking, 5)
		repo.AddStation(station1)
		repo.AddStation(station2)

		query := GetStationsByZoneQuery{Zone: "zone-a"}
		dtos, err := service.GetByZone(ctx, query)

		if err != nil {
			t.Fatalf("GetByZone() error = %v", err)
		}
		if len(dtos) != 1 {
			t.Errorf("Found %d stations, want 1", len(dtos))
		}
	})
}

func TestStationApplicationService_GetByType(t *testing.T) {
	t.Run("returns stations by type", func(t *testing.T) {
		service, repo := createTestService()
		ctx := context.Background()

		station1, _ := domain.NewStation("STN-001", "Station 1", "zone-a", domain.StationTypePacking, 5)
		station2, _ := domain.NewStation("STN-002", "Station 2", "zone-a", domain.StationTypeShipping, 5)
		repo.AddStation(station1)
		repo.AddStation(station2)

		query := GetStationsByTypeQuery{StationType: "packing"}
		dtos, err := service.GetByType(ctx, query)

		if err != nil {
			t.Fatalf("GetByType() error = %v", err)
		}
		if len(dtos) != 1 {
			t.Errorf("Found %d stations, want 1", len(dtos))
		}
	})
}

func TestStationApplicationService_GetByStatus(t *testing.T) {
	t.Run("returns stations by status", func(t *testing.T) {
		service, repo := createTestService()
		ctx := context.Background()

		station1, _ := domain.NewStation("STN-001", "Station 1", "zone-a", domain.StationTypePacking, 5)
		station2, _ := domain.NewStation("STN-002", "Station 2", "zone-a", domain.StationTypePacking, 5)
		station2.SetStatus(domain.StationStatusMaintenance)
		repo.AddStation(station1)
		repo.AddStation(station2)

		query := GetStationsByStatusQuery{Status: "active"}
		dtos, err := service.GetByStatus(ctx, query)

		if err != nil {
			t.Fatalf("GetByStatus() error = %v", err)
		}
		if len(dtos) != 1 {
			t.Errorf("Found %d stations, want 1", len(dtos))
		}
	})
}

// =============================================================================
// Delete Tests
// =============================================================================

func TestStationApplicationService_DeleteStation(t *testing.T) {
	t.Run("deletes station successfully", func(t *testing.T) {
		service, repo := createTestService()
		ctx := context.Background()

		station, _ := domain.NewStation("STN-001", "Station 1", "zone-a", domain.StationTypePacking, 5)
		repo.AddStation(station)

		cmd := DeleteStationCommand{StationID: "STN-001"}
		err := service.DeleteStation(ctx, cmd)

		if err != nil {
			t.Fatalf("DeleteStation() error = %v", err)
		}

		// Verify station is deleted
		query := GetStationQuery{StationID: "STN-001"}
		_, err = service.GetStation(ctx, query)
		if err == nil {
			t.Error("Station should be deleted")
		}
	})

	t.Run("returns not found error for nonexistent station", func(t *testing.T) {
		service, _ := createTestService()
		ctx := context.Background()

		cmd := DeleteStationCommand{StationID: "nonexistent"}
		err := service.DeleteStation(ctx, cmd)

		if err == nil {
			t.Fatal("DeleteStation() should return error for nonexistent station")
		}
	})
}
