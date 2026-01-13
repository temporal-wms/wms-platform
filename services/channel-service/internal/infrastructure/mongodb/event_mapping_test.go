package mongodb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wms-platform/services/channel-service/internal/domain"
	"github.com/wms-platform/shared/pkg/cloudevents"
)

type customEvent struct {
	when time.Time
}

func (e *customEvent) EventType() string    { return "custom.event" }
func (e *customEvent) OccurredAt() time.Time { return e.when }

func TestDomainEventToCloudEvent(t *testing.T) {
	repo := &ChannelRepository{
		eventFactory: cloudevents.NewEventFactory(cloudevents.SourceChannel),
	}
	ctx := context.Background()
	channelID := "ch-1"

	tests := []struct {
		name     string
		event    domain.DomainEvent
		expected string
	}{
		{
			name:     "connected",
			event:    &domain.ChannelConnectedEvent{ConnectedAt: time.Now()},
			expected: cloudevents.ChannelConnected,
		},
		{
			name:     "disconnected",
			event:    &domain.ChannelDisconnectedEvent{DisconnectedAt: time.Now()},
			expected: cloudevents.ChannelDisconnected,
		},
		{
			name:     "imported",
			event:    &domain.OrderImportedEvent{ImportedAt: time.Now()},
			expected: cloudevents.ChannelOrderImported,
		},
		{
			name:     "tracking",
			event:    &domain.TrackingPushedEvent{PushedAt: time.Now()},
			expected: cloudevents.ChannelTrackingPushed,
		},
		{
			name:     "inventory",
			event:    &domain.InventorySyncedEvent{SyncedAt: time.Now()},
			expected: cloudevents.ChannelInventorySynced,
		},
		{
			name:     "sync-completed",
			event:    &domain.SyncCompletedEvent{CompletedAt: time.Now()},
			expected: cloudevents.ChannelSyncCompleted,
		},
		{
			name:     "webhook",
			event:    &domain.WebhookReceivedEvent{ReceivedAt: time.Now()},
			expected: cloudevents.ChannelWebhookReceived,
		},
		{
			name:     "default",
			event:    &customEvent{when: time.Now()},
			expected: "wms.channel.custom.event",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cloudEvent := repo.domainEventToCloudEvent(ctx, channelID, tt.event)
			require.Equal(t, tt.expected, cloudEvent.Type)
			require.Equal(t, "channel/"+channelID, cloudEvent.Subject)
		})
	}
}
