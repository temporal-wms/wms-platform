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

func TestConsolidationServiceConsumer(t *testing.T) {
	ensurePactDir(t)

	t.Run("CreateConsolidation", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "consolidation-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		err = mockProvider.
			AddInteraction().
			Given("picking is complete").
			UponReceiving("a request to create a consolidation unit").
			WithRequest(http.MethodPost, "/api/v1/consolidations").
			WithHeader("Content-Type", matchers.String("application/json")).
			WithHeader("Accept", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"orderId": matchers.String("ord-123456"),
				"station": matchers.String("CONSOL-01"),
				"expectedItems": matchers.EachLike(matchers.Map{
					"sku":      matchers.String("SKU-001"),
					"quantity": matchers.Integer(2),
				}, 1),
			}).
			WillRespondWith(http.StatusCreated).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"id":            UUIDMatcher(),
				"orderId":       matchers.String("ord-123456"),
				"status":        matchers.String("pending"),
				"expectedItems": matchers.Integer(1),
				"scannedItems":  matchers.Integer(0),
				"station":       matchers.String("CONSOL-01"),
			}).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					ConsolidationServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				unit, err := client.CreateConsolidation(context.Background(), &clients.CreateConsolidationRequest{
					OrderID: "ord-123456",
					Station: "CONSOL-01",
					ExpectedItems: []clients.ExpectedItem{
						{SKU: "SKU-001", Quantity: 2},
					},
				})
				if err != nil {
					return err
				}

				assert.NotEmpty(t, unit.ID)
				assert.Equal(t, "pending", unit.Status)
				return nil
			})

		require.NoError(t, err)
	})

	t.Run("GetConsolidation", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "consolidation-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		consolidationID := "consol-123456"

		err = mockProvider.
			AddInteraction().
			Given("a consolidation unit exists").
			UponReceiving("a request to get a consolidation unit").
			WithRequest(http.MethodGet, fmt.Sprintf("/api/v1/consolidations/%s", consolidationID)).
			WithHeader("Accept", matchers.String("application/json")).
			WillRespondWith(http.StatusOK).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"id":            matchers.String(consolidationID),
				"orderId":       matchers.String("ord-123456"),
				"status":        matchers.String("in_progress"),
				"expectedItems": matchers.Integer(5),
				"scannedItems":  matchers.Integer(3),
				"station":       matchers.String("CONSOL-01"),
			}).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					ConsolidationServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				unit, err := client.GetConsolidation(context.Background(), consolidationID)
				if err != nil {
					return err
				}

				assert.Equal(t, consolidationID, unit.ID)
				return nil
			})

		require.NoError(t, err)
	})

	t.Run("CompleteConsolidation", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "consolidation-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		consolidationID := "consol-123456"

		err = mockProvider.
			AddInteraction().
			Given("consolidation is in progress with all items scanned").
			UponReceiving("a request to complete consolidation").
			WithRequest(http.MethodPost, fmt.Sprintf("/api/v1/consolidations/%s/complete", consolidationID)).
			WithHeader("Accept", matchers.String("application/json")).
			WillRespondWith(http.StatusOK).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"id":            matchers.String(consolidationID),
				"orderId":       matchers.String("ord-123456"),
				"status":        matchers.String("completed"),
				"expectedItems": matchers.Integer(5),
				"scannedItems":  matchers.Integer(5),
				"station":       matchers.String("CONSOL-01"),
			}).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					ConsolidationServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				unit, err := client.CompleteConsolidation(context.Background(), consolidationID)
				if err != nil {
					return err
				}

				assert.Equal(t, "completed", unit.Status)
				return nil
			})

		require.NoError(t, err)
	})
}
