package domain

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeAdapter struct {
	channelType ChannelType
}

func (f *fakeAdapter) GetType() ChannelType {
	return f.channelType
}

func (f *fakeAdapter) ValidateCredentials(context.Context, ChannelCredentials) error {
	return nil
}

func (f *fakeAdapter) FetchOrders(context.Context, *Channel, time.Time) ([]*ChannelOrder, error) {
	return nil, nil
}

func (f *fakeAdapter) FetchOrder(context.Context, *Channel, string) (*ChannelOrder, error) {
	return nil, nil
}

func (f *fakeAdapter) PushTracking(context.Context, *Channel, string, TrackingInfo) error {
	return nil
}

func (f *fakeAdapter) SyncInventory(context.Context, *Channel, []InventoryUpdate) error {
	return nil
}

func (f *fakeAdapter) GetInventoryLevels(context.Context, *Channel, []string) ([]InventoryLevel, error) {
	return nil, nil
}

func (f *fakeAdapter) CreateFulfillment(context.Context, *Channel, FulfillmentRequest) error {
	return nil
}

func (f *fakeAdapter) RegisterWebhooks(context.Context, *Channel, string) error {
	return nil
}

func (f *fakeAdapter) ValidateWebhook(context.Context, *Channel, string, []byte) bool {
	return true
}

func TestAdapterFactory(t *testing.T) {
	factory := NewAdapterFactory()
	adapter := &fakeAdapter{channelType: ChannelTypeShopify}
	factory.Register(adapter)

	got, err := factory.GetAdapter(ChannelTypeShopify)
	require.NoError(t, err)
	require.Equal(t, adapter, got)

	channel := &Channel{Type: ChannelTypeShopify}
	got, err = factory.GetAdapterForChannel(channel)
	require.NoError(t, err)
	require.Equal(t, adapter, got)

	_, err = factory.GetAdapter(ChannelTypeCustom)
	require.ErrorIs(t, err, ErrInvalidChannelType)
}

func TestEvents(t *testing.T) {
	now := time.Now().UTC()

	connected := &ChannelConnectedEvent{ConnectedAt: now}
	require.Equal(t, "channel.connected", connected.EventType())
	require.Equal(t, now, connected.OccurredAt())

	disconnected := &ChannelDisconnectedEvent{DisconnectedAt: now}
	require.Equal(t, "channel.disconnected", disconnected.EventType())
	require.Equal(t, now, disconnected.OccurredAt())

	imported := &OrderImportedEvent{ImportedAt: now}
	require.Equal(t, "channel.order.imported", imported.EventType())
	require.Equal(t, now, imported.OccurredAt())

	tracking := &TrackingPushedEvent{PushedAt: now}
	require.Equal(t, "channel.tracking.pushed", tracking.EventType())
	require.Equal(t, now, tracking.OccurredAt())

	inventory := &InventorySyncedEvent{SyncedAt: now}
	require.Equal(t, "channel.inventory.synced", inventory.EventType())
	require.Equal(t, now, inventory.OccurredAt())

	sync := &SyncCompletedEvent{CompletedAt: now}
	require.Equal(t, "channel.sync.completed", sync.EventType())
	require.Equal(t, now, sync.OccurredAt())

	webhook := &WebhookReceivedEvent{ReceivedAt: now}
	require.Equal(t, "channel.webhook.received", webhook.EventType())
	require.Equal(t, now, webhook.OccurredAt())
}

func TestPagination(t *testing.T) {
	pagination := DefaultPagination()
	require.Equal(t, int64(1), pagination.Page)
	require.Equal(t, int64(20), pagination.PageSize)
	require.Equal(t, int64(0), pagination.Skip())
	require.Equal(t, int64(20), pagination.Limit())

	custom := Pagination{Page: 2, PageSize: 5}
	require.Equal(t, int64(5), custom.Skip())
	require.Equal(t, int64(5), custom.Limit())
}

func TestAdapterFactoryMissing(t *testing.T) {
	factory := NewAdapterFactory()
	_, err := factory.GetAdapter(ChannelTypeAmazon)
	require.True(t, errors.Is(err, ErrInvalidChannelType))
}
