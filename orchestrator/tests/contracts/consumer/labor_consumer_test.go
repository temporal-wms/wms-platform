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

func TestLaborServiceConsumer(t *testing.T) {
	ensurePactDir(t)

	t.Run("GetAvailableWorkers", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "labor-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		taskType := "picking"
		zone := "ZONE-A"

		err = mockProvider.
			AddInteraction().
			Given("workers are available").
			UponReceiving("a request to get available workers").
			WithRequest(http.MethodGet, "/api/v1/workers/available").
			WithQuery("taskType", matchers.String(taskType)).
			WithQuery("zone", matchers.String(zone)).
			WithHeader("Accept", matchers.String("application/json")).
			WillRespondWith(http.StatusOK).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithJSONBody(matchers.EachLike(matchers.Map{
				"id":       matchers.String("worker-001"),
				"name":     matchers.String("John Doe"),
				"status":   matchers.String("available"),
				"zone":     matchers.String(zone),
				"skills":   matchers.EachLike(matchers.String("picking"), 1),
				"taskType": matchers.String(taskType),
			}, 1)).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					LaborServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				workers, err := client.GetAvailableWorkers(context.Background(), taskType, zone)
				if err != nil {
					return err
				}

				assert.NotEmpty(t, workers)
				assert.Equal(t, "available", workers[0].Status)
				return nil
			})

		require.NoError(t, err)
	})

	t.Run("AssignWorker", func(t *testing.T) {
		mockProvider, err := pact.NewV4Pact(pact.Config{
			Consumer: consumerName,
			Provider: "labor-service",
			PactDir:  pactDir,
		})
		require.NoError(t, err)

		err = mockProvider.
			AddInteraction().
			Given("a worker is available").
			UponReceiving("a request to assign a worker to a task").
			WithRequest(http.MethodPost, "/api/v1/tasks").
			WithHeader("Content-Type", matchers.String("application/json")).
			WithHeader("Accept", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"workerId": matchers.String("worker-001"),
				"taskType": matchers.String("picking"),
				"taskRef":  matchers.String("pick-task-123"),
				"zone":     matchers.String("ZONE-A"),
			}).
			WillRespondWith(http.StatusCreated).
			WithHeader("Content-Type", matchers.String("application/json")).
			WithJSONBody(matchers.Map{
				"id":       UUIDMatcher(),
				"workerId": matchers.String("worker-001"),
				"taskType": matchers.String("picking"),
				"taskRef":  matchers.String("pick-task-123"),
				"status":   matchers.String("assigned"),
			}).
			ExecuteTest(t, func(config consumer.MockServerConfig) error {
				client := clients.NewServiceClients(&clients.Config{
					LaborServiceURL: fmt.Sprintf("http://%s:%d", config.Host, config.Port),
				})

				task, err := client.AssignWorker(context.Background(), &clients.AssignWorkerRequest{
					WorkerID: "worker-001",
					TaskType: "picking",
					TaskRef:  "pick-task-123",
					Zone:     "ZONE-A",
				})
				if err != nil {
					return err
				}

				assert.NotEmpty(t, task.ID)
				assert.Equal(t, "assigned", task.Status)
				return nil
			})

		require.NoError(t, err)
	})
}
