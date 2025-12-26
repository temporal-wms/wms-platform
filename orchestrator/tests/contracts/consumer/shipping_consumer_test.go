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

func TestShippingServiceConsumer(t *testing.T) {
	ensurePactDir(t)

	t.Run("CreateShipment", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "shipping-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		err = mockProvider.
			AddInteraction().
			Given("packing is complete").
			UponReceiving("a request to create a shipment").
			WithRequest(http.MethodPost, "/api/v1/shipments").
			WithHeader("Content-Type", matchers.String("application/json")).
			WithHeader("Accept", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"orderId":   matchers.String("ord-123456"),
				"packageId": matchers.String("pkg-001"),
				"carrier":   matchers.String("UPS"),
				"address": matchers.Map{
					"street":  matchers.String("123 Main St"),
					"city":    matchers.String("New York"),
					"state":   matchers.String("NY"),
					"zipCode": matchers.String("10001"),
					"country": matchers.String("US"),
				},
			}).
			WillRespondWith(http.StatusCreated).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"id":        UUIDMatcher(),
				"orderId":   matchers.String("ord-123456"),
				"packageId": matchers.String("pkg-001"),
				"carrier":   matchers.String("UPS"),
				"status":    matchers.String("created"),
			}).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					ShippingServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				shipment, err := client.CreateShipment(context.Background(), &clients.CreateShipmentRequest{
					OrderID:   "ord-123456",
					PackageID: "pkg-001",
					Carrier:   "UPS",
					Address: clients.ShippingAddress{
						Street:  "123 Main St",
						City:    "New York",
						State:   "NY",
						ZipCode: "10001",
						Country: "US",
					},
				})
				if err != nil {
					return err
				}

				assert.NotEmpty(t, shipment.ID)
				assert.Equal(t, "created", shipment.Status)
				return nil
			})

		require.NoError(t, err)
	})

	t.Run("GetShipment", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "shipping-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		shipmentID := "ship-123456"

		err = mockProvider.
			AddInteraction().
			Given("a shipment exists").
			UponReceiving("a request to get a shipment").
			WithRequest(http.MethodGet, fmt.Sprintf("/api/v1/shipments/%s", shipmentID)).
			WithHeader("Accept", matchers.String("application/json")).
			WillRespondWith(http.StatusOK).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"id":           matchers.String(shipmentID),
				"orderId":      matchers.String("ord-123456"),
				"packageId":    matchers.String("pkg-001"),
				"carrier":      matchers.String("UPS"),
				"trackingCode": matchers.String("1Z999AA10123456784"),
				"status":       matchers.String("labeled"),
			}).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					ShippingServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				shipment, err := client.GetShipment(context.Background(), shipmentID)
				if err != nil {
					return err
				}

				assert.Equal(t, shipmentID, shipment.ID)
				return nil
			})

		require.NoError(t, err)
	})

	t.Run("GenerateLabel", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "shipping-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		shipmentID := "ship-123456"

		err = mockProvider.
			AddInteraction().
			Given("a shipment exists without a label").
			UponReceiving("a request to generate a shipping label").
			WithRequest(http.MethodPost, fmt.Sprintf("/api/v1/shipments/%s/label", shipmentID)).
			WithHeader("Accept", matchers.String("application/json")).
			WillRespondWith(http.StatusOK).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"id":           UUIDMatcher(),
				"shipmentId":   matchers.String(shipmentID),
				"carrier":      matchers.String("UPS"),
				"trackingCode": matchers.String("1Z999AA10123456784"),
				"labelUrl":     matchers.String("https://labels.example.com/label-123.pdf"),
				"format":       matchers.String("PDF"),
			}).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					ShippingServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				label, err := client.GenerateLabel(context.Background(), shipmentID)
				if err != nil {
					return err
				}

				assert.NotEmpty(t, label.ID)
				assert.NotEmpty(t, label.TrackingCode)
				return nil
			})

		require.NoError(t, err)
	})

	t.Run("MarkShipped", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "shipping-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		shipmentID := "ship-123456"

		err = mockProvider.
			AddInteraction().
			Given("a shipment has a label").
			UponReceiving("a request to mark shipment as shipped").
			WithRequest(http.MethodPost, fmt.Sprintf("/api/v1/shipments/%s/ship", shipmentID)).
			WithHeader("Accept", matchers.String("application/json")).
			WillRespondWith(http.StatusOK).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					ShippingServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				return client.MarkShipped(context.Background(), shipmentID)
			})

		require.NoError(t, err)
	})
}
