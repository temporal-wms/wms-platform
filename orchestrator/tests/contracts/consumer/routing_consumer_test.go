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

func TestRoutingServiceConsumer(t *testing.T) {
	ensurePactDir(t)

	t.Run("CalculateRoute", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "routing-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		err = mockProvider.
			AddInteraction().
			Given("pick locations exist").
			UponReceiving("a request to calculate route").
			WithRequest(http.MethodPost, "/api/v1/routes").
			WithHeader("Content-Type", matchers.String("application/json")).
			WithHeader("Accept", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"waveId": matchers.String("wave-001"),
				"locations": matchers.EachLike(matchers.Map{
					"locationId": matchers.String("LOC-A01"),
					"zone":       matchers.String("ZONE-A"),
					"aisle":      matchers.String("01"),
					"rack":       matchers.String("01"),
					"level":      matchers.String("A"),
				}, 1),
			}).
			WillRespondWith(http.StatusOK).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"id":            UUIDMatcher(),
				"stops":         matchers.EachLike(matchers.String("LOC-A01"), 1),
				"totalDistance": matchers.Decimal(150.5),
				"estimatedTime": matchers.Integer(1800),
				"status":        matchers.String("calculated"),
			}).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					RoutingServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				route, err := client.CalculateRoute(context.Background(), &clients.CalculateRouteRequest{
					WaveID: "wave-001",
					Locations: []clients.Location{
						{LocationID: "LOC-A01", Zone: "ZONE-A", Aisle: "01", Rack: "01", Level: "A"},
					},
				})
				if err != nil {
					return err
				}

				assert.NotEmpty(t, route.ID)
				assert.Equal(t, "calculated", route.Status)
				return nil
			})

		require.NoError(t, err)
	})

	t.Run("GetRoute", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "routing-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		routeID := "route-123456"

		err = mockProvider.
			AddInteraction().
			Given("a route exists").
			UponReceiving("a request to get a route").
			WithRequest(http.MethodGet, fmt.Sprintf("/api/v1/routes/%s", routeID)).
			WithHeader("Accept", matchers.String("application/json")).
			WillRespondWith(http.StatusOK).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"id":            matchers.String(routeID),
				"stops":         matchers.EachLike(matchers.String("LOC-A01"), 1),
				"totalDistance": matchers.Decimal(150.5),
				"estimatedTime": matchers.Integer(1800),
				"status":        matchers.String("in_progress"),
			}).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					RoutingServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				route, err := client.GetRoute(context.Background(), routeID)
				if err != nil {
					return err
				}

				assert.Equal(t, routeID, route.ID)
				return nil
			})

		require.NoError(t, err)
	})
}
