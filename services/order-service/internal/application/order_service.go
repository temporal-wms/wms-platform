package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/middleware"
	"github.com/wms-platform/shared/pkg/temporal"
	"github.com/wms-platform/shared/pkg/tenant"

	"github.com/wms-platform/services/order-service/internal/domain"
	"github.com/wms-platform/services/order-service/internal/infrastructure/projections"
)

// OrderApplicationService handles order-related use cases
type OrderApplicationService struct {
	orderRepo       domain.OrderRepository
	producer        *kafka.InstrumentedProducer
	eventFactory    *cloudevents.EventFactory
	temporalClient  *temporal.Client
	projector       *projections.OrderProjector // CQRS projector for read model
	logger          *logging.Logger
	businessMetrics *middleware.BusinessMetrics
}

// NewOrderApplicationService creates a new OrderApplicationService
func NewOrderApplicationService(
	orderRepo domain.OrderRepository,
	producer *kafka.InstrumentedProducer,
	eventFactory *cloudevents.EventFactory,
	temporalClient *temporal.Client,
	projector *projections.OrderProjector,
	logger *logging.Logger,
	businessMetrics *middleware.BusinessMetrics,
) *OrderApplicationService {
	return &OrderApplicationService{
		orderRepo:       orderRepo,
		producer:        producer,
		eventFactory:    eventFactory,
		temporalClient:  temporalClient,
		projector:       projector,
		logger:          logger,
		businessMetrics: businessMetrics,
	}
}

// CreateOrder creates a new order and starts the fulfillment workflow
func (s *OrderApplicationService) CreateOrder(ctx context.Context, cmd CreateOrderCommand) (*OrderCreatedResponse, error) {
	// Extract tenant context from Go context (set by TenantAuth middleware)
	tenantCtx := tenant.FromContextOptional(ctx)

	// Build tenant info - prefer explicit command values, fallback to context
	tenantInfo := &domain.TenantInfo{
		TenantID:        firstNonEmpty(cmd.TenantID, tenantCtx.TenantID),
		FacilityID:      firstNonEmpty(cmd.FacilityID, tenantCtx.FacilityID),
		WarehouseID:     firstNonEmpty(cmd.WarehouseID, tenantCtx.WarehouseID),
		SellerID:        firstNonEmpty(cmd.SellerID, tenantCtx.SellerID),
		ChannelID:       firstNonEmpty(cmd.ChannelID, tenantCtx.ChannelID),
		ExternalOrderID: cmd.ExternalOrderID,
	}

	// Generate order ID
	orderID := "ORD-" + uuid.New().String()[:8]

	// Create the order aggregate WITH tenant info
	order, err := domain.NewOrderWithTenant(
		orderID,
		cmd.CustomerID,
		cmd.ToDomainOrderItems(),
		cmd.ShippingAddress.ToDomainAddress(),
		cmd.ToDomainPriority(),
		cmd.PromisedDeliveryAt,
		tenantInfo,
	)
	if err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	// Capture events before save (they'll be cleared by repository)
	events := order.DomainEvents()

	// Save to repository
	if err := s.orderRepo.Save(ctx, order); err != nil {
		s.logger.WithError(err).Error("Failed to save order", "orderId", orderID)
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	// Update CQRS read model projections
	s.updateProjections(ctx, events)

	// Record business metric
	s.businessMetrics.RecordOrderCreated(cmd.Priority)

	// Events are saved to outbox by repository in transaction

	// Start the OrderFulfillment workflow
	workflowID, err := s.startOrderFulfillmentWorkflow(ctx, order, cmd)
	if err != nil {
		// Order is created, but workflow failed - log and continue
		s.logger.WithError(err).Error("Failed to start workflow", "orderId", orderID)
	} else {
		s.logger.Info("Started order fulfillment workflow", "orderId", orderID, "workflowId", workflowID)
	}

	// Log business event: order received
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "order.received",
		EntityType: "order",
		EntityID:   orderID,
		Action:     "created",
		RelatedIDs: map[string]string{
			"customerId": cmd.CustomerID,
			"workflowId": workflowID,
		},
	})

	return &OrderCreatedResponse{
		Order:      *ToOrderDTO(order),
		WorkflowID: workflowID,
	}, nil
}

// GetOrder retrieves an order by ID
func (s *OrderApplicationService) GetOrder(ctx context.Context, query GetOrderQuery) (*OrderDTO, error) {
	order, err := s.orderRepo.FindByID(ctx, query.OrderID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get order", "orderId", query.OrderID)
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if order == nil {
		return nil, errors.ErrNotFound("order")
	}

	return ToOrderDTO(order), nil
}

// ValidateOrder validates an order
func (s *OrderApplicationService) ValidateOrder(ctx context.Context, cmd ValidateOrderCommand) (*OrderDTO, error) {
	order, err := s.orderRepo.FindByID(ctx, cmd.OrderID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get order", "orderId", cmd.OrderID)
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if order == nil {
		return nil, errors.ErrNotFound("order")
	}

	// Validate the order (domain logic)
	if err := order.Validate(); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	// Capture events before save
	events := order.DomainEvents()

	// Save the updated order
	if err := s.orderRepo.Save(ctx, order); err != nil {
		s.logger.WithError(err).Error("Failed to save order", "orderId", cmd.OrderID)
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	// Update CQRS projections
	s.updateProjections(ctx, events)

	// Events are saved to outbox by repository in transaction

	// Log business event: order validated
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "order.validated",
		EntityType: "order",
		EntityID:   cmd.OrderID,
		Action:     "validated",
	})

	return ToOrderDTO(order), nil
}

// CancelOrder cancels an order with a reason
func (s *OrderApplicationService) CancelOrder(ctx context.Context, cmd CancelOrderCommand) (*OrderDTO, error) {
	order, err := s.orderRepo.FindByID(ctx, cmd.OrderID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get order", "orderId", cmd.OrderID)
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if order == nil {
		return nil, errors.ErrNotFound("order")
	}

	// Cancel the order (domain logic)
	if err := order.Cancel(cmd.Reason); err != nil {
		return nil, errors.ErrConflict(err.Error())
	}

	// Capture events before save
	events := order.DomainEvents()

	// Save the updated order
	if err := s.orderRepo.Save(ctx, order); err != nil {
		s.logger.WithError(err).Error("Failed to save order", "orderId", cmd.OrderID)
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	// Update CQRS projections
	s.updateProjections(ctx, events)

	// Events are saved to outbox by repository in transaction

	// Log business event: order cancelled
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "order.cancelled",
		EntityType: "order",
		EntityID:   cmd.OrderID,
		Action:     "cancelled",
		RelatedIDs: map[string]string{
			"reason": cmd.Reason,
		},
	})

	return ToOrderDTO(order), nil
}

// ListOrders lists orders with filters and pagination
func (s *OrderApplicationService) ListOrders(ctx context.Context, query ListOrdersQuery) (*PagedOrdersResult, error) {
	filter := query.ToDomainFilter()
	pagination := query.ToDomainPagination()

	// Get total count
	total, err := s.orderRepo.Count(ctx, filter)
	if err != nil {
		s.logger.WithError(err).Error("Failed to count orders")
		return nil, fmt.Errorf("failed to count orders: %w", err)
	}

	// Get orders
	var orders []*domain.Order
	if filter.Status != nil {
		orders, err = s.orderRepo.FindByStatus(ctx, *filter.Status, pagination)
	} else {
		// Default to finding validated orders ready for processing
		orders, err = s.orderRepo.FindValidatedOrders(ctx, "", int(pagination.PageSize))
	}

	if err != nil {
		s.logger.WithError(err).Error("Failed to list orders")
		return nil, fmt.Errorf("failed to list orders: %w", err)
	}

	// Calculate pagination metadata
	totalPages := (total + pagination.PageSize - 1) / pagination.PageSize

	return &PagedOrdersResult{
		Data:       ToOrderListDTOs(orders),
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}, nil
}

// AssignToWave assigns an order to a wave (called by waving-service)
func (s *OrderApplicationService) AssignToWave(ctx context.Context, cmd AssignToWaveCommand) (*OrderDTO, error) {
	order, err := s.orderRepo.FindByID(ctx, cmd.OrderID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get order", "orderId", cmd.OrderID)
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if order == nil {
		return nil, errors.ErrNotFound("order")
	}

	// Assign to wave (domain logic)
	if err := order.AssignToWave(cmd.WaveID); err != nil {
		return nil, errors.ErrConflict(err.Error())
	}

	// Capture events before save
	events := order.DomainEvents()

	// Save the updated order
	if err := s.orderRepo.Save(ctx, order); err != nil {
		s.logger.WithError(err).Error("Failed to save order", "orderId", cmd.OrderID)
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	// Update CQRS projections
	s.updateProjections(ctx, events)

	// Events are saved to outbox by repository in transaction

	// Log business event: order assigned to wave
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "order.wave_assigned",
		EntityType: "order",
		EntityID:   cmd.OrderID,
		Action:     "wave_assigned",
		RelatedIDs: map[string]string{
			"waveId": cmd.WaveID,
		},
	})

	return ToOrderDTO(order), nil
}

// MarkShipped marks an order as shipped (called by shipping-service)
func (s *OrderApplicationService) MarkShipped(ctx context.Context, cmd MarkShippedCommand) (*OrderDTO, error) {
	order, err := s.orderRepo.FindByID(ctx, cmd.OrderID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get order", "orderId", cmd.OrderID)
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if order == nil {
		return nil, errors.ErrNotFound("order")
	}

	// Mark as shipped (domain logic)
	if err := order.MarkShipped(cmd.TrackingNumber); err != nil {
		return nil, errors.ErrConflict(err.Error())
	}

	// Capture events before save
	events := order.DomainEvents()

	// Save the updated order
	if err := s.orderRepo.Save(ctx, order); err != nil {
		s.logger.WithError(err).Error("Failed to save order", "orderId", cmd.OrderID)
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	// Update CQRS projections
	s.updateProjections(ctx, events)

	// Events are saved to outbox by repository in transaction

	// Log business event: order shipped
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "order.shipped",
		EntityType: "order",
		EntityID:   cmd.OrderID,
		Action:     "shipped",
		RelatedIDs: map[string]string{
			"trackingNumber": cmd.TrackingNumber,
		},
	})

	return ToOrderDTO(order), nil
}

// StartPicking marks an order as picking in progress (called by orchestrator)
func (s *OrderApplicationService) StartPicking(ctx context.Context, cmd StartPickingCommand) (*OrderDTO, error) {
	order, err := s.orderRepo.FindByID(ctx, cmd.OrderID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get order", "orderId", cmd.OrderID)
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if order == nil {
		return nil, errors.ErrNotFound("order")
	}

	// Start picking (domain logic)
	if err := order.StartPicking(); err != nil {
		return nil, errors.ErrConflict(err.Error())
	}

	// Capture events before save
	events := order.DomainEvents()

	// Save the updated order
	if err := s.orderRepo.Save(ctx, order); err != nil {
		s.logger.WithError(err).Error("Failed to save order", "orderId", cmd.OrderID)
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	// Update CQRS projections
	s.updateProjections(ctx, events)

	// Log business event: picking started
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "order.picking_started",
		EntityType: "order",
		EntityID:   cmd.OrderID,
		Action:     "picking_started",
	})

	return ToOrderDTO(order), nil
}

// MarkConsolidated marks an order as consolidated (called by orchestrator)
func (s *OrderApplicationService) MarkConsolidated(ctx context.Context, cmd MarkConsolidatedCommand) (*OrderDTO, error) {
	order, err := s.orderRepo.FindByID(ctx, cmd.OrderID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get order", "orderId", cmd.OrderID)
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if order == nil {
		return nil, errors.ErrNotFound("order")
	}

	// Mark as consolidated (domain logic)
	if err := order.MarkConsolidated(); err != nil {
		return nil, errors.ErrConflict(err.Error())
	}

	// Capture events before save
	events := order.DomainEvents()

	// Save the updated order
	if err := s.orderRepo.Save(ctx, order); err != nil {
		s.logger.WithError(err).Error("Failed to save order", "orderId", cmd.OrderID)
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	// Update CQRS projections
	s.updateProjections(ctx, events)

	// Log business event: order consolidated
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "order.consolidated",
		EntityType: "order",
		EntityID:   cmd.OrderID,
		Action:     "consolidated",
	})

	return ToOrderDTO(order), nil
}

// MarkPacked marks an order as packed (called by orchestrator)
func (s *OrderApplicationService) MarkPacked(ctx context.Context, cmd MarkPackedCommand) (*OrderDTO, error) {
	order, err := s.orderRepo.FindByID(ctx, cmd.OrderID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get order", "orderId", cmd.OrderID)
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if order == nil {
		return nil, errors.ErrNotFound("order")
	}

	// Mark as packed (domain logic)
	if err := order.MarkPacked(); err != nil {
		return nil, errors.ErrConflict(err.Error())
	}

	// Capture events before save
	events := order.DomainEvents()

	// Save the updated order
	if err := s.orderRepo.Save(ctx, order); err != nil {
		s.logger.WithError(err).Error("Failed to save order", "orderId", cmd.OrderID)
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	// Update CQRS projections
	s.updateProjections(ctx, events)

	// Log business event: order packed
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "order.packed",
		EntityType: "order",
		EntityID:   cmd.OrderID,
		Action:     "packed",
	})

	return ToOrderDTO(order), nil
}

// startOrderFulfillmentWorkflow starts the Temporal workflow for order fulfillment
func (s *OrderApplicationService) startOrderFulfillmentWorkflow(ctx context.Context, order *domain.Order, cmd CreateOrderCommand) (string, error) {
	workflowInput := OrderFulfillmentInput{
		OrderID:            order.OrderID,
		CustomerID:         cmd.CustomerID,
		Priority:           cmd.Priority,
		PromisedDeliveryAt: cmd.PromisedDeliveryAt,
		IsMultiItem:        order.IsMultiItem(),
		Items:              make([]WorkflowItem, 0, len(cmd.Items)),
		// Propagate tenant context to workflow
		TenantID:    order.TenantID,
		FacilityID:  order.FacilityID,
		WarehouseID: order.WarehouseID,
	}

	for _, item := range cmd.Items {
		workflowInput.Items = append(workflowInput.Items, WorkflowItem{
			SKU:      item.SKU,
			Quantity: item.Quantity,
			Weight:   item.Weight,
		})
	}

	workflowID := "order-fulfillment-" + order.OrderID
	_, err := s.temporalClient.StartWorkflow(
		ctx,
		workflowID,
		temporal.TaskQueues.Orchestrator,
		temporal.WorkflowNames.OrderFulfillment,
		workflowInput,
	)
	if err != nil {
		return "", err
	}

	return workflowID, nil
}

// OrderFulfillmentInput matches the orchestrator workflow input
type OrderFulfillmentInput struct {
	OrderID            string         `json:"orderId"`
	CustomerID         string         `json:"customerId"`
	Items              []WorkflowItem `json:"items"`
	Priority           string         `json:"priority"`
	PromisedDeliveryAt time.Time      `json:"promisedDeliveryAt"`
	IsMultiItem        bool           `json:"isMultiItem"`
	// Multi-tenant context
	TenantID    string `json:"tenantId,omitempty"`
	FacilityID  string `json:"facilityId,omitempty"`
	WarehouseID string `json:"warehouseId,omitempty"`
}

// WorkflowItem represents an item in the workflow input
type WorkflowItem struct {
	SKU      string  `json:"sku"`
	Quantity int     `json:"quantity"`
	Weight   float64 `json:"weight"`
}

// updateProjections updates the CQRS read model based on domain events
// Call this after successfully saving an order to keep projections in sync
func (s *OrderApplicationService) updateProjections(ctx context.Context, events []domain.DomainEvent) {
	if s.projector == nil {
		return // Projector not configured (e.g., in tests)
	}

	for _, event := range events {
		var err error
		switch e := event.(type) {
		case *domain.OrderReceivedEvent:
			err = s.projector.OnOrderReceived(ctx, e)
		case *domain.OrderValidatedEvent:
			err = s.projector.OnOrderValidated(ctx, e)
		case *domain.OrderAssignedToWaveEvent:
			err = s.projector.OnOrderAssignedToWave(ctx, e)
		case *domain.OrderShippedEvent:
			err = s.projector.OnOrderShipped(ctx, e)
		case *domain.OrderCancelledEvent:
			err = s.projector.OnOrderCancelled(ctx, e)
		}

		if err != nil {
			// Log error but don't fail the operation - projection updates are eventually consistent
			s.logger.WithError(err).Error("Failed to update projection", "eventType", fmt.Sprintf("%T", event))
		}
	}
}

// firstNonEmpty returns the first non-empty string from the provided values
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
