package application

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/inventory-service/internal/domain"
)

// InventoryCoordinator coordinates operations across inventory aggregate and related collections
// This facade pattern simplifies complex operations that span multiple repositories
type InventoryCoordinator struct {
	inventoryRepo   inventoryRepository
	transactionRepo transactionRepository
	reservationRepo reservationRepository
	allocationRepo  allocationRepository
}

func NewInventoryCoordinator(
	inventoryRepo inventoryRepository,
	transactionRepo transactionRepository,
	reservationRepo reservationRepository,
	allocationRepo allocationRepository,
) *InventoryCoordinator {
	return &InventoryCoordinator{
		inventoryRepo:   inventoryRepo,
		transactionRepo: transactionRepo,
		reservationRepo: reservationRepo,
		allocationRepo:  allocationRepo,
	}
}

type inventoryRepository interface {
	FindBySKU(ctx context.Context, sku string) (*domain.InventoryItem, error)
	Save(ctx context.Context, item *domain.InventoryItem) error
}

type transactionRepository interface {
	Save(ctx context.Context, transaction *domain.InventoryTransactionAggregate) error
	FindBySKU(ctx context.Context, sku string, limit int) ([]*domain.InventoryTransactionAggregate, error)
}

type reservationRepository interface {
	Save(ctx context.Context, reservation *domain.InventoryReservationAggregate) error
	FindByID(ctx context.Context, reservationID string) (*domain.InventoryReservationAggregate, error)
	FindBySKU(ctx context.Context, sku string, status domain.ReservationStatus) ([]*domain.InventoryReservationAggregate, error)
}

type allocationRepository interface {
	Save(ctx context.Context, allocation *domain.InventoryAllocationAggregate) error
	FindByID(ctx context.Context, allocationID string) (*domain.InventoryAllocationAggregate, error)
	FindBySKU(ctx context.Context, sku string, status domain.AllocationStatus) ([]*domain.InventoryAllocationAggregate, error)
}

// ReceiveInventory receives stock and records the transaction
func (c *InventoryCoordinator) ReceiveInventory(
	ctx context.Context,
	sku string,
	locationID string,
	zone string,
	quantity int,
	referenceID string,
	receivedBy string,
) error {
	// Get inventory item
	item, err := c.inventoryRepo.FindBySKU(ctx, sku)
	if err != nil {
		return fmt.Errorf("failed to find inventory: %w", err)
	}
	if item == nil {
		return fmt.Errorf("inventory item not found: %s", sku)
	}

	// Update inventory aggregate
	if err := item.ReceiveStock(locationID, zone, quantity, referenceID, receivedBy); err != nil {
		return fmt.Errorf("failed to receive stock: %w", err)
	}

	// Save inventory (includes domain events)
	if err := c.inventoryRepo.Save(ctx, item); err != nil {
		return fmt.Errorf("failed to save inventory: %w", err)
	}

	// Record transaction in separate collection
	transaction := domain.NewInventoryTransaction(
		generateTransactionID(),
		sku,
		"receive",
		quantity,
		locationID,
		referenceID,
		"",
		receivedBy,
		&domain.TransactionTenantInfo{
			TenantID:    item.TenantID,
			FacilityID:  item.FacilityID,
			WarehouseID: item.WarehouseID,
			SellerID:    item.SellerID,
		},
	)

	if err := c.transactionRepo.Save(ctx, transaction); err != nil {
		// Log error but don't fail - transaction is in event stream
		// Consider compensating transaction or retry logic here
		return fmt.Errorf("failed to save transaction: %w", err)
	}

	return nil
}

// ReserveInventory creates a reservation and updates inventory counters
func (c *InventoryCoordinator) ReserveInventory(
	ctx context.Context,
	sku string,
	orderID string,
	locationID string,
	quantity int,
	unitIDs []string,
	createdBy string,
) (*domain.InventoryReservationAggregate, error) {
	// Get inventory item (use refactored version in production)
	item, err := c.inventoryRepo.FindBySKU(ctx, sku)
	if err != nil {
		return nil, fmt.Errorf("failed to find inventory: %w", err)
	}
	if item == nil {
		return nil, fmt.Errorf("inventory item not found: %s", sku)
	}

	// Check availability
	location := item.GetLocationStock(locationID)
	if location == nil {
		return nil, domain.ErrLocationNotFound
	}
	if location.Available < quantity {
		return nil, domain.ErrInsufficientStock
	}

	// Update inventory counters
	if err := item.Reserve(orderID, locationID, quantity); err != nil {
		return nil, fmt.Errorf("failed to reserve inventory: %w", err)
	}

	// Create reservation record
	reservation := domain.NewInventoryReservation(
		generateReservationID(),
		sku,
		orderID,
		locationID,
		quantity,
		unitIDs,
		createdBy,
		&domain.ReservationTenantInfo{
			TenantID:    item.TenantID,
			FacilityID:  item.FacilityID,
			WarehouseID: item.WarehouseID,
			SellerID:    item.SellerID,
		},
	)

	// Save both in transaction-like manner
	// In production, use MongoDB transactions for atomicity
	if err := c.inventoryRepo.Save(ctx, item); err != nil {
		return nil, fmt.Errorf("failed to save inventory: %w", err)
	}

	if err := c.reservationRepo.Save(ctx, reservation); err != nil {
		// Compensate by releasing reservation in inventory
		_ = item.ReleaseReservation(orderID)
		_ = c.inventoryRepo.Save(ctx, item)
		return nil, fmt.Errorf("failed to save reservation: %w", err)
	}

	return reservation, nil
}

// StageInventory creates a hard allocation and updates inventory/reservation
func (c *InventoryCoordinator) StageInventory(
	ctx context.Context,
	reservationID string,
	stagingLocationID string,
	stagedBy string,
) (*domain.InventoryAllocationAggregate, error) {
	// Get reservation
	reservation, err := c.reservationRepo.FindByID(ctx, reservationID)
	if err != nil {
		return nil, fmt.Errorf("failed to find reservation: %w", err)
	}
	if reservation == nil {
		return nil, domain.ErrReservationNotFound
	}

	// Validate reservation status
	if reservation.Status != domain.ReservationStatusActive {
		return nil, domain.ErrReservationNotActive
	}

	// Get inventory item
	item, err := c.inventoryRepo.FindBySKU(ctx, reservation.SKU)
	if err != nil {
		return nil, fmt.Errorf("failed to find inventory: %w", err)
	}

	// Update inventory (move from reserved to hard allocated)
	if err := item.Stage(reservationID, stagingLocationID, stagedBy); err != nil {
		return nil, fmt.Errorf("failed to stage inventory: %w", err)
	}

	// Update reservation status
	if err := reservation.MarkStaged(stagedBy); err != nil {
		return nil, fmt.Errorf("failed to mark reservation as staged: %w", err)
	}

	// Create allocation record
	allocation := domain.NewInventoryAllocation(
		generateAllocationID(),
		reservation.SKU,
		reservationID,
		reservation.OrderID,
		reservation.Quantity,
		reservation.LocationID,
		stagingLocationID,
		reservation.UnitIDs,
		stagedBy,
		&domain.AllocationTenantInfo{
			TenantID:    item.TenantID,
			FacilityID:  item.FacilityID,
			WarehouseID: item.WarehouseID,
			SellerID:    item.SellerID,
		},
	)

	// Save all changes
	if err := c.inventoryRepo.Save(ctx, item); err != nil {
		return nil, fmt.Errorf("failed to save inventory: %w", err)
	}

	if err := c.reservationRepo.Save(ctx, reservation); err != nil {
		return nil, fmt.Errorf("failed to save reservation: %w", err)
	}

	if err := c.allocationRepo.Save(ctx, allocation); err != nil {
		return nil, fmt.Errorf("failed to save allocation: %w", err)
	}

	return allocation, nil
}

// PackAllocation marks an allocation as packed
func (c *InventoryCoordinator) PackAllocation(
	ctx context.Context,
	allocationID string,
	packedBy string,
) error {
	allocation, err := c.allocationRepo.FindByID(ctx, allocationID)
	if err != nil {
		return fmt.Errorf("failed to find allocation: %w", err)
	}
	if allocation == nil {
		return domain.ErrAllocationNotFound
	}

	if err := allocation.MarkPacked(packedBy); err != nil {
		return fmt.Errorf("failed to mark as packed: %w", err)
	}

	return c.allocationRepo.Save(ctx, allocation)
}

// ShipAllocation ships allocated inventory (removes from warehouse)
func (c *InventoryCoordinator) ShipAllocation(
	ctx context.Context,
	allocationID string,
	shippedBy string,
) error {
	// Get allocation
	allocation, err := c.allocationRepo.FindByID(ctx, allocationID)
	if err != nil {
		return fmt.Errorf("failed to find allocation: %w", err)
	}
	if allocation == nil {
		return domain.ErrAllocationNotFound
	}

	// Get reservation
	reservation, err := c.reservationRepo.FindByID(ctx, allocation.ReservationID)
	if err != nil {
		return fmt.Errorf("failed to find reservation: %w", err)
	}

	// Get inventory
	item, err := c.inventoryRepo.FindBySKU(ctx, allocation.SKU)
	if err != nil {
		return fmt.Errorf("failed to find inventory: %w", err)
	}

	// Ship from inventory (removes quantity)
	if err := item.Ship(allocationID); err != nil {
		return fmt.Errorf("failed to ship inventory: %w", err)
	}

	// Update allocation status
	if err := allocation.MarkShipped(shippedBy); err != nil {
		return fmt.Errorf("failed to mark as shipped: %w", err)
	}

	// Update reservation status
	if reservation != nil {
		_ = reservation.MarkFulfilled(shippedBy)
		_ = c.reservationRepo.Save(ctx, reservation)
	}

	// Record transaction
	transaction := domain.NewInventoryTransaction(
		generateTransactionID(),
		allocation.SKU,
		"ship",
		-allocation.Quantity,
		allocation.SourceLocationID,
		allocation.OrderID,
		"",
		shippedBy,
		&domain.TransactionTenantInfo{
			TenantID:    allocation.TenantID,
			FacilityID:  allocation.FacilityID,
			WarehouseID: allocation.WarehouseID,
			SellerID:    allocation.SellerID,
		},
	)

	// Save all
	if err := c.inventoryRepo.Save(ctx, item); err != nil {
		return fmt.Errorf("failed to save inventory: %w", err)
	}

	if err := c.allocationRepo.Save(ctx, allocation); err != nil {
		return fmt.Errorf("failed to save allocation: %w", err)
	}

	if err := c.transactionRepo.Save(ctx, transaction); err != nil {
		// Log but don't fail
		return fmt.Errorf("failed to save transaction: %w", err)
	}

	return nil
}

// CancelReservation cancels a reservation and releases inventory
func (c *InventoryCoordinator) CancelReservation(
	ctx context.Context,
	reservationID string,
	cancelledBy string,
	reason string,
) error {
	// Get reservation
	reservation, err := c.reservationRepo.FindByID(ctx, reservationID)
	if err != nil {
		return fmt.Errorf("failed to find reservation: %w", err)
	}
	if reservation == nil {
		return domain.ErrReservationNotFound
	}

	// Get inventory
	item, err := c.inventoryRepo.FindBySKU(ctx, reservation.SKU)
	if err != nil {
		return fmt.Errorf("failed to find inventory: %w", err)
	}

	// Release reservation in inventory
	if err := item.ReleaseReservation(reservation.OrderID); err != nil {
		return fmt.Errorf("failed to release reservation: %w", err)
	}

	// Cancel reservation
	if err := reservation.Cancel(cancelledBy, reason); err != nil {
		return fmt.Errorf("failed to cancel reservation: %w", err)
	}

	// Save both
	if err := c.inventoryRepo.Save(ctx, item); err != nil {
		return fmt.Errorf("failed to save inventory: %w", err)
	}

	if err := c.reservationRepo.Save(ctx, reservation); err != nil {
		return fmt.Errorf("failed to save reservation: %w", err)
	}

	return nil
}

// GetInventoryWithDetails returns inventory with recent transactions and active reservations
func (c *InventoryCoordinator) GetInventoryWithDetails(
	ctx context.Context,
	sku string,
	includeTransactions bool,
	includeReservations bool,
	includeAllocations bool,
) (*InventoryDetails, error) {
	// Get main inventory
	item, err := c.inventoryRepo.FindBySKU(ctx, sku)
	if err != nil {
		return nil, fmt.Errorf("failed to find inventory: %w", err)
	}
	if item == nil {
		return nil, fmt.Errorf("inventory not found: %s", sku)
	}

	details := &InventoryDetails{
		Inventory: item,
	}

	// Optionally fetch related data
	if includeTransactions {
		txns, err := c.transactionRepo.FindBySKU(ctx, sku, 100) // Last 100 transactions
		if err != nil {
			return nil, fmt.Errorf("failed to fetch transactions: %w", err)
		}
		details.RecentTransactions = txns
	}

	if includeReservations {
		res, err := c.reservationRepo.FindBySKU(ctx, sku, domain.ReservationStatusActive)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch reservations: %w", err)
		}
		details.ActiveReservations = res
	}

	if includeAllocations {
		allocs, err := c.allocationRepo.FindBySKU(ctx, sku, "")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch allocations: %w", err)
		}
		details.ActiveAllocations = allocs
	}

	return details, nil
}

// InventoryDetails aggregates inventory with related data
type InventoryDetails struct {
	Inventory           *domain.InventoryItem
	RecentTransactions  []*domain.InventoryTransactionAggregate
	ActiveReservations  []*domain.InventoryReservationAggregate
	ActiveAllocations   []*domain.InventoryAllocationAggregate
}

// Helper ID generators
func generateTransactionID() string {
	return fmt.Sprintf("TXN-%d", time.Now().UnixNano())
}

func generateReservationID() string {
	return fmt.Sprintf("RES-%d", time.Now().UnixNano())
}

func generateAllocationID() string {
	return fmt.Sprintf("ALLOC-%d", time.Now().UnixNano())
}
