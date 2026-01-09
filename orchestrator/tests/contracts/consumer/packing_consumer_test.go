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

func TestPackingServiceConsumer(t *testing.T) {
	ensurePactDir(t)

	t.Run("CreatePackTask", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "packing-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		err = mockProvider.
			AddInteraction().
			Given("consolidation is complete").
			UponReceiving("a request to create a pack task").
			WithRequest(http.MethodPost, "/api/v1/tasks").
			WithHeader("Content-Type", matchers.String("application/json")).
			WithHeader("Accept", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"orderId":         matchers.String("ord-123456"),
				"consolidationId": matchers.String("consol-001"),
				"station":         matchers.String("PACK-01"),
				"items": matchers.EachLike(matchers.Map{
					"sku":      matchers.String("SKU-001"),
					"quantity": matchers.Integer(2),
				}, 1),
			}).
			WillRespondWith(http.StatusCreated).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"id":       UUIDMatcher(),
				"orderId":  matchers.String("ord-123456"),
				"status":   matchers.String("pending"),
				"hasLabel": matchers.Boolean(false),
			}).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					PackingServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				task, err := client.CreatePackTask(context.Background(), &clients.CreatePackTaskRequest{
					OrderID:         "ord-123456",
					ConsolidationID: "consol-001",
					Station:         "PACK-01",
					Items: []clients.PackItem{
						{SKU: "SKU-001", Quantity: 2},
					},
				})
				if err != nil {
					return err
				}

				assert.NotEmpty(t, task.ID)
				assert.Equal(t, "pending", task.Status)
				return nil
			})

		require.NoError(t, err)
	})

	t.Run("GetPackTask", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "packing-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		taskID := "pack-task-123456"

		err = mockProvider.
			AddInteraction().
			Given("a pack task exists").
			UponReceiving("a request to get a pack task").
			WithRequest(http.MethodGet, fmt.Sprintf("/api/v1/tasks/%s", taskID)).
			WithHeader("Accept", matchers.String("application/json")).
			WillRespondWith(http.StatusOK).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"id":          matchers.String(taskID),
				"orderId":     matchers.String("ord-123456"),
				"status":      matchers.String("in_progress"),
				"packageType": matchers.String("medium_box"),
				"weight":      matchers.Decimal(2.5),
				"hasLabel":    matchers.Boolean(false),
			}).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					PackingServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				task, err := client.GetPackTask(context.Background(), taskID)
				if err != nil {
					return err
				}

				assert.Equal(t, taskID, task.ID)
				return nil
			})

		require.NoError(t, err)
	})

	t.Run("SealPackage", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "packing-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		taskID := "pack-task-123456"

		err = mockProvider.
			AddInteraction().
			Given("a pack task is in progress with items verified").
			UponReceiving("a request to seal a package").
			WithRequest(http.MethodPost, fmt.Sprintf("/api/v1/tasks/%s/seal", taskID)).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithHeader("Accept", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"packageType": matchers.String("medium_box"),
				"weight":      matchers.Decimal(2.5),
			}).
			WillRespondWith(http.StatusOK).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					PackingServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				return client.SealPackage(context.Background(), taskID, &clients.SealPackageRequest{
					PackageType: "medium_box",
					Weight:      2.5,
				})
			})

		require.NoError(t, err)
	})

	t.Run("ApplyLabel", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "packing-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		taskID := "pack-task-123456"

		err = mockProvider.
			AddInteraction().
			Given("a pack task has a sealed package").
			UponReceiving("a request to apply a label").
			WithRequest(http.MethodPost, fmt.Sprintf("/api/v1/tasks/%s/label", taskID)).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithHeader("Accept", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"labelId":      matchers.String("label-001"),
				"trackingCode": matchers.String("1Z999AA10123456784"),
				"carrier":      matchers.String("UPS"),
			}).
			WillRespondWith(http.StatusOK).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					PackingServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				return client.ApplyLabel(context.Background(), taskID, &clients.ApplyLabelRequest{
					LabelID:      "label-001",
					TrackingCode: "1Z999AA10123456784",
					Carrier:      "UPS",
				})
			})

		require.NoError(t, err)
	})

	t.Run("CompletePackTask", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "packing-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		taskID := "pack-task-123456"

		err = mockProvider.
			AddInteraction().
			Given("a pack task has a labeled package").
			UponReceiving("a request to complete a pack task").
			WithRequest(http.MethodPost, fmt.Sprintf("/api/v1/tasks/%s/complete", taskID)).
			WithHeader("Accept", matchers.String("application/json")).
			WillRespondWith(http.StatusOK).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"id":          matchers.String(taskID),
				"orderId":     matchers.String("ord-123456"),
				"status":      matchers.String("completed"),
				"packageType": matchers.String("medium_box"),
				"weight":      matchers.Decimal(2.5),
				"hasLabel":    matchers.Boolean(true),
			}).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					PackingServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				task, err := client.CompletePackTask(context.Background(), taskID)
				if err != nil {
					return err
				}

				assert.Equal(t, "completed", task.Status)
				assert.True(t, task.HasLabel)
				return nil
			})

		require.NoError(t, err)
	})
}
