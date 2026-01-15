package activities

import (
	"context"
	"fmt"

	"github.com/wms-platform/orchestrator/internal/activities/clients"
	"go.temporal.io/sdk/activity"
)

// ReserveInventoryInput holds the input for reserving inventory
type ReserveInventoryInput struct {
	OrderID string                    `json:"orderId"`
	Items   []ReserveInventoryItem    `json:"items"`
}

// ReserveInventoryItem represents an item to reserve
type ReserveInventoryItem struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

// ReserveInventory creates reservations in the inventory service
func (a *InventoryActivities) ReserveInventory(ctx context.Context, input ReserveInventoryInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Reserving inventory", "orderId", input.OrderID, "itemCount", len(input.Items))

	// Convert items to client request format
	items := make([]clients.ReserveItemRequest, len(input.Items))
	for i, item := range input.Items {
		items[i] = clients.ReserveItemRequest{
			SKU:      item.SKU,
			Quantity: item.Quantity,
		}
	}

	req := &clients.ReserveInventoryRequest{
		OrderID: input.OrderID,
		Items:   items,
	}

	err := a.clients.ReserveInventory(ctx, req)
	if err != nil {
		logger.Error("Failed to reserve inventory", "orderId", input.OrderID, "error", err)
		return fmt.Errorf("failed to reserve inventory: %w", err)
	}

	logger.Info("Inventory reserved successfully", "orderId", input.OrderID)
	return nil
}

// ConfirmInventoryPickInput holds the input for confirming inventory picks
type ConfirmInventoryPickInput struct {
	OrderID     string                     `json:"orderId"`
	PickedItems []ConfirmInventoryPickItem `json:"pickedItems"`
}

// ConfirmInventoryPickItem represents a picked item for inventory decrement
type ConfirmInventoryPickItem struct {
	SKU        string `json:"sku"`
	Quantity   int    `json:"quantity"`
	LocationID string `json:"locationId"`
}

// ConfirmInventoryPick decrements inventory for all picked items
func (a *InventoryActivities) ConfirmInventoryPick(ctx context.Context, input ConfirmInventoryPickInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Confirming inventory pick", "orderId", input.OrderID, "itemCount", len(input.PickedItems))

	var lastError error
	successCount := 0
	totalItems := len(input.PickedItems)

	for i, item := range input.PickedItems {
		// Record heartbeat for long-running inventory operations
		activity.RecordHeartbeat(ctx, fmt.Sprintf("Processing inventory item %d/%d", i+1, totalItems))

		req := &clients.PickInventoryRequest{
			OrderID:    input.OrderID,
			LocationID: item.LocationID,
			Quantity:   item.Quantity,
			CreatedBy:  "orchestrator",
		}

		err := a.clients.PickInventory(ctx, item.SKU, req)
		if err != nil {
			logger.Error("Failed to pick inventory",
				"orderId", input.OrderID,
				"sku", item.SKU,
				"error", err)
			lastError = err
			// Continue with other items, don't fail the whole operation
			continue
		}

		successCount++
		logger.Info("Inventory picked successfully",
			"orderId", input.OrderID,
			"sku", item.SKU,
			"quantity", item.Quantity,
			"locationId", item.LocationID,
			"progress", fmt.Sprintf("%d/%d", i+1, totalItems),
		)
	}

	logger.Info("Inventory pick confirmation complete",
		"orderId", input.OrderID,
		"successCount", successCount,
		"totalItems", len(input.PickedItems),
	)

	// Return error only if all items failed
	if successCount == 0 && lastError != nil {
		return fmt.Errorf("failed to pick any inventory items: %w", lastError)
	}

	return nil
}

// ReleaseInventoryReservation releases inventory reservations for an order
func (a *InventoryActivities) ReleaseInventoryReservation(ctx context.Context, orderID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Releasing inventory reservation", "orderId", orderID)

	err := a.clients.ReleaseInventoryReservation(ctx, orderID)
	if err != nil {
		logger.Error("Failed to release inventory reservation", "orderId", orderID, "error", err)
		return fmt.Errorf("failed to release inventory reservation: %w", err)
	}

	logger.Info("Inventory reservation released successfully", "orderId", orderID)
	return nil
}

// StageInventoryInput holds the input for staging inventory (soft to hard allocation)
type StageInventoryInput struct {
	OrderID           string                `json:"orderId"`
	StagingLocationID string                `json:"stagingLocationId"`
	StagedBy          string                `json:"stagedBy"`
	Items             []StageInventoryItem  `json:"items"`
}

// StageInventoryItem represents an item to be staged
type StageInventoryItem struct {
	SKU           string `json:"sku"`
	ReservationID string `json:"reservationId"`
}

// StageInventoryOutput holds the output from staging inventory
type StageInventoryOutput struct {
	StagedItems   []StagedItem `json:"stagedItems"`
	FailedItems   []string     `json:"failedItems,omitempty"`
	AllocationIDs []string     `json:"allocationIds"`
}

// StagedItem represents a successfully staged item
type StagedItem struct {
	SKU          string `json:"sku"`
	AllocationID string `json:"allocationId"`
}

// StageInventory converts soft reservations to hard allocations (physical staging)
func (a *InventoryActivities) StageInventory(ctx context.Context, input StageInventoryInput) (*StageInventoryOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Staging inventory", "orderId", input.OrderID, "itemCount", len(input.Items))

	output := &StageInventoryOutput{
		StagedItems:   make([]StagedItem, 0),
		FailedItems:   make([]string, 0),
		AllocationIDs: make([]string, 0),
	}

	var lastError error
	totalItems := len(input.Items)
	for i, item := range input.Items {
		// Record heartbeat for long-running staging operations
		activity.RecordHeartbeat(ctx, fmt.Sprintf("Staging inventory item %d/%d", i+1, totalItems))

		req := &clients.StageInventoryRequest{
			ReservationID:     item.ReservationID,
			StagingLocationID: input.StagingLocationID,
			StagedBy:          input.StagedBy,
		}

		err := a.clients.StageInventory(ctx, item.SKU, req)
		if err != nil {
			logger.Error("Failed to stage inventory",
				"orderId", input.OrderID,
				"sku", item.SKU,
				"reservationId", item.ReservationID,
				"error", err)
			output.FailedItems = append(output.FailedItems, item.SKU)
			lastError = err
			continue
		}

		// Note: In a real implementation, we'd get the allocationID from the response
		// For now, we'll use a composite ID
		allocationID := fmt.Sprintf("%s-%s", input.OrderID, item.SKU)
		output.StagedItems = append(output.StagedItems, StagedItem{
			SKU:          item.SKU,
			AllocationID: allocationID,
		})
		output.AllocationIDs = append(output.AllocationIDs, allocationID)

		logger.Info("Inventory staged successfully",
			"orderId", input.OrderID,
			"sku", item.SKU,
			"allocationId", allocationID,
		)
	}

	logger.Info("Inventory staging complete",
		"orderId", input.OrderID,
		"stagedCount", len(output.StagedItems),
		"failedCount", len(output.FailedItems),
	)

	if len(output.StagedItems) == 0 && lastError != nil {
		return nil, fmt.Errorf("failed to stage any inventory items: %w", lastError)
	}

	return output, nil
}

// PackInventoryInput holds the input for marking inventory as packed
type PackInventoryInput struct {
	OrderID       string              `json:"orderId"`
	PackedBy      string              `json:"packedBy"`
	Items         []PackInventoryItem `json:"items"`
}

// PackInventoryItem represents an item to be marked as packed
type PackInventoryItem struct {
	SKU          string `json:"sku"`
	AllocationID string `json:"allocationId"`
}

// PackInventory marks hard allocations as packed
func (a *InventoryActivities) PackInventory(ctx context.Context, input PackInventoryInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Packing inventory", "orderId", input.OrderID, "itemCount", len(input.Items))

	var lastError error
	successCount := 0
	totalItems := len(input.Items)

	for i, item := range input.Items {
		// Record heartbeat for long-running packing operations
		activity.RecordHeartbeat(ctx, fmt.Sprintf("Packing inventory item %d/%d", i+1, totalItems))

		req := &clients.PackInventoryRequest{
			AllocationID: item.AllocationID,
			PackedBy:     input.PackedBy,
		}

		err := a.clients.PackInventory(ctx, item.SKU, req)
		if err != nil {
			logger.Error("Failed to pack inventory",
				"orderId", input.OrderID,
				"sku", item.SKU,
				"allocationId", item.AllocationID,
				"error", err)
			lastError = err
			continue
		}

		successCount++
		logger.Info("Inventory packed successfully",
			"orderId", input.OrderID,
			"sku", item.SKU,
			"allocationId", item.AllocationID,
		)
	}

	logger.Info("Inventory packing complete",
		"orderId", input.OrderID,
		"successCount", successCount,
		"totalItems", len(input.Items),
	)

	if successCount == 0 && lastError != nil {
		return fmt.Errorf("failed to pack any inventory items: %w", lastError)
	}

	return nil
}

// ShipInventoryInput holds the input for shipping inventory
type ShipInventoryInput struct {
	OrderID string              `json:"orderId"`
	Items   []ShipInventoryItem `json:"items"`
}

// ShipInventoryItem represents an item to be shipped
type ShipInventoryItem struct {
	SKU          string `json:"sku"`
	AllocationID string `json:"allocationId"`
}

// ShipInventory ships hard allocations (removes inventory from system)
func (a *InventoryActivities) ShipInventory(ctx context.Context, input ShipInventoryInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Shipping inventory", "orderId", input.OrderID, "itemCount", len(input.Items))

	var lastError error
	successCount := 0
	totalItems := len(input.Items)

	for i, item := range input.Items {
		// Record heartbeat for long-running shipping operations
		activity.RecordHeartbeat(ctx, fmt.Sprintf("Shipping inventory item %d/%d", i+1, totalItems))

		req := &clients.ShipInventoryRequest{
			AllocationID: item.AllocationID,
		}

		err := a.clients.ShipInventory(ctx, item.SKU, req)
		if err != nil {
			logger.Error("Failed to ship inventory",
				"orderId", input.OrderID,
				"sku", item.SKU,
				"allocationId", item.AllocationID,
				"error", err)
			lastError = err
			continue
		}

		successCount++
		logger.Info("Inventory shipped successfully",
			"orderId", input.OrderID,
			"sku", item.SKU,
			"allocationId", item.AllocationID,
		)
	}

	logger.Info("Inventory shipping complete",
		"orderId", input.OrderID,
		"successCount", successCount,
		"totalItems", len(input.Items),
	)

	if successCount == 0 && lastError != nil {
		return fmt.Errorf("failed to ship any inventory items: %w", lastError)
	}

	return nil
}

// ReturnInventoryToShelfInput holds the input for returning inventory to shelf
type ReturnInventoryToShelfInput struct {
	OrderID    string                       `json:"orderId"`
	ReturnedBy string                       `json:"returnedBy"`
	Reason     string                       `json:"reason"`
	Items      []ReturnInventoryToShelfItem `json:"items"`
}

// ReturnInventoryToShelfItem represents an item to be returned to shelf
type ReturnInventoryToShelfItem struct {
	SKU          string `json:"sku"`
	AllocationID string `json:"allocationId"`
}

// ReturnInventoryToShelf returns hard allocated inventory back to available stock
func (a *InventoryActivities) ReturnInventoryToShelf(ctx context.Context, input ReturnInventoryToShelfInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Returning inventory to shelf", "orderId", input.OrderID, "itemCount", len(input.Items), "reason", input.Reason)

	var lastError error
	successCount := 0
	totalItems := len(input.Items)

	for i, item := range input.Items {
		// Record heartbeat for long-running return operations
		activity.RecordHeartbeat(ctx, fmt.Sprintf("Returning inventory item %d/%d to shelf", i+1, totalItems))

		req := &clients.ReturnToShelfRequest{
			AllocationID: item.AllocationID,
			ReturnedBy:   input.ReturnedBy,
			Reason:       input.Reason,
		}

		err := a.clients.ReturnInventoryToShelf(ctx, item.SKU, req)
		if err != nil {
			logger.Error("Failed to return inventory to shelf",
				"orderId", input.OrderID,
				"sku", item.SKU,
				"allocationId", item.AllocationID,
				"error", err)
			lastError = err
			continue
		}

		successCount++
		logger.Info("Inventory returned to shelf successfully",
			"orderId", input.OrderID,
			"sku", item.SKU,
			"allocationId", item.AllocationID,
		)
	}

	logger.Info("Inventory return to shelf complete",
		"orderId", input.OrderID,
		"successCount", successCount,
		"totalItems", len(input.Items),
	)

	if successCount == 0 && lastError != nil {
		return fmt.Errorf("failed to return any inventory items to shelf: %w", lastError)
	}

	return nil
}

// RecordStockShortageInput holds the input for recording a stock shortage
type RecordStockShortageInput struct {
	SKU         string `json:"sku"`
	LocationID  string `json:"locationId"`
	OrderID     string `json:"orderId"`
	ExpectedQty int    `json:"expectedQty"`
	ActualQty   int    `json:"actualQty"`
	Reason      string `json:"reason"`
	ReportedBy  string `json:"reportedBy"`
}

// RecordStockShortage records a confirmed stock shortage discovered during picking
func (a *InventoryActivities) RecordStockShortage(ctx context.Context, input RecordStockShortageInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Recording stock shortage",
		"sku", input.SKU,
		"orderId", input.OrderID,
		"expectedQty", input.ExpectedQty,
		"actualQty", input.ActualQty,
		"shortageQty", input.ExpectedQty-input.ActualQty,
	)

	req := &clients.RecordShortageRequest{
		LocationID:  input.LocationID,
		OrderID:     input.OrderID,
		ExpectedQty: input.ExpectedQty,
		ActualQty:   input.ActualQty,
		Reason:      input.Reason,
		ReportedBy:  input.ReportedBy,
	}

	err := a.clients.RecordStockShortage(ctx, input.SKU, req)
	if err != nil {
		logger.Error("Failed to record stock shortage",
			"sku", input.SKU,
			"orderId", input.OrderID,
			"error", err,
		)
		return fmt.Errorf("failed to record stock shortage: %w", err)
	}

	logger.Info("Stock shortage recorded successfully",
		"sku", input.SKU,
		"orderId", input.OrderID,
		"shortageQty", input.ExpectedQty-input.ActualQty,
	)
	return nil
}

// GetReservationIDsInput holds the input for getting reservation IDs
type GetReservationIDsInput struct {
	OrderID string                    `json:"orderId"`
	Items   []GetReservationIDsItem   `json:"items"`
}

// GetReservationIDsItem represents an item to get reservation ID for
type GetReservationIDsItem struct {
	SKU string `json:"sku"`
}

// GetReservationIDsOutput holds the output from getting reservation IDs
type GetReservationIDsOutput struct {
	Reservations map[string]string `json:"reservations"` // SKU -> ReservationID
}

// GetReservationIDs retrieves reservation IDs for an order's items
func (a *InventoryActivities) GetReservationIDs(ctx context.Context, input GetReservationIDsInput) (*GetReservationIDsOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Getting reservation IDs", "orderId", input.OrderID, "itemCount", len(input.Items))

	output := &GetReservationIDsOutput{
		Reservations: make(map[string]string),
	}

	for _, item := range input.Items {
		// Get inventory item details which includes reservations
		inventoryItem, err := a.clients.GetInventoryItem(ctx, item.SKU)
		if err != nil {
			logger.Warn("Failed to get inventory item",
				"sku", item.SKU,
				"orderId", input.OrderID,
				"error", err)
			continue
		}

		// Find the reservation for this order
		for _, reservation := range inventoryItem.Reservations {
			if reservation.OrderID == input.OrderID {
				output.Reservations[item.SKU] = reservation.ReservationID
				logger.Info("Found reservation",
					"sku", item.SKU,
					"orderId", input.OrderID,
					"reservationId", reservation.ReservationID)
				break
			}
		}

		if _, found := output.Reservations[item.SKU]; !found {
			logger.Warn("No reservation found for item",
				"sku", item.SKU,
				"orderId", input.OrderID)
		}
	}

	logger.Info("Reservation IDs retrieved",
		"orderId", input.OrderID,
		"foundCount", len(output.Reservations),
		"requestedCount", len(input.Items))

	return output, nil
}
