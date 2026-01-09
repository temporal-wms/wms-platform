package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/wms-platform/walling-service/internal/domain"
)

// WallingApplicationService is the main application service for the Walling bounded context
type WallingApplicationService struct {
	taskRepo domain.WallingTaskRepository
	logger   *slog.Logger
}

// NewWallingApplicationService creates a new WallingApplicationService
func NewWallingApplicationService(taskRepo domain.WallingTaskRepository, logger *slog.Logger) *WallingApplicationService {
	return &WallingApplicationService{
		taskRepo: taskRepo,
		logger:   logger,
	}
}

// CreateWallingTaskCommand represents a command to create a walling task
type CreateWallingTaskCommand struct {
	OrderID        string              `json:"orderId"`
	WaveID         string              `json:"waveId"`
	RouteID        string              `json:"routeId,omitempty"`
	PutWallID      string              `json:"putWallId"`
	DestinationBin string              `json:"destinationBin"`
	SourceTotes    []SourceToteDTO     `json:"sourceTotes"`
	Items          []ItemToSortDTO     `json:"items"`
}

// SourceToteDTO represents a source tote
type SourceToteDTO struct {
	ToteID     string `json:"toteId"`
	PickTaskID string `json:"pickTaskId"`
	ItemCount  int    `json:"itemCount"`
}

// ItemToSortDTO represents an item to sort
type ItemToSortDTO struct {
	SKU        string `json:"sku"`
	Quantity   int    `json:"quantity"`
	FromToteID string `json:"fromToteId"`
}

// AssignWallinerCommand represents a command to assign a walliner
type AssignWallinerCommand struct {
	TaskID     string `json:"taskId"`
	WallinerID string `json:"wallinerId"`
	Station    string `json:"station"`
}

// SortItemCommand represents a command to sort an item
type SortItemCommand struct {
	TaskID     string `json:"taskId"`
	SKU        string `json:"sku"`
	Quantity   int    `json:"quantity"`
	FromToteID string `json:"fromToteId"`
}

// CompleteTaskCommand represents a command to complete a task
type CompleteTaskCommand struct {
	TaskID string `json:"taskId"`
}

// WallingTaskDTO represents a walling task for API responses
type WallingTaskDTO struct {
	TaskID         string            `json:"taskId"`
	OrderID        string            `json:"orderId"`
	WaveID         string            `json:"waveId"`
	RouteID        string            `json:"routeId,omitempty"`
	WallinerID     string            `json:"wallinerId,omitempty"`
	Status         string            `json:"status"`
	PutWallID      string            `json:"putWallId"`
	DestinationBin string            `json:"destinationBin"`
	SourceTotes    []SourceToteDTO   `json:"sourceTotes"`
	ItemsToSort    []ItemToSortDTO   `json:"itemsToSort"`
	SortedItems    []SortedItemDTO   `json:"sortedItems"`
	Station        string            `json:"station,omitempty"`
	Priority       int               `json:"priority"`
	Progress       ProgressDTO       `json:"progress"`
	CreatedAt      int64             `json:"createdAt"`
	AssignedAt     *int64            `json:"assignedAt,omitempty"`
	StartedAt      *int64            `json:"startedAt,omitempty"`
	CompletedAt    *int64            `json:"completedAt,omitempty"`
}

// SortedItemDTO represents a sorted item
type SortedItemDTO struct {
	SKU        string `json:"sku"`
	Quantity   int    `json:"quantity"`
	FromToteID string `json:"fromToteId"`
	ToBinID    string `json:"toBinId"`
	SortedAt   int64  `json:"sortedAt"`
	Verified   bool   `json:"verified"`
}

// ProgressDTO represents progress information
type ProgressDTO struct {
	Sorted int `json:"sorted"`
	Total  int `json:"total"`
}

// CreateWallingTask creates a new walling task
func (s *WallingApplicationService) CreateWallingTask(ctx context.Context, cmd CreateWallingTaskCommand) (*WallingTaskDTO, error) {
	s.logger.Info("Creating walling task", "orderId", cmd.OrderID, "putWallId", cmd.PutWallID)

	// Convert DTOs to domain objects
	sourceTotes := make([]domain.SourceTote, len(cmd.SourceTotes))
	for i, st := range cmd.SourceTotes {
		sourceTotes[i] = domain.SourceTote{
			ToteID:     st.ToteID,
			PickTaskID: st.PickTaskID,
			ItemCount:  st.ItemCount,
		}
	}

	items := make([]domain.ItemToSort, len(cmd.Items))
	for i, item := range cmd.Items {
		items[i] = domain.ItemToSort{
			SKU:        item.SKU,
			Quantity:   item.Quantity,
			FromToteID: item.FromToteID,
		}
	}

	task, err := domain.NewWallingTask(cmd.OrderID, cmd.WaveID, cmd.PutWallID, cmd.DestinationBin, sourceTotes, items)
	if err != nil {
		return nil, fmt.Errorf("failed to create walling task: %w", err)
	}

	if cmd.RouteID != "" {
		task.SetRouteID(cmd.RouteID)
	}

	if err := s.taskRepo.Save(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to save walling task: %w", err)
	}

	s.logger.Info("Walling task created", "taskId", task.TaskID, "orderId", cmd.OrderID)

	return mapTaskToDTO(task), nil
}

// AssignWalliner assigns a walliner to a task
func (s *WallingApplicationService) AssignWalliner(ctx context.Context, cmd AssignWallinerCommand) (*WallingTaskDTO, error) {
	s.logger.Info("Assigning walliner", "taskId", cmd.TaskID, "wallinerId", cmd.WallinerID)

	task, err := s.taskRepo.FindByTaskID(ctx, cmd.TaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to find task: %w", err)
	}
	if task == nil {
		return nil, fmt.Errorf("task not found: %s", cmd.TaskID)
	}

	if err := task.Assign(cmd.WallinerID, cmd.Station); err != nil {
		return nil, fmt.Errorf("failed to assign walliner: %w", err)
	}

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	s.logger.Info("Walliner assigned", "taskId", cmd.TaskID, "wallinerId", cmd.WallinerID)

	return mapTaskToDTO(task), nil
}

// SortItem sorts an item to the destination bin
func (s *WallingApplicationService) SortItem(ctx context.Context, cmd SortItemCommand) (*WallingTaskDTO, error) {
	s.logger.Info("Sorting item", "taskId", cmd.TaskID, "sku", cmd.SKU, "quantity", cmd.Quantity)

	task, err := s.taskRepo.FindByTaskID(ctx, cmd.TaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to find task: %w", err)
	}
	if task == nil {
		return nil, fmt.Errorf("task not found: %s", cmd.TaskID)
	}

	if err := task.SortItem(cmd.SKU, cmd.Quantity, cmd.FromToteID); err != nil {
		return nil, fmt.Errorf("failed to sort item: %w", err)
	}

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	s.logger.Info("Item sorted", "taskId", cmd.TaskID, "sku", cmd.SKU, "status", task.Status)

	return mapTaskToDTO(task), nil
}

// CompleteTask completes a walling task
func (s *WallingApplicationService) CompleteTask(ctx context.Context, cmd CompleteTaskCommand) (*WallingTaskDTO, error) {
	s.logger.Info("Completing task", "taskId", cmd.TaskID)

	task, err := s.taskRepo.FindByTaskID(ctx, cmd.TaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to find task: %w", err)
	}
	if task == nil {
		return nil, fmt.Errorf("task not found: %s", cmd.TaskID)
	}

	if err := task.Complete(); err != nil {
		return nil, fmt.Errorf("failed to complete task: %w", err)
	}

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	s.logger.Info("Task completed", "taskId", cmd.TaskID)

	return mapTaskToDTO(task), nil
}

// GetTask gets a task by ID
func (s *WallingApplicationService) GetTask(ctx context.Context, taskID string) (*WallingTaskDTO, error) {
	task, err := s.taskRepo.FindByTaskID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to find task: %w", err)
	}
	return mapTaskToDTO(task), nil
}

// GetActiveTaskByWalliner gets the active task for a walliner
func (s *WallingApplicationService) GetActiveTaskByWalliner(ctx context.Context, wallinerID string) (*WallingTaskDTO, error) {
	task, err := s.taskRepo.FindActiveByWalliner(ctx, wallinerID)
	if err != nil {
		return nil, fmt.Errorf("failed to find active task: %w", err)
	}
	return mapTaskToDTO(task), nil
}

// GetPendingTasksByPutWall gets pending tasks for a put wall
func (s *WallingApplicationService) GetPendingTasksByPutWall(ctx context.Context, putWallID string, limit int) ([]*WallingTaskDTO, error) {
	tasks, err := s.taskRepo.FindPendingByPutWall(ctx, putWallID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to find pending tasks: %w", err)
	}

	dtos := make([]*WallingTaskDTO, len(tasks))
	for i, task := range tasks {
		dtos[i] = mapTaskToDTO(task)
	}
	return dtos, nil
}

// mapTaskToDTO maps a domain WallingTask to DTO
func mapTaskToDTO(task *domain.WallingTask) *WallingTaskDTO {
	if task == nil {
		return nil
	}

	sourceTotes := make([]SourceToteDTO, len(task.SourceTotes))
	for i, st := range task.SourceTotes {
		sourceTotes[i] = SourceToteDTO{
			ToteID:     st.ToteID,
			PickTaskID: st.PickTaskID,
			ItemCount:  st.ItemCount,
		}
	}

	itemsToSort := make([]ItemToSortDTO, len(task.ItemsToSort))
	for i, item := range task.ItemsToSort {
		itemsToSort[i] = ItemToSortDTO{
			SKU:        item.SKU,
			Quantity:   item.Quantity,
			FromToteID: item.FromToteID,
		}
	}

	sortedItems := make([]SortedItemDTO, len(task.SortedItems))
	for i, item := range task.SortedItems {
		sortedItems[i] = SortedItemDTO{
			SKU:        item.SKU,
			Quantity:   item.Quantity,
			FromToteID: item.FromToteID,
			ToBinID:    item.ToBinID,
			SortedAt:   item.SortedAt.UnixMilli(),
			Verified:   item.Verified,
		}
	}

	sorted, total := task.GetProgress()

	dto := &WallingTaskDTO{
		TaskID:         task.TaskID,
		OrderID:        task.OrderID,
		WaveID:         task.WaveID,
		RouteID:        task.RouteID,
		WallinerID:     task.WallinerID,
		Status:         string(task.Status),
		PutWallID:      task.PutWallID,
		DestinationBin: task.DestinationBin,
		SourceTotes:    sourceTotes,
		ItemsToSort:    itemsToSort,
		SortedItems:    sortedItems,
		Station:        task.Station,
		Priority:       task.Priority,
		Progress: ProgressDTO{
			Sorted: sorted,
			Total:  total,
		},
		CreatedAt: task.CreatedAt.UnixMilli(),
	}

	if task.AssignedAt != nil {
		ts := task.AssignedAt.UnixMilli()
		dto.AssignedAt = &ts
	}
	if task.StartedAt != nil {
		ts := task.StartedAt.UnixMilli()
		dto.StartedAt = &ts
	}
	if task.CompletedAt != nil {
		ts := task.CompletedAt.UnixMilli()
		dto.CompletedAt = &ts
	}

	return dto
}
