package activities

import (
	"context"
	"fmt"

	"github.com/wms-platform/orchestrator/internal/activities/clients"
	"go.temporal.io/sdk/activity"
)

// LaborActivities contains labor and certification validation activities
type LaborActivities struct {
	clients *clients.ServiceClients
}

// NewLaborActivities creates a new LaborActivities instance
func NewLaborActivities(clients *clients.ServiceClients) *LaborActivities {
	return &LaborActivities{
		clients: clients,
	}
}

// ValidateCertificationInput represents input for validating worker certifications
type ValidateCertificationInput struct {
	RequiredSkills []string `json:"requiredSkills"`
	Zone           string   `json:"zone,omitempty"`
	ShiftTime      string   `json:"shiftTime,omitempty"` // Expected shift time
	MinWorkers     int      `json:"minWorkers"`          // Minimum certified workers needed
}

// ValidateCertificationResult represents the result of certification validation
type ValidateCertificationResult struct {
	CertifiedWorkersAvailable int      `json:"certifiedWorkersAvailable"`
	AvailableWorkerIDs        []string `json:"availableWorkerIds"`
	MissingSkills             []string `json:"missingSkills,omitempty"`
	SufficientLabor           bool     `json:"sufficientLabor"`
	Success                   bool     `json:"success"`
}

// ValidateWorkerCertification validates that certified workers are available for required skills
func (a *LaborActivities) ValidateWorkerCertification(ctx context.Context, input ValidateCertificationInput) (*ValidateCertificationResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Validating worker certifications",
		"requiredSkills", input.RequiredSkills,
		"zone", input.Zone,
		"minWorkers", input.MinWorkers,
	)

	if len(input.RequiredSkills) == 0 {
		// No special skills required, validation passes
		return &ValidateCertificationResult{
			CertifiedWorkersAvailable: 0,
			AvailableWorkerIDs:        []string{},
			SufficientLabor:           true,
			Success:                   true,
		}, nil
	}

	// Call labor service to find certified workers
	req := &clients.FindCertifiedWorkersRequest{
		RequiredSkills: input.RequiredSkills,
		Zone:           input.Zone,
		ShiftTime:      input.ShiftTime,
		MinCount:       input.MinWorkers,
	}

	workers, err := a.clients.FindCertifiedWorkers(ctx, req)
	if err != nil {
		logger.Error("Failed to find certified workers",
			"requiredSkills", input.RequiredSkills,
			"error", err,
		)
		return &ValidateCertificationResult{
			CertifiedWorkersAvailable: 0,
			SufficientLabor:           false,
			Success:                   false,
		}, fmt.Errorf("failed to find certified workers: %w", err)
	}

	// Extract worker IDs
	workerIDs := make([]string, len(workers))
	for i, worker := range workers {
		workerIDs[i] = worker.WorkerID
	}

	// Check if we have sufficient certified labor
	sufficientLabor := len(workers) >= input.MinWorkers

	// Identify missing skills if insufficient workers
	var missingSkills []string
	if !sufficientLabor {
		// Check which skills have insufficient coverage
		skillCoverage := make(map[string]int)
		for _, worker := range workers {
			for _, skill := range worker.Skills {
				skillCoverage[skill]++
			}
		}

		for _, requiredSkill := range input.RequiredSkills {
			if skillCoverage[requiredSkill] < input.MinWorkers {
				missingSkills = append(missingSkills, requiredSkill)
			}
		}
	}

	logger.Info("Worker certification validation complete",
		"certifiedWorkers", len(workers),
		"requiredMinimum", input.MinWorkers,
		"sufficientLabor", sufficientLabor,
		"missingSkills", missingSkills,
	)

	return &ValidateCertificationResult{
		CertifiedWorkersAvailable: len(workers),
		AvailableWorkerIDs:        workerIDs,
		MissingSkills:             missingSkills,
		SufficientLabor:           sufficientLabor,
		Success:                   true,
	}, nil
}

// AssignCertifiedWorkerInput represents input for assigning a certified worker
type AssignCertifiedWorkerInput struct {
	OrderID        string   `json:"orderId"`
	StationID      string   `json:"stationId"`
	RequiredSkills []string `json:"requiredSkills"`
	Zone           string   `json:"zone,omitempty"`
	Priority       string   `json:"priority,omitempty"`
}

// AssignCertifiedWorker assigns a certified worker to an order/station
func (a *LaborActivities) AssignCertifiedWorker(ctx context.Context, input AssignCertifiedWorkerInput) (*clients.Worker, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Assigning certified worker",
		"orderId", input.OrderID,
		"stationId", input.StationID,
		"requiredSkills", input.RequiredSkills,
	)

	// Call labor service to assign worker
	req := &clients.AssignWorkerRequest{
		OrderID:        input.OrderID,
		StationID:      input.StationID,
		RequiredSkills: input.RequiredSkills,
		Zone:           input.Zone,
		Priority:       input.Priority,
	}

	worker, err := a.clients.AssignWorker(ctx, req)
	if err != nil {
		logger.Error("Failed to assign certified worker",
			"orderId", input.OrderID,
			"stationId", input.StationID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to assign certified worker: %w", err)
	}

	logger.Info("Certified worker assigned",
		"orderId", input.OrderID,
		"workerId", worker.WorkerID,
		"workerName", worker.Name,
		"skills", worker.Skills,
	)

	return worker, nil
}

// GetAvailableWorkersInput represents input for getting available workers
type GetAvailableWorkersInput struct {
	Zone           string   `json:"zone,omitempty"`
	RequiredSkills []string `json:"requiredSkills,omitempty"`
	ShiftTime      string   `json:"shiftTime,omitempty"`
}

// GetAvailableWorkers retrieves available workers (optionally filtered by skills)
func (a *LaborActivities) GetAvailableWorkers(ctx context.Context, input GetAvailableWorkersInput) ([]clients.Worker, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Getting available workers",
		"zone", input.Zone,
		"requiredSkills", input.RequiredSkills,
	)

	req := &clients.GetAvailableWorkersRequest{
		Zone:           input.Zone,
		RequiredSkills: input.RequiredSkills,
		ShiftTime:      input.ShiftTime,
	}

	workers, err := a.clients.GetAvailableWorkers(ctx, req)
	if err != nil {
		logger.Error("Failed to get available workers", "error", err)
		return nil, fmt.Errorf("failed to get available workers: %w", err)
	}

	logger.Info("Available workers retrieved", "count", len(workers))
	return workers, nil
}
