package activities

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"go.temporal.io/sdk/activity"
)

// StowActivities contains activities for the stow workflow
type StowActivities struct {
	// In a real implementation, these would be service clients
}

// NewStowActivities creates a new StowActivities instance
func NewStowActivities() *StowActivities {
	return &StowActivities{}
}

// StowTaskInput represents a stow task input
type StowTaskInput struct {
	TaskID            string  `json:"taskId"`
	ShipmentID        string  `json:"shipmentId"`
	SKU               string  `json:"sku"`
	ProductName       string  `json:"productName"`
	Quantity          int     `json:"quantity"`
	SourceToteID      string  `json:"sourceToteId"`
	IsHazmat          bool    `json:"isHazmat"`
	RequiresColdChain bool    `json:"requiresColdChain"`
	IsOversized       bool    `json:"isOversized"`
	Weight            float64 `json:"weight"`
}

// StorageLocation represents a storage location
type StorageLocation struct {
	LocationID string `json:"locationId"`
	Zone       string `json:"zone"`
	Aisle      string `json:"aisle"`
	Rack       int    `json:"rack"`
	Level      int    `json:"level"`
	Bin        string `json:"bin"`
}

// FindStorageLocationInput represents input for finding a location
type FindStorageLocationInput struct {
	TaskID            string `json:"taskId"`
	SKU               string `json:"sku"`
	Quantity          int    `json:"quantity"`
	IsHazmat          bool   `json:"isHazmat"`
	RequiresColdChain bool   `json:"requiresColdChain"`
	IsOversized       bool   `json:"isOversized"`
	Strategy          string `json:"strategy"` // chaotic, directed, velocity
}

// FindStorageLocation finds a storage location for an item using chaotic storage
func (a *StowActivities) FindStorageLocation(ctx context.Context, input FindStorageLocationInput) (*StorageLocation, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Finding storage location",
		"taskId", input.TaskID,
		"sku", input.SKU,
		"strategy", input.Strategy,
	)

	// Determine zone based on constraints
	zone := "GENERAL"
	if input.IsHazmat {
		zone = "HAZMAT"
	} else if input.RequiresColdChain {
		zone = "COLD"
	} else if input.IsOversized {
		zone = "OVERSIZE"
	}

	// Chaotic storage: random aisle and rack within zone
	aisles := []string{"A", "B", "C", "D", "E"}
	aisle := aisles[rand.Intn(len(aisles))]
	rack := rand.Intn(10) + 1
	level := rand.Intn(4) + 1
	bin := fmt.Sprintf("%s%02d", string(rune('A'+rand.Intn(6))), rand.Intn(10)+1)

	location := &StorageLocation{
		LocationID: fmt.Sprintf("%s-%s-%02d-%d-%s", zone, aisle, rack, level, bin),
		Zone:       zone,
		Aisle:      aisle,
		Rack:       rack,
		Level:      level,
		Bin:        bin,
	}

	logger.Info("Found storage location",
		"locationId", location.LocationID,
		"zone", location.Zone,
	)

	return location, nil
}

// AssignLocationInput represents input for assigning a location
type AssignLocationInput struct {
	TaskID     string           `json:"taskId"`
	Location   *StorageLocation `json:"location"`
}

// AssignLocation assigns a storage location to a stow task
func (a *StowActivities) AssignLocation(ctx context.Context, input AssignLocationInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Assigning location",
		"taskId", input.TaskID,
		"locationId", input.Location.LocationID,
	)

	// In production, this would update the task with the assigned location
	return nil
}

// ExecuteStowInput represents input for executing stow
type ExecuteStowInput struct {
	TaskID       string           `json:"taskId"`
	SKU          string           `json:"sku"`
	Quantity     int              `json:"quantity"`
	SourceToteID string           `json:"sourceToteId"`
	Location     *StorageLocation `json:"location"`
	WorkerID     string           `json:"workerId,omitempty"`
}

// ExecuteStowResult represents the result of stow execution
type ExecuteStowResult struct {
	TaskID        string    `json:"taskId"`
	SKU           string    `json:"sku"`
	StowedQuantity int      `json:"stowedQuantity"`
	LocationID    string    `json:"locationId"`
	StowedAt      time.Time `json:"stowedAt"`
	Success       bool      `json:"success"`
}

// ExecuteStow executes the stow operation
func (a *StowActivities) ExecuteStow(ctx context.Context, input ExecuteStowInput) (*ExecuteStowResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Executing stow",
		"taskId", input.TaskID,
		"sku", input.SKU,
		"quantity", input.Quantity,
		"locationId", input.Location.LocationID,
	)

	// In production, this would:
	// 1. Verify location is accessible
	// 2. Record stow action
	// 3. Update inventory system
	// 4. Update location capacity

	result := &ExecuteStowResult{
		TaskID:         input.TaskID,
		SKU:            input.SKU,
		StowedQuantity: input.Quantity,
		LocationID:     input.Location.LocationID,
		StowedAt:       time.Now(),
		Success:        true,
	}

	return result, nil
}

// UpdateInventoryLocationInput represents input for updating inventory location
type UpdateInventoryLocationInput struct {
	SKU        string `json:"sku"`
	LocationID string `json:"locationId"`
	Quantity   int    `json:"quantity"`
	Weight     float64 `json:"weight"`
}

// UpdateInventoryLocation updates inventory with the new location
func (a *StowActivities) UpdateInventoryLocation(ctx context.Context, input UpdateInventoryLocationInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Updating inventory location",
		"sku", input.SKU,
		"locationId", input.LocationID,
		"quantity", input.Quantity,
	)

	// In production, this would update the inventory service with the new location
	return nil
}

// StowWorkflowInput represents input for the stow child workflow
type StowWorkflowInput struct {
	ShipmentID string   `json:"shipmentId"`
	TaskIDs    []string `json:"taskIds"`
}

// StowWorkflowResult represents the result of the stow workflow
type StowWorkflowResult struct {
	ShipmentID  string `json:"shipmentId"`
	StowedCount int    `json:"stowedCount"`
	FailedCount int    `json:"failedCount"`
	Success     bool   `json:"success"`
}

// ProcessStow processes stow tasks (simulated child workflow activity)
func (a *StowActivities) ProcessStow(ctx context.Context, input StowWorkflowInput) (*StowWorkflowResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Processing stow workflow",
		"shipmentId", input.ShipmentID,
		"taskCount", len(input.TaskIDs),
	)

	// Simulated: all tasks succeed
	result := &StowWorkflowResult{
		ShipmentID:  input.ShipmentID,
		StowedCount: len(input.TaskIDs),
		FailedCount: 0,
		Success:     true,
	}

	return result, nil
}

// RegisterStowActivities registers all stow activities with the worker
func RegisterStowActivities(activities *StowActivities) map[string]interface{} {
	return map[string]interface{}{
		"FindStorageLocation":     activities.FindStorageLocation,
		"AssignLocation":          activities.AssignLocation,
		"ExecuteStow":             activities.ExecuteStow,
		"UpdateInventoryLocation": activities.UpdateInventoryLocation,
		"StowWorkflow":            activities.ProcessStow,
	}
}
