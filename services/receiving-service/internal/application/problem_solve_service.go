package application

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/services/receiving-service/internal/domain"
	"github.com/wms-platform/shared/pkg/logging"
)

// ProblemSolveService handles problem solve operations
type ProblemSolveService struct {
	problemRepo  domain.ProblemTicketRepository
	shipmentRepo domain.InboundShipmentRepository
	logger       *logging.Logger
}

// NewProblemSolveService creates a new ProblemSolveService
func NewProblemSolveService(
	problemRepo domain.ProblemTicketRepository,
	shipmentRepo domain.InboundShipmentRepository,
	logger *logging.Logger,
) *ProblemSolveService {
	return &ProblemSolveService{
		problemRepo:  problemRepo,
		shipmentRepo: shipmentRepo,
		logger:       logger,
	}
}

// CreateProblemTicket creates a new problem ticket
func (s *ProblemSolveService) CreateProblemTicket(ctx context.Context, cmd CreateProblemTicketCommand) (*domain.ProblemTicket, error) {
	// Generate ticket ID
	ticketID := fmt.Sprintf("PROB-%s", time.Now().Format("20060102150405"))

	// Validate shipment exists
	shipment, err := s.shipmentRepo.FindByID(ctx, cmd.ShipmentID)
	if err != nil {
		return nil, err
	}
	if shipment == nil {
		return nil, fmt.Errorf("shipment not found: %s", cmd.ShipmentID)
	}

	// Create problem ticket
	ticket, err := domain.NewProblemTicket(
		ticketID,
		cmd.ShipmentID,
		cmd.SKU,
		cmd.ProductName,
		domain.ProblemType(cmd.ProblemType),
		cmd.Description,
		cmd.Quantity,
		cmd.CreatedBy,
		cmd.Priority,
	)
	if err != nil {
		return nil, err
	}

	// Add images if provided
	for _, imageURL := range cmd.ImageURLs {
		ticket.AddImage(imageURL)
	}

	// Save ticket
	if err := s.problemRepo.Save(ticket); err != nil {
		return nil, err
	}

	s.logger.Info("Created problem ticket",
		"ticketId", ticketID,
		"shipmentId", cmd.ShipmentID,
		"problemType", cmd.ProblemType,
	)

	return ticket, nil
}

// GetProblemTicket retrieves a problem ticket by ID
func (s *ProblemSolveService) GetProblemTicket(ctx context.Context, ticketID string) (*domain.ProblemTicket, error) {
	return s.problemRepo.FindByID(ticketID)
}

// GetProblemTicketsByShipment retrieves all problem tickets for a shipment
func (s *ProblemSolveService) GetProblemTicketsByShipment(ctx context.Context, shipmentID string) ([]*domain.ProblemTicket, error) {
	return s.problemRepo.FindByShipmentID(shipmentID)
}

// GetPendingProblemTickets retrieves pending problem tickets
func (s *ProblemSolveService) GetPendingProblemTickets(ctx context.Context, limit int) ([]*domain.ProblemTicket, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	return s.problemRepo.FindPending(limit)
}

// AssignProblemTicket assigns a ticket to a resolver
func (s *ProblemSolveService) AssignProblemTicket(ctx context.Context, cmd AssignProblemTicketCommand) (*domain.ProblemTicket, error) {
	ticket, err := s.problemRepo.FindByID(cmd.TicketID)
	if err != nil {
		return nil, err
	}
	if ticket == nil {
		return nil, fmt.Errorf("problem ticket not found: %s", cmd.TicketID)
	}

	if err := ticket.AssignTo(cmd.AssignedTo); err != nil {
		return nil, err
	}

	if err := s.problemRepo.Save(ticket); err != nil {
		return nil, err
	}

	s.logger.Info("Assigned problem ticket",
		"ticketId", cmd.TicketID,
		"assignedTo", cmd.AssignedTo,
	)

	return ticket, nil
}

// ResolveProblemTicket resolves a problem ticket
func (s *ProblemSolveService) ResolveProblemTicket(ctx context.Context, cmd ResolveProblemTicketCommand) (*domain.ProblemTicket, error) {
	ticket, err := s.problemRepo.FindByID(cmd.TicketID)
	if err != nil {
		return nil, err
	}
	if ticket == nil {
		return nil, fmt.Errorf("problem ticket not found: %s", cmd.TicketID)
	}

	resolution := domain.ProblemResolution(cmd.Resolution)
	if err := ticket.Resolve(resolution, cmd.ResolutionNotes, cmd.ResolvedBy); err != nil {
		return nil, err
	}

	if err := s.problemRepo.Save(ticket); err != nil {
		return nil, err
	}

	s.logger.Info("Resolved problem ticket",
		"ticketId", cmd.TicketID,
		"resolution", cmd.Resolution,
		"resolvedBy", cmd.ResolvedBy,
	)

	return ticket, nil
}

// GetProblemTicketsByResolution retrieves tickets by resolution status
func (s *ProblemSolveService) GetProblemTicketsByResolution(ctx context.Context, resolution string, limit int) ([]*domain.ProblemTicket, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	return s.problemRepo.FindByResolution(domain.ProblemResolution(resolution), limit)
}
