//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type healthResponse struct {
	Status string `json:"status"`
}

func TestHealthEndpoint(t *testing.T) {
	baseURL := os.Getenv("WES_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8016"
	}
	url := fmt.Sprintf("%s/health", baseURL)

	client := &http.Client{Timeout: 5 * time.Second}
	deadline := time.Now().Add(1 * time.Minute)

	var resp *http.Response
	var err error

	for time.Now().Before(deadline) {
		resp, err = client.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		time.Sleep(2 * time.Second)
	}

	require.NoError(t, err, "health endpoint did not become available")
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var payload healthResponse
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "healthy", payload.Status)
}
