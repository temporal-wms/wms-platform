package application

import (
	"context"
	"fmt"

	"github.com/wms-platform/services/seller-service/internal/domain"
	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/logging"
)

// SellerApplicationService handles seller-related use cases
type SellerApplicationService struct {
	sellerRepo domain.SellerRepository
	logger     *logging.Logger
}

// NewSellerApplicationService creates a new SellerApplicationService
func NewSellerApplicationService(
	sellerRepo domain.SellerRepository,
	logger *logging.Logger,
) *SellerApplicationService {
	return &SellerApplicationService{
		sellerRepo: sellerRepo,
		logger:     logger,
	}
}

// CreateSeller creates a new seller
func (s *SellerApplicationService) CreateSeller(ctx context.Context, cmd CreateSellerCommand) (*SellerDTO, error) {
	// Check if email already exists
	existing, err := s.sellerRepo.FindByEmail(ctx, cmd.ContactEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if existing != nil {
		return nil, errors.ErrConflict("seller with this email already exists")
	}

	// Create the seller aggregate
	seller, err := domain.NewSeller(
		cmd.TenantID,
		cmd.CompanyName,
		cmd.ContactName,
		cmd.ContactEmail,
		domain.BillingCycle(cmd.BillingCycle),
	)
	if err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	// Set optional phone
	if cmd.ContactPhone != "" {
		seller.ContactPhone = cmd.ContactPhone
	}

	// Save to repository
	if err := s.sellerRepo.Save(ctx, seller); err != nil {
		s.logger.WithError(err).Error("Failed to save seller", "sellerId", seller.SellerID)
		return nil, fmt.Errorf("failed to save seller: %w", err)
	}

	s.logger.Info("Seller created", "sellerId", seller.SellerID, "companyName", seller.CompanyName)

	return ToSellerDTO(seller), nil
}

// GetSeller retrieves a seller by ID
func (s *SellerApplicationService) GetSeller(ctx context.Context, query GetSellerQuery) (*SellerDTO, error) {
	seller, err := s.sellerRepo.FindByID(ctx, query.SellerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get seller: %w", err)
	}
	if seller == nil {
		return nil, errors.ErrNotFound("seller not found")
	}

	return ToSellerDTO(seller), nil
}

// ListSellers retrieves a paginated list of sellers
func (s *SellerApplicationService) ListSellers(ctx context.Context, query ListSellersQuery) (*SellerListResponse, error) {
	pagination := domain.Pagination{
		Page:     query.Page,
		PageSize: query.PageSize,
	}

	if pagination.Page < 1 {
		pagination.Page = 1
	}
	if pagination.PageSize < 1 || pagination.PageSize > 100 {
		pagination.PageSize = 20
	}

	// Build filter
	filter := domain.SellerFilter{
		TenantID:   query.TenantID,
		FacilityID: query.FacilityID,
		HasChannel: query.HasChannel,
	}

	if query.Status != nil {
		status := domain.SellerStatus(*query.Status)
		filter.Status = &status
	}

	// Get total count
	total, err := s.sellerRepo.Count(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count sellers: %w", err)
	}

	// Get sellers based on filter
	var sellers []*domain.Seller
	if query.Status != nil {
		sellers, err = s.sellerRepo.FindByStatus(ctx, domain.SellerStatus(*query.Status), pagination)
	} else if query.TenantID != nil {
		sellers, err = s.sellerRepo.FindByTenantID(ctx, *query.TenantID, pagination)
	} else {
		sellers, err = s.sellerRepo.FindByTenantID(ctx, "", pagination)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list sellers: %w", err)
	}

	// Convert to DTOs
	dtos := make([]SellerDTO, len(sellers))
	for i, seller := range sellers {
		dtos[i] = *ToSellerDTO(seller)
	}

	totalPages := (total + pagination.PageSize - 1) / pagination.PageSize

	return &SellerListResponse{
		Data:       dtos,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	}, nil
}

// ActivateSeller activates a seller account
func (s *SellerApplicationService) ActivateSeller(ctx context.Context, cmd ActivateSellerCommand) (*SellerDTO, error) {
	seller, err := s.sellerRepo.FindByID(ctx, cmd.SellerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get seller: %w", err)
	}
	if seller == nil {
		return nil, errors.ErrNotFound("seller not found")
	}

	if err := seller.Activate(); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.sellerRepo.Save(ctx, seller); err != nil {
		return nil, fmt.Errorf("failed to save seller: %w", err)
	}

	s.logger.Info("Seller activated", "sellerId", seller.SellerID)

	return ToSellerDTO(seller), nil
}

// SuspendSeller suspends a seller account
func (s *SellerApplicationService) SuspendSeller(ctx context.Context, cmd SuspendSellerCommand) (*SellerDTO, error) {
	seller, err := s.sellerRepo.FindByID(ctx, cmd.SellerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get seller: %w", err)
	}
	if seller == nil {
		return nil, errors.ErrNotFound("seller not found")
	}

	if err := seller.Suspend(cmd.Reason); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.sellerRepo.Save(ctx, seller); err != nil {
		return nil, fmt.Errorf("failed to save seller: %w", err)
	}

	s.logger.Info("Seller suspended", "sellerId", seller.SellerID, "reason", cmd.Reason)

	return ToSellerDTO(seller), nil
}

// CloseSeller closes a seller account
func (s *SellerApplicationService) CloseSeller(ctx context.Context, cmd CloseSellerCommand) (*SellerDTO, error) {
	seller, err := s.sellerRepo.FindByID(ctx, cmd.SellerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get seller: %w", err)
	}
	if seller == nil {
		return nil, errors.ErrNotFound("seller not found")
	}

	if err := seller.Close(cmd.Reason); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.sellerRepo.Save(ctx, seller); err != nil {
		return nil, fmt.Errorf("failed to save seller: %w", err)
	}

	s.logger.Info("Seller closed", "sellerId", seller.SellerID, "reason", cmd.Reason)

	return ToSellerDTO(seller), nil
}

// AssignFacility assigns a facility to a seller
func (s *SellerApplicationService) AssignFacility(ctx context.Context, cmd AssignFacilityCommand) (*SellerDTO, error) {
	seller, err := s.sellerRepo.FindByID(ctx, cmd.SellerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get seller: %w", err)
	}
	if seller == nil {
		return nil, errors.ErrNotFound("seller not found")
	}

	warehouseIDs := cmd.WarehouseIDs
	if warehouseIDs == nil {
		warehouseIDs = []string{}
	}

	if err := seller.AssignFacility(cmd.FacilityID, cmd.FacilityName, warehouseIDs, cmd.AllocatedSpace, cmd.IsDefault); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.sellerRepo.Save(ctx, seller); err != nil {
		return nil, fmt.Errorf("failed to save seller: %w", err)
	}

	s.logger.Info("Facility assigned to seller", "sellerId", seller.SellerID, "facilityId", cmd.FacilityID)

	return ToSellerDTO(seller), nil
}

// RemoveFacility removes a facility from a seller
func (s *SellerApplicationService) RemoveFacility(ctx context.Context, cmd RemoveFacilityCommand) (*SellerDTO, error) {
	seller, err := s.sellerRepo.FindByID(ctx, cmd.SellerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get seller: %w", err)
	}
	if seller == nil {
		return nil, errors.ErrNotFound("seller not found")
	}

	if err := seller.RemoveFacility(cmd.FacilityID); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.sellerRepo.Save(ctx, seller); err != nil {
		return nil, fmt.Errorf("failed to save seller: %w", err)
	}

	s.logger.Info("Facility removed from seller", "sellerId", seller.SellerID, "facilityId", cmd.FacilityID)

	return ToSellerDTO(seller), nil
}

// UpdateFeeSchedule updates a seller's fee schedule
func (s *SellerApplicationService) UpdateFeeSchedule(ctx context.Context, cmd UpdateFeeScheduleCommand) (*SellerDTO, error) {
	seller, err := s.sellerRepo.FindByID(ctx, cmd.SellerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get seller: %w", err)
	}
	if seller == nil {
		return nil, errors.ErrNotFound("seller not found")
	}

	seller.UpdateFeeSchedule(cmd.ToDomainFeeSchedule())

	if err := s.sellerRepo.Save(ctx, seller); err != nil {
		return nil, fmt.Errorf("failed to save seller: %w", err)
	}

	s.logger.Info("Fee schedule updated", "sellerId", seller.SellerID)

	return ToSellerDTO(seller), nil
}

// ConnectChannel connects a sales channel to a seller
func (s *SellerApplicationService) ConnectChannel(ctx context.Context, cmd ConnectChannelCommand) (*SellerDTO, error) {
	seller, err := s.sellerRepo.FindByID(ctx, cmd.SellerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get seller: %w", err)
	}
	if seller == nil {
		return nil, errors.ErrNotFound("seller not found")
	}

	if err := seller.AddChannelIntegration(
		cmd.ChannelType,
		cmd.StoreName,
		cmd.StoreURL,
		cmd.Credentials,
		cmd.ToDomainSyncSettings(),
	); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.sellerRepo.Save(ctx, seller); err != nil {
		return nil, fmt.Errorf("failed to save seller: %w", err)
	}

	s.logger.Info("Channel connected", "sellerId", seller.SellerID, "channelType", cmd.ChannelType)

	return ToSellerDTO(seller), nil
}

// DisconnectChannel disconnects a sales channel from a seller
func (s *SellerApplicationService) DisconnectChannel(ctx context.Context, cmd DisconnectChannelCommand) (*SellerDTO, error) {
	seller, err := s.sellerRepo.FindByID(ctx, cmd.SellerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get seller: %w", err)
	}
	if seller == nil {
		return nil, errors.ErrNotFound("seller not found")
	}

	if err := seller.DisconnectChannel(cmd.ChannelID); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.sellerRepo.Save(ctx, seller); err != nil {
		return nil, fmt.Errorf("failed to save seller: %w", err)
	}

	s.logger.Info("Channel disconnected", "sellerId", seller.SellerID, "channelId", cmd.ChannelID)

	return ToSellerDTO(seller), nil
}

// GenerateAPIKey generates a new API key for a seller
func (s *SellerApplicationService) GenerateAPIKey(ctx context.Context, cmd GenerateAPIKeyCommand) (*APIKeyCreatedDTO, error) {
	seller, err := s.sellerRepo.FindByID(ctx, cmd.SellerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get seller: %w", err)
	}
	if seller == nil {
		return nil, errors.ErrNotFound("seller not found")
	}

	apiKey, rawKey, err := seller.GenerateAPIKey(cmd.Name, cmd.Scopes, cmd.ExpiresAt)
	if err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.sellerRepo.Save(ctx, seller); err != nil {
		return nil, fmt.Errorf("failed to save seller: %w", err)
	}

	s.logger.Info("API key generated", "sellerId", seller.SellerID, "keyId", apiKey.KeyID)

	return &APIKeyCreatedDTO{
		KeyID:     apiKey.KeyID,
		Name:      apiKey.Name,
		RawKey:    rawKey, // Only returned at creation
		Scopes:    apiKey.Scopes,
		ExpiresAt: apiKey.ExpiresAt,
		CreatedAt: apiKey.CreatedAt,
	}, nil
}

// RevokeAPIKey revokes an API key
func (s *SellerApplicationService) RevokeAPIKey(ctx context.Context, cmd RevokeAPIKeyCommand) error {
	seller, err := s.sellerRepo.FindByID(ctx, cmd.SellerID)
	if err != nil {
		return fmt.Errorf("failed to get seller: %w", err)
	}
	if seller == nil {
		return errors.ErrNotFound("seller not found")
	}

	if err := seller.RevokeAPIKey(cmd.KeyID); err != nil {
		return errors.ErrValidation(err.Error())
	}

	if err := s.sellerRepo.Save(ctx, seller); err != nil {
		return fmt.Errorf("failed to save seller: %w", err)
	}

	s.logger.Info("API key revoked", "sellerId", seller.SellerID, "keyId", cmd.KeyID)

	return nil
}

// GetAPIKeys returns all API keys for a seller
func (s *SellerApplicationService) GetAPIKeys(ctx context.Context, sellerID string) ([]APIKeyDTO, error) {
	seller, err := s.sellerRepo.FindByID(ctx, sellerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get seller: %w", err)
	}
	if seller == nil {
		return nil, errors.ErrNotFound("seller not found")
	}

	return ToAPIKeyDTOs(seller.GetActiveAPIKeys()), nil
}

// SearchSellers searches sellers by company name or email
func (s *SellerApplicationService) SearchSellers(ctx context.Context, query SearchSellersQuery) (*SellerListResponse, error) {
	pagination := domain.Pagination{
		Page:     query.Page,
		PageSize: query.PageSize,
	}

	if pagination.Page < 1 {
		pagination.Page = 1
	}
	if pagination.PageSize < 1 || pagination.PageSize > 100 {
		pagination.PageSize = 20
	}

	sellers, err := s.sellerRepo.Search(ctx, query.Query, pagination)
	if err != nil {
		return nil, fmt.Errorf("failed to search sellers: %w", err)
	}

	// Convert to DTOs
	dtos := make([]SellerDTO, len(sellers))
	for i, seller := range sellers {
		dtos[i] = *ToSellerDTO(seller)
	}

	return &SellerListResponse{
		Data:       dtos,
		Total:      int64(len(dtos)), // Search doesn't return total count efficiently
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: 1, // Search is estimated
	}, nil
}
