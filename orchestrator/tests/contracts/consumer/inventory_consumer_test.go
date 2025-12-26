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

func TestInventoryServiceConsumer(t *testing.T) {
	ensurePactDir(t)

	t.Run("ReserveInventory", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "inventory-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		err = mockProvider.
			AddInteraction().
			Given("inventory is available").
			UponReceiving("a request to reserve inventory").
			WithRequest(http.MethodPost, "/api/v1/inventory/reserve").
			WithHeader("Content-Type", matchers.String("application/json")).
			WithHeader("Accept", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"orderId": matchers.String("ord-123456"),
				"items": matchers.EachLike(matchers.Map{
					"sku":      matchers.String("SKU-001"),
					"quantity": matchers.Integer(2),
				}, 1),
			}).
			WillRespondWith(http.StatusOK).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					InventoryServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				return client.ReserveInventory(context.Background(), &clients.ReserveInventoryRequest{
					OrderID: "ord-123456",
					Items: []clients.InventoryReservationItem{
						{SKU: "SKU-001", Quantity: 2},
					},
				})
			})

		require.NoError(t, err)
	})

	t.Run("ReleaseInventoryReservation", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "inventory-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		orderID := "ord-123456"

		err = mockProvider.
			AddInteraction().
			Given("a reservation exists").
			UponReceiving("a request to release inventory reservation").
			WithRequest(http.MethodPost, fmt.Sprintf("/api/v1/inventory/release/%s", orderID)).
			WithHeader("Accept", matchers.String("application/json")).
			WillRespondWith(http.StatusOK).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					InventoryServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				return client.ReleaseInventoryReservation(context.Background(), orderID)
			})

		require.NoError(t, err)
	})

	t.Run("GetInventoryBySKU", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "inventory-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		sku := "SKU-001"

		err = mockProvider.
			AddInteraction().
			Given("inventory exists for SKU").
			UponReceiving("a request to get inventory by SKU").
			WithRequest(http.MethodGet, fmt.Sprintf("/api/v1/inventory/sku/%s", sku)).
			WithHeader("Accept", matchers.String("application/json")).
			WillRespondWith(http.StatusOK).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"sku":          matchers.String(sku),
				"availableQty": matchers.Integer(100),
				"reservedQty":  matchers.Integer(10),
				"location":     matchers.String("A-01-01"),
				"zone":         matchers.String("ZONE-A"),
				"lastUpdated":  TimestampMatcher(),
			}).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					InventoryServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				item, err := client.GetInventoryBySKU(context.Background(), sku)
				if err != nil {
					return err
				}

				assert.Equal(t, sku, item.SKU)
				assert.GreaterOrEqual(t, item.AvailableQty, 0)
				return nil
			})

		require.NoError(t, err)
	})
}
