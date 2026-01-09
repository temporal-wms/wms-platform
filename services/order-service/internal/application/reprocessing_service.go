package application

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/middleware"

	"github.com/wms-platform/services/order-service/internal/domain"
	mongoRepo "github.com/wms-platform/services/order-service/internal/infrastructure/mongodb"
)

// ReprocessingService handles reprocessing-related use cases
type ReprocessingService struct {
	orderRepo      domain.OrderRepository
	retryRepo      *mongoRepo.RetryMetadataRepository
	deadLetterRepo *mongoRepo.DeadLetterRepository
	logger         *logging.Logger
	failureMetrics *middleware.FailureMetrics
}

// NewReprocessingService creates a new ReprocessingService
func NewReprocessingService(
	orderRepo domain.OrderRepository,
	retryRepo *mongoRepo.RetryMetadataRepository,
	deadLetterRepo *mongoRepo.DeadLetterRepository,
	logger *logging.Logger,
	failureMetrics *middleware.FailureMetrics,
) *ReprocessingService {
	return &ReprocessingService{
		orderRepo:      orderRepo,
		retryRepo:      retryRepo,
		deadLetterRepo: deadLetterRepo,
		logger:         logger,
		failureMetrics: failureMetrics,
	}
}

// --- DTOs ---

// RetryMetadataDTO represents retry metadata in API responses
type RetryMetadataDTO struct {
	OrderID        string    `json:"orderId"`
	RetryCount     int       `json:"retryCount"`
	MaxRetries     int       `json:"maxRetries"`
	FailureStatus  string    `json:"failureStatus"`
	FailureReason  string    `json:"failureReason,omitempty"`
	LastWorkflowID string    `json:"lastWorkflowId,omitempty"`
	LastRunID      string    `json:"lastRunId,omitempty"`
	LastFailureAt  time.Time `json:"lastFailureAt"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// FailedWorkflowDTO represents a failed workflow eligible for reprocessing
type FailedWorkflowDTO struct {
	OrderID       string    `json:"orderId"`
	WorkflowID    string    `json:"workflowId"`
	RunID         string    `json:"runId"`
	FailureStatus string    `json:"failureStatus"`
	FailureReason string    `json:"failureReason"`
	FailedAt      time.Time `json:"failedAt"`
	RetryCount    int       `json:"retryCount"`
	CustomerID    string    `json:"customerId"`
	Priority      string    `json:"priority"`
}

// EligibleOrdersResponse is the response for eligible orders query
type EligibleOrdersResponse struct {
	Data  []FailedWorkflowDTO `json:"data"`
	Total int64               `json:"total"`
}

// DeadLetterEntryDTO represents a DLQ entry in API responses
type DeadLetterEntryDTO struct {
	OrderID            string           `json:"orderId"`
	CustomerID         string           `json:"customerId"`
	OriginalWorkflowID string           `json:"originalWorkflowId"`
	FinalFailureStatus string           `json:"finalFailureStatus"`
	FinalFailureReason string           `json:"finalFailureReason"`
	TotalRetryAttempts int              `json:"totalRetryAttempts"`
	RetryHistory       []RetryAttemptDTO `json:"retryHistory,omitempty"`
	MovedToQueueAt     time.Time        `json:"movedToQueueAt"`
	Resolution         string           `json:"resolution,omitempty"`
	ResolutionNotes    string           `json:"resolutionNotes,omitempty"`
	ResolvedBy         string           `json:"resolvedBy,omitempty"`
	ResolvedAt         *time.Time       `json:"resolvedAt,omitempty"`
}

// RetryAttemptDTO represents a single retry attempt
type RetryAttemptDTO struct {
	AttemptNumber int       `json:"attemptNumber"`
	WorkflowID    string    `json:"workflowId"`
	FailedAt      time.Time `json:"failedAt"`
	FailureReason string    `json:"failureReason"`
}

// DeadLetterListResponse is the response for DLQ list query
type DeadLetterListResponse struct {
	Data       []DeadLetterEntryDTO `json:"data"`
	Total      int64                `json:"total"`
	Limit      int                  `json:"limit"`
	Offset     int                  `json:"offset"`
}

// DLQStatsDTO represents DLQ statistics
type DLQStatsDTO struct {
	TotalEntries     int64            `json:"totalEntries"`
	UnresolvedCount  int64            `json:"unresolvedCount"`
	ResolvedCount    int64            `json:"resolvedCount"`
	ByFailureStatus  map[string]int64 `json:"byFailureStatus"`
	ByResolution     map[string]int64 `json:"byResolution"`
	AverageRetries   float64          `json:"averageRetries"`
	OldestUnresolved *time.Time       `json:"oldestUnresolved,omitempty"`
}

// --- Commands ---

// IncrementRetryCountCommand is the command to increment retry count
type IncrementRetryCountCommand struct {
	OrderID       string `json:"orderId"`
	FailureStatus string `json:"failureStatus"`
	FailureReason string `json:"failureReason"`
	WorkflowID    string `json:"workflowId"`
	RunID         string `json:"runId"`
}

// MoveToDLQCommand is the command to move an order to DLQ
type MoveToDLQCommand struct {
	OrderID       string `json:"orderId"`
	FailureStatus string `json:"failureStatus"`
	FailureReason string `json:"failureReason"`
	RetryCount    int    `json:"retryCount"`
	WorkflowID    string `json:"workflowId"`
	RunID         string `json:"runId"`
}

// ResolveDLQCommand is the command to resolve a DLQ entry
type ResolveDLQCommand struct {
	OrderID    string `json:"orderId"`
	Resolution string `json:"resolution" binding:"required,oneof=manual_retry cancelled escalated"`
	Notes      string `json:"notes"`
	ResolvedBy string `json:"resolvedBy"`
}

// --- Queries ---

// GetEligibleOrdersQuery query parameters for eligible orders
type GetEligibleOrdersQuery struct {
	FailureStatuses []string
	MaxRetries      int
	Limit           int
}

// ListDLQQuery query parameters for DLQ list
type ListDLQQuery struct {
	Resolved       *bool
	FailureStatus  *string
	CustomerID     *string
	OlderThanHours *float64
	Limit          int
	Offset         int
}

// --- Service Methods ---

// GetRetryMetadata retrieves retry metadata for an order
func (s *ReprocessingService) GetRetryMetadata(ctx context.Context, orderID string) (*RetryMetadataDTO, error) {
	metadata, err := s.retryRepo.FindByOrderID(ctx, orderID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get retry metadata", "orderId", orderID)
		return nil, fmt.Errorf("failed to get retry metadata: %w", err)
	}

	if metadata == nil {
		return nil, errors.ErrNotFound("retry metadata")
	}

	return toRetryMetadataDTO(metadata), nil
}

// GetEligibleOrders retrieves orders eligible for reprocessing
func (s *ReprocessingService) GetEligibleOrders(ctx context.Context, query GetEligibleOrdersQuery) (*EligibleOrdersResponse, error) {
	// Default values
	if query.Limit <= 0 {
		query.Limit = 100
	}
	if query.MaxRetries <= 0 {
		query.MaxRetries = 5
	}
	if len(query.FailureStatuses) == 0 {
		query.FailureStatuses = []string{"wave_timeout", "pick_timeout"}
	}

	// Get eligible orders from retry metadata
	metadataList, err := s.retryRepo.FindOrdersEligibleForRetry(ctx, query.FailureStatuses, query.MaxRetries, query.Limit)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get eligible orders")
		return nil, fmt.Errorf("failed to get eligible orders: %w", err)
	}

	// Get count
	total, err := s.retryRepo.Count(ctx, query.FailureStatuses)
	if err != nil {
		s.logger.WithError(err).Error("Failed to count eligible orders")
		return nil, fmt.Errorf("failed to count eligible orders: %w", err)
	}

	// Convert to DTOs and enrich with order info
	result := make([]FailedWorkflowDTO, 0, len(metadataList))
	for _, m := range metadataList {
		order, err := s.orderRepo.FindByID(ctx, m.OrderID)
		if err != nil || order == nil {
			s.logger.Warn("Order not found for retry metadata", "orderId", m.OrderID)
			continue
		}

		result = append(result, FailedWorkflowDTO{
			OrderID:       m.OrderID,
			WorkflowID:    m.LastWorkflowID,
			RunID:         m.LastRunID,
			FailureStatus: m.FailureStatus,
			FailureReason: m.FailureReason,
			FailedAt:      m.LastFailureAt,
			RetryCount:    m.RetryCount,
			CustomerID:    order.CustomerID,
			Priority:      string(order.Priority),
		})
	}

	return &EligibleOrdersResponse{
		Data:  result,
		Total: total,
	}, nil
}

// IncrementRetryCount increments the retry count for an order
func (s *ReprocessingService) IncrementRetryCount(ctx context.Context, cmd IncrementRetryCountCommand) error {
	// Check if metadata exists, if not create it
	existing, err := s.retryRepo.FindByOrderID(ctx, cmd.OrderID)
	if err != nil {
		return fmt.Errorf("failed to check existing metadata: %w", err)
	}

	var attemptNumber int
	if existing == nil {
		// Create new metadata
		metadata := &domain.RetryMetadata{
			OrderID:        cmd.OrderID,
			RetryCount:     1,
			MaxRetries:     5,
			FailureStatus:  cmd.FailureStatus,
			FailureReason:  cmd.FailureReason,
			LastWorkflowID: cmd.WorkflowID,
			LastRunID:      cmd.RunID,
			LastFailureAt:  time.Now().UTC(),
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}
		if err := s.retryRepo.Save(ctx, metadata); err != nil {
			return fmt.Errorf("failed to create retry metadata: %w", err)
		}
		attemptNumber = 1
	} else {
		// Increment existing
		if err := s.retryRepo.IncrementRetryCount(ctx, cmd.OrderID, cmd.FailureStatus, cmd.FailureReason, cmd.WorkflowID, cmd.RunID); err != nil {
			return fmt.Errorf("failed to increment retry count: %w", err)
		}
		attemptNumber = existing.RetryCount + 1
	}

	// Record retry attempt metric
	if s.failureMetrics != nil {
		s.failureMetrics.RecordRetryAttempt(cmd.FailureStatus, attemptNumber)
	}

	// Log business event: retry scheduled
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "order.retry_scheduled",
		EntityType: "order",
		EntityID:   cmd.OrderID,
		Action:     "retry_scheduled",
		RelatedIDs: map[string]string{
			"failureStatus": cmd.FailureStatus,
			"retryCount":    fmt.Sprintf("%d", attemptNumber),
			"workflowId":    cmd.WorkflowID,
		},
	})

	return nil
}

// ResetOrderForRetry resets an order for reprocessing
func (s *ReprocessingService) ResetOrderForRetry(ctx context.Context, orderID string) (*OrderDTO, error) {
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get order", "orderId", orderID)
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if order == nil {
		return nil, errors.ErrNotFound("order")
	}

	// Reset the order using domain method
	if err := order.ResetForRetry(); err != nil {
		return nil, errors.ErrConflict(err.Error())
	}

	// Save the updated order
	if err := s.orderRepo.Save(ctx, order); err != nil {
		s.logger.WithError(err).Error("Failed to save order", "orderId", orderID)
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	s.logger.Info("Reset order for retry", "orderId", orderID)
	return ToOrderDTO(order), nil
}

// MoveToDeadLetterQueue moves an order to the dead letter queue
func (s *ReprocessingService) MoveToDeadLetterQueue(ctx context.Context, cmd MoveToDLQCommand) error {
	// Get order details
	order, err := s.orderRepo.FindByID(ctx, cmd.OrderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}
	if order == nil {
		return errors.ErrNotFound("order")
	}

	// Get retry metadata for history
	retryMetadata, _ := s.retryRepo.FindByOrderID(ctx, cmd.OrderID)

	// Build retry history
	var retryHistory []domain.RetryAttempt
	if retryMetadata != nil {
		// Build a summary from the metadata
		retryHistory = append(retryHistory, domain.RetryAttempt{
			AttemptNumber: retryMetadata.RetryCount,
			WorkflowID:    retryMetadata.LastWorkflowID,
			FailedAt:      retryMetadata.LastFailureAt,
			FailureReason: retryMetadata.FailureReason,
		})
	}

	// Create DLQ entry
	entry := &domain.DeadLetterEntry{
		OrderID:            cmd.OrderID,
		CustomerID:         order.CustomerID,
		OriginalWorkflowID: cmd.WorkflowID,
		FinalFailureStatus: cmd.FailureStatus,
		FinalFailureReason: cmd.FailureReason,
		TotalRetryAttempts: cmd.RetryCount,
		RetryHistory:       retryHistory,
		OrderSnapshot: domain.OrderSnapshot{
			ItemCount:          order.TotalItems(),
			TotalWeight:        order.TotalWeight(),
			Priority:           string(order.Priority),
			ShippingCity:       order.ShippingAddress.City,
			ShippingState:      order.ShippingAddress.State,
			PromisedDeliveryAt: order.PromisedDeliveryAt,
		},
		MovedToQueueAt: time.Now().UTC(),
	}

	// Save DLQ entry
	if err := s.deadLetterRepo.Create(ctx, entry); err != nil {
		return fmt.Errorf("failed to create DLQ entry: %w", err)
	}

	// Update order status
	if err := order.MoveToDeadLetter(cmd.FailureReason); err != nil {
		s.logger.WithError(err).Warn("Failed to update order status to dead_letter", "orderId", cmd.OrderID)
	} else {
		if err := s.orderRepo.Save(ctx, order); err != nil {
			s.logger.WithError(err).Warn("Failed to save order after DLQ move", "orderId", cmd.OrderID)
		}
	}

	// Clean up retry metadata
	if err := s.retryRepo.Delete(ctx, cmd.OrderID); err != nil {
		s.logger.WithError(err).Warn("Failed to delete retry metadata after DLQ move", "orderId", cmd.OrderID)
	}

	// Record DLQ entry metric
	if s.failureMetrics != nil {
		s.failureMetrics.RecordMovedToDLQ(cmd.FailureStatus)
	}

	// Log business event: moved to DLQ
	s.logger.LogBusinessEvent(ctx, logging.BusinessEvent{
		EventType:  "order.moved_to_dlq",
		EntityType: "order",
		EntityID:   cmd.OrderID,
		Action:     "moved_to_dlq",
		RelatedIDs: map[string]string{
			"failureStatus": cmd.FailureStatus,
			"retryCount":    fmt.Sprintf("%d", cmd.RetryCount),
			"customerId":    order.CustomerID,
		},
	})

	return nil
}

// GetDeadLetterEntry retrieves a specific DLQ entry
func (s *ReprocessingService) GetDeadLetterEntry(ctx context.Context, orderID string) (*DeadLetterEntryDTO, error) {
	entry, err := s.deadLetterRepo.FindByOrderID(ctx, orderID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get DLQ entry", "orderId", orderID)
		return nil, fmt.Errorf("failed to get DLQ entry: %w", err)
	}

	if entry == nil {
		return nil, errors.ErrNotFound("dead letter entry")
	}

	return toDeadLetterEntryDTO(entry), nil
}

// ListDeadLetterQueue lists DLQ entries
func (s *ReprocessingService) ListDeadLetterQueue(ctx context.Context, query ListDLQQuery) (*DeadLetterListResponse, error) {
	// Default values
	if query.Limit <= 0 {
		query.Limit = 50
	}

	filter := mongoRepo.DLQFilter{
		Resolved:       query.Resolved,
		FailureStatus:  query.FailureStatus,
		CustomerID:     query.CustomerID,
		OlderThanHours: query.OlderThanHours,
	}

	entries, err := s.deadLetterRepo.List(ctx, filter, query.Limit, query.Offset)
	if err != nil {
		s.logger.WithError(err).Error("Failed to list DLQ entries")
		return nil, fmt.Errorf("failed to list DLQ entries: %w", err)
	}

	total, err := s.deadLetterRepo.Count(ctx, filter)
	if err != nil {
		s.logger.WithError(err).Error("Failed to count DLQ entries")
		return nil, fmt.Errorf("failed to count DLQ entries: %w", err)
	}

	dtos := make([]DeadLetterEntryDTO, 0, len(entries))
	for _, entry := range entries {
		dtos = append(dtos, *toDeadLetterEntryDTO(entry))
	}

	return &DeadLetterListResponse{
		Data:   dtos,
		Total:  total,
		Limit:  query.Limit,
		Offset: query.Offset,
	}, nil
}

// GetDLQStats retrieves DLQ statistics
func (s *ReprocessingService) GetDLQStats(ctx context.Context) (*DLQStatsDTO, error) {
	stats, err := s.deadLetterRepo.GetStats(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get DLQ stats")
		return nil, fmt.Errorf("failed to get DLQ stats: %w", err)
	}

	return &DLQStatsDTO{
		TotalEntries:     stats.TotalEntries,
		UnresolvedCount:  stats.UnresolvedCount,
		ResolvedCount:    stats.ResolvedCount,
		ByFailureStatus:  stats.ByFailureStatus,
		ByResolution:     stats.ByResolution,
		AverageRetries:   stats.AverageRetries,
		OldestUnresolved: stats.OldestUnresolved,
	}, nil
}

// ResolveDLQEntry resolves a DLQ entry
func (s *ReprocessingService) ResolveDLQEntry(ctx context.Context, cmd ResolveDLQCommand) error {
	// Check entry exists
	entry, err := s.deadLetterRepo.FindByOrderID(ctx, cmd.OrderID)
	if err != nil {
		return fmt.Errorf("failed to get DLQ entry: %w", err)
	}
	if entry == nil {
		return errors.ErrNotFound("dead letter entry")
	}

	// Calculate age for metrics
	ageHours := time.Since(entry.MovedToQueueAt).Hours()

	if err := s.deadLetterRepo.Resolve(ctx, cmd.OrderID, cmd.Resolution, cmd.Notes, cmd.ResolvedBy); err != nil {
		return fmt.Errorf("failed to resolve DLQ entry: %w", err)
	}

	// Record DLQ resolution metric
	if s.failureMetrics != nil {
		s.failureMetrics.RecordDLQResolution(cmd.Resolution, ageHours)
	}

	s.logger.Info("Resolved DLQ entry", "orderId", cmd.OrderID, "resolution", cmd.Resolution)
	return nil
}

// --- Helper Functions ---

func toRetryMetadataDTO(m *domain.RetryMetadata) *RetryMetadataDTO {
	return &RetryMetadataDTO{
		OrderID:        m.OrderID,
		RetryCount:     m.RetryCount,
		MaxRetries:     m.MaxRetries,
		FailureStatus:  m.FailureStatus,
		FailureReason:  m.FailureReason,
		LastWorkflowID: m.LastWorkflowID,
		LastRunID:      m.LastRunID,
		LastFailureAt:  m.LastFailureAt,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

func toDeadLetterEntryDTO(e *domain.DeadLetterEntry) *DeadLetterEntryDTO {
	dto := &DeadLetterEntryDTO{
		OrderID:            e.OrderID,
		CustomerID:         e.CustomerID,
		OriginalWorkflowID: e.OriginalWorkflowID,
		FinalFailureStatus: e.FinalFailureStatus,
		FinalFailureReason: e.FinalFailureReason,
		TotalRetryAttempts: e.TotalRetryAttempts,
		MovedToQueueAt:     e.MovedToQueueAt,
		Resolution:         e.Resolution,
		ResolutionNotes:    e.ResolutionNotes,
		ResolvedBy:         e.ResolvedBy,
		ResolvedAt:         e.ResolvedAt,
	}

	// Convert retry history
	dto.RetryHistory = make([]RetryAttemptDTO, 0, len(e.RetryHistory))
	for _, attempt := range e.RetryHistory {
		dto.RetryHistory = append(dto.RetryHistory, RetryAttemptDTO{
			AttemptNumber: attempt.AttemptNumber,
			WorkflowID:    attempt.WorkflowID,
			FailedAt:      attempt.FailedAt,
			FailureReason: attempt.FailureReason,
		})
	}

	return dto
}
