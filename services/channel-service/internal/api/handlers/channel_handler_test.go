package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/wms-platform/services/channel-service/internal/application"
	"github.com/wms-platform/services/channel-service/internal/domain"
	"github.com/wms-platform/shared/pkg/logging"
)

var errUnexpected = errors.New("unexpected call")

type fakeChannelRepo struct {
	saveFn           func(context.Context, *domain.Channel) error
	findByIDFn       func(context.Context, string) (*domain.Channel, error)
	findBySellerIDFn func(context.Context, string) ([]*domain.Channel, error)
}

func (f *fakeChannelRepo) Save(ctx context.Context, channel *domain.Channel) error {
	if f.saveFn == nil {
		return errUnexpected
	}
	return f.saveFn(ctx, channel)
}

func (f *fakeChannelRepo) FindByID(ctx context.Context, channelID string) (*domain.Channel, error) {
	if f.findByIDFn == nil {
		return nil, errUnexpected
	}
	return f.findByIDFn(ctx, channelID)
}

func (f *fakeChannelRepo) FindBySellerID(ctx context.Context, sellerID string) ([]*domain.Channel, error) {
	if f.findBySellerIDFn == nil {
		return nil, errUnexpected
	}
	return f.findBySellerIDFn(ctx, sellerID)
}

func (f *fakeChannelRepo) FindByType(context.Context, domain.ChannelType) ([]*domain.Channel, error) {
	return nil, errUnexpected
}

func (f *fakeChannelRepo) FindActiveChannels(context.Context) ([]*domain.Channel, error) {
	return nil, errUnexpected
}

func (f *fakeChannelRepo) FindChannelsNeedingSync(context.Context, domain.SyncType, time.Duration) ([]*domain.Channel, error) {
	return nil, errUnexpected
}

func (f *fakeChannelRepo) UpdateStatus(context.Context, string, domain.ChannelStatus) error {
	return errUnexpected
}

func (f *fakeChannelRepo) Delete(context.Context, string) error {
	return errUnexpected
}

type fakeOrderRepo struct {
	findByChannelIDFn func(context.Context, string, domain.Pagination) ([]*domain.ChannelOrder, error)
	findUnimportedFn  func(context.Context, string) ([]*domain.ChannelOrder, error)
	markImportedFn    func(context.Context, string, string) error
	markTrackingFn    func(context.Context, string) error
	findByExternalFn  func(context.Context, string, string) (*domain.ChannelOrder, error)
	saveAllFn         func(context.Context, []*domain.ChannelOrder) error
}

func (f *fakeOrderRepo) Save(context.Context, *domain.ChannelOrder) error {
	return errUnexpected
}

func (f *fakeOrderRepo) SaveAll(ctx context.Context, orders []*domain.ChannelOrder) error {
	if f.saveAllFn == nil {
		return errUnexpected
	}
	return f.saveAllFn(ctx, orders)
}

func (f *fakeOrderRepo) FindByExternalID(ctx context.Context, channelID, externalOrderID string) (*domain.ChannelOrder, error) {
	if f.findByExternalFn == nil {
		return nil, errUnexpected
	}
	return f.findByExternalFn(ctx, channelID, externalOrderID)
}

func (f *fakeOrderRepo) FindByChannelID(ctx context.Context, channelID string, pagination domain.Pagination) ([]*domain.ChannelOrder, error) {
	if f.findByChannelIDFn == nil {
		return nil, errUnexpected
	}
	return f.findByChannelIDFn(ctx, channelID, pagination)
}

func (f *fakeOrderRepo) FindUnimported(ctx context.Context, channelID string) ([]*domain.ChannelOrder, error) {
	if f.findUnimportedFn == nil {
		return nil, errUnexpected
	}
	return f.findUnimportedFn(ctx, channelID)
}

func (f *fakeOrderRepo) FindWithoutTracking(context.Context, string) ([]*domain.ChannelOrder, error) {
	return nil, errUnexpected
}

func (f *fakeOrderRepo) MarkImported(ctx context.Context, externalOrderID, wmsOrderID string) error {
	if f.markImportedFn == nil {
		return errUnexpected
	}
	return f.markImportedFn(ctx, externalOrderID, wmsOrderID)
}

func (f *fakeOrderRepo) MarkTrackingPushed(ctx context.Context, externalOrderID string) error {
	if f.markTrackingFn == nil {
		return errUnexpected
	}
	return f.markTrackingFn(ctx, externalOrderID)
}

func (f *fakeOrderRepo) Count(context.Context, string) (int64, error) {
	return 0, errUnexpected
}

type fakeSyncJobRepo struct {
	findByChannelFn func(context.Context, string, domain.Pagination) ([]*domain.SyncJob, error)
	findRunningFn   func(context.Context, string, domain.SyncType) (*domain.SyncJob, error)
	saveFn          func(context.Context, *domain.SyncJob) error
}

func (f *fakeSyncJobRepo) Save(ctx context.Context, job *domain.SyncJob) error {
	if f.saveFn == nil {
		return errUnexpected
	}
	return f.saveFn(ctx, job)
}

func (f *fakeSyncJobRepo) FindByID(context.Context, string) (*domain.SyncJob, error) {
	return nil, errUnexpected
}

func (f *fakeSyncJobRepo) FindByChannelID(ctx context.Context, channelID string, pagination domain.Pagination) ([]*domain.SyncJob, error) {
	if f.findByChannelFn == nil {
		return nil, errUnexpected
	}
	return f.findByChannelFn(ctx, channelID, pagination)
}

func (f *fakeSyncJobRepo) FindRunning(ctx context.Context, channelID string, syncType domain.SyncType) (*domain.SyncJob, error) {
	if f.findRunningFn == nil {
		return nil, errUnexpected
	}
	return f.findRunningFn(ctx, channelID, syncType)
}

func (f *fakeSyncJobRepo) FindLatest(context.Context, string, domain.SyncType) (*domain.SyncJob, error) {
	return nil, errUnexpected
}

type fakeAdapter struct {
	channelType          domain.ChannelType
	validateFn           func(context.Context, domain.ChannelCredentials) error
	fetchOrdersFn        func(context.Context, *domain.Channel, time.Time) ([]*domain.ChannelOrder, error)
	pushTrackingFn       func(context.Context, *domain.Channel, string, domain.TrackingInfo) error
	syncInventoryFn      func(context.Context, *domain.Channel, []domain.InventoryUpdate) error
	getInventoryLevelsFn func(context.Context, *domain.Channel, []string) ([]domain.InventoryLevel, error)
	createFulfillmentFn  func(context.Context, *domain.Channel, domain.FulfillmentRequest) error
	registerWebhooksFn   func(context.Context, *domain.Channel, string) error
	validateWebhookFn    func(context.Context, *domain.Channel, string, []byte) bool
}

func (f *fakeAdapter) GetType() domain.ChannelType {
	return f.channelType
}

func (f *fakeAdapter) ValidateCredentials(ctx context.Context, creds domain.ChannelCredentials) error {
	if f.validateFn == nil {
		return errUnexpected
	}
	return f.validateFn(ctx, creds)
}

func (f *fakeAdapter) FetchOrders(ctx context.Context, channel *domain.Channel, since time.Time) ([]*domain.ChannelOrder, error) {
	if f.fetchOrdersFn == nil {
		return nil, errUnexpected
	}
	return f.fetchOrdersFn(ctx, channel, since)
}

func (f *fakeAdapter) FetchOrder(context.Context, *domain.Channel, string) (*domain.ChannelOrder, error) {
	return nil, errUnexpected
}

func (f *fakeAdapter) PushTracking(ctx context.Context, channel *domain.Channel, externalOrderID string, tracking domain.TrackingInfo) error {
	if f.pushTrackingFn == nil {
		return errUnexpected
	}
	return f.pushTrackingFn(ctx, channel, externalOrderID, tracking)
}

func (f *fakeAdapter) SyncInventory(ctx context.Context, channel *domain.Channel, items []domain.InventoryUpdate) error {
	if f.syncInventoryFn == nil {
		return errUnexpected
	}
	return f.syncInventoryFn(ctx, channel, items)
}

func (f *fakeAdapter) GetInventoryLevels(ctx context.Context, channel *domain.Channel, skus []string) ([]domain.InventoryLevel, error) {
	if f.getInventoryLevelsFn == nil {
		return nil, errUnexpected
	}
	return f.getInventoryLevelsFn(ctx, channel, skus)
}

func (f *fakeAdapter) CreateFulfillment(ctx context.Context, channel *domain.Channel, fulfillment domain.FulfillmentRequest) error {
	if f.createFulfillmentFn == nil {
		return errUnexpected
	}
	return f.createFulfillmentFn(ctx, channel, fulfillment)
}

func (f *fakeAdapter) RegisterWebhooks(ctx context.Context, channel *domain.Channel, webhookURL string) error {
	if f.registerWebhooksFn == nil {
		return errUnexpected
	}
	return f.registerWebhooksFn(ctx, channel, webhookURL)
}

func (f *fakeAdapter) ValidateWebhook(ctx context.Context, channel *domain.Channel, signature string, body []byte) bool {
	if f.validateWebhookFn == nil {
		return false
	}
	return f.validateWebhookFn(ctx, channel, signature, body)
}

type fakeMetrics struct {
	syncStatus       string
	ordersImported   int
	webhookStatus    string
	apiLatencyCalled bool
}

func (f *fakeMetrics) RecordSyncOperation(channel, syncType, status string, duration time.Duration) {
	f.syncStatus = status
}

func (f *fakeMetrics) RecordOrdersImported(channel string, count int) {
	f.ordersImported += count
}

func (f *fakeMetrics) RecordAPILatency(channel, operation, status string, duration time.Duration) {
	f.apiLatencyCalled = true
}

func (f *fakeMetrics) RecordWebhookReceived(channel, topic, status string) {
	f.webhookStatus = status
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("read failure")
}

func (errReader) Close() error {
	return nil
}

type handlerEnv struct {
	router      *gin.Engine
	service     *application.ChannelService
	channelRepo *fakeChannelRepo
	orderRepo   *fakeOrderRepo
	syncRepo    *fakeSyncJobRepo
	adapter     *fakeAdapter
	metrics     *fakeMetrics
}

func newHandlerEnv(adapter *fakeAdapter) *handlerEnv {
	channelRepo := &fakeChannelRepo{}
	orderRepo := &fakeOrderRepo{}
	syncRepo := &fakeSyncJobRepo{}
	factory := domain.NewAdapterFactory()
	if adapter != nil {
		factory.Register(adapter)
	}
	service := application.NewChannelService(channelRepo, orderRepo, syncRepo, factory)
	logger := logging.New(&logging.Config{
		Level:       logging.LevelInfo,
		ServiceName: "test",
		Environment: "test",
		Version:     "test",
		Output:      io.Discard,
	})
	metrics := &fakeMetrics{}
	handler := NewChannelHandler(service, logger, metrics)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler.RegisterRoutes(router.Group(""))

	return &handlerEnv{
		router:      router,
		service:     service,
		channelRepo: channelRepo,
		orderRepo:   orderRepo,
		syncRepo:    syncRepo,
		adapter:     adapter,
		metrics:     metrics,
	}
}

func newTestChannel(t *testing.T, channelType domain.ChannelType) *domain.Channel {
	t.Helper()
	channel, err := domain.NewChannel("tenant-1", "seller-1", channelType, "Test", "", domain.ChannelCredentials{}, domain.SyncSettings{})
	require.NoError(t, err)
	return channel
}

func performRequest(router *gin.Engine, method, path string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestConnectChannelBadJSON(t *testing.T) {
	env := newHandlerEnv(nil)
	resp := performRequest(env.router, http.MethodPost, "/channels", []byte("{bad"), nil)
	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestConnectChannelSuccess(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeShopify,
		validateFn: func(context.Context, domain.ChannelCredentials) error {
			return nil
		},
		registerWebhooksFn: func(context.Context, *domain.Channel, string) error {
			return nil
		},
	}
	env := newHandlerEnv(adapter)
	env.channelRepo.saveFn = func(context.Context, *domain.Channel) error { return nil }

	payload := map[string]any{
		"tenantId": "tenant-1",
		"sellerId": "seller-1",
		"type":     "shopify",
		"name":     "Shop",
		"credentials": map[string]string{
			"storeDomain": "example.myshopify.com",
			"accessToken": "token",
		},
		"webhookUrl": "https://example.com/webhook",
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	resp := performRequest(env.router, http.MethodPost, "/channels", body, nil)
	require.Equal(t, http.StatusCreated, resp.Code)
}

func TestGetChannelNotFound(t *testing.T) {
	env := newHandlerEnv(nil)
	env.channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return nil, errors.New("not found")
	}

	resp := performRequest(env.router, http.MethodGet, "/channels/ch-1", nil, nil)
	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestGetChannelSuccess(t *testing.T) {
	env := newHandlerEnv(nil)
	channel := newTestChannel(t, domain.ChannelTypeAmazon)
	env.channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}

	resp := performRequest(env.router, http.MethodGet, "/channels/"+channel.ChannelID, nil, nil)
	require.Equal(t, http.StatusOK, resp.Code)
}

func TestGetChannelsBySellerError(t *testing.T) {
	env := newHandlerEnv(nil)
	env.channelRepo.findBySellerIDFn = func(context.Context, string) ([]*domain.Channel, error) {
		return nil, errors.New("failed")
	}

	resp := performRequest(env.router, http.MethodGet, "/sellers/seller-1/channels", nil, nil)
	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestGetChannelsBySellerSuccess(t *testing.T) {
	env := newHandlerEnv(nil)
	env.channelRepo.findBySellerIDFn = func(context.Context, string) ([]*domain.Channel, error) {
		return []*domain.Channel{newTestChannel(t, domain.ChannelTypeAmazon)}, nil
	}

	resp := performRequest(env.router, http.MethodGet, "/sellers/seller-1/channels", nil, nil)
	require.Equal(t, http.StatusOK, resp.Code)
}

func TestUpdateChannelBadJSON(t *testing.T) {
	env := newHandlerEnv(nil)
	resp := performRequest(env.router, http.MethodPut, "/channels/ch-1", []byte("{bad"), nil)
	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestUpdateChannelError(t *testing.T) {
	env := newHandlerEnv(nil)
	env.channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return newTestChannel(t, domain.ChannelTypeEbay), nil
	}
	env.channelRepo.saveFn = func(context.Context, *domain.Channel) error {
		return errors.New("save failed")
	}

	resp := performRequest(env.router, http.MethodPut, "/channels/ch-1", []byte(`{"name":"new"}`), nil)
	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestUpdateChannelSuccess(t *testing.T) {
	env := newHandlerEnv(nil)
	env.channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return newTestChannel(t, domain.ChannelTypeEbay), nil
	}
	env.channelRepo.saveFn = func(context.Context, *domain.Channel) error { return nil }

	resp := performRequest(env.router, http.MethodPut, "/channels/ch-1", []byte(`{"name":"new"}`), nil)
	require.Equal(t, http.StatusOK, resp.Code)
}

func TestDisconnectChannelError(t *testing.T) {
	env := newHandlerEnv(nil)
	env.channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return nil, errors.New("fail")
	}

	resp := performRequest(env.router, http.MethodDelete, "/channels/ch-1", nil, nil)
	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestDisconnectChannelSuccess(t *testing.T) {
	env := newHandlerEnv(nil)
	env.channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return newTestChannel(t, domain.ChannelTypeWooCommerce), nil
	}
	env.channelRepo.saveFn = func(context.Context, *domain.Channel) error { return nil }

	resp := performRequest(env.router, http.MethodDelete, "/channels/ch-1", nil, nil)
	require.Equal(t, http.StatusOK, resp.Code)
}

func TestGetChannelOrdersError(t *testing.T) {
	env := newHandlerEnv(nil)
	env.orderRepo.findByChannelIDFn = func(context.Context, string, domain.Pagination) ([]*domain.ChannelOrder, error) {
		return nil, errors.New("fail")
	}

	resp := performRequest(env.router, http.MethodGet, "/channels/ch-1/orders", nil, nil)
	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestGetChannelOrdersSuccess(t *testing.T) {
	env := newHandlerEnv(nil)
	env.orderRepo.findByChannelIDFn = func(context.Context, string, domain.Pagination) ([]*domain.ChannelOrder, error) {
		return []*domain.ChannelOrder{{ExternalOrderID: "ext-1"}}, nil
	}

	resp := performRequest(env.router, http.MethodGet, "/channels/ch-1/orders?page=2&pageSize=10", nil, nil)
	require.Equal(t, http.StatusOK, resp.Code)
}

func TestGetUnimportedOrdersError(t *testing.T) {
	env := newHandlerEnv(nil)
	env.orderRepo.findUnimportedFn = func(context.Context, string) ([]*domain.ChannelOrder, error) {
		return nil, errors.New("fail")
	}

	resp := performRequest(env.router, http.MethodGet, "/channels/ch-1/orders/unimported", nil, nil)
	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestGetUnimportedOrdersSuccess(t *testing.T) {
	env := newHandlerEnv(nil)
	env.orderRepo.findUnimportedFn = func(context.Context, string) ([]*domain.ChannelOrder, error) {
		return []*domain.ChannelOrder{{ExternalOrderID: "ext-1"}}, nil
	}

	resp := performRequest(env.router, http.MethodGet, "/channels/ch-1/orders/unimported", nil, nil)
	require.Equal(t, http.StatusOK, resp.Code)
}

func TestGetSyncJobsError(t *testing.T) {
	env := newHandlerEnv(nil)
	env.syncRepo.findByChannelFn = func(context.Context, string, domain.Pagination) ([]*domain.SyncJob, error) {
		return nil, errors.New("fail")
	}

	resp := performRequest(env.router, http.MethodGet, "/channels/ch-1/sync-jobs", nil, nil)
	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestGetSyncJobsSuccess(t *testing.T) {
	env := newHandlerEnv(nil)
	env.syncRepo.findByChannelFn = func(context.Context, string, domain.Pagination) ([]*domain.SyncJob, error) {
		return []*domain.SyncJob{domain.NewSyncJob("t", "s", "ch-1", domain.SyncTypeOrders, "inbound")}, nil
	}

	resp := performRequest(env.router, http.MethodGet, "/channels/ch-1/sync-jobs?page=1&pageSize=5", nil, nil)
	require.Equal(t, http.StatusOK, resp.Code)
}

func TestSyncOrdersError(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeShopify,
		fetchOrdersFn: func(context.Context, *domain.Channel, time.Time) ([]*domain.ChannelOrder, error) {
			return nil, errors.New("fetch failed")
		},
	}
	env := newHandlerEnv(adapter)
	channel := newTestChannel(t, domain.ChannelTypeShopify)
	env.channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}
	env.syncRepo.findRunningFn = func(context.Context, string, domain.SyncType) (*domain.SyncJob, error) {
		return nil, nil
	}
	env.syncRepo.saveFn = func(context.Context, *domain.SyncJob) error { return nil }

	resp := performRequest(env.router, http.MethodPost, "/channels/"+channel.ChannelID+"/sync/orders", nil, nil)
	require.Equal(t, http.StatusInternalServerError, resp.Code)
	require.Equal(t, "error", env.metrics.syncStatus)
}

func TestSyncOrdersSuccess(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeShopify,
		fetchOrdersFn: func(context.Context, *domain.Channel, time.Time) ([]*domain.ChannelOrder, error) {
			return []*domain.ChannelOrder{}, nil
		},
	}
	env := newHandlerEnv(adapter)
	channel := newTestChannel(t, domain.ChannelTypeShopify)
	env.channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}
	env.channelRepo.saveFn = func(context.Context, *domain.Channel) error { return nil }
	env.syncRepo.findRunningFn = func(context.Context, string, domain.SyncType) (*domain.SyncJob, error) {
		return nil, nil
	}
	env.syncRepo.saveFn = func(context.Context, *domain.SyncJob) error { return nil }
	env.orderRepo.findByExternalFn = func(context.Context, string, string) (*domain.ChannelOrder, error) {
		return nil, nil
	}
	env.orderRepo.saveAllFn = func(context.Context, []*domain.ChannelOrder) error { return nil }

	resp := performRequest(env.router, http.MethodPost, "/channels/"+channel.ChannelID+"/sync/orders", nil, nil)
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "success", env.metrics.syncStatus)
}

func TestSyncInventoryBadJSON(t *testing.T) {
	env := newHandlerEnv(nil)
	resp := performRequest(env.router, http.MethodPost, "/channels/ch-1/sync/inventory", []byte("{bad"), nil)
	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestSyncInventoryError(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeAmazon,
		syncInventoryFn: func(context.Context, *domain.Channel, []domain.InventoryUpdate) error {
			return errors.New("sync failed")
		},
	}
	env := newHandlerEnv(adapter)
	channel := newTestChannel(t, domain.ChannelTypeAmazon)
	env.channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}
	env.syncRepo.saveFn = func(context.Context, *domain.SyncJob) error { return nil }

	body := []byte(`{"channelId":"` + channel.ChannelID + `","items":[{"sku":"sku-1","quantity":1,"available":1}]}`)
	resp := performRequest(env.router, http.MethodPost, "/channels/"+channel.ChannelID+"/sync/inventory", body, nil)
	require.Equal(t, http.StatusInternalServerError, resp.Code)
	require.Equal(t, "error", env.metrics.syncStatus)
}

func TestSyncInventorySuccess(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeAmazon,
		syncInventoryFn: func(context.Context, *domain.Channel, []domain.InventoryUpdate) error {
			return nil
		},
	}
	env := newHandlerEnv(adapter)
	channel := newTestChannel(t, domain.ChannelTypeAmazon)
	env.channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}
	env.channelRepo.saveFn = func(context.Context, *domain.Channel) error { return nil }
	env.syncRepo.saveFn = func(context.Context, *domain.SyncJob) error { return nil }

	body := []byte(`{"channelId":"` + channel.ChannelID + `","items":[{"sku":"sku-1","quantity":1,"available":1}]}`)
	resp := performRequest(env.router, http.MethodPost, "/channels/"+channel.ChannelID+"/sync/inventory", body, nil)
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "success", env.metrics.syncStatus)
}

func TestPushTrackingBadJSON(t *testing.T) {
	env := newHandlerEnv(nil)
	resp := performRequest(env.router, http.MethodPost, "/channels/ch-1/tracking", []byte("{bad"), nil)
	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestPushTrackingError(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeEbay,
		pushTrackingFn: func(context.Context, *domain.Channel, string, domain.TrackingInfo) error {
			return errors.New("push failed")
		},
	}
	env := newHandlerEnv(adapter)
	channel := newTestChannel(t, domain.ChannelTypeEbay)
	env.channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}

	body := []byte(`{"channelId":"` + channel.ChannelID + `","externalOrderId":"ext-1","trackingNumber":"track-1","carrier":"ups"}`)
	resp := performRequest(env.router, http.MethodPost, "/channels/"+channel.ChannelID+"/tracking", body, nil)
	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestPushTrackingSuccess(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeEbay,
		pushTrackingFn: func(context.Context, *domain.Channel, string, domain.TrackingInfo) error {
			return nil
		},
	}
	env := newHandlerEnv(adapter)
	channel := newTestChannel(t, domain.ChannelTypeEbay)
	env.channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}
	env.orderRepo.markTrackingFn = func(context.Context, string) error { return nil }

	body := []byte(`{"channelId":"` + channel.ChannelID + `","externalOrderId":"ext-1","trackingNumber":"track-1","carrier":"ups"}`)
	resp := performRequest(env.router, http.MethodPost, "/channels/"+channel.ChannelID+"/tracking", body, nil)
	require.Equal(t, http.StatusOK, resp.Code)
}

func TestCreateFulfillmentBadJSON(t *testing.T) {
	env := newHandlerEnv(nil)
	resp := performRequest(env.router, http.MethodPost, "/channels/ch-1/fulfillment", []byte("{bad"), nil)
	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestCreateFulfillmentError(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeWooCommerce,
		createFulfillmentFn: func(context.Context, *domain.Channel, domain.FulfillmentRequest) error {
			return errors.New("fulfillment failed")
		},
	}
	env := newHandlerEnv(adapter)
	channel := newTestChannel(t, domain.ChannelTypeWooCommerce)
	env.channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}

	body := []byte(`{"channelId":"` + channel.ChannelID + `","externalOrderId":"ext-1","trackingNumber":"track-1","carrier":"ups"}`)
	resp := performRequest(env.router, http.MethodPost, "/channels/"+channel.ChannelID+"/fulfillment", body, nil)
	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestCreateFulfillmentSuccess(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeWooCommerce,
		createFulfillmentFn: func(context.Context, *domain.Channel, domain.FulfillmentRequest) error {
			return nil
		},
	}
	env := newHandlerEnv(adapter)
	channel := newTestChannel(t, domain.ChannelTypeWooCommerce)
	env.channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}
	env.orderRepo.markTrackingFn = func(context.Context, string) error { return nil }

	body := []byte(`{"channelId":"` + channel.ChannelID + `","externalOrderId":"ext-1","trackingNumber":"track-1","carrier":"ups"}`)
	resp := performRequest(env.router, http.MethodPost, "/channels/"+channel.ChannelID+"/fulfillment", body, nil)
	require.Equal(t, http.StatusOK, resp.Code)
}

func TestImportOrderBadJSON(t *testing.T) {
	env := newHandlerEnv(nil)
	resp := performRequest(env.router, http.MethodPost, "/channels/ch-1/orders/import", []byte("{bad"), nil)
	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestImportOrderError(t *testing.T) {
	env := newHandlerEnv(nil)
	env.orderRepo.markImportedFn = func(context.Context, string, string) error {
		return errors.New("mark failed")
	}

	body := []byte(`{"channelId":"ch-1","externalOrderId":"ext-1","wmsOrderId":"wms-1"}`)
	resp := performRequest(env.router, http.MethodPost, "/channels/ch-1/orders/import", body, nil)
	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestImportOrderSuccess(t *testing.T) {
	env := newHandlerEnv(nil)
	env.orderRepo.markImportedFn = func(context.Context, string, string) error { return nil }

	body := []byte(`{"channelId":"ch-1","externalOrderId":"ext-1","wmsOrderId":"wms-1"}`)
	resp := performRequest(env.router, http.MethodPost, "/channels/ch-1/orders/import", body, nil)
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, 1, env.metrics.ordersImported)
}

func TestGetInventoryLevelsNoSKU(t *testing.T) {
	env := newHandlerEnv(nil)
	resp := performRequest(env.router, http.MethodGet, "/channels/ch-1/inventory", nil, nil)
	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestGetInventoryLevelsError(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeAmazon,
		getInventoryLevelsFn: func(context.Context, *domain.Channel, []string) ([]domain.InventoryLevel, error) {
			return nil, errors.New("fail")
		},
	}
	env := newHandlerEnv(adapter)
	channel := newTestChannel(t, domain.ChannelTypeAmazon)
	env.channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}

	resp := performRequest(env.router, http.MethodGet, "/channels/"+channel.ChannelID+"/inventory?sku=sku-1", nil, nil)
	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestGetInventoryLevelsSuccess(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeAmazon,
		getInventoryLevelsFn: func(context.Context, *domain.Channel, []string) ([]domain.InventoryLevel, error) {
			return []domain.InventoryLevel{{SKU: "sku-1"}}, nil
		},
	}
	env := newHandlerEnv(adapter)
	channel := newTestChannel(t, domain.ChannelTypeAmazon)
	env.channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}

	resp := performRequest(env.router, http.MethodGet, "/channels/"+channel.ChannelID+"/inventory?sku=sku-1", nil, nil)
	require.Equal(t, http.StatusOK, resp.Code)
}

func TestHandleWebhookBodyError(t *testing.T) {
	env := newHandlerEnv(nil)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/ch-1", errReader{})
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleWebhookUnauthorized(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeShopify,
		validateWebhookFn: func(context.Context, *domain.Channel, string, []byte) bool {
			return false
		},
	}
	env := newHandlerEnv(adapter)
	channel := newTestChannel(t, domain.ChannelTypeShopify)
	env.channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}

	headers := map[string]string{"X-Shopify-Hmac-Sha256": "sig"}
	resp := performRequest(env.router, http.MethodPost, "/webhooks/"+channel.ChannelID+"/orders-create", []byte(`{}`), headers)
	require.Equal(t, http.StatusUnauthorized, resp.Code)
	require.Equal(t, "error", env.metrics.webhookStatus)
}

func TestHandleWebhookSuccess(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeShopify,
		validateWebhookFn: func(context.Context, *domain.Channel, string, []byte) bool {
			return true
		},
	}
	env := newHandlerEnv(adapter)
	channel := newTestChannel(t, domain.ChannelTypeShopify)
	env.channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}

	headers := map[string]string{"X-Shopify-Hmac-Sha256": "sig"}
	resp := performRequest(env.router, http.MethodPost, "/webhooks/"+channel.ChannelID+"/orders-create", []byte(`{}`), headers)
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "success", env.metrics.webhookStatus)
}
