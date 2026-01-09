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

func TestOrderServiceConsumer(t *testing.T) {
	ensurePactDir(t)

	t.Run("GetOrder", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "order-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		orderID := "ord-123456"

		err = mockProvider.
			AddInteraction().
			Given("an order exists").
			UponReceiving("a request to get an order").
			WithRequest(http.MethodGet, fmt.Sprintf("/api/v1/orders/%s", orderID)).
			WithHeader("Accept", matchers.String("application/json")).
			WillRespondWith(http.StatusOK).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"id":         matchers.String(orderID),
				"customerId": matchers.String("cust-001"),
				"status":     matchers.String("pending"),
				"priority":   matchers.String("standard"),
				"totalItems": matchers.Integer(5),
				"totalValue": matchers.Decimal(150.50),
				"createdAt":  TimestampMatcher(),
				"updatedAt":  TimestampMatcher(),
			}).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					OrderServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				order, err := client.GetOrder(context.Background(), orderID)
				if err != nil {
					return err
				}

				assert.Equal(t, orderID, order.ID)
				assert.NotEmpty(t, order.CustomerID)
				return nil
			})

		require.NoError(t, err)
	})

	t.Run("ValidateOrder", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "order-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		orderID := "ord-123456"

		err = mockProvider.
			AddInteraction().
			Given("an order exists and is valid").
			UponReceiving("a request to validate an order").
			WithRequest(http.MethodPost, fmt.Sprintf("/api/v1/orders/%s/validate", orderID)).
			WithHeader("Accept", matchers.String("application/json")).
			WillRespondWith(http.StatusOK).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"valid":   matchers.Boolean(true),
				"orderId": matchers.String(orderID),
				"errors":  matchers.EachLike(matchers.String(""), 0),
			}).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					OrderServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				result, err := client.ValidateOrder(context.Background(), orderID)
				if err != nil {
					return err
				}

				assert.True(t, result.Valid)
				assert.Equal(t, orderID, result.OrderID)
				return nil
			})

		require.NoError(t, err)
	})

	t.Run("CancelOrder", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "order-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		orderID := "ord-123456"

		err = mockProvider.
			AddInteraction().
			Given("an order exists and can be cancelled").
			UponReceiving("a request to cancel an order").
			WithRequest(http.MethodPost, fmt.Sprintf("/api/v1/orders/%s/cancel", orderID)).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithHeader("Accept", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"reason": matchers.String("customer request"),
			}).
			WillRespondWith(http.StatusOK).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					OrderServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				return client.CancelOrder(context.Background(), orderID, "customer request")
			})

		require.NoError(t, err)
	})
}
