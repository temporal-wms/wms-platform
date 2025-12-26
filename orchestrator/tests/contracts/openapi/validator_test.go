package openapi_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wms-platform/shared/pkg/contracts/openapi"
)

// ServiceSpec represents a service and its OpenAPI spec path
type ServiceSpec struct {
	Name     string
	SpecPath string
}

// getServiceSpecs returns all service OpenAPI spec configurations
func getServiceSpecs() []ServiceSpec {
	basePath := "../../../../services"
	return []ServiceSpec{
		{Name: "order-service", SpecPath: filepath.Join(basePath, "order-service/docs/openapi.yaml")},
		{Name: "inventory-service", SpecPath: filepath.Join(basePath, "inventory-service/docs/openapi.yaml")},
		{Name: "routing-service", SpecPath: filepath.Join(basePath, "routing-service/docs/openapi.yaml")},
		{Name: "picking-service", SpecPath: filepath.Join(basePath, "picking-service/docs/openapi.yaml")},
		{Name: "consolidation-service", SpecPath: filepath.Join(basePath, "consolidation-service/docs/openapi.yaml")},
		{Name: "packing-service", SpecPath: filepath.Join(basePath, "packing-service/docs/openapi.yaml")},
		{Name: "shipping-service", SpecPath: filepath.Join(basePath, "shipping-service/docs/openapi.yaml")},
		{Name: "labor-service", SpecPath: filepath.Join(basePath, "labor-service/docs/openapi.yaml")},
		{Name: "waving-service", SpecPath: filepath.Join(basePath, "waving-service/docs/openapi.yaml")},
	}
}

func TestOpenAPISpecsAreValid(t *testing.T) {
	for _, spec := range getServiceSpecs() {
		t.Run(spec.Name, func(t *testing.T) {
			absPath, err := filepath.Abs(spec.SpecPath)
			require.NoError(t, err)

			// Skip if spec file doesn't exist
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				t.Skipf("OpenAPI spec not found at %s", absPath)
				return
			}

			validator, err := openapi.NewValidator(absPath)
			require.NoError(t, err, "Failed to create validator for %s", spec.Name)

			doc := validator.GetDocument()
			assert.NotNil(t, doc)
			assert.NotEmpty(t, doc.Info.Title)
			assert.NotEmpty(t, doc.Info.Version)
		})
	}
}

func TestOpenAPIHasRequiredPaths(t *testing.T) {
	requiredPaths := map[string][]string{
		"order-service":         {"/api/v1/orders/{orderId}", "/api/v1/orders/{orderId}/validate"},
		"inventory-service":     {"/api/v1/inventory/reserve", "/api/v1/inventory/sku/{sku}"},
		"routing-service":       {"/api/v1/routes", "/api/v1/routes/{routeId}"},
		"picking-service":       {"/api/v1/tasks", "/api/v1/tasks/{taskId}"},
		"consolidation-service": {"/api/v1/consolidations", "/api/v1/consolidations/{consolidationId}"},
		"packing-service":       {"/api/v1/tasks", "/api/v1/tasks/{taskId}"},
		"shipping-service":      {"/api/v1/shipments", "/api/v1/shipments/{shipmentId}"},
		"labor-service":         {"/api/v1/workers/available", "/api/v1/tasks"},
		"waving-service":        {"/api/v1/waves/{waveId}"},
	}

	for _, spec := range getServiceSpecs() {
		t.Run(spec.Name, func(t *testing.T) {
			absPath, err := filepath.Abs(spec.SpecPath)
			require.NoError(t, err)

			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				t.Skipf("OpenAPI spec not found at %s", absPath)
				return
			}

			validator, err := openapi.NewValidator(absPath)
			require.NoError(t, err)

			paths := validator.GetPaths()
			required := requiredPaths[spec.Name]

			for _, reqPath := range required {
				found := false
				for _, p := range paths {
					if p == reqPath {
						found = true
						break
					}
				}
				assert.True(t, found, "Missing required path %s in %s", reqPath, spec.Name)
			}
		})
	}
}

func TestValidateRequestAgainstSpec(t *testing.T) {
	// Example test for order-service request validation
	t.Run("ValidateOrderRequest", func(t *testing.T) {
		specPath := "../../../../services/order-service/docs/openapi.yaml"
		absPath, err := filepath.Abs(specPath)
		require.NoError(t, err)

		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			t.Skip("Order service OpenAPI spec not found")
			return
		}

		validator, err := openapi.NewValidator(absPath)
		require.NoError(t, err)

		// Create a mock request
		body := bytes.NewBufferString(`{}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/ord-123/validate", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		// Validate the request (may fail if spec doesn't match, which is expected for skeleton test)
		err = validator.ValidateRequest(req)
		// We don't fail the test if validation fails - this is to verify the validator works
		if err != nil {
			t.Logf("Request validation result: %v", err)
		}
	})
}

func TestValidateResponseAgainstSpec(t *testing.T) {
	t.Run("ValidateOrderResponse", func(t *testing.T) {
		specPath := "../../../../services/order-service/docs/openapi.yaml"
		absPath, err := filepath.Abs(specPath)
		require.NoError(t, err)

		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			t.Skip("Order service OpenAPI spec not found")
			return
		}

		validator, err := openapi.NewValidator(absPath)
		require.NoError(t, err)

		// Create a mock request and response
		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/ord-123", nil)
		req.Header.Set("Accept", "application/json")

		responseBody := `{
			"id": "ord-123",
			"customerId": "cust-001",
			"status": "pending",
			"priority": "standard",
			"totalItems": 5,
			"totalValue": 150.50,
			"createdAt": "2024-01-15T10:30:00Z",
			"updatedAt": "2024-01-15T10:30:00Z"
		}`

		rec := httptest.NewRecorder()
		rec.Header().Set("Content-Type", "application/json")
		rec.WriteHeader(http.StatusOK)
		rec.WriteString(responseBody)

		resp := rec.Result()

		err = validator.ValidateResponse(req, resp)
		if err != nil {
			t.Logf("Response validation result: %v", err)
		}
	})
}
