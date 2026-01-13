package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// SortationWorkflowInput represents the input for the sortation workflow
type SortationWorkflowInput struct {
	OrderID          string  `json:"orderId"`
	PackageID        string  `json:"packageId"`
	TrackingNumber   string  `json:"trackingNumber"`
	ManifestID       string  `json:"manifestId"`
	CarrierID        string  `json:"carrierId"`
	Destination      string  `json:"destination"` // zip code or region
	Weight           float64 `json:"weight"`
	SortationCenter  string  `json:"sortationCenter,omitempty"`
	// Multi-tenant context
	TenantID    string `json:"tenantId"`
	FacilityID  string `json:"facilityId"`
	WarehouseID string `json:"warehouseId"`
}

// SortationWorkflowResult represents the result of the sortation workflow
type SortationWorkflowResult struct {
	BatchID          string    `json:"batchId"`
	PackageID        string    `json:"packageId"`
	ChuteID          string    `json:"chuteId"`
	ChuteNumber      int       `json:"chuteNumber"`
	Zone             string    `json:"zone"`
	DestinationGroup string    `json:"destinationGroup"`
	SortedAt         time.Time `json:"sortedAt"`
	Success          bool      `json:"success"`
}

// SortationWorkflow orchestrates the sortation process for a package
// This workflow handles: Batch Creation/Assignment -> Chute Assignment -> Package Sorting
func SortationWorkflow(ctx workflow.Context, input SortationWorkflowInput) (*SortationWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting sortation workflow",
		"orderId", input.OrderID,
		"packageId", input.PackageID,
		"destination", input.Destination,
	)

	result := &SortationWorkflowResult{
		PackageID: input.PackageID,
		Success:   false,
	}

	// Activity options with retry policy
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Determine sortation center (default if not specified)
	sortationCenter := input.SortationCenter
	if sortationCenter == "" {
		sortationCenter = "MAIN-SORT"
	}

	// Calculate destination group from zip code (first 3 digits)
	destinationGroup := input.Destination
	if len(destinationGroup) > 3 {
		destinationGroup = destinationGroup[:3]
	}

	// ========================================
	// Step 1: Create or Get Sortation Batch
	// ========================================
	logger.Info("Step 1: Creating/getting sortation batch",
		"packageId", input.PackageID,
		"destinationGroup", destinationGroup,
		"carrierId", input.CarrierID,
	)

	var batchResult struct {
		BatchID          string    `json:"batchId"`
		SortationCenter  string    `json:"sortationCenter"`
		DestinationGroup string    `json:"destinationGroup"`
		CarrierID        string    `json:"carrierId"`
		CreatedAt        time.Time `json:"createdAt"`
	}

	err := workflow.ExecuteActivity(ctx, "CreateSortationBatch", map[string]interface{}{
		"sortationCenter":  sortationCenter,
		"destinationGroup": destinationGroup,
		"carrierId":        input.CarrierID,
	}).Get(ctx, &batchResult)
	if err != nil {
		logger.Error("Failed to create sortation batch", "error", err)
		return result, fmt.Errorf("failed to create sortation batch: %w", err)
	}

	result.BatchID = batchResult.BatchID
	result.DestinationGroup = destinationGroup

	// ========================================
	// Step 2: Assign Chute
	// ========================================
	logger.Info("Step 2: Assigning chute",
		"packageId", input.PackageID,
		"destination", input.Destination,
	)

	var chuteResult struct {
		PackageID   string `json:"packageId"`
		ChuteID     string `json:"chuteId"`
		ChuteNumber int    `json:"chuteNumber"`
		Zone        string `json:"zone"`
	}

	err = workflow.ExecuteActivity(ctx, "AssignChute", map[string]interface{}{
		"packageId":   input.PackageID,
		"destination": input.Destination,
		"carrierId":   input.CarrierID,
	}).Get(ctx, &chuteResult)
	if err != nil {
		logger.Error("Failed to assign chute", "error", err)
		return result, fmt.Errorf("failed to assign chute: %w", err)
	}

	result.ChuteID = chuteResult.ChuteID
	result.ChuteNumber = chuteResult.ChuteNumber
	result.Zone = chuteResult.Zone

	// ========================================
	// Step 3: Add Package to Batch
	// ========================================
	logger.Info("Step 3: Adding package to batch",
		"batchId", result.BatchID,
		"packageId", input.PackageID,
	)

	err = workflow.ExecuteActivity(ctx, "AddPackageToBatch", map[string]interface{}{
		"batchId": result.BatchID,
		"package": map[string]interface{}{
			"packageId":      input.PackageID,
			"orderId":        input.OrderID,
			"trackingNumber": input.TrackingNumber,
			"destination":    input.Destination,
			"carrierId":      input.CarrierID,
			"weight":         input.Weight,
		},
	}).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to add package to batch", "error", err)
		return result, fmt.Errorf("failed to add package to batch: %w", err)
	}

	// ========================================
	// Step 4: Sort Package to Chute
	// ========================================
	logger.Info("Step 4: Sorting package to chute",
		"batchId", result.BatchID,
		"packageId", input.PackageID,
		"chuteId", result.ChuteID,
	)

	var sortResult struct {
		BatchID   string    `json:"batchId"`
		PackageID string    `json:"packageId"`
		ChuteID   string    `json:"chuteId"`
		SortedBy  string    `json:"sortedBy"`
		SortedAt  time.Time `json:"sortedAt"`
		Success   bool      `json:"success"`
	}

	err = workflow.ExecuteActivity(ctx, "SortPackage", map[string]interface{}{
		"batchId":   result.BatchID,
		"packageId": input.PackageID,
		"chuteId":   result.ChuteID,
		"workerId":  "SYSTEM", // Could be passed in for manual sorting
	}).Get(ctx, &sortResult)
	if err != nil {
		logger.Error("Failed to sort package", "error", err)
		return result, fmt.Errorf("failed to sort package: %w", err)
	}

	result.SortedAt = sortResult.SortedAt
	result.Success = sortResult.Success

	logger.Info("Sortation workflow completed successfully",
		"orderId", input.OrderID,
		"packageId", input.PackageID,
		"batchId", result.BatchID,
		"chuteId", result.ChuteID,
		"zone", result.Zone,
	)

	return result, nil
}

// BatchSortationWorkflowInput represents input for batch sortation operations
type BatchSortationWorkflowInput struct {
	SortationCenter  string                    `json:"sortationCenter"`
	CarrierID        string                    `json:"carrierId"`
	Packages         []SortationPackageInput   `json:"packages"`
}

// SortationPackageInput represents a package for sortation
type SortationPackageInput struct {
	PackageID      string  `json:"packageId"`
	OrderID        string  `json:"orderId"`
	TrackingNumber string  `json:"trackingNumber"`
	Destination    string  `json:"destination"`
	Weight         float64 `json:"weight"`
}

// BatchSortationWorkflowResult represents the result of batch sortation
type BatchSortationWorkflowResult struct {
	BatchID       string `json:"batchId"`
	TotalPackages int    `json:"totalPackages"`
	SortedCount   int    `json:"sortedCount"`
	FailedCount   int    `json:"failedCount"`
	Success       bool   `json:"success"`
}

// BatchSortationWorkflow processes multiple packages for sortation
func BatchSortationWorkflow(ctx workflow.Context, input BatchSortationWorkflowInput) (*BatchSortationWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting batch sortation workflow",
		"sortationCenter", input.SortationCenter,
		"carrierId", input.CarrierID,
		"packageCount", len(input.Packages),
	)

	// Activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Execute batch sortation using the activity
	var processResult struct {
		BatchID       string `json:"batchId"`
		TotalPackages int    `json:"totalPackages"`
		SortedCount   int    `json:"sortedCount"`
		FailedCount   int    `json:"failedCount"`
		Success       bool   `json:"success"`
	}

	// Convert packages to activity format
	packages := make([]map[string]interface{}, len(input.Packages))
	for i, pkg := range input.Packages {
		packages[i] = map[string]interface{}{
			"packageId":      pkg.PackageID,
			"orderId":        pkg.OrderID,
			"trackingNumber": pkg.TrackingNumber,
			"destination":    pkg.Destination,
			"weight":         pkg.Weight,
		}
	}

	err := workflow.ExecuteActivity(ctx, "ProcessSortation", map[string]interface{}{
		"sortationCenter": input.SortationCenter,
		"carrierId":       input.CarrierID,
		"packages":        packages,
	}).Get(ctx, &processResult)
	if err != nil {
		logger.Error("Batch sortation failed", "error", err)
		return nil, fmt.Errorf("batch sortation failed: %w", err)
	}

	result := &BatchSortationWorkflowResult{
		BatchID:       processResult.BatchID,
		TotalPackages: processResult.TotalPackages,
		SortedCount:   processResult.SortedCount,
		FailedCount:   processResult.FailedCount,
		Success:       processResult.Success,
	}

	logger.Info("Batch sortation workflow completed",
		"batchId", result.BatchID,
		"sorted", result.SortedCount,
		"failed", result.FailedCount,
	)

	return result, nil
}
