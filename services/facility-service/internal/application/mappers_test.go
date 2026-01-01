package application

import (
	"testing"
	"time"

	"github.com/wms-platform/facility-service/internal/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestToStationDTO(t *testing.T) {
	t.Run("converts station to DTO correctly", func(t *testing.T) {
		now := time.Now()
		station := &domain.Station{
			ID:                 primitive.NewObjectID(),
			StationID:          "STN-001",
			Name:               "Packing Station 1",
			Zone:               "zone-a",
			StationType:        domain.StationTypePacking,
			Status:             domain.StationStatusActive,
			Capabilities:       []domain.StationCapability{domain.CapabilitySingleItem, domain.CapabilityGiftWrap},
			MaxConcurrentTasks: 5,
			CurrentTasks:       2,
			AssignedWorkerID:   "worker-123",
			Equipment: []domain.StationEquipment{
				{EquipmentID: "EQ-001", EquipmentType: "scale", Status: "active"},
				{EquipmentID: "EQ-002", EquipmentType: "printer", Status: "active"},
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		dto := ToStationDTO(station)

		if dto == nil {
			t.Fatal("ToStationDTO returned nil")
		}
		if dto.StationID != "STN-001" {
			t.Errorf("StationID = %v, want STN-001", dto.StationID)
		}
		if dto.Name != "Packing Station 1" {
			t.Errorf("Name = %v, want Packing Station 1", dto.Name)
		}
		if dto.Zone != "zone-a" {
			t.Errorf("Zone = %v, want zone-a", dto.Zone)
		}
		if dto.StationType != "packing" {
			t.Errorf("StationType = %v, want packing", dto.StationType)
		}
		if dto.Status != "active" {
			t.Errorf("Status = %v, want active", dto.Status)
		}
		if len(dto.Capabilities) != 2 {
			t.Errorf("Capabilities length = %v, want 2", len(dto.Capabilities))
		}
		if dto.Capabilities[0] != "single_item" {
			t.Errorf("Capabilities[0] = %v, want single_item", dto.Capabilities[0])
		}
		if dto.MaxConcurrentTasks != 5 {
			t.Errorf("MaxConcurrentTasks = %v, want 5", dto.MaxConcurrentTasks)
		}
		if dto.CurrentTasks != 2 {
			t.Errorf("CurrentTasks = %v, want 2", dto.CurrentTasks)
		}
		if dto.AvailableCapacity != 3 {
			t.Errorf("AvailableCapacity = %v, want 3", dto.AvailableCapacity)
		}
		if dto.AssignedWorkerID != "worker-123" {
			t.Errorf("AssignedWorkerID = %v, want worker-123", dto.AssignedWorkerID)
		}
		if len(dto.Equipment) != 2 {
			t.Errorf("Equipment length = %v, want 2", len(dto.Equipment))
		}
		if dto.Equipment[0].EquipmentID != "EQ-001" {
			t.Errorf("Equipment[0].EquipmentID = %v, want EQ-001", dto.Equipment[0].EquipmentID)
		}
	})

	t.Run("returns nil for nil station", func(t *testing.T) {
		dto := ToStationDTO(nil)
		if dto != nil {
			t.Error("ToStationDTO(nil) should return nil")
		}
	})

	t.Run("handles empty capabilities and equipment", func(t *testing.T) {
		station := &domain.Station{
			StationID:    "STN-002",
			Name:         "Empty Station",
			Zone:         "zone-b",
			StationType:  domain.StationTypeShipping,
			Status:       domain.StationStatusActive,
			Capabilities: []domain.StationCapability{},
			Equipment:    []domain.StationEquipment{},
		}

		dto := ToStationDTO(station)

		if dto == nil {
			t.Fatal("ToStationDTO returned nil")
		}
		if len(dto.Capabilities) != 0 {
			t.Errorf("Capabilities length = %v, want 0", len(dto.Capabilities))
		}
		if len(dto.Equipment) != 0 {
			t.Errorf("Equipment length = %v, want 0", len(dto.Equipment))
		}
	})

	t.Run("calculates available capacity for inactive station", func(t *testing.T) {
		station := &domain.Station{
			StationID:          "STN-003",
			Name:               "Inactive Station",
			Zone:               "zone-c",
			StationType:        domain.StationTypePacking,
			Status:             domain.StationStatusInactive,
			MaxConcurrentTasks: 5,
			CurrentTasks:       0,
			Capabilities:       []domain.StationCapability{},
			Equipment:          []domain.StationEquipment{},
		}

		dto := ToStationDTO(station)

		if dto.AvailableCapacity != 0 {
			t.Errorf("AvailableCapacity for inactive station = %v, want 0", dto.AvailableCapacity)
		}
	})
}

func TestToStationDTOs(t *testing.T) {
	t.Run("converts multiple stations", func(t *testing.T) {
		stations := []*domain.Station{
			{
				StationID:    "STN-001",
				Name:         "Station 1",
				Zone:         "zone-a",
				StationType:  domain.StationTypePacking,
				Status:       domain.StationStatusActive,
				Capabilities: []domain.StationCapability{domain.CapabilitySingleItem},
				Equipment:    []domain.StationEquipment{},
			},
			{
				StationID:    "STN-002",
				Name:         "Station 2",
				Zone:         "zone-b",
				StationType:  domain.StationTypeShipping,
				Status:       domain.StationStatusActive,
				Capabilities: []domain.StationCapability{domain.CapabilityMultiItem},
				Equipment:    []domain.StationEquipment{},
			},
		}

		dtos := ToStationDTOs(stations)

		if len(dtos) != 2 {
			t.Fatalf("DTOs length = %v, want 2", len(dtos))
		}
		if dtos[0].StationID != "STN-001" {
			t.Errorf("dtos[0].StationID = %v, want STN-001", dtos[0].StationID)
		}
		if dtos[1].StationID != "STN-002" {
			t.Errorf("dtos[1].StationID = %v, want STN-002", dtos[1].StationID)
		}
	})

	t.Run("handles empty slice", func(t *testing.T) {
		dtos := ToStationDTOs([]*domain.Station{})
		if len(dtos) != 0 {
			t.Errorf("DTOs length = %v, want 0", len(dtos))
		}
	})

	t.Run("skips nil stations in slice", func(t *testing.T) {
		stations := []*domain.Station{
			{
				StationID:    "STN-001",
				Name:         "Station 1",
				Zone:         "zone-a",
				StationType:  domain.StationTypePacking,
				Status:       domain.StationStatusActive,
				Capabilities: []domain.StationCapability{},
				Equipment:    []domain.StationEquipment{},
			},
			nil,
			{
				StationID:    "STN-003",
				Name:         "Station 3",
				Zone:         "zone-c",
				StationType:  domain.StationTypeConsolidation,
				Status:       domain.StationStatusActive,
				Capabilities: []domain.StationCapability{},
				Equipment:    []domain.StationEquipment{},
			},
		}

		dtos := ToStationDTOs(stations)

		if len(dtos) != 2 {
			t.Fatalf("DTOs length = %v, want 2 (nil should be skipped)", len(dtos))
		}
		if dtos[0].StationID != "STN-001" {
			t.Errorf("dtos[0].StationID = %v, want STN-001", dtos[0].StationID)
		}
		if dtos[1].StationID != "STN-003" {
			t.Errorf("dtos[1].StationID = %v, want STN-003", dtos[1].StationID)
		}
	})

	t.Run("handles nil slice", func(t *testing.T) {
		dtos := ToStationDTOs(nil)
		if dtos == nil {
			t.Error("ToStationDTOs(nil) should return empty slice, not nil")
		}
		if len(dtos) != 0 {
			t.Errorf("DTOs length = %v, want 0", len(dtos))
		}
	})
}

func TestStationEquipmentDTOMapping(t *testing.T) {
	t.Run("maps all equipment fields correctly", func(t *testing.T) {
		station := &domain.Station{
			StationID:    "STN-001",
			Name:         "Station with Equipment",
			Zone:         "zone-a",
			StationType:  domain.StationTypePacking,
			Status:       domain.StationStatusActive,
			Capabilities: []domain.StationCapability{},
			Equipment: []domain.StationEquipment{
				{
					EquipmentID:   "EQ-001",
					EquipmentType: "cold_storage",
					Status:        "maintenance",
				},
			},
		}

		dto := ToStationDTO(station)

		if len(dto.Equipment) != 1 {
			t.Fatalf("Equipment length = %v, want 1", len(dto.Equipment))
		}

		eq := dto.Equipment[0]
		if eq.EquipmentID != "EQ-001" {
			t.Errorf("EquipmentID = %v, want EQ-001", eq.EquipmentID)
		}
		if eq.EquipmentType != "cold_storage" {
			t.Errorf("EquipmentType = %v, want cold_storage", eq.EquipmentType)
		}
		if eq.Status != "maintenance" {
			t.Errorf("Status = %v, want maintenance", eq.Status)
		}
	})
}
