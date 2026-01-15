package activities

import (
	"context"
	"fmt"

	"github.com/wms-platform/orchestrator/internal/activities/clients"
	"github.com/wms-platform/shared/pkg/tenant"
	"go.temporal.io/sdk/activity"
)

// CreateUnitsInput holds the input for creating units at receiving
type CreateUnitsInput struct {
	SKU        string `json:"sku"`
	ShipmentID string `json:"shipmentId"`
	LocationID string `json:"locationId"`
	Quantity   int    `json:"quantity"`
	CreatedBy  string `json:"createdBy"`
	// Multi-tenant context
	TenantID    string `json:"tenantId,omitempty"`
	FacilityID  string `json:"facilityId,omitempty"`
	WarehouseID string `json:"warehouseId,omitempty"`
	SellerID    string `json:"sellerId,omitempty"`
}

// CreateUnitsOutput holds the result of creating units
type CreateUnitsOutput struct {
	UnitIDs []string `json:"unitIds"`
	SKU     string   `json:"sku"`
	Count   int      `json:"count"`
}

// CreateUnits generates UUIDs for units at receiving
func (a *UnitActivities) CreateUnits(ctx context.Context, input CreateUnitsInput) (*CreateUnitsOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Creating units", "sku", input.SKU, "quantity", input.Quantity, "shipmentId", input.ShipmentID)

	// Set tenant context from input for HTTP client
	if input.TenantID != "" {
		ctx = tenant.WithTenantID(ctx, input.TenantID)
	}
	if input.FacilityID != "" {
		ctx = tenant.WithFacilityID(ctx, input.FacilityID)
	}
	if input.WarehouseID != "" {
		ctx = tenant.WithWarehouseID(ctx, input.WarehouseID)
	}
	if input.SellerID != "" {
		ctx = tenant.WithSellerID(ctx, input.SellerID)
	}

	result, err := a.clients.CreateUnits(ctx, input.SKU, input.ShipmentID, input.LocationID, input.Quantity, input.CreatedBy)
	if err != nil {
		logger.Error("Failed to create units", "error", err)
		return nil, fmt.Errorf("failed to create units: %w", err)
	}

	logger.Info("Units created successfully", "count", result.Count, "unitIds", result.UnitIDs)

	return &CreateUnitsOutput{
		UnitIDs: result.UnitIDs,
		SKU:     result.SKU,
		Count:   result.Count,
	}, nil
}

// ReserveUnitsInput holds the input for reserving units for an order
type ReserveUnitsInput struct {
	OrderID   string            `json:"orderId"`
	PathID    string            `json:"pathId"`
	Items     []ReserveItemSpec `json:"items"`
	HandlerID string            `json:"handlerId"`
	// Multi-tenant context
	TenantID    string `json:"tenantId,omitempty"`
	FacilityID  string `json:"facilityId,omitempty"`
	WarehouseID string `json:"warehouseId,omitempty"`
	SellerID    string `json:"sellerId,omitempty"`
}

// ReserveItemSpec specifies SKU and quantity to reserve
type ReserveItemSpec struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

// ReserveUnitsOutput holds the result of reserving units
type ReserveUnitsOutput struct {
	ReservedUnits []ReservedUnitInfo `json:"reservedUnits"`
	FailedItems   []FailedReserve    `json:"failedItems,omitempty"`
}

// ReservedUnitInfo holds info about a reserved unit
type ReservedUnitInfo struct {
	UnitID     string `json:"unitId"`
	SKU        string `json:"sku"`
	LocationID string `json:"locationId"`
}

// FailedReserve holds info about a failed reservation
type FailedReserve struct {
	SKU       string `json:"sku"`
	Requested int    `json:"requested"`
	Available int    `json:"available"`
	Reason    string `json:"reason"`
}

// ReserveUnits reserves specific units for an order with a path
func (a *UnitActivities) ReserveUnits(ctx context.Context, input ReserveUnitsInput) (*ReserveUnitsOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Reserving units", "orderId", input.OrderID, "pathId", input.PathID, "itemCount", len(input.Items))

	// Set tenant context from input for HTTP client
	if input.TenantID != "" {
		ctx = tenant.WithTenantID(ctx, input.TenantID)
	}
	if input.FacilityID != "" {
		ctx = tenant.WithFacilityID(ctx, input.FacilityID)
	}
	if input.WarehouseID != "" {
		ctx = tenant.WithWarehouseID(ctx, input.WarehouseID)
	}
	if input.SellerID != "" {
		ctx = tenant.WithSellerID(ctx, input.SellerID)
	}

	result, err := a.clients.ReserveUnits(ctx, input.OrderID, input.PathID, input.Items, input.HandlerID)
	if err != nil {
		logger.Error("Failed to reserve units", "error", err)
		return nil, fmt.Errorf("failed to reserve units: %w", err)
	}

	output := &ReserveUnitsOutput{
		ReservedUnits: make([]ReservedUnitInfo, len(result.ReservedUnits)),
		FailedItems:   make([]FailedReserve, len(result.FailedItems)),
	}

	for i, u := range result.ReservedUnits {
		output.ReservedUnits[i] = ReservedUnitInfo{
			UnitID:     u.UnitID,
			SKU:        u.SKU,
			LocationID: u.LocationID,
		}
	}

	for i, f := range result.FailedItems {
		output.FailedItems[i] = FailedReserve{
			SKU:       f.SKU,
			Requested: f.Requested,
			Available: f.Available,
			Reason:    f.Reason,
		}
	}

	logger.Info("Units reserved", "reservedCount", len(output.ReservedUnits), "failedCount", len(output.FailedItems))

	return output, nil
}

// GetUnitsForOrderInput holds the input for getting units for an order
type GetUnitsForOrderInput struct {
	OrderID string `json:"orderId"`
}

// UnitInfo holds information about a unit
type UnitInfo struct {
	UnitID     string `json:"unitId"`
	SKU        string `json:"sku"`
	Status     string `json:"status"`
	LocationID string `json:"locationId"`
}

// GetUnitsForOrderOutput holds the result of getting units for an order
type GetUnitsForOrderOutput struct {
	Units []UnitInfo `json:"units"`
}

// GetUnitsForOrder retrieves all units reserved for an order
func (a *UnitActivities) GetUnitsForOrder(ctx context.Context, input GetUnitsForOrderInput) (*GetUnitsForOrderOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Getting units for order", "orderId", input.OrderID)

	units, err := a.clients.GetUnitsForOrder(ctx, input.OrderID)
	if err != nil {
		logger.Error("Failed to get units for order", "error", err)
		return nil, fmt.Errorf("failed to get units for order: %w", err)
	}

	output := &GetUnitsForOrderOutput{
		Units: make([]UnitInfo, len(units)),
	}

	for i, u := range units {
		output.Units[i] = UnitInfo{
			UnitID:     u.UnitID,
			SKU:        u.SKU,
			Status:     u.Status,
			LocationID: u.LocationID,
		}
	}

	logger.Info("Units retrieved", "count", len(output.Units))

	return output, nil
}

// ConfirmUnitPickInput holds the input for confirming a unit pick
type ConfirmUnitPickInput struct {
	UnitID    string `json:"unitId"`
	ToteID    string `json:"toteId"`
	PickerID  string `json:"pickerId"`
	StationID string `json:"stationId"`
}

// ConfirmUnitPick confirms that a specific unit has been picked
func (a *UnitActivities) ConfirmUnitPick(ctx context.Context, input ConfirmUnitPickInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Confirming unit pick", "unitId", input.UnitID, "toteId", input.ToteID, "pickerId", input.PickerID)

	err := a.clients.ConfirmUnitPick(ctx, input.UnitID, input.ToteID, input.PickerID, input.StationID)
	if err != nil {
		logger.Error("Failed to confirm unit pick", "error", err)
		return fmt.Errorf("failed to confirm unit pick: %w", err)
	}

	logger.Info("Unit pick confirmed", "unitId", input.UnitID)

	return nil
}

// ConfirmUnitConsolidationInput holds the input for confirming consolidation
type ConfirmUnitConsolidationInput struct {
	UnitID         string `json:"unitId"`
	DestinationBin string `json:"destinationBin"`
	WorkerID       string `json:"workerId"`
	StationID      string `json:"stationId"`
}

// ConfirmUnitConsolidation confirms that a specific unit has been consolidated
func (a *UnitActivities) ConfirmUnitConsolidation(ctx context.Context, input ConfirmUnitConsolidationInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Confirming unit consolidation", "unitId", input.UnitID, "destinationBin", input.DestinationBin)

	err := a.clients.ConfirmUnitConsolidation(ctx, input.UnitID, input.DestinationBin, input.WorkerID, input.StationID)
	if err != nil {
		logger.Error("Failed to confirm unit consolidation", "error", err)
		return fmt.Errorf("failed to confirm unit consolidation: %w", err)
	}

	logger.Info("Unit consolidation confirmed", "unitId", input.UnitID)

	return nil
}

// ConfirmUnitPackedInput holds the input for confirming packing
type ConfirmUnitPackedInput struct {
	UnitID    string `json:"unitId"`
	PackageID string `json:"packageId"`
	PackerID  string `json:"packerId"`
	StationID string `json:"stationId"`
}

// ConfirmUnitPacked confirms that a specific unit has been packed
func (a *UnitActivities) ConfirmUnitPacked(ctx context.Context, input ConfirmUnitPackedInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Confirming unit packed", "unitId", input.UnitID, "packageId", input.PackageID)

	err := a.clients.ConfirmUnitPacked(ctx, input.UnitID, input.PackageID, input.PackerID, input.StationID)
	if err != nil {
		logger.Error("Failed to confirm unit packed", "error", err)
		return fmt.Errorf("failed to confirm unit packed: %w", err)
	}

	logger.Info("Unit packed confirmed", "unitId", input.UnitID)

	return nil
}

// ConfirmUnitShippedInput holds the input for confirming shipping
type ConfirmUnitShippedInput struct {
	UnitID         string `json:"unitId"`
	ShipmentID     string `json:"shipmentId"`
	TrackingNumber string `json:"trackingNumber"`
	HandlerID      string `json:"handlerId"`
}

// ConfirmUnitShipped confirms that a specific unit has been shipped
func (a *UnitActivities) ConfirmUnitShipped(ctx context.Context, input ConfirmUnitShippedInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Confirming unit shipped", "unitId", input.UnitID, "trackingNumber", input.TrackingNumber)

	err := a.clients.ConfirmUnitShipped(ctx, input.UnitID, input.ShipmentID, input.TrackingNumber, input.HandlerID)
	if err != nil {
		logger.Error("Failed to confirm unit shipped", "error", err)
		return fmt.Errorf("failed to confirm unit shipped: %w", err)
	}

	logger.Info("Unit shipped confirmed", "unitId", input.UnitID)

	return nil
}

// CreateUnitExceptionInput holds the input for creating a unit exception
type CreateUnitExceptionInput struct {
	UnitID        string `json:"unitId"`
	ExceptionType string `json:"exceptionType"`
	Stage         string `json:"stage"`
	Description   string `json:"description"`
	StationID     string `json:"stationId"`
	ReportedBy    string `json:"reportedBy"`
}

// CreateUnitExceptionOutput holds the result of creating an exception
type CreateUnitExceptionOutput struct {
	ExceptionID string `json:"exceptionId"`
	UnitID      string `json:"unitId"`
}

// CreateUnitException creates an exception for a failed unit
func (a *UnitActivities) CreateUnitException(ctx context.Context, input CreateUnitExceptionInput) (*CreateUnitExceptionOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Creating unit exception", "unitId", input.UnitID, "exceptionType", input.ExceptionType, "stage", input.Stage)

	exception, err := a.clients.CreateUnitException(ctx, input.UnitID, input.ExceptionType, input.Stage, input.Description, input.StationID, input.ReportedBy)
	if err != nil {
		logger.Error("Failed to create unit exception", "error", err)
		return nil, fmt.Errorf("failed to create unit exception: %w", err)
	}

	logger.Info("Unit exception created", "exceptionId", exception.ExceptionID)

	return &CreateUnitExceptionOutput{
		ExceptionID: exception.ExceptionID,
		UnitID:      input.UnitID,
	}, nil
}

// GetUnitAuditTrailInput holds the input for getting a unit's audit trail
type GetUnitAuditTrailInput struct {
	UnitID string `json:"unitId"`
}

// UnitMovementInfo holds information about a unit movement
type UnitMovementInfo struct {
	MovementID     string `json:"movementId"`
	FromLocationID string `json:"fromLocationId"`
	ToLocationID   string `json:"toLocationId"`
	FromStatus     string `json:"fromStatus"`
	ToStatus       string `json:"toStatus"`
	StationID      string `json:"stationId"`
	HandlerID      string `json:"handlerId"`
	Timestamp      string `json:"timestamp"`
	Notes          string `json:"notes"`
}

// GetUnitAuditTrailOutput holds the unit's movement history
type GetUnitAuditTrailOutput struct {
	UnitID    string             `json:"unitId"`
	Movements []UnitMovementInfo `json:"movements"`
}

// GetUnitAuditTrail retrieves the full movement history for a unit
func (a *UnitActivities) GetUnitAuditTrail(ctx context.Context, input GetUnitAuditTrailInput) (*GetUnitAuditTrailOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Getting unit audit trail", "unitId", input.UnitID)

	movements, err := a.clients.GetUnitAuditTrail(ctx, input.UnitID)
	if err != nil {
		logger.Error("Failed to get unit audit trail", "error", err)
		return nil, fmt.Errorf("failed to get unit audit trail: %w", err)
	}

	// Convert client types to activity types
	outputMovements := make([]UnitMovementInfo, len(movements))
	for i, m := range movements {
		outputMovements[i] = UnitMovementInfo{
			MovementID:     m.MovementID,
			FromLocationID: m.FromLocationID,
			ToLocationID:   m.ToLocationID,
			FromStatus:     m.FromStatus,
			ToStatus:       m.ToStatus,
			StationID:      m.StationID,
			HandlerID:      m.HandlerID,
			Timestamp:      m.Timestamp,
			Notes:          m.Notes,
		}
	}

	output := &GetUnitAuditTrailOutput{
		UnitID:    input.UnitID,
		Movements: outputMovements,
	}

	logger.Info("Unit audit trail retrieved", "movementCount", len(movements))

	return output, nil
}

// PersistProcessPathInput holds the input for persisting a process path
type PersistProcessPathInput struct {
	OrderID      string            `json:"orderId"`
	Items        []ProcessPathItem `json:"items"` // Uses ProcessPathItem from process_path_activities.go
	GiftWrap     bool              `json:"giftWrap"`
	TotalValue   float64           `json:"totalValue"`
}

// PersistProcessPathOutput holds the result of persisting a process path
type PersistProcessPathOutput struct {
	PathID  string `json:"pathId"`
	OrderID string `json:"orderId"`
}

// PersistProcessPath saves the process path to ensure all units follow the same path
func (a *UnitActivities) PersistProcessPath(ctx context.Context, input PersistProcessPathInput) (*PersistProcessPathOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Persisting process path", "orderId", input.OrderID, "itemCount", len(input.Items))

	// Convert activity items to client items
	clientItems := make([]clients.ProcessPathItem, len(input.Items))
	for i, item := range input.Items {
		clientItems[i] = clients.ProcessPathItem{
			SKU:               item.SKU,
			Quantity:          item.Quantity,
			Weight:            item.Weight,
			IsFragile:         item.IsFragile,
			IsHazmat:          item.IsHazmat,
			RequiresColdChain: item.RequiresColdChain,
		}
	}

	result, err := a.clients.PersistProcessPath(ctx, input.OrderID, clientItems, input.GiftWrap, input.TotalValue)
	if err != nil {
		logger.Error("Failed to persist process path", "error", err)
		return nil, fmt.Errorf("failed to persist process path: %w", err)
	}

	logger.Info("Process path persisted", "pathId", result.PathID)

	return &PersistProcessPathOutput{
		PathID:  result.PathID,
		OrderID: input.OrderID,
	}, nil
}

// GetProcessPathInput holds the input for retrieving a process path
type GetProcessPathInput struct {
	PathID string `json:"pathId"`
}

// ProcessPathInfo holds information about a process path
type ProcessPathInfo struct {
	PathID                string   `json:"pathId"`
	OrderID               string   `json:"orderId"`
	Requirements          []string `json:"requirements"`
	ConsolidationRequired bool     `json:"consolidationRequired"`
	GiftWrapRequired      bool     `json:"giftWrapRequired"`
	SpecialHandling       []string `json:"specialHandling"`
	TargetStationID       string   `json:"targetStationId,omitempty"`
}

// GetProcessPath retrieves a persisted process path
func (a *UnitActivities) GetProcessPath(ctx context.Context, input GetProcessPathInput) (*ProcessPathInfo, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Getting process path", "pathId", input.PathID)

	path, err := a.clients.GetProcessPath(ctx, input.PathID)
	if err != nil {
		logger.Error("Failed to get process path", "error", err)
		return nil, fmt.Errorf("failed to get process path: %w", err)
	}

	logger.Info("Process path retrieved", "pathId", path.PathID, "orderId", path.OrderID)

	// Convert client type to activity type
	return &ProcessPathInfo{
		PathID:                path.PathID,
		OrderID:               path.OrderID,
		Requirements:          path.Requirements,
		ConsolidationRequired: path.ConsolidationRequired,
		GiftWrapRequired:      path.GiftWrapRequired,
		SpecialHandling:       path.SpecialHandling,
		TargetStationID:       path.TargetStationID,
	}, nil
}

// ReleaseUnitsInput holds the input for releasing unit reservations
type ReleaseUnitsInput struct {
	OrderID string `json:"orderId"`
	Reason  string `json:"reason"`
	// Multi-tenant context
	TenantID    string `json:"tenantId,omitempty"`
	FacilityID  string `json:"facilityId,omitempty"`
	WarehouseID string `json:"warehouseId,omitempty"`
	SellerID    string `json:"sellerId,omitempty"`
}

// ReleaseUnits releases all unit reservations for an order (compensation activity)
func (a *UnitActivities) ReleaseUnits(ctx context.Context, input ReleaseUnitsInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Releasing unit reservations", "orderId", input.OrderID, "reason", input.Reason)

	// Set tenant context from input for HTTP client
	if input.TenantID != "" {
		ctx = tenant.WithTenantID(ctx, input.TenantID)
	}
	if input.FacilityID != "" {
		ctx = tenant.WithFacilityID(ctx, input.FacilityID)
	}
	if input.WarehouseID != "" {
		ctx = tenant.WithWarehouseID(ctx, input.WarehouseID)
	}
	if input.SellerID != "" {
		ctx = tenant.WithSellerID(ctx, input.SellerID)
	}

	err := a.clients.ReleaseUnits(ctx, input.OrderID)
	if err != nil {
		logger.Error("Failed to release unit reservations", "orderId", input.OrderID, "error", err)
		return fmt.Errorf("failed to release unit reservations: %w", err)
	}

	logger.Info("Unit reservations released successfully", "orderId", input.OrderID)
	return nil
}
