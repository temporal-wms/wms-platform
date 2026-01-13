package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/wms-platform/services/billing-service/internal/application"
	"github.com/wms-platform/services/billing-service/internal/domain"
	"github.com/wms-platform/shared/pkg/logging"
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
	cfg := logging.DefaultConfig("billing-handler-test")
	cfg.Output = io.Discard
	return logging.New(cfg)
}

func makeRequest(router *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}

	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func newHandler(activityRepo domain.BillableActivityRepository, invoiceRepo domain.InvoiceRepository, storageRepo domain.StorageCalculationRepository) *BillingHandler {
	service := application.NewBillingService(activityRepo, invoiceRepo, storageRepo, testLogger())
	return NewBillingHandler(service, testLogger())
}

func TestBillingHandlerRecordActivity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := newHandler(&fakeActivityRepo{}, &fakeInvoiceRepo{}, &fakeStorageRepo{})
	router.POST("/api/v1/activities", handler.RecordActivity)

	rec := makeRequest(router, http.MethodPost, "/api/v1/activities", map[string]interface{}{
		"tenantId":      "TNT-001",
		"sellerId":      "SLR-001",
		"facilityId":    "FAC-001",
		"type":          "pick",
		"description":   "Pick fee",
		"quantity":      10,
		"unitPrice":     0.25,
		"referenceType": "order",
		"referenceId":   "ORD-001",
	})

	assert.Equal(t, http.StatusCreated, rec.Code)

	rec = makeRequest(router, http.MethodPost, "/api/v1/activities", map[string]interface{}{
		"tenantId": "TNT-001",
	})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBillingHandlerRecordActivityRepoError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	activityRepo := &fakeActivityRepo{
		saveFn: func(_ context.Context, _ *domain.BillableActivity) error {
			return assert.AnError
		},
	}
	handler := newHandler(activityRepo, &fakeInvoiceRepo{}, &fakeStorageRepo{})
	router.POST("/api/v1/activities", handler.RecordActivity)

	rec := makeRequest(router, http.MethodPost, "/api/v1/activities", map[string]interface{}{
		"tenantId":      "TNT-001",
		"sellerId":      "SLR-001",
		"facilityId":    "FAC-001",
		"type":          "pick",
		"description":   "Pick fee",
		"quantity":      10,
		"unitPrice":     0.25,
		"referenceType": "order",
		"referenceId":   "ORD-001",
	})

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestBillingHandlerRecordActivities(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := newHandler(&fakeActivityRepo{}, &fakeInvoiceRepo{}, &fakeStorageRepo{})
	router.POST("/api/v1/activities/batch", handler.RecordActivities)

	rec := makeRequest(router, http.MethodPost, "/api/v1/activities/batch", map[string]interface{}{
		"activities": []map[string]interface{}{
			{
				"tenantId":      "TNT-001",
				"sellerId":      "SLR-001",
				"facilityId":    "FAC-001",
				"type":          "pick",
				"description":   "Pick fee",
				"quantity":      10,
				"unitPrice":     0.25,
				"referenceType": "order",
				"referenceId":   "ORD-001",
			},
		},
	})
	assert.Equal(t, http.StatusCreated, rec.Code)

	rec = makeRequest(router, http.MethodPost, "/api/v1/activities/batch", map[string]interface{}{
		"activities": []map[string]interface{}{},
	})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBillingHandlerRecordActivitiesRepoError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	activityRepo := &fakeActivityRepo{
		saveAllFn: func(_ context.Context, _ []*domain.BillableActivity) error {
			return assert.AnError
		},
	}
	handler := newHandler(activityRepo, &fakeInvoiceRepo{}, &fakeStorageRepo{})
	router.POST("/api/v1/activities/batch", handler.RecordActivities)

	rec := makeRequest(router, http.MethodPost, "/api/v1/activities/batch", map[string]interface{}{
		"activities": []map[string]interface{}{
			{
				"tenantId":      "TNT-001",
				"sellerId":      "SLR-001",
				"facilityId":    "FAC-001",
				"type":          "pick",
				"description":   "Pick fee",
				"quantity":      10,
				"unitPrice":     0.25,
				"referenceType": "order",
				"referenceId":   "ORD-001",
			},
		},
	})
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestBillingHandlerGetActivity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	activityRepo := &fakeActivityRepo{
		findByIDFn: func(_ context.Context, activityID string) (*domain.BillableActivity, error) {
			if activityID == "ACT-001" {
				return &domain.BillableActivity{ActivityID: "ACT-001", Type: domain.ActivityTypePick}, nil
			}
			return nil, nil
		},
	}
	handler := newHandler(activityRepo, &fakeInvoiceRepo{}, &fakeStorageRepo{})
	router.GET("/api/v1/activities/:activityId", handler.GetActivity)

	rec := makeRequest(router, http.MethodGet, "/api/v1/activities/ACT-001", nil)
	assert.Equal(t, http.StatusOK, rec.Code)

	rec = makeRequest(router, http.MethodGet, "/api/v1/activities/ACT-404", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestBillingHandlerListActivities(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	activityRepo := &fakeActivityRepo{
		findBySellerIDFn: func(_ context.Context, _ string, _ domain.Pagination) ([]*domain.BillableActivity, error) {
			return []*domain.BillableActivity{{ActivityID: "ACT-001"}}, nil
		},
	}
	handler := newHandler(activityRepo, &fakeInvoiceRepo{}, &fakeStorageRepo{})
	router.GET("/api/v1/sellers/:sellerId/activities", handler.ListActivities)

	rec := makeRequest(router, http.MethodGet, "/api/v1/sellers/SLR-001/activities?page=2&pageSize=10", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestBillingHandlerListActivitiesError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	activityRepo := &fakeActivityRepo{
		findBySellerIDFn: func(_ context.Context, _ string, _ domain.Pagination) ([]*domain.BillableActivity, error) {
			return nil, assert.AnError
		},
	}
	handler := newHandler(activityRepo, &fakeInvoiceRepo{}, &fakeStorageRepo{})
	router.GET("/api/v1/sellers/:sellerId/activities", handler.ListActivities)

	rec := makeRequest(router, http.MethodGet, "/api/v1/sellers/SLR-001/activities", nil)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestBillingHandlerGetActivitySummary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	activityRepo := &fakeActivityRepo{
		sumBySellerAndType: func(_ context.Context, _ string, _, _ time.Time) (map[domain.ActivityType]float64, error) {
			return map[domain.ActivityType]float64{domain.ActivityTypePick: 5}, nil
		},
	}
	handler := newHandler(activityRepo, &fakeInvoiceRepo{}, &fakeStorageRepo{})
	router.GET("/api/v1/sellers/:sellerId/activities/summary", handler.GetActivitySummary)

	rec := makeRequest(router, http.MethodGet, "/api/v1/sellers/SLR-001/activities/summary", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = makeRequest(router, http.MethodGet, "/api/v1/sellers/SLR-001/activities/summary?periodStart=bad&periodEnd=bad", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	start := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	end := time.Now().Format(time.RFC3339)
	rec = makeRequest(router, http.MethodGet, "/api/v1/sellers/SLR-001/activities/summary?periodStart="+start+"&periodEnd="+end, nil)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestBillingHandlerGetActivitySummaryError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	activityRepo := &fakeActivityRepo{
		sumBySellerAndType: func(_ context.Context, _ string, _, _ time.Time) (map[domain.ActivityType]float64, error) {
			return nil, assert.AnError
		},
	}
	handler := newHandler(activityRepo, &fakeInvoiceRepo{}, &fakeStorageRepo{})
	router.GET("/api/v1/sellers/:sellerId/activities/summary", handler.GetActivitySummary)

	start := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	end := time.Now().Format(time.RFC3339)
	rec := makeRequest(router, http.MethodGet, "/api/v1/sellers/SLR-001/activities/summary?periodStart="+start+"&periodEnd="+end, nil)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestBillingHandlerCreateInvoice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	activityRepo := &fakeActivityRepo{
		findUninvoicedFn: func(_ context.Context, _ string, _, _ time.Time) ([]*domain.BillableActivity, error) {
			return []*domain.BillableActivity{
				{ActivityID: "ACT-001", Type: domain.ActivityTypePick, Quantity: 10, Amount: 5},
			}, nil
		},
	}

	conflictInvoiceRepo := &fakeInvoiceRepo{
		findByPeriod: func(_ context.Context, _ string, _, _ time.Time) (*domain.Invoice, error) {
			return &domain.Invoice{InvoiceID: "INV-001"}, nil
		},
	}

	handler := newHandler(activityRepo, conflictInvoiceRepo, &fakeStorageRepo{})
	router.POST("/api/v1/invoices", handler.CreateInvoice)

	rec := makeRequest(router, http.MethodPost, "/api/v1/invoices", map[string]interface{}{
		"tenantId":    "TNT-001",
		"sellerId":    "SLR-001",
		"periodStart": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		"periodEnd":   time.Now().Format(time.RFC3339),
		"sellerName":  "Acme",
		"sellerEmail": "billing@acme.com",
	})
	assert.Equal(t, http.StatusConflict, rec.Code)

	successInvoiceRepo := &fakeInvoiceRepo{
		findByPeriod: func(_ context.Context, _ string, _, _ time.Time) (*domain.Invoice, error) {
			return nil, nil
		},
	}
	router = gin.New()
	handler = newHandler(activityRepo, successInvoiceRepo, &fakeStorageRepo{})
	router.POST("/api/v1/invoices", handler.CreateInvoice)

	rec = makeRequest(router, http.MethodPost, "/api/v1/invoices", map[string]interface{}{
		"tenantId":    "TNT-001",
		"sellerId":    "SLR-001",
		"periodStart": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		"periodEnd":   time.Now().Format(time.RFC3339),
		"sellerName":  "Acme",
		"sellerEmail": "billing@acme.com",
	})
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestBillingHandlerCreateInvoiceRepoError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	invoiceRepo := &fakeInvoiceRepo{
		findByPeriod: func(_ context.Context, _ string, _, _ time.Time) (*domain.Invoice, error) {
			return nil, assert.AnError
		},
	}
	handler := newHandler(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{})
	router.POST("/api/v1/invoices", handler.CreateInvoice)

	rec := makeRequest(router, http.MethodPost, "/api/v1/invoices", map[string]interface{}{
		"tenantId":    "TNT-001",
		"sellerId":    "SLR-001",
		"periodStart": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		"periodEnd":   time.Now().Format(time.RFC3339),
		"sellerName":  "Acme",
		"sellerEmail": "billing@acme.com",
	})
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestBillingHandlerGetInvoice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	invoiceRepo := &fakeInvoiceRepo{
		findByIDFn: func(_ context.Context, invoiceID string) (*domain.Invoice, error) {
			if invoiceID == "INV-001" {
				return &domain.Invoice{InvoiceID: "INV-001"}, nil
			}
			return nil, nil
		},
	}
	handler := newHandler(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{})
	router.GET("/api/v1/invoices/:invoiceId", handler.GetInvoice)

	rec := makeRequest(router, http.MethodGet, "/api/v1/invoices/INV-001", nil)
	assert.Equal(t, http.StatusOK, rec.Code)

	rec = makeRequest(router, http.MethodGet, "/api/v1/invoices/INV-404", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestBillingHandlerListInvoices(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	invoiceRepo := &fakeInvoiceRepo{
		findBySeller: func(_ context.Context, _ string, _ domain.Pagination) ([]*domain.Invoice, error) {
			return []*domain.Invoice{{InvoiceID: "INV-001"}}, nil
		},
		findByStatus: func(_ context.Context, _ domain.InvoiceStatus, _ domain.Pagination) ([]*domain.Invoice, error) {
			return []*domain.Invoice{{InvoiceID: "INV-002"}}, nil
		},
	}
	handler := newHandler(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{})
	router.GET("/api/v1/sellers/:sellerId/invoices", handler.ListInvoices)

	rec := makeRequest(router, http.MethodGet, "/api/v1/sellers/SLR-001/invoices", nil)
	assert.Equal(t, http.StatusOK, rec.Code)

	rec = makeRequest(router, http.MethodGet, "/api/v1/sellers/SLR-001/invoices?status=paid", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestBillingHandlerListInvoicesError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	invoiceRepo := &fakeInvoiceRepo{
		findBySeller: func(_ context.Context, _ string, _ domain.Pagination) ([]*domain.Invoice, error) {
			return nil, assert.AnError
		},
	}
	handler := newHandler(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{})
	router.GET("/api/v1/sellers/:sellerId/invoices", handler.ListInvoices)

	rec := makeRequest(router, http.MethodGet, "/api/v1/sellers/SLR-001/invoices", nil)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestBillingHandlerFinalizeInvoice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	invoiceRepo := &fakeInvoiceRepo{
		findByIDFn: func(_ context.Context, invoiceID string) (*domain.Invoice, error) {
			if invoiceID == "INV-001" {
				return domain.NewInvoice("TNT-001", "SLR-001", time.Now().Add(-24*time.Hour), time.Now(), "Acme", "billing@acme.com"), nil
			}
			return nil, nil
		},
	}
	handler := newHandler(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{})
	router.PUT("/api/v1/invoices/:invoiceId/finalize", handler.FinalizeInvoice)

	rec := makeRequest(router, http.MethodPut, "/api/v1/invoices/INV-001/finalize", nil)
	assert.Equal(t, http.StatusOK, rec.Code)

	rec = makeRequest(router, http.MethodPut, "/api/v1/invoices/INV-404/finalize", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestBillingHandlerFinalizeInvoiceSaveError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	invoiceRepo := &fakeInvoiceRepo{
		findByIDFn: func(_ context.Context, _ string) (*domain.Invoice, error) {
			return domain.NewInvoice("TNT-001", "SLR-001", time.Now().Add(-24*time.Hour), time.Now(), "Acme", "billing@acme.com"), nil
		},
		saveFn: func(_ context.Context, _ *domain.Invoice) error {
			return assert.AnError
		},
	}
	handler := newHandler(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{})
	router.PUT("/api/v1/invoices/:invoiceId/finalize", handler.FinalizeInvoice)

	rec := makeRequest(router, http.MethodPut, "/api/v1/invoices/INV-001/finalize", nil)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestBillingHandlerMarkInvoicePaid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	invoiceRepo := &fakeInvoiceRepo{
		findByIDFn: func(_ context.Context, invoiceID string) (*domain.Invoice, error) {
			if invoiceID == "INV-001" {
				inv := domain.NewInvoice("TNT-001", "SLR-001", time.Now().Add(-24*time.Hour), time.Now(), "Acme", "billing@acme.com")
				_ = inv.Finalize()
				return inv, nil
			}
			return nil, nil
		},
	}
	handler := newHandler(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{})
	router.PUT("/api/v1/invoices/:invoiceId/pay", handler.MarkInvoicePaid)

	rec := makeRequest(router, http.MethodPut, "/api/v1/invoices/INV-001/pay", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = makeRequest(router, http.MethodPut, "/api/v1/invoices/INV-001/pay", map[string]interface{}{
		"paymentMethod": "card",
		"paymentRef":    "TXN-001",
	})
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestBillingHandlerMarkInvoicePaidValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	invoiceRepo := &fakeInvoiceRepo{
		findByIDFn: func(_ context.Context, _ string) (*domain.Invoice, error) {
			return domain.NewInvoice("TNT-001", "SLR-001", time.Now().Add(-24*time.Hour), time.Now(), "Acme", "billing@acme.com"), nil
		},
	}
	handler := newHandler(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{})
	router.PUT("/api/v1/invoices/:invoiceId/pay", handler.MarkInvoicePaid)

	rec := makeRequest(router, http.MethodPut, "/api/v1/invoices/INV-001/pay", map[string]interface{}{
		"paymentMethod": "card",
	})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBillingHandlerVoidInvoice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	invoiceRepo := &fakeInvoiceRepo{
		findByIDFn: func(_ context.Context, invoiceID string) (*domain.Invoice, error) {
			if invoiceID == "INV-001" {
				return domain.NewInvoice("TNT-001", "SLR-001", time.Now().Add(-24*time.Hour), time.Now(), "Acme", "billing@acme.com"), nil
			}
			return nil, nil
		},
	}
	handler := newHandler(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{})
	router.PUT("/api/v1/invoices/:invoiceId/void", handler.VoidInvoice)

	rec := makeRequest(router, http.MethodPut, "/api/v1/invoices/INV-001/void", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = makeRequest(router, http.MethodPut, "/api/v1/invoices/INV-001/void", map[string]interface{}{
		"reason": "Duplicate",
	})
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestBillingHandlerVoidInvoiceValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	invoiceRepo := &fakeInvoiceRepo{
		findByIDFn: func(_ context.Context, _ string) (*domain.Invoice, error) {
			inv := domain.NewInvoice("TNT-001", "SLR-001", time.Now().Add(-24*time.Hour), time.Now(), "Acme", "billing@acme.com")
			_ = inv.Finalize()
			_ = inv.MarkPaid("card", "TXN-001")
			return inv, nil
		},
	}
	handler := newHandler(&fakeActivityRepo{}, invoiceRepo, &fakeStorageRepo{})
	router.PUT("/api/v1/invoices/:invoiceId/void", handler.VoidInvoice)

	rec := makeRequest(router, http.MethodPut, "/api/v1/invoices/INV-001/void", map[string]interface{}{
		"reason": "Duplicate",
	})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBillingHandlerCalculateFees(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := newHandler(&fakeActivityRepo{}, &fakeInvoiceRepo{}, &fakeStorageRepo{})
	router.POST("/api/v1/fees/calculate", handler.CalculateFees)

	rec := makeRequest(router, http.MethodPost, "/api/v1/fees/calculate", map[string]interface{}{
		"tenantId":   "TNT-001",
		"sellerId":   "SLR-001",
		"facilityId": "FAC-001",
		"feeSchedule": map[string]interface{}{
			"storageFeePerCubicFtPerDay": 0.10,
			"pickFeePerUnit":             0.50,
			"packFeePerOrder":            1.00,
			"receivingFeePerUnit":        0.25,
			"shippingMarkupPercent":      20,
			"returnProcessingFee":        2.00,
			"giftWrapFee":                0.50,
			"hazmatHandlingFee":          3.00,
			"oversizedItemFee":           4.00,
			"coldChainFeePerUnit":        1.50,
			"fragileHandlingFee":         0.75,
		},
		"storageCubicFeet": 100,
		"unitsPicked":      10,
		"ordersPacked":     4,
	})
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestBillingHandlerRecordStorage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := newHandler(&fakeActivityRepo{}, &fakeInvoiceRepo{}, &fakeStorageRepo{})
	router.POST("/api/v1/storage/calculate", handler.RecordStorage)

	rec := makeRequest(router, http.MethodPost, "/api/v1/storage/calculate", map[string]interface{}{
		"tenantId":        "TNT-001",
		"sellerId":        "SLR-001",
		"facilityId":      "FAC-001",
		"calculationDate": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		"totalCubicFeet":  100,
		"ratePerCubicFt":  0.10,
	})
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestBillingHandlerRecordStorageError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	storageRepo := &fakeStorageRepo{
		saveFn: func(_ context.Context, _ *domain.StorageCalculation) error {
			return assert.AnError
		},
	}
	handler := newHandler(&fakeActivityRepo{}, &fakeInvoiceRepo{}, storageRepo)
	router.POST("/api/v1/storage/calculate", handler.RecordStorage)

	rec := makeRequest(router, http.MethodPost, "/api/v1/storage/calculate", map[string]interface{}{
		"tenantId":        "TNT-001",
		"sellerId":        "SLR-001",
		"facilityId":      "FAC-001",
		"calculationDate": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		"totalCubicFeet":  100,
		"ratePerCubicFt":  0.10,
	})
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
