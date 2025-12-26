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

func TestPickingServiceConsumer(t *testing.T) {
	ensurePactDir(t)

	t.Run("CreatePickTask", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "picking-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		err = mockProvider.
			AddInteraction().
			Given("wave is released").
			UponReceiving("a request to create a pick task").
			WithRequest(http.MethodPost, "/api/v1/tasks").
			WithHeader("Content-Type", matchers.String("application/json")).
			WithHeader("Accept", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"orderId":  matchers.String("ord-123456"),
				"waveId":   matchers.String("wave-001"),
				"priority": matchers.Integer(1),
				"items": matchers.EachLike(matchers.Map{
					"sku":      matchers.String("SKU-001"),
					"quantity": matchers.Integer(2),
					"location": matchers.String("A-01-01"),
				}, 1),
			}).
			WillRespondWith(http.StatusCreated).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"id":        UUIDMatcher(),
				"orderId":   matchers.String("ord-123456"),
				"waveId":    matchers.String("wave-001"),
				"status":    matchers.String("pending"),
				"priority":  matchers.Integer(1),
				"itemCount": matchers.Integer(1),
				"createdAt": TimestampMatcher(),
			}).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					PickingServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				task, err := client.CreatePickTask(context.Background(), &clients.CreatePickTaskRequest{
					OrderID:  "ord-123456",
					WaveID:   "wave-001",
					Priority: 1,
					Items: []clients.PickItem{
						{SKU: "SKU-001", Quantity: 2, Location: "A-01-01"},
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

	t.Run("GetPickTask", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "picking-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		taskID := "task-123456"

		err = mockProvider.
			AddInteraction().
			Given("a pick task exists").
			UponReceiving("a request to get a pick task").
			WithRequest(http.MethodGet, fmt.Sprintf("/api/v1/tasks/%s", taskID)).
			WithHeader("Accept", matchers.String("application/json")).
			WillRespondWith(http.StatusOK).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"id":        matchers.String(taskID),
				"orderId":   matchers.String("ord-123456"),
				"waveId":    matchers.String("wave-001"),
				"status":    matchers.String("in_progress"),
				"workerId":  matchers.String("worker-001"),
				"priority":  matchers.Integer(1),
				"itemCount": matchers.Integer(5),
				"createdAt": TimestampMatcher(),
			}).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					PickingServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				task, err := client.GetPickTask(context.Background(), taskID)
				if err != nil {
					return err
				}

				assert.Equal(t, taskID, task.ID)
				return nil
			})

		require.NoError(t, err)
	})

	t.Run("AssignPickTask", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "picking-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		taskID := "task-123456"
		workerID := "worker-001"

		err = mockProvider.
			AddInteraction().
			Given("a pick task exists and worker is available").
			UponReceiving("a request to assign a pick task").
			WithRequest(http.MethodPost, fmt.Sprintf("/api/v1/tasks/%s/assign", taskID)).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithHeader("Accept", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"workerId": matchers.String(workerID),
			}).
			WillRespondWith(http.StatusOK).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					PickingServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				return client.AssignPickTask(context.Background(), taskID, workerID)
			})

		require.NoError(t, err)
	})

	t.Run("CompletePickTask", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "picking-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		taskID := "task-123456"

		err = mockProvider.
			AddInteraction().
			Given("a pick task is in progress").
			UponReceiving("a request to complete a pick task").
			WithRequest(http.MethodPost, fmt.Sprintf("/api/v1/tasks/%s/complete", taskID)).
			WithHeader("Accept", matchers.String("application/json")).
			WillRespondWith(http.StatusOK).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"id":        matchers.String(taskID),
				"orderId":   matchers.String("ord-123456"),
				"waveId":    matchers.String("wave-001"),
				"status":    matchers.String("completed"),
				"priority":  matchers.Integer(1),
				"itemCount": matchers.Integer(5),
				"createdAt": TimestampMatcher(),
			}).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					PickingServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				task, err := client.CompletePickTask(context.Background(), taskID)
				if err != nil {
					return err
				}

				assert.Equal(t, "completed", task.Status)
				return nil
			})

		require.NoError(t, err)
	})
}
