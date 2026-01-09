package application

import (
	"time"

	"github.com/wms-platform/services/billing-service/internal/domain"
)

// RecordActivityCommand represents command to record a billable activity
type RecordActivityCommand struct {
	TenantID      string                 `json:"tenantId" binding:"required"`
	SellerID      string                 `json:"sellerId" binding:"required"`
	FacilityID    string                 `json:"facilityId" binding:"required"`
	Type          string                 `json:"type" binding:"required"`
	Description   string                 `json:"description" binding:"required"`
	Quantity      float64                `json:"quantity" binding:"required,gt=0"`
	UnitPrice     float64                `json:"unitPrice" binding:"required,gte=0"`
	ReferenceType string                 `json:"referenceType" binding:"required"`
	ReferenceID   string                 `json:"referenceId" binding:"required"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// RecordActivitiesCommand represents command to record multiple activities
type RecordActivitiesCommand struct {
	Activities []RecordActivityCommand `json:"activities" binding:"required,min=1"`
}

// ListActivitiesQuery represents query to list activities
type ListActivitiesQuery struct {
	SellerID string
	Page     int64
	PageSize int64
}

// CreateInvoiceCommand represents command to create an invoice
type CreateInvoiceCommand struct {
	TenantID    string    `json:"tenantId" binding:"required"`
	SellerID    string    `json:"sellerId" binding:"required"`
	PeriodStart time.Time `json:"periodStart" binding:"required"`
	PeriodEnd   time.Time `json:"periodEnd" binding:"required"`
	SellerName  string    `json:"sellerName" binding:"required"`
	SellerEmail string    `json:"sellerEmail" binding:"required,email"`
	TaxRate     float64   `json:"taxRate"`
}

// ListInvoicesQuery represents query to list invoices
type ListInvoicesQuery struct {
	SellerID string
	Status   *string
	Page     int64
	PageSize int64
}

// MarkPaidCommand represents command to mark invoice as paid
type MarkPaidCommand struct {
	InvoiceID     string `json:"invoiceId" binding:"required"`
	PaymentMethod string `json:"paymentMethod" binding:"required"`
	PaymentRef    string `json:"paymentRef"`
}

// CalculateFeesCommand represents command to calculate fees
type CalculateFeesCommand struct {
	TenantID         string          `json:"tenantId" binding:"required"`
	SellerID         string          `json:"sellerId" binding:"required"`
	FacilityID       string          `json:"facilityId" binding:"required"`
	FeeSchedule      FeeScheduleDTO  `json:"feeSchedule" binding:"required"`
	StorageCubicFeet float64         `json:"storageCubicFeet"`
	UnitsPicked      int             `json:"unitsPicked"`
	OrdersPacked     int             `json:"ordersPacked"`
	UnitsReceived    int             `json:"unitsReceived"`
	ShippingBaseCost float64         `json:"shippingBaseCost"`
	ReturnsProcessed int             `json:"returnsProcessed"`
	GiftWrapItems    int             `json:"giftWrapItems"`
	HazmatUnits      int             `json:"hazmatUnits"`
	OversizedItems   int             `json:"oversizedItems"`
	ColdChainUnits   int             `json:"coldChainUnits"`
	FragileItems     int             `json:"fragileItems"`
}

// RecordStorageCommand represents command to record storage calculation
type RecordStorageCommand struct {
	TenantID        string    `json:"tenantId" binding:"required"`
	SellerID        string    `json:"sellerId" binding:"required"`
	FacilityID      string    `json:"facilityId" binding:"required"`
	CalculationDate time.Time `json:"calculationDate" binding:"required"`
	TotalCubicFeet  float64   `json:"totalCubicFeet" binding:"required,gte=0"`
	RatePerCubicFt  float64   `json:"ratePerCubicFt" binding:"required,gte=0"`
}

// DTOs

// ActivityDTO represents a billable activity response
type ActivityDTO struct {
	ActivityID    string                 `json:"activityId"`
	TenantID      string                 `json:"tenantId"`
	SellerID      string                 `json:"sellerId"`
	FacilityID    string                 `json:"facilityId"`
	Type          string                 `json:"type"`
	Description   string                 `json:"description"`
	Quantity      float64                `json:"quantity"`
	UnitPrice     float64                `json:"unitPrice"`
	Amount        float64                `json:"amount"`
	Currency      string                 `json:"currency"`
	ReferenceType string                 `json:"referenceType"`
	ReferenceID   string                 `json:"referenceId"`
	ActivityDate  time.Time              `json:"activityDate"`
	Invoiced      bool                   `json:"invoiced"`
	InvoiceID     *string                `json:"invoiceId,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     time.Time              `json:"createdAt"`
}

// ActivityListResponse represents paginated activities
type ActivityListResponse struct {
	Data     []ActivityDTO `json:"data"`
	Page     int64         `json:"page"`
	PageSize int64         `json:"pageSize"`
}

// ActivitySummaryDTO represents activity summary
type ActivitySummaryDTO struct {
	SellerID    string             `json:"sellerId"`
	PeriodStart time.Time          `json:"periodStart"`
	PeriodEnd   time.Time          `json:"periodEnd"`
	ByType      map[string]float64 `json:"byType"`
	Total       float64            `json:"total"`
}

// InvoiceDTO represents an invoice response
type InvoiceDTO struct {
	InvoiceID     string               `json:"invoiceId"`
	TenantID      string               `json:"tenantId"`
	SellerID      string               `json:"sellerId"`
	Status        string               `json:"status"`
	InvoiceNumber string               `json:"invoiceNumber"`
	PeriodStart   time.Time            `json:"periodStart"`
	PeriodEnd     time.Time            `json:"periodEnd"`
	LineItems     []InvoiceLineItemDTO `json:"lineItems"`
	Subtotal      float64              `json:"subtotal"`
	TaxRate       float64              `json:"taxRate"`
	TaxAmount     float64              `json:"taxAmount"`
	Discount      float64              `json:"discount"`
	Total         float64              `json:"total"`
	Currency      string               `json:"currency"`
	DueDate       time.Time            `json:"dueDate"`
	PaidAt        *time.Time           `json:"paidAt,omitempty"`
	PaymentMethod string               `json:"paymentMethod,omitempty"`
	SellerName    string               `json:"sellerName"`
	SellerEmail   string               `json:"sellerEmail"`
	Notes         string               `json:"notes,omitempty"`
	CreatedAt     time.Time            `json:"createdAt"`
	FinalizedAt   *time.Time           `json:"finalizedAt,omitempty"`
}

// InvoiceLineItemDTO represents a line item
type InvoiceLineItemDTO struct {
	ActivityType string  `json:"activityType"`
	Description  string  `json:"description"`
	Quantity     float64 `json:"quantity"`
	UnitPrice    float64 `json:"unitPrice"`
	Amount       float64 `json:"amount"`
}

// InvoiceListResponse represents paginated invoices
type InvoiceListResponse struct {
	Data     []InvoiceDTO `json:"data"`
	Page     int64        `json:"page"`
	PageSize int64        `json:"pageSize"`
}

// FeeScheduleDTO represents fee schedule input
type FeeScheduleDTO struct {
	StorageFeePerCubicFtPerDay float64 `json:"storageFeePerCubicFtPerDay"`
	PickFeePerUnit             float64 `json:"pickFeePerUnit"`
	PackFeePerOrder            float64 `json:"packFeePerOrder"`
	ReceivingFeePerUnit        float64 `json:"receivingFeePerUnit"`
	ShippingMarkupPercent      float64 `json:"shippingMarkupPercent"`
	ReturnProcessingFee        float64 `json:"returnProcessingFee"`
	GiftWrapFee                float64 `json:"giftWrapFee"`
	HazmatHandlingFee          float64 `json:"hazmatHandlingFee"`
	OversizedItemFee           float64 `json:"oversizedItemFee"`
	ColdChainFeePerUnit        float64 `json:"coldChainFeePerUnit"`
	FragileHandlingFee         float64 `json:"fragileHandlingFee"`
}

// FeeCalculationResultDTO represents fee calculation result
type FeeCalculationResultDTO struct {
	StorageFee          float64 `json:"storageFee"`
	PickFee             float64 `json:"pickFee"`
	PackFee             float64 `json:"packFee"`
	ReceivingFee        float64 `json:"receivingFee"`
	ShippingFee         float64 `json:"shippingFee"`
	ReturnProcessingFee float64 `json:"returnProcessingFee"`
	GiftWrapFee         float64 `json:"giftWrapFee"`
	HazmatFee           float64 `json:"hazmatFee"`
	OversizedFee        float64 `json:"oversizedFee"`
	ColdChainFee        float64 `json:"coldChainFee"`
	FragileFee          float64 `json:"fragileFee"`
	TotalFees           float64 `json:"totalFees"`
}

// Conversion functions

// ToActivityDTO converts domain activity to DTO
func ToActivityDTO(a *domain.BillableActivity) *ActivityDTO {
	return &ActivityDTO{
		ActivityID:    a.ActivityID,
		TenantID:      a.TenantID,
		SellerID:      a.SellerID,
		FacilityID:    a.FacilityID,
		Type:          string(a.Type),
		Description:   a.Description,
		Quantity:      a.Quantity,
		UnitPrice:     a.UnitPrice,
		Amount:        a.Amount,
		Currency:      a.Currency,
		ReferenceType: a.ReferenceType,
		ReferenceID:   a.ReferenceID,
		ActivityDate:  a.ActivityDate,
		Invoiced:      a.Invoiced,
		InvoiceID:     a.InvoiceID,
		Metadata:      a.Metadata,
		CreatedAt:     a.CreatedAt,
	}
}

// ToInvoiceDTO converts domain invoice to DTO
func ToInvoiceDTO(inv *domain.Invoice) *InvoiceDTO {
	lineItems := make([]InvoiceLineItemDTO, len(inv.LineItems))
	for i, li := range inv.LineItems {
		lineItems[i] = InvoiceLineItemDTO{
			ActivityType: string(li.ActivityType),
			Description:  li.Description,
			Quantity:     li.Quantity,
			UnitPrice:    li.UnitPrice,
			Amount:       li.Amount,
		}
	}

	return &InvoiceDTO{
		InvoiceID:     inv.InvoiceID,
		TenantID:      inv.TenantID,
		SellerID:      inv.SellerID,
		Status:        string(inv.Status),
		InvoiceNumber: inv.InvoiceNumber,
		PeriodStart:   inv.PeriodStart,
		PeriodEnd:     inv.PeriodEnd,
		LineItems:     lineItems,
		Subtotal:      inv.Subtotal,
		TaxRate:       inv.TaxRate,
		TaxAmount:     inv.TaxAmount,
		Discount:      inv.Discount,
		Total:         inv.Total,
		Currency:      inv.Currency,
		DueDate:       inv.DueDate,
		PaidAt:        inv.PaidAt,
		PaymentMethod: inv.PaymentMethod,
		SellerName:    inv.SellerName,
		SellerEmail:   inv.SellerEmail,
		Notes:         inv.Notes,
		CreatedAt:     inv.CreatedAt,
		FinalizedAt:   inv.FinalizedAt,
	}
}
