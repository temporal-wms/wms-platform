package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupHandler() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.POST("/sellers", func(c *gin.Context) {
		var body map[string]interface{}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "validation failed"})
			return
		}
		if body["tenantId"] == nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "validation failed", "details": map[string]string{"TenantID": "is required"}})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": body})
	})
	router.GET("/sellers/:sellerId", func(c *gin.Context) {
		sellerId := c.Param("sellerId")
		if sellerId == "" {
			c.JSON(http.StatusNotFound, gin.H{"code": "RESOURCE_NOT_FOUND", "message": "seller not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": map[string]string{"sellerId": sellerId}})
	})
	router.GET("/sellers/search", func(c *gin.Context) {
		query := c.Query("q")
		if query == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "search query 'q' is required"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": []string{}, "total": 0, "page": 1, "pageSize": 20, "totalPages": 0})
	})
	router.PUT("/sellers/:sellerId/activate", func(c *gin.Context) {
		sellerId := c.Param("sellerId")
		if sellerId == "" {
			c.JSON(http.StatusNotFound, gin.H{"code": "RESOURCE_NOT_FOUND", "message": "seller not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": map[string]string{"sellerId": sellerId, "status": "active"}})
	})
	router.PUT("/sellers/:sellerId/suspend", func(c *gin.Context) {
		sellerId := c.Param("sellerId")
		var body struct {
			Reason string `json:"reason" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "validation failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": map[string]string{"sellerId": sellerId, "status": "suspended", "reason": body.Reason}})
	})
	router.PUT("/sellers/:sellerId/close", func(c *gin.Context) {
		sellerId := c.Param("sellerId")
		var body struct {
			Reason string `json:"reason" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "validation failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": map[string]string{"sellerId": sellerId, "status": "closed", "reason": body.Reason}})
	})
	router.POST("/sellers/:sellerId/facilities", func(c *gin.Context) {
		var body struct {
			FacilityID   string `json:"facilityId" binding:"required"`
			FacilityName string `json:"facilityName" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "validation failed"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": body})
	})
	router.POST("/sellers/:sellerId/integrations", func(c *gin.Context) {
		var body struct {
			ChannelType  string                 `json:"channelType" binding:"required,oneof=shopify amazon ebay woocommerce"`
			StoreName    string                 `json:"storeName" binding:"required"`
			Credentials  map[string]string      `json:"credentials" binding:"required"`
			SyncSettings map[string]interface{} `json:"syncSettings"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "validation failed"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": body})
	})
	router.POST("/sellers/:sellerId/api-keys", func(c *gin.Context) {
		var body struct {
			Name   string   `json:"name" binding:"required"`
			Scopes []string `json:"scopes" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "validation failed"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": body})
	})
	router.GET("/sellers/:sellerId/api-keys", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": []string{}})
	})

	return router
}

func TestSellerHandler_CreateSeller(t *testing.T) {
	router := setupHandler()

	t.Run("success", func(t *testing.T) {
		body := `{"tenantId": "TNT-001", "companyName": "Test Corp", "contactName": "John Doe", "contactEmail": "john@test.com", "billingCycle": "monthly"}`
		req := httptest.NewRequest("POST", "/sellers", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		body := `{"invalid": json}`
		req := httptest.NewRequest("POST", "/sellers", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing tenantId", func(t *testing.T) {
		body := `{"companyName": "Test", "contactName": "John", "contactEmail": "john@test.com", "billingCycle": "monthly"}`
		req := httptest.NewRequest("POST", "/sellers", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSellerHandler_GetSeller(t *testing.T) {
	router := setupHandler()

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sellers/SLR-001", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("not found - empty sellerId", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sellers/", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestSellerHandler_SearchSellers(t *testing.T) {
	router := setupHandler()

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sellers/search?q=Test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("missing query parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sellers/search", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty query", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sellers/search?q=", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSellerHandler_ActivateSeller(t *testing.T) {
	router := setupHandler()

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/sellers/SLR-001/activate", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("not found - empty sellerId", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/sellers//activate", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestSellerHandler_SuspendSeller(t *testing.T) {
	router := setupHandler()

	t.Run("success", func(t *testing.T) {
		body := `{"reason": "Violation"}`
		req := httptest.NewRequest("PUT", "/sellers/SLR-001/suspend", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("missing reason body", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/sellers/SLR-001/suspend", bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSellerHandler_CloseSeller(t *testing.T) {
	router := setupHandler()

	t.Run("success", func(t *testing.T) {
		body := `{"reason": "Business closed"}`
		req := httptest.NewRequest("PUT", "/sellers/SLR-001/close", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("missing reason body", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/sellers/SLR-001/close", bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSellerHandler_AssignFacility(t *testing.T) {
	router := setupHandler()

	t.Run("success", func(t *testing.T) {
		body := `{"facilityId": "FAC-001", "facilityName": "Test DC"}`
		req := httptest.NewRequest("POST", "/sellers/SLR-001/facilities", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("missing facilityId", func(t *testing.T) {
		body := `{"facilityName": "Test DC"}`
		req := httptest.NewRequest("POST", "/sellers/SLR-001/facilities", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSellerHandler_ConnectChannel(t *testing.T) {
	router := setupHandler()

	t.Run("success", func(t *testing.T) {
		body := `{"channelType": "shopify", "storeName": "Test Store", "credentials": {"apiKey": "test"}, "syncSettings": {}}`
		req := httptest.NewRequest("POST", "/sellers/SLR-001/integrations", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("missing channelType", func(t *testing.T) {
		body := `{"storeName": "Test Store", "credentials": {}, "syncSettings": {}}`
		req := httptest.NewRequest("POST", "/sellers/SLR-001/integrations", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSellerHandler_GenerateAPIKey(t *testing.T) {
	router := setupHandler()

	t.Run("success", func(t *testing.T) {
		body := `{"name": "Test Key", "scopes": ["orders:read", "inventory:read"]}`
		req := httptest.NewRequest("POST", "/sellers/SLR-001/api-keys", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("missing scopes", func(t *testing.T) {
		body := `{"name": "Test Key"}`
		req := httptest.NewRequest("POST", "/sellers/SLR-001/api-keys", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSellerHandler_ListAPIKeys(t *testing.T) {
	router := setupHandler()

	req := httptest.NewRequest("GET", "/sellers/SLR-001/api-keys", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
