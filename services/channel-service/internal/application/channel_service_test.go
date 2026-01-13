package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wms-platform/services/channel-service/internal/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var errUnexpected = errors.New("unexpected call")

type fakeChannelRepo struct {
	saveFn            func(context.Context, *domain.Channel) error
	findByIDFn        func(context.Context, string) (*domain.Channel, error)
	findBySellerIDFn  func(context.Context, string) ([]*domain.Channel, error)
	updateStatusFn    func(context.Context, string, domain.ChannelStatus) error
	deleteFn          func(context.Context, string) error
	findByTypeFn      func(context.Context, domain.ChannelType) ([]*domain.Channel, error)
	findActiveFn      func(context.Context) ([]*domain.Channel, error)
	findNeedingSyncFn func(context.Context, domain.SyncType, time.Duration) ([]*domain.Channel, error)
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

func (f *fakeChannelRepo) FindByType(ctx context.Context, channelType domain.ChannelType) ([]*domain.Channel, error) {
	if f.findByTypeFn == nil {
		return nil, errUnexpected
	}
	return f.findByTypeFn(ctx, channelType)
}

func (f *fakeChannelRepo) FindActiveChannels(ctx context.Context) ([]*domain.Channel, error) {
	if f.findActiveFn == nil {
		return nil, errUnexpected
	}
	return f.findActiveFn(ctx)
}

func (f *fakeChannelRepo) FindChannelsNeedingSync(ctx context.Context, syncType domain.SyncType, threshold time.Duration) ([]*domain.Channel, error) {
	if f.findNeedingSyncFn == nil {
		return nil, errUnexpected
	}
	return f.findNeedingSyncFn(ctx, syncType, threshold)
}

func (f *fakeChannelRepo) UpdateStatus(ctx context.Context, channelID string, status domain.ChannelStatus) error {
	if f.updateStatusFn == nil {
		return errUnexpected
	}
	return f.updateStatusFn(ctx, channelID, status)
}

func (f *fakeChannelRepo) Delete(ctx context.Context, channelID string) error {
	if f.deleteFn == nil {
		return errUnexpected
	}
	return f.deleteFn(ctx, channelID)
}

type fakeOrderRepo struct {
	saveFn             func(context.Context, *domain.ChannelOrder) error
	saveAllFn          func(context.Context, []*domain.ChannelOrder) error
	findByExternalIDFn func(context.Context, string, string) (*domain.ChannelOrder, error)
	findByChannelIDFn  func(context.Context, string, domain.Pagination) ([]*domain.ChannelOrder, error)
	findUnimportedFn   func(context.Context, string) ([]*domain.ChannelOrder, error)
	findWithoutTrackFn func(context.Context, string) ([]*domain.ChannelOrder, error)
	markImportedFn     func(context.Context, string, string) error
	markTrackingFn     func(context.Context, string) error
	countFn            func(context.Context, string) (int64, error)
}

func (f *fakeOrderRepo) Save(ctx context.Context, order *domain.ChannelOrder) error {
	if f.saveFn == nil {
		return errUnexpected
	}
	return f.saveFn(ctx, order)
}

func (f *fakeOrderRepo) SaveAll(ctx context.Context, orders []*domain.ChannelOrder) error {
	if f.saveAllFn == nil {
		return errUnexpected
	}
	return f.saveAllFn(ctx, orders)
}

func (f *fakeOrderRepo) FindByExternalID(ctx context.Context, channelID, externalOrderID string) (*domain.ChannelOrder, error) {
	if f.findByExternalIDFn == nil {
		return nil, errUnexpected
	}
	return f.findByExternalIDFn(ctx, channelID, externalOrderID)
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

func (f *fakeOrderRepo) FindWithoutTracking(ctx context.Context, channelID string) ([]*domain.ChannelOrder, error) {
	if f.findWithoutTrackFn == nil {
		return nil, errUnexpected
	}
	return f.findWithoutTrackFn(ctx, channelID)
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

func (f *fakeOrderRepo) Count(ctx context.Context, channelID string) (int64, error) {
	if f.countFn == nil {
		return 0, errUnexpected
	}
	return f.countFn(ctx, channelID)
}

type fakeSyncJobRepo struct {
	saveFn          func(context.Context, *domain.SyncJob) error
	findByIDFn      func(context.Context, string) (*domain.SyncJob, error)
	findByChannelFn func(context.Context, string, domain.Pagination) ([]*domain.SyncJob, error)
	findRunningFn   func(context.Context, string, domain.SyncType) (*domain.SyncJob, error)
	findLatestFn    func(context.Context, string, domain.SyncType) (*domain.SyncJob, error)
}

func (f *fakeSyncJobRepo) Save(ctx context.Context, job *domain.SyncJob) error {
	if f.saveFn == nil {
		return errUnexpected
	}
	return f.saveFn(ctx, job)
}

func (f *fakeSyncJobRepo) FindByID(ctx context.Context, jobID string) (*domain.SyncJob, error) {
	if f.findByIDFn == nil {
		return nil, errUnexpected
	}
	return f.findByIDFn(ctx, jobID)
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

func (f *fakeSyncJobRepo) FindLatest(ctx context.Context, channelID string, syncType domain.SyncType) (*domain.SyncJob, error) {
	if f.findLatestFn == nil {
		return nil, errUnexpected
	}
	return f.findLatestFn(ctx, channelID, syncType)
}

type fakeAdapter struct {
	channelType          domain.ChannelType
	validateFn           func(context.Context, domain.ChannelCredentials) error
	fetchOrdersFn        func(context.Context, *domain.Channel, time.Time) ([]*domain.ChannelOrder, error)
	fetchOrderFn         func(context.Context, *domain.Channel, string) (*domain.ChannelOrder, error)
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

func (f *fakeAdapter) FetchOrder(ctx context.Context, channel *domain.Channel, externalOrderID string) (*domain.ChannelOrder, error) {
	if f.fetchOrderFn == nil {
		return nil, errUnexpected
	}
	return f.fetchOrderFn(ctx, channel, externalOrderID)
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

func newServiceWithAdapter(adapter domain.ChannelAdapter) (*ChannelService, *fakeChannelRepo, *fakeOrderRepo, *fakeSyncJobRepo) {
	channelRepo := &fakeChannelRepo{}
	orderRepo := &fakeOrderRepo{}
	syncJobRepo := &fakeSyncJobRepo{}
	factory := domain.NewAdapterFactory()
	if adapter != nil {
		factory.Register(adapter)
	}
	return NewChannelService(channelRepo, orderRepo, syncJobRepo, factory), channelRepo, orderRepo, syncJobRepo
}

func newTestChannel(t *testing.T, channelType domain.ChannelType) *domain.Channel {
	t.Helper()
	channel, err := domain.NewChannel("tenant-1", "seller-1", channelType, "Test", "", domain.ChannelCredentials{}, domain.SyncSettings{})
	require.NoError(t, err)
	return channel
}

func TestConnectChannelUnsupportedType(t *testing.T) {
	service, _, _, _ := newServiceWithAdapter(nil)
	_, err := service.ConnectChannel(context.Background(), ConnectChannelCommand{
		TenantID: "tenant-1",
		SellerID: "seller-1",
		Type:     "unknown",
		Name:     "Test",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported channel type")
}

func TestConnectChannelInvalidCredentials(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeShopify,
		validateFn: func(context.Context, domain.ChannelCredentials) error {
			return errors.New("bad creds")
		},
	}
	service, _, _, _ := newServiceWithAdapter(adapter)
	_, err := service.ConnectChannel(context.Background(), ConnectChannelCommand{
		TenantID: "tenant-1",
		SellerID: "seller-1",
		Type:     string(domain.ChannelTypeShopify),
		Name:     "Test",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid credentials")
}

func TestConnectChannelSuccessWithWebhookWarning(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeShopify,
		validateFn: func(context.Context, domain.ChannelCredentials) error {
			return nil
		},
		registerWebhooksFn: func(context.Context, *domain.Channel, string) error {
			return errors.New("webhook down")
		},
	}
	service, channelRepo, _, _ := newServiceWithAdapter(adapter)
	var saved *domain.Channel
	channelRepo.saveFn = func(ctx context.Context, channel *domain.Channel) error {
		saved = channel
		return nil
	}

	dto, err := service.ConnectChannel(context.Background(), ConnectChannelCommand{
		TenantID:   "tenant-1",
		SellerID:   "seller-1",
		Type:       string(domain.ChannelTypeShopify),
		Name:       "Test",
		WebhookURL: "https://example.com/webhook",
	})
	require.NoError(t, err)
	require.NotNil(t, saved)
	require.Equal(t, saved.ChannelID, dto.ID)
	require.Equal(t, "shopify", dto.Type)
}

func TestConnectChannelSaveError(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeShopify,
		validateFn: func(context.Context, domain.ChannelCredentials) error {
			return nil
		},
	}
	service, channelRepo, _, _ := newServiceWithAdapter(adapter)
	channelRepo.saveFn = func(context.Context, *domain.Channel) error {
		return errors.New("save failed")
	}

	_, err := service.ConnectChannel(context.Background(), ConnectChannelCommand{
		TenantID: "tenant-1",
		SellerID: "seller-1",
		Type:     string(domain.ChannelTypeShopify),
		Name:     "Test",
	})
	require.Error(t, err)
}

func TestGetChannel(t *testing.T) {
	service, channelRepo, _, _ := newServiceWithAdapter(nil)
	channel := newTestChannel(t, domain.ChannelTypeAmazon)
	channelRepo.findByIDFn = func(ctx context.Context, channelID string) (*domain.Channel, error) {
		require.Equal(t, channel.ChannelID, channelID)
		return channel, nil
	}

	dto, err := service.GetChannel(context.Background(), channel.ChannelID)
	require.NoError(t, err)
	require.Equal(t, channel.ChannelID, dto.ID)
	require.Equal(t, string(channel.Type), dto.Type)
}

func TestGetChannelError(t *testing.T) {
	service, channelRepo, _, _ := newServiceWithAdapter(nil)
	channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return nil, errors.New("not found")
	}

	_, err := service.GetChannel(context.Background(), "missing")
	require.Error(t, err)
}

func TestGetChannelsBySeller(t *testing.T) {
	service, channelRepo, _, _ := newServiceWithAdapter(nil)
	ch1 := newTestChannel(t, domain.ChannelTypeAmazon)
	ch2 := newTestChannel(t, domain.ChannelTypeShopify)
	channelRepo.findBySellerIDFn = func(ctx context.Context, sellerID string) ([]*domain.Channel, error) {
		require.Equal(t, "seller-1", sellerID)
		return []*domain.Channel{ch1, ch2}, nil
	}

	dtos, err := service.GetChannelsBySeller(context.Background(), "seller-1")
	require.NoError(t, err)
	require.Len(t, dtos, 2)
	require.Equal(t, ch2.ChannelID, dtos[1].ID)
}

func TestGetChannelsBySellerError(t *testing.T) {
	service, channelRepo, _, _ := newServiceWithAdapter(nil)
	channelRepo.findBySellerIDFn = func(context.Context, string) ([]*domain.Channel, error) {
		return nil, errors.New("fail")
	}

	_, err := service.GetChannelsBySeller(context.Background(), "seller-1")
	require.Error(t, err)
}

func TestUpdateChannel(t *testing.T) {
	service, channelRepo, _, _ := newServiceWithAdapter(nil)
	channel := newTestChannel(t, domain.ChannelTypeEbay)
	channelRepo.findByIDFn = func(ctx context.Context, channelID string) (*domain.Channel, error) {
		return channel, nil
	}
	var saved *domain.Channel
	channelRepo.saveFn = func(ctx context.Context, channel *domain.Channel) error {
		saved = channel
		return nil
	}

	syncSettings := domain.SyncSettings{AutoImportOrders: false}
	dto, err := service.UpdateChannel(context.Background(), channel.ChannelID, UpdateChannelCommand{
		Name:         "Renamed",
		SyncSettings: &syncSettings,
		Metadata:     map[string]interface{}{"key": "value"},
	})
	require.NoError(t, err)
	require.NotNil(t, saved)
	require.Equal(t, "Renamed", saved.Name)
	require.False(t, saved.SyncSettings.AutoImportOrders)
	require.Equal(t, "value", saved.Metadata["key"])
	require.Equal(t, saved.ChannelID, dto.ID)
}

func TestUpdateChannelSaveError(t *testing.T) {
	service, channelRepo, _, _ := newServiceWithAdapter(nil)
	channel := newTestChannel(t, domain.ChannelTypeEbay)
	channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}
	channelRepo.saveFn = func(context.Context, *domain.Channel) error {
		return errors.New("save failed")
	}

	_, err := service.UpdateChannel(context.Background(), channel.ChannelID, UpdateChannelCommand{Name: "new"})
	require.Error(t, err)
}

func TestDisconnectChannel(t *testing.T) {
	service, channelRepo, _, _ := newServiceWithAdapter(nil)
	channel := newTestChannel(t, domain.ChannelTypeWooCommerce)
	channelRepo.findByIDFn = func(ctx context.Context, channelID string) (*domain.Channel, error) {
		return channel, nil
	}
	var saved *domain.Channel
	channelRepo.saveFn = func(ctx context.Context, channel *domain.Channel) error {
		saved = channel
		return nil
	}

	err := service.DisconnectChannel(context.Background(), channel.ChannelID)
	require.NoError(t, err)
	require.NotNil(t, saved)
	require.Equal(t, domain.ChannelStatusDisconnected, saved.Status)
}

func TestDisconnectChannelFindError(t *testing.T) {
	service, channelRepo, _, _ := newServiceWithAdapter(nil)
	channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return nil, errors.New("not found")
	}

	err := service.DisconnectChannel(context.Background(), "missing")
	require.Error(t, err)
}

func TestDisconnectChannelSaveError(t *testing.T) {
	service, channelRepo, _, _ := newServiceWithAdapter(nil)
	channel := newTestChannel(t, domain.ChannelTypeWooCommerce)
	channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}
	channelRepo.saveFn = func(context.Context, *domain.Channel) error {
		return errors.New("save failed")
	}

	err := service.DisconnectChannel(context.Background(), channel.ChannelID)
	require.Error(t, err)
}

func TestSyncOrdersAlreadyRunning(t *testing.T) {
	service, channelRepo, _, syncRepo := newServiceWithAdapter(nil)
	channel := newTestChannel(t, domain.ChannelTypeAmazon)
	channelRepo.findByIDFn = func(ctx context.Context, channelID string) (*domain.Channel, error) {
		return channel, nil
	}
	syncRepo.findRunningFn = func(ctx context.Context, channelID string, syncType domain.SyncType) (*domain.SyncJob, error) {
		return domain.NewSyncJob(channel.TenantID, channel.SellerID, channel.ChannelID, domain.SyncTypeOrders, "inbound"), nil
	}

	_, err := service.SyncOrders(context.Background(), SyncOrdersCommand{ChannelID: channel.ChannelID})
	require.Error(t, err)
	require.Contains(t, err.Error(), "order sync already in progress")
}

func TestSyncOrdersFindRunningError(t *testing.T) {
	service, channelRepo, _, syncRepo := newServiceWithAdapter(nil)
	channel := newTestChannel(t, domain.ChannelTypeAmazon)
	channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}
	syncRepo.findRunningFn = func(context.Context, string, domain.SyncType) (*domain.SyncJob, error) {
		return nil, errors.New("fail")
	}

	_, err := service.SyncOrders(context.Background(), SyncOrdersCommand{ChannelID: channel.ChannelID})
	require.Error(t, err)
}

func TestSyncOrdersFetchError(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeAmazon,
		fetchOrdersFn: func(context.Context, *domain.Channel, time.Time) ([]*domain.ChannelOrder, error) {
			return nil, errors.New("fetch failed")
		},
	}
	service, channelRepo, _, syncRepo := newServiceWithAdapter(adapter)
	channel := newTestChannel(t, domain.ChannelTypeAmazon)
	channelRepo.findByIDFn = func(ctx context.Context, channelID string) (*domain.Channel, error) {
		return channel, nil
	}
	syncRepo.findRunningFn = func(context.Context, string, domain.SyncType) (*domain.SyncJob, error) {
		return nil, nil
	}
	saveCount := 0
	syncRepo.saveFn = func(ctx context.Context, job *domain.SyncJob) error {
		saveCount++
		return nil
	}

	_, err := service.SyncOrders(context.Background(), SyncOrdersCommand{ChannelID: channel.ChannelID})
	require.Error(t, err)
	require.Equal(t, 2, saveCount)
}

func TestSyncOrdersSaveAllError(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeShopify,
		fetchOrdersFn: func(context.Context, *domain.Channel, time.Time) ([]*domain.ChannelOrder, error) {
			return []*domain.ChannelOrder{{ExternalOrderID: "ext-1"}}, nil
		},
	}
	service, channelRepo, orderRepo, syncRepo := newServiceWithAdapter(adapter)
	channel := newTestChannel(t, domain.ChannelTypeShopify)
	channelRepo.findByIDFn = func(ctx context.Context, channelID string) (*domain.Channel, error) {
		return channel, nil
	}
	syncRepo.findRunningFn = func(context.Context, string, domain.SyncType) (*domain.SyncJob, error) {
		return nil, nil
	}
	syncRepo.saveFn = func(context.Context, *domain.SyncJob) error { return nil }
	orderRepo.findByExternalIDFn = func(context.Context, string, string) (*domain.ChannelOrder, error) {
		return nil, nil
	}
	orderRepo.saveAllFn = func(context.Context, []*domain.ChannelOrder) error {
		return errors.New("save failed")
	}

	_, err := service.SyncOrders(context.Background(), SyncOrdersCommand{ChannelID: channel.ChannelID})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to save orders")
}

func TestSyncOrdersSuccess(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeShopify,
		fetchOrdersFn: func(context.Context, *domain.Channel, time.Time) ([]*domain.ChannelOrder, error) {
			return []*domain.ChannelOrder{
				{ExternalOrderID: "ext-1"},
				{ExternalOrderID: "ext-2"},
			}, nil
		},
	}
	service, channelRepo, orderRepo, syncRepo := newServiceWithAdapter(adapter)
	channel := newTestChannel(t, domain.ChannelTypeShopify)
	channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}
	channelRepo.saveFn = func(context.Context, *domain.Channel) error { return nil }
	syncRepo.findRunningFn = func(context.Context, string, domain.SyncType) (*domain.SyncJob, error) {
		return nil, nil
	}
	syncRepo.saveFn = func(context.Context, *domain.SyncJob) error { return nil }
	orderRepo.findByExternalIDFn = func(ctx context.Context, channelID, externalOrderID string) (*domain.ChannelOrder, error) {
		if externalOrderID == "ext-1" {
			return &domain.ChannelOrder{}, nil
		}
		return nil, nil
	}
	var saved []*domain.ChannelOrder
	orderRepo.saveAllFn = func(ctx context.Context, orders []*domain.ChannelOrder) error {
		saved = orders
		return nil
	}

	dto, err := service.SyncOrders(context.Background(), SyncOrdersCommand{ChannelID: channel.ChannelID})
	require.NoError(t, err)
	require.Len(t, saved, 1)
	require.Equal(t, "ext-2", saved[0].ExternalOrderID)
	require.Equal(t, string(domain.SyncStatusCompleted), dto.Status)
	require.NotNil(t, channel.LastOrderSync)
}

func TestSyncInventoryError(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeAmazon,
		syncInventoryFn: func(context.Context, *domain.Channel, []domain.InventoryUpdate) error {
			return errors.New("sync failed")
		},
	}
	service, channelRepo, _, syncRepo := newServiceWithAdapter(adapter)
	channel := newTestChannel(t, domain.ChannelTypeAmazon)
	channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}
	syncRepo.saveFn = func(context.Context, *domain.SyncJob) error { return nil }

	_, err := service.SyncInventory(context.Background(), SyncInventoryCommand{
		ChannelID: channel.ChannelID,
		Items:     []domain.InventoryUpdate{{SKU: "sku-1"}},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to sync inventory")
}

func TestSyncInventoryAdapterError(t *testing.T) {
	service, channelRepo, _, syncRepo := newServiceWithAdapter(nil)
	channel := newTestChannel(t, domain.ChannelTypeAmazon)
	channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}
	syncRepo.saveFn = func(context.Context, *domain.SyncJob) error { return nil }

	_, err := service.SyncInventory(context.Background(), SyncInventoryCommand{
		ChannelID: channel.ChannelID,
		Items:     []domain.InventoryUpdate{{SKU: "sku-1"}},
	})
	require.Error(t, err)
}

func TestSyncInventorySuccess(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeAmazon,
		syncInventoryFn: func(context.Context, *domain.Channel, []domain.InventoryUpdate) error {
			return nil
		},
	}
	service, channelRepo, _, syncRepo := newServiceWithAdapter(adapter)
	channel := newTestChannel(t, domain.ChannelTypeAmazon)
	channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}
	channelRepo.saveFn = func(context.Context, *domain.Channel) error { return nil }
	syncRepo.saveFn = func(context.Context, *domain.SyncJob) error { return nil }

	dto, err := service.SyncInventory(context.Background(), SyncInventoryCommand{
		ChannelID: channel.ChannelID,
		Items:     []domain.InventoryUpdate{{SKU: "sku-1"}, {SKU: "sku-2"}},
	})
	require.NoError(t, err)
	require.Equal(t, 2, dto.TotalItems)
	require.NotNil(t, channel.LastInventorySync)
}

func TestPushTrackingError(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeEbay,
		pushTrackingFn: func(context.Context, *domain.Channel, string, domain.TrackingInfo) error {
			return errors.New("push failed")
		},
	}
	service, channelRepo, _, _ := newServiceWithAdapter(adapter)
	channel := newTestChannel(t, domain.ChannelTypeEbay)
	channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}

	err := service.PushTracking(context.Background(), PushTrackingCommand{
		ChannelID:       channel.ChannelID,
		ExternalOrderID: "ext-1",
		TrackingNumber:  "track-1",
		Carrier:         "carrier",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to push tracking")
}

func TestPushTrackingAdapterError(t *testing.T) {
	service, channelRepo, _, _ := newServiceWithAdapter(nil)
	channel := newTestChannel(t, domain.ChannelTypeEbay)
	channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}

	err := service.PushTracking(context.Background(), PushTrackingCommand{
		ChannelID:       channel.ChannelID,
		ExternalOrderID: "ext-1",
		TrackingNumber:  "track-1",
		Carrier:         "carrier",
	})
	require.Error(t, err)
}

func TestPushTrackingMarkErrorIgnored(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeEbay,
		pushTrackingFn: func(context.Context, *domain.Channel, string, domain.TrackingInfo) error {
			return nil
		},
	}
	service, channelRepo, orderRepo, _ := newServiceWithAdapter(adapter)
	channel := newTestChannel(t, domain.ChannelTypeEbay)
	channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}
	orderRepo.markTrackingFn = func(context.Context, string) error {
		return errors.New("mark failed")
	}

	err := service.PushTracking(context.Background(), PushTrackingCommand{
		ChannelID:       channel.ChannelID,
		ExternalOrderID: "ext-1",
		TrackingNumber:  "track-1",
		Carrier:         "carrier",
	})
	require.NoError(t, err)
}

func TestCreateFulfillmentError(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeWooCommerce,
		createFulfillmentFn: func(context.Context, *domain.Channel, domain.FulfillmentRequest) error {
			return errors.New("fulfillment failed")
		},
	}
	service, channelRepo, _, _ := newServiceWithAdapter(adapter)
	channel := newTestChannel(t, domain.ChannelTypeWooCommerce)
	channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}

	err := service.CreateFulfillment(context.Background(), CreateFulfillmentCommand{
		ChannelID:       channel.ChannelID,
		ExternalOrderID: "ext-1",
		TrackingNumber:  "track-1",
		Carrier:         "carrier",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create fulfillment")
}

func TestCreateFulfillmentAdapterError(t *testing.T) {
	service, channelRepo, _, _ := newServiceWithAdapter(nil)
	channel := newTestChannel(t, domain.ChannelTypeWooCommerce)
	channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}

	err := service.CreateFulfillment(context.Background(), CreateFulfillmentCommand{
		ChannelID:       channel.ChannelID,
		ExternalOrderID: "ext-1",
		TrackingNumber:  "track-1",
		Carrier:         "carrier",
	})
	require.Error(t, err)
}

func TestCreateFulfillmentMarkErrorIgnored(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeWooCommerce,
		createFulfillmentFn: func(context.Context, *domain.Channel, domain.FulfillmentRequest) error {
			return nil
		},
	}
	service, channelRepo, orderRepo, _ := newServiceWithAdapter(adapter)
	channel := newTestChannel(t, domain.ChannelTypeWooCommerce)
	channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}
	orderRepo.markTrackingFn = func(context.Context, string) error {
		return errors.New("mark failed")
	}

	err := service.CreateFulfillment(context.Background(), CreateFulfillmentCommand{
		ChannelID:       channel.ChannelID,
		ExternalOrderID: "ext-1",
		TrackingNumber:  "track-1",
		Carrier:         "carrier",
	})
	require.NoError(t, err)
}

func TestImportOrder(t *testing.T) {
	service, _, orderRepo, _ := newServiceWithAdapter(nil)
	var gotExternal string
	var gotWMS string
	orderRepo.markImportedFn = func(ctx context.Context, externalOrderID, wmsOrderID string) error {
		gotExternal = externalOrderID
		gotWMS = wmsOrderID
		return nil
	}

	err := service.ImportOrder(context.Background(), ImportOrderCommand{
		ChannelID:       "ch-1",
		ExternalOrderID: "ext-1",
		WMSOrderID:      "wms-1",
	})
	require.NoError(t, err)
	require.Equal(t, "ext-1", gotExternal)
	require.Equal(t, "wms-1", gotWMS)
}

func TestGetChannelOrdersDefaults(t *testing.T) {
	service, _, orderRepo, _ := newServiceWithAdapter(nil)
	var gotPagination domain.Pagination
	orderRepo.findByChannelIDFn = func(ctx context.Context, channelID string, pagination domain.Pagination) ([]*domain.ChannelOrder, error) {
		gotPagination = pagination
		return []*domain.ChannelOrder{}, nil
	}

	_, err := service.GetChannelOrders(context.Background(), "ch-1", 0, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), gotPagination.Page)
	require.Equal(t, int64(20), gotPagination.PageSize)
}

func TestGetChannelOrdersError(t *testing.T) {
	service, _, orderRepo, _ := newServiceWithAdapter(nil)
	orderRepo.findByChannelIDFn = func(context.Context, string, domain.Pagination) ([]*domain.ChannelOrder, error) {
		return nil, errors.New("fail")
	}

	_, err := service.GetChannelOrders(context.Background(), "ch-1", 1, 10)
	require.Error(t, err)
}

func TestGetUnimportedOrders(t *testing.T) {
	service, _, orderRepo, _ := newServiceWithAdapter(nil)
	orderRepo.findUnimportedFn = func(ctx context.Context, channelID string) ([]*domain.ChannelOrder, error) {
		return []*domain.ChannelOrder{{ExternalOrderID: "ext-1"}}, nil
	}

	dtos, err := service.GetUnimportedOrders(context.Background(), "ch-1")
	require.NoError(t, err)
	require.Len(t, dtos, 1)
	require.Equal(t, "ext-1", dtos[0].ExternalOrderID)
}

func TestGetUnimportedOrdersError(t *testing.T) {
	service, _, orderRepo, _ := newServiceWithAdapter(nil)
	orderRepo.findUnimportedFn = func(context.Context, string) ([]*domain.ChannelOrder, error) {
		return nil, errors.New("fail")
	}

	_, err := service.GetUnimportedOrders(context.Background(), "ch-1")
	require.Error(t, err)
}

func TestGetSyncJobsDefaults(t *testing.T) {
	service, _, _, syncRepo := newServiceWithAdapter(nil)
	var gotPagination domain.Pagination
	syncRepo.findByChannelFn = func(ctx context.Context, channelID string, pagination domain.Pagination) ([]*domain.SyncJob, error) {
		gotPagination = pagination
		return []*domain.SyncJob{}, nil
	}

	_, err := service.GetSyncJobs(context.Background(), "ch-1", 0, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), gotPagination.Page)
	require.Equal(t, int64(20), gotPagination.PageSize)
}

func TestGetSyncJobsError(t *testing.T) {
	service, _, _, syncRepo := newServiceWithAdapter(nil)
	syncRepo.findByChannelFn = func(context.Context, string, domain.Pagination) ([]*domain.SyncJob, error) {
		return nil, errors.New("fail")
	}

	_, err := service.GetSyncJobs(context.Background(), "ch-1", 1, 10)
	require.Error(t, err)
}

func TestHandleWebhookInvalidSignature(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeCustom,
		validateWebhookFn: func(context.Context, *domain.Channel, string, []byte) bool {
			return false
		},
	}
	service, channelRepo, _, _ := newServiceWithAdapter(adapter)
	channel := newTestChannel(t, domain.ChannelTypeCustom)
	channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}

	err := service.HandleWebhook(context.Background(), WebhookCommand{
		ChannelID: channel.ChannelID,
		Signature: "sig",
		Body:      []byte("payload"),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid webhook signature")
}

func TestHandleWebhookChannelError(t *testing.T) {
	service, channelRepo, _, _ := newServiceWithAdapter(nil)
	channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return nil, errors.New("not found")
	}

	err := service.HandleWebhook(context.Background(), WebhookCommand{
		ChannelID: "missing",
	})
	require.Error(t, err)
}

func TestHandleWebhookValidSignature(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeCustom,
		validateWebhookFn: func(context.Context, *domain.Channel, string, []byte) bool {
			return true
		},
	}
	service, channelRepo, _, _ := newServiceWithAdapter(adapter)
	channel := newTestChannel(t, domain.ChannelTypeCustom)
	channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}

	err := service.HandleWebhook(context.Background(), WebhookCommand{
		ChannelID: channel.ChannelID,
		Signature: "sig",
		Body:      []byte("payload"),
	})
	require.NoError(t, err)
}

func TestGetInventoryLevels(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeAmazon,
		getInventoryLevelsFn: func(context.Context, *domain.Channel, []string) ([]domain.InventoryLevel, error) {
			return []domain.InventoryLevel{{SKU: "sku-1"}}, nil
		},
	}
	service, channelRepo, _, _ := newServiceWithAdapter(adapter)
	channel := newTestChannel(t, domain.ChannelTypeAmazon)
	channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}

	levels, err := service.GetInventoryLevels(context.Background(), channel.ChannelID, []string{"sku-1"})
	require.NoError(t, err)
	require.Len(t, levels, 1)
	require.Equal(t, "sku-1", levels[0].SKU)
}

func TestGetInventoryLevelsChannelError(t *testing.T) {
	service, channelRepo, _, _ := newServiceWithAdapter(nil)
	channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return nil, errors.New("not found")
	}

	_, err := service.GetInventoryLevels(context.Background(), "missing", []string{"sku-1"})
	require.Error(t, err)
}

func TestGetInventoryLevelsAdapterError(t *testing.T) {
	adapter := &fakeAdapter{
		channelType: domain.ChannelTypeAmazon,
		getInventoryLevelsFn: func(context.Context, *domain.Channel, []string) ([]domain.InventoryLevel, error) {
			return nil, errors.New("fail")
		},
	}
	service, channelRepo, _, _ := newServiceWithAdapter(adapter)
	channel := newTestChannel(t, domain.ChannelTypeAmazon)
	channelRepo.findByIDFn = func(context.Context, string) (*domain.Channel, error) {
		return channel, nil
	}

	_, err := service.GetInventoryLevels(context.Background(), channel.ChannelID, []string{"sku-1"})
	require.Error(t, err)
}

func TestGetSyncSettingsValue(t *testing.T) {
	cmd := UpdateChannelCommand{}
	require.Equal(t, domain.SyncSettings{}, cmd.GetSyncSettingsValue())

	settings := domain.SyncSettings{AutoSyncInventory: true}
	cmd.SyncSettings = &settings
	require.Equal(t, settings, cmd.GetSyncSettingsValue())
}

func TestToChannelDTO(t *testing.T) {
	channel := newTestChannel(t, domain.ChannelTypeAmazon)
	channel.LastError = "boom"
	lastSync := time.Now().UTC()
	channel.LastOrderSync = &lastSync
	dto := ToChannelDTO(channel)
	require.Equal(t, channel.ChannelID, dto.ID)
	require.Equal(t, "amazon", dto.Type)
	require.Equal(t, "boom", dto.SyncSettings.OrderSync.LastError)
	require.NotNil(t, dto.SyncSettings.OrderSync.LastSyncAt)
}

func TestToChannelOrderDTO(t *testing.T) {
	now := time.Now().UTC()
	order := &domain.ChannelOrder{
		ID:                primitive.NewObjectID(),
		ChannelID:         "ch-1",
		ExternalOrderID:   "ext-1",
		ExternalOrderNumber: "1001",
		FulfillmentStatus: "fulfilled",
		Total:             10.5,
		Currency:          "USD",
		Imported:          true,
		WMSOrderID:        "wms-1",
		TrackingPushed:    true,
		ExternalCreatedAt: now,
		ImportedAt:        &now,
		Customer: domain.ChannelCustomer{
			ExternalID: "cust-1",
			Email:      "cust@example.com",
			FirstName:  "Ada",
			LastName:   "Lovelace",
		},
		ShippingAddr: domain.ChannelAddress{
			FirstName: "Ada",
			LastName:  "Lovelace",
			Address1:  "Main",
			City:      "City",
			Province:  "ST",
			Zip:       "12345",
			Country:   "US",
		},
		LineItems: []domain.ChannelLineItem{
			{
				ExternalID: "line-1",
				SKU:        "sku-1",
				Title:      "Item",
				Quantity:   2,
				Price:      3.5,
			},
		},
	}
	dto := ToChannelOrderDTO(order)
	require.Equal(t, "ext-1", dto.ExternalOrderID)
	require.Equal(t, 7.0, dto.LineItems[0].TotalPrice)
	require.Equal(t, "Ada Lovelace", dto.ShippingAddress.Name)
}

func TestToSyncJobDTO(t *testing.T) {
	job := domain.NewSyncJob("tenant-1", "seller-1", "ch-1", domain.SyncTypeOrders, "inbound")
	job.Start()
	job.Fail("boom")
	dto := ToSyncJobDTO(job)
	require.Equal(t, job.JobID, dto.ID)
	require.Equal(t, "boom", dto.Error)
	require.NotZero(t, dto.StartedAt)
	require.NotNil(t, dto.CompletedAt)
}

func TestPaginationDefaults(t *testing.T) {
	pagination := domain.DefaultPagination()
	require.Equal(t, int64(1), pagination.Page)
	require.Equal(t, int64(20), pagination.PageSize)
	require.Equal(t, int64(0), pagination.Skip())
	require.Equal(t, int64(20), pagination.Limit())
}
