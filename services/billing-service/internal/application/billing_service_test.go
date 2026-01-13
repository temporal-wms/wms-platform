package application

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wms-platform/services/billing-service/internal/domain"
	"github.com/wms-platform/shared/pkg/logging"
	sharedErrors "github.com/wms-platform/shared/pkg/errors"
)

type fakeActivityRepo struct {
	saveFn             func(context.Context, *domain.BillableActivity) error
	saveAllFn          func(context.Context, []*domain.BillableActivity) error
	findByIDFn         func(context.Context, string) (*domain.BillableActivity, error)
	findBySellerIDFn   func(context.Context, string, domain.Pagination) ([]*domain.BillableActivity, error)
	findUninvoicedFn   func(context.Context, string, time.Time, time.Time) ([]*domain.BillableActivity, error)
	markAsInvoicedFn   func(context.Context, []string, string) error
	sumBySellerAndType func(context.Context, string, time.Time, time.Time) (map[domain.ActivityType]float64, error)
}

func (f *fakeActivityRepo) Save(ctx context.Context, activity *domain.BillableActivity) error {
	if f.saveFn != nil {
		return f.saveFn(ctx, activity)
	}
	return nil
}

func (f *fakeActivityRepo) SaveAll(ctx context.Context, activities []*domain.BillableActivity) error {
	if f.saveAllFn != nil {
		return f.saveAllFn(ctx, activities)
	}
	return nil
}

func (f *fakeActivityRepo) FindByID(ctx context.Context, activityID string) (*domain.BillableActivity, error) {
	if f.findByIDFn != nil {
		return f.findByIDFn(ctx, activityID)
	}
	return nil, nil
}

func (f *fakeActivityRepo) FindBySellerID(ctx context.Context, sellerID string, pagination domain.Pagination) ([]*domain.BillableActivity, error) {
	if f.findBySellerIDFn != nil {
		return f.findBySellerIDFn(ctx, sellerID, pagination)
	}
	return nil, nil
}

func (f *fakeActivityRepo) FindUninvoiced(ctx context.Context, sellerID string, periodStart, periodEnd time.Time) ([]*domain.BillableActivity, error) {
	if f.findUninvoicedFn != nil {
		return f.findUninvoicedFn(ctx, sellerID, periodStart, periodEnd)
	}
	return nil, nil
}

func (f *fakeActivityRepo) FindByInvoiceID(ctx context.Context, invoiceID string) ([]*domain.BillableActivity, error) {
	return nil, nil
}

func (f *fakeActivityRepo) MarkAsInvoiced(ctx context.Context, activityIDs []string, invoiceID string) error {
	if f.markAsInvoicedFn != nil {
		return f.markAsInvoicedFn(ctx, activityIDs, invoiceID)
	}
	return nil
}

func (f *fakeActivityRepo) SumBySellerAndType(ctx context.Context, sellerID string, periodStart, periodEnd time.Time) (map[domain.ActivityType]float64, error) {
	if f.sumBySellerAndType != nil {
		return f.sumBySellerAndType(ctx, sellerID, periodStart, periodEnd)
	}
	return nil, nil
}

func (f *fakeActivityRepo) Count(ctx context.Context, filter domain.ActivityFilter) (int64, error) {
	return 0, nil
}

type fakeInvoiceRepo struct {
	saveFn        func(context.Context, *domain.Invoice) error
	findByIDFn    func(context.Context, string) (*domain.Invoice, error)
	findBySeller  func(context.Context, string, domain.Pagination) ([]*domain.Invoice, error)
	findByStatus  func(context.Context, domain.InvoiceStatus, domain.Pagination) ([]*domain.Invoice, error)
	findOverdueFn func(context.Context) ([]*domain.Invoice, error)
	findByPeriod  func(context.Context, string, time.Time, time.Time) (*domain.Invoice, error)
}

func (f *fakeInvoiceRepo) Save(ctx context.Context, invoice *domain.Invoice) error {
	if f.saveFn != nil {
		return f.saveFn(ctx, invoice)
	}
	return nil
}

func (f *fakeInvoiceRepo) FindByID(ctx context.Context, invoiceID string) (*domain.Invoice, error) {
	if f.findByIDFn != nil {
		return f.findByIDFn(ctx, invoiceID)
	}
	return nil, nil
}

func (f *fakeInvoiceRepo) FindBySellerID(ctx context.Context, sellerID string, pagination domain.Pagination) ([]*domain.Invoice, error) {
	if f.findBySeller != nil {
		return f.findBySeller(ctx, sellerID, pagination)
	}
	return nil, nil
}

func (f *fakeInvoiceRepo) FindByStatus(ctx context.Context, status domain.InvoiceStatus, pagination domain.Pagination) ([]*domain.Invoice, error) {
	if f.findByStatus != nil {
		return f.findByStatus(ctx, status, pagination)
	}
	return nil, nil
}

func (f *fakeInvoiceRepo) FindOverdue(ctx context.Context) ([]*domain.Invoice, error) {
	if f.findOverdueFn != nil {
		return f.findOverdueFn(ctx)
	}
	return nil, nil
}

func (f *fakeInvoiceRepo) FindByPeriod(ctx context.Context, sellerID string, periodStart, periodEnd time.Time) (*domain.Invoice, error) {
	if f.findByPeriod != nil {
		return f.findByPeriod(ctx, sellerID, periodStart, periodEnd)
	}
	return nil, nil
}

func (f *fakeInvoiceRepo) UpdateStatus(ctx context.Context, invoiceID string, status domain.InvoiceStatus) error {
	return nil
}

func (f *fakeInvoiceRepo) Count(ctx context.Context, filter domain.InvoiceFilter) (int64, error) {
	return 0, nil
}

type fakeStorageRepo struct {
	saveFn func(context.Context, *domain.StorageCalculation) error
}

func (f *fakeStorageRepo) Save(ctx context.Context, calc *domain.StorageCalculation) error {
	if f.saveFn != nil {
		return f.saveFn(ctx, calc)
	}
	return nil
}

func (f *fakeStorageRepo) FindBySellerAndDate(ctx context.Context, sellerID string, date time.Time) (*domain.StorageCalculation, error) {
	return nil, nil
}

func (f *fakeStorageRepo) FindBySellerAndPeriod(ctx context.Context, sellerID string, start, end time.Time) ([]*domain.StorageCalculation, error) {
	return nil, nil
}

func (f *fakeStorageRepo) SumByPeriod(ctx context.Context, sellerID string, start, end time.Time) (float64, error) {
	return 0, nil
}

func testLogger() *logging.Logger {
	cfg := logging.DefaultConfig("billing-test")
	cfg.Output = io.Discard
	return logging.New(cfg)
}

func TestRecordActivitySuccess(t *testing.T) {
	var saved *domain.BillableActivity
	activityRepo := &fakeActivityRepo{
		saveFn: func(_ context.Context, activity *domain.BillableActivity) error {
			saved = activity
			return nil
		},
	}

	service := NewBillingService(activityRepo, &fakeInvoiceRepo{}, &fakeStorageRepo{}, testLogger())

	cmd := RecordActivityCommand{
		TenantID:      "TNT-001",
		SellerID:      "SLR-001",
		FacilityID:    "FAC-001",
		Type:          string(domain.ActivityTypePick),
		Description:   "Pick fee",
		Quantity:      10,
		UnitPrice:     0.25,
		ReferenceType: "order",
		ReferenceID:   "ORD-001",
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	dto, err := service.RecordActivity(context.Background(), cmd)
	require.NoError(t, err)
	require.NotNil(t, dto)
	require.NotNil(t, saved)

	assert.Equal(t, saved.ActivityID, dto.ActivityID)
	assert.Equal(t, "USD", dto.Currency)
	assert.Equal(t, 2.5, dto.Amount)
	assert.Equal(t, "test", dto.Metadata["source"])
}

func TestRecordActivityInvalidType(t *testing.T) {
	service := NewBillingService(&fakeActivityRepo{}, &fakeInvoiceRepo{}, &fakeStorageRepo{}, testLogger())

	_, err := service.RecordActivity(context.Background(), RecordActivityCommand{
		TenantID:      "TNT-001",
		SellerID:      "SLR-001",
		FacilityID:    "FAC-001",
		Type:          "invalid",
		Description:   "Bad",
		Quantity:      1,
		UnitPrice:     1,
		ReferenceType: "order",
		ReferenceID:   "ORD-001",
	})

	require.Error(t, err)
	var appErr *sharedErrors.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, sharedErrors.CodeValidationError, appErr.Code)
}

func TestRecordActivityRepoError(t *testing.T) {
	activityRepo := &fakeActivityRepo{
		saveFn: func(_ context.Context, _ *domain.BillableActivity) error {
			return errors.New("db error")
		},
	}

	service := NewBillingService(activityRepo, &fakeInvoiceRepo{}, &fakeStorageRepo{}, testLogger())

	_, err := service.RecordActivity(context.Background(), RecordActivityCommand{
		TenantID:      "TNT-001",
		SellerID:      "SLR-001",
		FacilityID:    "FAC-001",
		Type:          string(domain.ActivityTypePick),
		Description:   "Pick fee",
		Quantity:      10,
		UnitPrice:     0.25,
		ReferenceType: "order",
		ReferenceID:   "ORD-001",
	})

	assert.Error(t, err)
}

func TestRecordActivitiesSuccess(t *testing.T) {
	var saved []*domain.BillableActivity
	activityRepo := &fakeActivityRepo{
		saveAllFn: func(_ context.Context, activities []*domain.BillableActivity) error {
			saved = activities
			return nil
		},
	}

	service := NewBillingService(activityRepo, &fakeInvoiceRepo{}, &fakeStorageRepo{}, testLogger())

	cmd := RecordActivitiesCommand{
		Activities: []RecordActivityCommand{
			{
				TenantID:      "TNT-001",
				SellerID:      "SLR-001",
				FacilityID:    "FAC-001",
				Type:          string(domain.ActivityTypePick),
				Description:   "Pick fee",
				Quantity:      10,
				UnitPrice:     0.25,
				ReferenceType: "order",
				ReferenceID:   "ORD-001",
			},
			{
				TenantID:      "TNT-001",
				SellerID:      "SLR-001",
				FacilityID:    "FAC-001",
				Type:          string(domain.ActivityTypePack),
				Description:   "Pack fee",
				Quantity:      2,
				UnitPrice:     1.5,
				ReferenceType: "order",
				ReferenceID:   "ORD-002",
			},
		},
	}

	dtos, err := service.RecordActivities(context.Background(), cmd)
	require.NoError(t, err)
	require.Len(t, dtos, 2)
	require.Len(t, saved, 2)
	assert.Equal(t, saved[0].ActivityID, dtos[0].ActivityID)
	assert.Equal(t, saved[1].ActivityID, dtos[1].ActivityID)
}

func TestRecordActivitiesInvalidType(t *testing.T) {
	service := NewBillingService(&fakeActivityRepo{}, &fakeInvoiceRepo{}, &fakeStorageRepo{}, testLogger())

	_, err := service.RecordActivities(context.Background(), RecordActivitiesCommand{
		Activities: []RecordActivityCommand{
			{
				TenantID:      "TNT-001",
				SellerID:      "SLR-001",
				FacilityID:    "FAC-001",
				Type:          "invalid",
				Description:   "Bad",
				Quantity:      10,
				UnitPrice:     0.25,
				ReferenceType: "order",
				ReferenceID:   "ORD-001",
			},
		},
	})

	require.Error(t, err)
	var appErr *sharedErrors.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, sharedErrors.CodeValidationError, appErr.Code)
}

func TestGetActivity(t *testing.T) {
	expected := &domain.BillableActivity{
		ActivityID: "ACT-001",
		Type:       domain.ActivityTypePick,
	}
	activityRepo := &fakeActivityRepo{
		findByIDFn: func(_ context.Context, _ string) (*domain.BillableActivity, error) {
			return expected, nil
		},
	}

	service := NewBillingService(activityRepo, &fakeInvoiceRepo{}, &fakeStorageRepo{}, testLogger())

	dto, err := service.GetActivity(context.Background(), "ACT-001")
	require.NoError(t, err)
	require.NotNil(t, dto)
	assert.Equal(t, expected.ActivityID, dto.ActivityID)
}

func TestGetActivityNotFound(t *testing.T) {
	activityRepo := &fakeActivityRepo{
		findByIDFn: func(_ context.Context, _ string) (*domain.BillableActivity, error) {
			return nil, nil
		},
	}

	service := NewBillingService(activityRepo, &fakeInvoiceRepo{}, &fakeStorageRepo{}, testLogger())

	_, err := service.GetActivity(context.Background(), "ACT-404")
	require.Error(t, err)
	var appErr *sharedErrors.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, sharedErrors.CodeNotFound, appErr.Code)
}

func TestGetInvoiceSuccess(t *testing.T) {
	inv := domain.NewInvoice("TNT-001", "SLR-001", time.Now().Add(-24*time.Hour), time.Now(), "Acme", "billing@acme.com")
	invoiceRepo := &fakeInvoiceRepo{
		findByIDFn: func(_ context.Context, _ string) (*domain.Invoice, error) {
			return inv, nil
		},
	}

	service := NewBillingService(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{}, testLogger())
	dto, err := service.GetInvoice(context.Background(), inv.InvoiceID)

	require.NoError(t, err)
	require.NotNil(t, dto)
	assert.Equal(t, inv.InvoiceID, dto.InvoiceID)
}

func TestListInvoicesBySeller(t *testing.T) {
	invoiceRepo := &fakeInvoiceRepo{
		findBySeller: func(_ context.Context, _ string, _ domain.Pagination) ([]*domain.Invoice, error) {
			return []*domain.Invoice{{InvoiceID: "INV-001"}}, nil
		},
	}

	service := NewBillingService(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{}, testLogger())
	result, err := service.ListInvoices(context.Background(), ListInvoicesQuery{
		SellerID: "SLR-001",
		Page:     1,
		PageSize: 20,
	})

	require.NoError(t, err)
	require.Len(t, result.Data, 1)
	assert.Equal(t, "INV-001", result.Data[0].InvoiceID)
}

func TestListInvoicesByStatus(t *testing.T) {
	invoiceRepo := &fakeInvoiceRepo{
		findByStatus: func(_ context.Context, _ domain.InvoiceStatus, _ domain.Pagination) ([]*domain.Invoice, error) {
			return []*domain.Invoice{{InvoiceID: "INV-PAID"}}, nil
		},
	}

	service := NewBillingService(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{}, testLogger())
	status := string(domain.InvoiceStatusPaid)
	result, err := service.ListInvoices(context.Background(), ListInvoicesQuery{
		SellerID: "SLR-001",
		Status:   &status,
		Page:     1,
		PageSize: 20,
	})

	require.NoError(t, err)
	require.Len(t, result.Data, 1)
	assert.Equal(t, "INV-PAID", result.Data[0].InvoiceID)
}

func TestFinalizeInvoiceSuccess(t *testing.T) {
	inv := domain.NewInvoice("TNT-001", "SLR-001", time.Now().Add(-24*time.Hour), time.Now(), "Acme", "billing@acme.com")
	var saved *domain.Invoice

	invoiceRepo := &fakeInvoiceRepo{
		findByIDFn: func(_ context.Context, _ string) (*domain.Invoice, error) {
			return inv, nil
		},
		saveFn: func(_ context.Context, invoice *domain.Invoice) error {
			saved = invoice
			return nil
		},
	}

	service := NewBillingService(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{}, testLogger())
	dto, err := service.FinalizeInvoice(context.Background(), inv.InvoiceID)

	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, domain.InvoiceStatusFinalized, saved.Status)
	assert.Equal(t, inv.InvoiceID, dto.InvoiceID)
}

func TestMarkInvoicePaidSuccess(t *testing.T) {
	inv := domain.NewInvoice("TNT-001", "SLR-001", time.Now().Add(-24*time.Hour), time.Now(), "Acme", "billing@acme.com")
	_ = inv.Finalize()

	var saved *domain.Invoice
	invoiceRepo := &fakeInvoiceRepo{
		findByIDFn: func(_ context.Context, _ string) (*domain.Invoice, error) {
			return inv, nil
		},
		saveFn: func(_ context.Context, invoice *domain.Invoice) error {
			saved = invoice
			return nil
		},
	}

	service := NewBillingService(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{}, testLogger())
	dto, err := service.MarkInvoicePaid(context.Background(), MarkPaidCommand{
		InvoiceID:     inv.InvoiceID,
		PaymentMethod: "card",
		PaymentRef:    "TXN-001",
	})

	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, domain.InvoiceStatusPaid, saved.Status)
	assert.Equal(t, inv.InvoiceID, dto.InvoiceID)
}

func TestVoidInvoiceSuccess(t *testing.T) {
	inv := domain.NewInvoice("TNT-001", "SLR-001", time.Now().Add(-24*time.Hour), time.Now(), "Acme", "billing@acme.com")
	var saved *domain.Invoice

	invoiceRepo := &fakeInvoiceRepo{
		findByIDFn: func(_ context.Context, _ string) (*domain.Invoice, error) {
			return inv, nil
		},
		saveFn: func(_ context.Context, invoice *domain.Invoice) error {
			saved = invoice
			return nil
		},
	}

	service := NewBillingService(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{}, testLogger())
	dto, err := service.VoidInvoice(context.Background(), inv.InvoiceID, "Duplicate")

	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, domain.InvoiceStatusVoided, saved.Status)
	assert.Equal(t, inv.InvoiceID, dto.InvoiceID)
}

func TestListActivities(t *testing.T) {
	activityRepo := &fakeActivityRepo{
		findBySellerIDFn: func(_ context.Context, _ string, _ domain.Pagination) ([]*domain.BillableActivity, error) {
			return []*domain.BillableActivity{
				{ActivityID: "ACT-001", Type: domain.ActivityTypePick},
			}, nil
		},
	}

	service := NewBillingService(activityRepo, &fakeInvoiceRepo{}, &fakeStorageRepo{}, testLogger())

	result, err := service.ListActivities(context.Background(), ListActivitiesQuery{
		SellerID: "SLR-001",
		Page:     2,
		PageSize: 10,
	})
	require.NoError(t, err)
	require.Len(t, result.Data, 1)
	assert.Equal(t, int64(2), result.Page)
	assert.Equal(t, int64(10), result.PageSize)
}

func TestListActivitiesRepoError(t *testing.T) {
	activityRepo := &fakeActivityRepo{
		findBySellerIDFn: func(_ context.Context, _ string, _ domain.Pagination) ([]*domain.BillableActivity, error) {
			return nil, errors.New("repo error")
		},
	}

	service := NewBillingService(activityRepo, &fakeInvoiceRepo{}, &fakeStorageRepo{}, testLogger())

	_, err := service.ListActivities(context.Background(), ListActivitiesQuery{
		SellerID: "SLR-001",
		Page:     1,
		PageSize: 10,
	})
	assert.Error(t, err)
}

func TestGetActivitySummary(t *testing.T) {
	activityRepo := &fakeActivityRepo{
		sumBySellerAndType: func(_ context.Context, _ string, _, _ time.Time) (map[domain.ActivityType]float64, error) {
			return map[domain.ActivityType]float64{
				domain.ActivityTypePick: 10,
				domain.ActivityTypePack: 5,
			}, nil
		},
	}

	service := NewBillingService(activityRepo, &fakeInvoiceRepo{}, &fakeStorageRepo{}, testLogger())
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()

	summary, err := service.GetActivitySummary(context.Background(), "SLR-001", start, end)
	require.NoError(t, err)
	assert.Equal(t, 15.0, summary.Total)
	assert.Equal(t, 10.0, summary.ByType[string(domain.ActivityTypePick)])
}

func TestGetActivitySummaryError(t *testing.T) {
	activityRepo := &fakeActivityRepo{
		sumBySellerAndType: func(_ context.Context, _ string, _, _ time.Time) (map[domain.ActivityType]float64, error) {
			return nil, errors.New("summary error")
		},
	}

	service := NewBillingService(activityRepo, &fakeInvoiceRepo{}, &fakeStorageRepo{}, testLogger())
	_, err := service.GetActivitySummary(context.Background(), "SLR-001", time.Now().Add(-24*time.Hour), time.Now())
	assert.Error(t, err)
}

func TestCreateInvoiceSuccess(t *testing.T) {
	periodStart := time.Now().Add(-30 * 24 * time.Hour)
	periodEnd := time.Now()
	activities := []*domain.BillableActivity{
		{
			ActivityID: "ACT-001",
			Type:       domain.ActivityTypePick,
			Quantity:   10,
			Amount:     5,
		},
		{
			ActivityID: "ACT-002",
			Type:       domain.ActivityTypePick,
			Quantity:   5,
			Amount:     2.5,
		},
		{
			ActivityID: "ACT-003",
			Type:       domain.ActivityTypePack,
			Quantity:   2,
			Amount:     3,
		},
	}

	var savedInvoice *domain.Invoice
	var marked bool

	activityRepo := &fakeActivityRepo{
		findUninvoicedFn: func(_ context.Context, _ string, _, _ time.Time) ([]*domain.BillableActivity, error) {
			return activities, nil
		},
		markAsInvoicedFn: func(_ context.Context, _ []string, _ string) error {
			marked = true
			return nil
		},
	}

	invoiceRepo := &fakeInvoiceRepo{
		findByPeriod: func(_ context.Context, _ string, _, _ time.Time) (*domain.Invoice, error) {
			return nil, nil
		},
		saveFn: func(_ context.Context, inv *domain.Invoice) error {
			savedInvoice = inv
			return nil
		},
	}

	service := NewBillingService(activityRepo, invoiceRepo, &fakeStorageRepo{}, testLogger())

	dto, err := service.CreateInvoice(context.Background(), CreateInvoiceCommand{
		TenantID:    "TNT-001",
		SellerID:    "SLR-001",
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		SellerName:  "Acme Corp",
		SellerEmail: "billing@acme.com",
		TaxRate:     0.1,
	})
	require.NoError(t, err)
	require.NotNil(t, dto)
	require.NotNil(t, savedInvoice)
	assert.Len(t, savedInvoice.LineItems, 2)
	assert.True(t, marked)
	assert.Equal(t, 0.1, savedInvoice.TaxRate)
}

func TestCreateInvoiceFindUninvoicedError(t *testing.T) {
	activityRepo := &fakeActivityRepo{
		findUninvoicedFn: func(_ context.Context, _ string, _, _ time.Time) ([]*domain.BillableActivity, error) {
			return nil, errors.New("query error")
		},
	}

	invoiceRepo := &fakeInvoiceRepo{
		findByPeriod: func(_ context.Context, _ string, _, _ time.Time) (*domain.Invoice, error) {
			return nil, nil
		},
	}

	service := NewBillingService(activityRepo, invoiceRepo, &fakeStorageRepo{}, testLogger())
	_, err := service.CreateInvoice(context.Background(), CreateInvoiceCommand{
		TenantID:    "TNT-001",
		SellerID:    "SLR-001",
		PeriodStart: time.Now().Add(-24 * time.Hour),
		PeriodEnd:   time.Now(),
		SellerName:  "Acme Corp",
		SellerEmail: "billing@acme.com",
	})
	assert.Error(t, err)
}

func TestCreateInvoiceConflict(t *testing.T) {
	invoiceRepo := &fakeInvoiceRepo{
		findByPeriod: func(_ context.Context, _ string, _, _ time.Time) (*domain.Invoice, error) {
			return &domain.Invoice{InvoiceID: "INV-EXISTING"}, nil
		},
	}

	service := NewBillingService(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{}, testLogger())

	_, err := service.CreateInvoice(context.Background(), CreateInvoiceCommand{
		TenantID:    "TNT-001",
		SellerID:    "SLR-001",
		PeriodStart: time.Now().Add(-24 * time.Hour),
		PeriodEnd:   time.Now(),
		SellerName:  "Acme Corp",
		SellerEmail: "billing@acme.com",
	})

	require.Error(t, err)
	var appErr *sharedErrors.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, sharedErrors.CodeConflict, appErr.Code)
}

func TestGetInvoiceNotFound(t *testing.T) {
	invoiceRepo := &fakeInvoiceRepo{
		findByIDFn: func(_ context.Context, _ string) (*domain.Invoice, error) {
			return nil, nil
		},
	}

	service := NewBillingService(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{}, testLogger())
	_, err := service.GetInvoice(context.Background(), "INV-404")
	require.Error(t, err)

	var appErr *sharedErrors.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, sharedErrors.CodeNotFound, appErr.Code)
}

func TestListInvoicesRepoError(t *testing.T) {
	invoiceRepo := &fakeInvoiceRepo{
		findBySeller: func(_ context.Context, _ string, _ domain.Pagination) ([]*domain.Invoice, error) {
			return nil, errors.New("repo error")
		},
	}

	service := NewBillingService(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{}, testLogger())
	_, err := service.ListInvoices(context.Background(), ListInvoicesQuery{
		SellerID: "SLR-001",
		Page:     1,
		PageSize: 20,
	})
	assert.Error(t, err)
}

func TestFinalizeInvoiceValidationError(t *testing.T) {
	inv := domain.NewInvoice("TNT-001", "SLR-001", time.Now().Add(-24*time.Hour), time.Now(), "Acme", "billing@acme.com")
	inv.Status = domain.InvoiceStatusFinalized

	invoiceRepo := &fakeInvoiceRepo{
		findByIDFn: func(_ context.Context, _ string) (*domain.Invoice, error) {
			return inv, nil
		},
	}

	service := NewBillingService(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{}, testLogger())
	_, err := service.FinalizeInvoice(context.Background(), inv.InvoiceID)

	require.Error(t, err)
	var appErr *sharedErrors.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, sharedErrors.CodeValidationError, appErr.Code)
}

func TestFinalizeInvoiceSaveError(t *testing.T) {
	inv := domain.NewInvoice("TNT-001", "SLR-001", time.Now().Add(-24*time.Hour), time.Now(), "Acme", "billing@acme.com")

	invoiceRepo := &fakeInvoiceRepo{
		findByIDFn: func(_ context.Context, _ string) (*domain.Invoice, error) {
			return inv, nil
		},
		saveFn: func(_ context.Context, _ *domain.Invoice) error {
			return errors.New("save error")
		},
	}

	service := NewBillingService(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{}, testLogger())
	_, err := service.FinalizeInvoice(context.Background(), inv.InvoiceID)
	assert.Error(t, err)
}

func TestMarkInvoicePaidNotFound(t *testing.T) {
	service := NewBillingService(&fakeActivityRepo{}, &fakeInvoiceRepo{}, &fakeStorageRepo{}, testLogger())

	_, err := service.MarkInvoicePaid(context.Background(), MarkPaidCommand{
		InvoiceID:     "INV-404",
		PaymentMethod: "card",
	})

	require.Error(t, err)
	var appErr *sharedErrors.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, sharedErrors.CodeNotFound, appErr.Code)
}

func TestMarkInvoicePaidValidationError(t *testing.T) {
	inv := domain.NewInvoice("TNT-001", "SLR-001", time.Now().Add(-24*time.Hour), time.Now(), "Acme", "billing@acme.com")

	invoiceRepo := &fakeInvoiceRepo{
		findByIDFn: func(_ context.Context, _ string) (*domain.Invoice, error) {
			return inv, nil
		},
	}

	service := NewBillingService(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{}, testLogger())
	_, err := service.MarkInvoicePaid(context.Background(), MarkPaidCommand{
		InvoiceID:     inv.InvoiceID,
		PaymentMethod: "card",
	})
	require.Error(t, err)

	var appErr *sharedErrors.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, sharedErrors.CodeValidationError, appErr.Code)
}

func TestVoidInvoicePaidValidation(t *testing.T) {
	inv := domain.NewInvoice("TNT-001", "SLR-001", time.Now().Add(-24*time.Hour), time.Now(), "Acme", "billing@acme.com")
	inv.Status = domain.InvoiceStatusPaid

	invoiceRepo := &fakeInvoiceRepo{
		findByIDFn: func(_ context.Context, _ string) (*domain.Invoice, error) {
			return inv, nil
		},
	}

	service := NewBillingService(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{}, testLogger())
	_, err := service.VoidInvoice(context.Background(), inv.InvoiceID, "Test")

	require.Error(t, err)
	var appErr *sharedErrors.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, sharedErrors.CodeValidationError, appErr.Code)
}

func TestVoidInvoiceSaveError(t *testing.T) {
	inv := domain.NewInvoice("TNT-001", "SLR-001", time.Now().Add(-24*time.Hour), time.Now(), "Acme", "billing@acme.com")

	invoiceRepo := &fakeInvoiceRepo{
		findByIDFn: func(_ context.Context, _ string) (*domain.Invoice, error) {
			return inv, nil
		},
		saveFn: func(_ context.Context, _ *domain.Invoice) error {
			return errors.New("save error")
		},
	}

	service := NewBillingService(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{}, testLogger())
	_, err := service.VoidInvoice(context.Background(), inv.InvoiceID, "Bad")
	assert.Error(t, err)
}

func TestCalculateFees(t *testing.T) {
	service := NewBillingService(&fakeActivityRepo{}, &fakeInvoiceRepo{}, &fakeStorageRepo{}, testLogger())

	result, err := service.CalculateFees(context.Background(), CalculateFeesCommand{
		TenantID:   "TNT-001",
		SellerID:   "SLR-001",
		FacilityID: "FAC-001",
		FeeSchedule: FeeScheduleDTO{
			StorageFeePerCubicFtPerDay: 0.10,
			PickFeePerUnit:             0.50,
			PackFeePerOrder:            1.00,
			ReceivingFeePerUnit:        0.25,
			ShippingMarkupPercent:      20,
			ReturnProcessingFee:        2.00,
			GiftWrapFee:                0.50,
			HazmatHandlingFee:          3.00,
			OversizedItemFee:           4.00,
			ColdChainFeePerUnit:        1.50,
			FragileHandlingFee:         0.75,
		},
		StorageCubicFeet: 100,
		UnitsPicked:      10,
		OrdersPacked:     4,
		UnitsReceived:    8,
		ShippingBaseCost: 25,
		ReturnsProcessed: 2,
		GiftWrapItems:    5,
		HazmatUnits:      1,
		OversizedItems:   2,
		ColdChainUnits:   3,
		FragileItems:     4,
	})

	require.NoError(t, err)
	assert.Equal(t, 10.0, result.StorageFee)
	assert.Equal(t, 5.0, result.PickFee)
	assert.Equal(t, 4.0, result.PackFee)
	assert.Equal(t, 2.0, result.ReceivingFee)
	assert.Equal(t, 30.0, result.ShippingFee)
}

func TestRecordStorageCalculationStorageRepoError(t *testing.T) {
	storageRepo := &fakeStorageRepo{
		saveFn: func(_ context.Context, _ *domain.StorageCalculation) error {
			return errors.New("db error")
		},
	}

	service := NewBillingService(&fakeActivityRepo{}, &fakeInvoiceRepo{}, storageRepo, testLogger())
	err := service.RecordStorageCalculation(context.Background(), RecordStorageCommand{
		TenantID:        "TNT-001",
		SellerID:        "SLR-001",
		FacilityID:      "FAC-001",
		CalculationDate: time.Now().Add(-24 * time.Hour),
		TotalCubicFeet:  100,
		RatePerCubicFt:  0.10,
	})

	assert.Error(t, err)
}

func TestRecordStorageCalculationSuccess(t *testing.T) {
	var savedCalc *domain.StorageCalculation
	storageRepo := &fakeStorageRepo{
		saveFn: func(_ context.Context, calc *domain.StorageCalculation) error {
			savedCalc = calc
			return nil
		},
	}

	activityRepo := &fakeActivityRepo{
		saveFn: func(_ context.Context, _ *domain.BillableActivity) error {
			return errors.New("activity save failed")
		},
	}

	service := NewBillingService(activityRepo, &fakeInvoiceRepo{}, storageRepo, testLogger())
	err := service.RecordStorageCalculation(context.Background(), RecordStorageCommand{
		TenantID:        "TNT-001",
		SellerID:        "SLR-001",
		FacilityID:      "FAC-001",
		CalculationDate: time.Now().Add(-24 * time.Hour),
		TotalCubicFeet:  100,
		RatePerCubicFt:  0.10,
	})

	require.NoError(t, err)
	require.NotNil(t, savedCalc)
	assert.Equal(t, 10.0, savedCalc.TotalAmount)
}

func TestCheckOverdueInvoices(t *testing.T) {
	inv1 := domain.NewInvoice("TNT-001", "SLR-001", time.Now().Add(-48*time.Hour), time.Now().Add(-24*time.Hour), "Acme", "billing@acme.com")
	inv1.Status = domain.InvoiceStatusFinalized
	inv1.DueDate = time.Now().Add(-2 * time.Hour)

	inv2 := domain.NewInvoice("TNT-001", "SLR-001", time.Now().Add(-48*time.Hour), time.Now().Add(-24*time.Hour), "Acme", "billing@acme.com")
	inv2.Status = domain.InvoiceStatusFinalized
	inv2.DueDate = time.Now().Add(-2 * time.Hour)

	saveCalls := 0
	invoiceRepo := &fakeInvoiceRepo{
		findOverdueFn: func(_ context.Context) ([]*domain.Invoice, error) {
			return []*domain.Invoice{inv1, inv2}, nil
		},
		saveFn: func(_ context.Context, _ *domain.Invoice) error {
			saveCalls++
			if saveCalls == 2 {
				return errors.New("save failed")
			}
			return nil
		},
	}

	service := NewBillingService(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{}, testLogger())
	count, err := service.CheckOverdueInvoices(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestToActivityAndInvoiceDTOs(t *testing.T) {
	activity := &domain.BillableActivity{
		ActivityID:    "ACT-001",
		TenantID:      "TNT-001",
		SellerID:      "SLR-001",
		FacilityID:    "FAC-001",
		Type:          domain.ActivityTypePick,
		Description:   "Pick fee",
		Quantity:      10,
		UnitPrice:     0.25,
		Amount:        2.5,
		Currency:      "USD",
		ReferenceType: "order",
		ReferenceID:   "ORD-001",
		Invoiced:      false,
		CreatedAt:     time.Now(),
	}

	activityDTO := ToActivityDTO(activity)
	assert.Equal(t, "ACT-001", activityDTO.ActivityID)
	assert.Equal(t, "pick", activityDTO.Type)
	assert.Equal(t, 2.5, activityDTO.Amount)

	invoice := domain.NewInvoice("TNT-001", "SLR-001", time.Now().Add(-24*time.Hour), time.Now(), "Acme", "billing@acme.com")
	_ = invoice.AddLineItem(domain.ActivityTypePick, "Pick fee", 10, 0.25, []string{"ACT-001"})

	invoiceDTO := ToInvoiceDTO(invoice)
	require.Len(t, invoiceDTO.LineItems, 1)
	assert.Equal(t, "pick", invoiceDTO.LineItems[0].ActivityType)
	assert.Equal(t, invoice.InvoiceID, invoiceDTO.InvoiceID)
}

func TestGetActivityDescription(t *testing.T) {
	assert.Equal(t, "Picking fees", getActivityDescription(domain.ActivityTypePick))
	assert.Equal(t, "custom", getActivityDescription(domain.ActivityType("custom")))
}
