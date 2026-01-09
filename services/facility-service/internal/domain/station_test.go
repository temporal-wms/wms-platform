package domain

import (
	"testing"
)

// =============================================================================
// Type Validation Tests
// =============================================================================

func TestStationCapability_IsValid(t *testing.T) {
	tests := []struct {
		name       string
		capability StationCapability
		want       bool
	}{
		{"single_item is valid", CapabilitySingleItem, true},
		{"multi_item is valid", CapabilityMultiItem, true},
		{"gift_wrap is valid", CapabilityGiftWrap, true},
		{"hazmat is valid", CapabilityHazmat, true},
		{"oversized is valid", CapabilityOversized, true},
		{"fragile is valid", CapabilityFragile, true},
		{"cold_chain is valid", CapabilityColdChain, true},
		{"high_value is valid", CapabilityHighValue, true},
		{"unknown capability is invalid", StationCapability("unknown"), false},
		{"empty capability is invalid", StationCapability(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.capability.IsValid(); got != tt.want {
				t.Errorf("StationCapability.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStationStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status StationStatus
		want   bool
	}{
		{"active is valid", StationStatusActive, true},
		{"inactive is valid", StationStatusInactive, true},
		{"maintenance is valid", StationStatusMaintenance, true},
		{"unknown status is invalid", StationStatus("unknown"), false},
		{"empty status is invalid", StationStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("StationStatus.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStationType_IsValid(t *testing.T) {
	tests := []struct {
		name        string
		stationType StationType
		want        bool
	}{
		{"packing is valid", StationTypePacking, true},
		{"consolidation is valid", StationTypeConsolidation, true},
		{"shipping is valid", StationTypeShipping, true},
		{"receiving is valid", StationTypeReceiving, true},
		{"unknown type is invalid", StationType("unknown"), false},
		{"empty type is invalid", StationType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.stationType.IsValid(); got != tt.want {
				t.Errorf("StationType.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

// =============================================================================
// NewStation Tests
// =============================================================================

func TestNewStation(t *testing.T) {
	t.Run("creates station with valid parameters", func(t *testing.T) {
		station, err := NewStation("STN-001", "Packing Station 1", "zone-a", StationTypePacking, 5)
		if err != nil {
			t.Fatalf("NewStation() error = %v, want nil", err)
		}

		if station.StationID != "STN-001" {
			t.Errorf("StationID = %v, want %v", station.StationID, "STN-001")
		}
		if station.Name != "Packing Station 1" {
			t.Errorf("Name = %v, want %v", station.Name, "Packing Station 1")
		}
		if station.Zone != "zone-a" {
			t.Errorf("Zone = %v, want %v", station.Zone, "zone-a")
		}
		if station.StationType != StationTypePacking {
			t.Errorf("StationType = %v, want %v", station.StationType, StationTypePacking)
		}
		if station.Status != StationStatusActive {
			t.Errorf("Status = %v, want %v", station.Status, StationStatusActive)
		}
		if station.MaxConcurrentTasks != 5 {
			t.Errorf("MaxConcurrentTasks = %v, want %v", station.MaxConcurrentTasks, 5)
		}
		if station.CurrentTasks != 0 {
			t.Errorf("CurrentTasks = %v, want %v", station.CurrentTasks, 0)
		}
		if len(station.Capabilities) != 0 {
			t.Errorf("Capabilities length = %v, want %v", len(station.Capabilities), 0)
		}
		if len(station.Equipment) != 0 {
			t.Errorf("Equipment length = %v, want %v", len(station.Equipment), 0)
		}
	})

	t.Run("sets default MaxConcurrentTasks when zero or negative", func(t *testing.T) {
		station, err := NewStation("STN-002", "Station 2", "zone-b", StationTypePacking, 0)
		if err != nil {
			t.Fatalf("NewStation() error = %v", err)
		}
		if station.MaxConcurrentTasks != 1 {
			t.Errorf("MaxConcurrentTasks = %v, want 1 (default)", station.MaxConcurrentTasks)
		}

		station2, err := NewStation("STN-003", "Station 3", "zone-c", StationTypePacking, -5)
		if err != nil {
			t.Fatalf("NewStation() error = %v", err)
		}
		if station2.MaxConcurrentTasks != 1 {
			t.Errorf("MaxConcurrentTasks = %v, want 1 (default)", station2.MaxConcurrentTasks)
		}
	})

	t.Run("returns error for invalid station type", func(t *testing.T) {
		_, err := NewStation("STN-004", "Station 4", "zone-d", StationType("invalid"), 5)
		if err != ErrInvalidStationType {
			t.Errorf("NewStation() error = %v, want %v", err, ErrInvalidStationType)
		}
	})

	t.Run("emits StationCreatedEvent", func(t *testing.T) {
		station, _ := NewStation("STN-005", "Station 5", "zone-e", StationTypeShipping, 3)
		events := station.GetDomainEvents()
		if len(events) != 1 {
			t.Fatalf("Expected 1 domain event, got %d", len(events))
		}

		createdEvent, ok := events[0].(*StationCreatedEvent)
		if !ok {
			t.Fatalf("Expected StationCreatedEvent, got %T", events[0])
		}
		if createdEvent.StationID != "STN-005" {
			t.Errorf("Event StationID = %v, want %v", createdEvent.StationID, "STN-005")
		}
		if createdEvent.EventType() != "station.created" {
			t.Errorf("EventType() = %v, want %v", createdEvent.EventType(), "station.created")
		}
	})
}

// =============================================================================
// Capability Tests
// =============================================================================

func TestStation_HasCapability(t *testing.T) {
	station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
	station.Capabilities = []StationCapability{CapabilitySingleItem, CapabilityGiftWrap}

	t.Run("returns true for existing capability", func(t *testing.T) {
		if !station.HasCapability(CapabilitySingleItem) {
			t.Error("HasCapability(single_item) = false, want true")
		}
		if !station.HasCapability(CapabilityGiftWrap) {
			t.Error("HasCapability(gift_wrap) = false, want true")
		}
	})

	t.Run("returns false for non-existing capability", func(t *testing.T) {
		if station.HasCapability(CapabilityHazmat) {
			t.Error("HasCapability(hazmat) = true, want false")
		}
	})
}

func TestStation_HasAllCapabilities(t *testing.T) {
	station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
	station.Capabilities = []StationCapability{CapabilitySingleItem, CapabilityGiftWrap, CapabilityFragile}

	t.Run("returns true when station has all required capabilities", func(t *testing.T) {
		required := []StationCapability{CapabilitySingleItem, CapabilityGiftWrap}
		if !station.HasAllCapabilities(required) {
			t.Error("HasAllCapabilities() = false, want true")
		}
	})

	t.Run("returns true for empty required list", func(t *testing.T) {
		if !station.HasAllCapabilities([]StationCapability{}) {
			t.Error("HasAllCapabilities([]) = false, want true")
		}
	})

	t.Run("returns false when missing a capability", func(t *testing.T) {
		required := []StationCapability{CapabilitySingleItem, CapabilityHazmat}
		if station.HasAllCapabilities(required) {
			t.Error("HasAllCapabilities() = true, want false")
		}
	})
}

func TestStation_AddCapability(t *testing.T) {
	t.Run("adds valid capability", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		station.ClearDomainEvents()

		err := station.AddCapability(CapabilityHazmat)
		if err != nil {
			t.Fatalf("AddCapability() error = %v", err)
		}
		if !station.HasCapability(CapabilityHazmat) {
			t.Error("Station should have hazmat capability after adding")
		}
	})

	t.Run("emits StationCapabilityAddedEvent", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		station.ClearDomainEvents()

		station.AddCapability(CapabilityGiftWrap)
		events := station.GetDomainEvents()
		if len(events) != 1 {
			t.Fatalf("Expected 1 event, got %d", len(events))
		}

		addedEvent, ok := events[0].(*StationCapabilityAddedEvent)
		if !ok {
			t.Fatalf("Expected StationCapabilityAddedEvent, got %T", events[0])
		}
		if addedEvent.Capability != "gift_wrap" {
			t.Errorf("Event Capability = %v, want gift_wrap", addedEvent.Capability)
		}
	})

	t.Run("returns error for duplicate capability", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		station.AddCapability(CapabilityHazmat)

		err := station.AddCapability(CapabilityHazmat)
		if err != ErrCapabilityExists {
			t.Errorf("AddCapability() error = %v, want %v", err, ErrCapabilityExists)
		}
	})

	t.Run("returns error for invalid capability", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		err := station.AddCapability(StationCapability("invalid"))
		if err == nil {
			t.Error("AddCapability() should return error for invalid capability")
		}
	})
}

func TestStation_RemoveCapability(t *testing.T) {
	t.Run("removes existing capability", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		station.AddCapability(CapabilityHazmat)
		station.ClearDomainEvents()

		err := station.RemoveCapability(CapabilityHazmat)
		if err != nil {
			t.Fatalf("RemoveCapability() error = %v", err)
		}
		if station.HasCapability(CapabilityHazmat) {
			t.Error("Station should not have hazmat capability after removal")
		}
	})

	t.Run("emits StationCapabilityRemovedEvent", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		station.AddCapability(CapabilityFragile)
		station.ClearDomainEvents()

		station.RemoveCapability(CapabilityFragile)
		events := station.GetDomainEvents()
		if len(events) != 1 {
			t.Fatalf("Expected 1 event, got %d", len(events))
		}

		removedEvent, ok := events[0].(*StationCapabilityRemovedEvent)
		if !ok {
			t.Fatalf("Expected StationCapabilityRemovedEvent, got %T", events[0])
		}
		if removedEvent.EventType() != "station.capability.removed" {
			t.Errorf("EventType() = %v, want station.capability.removed", removedEvent.EventType())
		}
	})

	t.Run("returns error for non-existing capability", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		err := station.RemoveCapability(CapabilityHazmat)
		if err != ErrCapabilityNotFound {
			t.Errorf("RemoveCapability() error = %v, want %v", err, ErrCapabilityNotFound)
		}
	})
}

func TestStation_SetCapabilities(t *testing.T) {
	t.Run("replaces all capabilities", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		station.AddCapability(CapabilityHazmat)
		station.ClearDomainEvents()

		newCaps := []StationCapability{CapabilitySingleItem, CapabilityFragile}
		err := station.SetCapabilities(newCaps)
		if err != nil {
			t.Fatalf("SetCapabilities() error = %v", err)
		}

		if station.HasCapability(CapabilityHazmat) {
			t.Error("Station should not have hazmat capability")
		}
		if !station.HasCapability(CapabilitySingleItem) {
			t.Error("Station should have single_item capability")
		}
		if !station.HasCapability(CapabilityFragile) {
			t.Error("Station should have fragile capability")
		}
	})

	t.Run("emits StationCapabilitiesUpdatedEvent", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		station.ClearDomainEvents()

		station.SetCapabilities([]StationCapability{CapabilityGiftWrap})
		events := station.GetDomainEvents()
		if len(events) != 1 {
			t.Fatalf("Expected 1 event, got %d", len(events))
		}

		updatedEvent, ok := events[0].(*StationCapabilitiesUpdatedEvent)
		if !ok {
			t.Fatalf("Expected StationCapabilitiesUpdatedEvent, got %T", events[0])
		}
		if updatedEvent.EventType() != "station.capabilities.updated" {
			t.Errorf("EventType() = %v, want station.capabilities.updated", updatedEvent.EventType())
		}
	})

	t.Run("returns error for invalid capability in list", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		err := station.SetCapabilities([]StationCapability{CapabilitySingleItem, StationCapability("invalid")})
		if err == nil {
			t.Error("SetCapabilities() should return error for invalid capability")
		}
	})
}

func TestStation_GetCapabilityStrings(t *testing.T) {
	station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
	station.Capabilities = []StationCapability{CapabilitySingleItem, CapabilityGiftWrap}

	strings := station.GetCapabilityStrings()
	if len(strings) != 2 {
		t.Fatalf("Expected 2 strings, got %d", len(strings))
	}
	if strings[0] != "single_item" {
		t.Errorf("strings[0] = %v, want single_item", strings[0])
	}
	if strings[1] != "gift_wrap" {
		t.Errorf("strings[1] = %v, want gift_wrap", strings[1])
	}
}

// =============================================================================
// Task Management Tests
// =============================================================================

func TestStation_CanAcceptTask(t *testing.T) {
	t.Run("returns true when active and under capacity", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		if !station.CanAcceptTask() {
			t.Error("CanAcceptTask() = false, want true")
		}
	})

	t.Run("returns false when at capacity", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 2)
		station.CurrentTasks = 2
		if station.CanAcceptTask() {
			t.Error("CanAcceptTask() = true, want false (at capacity)")
		}
	})

	t.Run("returns false when inactive", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		station.Status = StationStatusInactive
		if station.CanAcceptTask() {
			t.Error("CanAcceptTask() = true, want false (inactive)")
		}
	})

	t.Run("returns false when in maintenance", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		station.Status = StationStatusMaintenance
		if station.CanAcceptTask() {
			t.Error("CanAcceptTask() = true, want false (maintenance)")
		}
	})
}

func TestStation_IncrementTasks(t *testing.T) {
	t.Run("increments task count", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		initialTasks := station.CurrentTasks

		err := station.IncrementTasks()
		if err != nil {
			t.Fatalf("IncrementTasks() error = %v", err)
		}
		if station.CurrentTasks != initialTasks+1 {
			t.Errorf("CurrentTasks = %v, want %v", station.CurrentTasks, initialTasks+1)
		}
	})

	t.Run("returns error when at capacity", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 2)
		station.CurrentTasks = 2

		err := station.IncrementTasks()
		if err != ErrStationAtCapacity {
			t.Errorf("IncrementTasks() error = %v, want %v", err, ErrStationAtCapacity)
		}
	})

	t.Run("returns error when not active", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		station.Status = StationStatusInactive

		err := station.IncrementTasks()
		if err != ErrStationNotActive {
			t.Errorf("IncrementTasks() error = %v, want %v", err, ErrStationNotActive)
		}
	})
}

func TestStation_DecrementTasks(t *testing.T) {
	t.Run("decrements task count", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		station.CurrentTasks = 3

		station.DecrementTasks()
		if station.CurrentTasks != 2 {
			t.Errorf("CurrentTasks = %v, want %v", station.CurrentTasks, 2)
		}
	})

	t.Run("does not go below zero", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		station.CurrentTasks = 0

		station.DecrementTasks()
		if station.CurrentTasks != 0 {
			t.Errorf("CurrentTasks = %v, want %v", station.CurrentTasks, 0)
		}
	})
}

func TestStation_GetAvailableCapacity(t *testing.T) {
	t.Run("returns available slots when active", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		station.CurrentTasks = 2

		capacity := station.GetAvailableCapacity()
		if capacity != 3 {
			t.Errorf("GetAvailableCapacity() = %v, want %v", capacity, 3)
		}
	})

	t.Run("returns 0 when inactive", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		station.Status = StationStatusInactive

		capacity := station.GetAvailableCapacity()
		if capacity != 0 {
			t.Errorf("GetAvailableCapacity() = %v, want %v", capacity, 0)
		}
	})

	t.Run("returns 0 when in maintenance", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		station.Status = StationStatusMaintenance

		capacity := station.GetAvailableCapacity()
		if capacity != 0 {
			t.Errorf("GetAvailableCapacity() = %v, want %v", capacity, 0)
		}
	})
}

// =============================================================================
// Worker Assignment Tests
// =============================================================================

func TestStation_AssignWorker(t *testing.T) {
	t.Run("assigns worker to active station", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		station.ClearDomainEvents()

		err := station.AssignWorker("worker-123")
		if err != nil {
			t.Fatalf("AssignWorker() error = %v", err)
		}
		if station.AssignedWorkerID != "worker-123" {
			t.Errorf("AssignedWorkerID = %v, want %v", station.AssignedWorkerID, "worker-123")
		}
	})

	t.Run("emits WorkerAssignedToStationEvent", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		station.ClearDomainEvents()

		station.AssignWorker("worker-456")
		events := station.GetDomainEvents()
		if len(events) != 1 {
			t.Fatalf("Expected 1 event, got %d", len(events))
		}

		assignedEvent, ok := events[0].(*WorkerAssignedToStationEvent)
		if !ok {
			t.Fatalf("Expected WorkerAssignedToStationEvent, got %T", events[0])
		}
		if assignedEvent.WorkerID != "worker-456" {
			t.Errorf("Event WorkerID = %v, want worker-456", assignedEvent.WorkerID)
		}
		if assignedEvent.EventType() != "station.worker.assigned" {
			t.Errorf("EventType() = %v, want station.worker.assigned", assignedEvent.EventType())
		}
	})

	t.Run("returns error for inactive station", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		station.Status = StationStatusInactive

		err := station.AssignWorker("worker-789")
		if err != ErrStationNotActive {
			t.Errorf("AssignWorker() error = %v, want %v", err, ErrStationNotActive)
		}
	})
}

func TestStation_UnassignWorker(t *testing.T) {
	station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
	station.AssignedWorkerID = "worker-123"

	station.UnassignWorker()
	if station.AssignedWorkerID != "" {
		t.Errorf("AssignedWorkerID = %v, want empty string", station.AssignedWorkerID)
	}
}

// =============================================================================
// Status Management Tests
// =============================================================================

func TestStation_SetStatus(t *testing.T) {
	t.Run("changes status to valid value", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		station.ClearDomainEvents()

		err := station.SetStatus(StationStatusMaintenance)
		if err != nil {
			t.Fatalf("SetStatus() error = %v", err)
		}
		if station.Status != StationStatusMaintenance {
			t.Errorf("Status = %v, want %v", station.Status, StationStatusMaintenance)
		}
	})

	t.Run("emits StationStatusChangedEvent", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		station.ClearDomainEvents()

		station.SetStatus(StationStatusInactive)
		events := station.GetDomainEvents()
		if len(events) != 1 {
			t.Fatalf("Expected 1 event, got %d", len(events))
		}

		statusEvent, ok := events[0].(*StationStatusChangedEvent)
		if !ok {
			t.Fatalf("Expected StationStatusChangedEvent, got %T", events[0])
		}
		if statusEvent.OldStatus != "active" {
			t.Errorf("OldStatus = %v, want active", statusEvent.OldStatus)
		}
		if statusEvent.NewStatus != "inactive" {
			t.Errorf("NewStatus = %v, want inactive", statusEvent.NewStatus)
		}
		if statusEvent.EventType() != "station.status.changed" {
			t.Errorf("EventType() = %v, want station.status.changed", statusEvent.EventType())
		}
	})

	t.Run("returns error for invalid status", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		err := station.SetStatus(StationStatus("invalid"))
		if err != ErrInvalidStationStatus {
			t.Errorf("SetStatus() error = %v, want %v", err, ErrInvalidStationStatus)
		}
	})
}

func TestStation_Activate(t *testing.T) {
	station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
	station.Status = StationStatusInactive

	err := station.Activate()
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	if station.Status != StationStatusActive {
		t.Errorf("Status = %v, want %v", station.Status, StationStatusActive)
	}
}

func TestStation_Deactivate(t *testing.T) {
	station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)

	err := station.Deactivate()
	if err != nil {
		t.Fatalf("Deactivate() error = %v", err)
	}
	if station.Status != StationStatusInactive {
		t.Errorf("Status = %v, want %v", station.Status, StationStatusInactive)
	}
}

func TestStation_SetMaintenance(t *testing.T) {
	station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)

	err := station.SetMaintenance()
	if err != nil {
		t.Fatalf("SetMaintenance() error = %v", err)
	}
	if station.Status != StationStatusMaintenance {
		t.Errorf("Status = %v, want %v", station.Status, StationStatusMaintenance)
	}
}

// =============================================================================
// Equipment Management Tests
// =============================================================================

func TestStation_AddEquipment(t *testing.T) {
	station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
	equipment := StationEquipment{
		EquipmentID:   "EQ-001",
		EquipmentType: "scale",
		Status:        "active",
	}

	station.AddEquipment(equipment)
	if len(station.Equipment) != 1 {
		t.Fatalf("Equipment length = %v, want 1", len(station.Equipment))
	}
	if station.Equipment[0].EquipmentID != "EQ-001" {
		t.Errorf("Equipment[0].EquipmentID = %v, want EQ-001", station.Equipment[0].EquipmentID)
	}
}

func TestStation_RemoveEquipment(t *testing.T) {
	station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
	station.Equipment = []StationEquipment{
		{EquipmentID: "EQ-001", EquipmentType: "scale", Status: "active"},
		{EquipmentID: "EQ-002", EquipmentType: "printer", Status: "active"},
	}

	t.Run("removes existing equipment", func(t *testing.T) {
		removed := station.RemoveEquipment("EQ-001")
		if !removed {
			t.Error("RemoveEquipment() = false, want true")
		}
		if len(station.Equipment) != 1 {
			t.Errorf("Equipment length = %v, want 1", len(station.Equipment))
		}
	})

	t.Run("returns false for non-existing equipment", func(t *testing.T) {
		removed := station.RemoveEquipment("EQ-999")
		if removed {
			t.Error("RemoveEquipment() = true, want false")
		}
	})
}

// =============================================================================
// Domain Event Tests
// =============================================================================

func TestStation_DomainEventManagement(t *testing.T) {
	t.Run("AddDomainEvent adds event", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		initialCount := len(station.GetDomainEvents())

		station.AddDomainEvent(&StationStatusChangedEvent{
			StationID: "STN-001",
			OldStatus: "active",
			NewStatus: "inactive",
		})

		if len(station.GetDomainEvents()) != initialCount+1 {
			t.Errorf("Event count = %v, want %v", len(station.GetDomainEvents()), initialCount+1)
		}
	})

	t.Run("ClearDomainEvents removes all events", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		station.ClearDomainEvents()

		if len(station.GetDomainEvents()) != 0 {
			t.Errorf("Event count after clear = %v, want 0", len(station.GetDomainEvents()))
		}
	})

	t.Run("GetDomainEvents returns all events", func(t *testing.T) {
		station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
		events := station.GetDomainEvents()

		if events == nil {
			t.Error("GetDomainEvents() returned nil")
		}
	})
}

func TestDomainEvents_OccurredAt(t *testing.T) {
	station, _ := NewStation("STN-001", "Station", "zone-a", StationTypePacking, 5)
	events := station.GetDomainEvents()

	if len(events) == 0 {
		t.Fatal("Expected at least one event from NewStation")
	}

	createdEvent := events[0].(*StationCreatedEvent)
	if createdEvent.OccurredAt().IsZero() {
		t.Error("OccurredAt() should not be zero")
	}
}
