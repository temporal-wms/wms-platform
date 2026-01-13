package activities

import (
	"context"
	"fmt"

	"github.com/wms-platform/orchestrator/internal/activities/clients"
	"go.temporal.io/sdk/activity"
)

// EquipmentActivities contains equipment availability and reservation activities
type EquipmentActivities struct {
	clients *clients.ServiceClients
}

// NewEquipmentActivities creates a new EquipmentActivities instance
func NewEquipmentActivities(clients *clients.ServiceClients) *EquipmentActivities {
	return &EquipmentActivities{
		clients: clients,
	}
}

// CheckEquipmentAvailabilityInput represents input for checking equipment availability
type CheckEquipmentAvailabilityInput struct {
	EquipmentTypes []string `json:"equipmentTypes"` // Types of equipment needed
	Zone           string   `json:"zone,omitempty"`
	Quantity       int      `json:"quantity"` // Number of units needed
}

// CheckEquipmentAvailabilityResult represents the result of checking equipment availability
type CheckEquipmentAvailabilityResult struct {
	AvailableEquipment    map[string]int `json:"availableEquipment"`    // Type -> available count
	InsufficientEquipment []string       `json:"insufficientEquipment"` // Types with insufficient availability
	AllAvailable          bool           `json:"allAvailable"`
	Success               bool           `json:"success"`
}

// CheckEquipmentAvailability checks if required equipment is available
func (a *EquipmentActivities) CheckEquipmentAvailability(ctx context.Context, input CheckEquipmentAvailabilityInput) (*CheckEquipmentAvailabilityResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Checking equipment availability",
		"equipmentTypes", input.EquipmentTypes,
		"zone", input.Zone,
		"quantity", input.Quantity,
	)

	if len(input.EquipmentTypes) == 0 {
		// No equipment required
		return &CheckEquipmentAvailabilityResult{
			AvailableEquipment:    make(map[string]int),
			InsufficientEquipment: []string{},
			AllAvailable:          true,
			Success:               true,
		}, nil
	}

	// Call facility service to check equipment availability
	req := &clients.CheckEquipmentAvailabilityRequest{
		EquipmentTypes: input.EquipmentTypes,
		Zone:           input.Zone,
		RequiredCount:  input.Quantity,
	}

	availability, err := a.clients.CheckEquipmentAvailability(ctx, req)
	if err != nil {
		logger.Error("Failed to check equipment availability",
			"equipmentTypes", input.EquipmentTypes,
			"error", err,
		)
		return &CheckEquipmentAvailabilityResult{
			AvailableEquipment:    make(map[string]int),
			InsufficientEquipment: input.EquipmentTypes,
			AllAvailable:          false,
			Success:               false,
		}, fmt.Errorf("failed to check equipment availability: %w", err)
	}

	// Check if all equipment types have sufficient availability
	insufficientEquipment := make([]string, 0)
	allAvailable := true

	for _, equipType := range input.EquipmentTypes {
		available, exists := availability.AvailableEquipment[equipType]
		if !exists || available < input.Quantity {
			insufficientEquipment = append(insufficientEquipment, equipType)
			allAvailable = false
		}
	}

	logger.Info("Equipment availability check complete",
		"allAvailable", allAvailable,
		"insufficientEquipment", insufficientEquipment,
		"availableEquipment", availability.AvailableEquipment,
	)

	return &CheckEquipmentAvailabilityResult{
		AvailableEquipment:    availability.AvailableEquipment,
		InsufficientEquipment: insufficientEquipment,
		AllAvailable:          allAvailable,
		Success:               true,
	}, nil
}

// ReserveEquipmentInput represents input for reserving equipment
type ReserveEquipmentInput struct {
	EquipmentType string `json:"equipmentType"`
	OrderID       string `json:"orderId"`
	Quantity      int    `json:"quantity"`
	Zone          string `json:"zone,omitempty"`
	ReservationID string `json:"reservationId"`
}

// ReserveEquipmentResult represents the result of reserving equipment
type ReserveEquipmentResult struct {
	ReservationID        string   `json:"reservationId"`
	EquipmentType        string   `json:"equipmentType"`
	ReservedEquipmentIDs []string `json:"reservedEquipmentIds"`
	Quantity             int      `json:"quantity"`
	Success              bool     `json:"success"`
}

// ReserveEquipment reserves specific equipment for an order
func (a *EquipmentActivities) ReserveEquipment(ctx context.Context, input ReserveEquipmentInput) (*ReserveEquipmentResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Reserving equipment",
		"equipmentType", input.EquipmentType,
		"orderId", input.OrderID,
		"quantity", input.Quantity,
		"reservationId", input.ReservationID,
	)

	// Call facility service to reserve equipment
	req := &clients.ReserveEquipmentRequest{
		EquipmentType: input.EquipmentType,
		OrderID:       input.OrderID,
		Quantity:      input.Quantity,
		Zone:          input.Zone,
		ReservationID: input.ReservationID,
	}

	reservation, err := a.clients.ReserveEquipment(ctx, req)
	if err != nil {
		logger.Error("Failed to reserve equipment",
			"equipmentType", input.EquipmentType,
			"orderId", input.OrderID,
			"error", err,
		)
		return &ReserveEquipmentResult{
			ReservationID: input.ReservationID,
			EquipmentType: input.EquipmentType,
			Quantity:      0,
			Success:       false,
		}, fmt.Errorf("failed to reserve equipment: %w", err)
	}

	logger.Info("Equipment reserved",
		"equipmentType", input.EquipmentType,
		"orderId", input.OrderID,
		"reservationId", reservation.ReservationID,
		"quantity", len(reservation.ReservedEquipmentIDs),
	)

	return &ReserveEquipmentResult{
		ReservationID:        reservation.ReservationID,
		EquipmentType:        reservation.EquipmentType,
		ReservedEquipmentIDs: reservation.ReservedEquipmentIDs,
		Quantity:             len(reservation.ReservedEquipmentIDs),
		Success:              true,
	}, nil
}

// ReleaseEquipmentInput represents input for releasing equipment
type ReleaseEquipmentInput struct {
	ReservationID string `json:"reservationId"`
	EquipmentType string `json:"equipmentType"`
	OrderID       string `json:"orderId"`
	Reason        string `json:"reason,omitempty"`
}

// ReleaseEquipment releases previously reserved equipment
func (a *EquipmentActivities) ReleaseEquipment(ctx context.Context, input ReleaseEquipmentInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Releasing equipment",
		"equipmentType", input.EquipmentType,
		"orderId", input.OrderID,
		"reservationId", input.ReservationID,
		"reason", input.Reason,
	)

	// Call facility service to release equipment
	req := &clients.ReleaseEquipmentRequest{
		ReservationID: input.ReservationID,
		EquipmentType: input.EquipmentType,
		OrderID:       input.OrderID,
		Reason:        input.Reason,
	}

	err := a.clients.ReleaseEquipment(ctx, req)
	if err != nil {
		logger.Error("Failed to release equipment",
			"equipmentType", input.EquipmentType,
			"orderId", input.OrderID,
			"reservationId", input.ReservationID,
			"error", err,
		)
		return fmt.Errorf("failed to release equipment: %w", err)
	}

	logger.Info("Equipment released",
		"equipmentType", input.EquipmentType,
		"orderId", input.OrderID,
		"reservationId", input.ReservationID,
	)

	return nil
}

// GetEquipmentByTypeInput represents input for getting equipment by type
type GetEquipmentByTypeInput struct {
	EquipmentType string `json:"equipmentType"`
	Zone          string `json:"zone,omitempty"`
	Status        string `json:"status,omitempty"` // available, in_use, maintenance
}

// GetEquipmentByType retrieves equipment by type
func (a *EquipmentActivities) GetEquipmentByType(ctx context.Context, input GetEquipmentByTypeInput) ([]clients.Equipment, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Getting equipment by type",
		"equipmentType", input.EquipmentType,
		"zone", input.Zone,
		"status", input.Status,
	)

	req := &clients.GetEquipmentByTypeRequest{
		EquipmentType: input.EquipmentType,
		Zone:          input.Zone,
		Status:        input.Status,
	}

	equipment, err := a.clients.GetEquipmentByType(ctx, req)
	if err != nil {
		logger.Error("Failed to get equipment by type",
			"equipmentType", input.EquipmentType,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get equipment by type: %w", err)
	}

	logger.Info("Equipment retrieved", "count", len(equipment))
	return equipment, nil
}
