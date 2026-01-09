package application

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/services/billing-service/internal/domain"
	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/logging"
)

// BillingService handles billing-related use cases
type BillingService struct {
	activityRepo  domain.BillableActivityRepository
	invoiceRepo   domain.InvoiceRepository
	storageRepo   domain.StorageCalculationRepository
	logger        *logging.Logger
}

// NewBillingService creates a new BillingService
func NewBillingService(
	activityRepo domain.BillableActivityRepository,
	invoiceRepo domain.InvoiceRepository,
	storageRepo domain.StorageCalculationRepository,
	logger *logging.Logger,
) *BillingService {
	return &BillingService{
		activityRepo:  activityRepo,
		invoiceRepo:   invoiceRepo,
		storageRepo:   storageRepo,
		logger:        logger,
	}
}

// RecordActivity records a billable activity
func (s *BillingService) RecordActivity(ctx context.Context, cmd RecordActivityCommand) (*ActivityDTO, error) {
	activityType := domain.ActivityType(cmd.Type)
	if !activityType.IsValid() {
		return nil, errors.ErrValidation("invalid activity type")
	}

	activity, err := domain.NewBillableActivity(
		cmd.TenantID,
		cmd.SellerID,
		cmd.FacilityID,
		activityType,
		cmd.Description,
		cmd.Quantity,
		cmd.UnitPrice,
		cmd.ReferenceType,
		cmd.ReferenceID,
	)
	if err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if cmd.Metadata != nil {
		activity.Metadata = cmd.Metadata
	}

	if err := s.activityRepo.Save(ctx, activity); err != nil {
		s.logger.WithError(err).Error("Failed to save activity", "activityId", activity.ActivityID)
		return nil, fmt.Errorf("failed to save activity: %w", err)
	}

	s.logger.Info("Activity recorded",
		"activityId", activity.ActivityID,
		"sellerId", cmd.SellerID,
		"type", cmd.Type,
		"amount", activity.Amount,
	)

	return ToActivityDTO(activity), nil
}

// RecordActivities records multiple billable activities
func (s *BillingService) RecordActivities(ctx context.Context, cmd RecordActivitiesCommand) ([]ActivityDTO, error) {
	var activities []*domain.BillableActivity

	for _, actCmd := range cmd.Activities {
		activityType := domain.ActivityType(actCmd.Type)
		if !activityType.IsValid() {
			return nil, errors.ErrValidation(fmt.Sprintf("invalid activity type: %s", actCmd.Type))
		}

		activity, err := domain.NewBillableActivity(
			actCmd.TenantID,
			actCmd.SellerID,
			actCmd.FacilityID,
			activityType,
			actCmd.Description,
			actCmd.Quantity,
			actCmd.UnitPrice,
			actCmd.ReferenceType,
			actCmd.ReferenceID,
		)
		if err != nil {
			return nil, errors.ErrValidation(err.Error())
		}

		if actCmd.Metadata != nil {
			activity.Metadata = actCmd.Metadata
		}

		activities = append(activities, activity)
	}

	if err := s.activityRepo.SaveAll(ctx, activities); err != nil {
		s.logger.WithError(err).Error("Failed to save activities")
		return nil, fmt.Errorf("failed to save activities: %w", err)
	}

	dtos := make([]ActivityDTO, len(activities))
	for i, a := range activities {
		dtos[i] = *ToActivityDTO(a)
	}

	s.logger.Info("Activities recorded", "count", len(activities))
	return dtos, nil
}

// GetActivity retrieves an activity by ID
func (s *BillingService) GetActivity(ctx context.Context, activityID string) (*ActivityDTO, error) {
	activity, err := s.activityRepo.FindByID(ctx, activityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity: %w", err)
	}
	if activity == nil {
		return nil, errors.ErrNotFound("activity not found")
	}
	return ToActivityDTO(activity), nil
}

// ListActivities lists activities for a seller
func (s *BillingService) ListActivities(ctx context.Context, query ListActivitiesQuery) (*ActivityListResponse, error) {
	pagination := domain.Pagination{
		Page:     query.Page,
		PageSize: query.PageSize,
	}

	activities, err := s.activityRepo.FindBySellerID(ctx, query.SellerID, pagination)
	if err != nil {
		return nil, fmt.Errorf("failed to list activities: %w", err)
	}

	dtos := make([]ActivityDTO, len(activities))
	for i, a := range activities {
		dtos[i] = *ToActivityDTO(a)
	}

	return &ActivityListResponse{
		Data:     dtos,
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
	}, nil
}

// GetActivitySummary returns a summary of activities by type for a seller
func (s *BillingService) GetActivitySummary(ctx context.Context, sellerID string, periodStart, periodEnd time.Time) (*ActivitySummaryDTO, error) {
	sums, err := s.activityRepo.SumBySellerAndType(ctx, sellerID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity summary: %w", err)
	}

	summary := &ActivitySummaryDTO{
		SellerID:    sellerID,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		ByType:      make(map[string]float64),
		Total:       0,
	}

	for actType, amount := range sums {
		summary.ByType[string(actType)] = amount
		summary.Total += amount
	}

	return summary, nil
}

// CreateInvoice creates a new invoice for a billing period
func (s *BillingService) CreateInvoice(ctx context.Context, cmd CreateInvoiceCommand) (*InvoiceDTO, error) {
	// Check if invoice already exists for this period
	existing, err := s.invoiceRepo.FindByPeriod(ctx, cmd.SellerID, cmd.PeriodStart, cmd.PeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing invoice: %w", err)
	}
	if existing != nil {
		return nil, errors.ErrConflict("invoice already exists for this period")
	}

	// Create the invoice
	invoice := domain.NewInvoice(
		cmd.TenantID,
		cmd.SellerID,
		cmd.PeriodStart,
		cmd.PeriodEnd,
		cmd.SellerName,
		cmd.SellerEmail,
	)

	// Get uninvoiced activities for the period
	activities, err := s.activityRepo.FindUninvoiced(ctx, cmd.SellerID, cmd.PeriodStart, cmd.PeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get uninvoiced activities: %w", err)
	}

	// Group activities by type and add to invoice
	activityGroups := make(map[domain.ActivityType][]*domain.BillableActivity)
	for _, a := range activities {
		activityGroups[a.Type] = append(activityGroups[a.Type], a)
	}

	for actType, groupActivities := range activityGroups {
		var totalQty, totalAmount float64
		var activityIDs []string
		var avgUnitPrice float64

		for _, a := range groupActivities {
			totalQty += a.Quantity
			totalAmount += a.Amount
			activityIDs = append(activityIDs, a.ActivityID)
		}

		if totalQty > 0 {
			avgUnitPrice = totalAmount / totalQty
		}

		description := getActivityDescription(actType)
		if err := invoice.AddLineItem(actType, description, totalQty, avgUnitPrice, activityIDs); err != nil {
			return nil, fmt.Errorf("failed to add line item: %w", err)
		}
	}

	// Apply tax if specified
	if cmd.TaxRate > 0 {
		invoice.SetTaxRate(cmd.TaxRate)
	}

	// Save invoice
	if err := s.invoiceRepo.Save(ctx, invoice); err != nil {
		s.logger.WithError(err).Error("Failed to save invoice", "invoiceId", invoice.InvoiceID)
		return nil, fmt.Errorf("failed to save invoice: %w", err)
	}

	// Mark activities as invoiced
	var allActivityIDs []string
	for _, items := range activityGroups {
		for _, a := range items {
			allActivityIDs = append(allActivityIDs, a.ActivityID)
		}
	}
	if len(allActivityIDs) > 0 {
		if err := s.activityRepo.MarkAsInvoiced(ctx, allActivityIDs, invoice.InvoiceID); err != nil {
			s.logger.WithError(err).Warn("Failed to mark activities as invoiced", "invoiceId", invoice.InvoiceID)
		}
	}

	s.logger.Info("Invoice created",
		"invoiceId", invoice.InvoiceID,
		"sellerId", cmd.SellerID,
		"total", invoice.Total,
		"lineItems", len(invoice.LineItems),
	)

	return ToInvoiceDTO(invoice), nil
}

// GetInvoice retrieves an invoice by ID
func (s *BillingService) GetInvoice(ctx context.Context, invoiceID string) (*InvoiceDTO, error) {
	invoice, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}
	if invoice == nil {
		return nil, errors.ErrNotFound("invoice not found")
	}
	return ToInvoiceDTO(invoice), nil
}

// ListInvoices lists invoices for a seller
func (s *BillingService) ListInvoices(ctx context.Context, query ListInvoicesQuery) (*InvoiceListResponse, error) {
	pagination := domain.Pagination{
		Page:     query.Page,
		PageSize: query.PageSize,
	}

	var invoices []*domain.Invoice
	var err error

	if query.Status != nil {
		invoices, err = s.invoiceRepo.FindByStatus(ctx, domain.InvoiceStatus(*query.Status), pagination)
	} else {
		invoices, err = s.invoiceRepo.FindBySellerID(ctx, query.SellerID, pagination)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list invoices: %w", err)
	}

	dtos := make([]InvoiceDTO, len(invoices))
	for i, inv := range invoices {
		dtos[i] = *ToInvoiceDTO(inv)
	}

	return &InvoiceListResponse{
		Data:     dtos,
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
	}, nil
}

// FinalizeInvoice finalizes an invoice
func (s *BillingService) FinalizeInvoice(ctx context.Context, invoiceID string) (*InvoiceDTO, error) {
	invoice, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}
	if invoice == nil {
		return nil, errors.ErrNotFound("invoice not found")
	}

	if err := invoice.Finalize(); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.invoiceRepo.Save(ctx, invoice); err != nil {
		return nil, fmt.Errorf("failed to save invoice: %w", err)
	}

	s.logger.Info("Invoice finalized", "invoiceId", invoiceID, "total", invoice.Total)

	return ToInvoiceDTO(invoice), nil
}

// MarkInvoicePaid marks an invoice as paid
func (s *BillingService) MarkInvoicePaid(ctx context.Context, cmd MarkPaidCommand) (*InvoiceDTO, error) {
	invoice, err := s.invoiceRepo.FindByID(ctx, cmd.InvoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}
	if invoice == nil {
		return nil, errors.ErrNotFound("invoice not found")
	}

	if err := invoice.MarkPaid(cmd.PaymentMethod, cmd.PaymentRef); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.invoiceRepo.Save(ctx, invoice); err != nil {
		return nil, fmt.Errorf("failed to save invoice: %w", err)
	}

	s.logger.Info("Invoice marked as paid",
		"invoiceId", cmd.InvoiceID,
		"paymentMethod", cmd.PaymentMethod,
	)

	return ToInvoiceDTO(invoice), nil
}

// VoidInvoice voids an invoice
func (s *BillingService) VoidInvoice(ctx context.Context, invoiceID, reason string) (*InvoiceDTO, error) {
	invoice, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}
	if invoice == nil {
		return nil, errors.ErrNotFound("invoice not found")
	}

	if err := invoice.Void(reason); err != nil {
		return nil, errors.ErrValidation(err.Error())
	}

	if err := s.invoiceRepo.Save(ctx, invoice); err != nil {
		return nil, fmt.Errorf("failed to save invoice: %w", err)
	}

	s.logger.Info("Invoice voided", "invoiceId", invoiceID, "reason", reason)

	return ToInvoiceDTO(invoice), nil
}

// CalculateFees calculates fees for a request (preview without saving)
func (s *BillingService) CalculateFees(ctx context.Context, cmd CalculateFeesCommand) (*FeeCalculationResultDTO, error) {
	schedule := &domain.FeeSchedule{
		StorageFeePerCubicFtPerDay: cmd.FeeSchedule.StorageFeePerCubicFtPerDay,
		PickFeePerUnit:             cmd.FeeSchedule.PickFeePerUnit,
		PackFeePerOrder:            cmd.FeeSchedule.PackFeePerOrder,
		ReceivingFeePerUnit:        cmd.FeeSchedule.ReceivingFeePerUnit,
		ShippingMarkupPercent:      cmd.FeeSchedule.ShippingMarkupPercent,
		ReturnProcessingFee:        cmd.FeeSchedule.ReturnProcessingFee,
		GiftWrapFee:                cmd.FeeSchedule.GiftWrapFee,
		HazmatHandlingFee:          cmd.FeeSchedule.HazmatHandlingFee,
		OversizedItemFee:           cmd.FeeSchedule.OversizedItemFee,
		ColdChainFeePerUnit:        cmd.FeeSchedule.ColdChainFeePerUnit,
		FragileHandlingFee:         cmd.FeeSchedule.FragileHandlingFee,
	}

	calculator := domain.NewFeeCalculator(schedule)

	req := domain.FeeCalculationRequest{
		TenantID:         cmd.TenantID,
		SellerID:         cmd.SellerID,
		FacilityID:       cmd.FacilityID,
		StorageCubicFeet: cmd.StorageCubicFeet,
		UnitsPicked:      cmd.UnitsPicked,
		OrdersPacked:     cmd.OrdersPacked,
		UnitsReceived:    cmd.UnitsReceived,
		ShippingBaseCost: cmd.ShippingBaseCost,
		ReturnsProcessed: cmd.ReturnsProcessed,
		GiftWrapItems:    cmd.GiftWrapItems,
		HazmatUnits:      cmd.HazmatUnits,
		OversizedItems:   cmd.OversizedItems,
		ColdChainUnits:   cmd.ColdChainUnits,
		FragileItems:     cmd.FragileItems,
	}

	result := calculator.CalculateAllFees(req)

	return &FeeCalculationResultDTO{
		StorageFee:          result.StorageFee,
		PickFee:             result.PickFee,
		PackFee:             result.PackFee,
		ReceivingFee:        result.ReceivingFee,
		ShippingFee:         result.ShippingFee,
		ReturnProcessingFee: result.ReturnProcessingFee,
		GiftWrapFee:         result.GiftWrapFee,
		HazmatFee:           result.HazmatFee,
		OversizedFee:        result.OversizedFee,
		ColdChainFee:        result.ColdChainFee,
		FragileFee:          result.FragileFee,
		TotalFees:           result.TotalFees,
	}, nil
}

// RecordStorageCalculation records daily storage calculation
func (s *BillingService) RecordStorageCalculation(ctx context.Context, cmd RecordStorageCommand) error {
	calc := domain.NewStorageCalculation(
		cmd.TenantID,
		cmd.SellerID,
		cmd.FacilityID,
		cmd.CalculationDate,
		cmd.TotalCubicFeet,
		cmd.RatePerCubicFt,
	)

	if err := s.storageRepo.Save(ctx, calc); err != nil {
		s.logger.WithError(err).Error("Failed to save storage calculation")
		return fmt.Errorf("failed to save storage calculation: %w", err)
	}

	// Also record as billable activity
	activity, _ := domain.NewBillableActivity(
		cmd.TenantID,
		cmd.SellerID,
		cmd.FacilityID,
		domain.ActivityTypeStorage,
		"Daily storage fee",
		cmd.TotalCubicFeet,
		cmd.RatePerCubicFt,
		"storage_calculation",
		calc.CalculationID,
	)
	if activity != nil {
		if err := s.activityRepo.Save(ctx, activity); err != nil {
			s.logger.WithError(err).Warn("Failed to save storage activity")
		}
	}

	s.logger.Info("Storage calculation recorded",
		"sellerId", cmd.SellerID,
		"cubicFeet", cmd.TotalCubicFeet,
		"amount", calc.TotalAmount,
	)

	return nil
}

// CheckOverdueInvoices marks overdue invoices
func (s *BillingService) CheckOverdueInvoices(ctx context.Context) (int, error) {
	overdueInvoices, err := s.invoiceRepo.FindOverdue(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to find overdue invoices: %w", err)
	}

	count := 0
	for _, inv := range overdueInvoices {
		inv.MarkOverdue()
		if err := s.invoiceRepo.Save(ctx, inv); err != nil {
			s.logger.WithError(err).Warn("Failed to mark invoice as overdue", "invoiceId", inv.InvoiceID)
			continue
		}
		count++
	}

	if count > 0 {
		s.logger.Info("Marked invoices as overdue", "count", count)
	}

	return count, nil
}

// Helper function
func getActivityDescription(actType domain.ActivityType) string {
	descriptions := map[domain.ActivityType]string{
		domain.ActivityTypeStorage:          "Storage fees",
		domain.ActivityTypePick:             "Picking fees",
		domain.ActivityTypePack:             "Packing fees",
		domain.ActivityTypeReceiving:        "Receiving fees",
		domain.ActivityTypeShipping:         "Shipping fees",
		domain.ActivityTypeReturnProcessing: "Return processing fees",
		domain.ActivityTypeGiftWrap:         "Gift wrap fees",
		domain.ActivityTypeHazmat:           "Hazmat handling fees",
		domain.ActivityTypeOversized:        "Oversized item fees",
		domain.ActivityTypeColdChain:        "Cold chain fees",
		domain.ActivityTypeFragile:          "Fragile handling fees",
		domain.ActivityTypeSpecialHandling:  "Special handling fees",
	}
	if desc, ok := descriptions[actType]; ok {
		return desc
	}
	return string(actType)
}
