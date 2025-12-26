package consumer_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	pact "github.com/pact-foundation/pact-go/v2"
	"github.com/pact-foundation/pact-go/v2/consumer"
	"github.com/pact-foundation/pact-go/v2/matchers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wms-platform/orchestrator/internal/activities/clients"
)

func TestWavingServiceConsumer(t *testing.T) {
	ensurePactDir(t)

	t.Run("GetWave", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "waving-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		waveID := "wave-123456"

		err = mockProvider.
			AddInteraction().
			Given("a wave exists").
			UponReceiving("a request to get a wave").
			WithRequest(http.MethodGet, fmt.Sprintf("/api/v1/waves/%s", waveID)).
			WithHeader("Accept", matchers.String("application/json")).
			WillRespondWith(http.StatusOK).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"id":         matchers.String(waveID),
				"status":     matchers.String("released"),
				"orderIds":   matchers.EachLike(matchers.String("ord-123456"), 1),
				"orderCount": matchers.Integer(5),
				"priority":   matchers.Integer(1),
				"createdAt":  TimestampMatcher(),
			}).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					WavingServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				wave, err := client.GetWave(context.Background(), waveID)
				if err != nil {
					return err
				}

				assert.Equal(t, waveID, wave.ID)
				assert.Equal(t, "released", wave.Status)
				return nil
			})

		require.NoError(t, err)
	})

	t.Run("AssignOrderToWave", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "waving-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		waveID := "wave-123456"
		orderID := "ord-123456"

		err = mockProvider.
			AddInteraction().
			Given("a wave exists and can accept orders").
			UponReceiving("a request to assign an order to a wave").
			WithRequest(http.MethodPost, fmt.Sprintf("/api/v1/waves/%s/orders", waveID)).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithHeader("Accept", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"orderId": matchers.String(orderID),
			}).
			WillRespondWith(http.StatusOK).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					WavingServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				return client.AssignOrderToWave(context.Background(), waveID, orderID)
			})

		require.NoError(t, err)
	})
}
