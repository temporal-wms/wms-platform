package idempotency

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// mockKeyRepository is a mock implementation of KeyRepository for testing
type mockKeyRepository struct {
	acquireLockFunc func(ctx context.Context, key *IdempotencyKey) (*IdempotencyKey, bool, error)
	storeResponseFunc func(ctx context.Context, keyID string, responseCode int, responseBody []byte, headers map[string]string) error
}

func (m *mockKeyRepository) AcquireLock(ctx context.Context, key *IdempotencyKey) (*IdempotencyKey, bool, error) {
	if m.acquireLockFunc != nil {
		return m.acquireLockFunc(ctx, key)
	}
	return key, true, nil
}

func (m *mockKeyRepository) ReleaseLock(ctx context.Context, keyID string) error {
	return nil
}

func (m *mockKeyRepository) StoreResponse(ctx context.Context, keyID string, responseCode int, responseBody []byte, headers map[string]string) error {
	if m.storeResponseFunc != nil {
		return m.storeResponseFunc(ctx, keyID, responseCode, responseBody, headers)
	}
	return nil
}

func (m *mockKeyRepository) UpdateRecoveryPoint(ctx context.Context, keyID string, phase string) error {
	return nil
}

func (m *mockKeyRepository) Get(ctx context.Context, key, serviceID string) (*IdempotencyKey, error) {
	return nil, ErrNotFound
}

func (m *mockKeyRepository) GetByID(ctx context.Context, keyID string) (*IdempotencyKey, error) {
	return nil, ErrNotFound
}

func (m *mockKeyRepository) Clean(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

func (m *mockKeyRepository) EnsureIndexes(ctx context.Context) error {
	return nil
}

func TestMiddleware_NoKey_Optional(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockKeyRepository{}
	config := &Config{
		ServiceName:     "test-service",
		Repository:      repo,
		RequireKey:      false, // Optional mode
		OnlyMutating:    true,
		MaxKeyLength:    255,
		LockTimeout:     5 * time.Minute,
		RetentionPeriod: 24 * time.Hour,
		MaxResponseSize: 1024 * 1024,
	}

	router := gin.New()
	router.Use(Middleware(config))
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(`{"data":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestMiddleware_NoKey_Required(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockKeyRepository{}
	config := &Config{
		ServiceName:     "test-service",
		Repository:      repo,
		RequireKey:      true, // Required mode
		OnlyMutating:    true,
		MaxKeyLength:    255,
		LockTimeout:     5 * time.Minute,
		RetentionPeriod: 24 * time.Hour,
		MaxResponseSize: 1024 * 1024,
	}

	router := gin.New()
	router.Use(Middleware(config))
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(`{"data":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestMiddleware_InvalidKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockKeyRepository{}
	config := DefaultConfig("test-service", repo)

	router := gin.New()
	router.Use(Middleware(config))
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(`{"data":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(HeaderIdempotencyKey, "invalid key with spaces")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestMiddleware_NewRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockKeyRepository{
		acquireLockFunc: func(ctx context.Context, key *IdempotencyKey) (*IdempotencyKey, bool, error) {
			// Return new key
			key.ID = [12]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
			return key, true, nil
		},
	}

	config := DefaultConfig("test-service", repo)

	router := gin.New()
	router.Use(Middleware(config))
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"message": "created"})
	})

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(`{"data":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(HeaderIdempotencyKey, "test-key-123")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
}

func TestMiddleware_CachedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	completedAt := time.Now().UTC()
	cachedResponse := []byte(`{"message":"cached"}`)

	repo := &mockKeyRepository{
		acquireLockFunc: func(ctx context.Context, key *IdempotencyKey) (*IdempotencyKey, bool, error) {
			// Return completed key with cached response
			existingKey := &IdempotencyKey{
				ID:                 [12]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
				Key:                key.Key,
				ServiceID:          key.ServiceID,
				RequestPath:        key.RequestPath,
				RequestMethod:      key.RequestMethod,
				RequestFingerprint: key.RequestFingerprint,
				ResponseCode:       http.StatusOK,
				ResponseBody:       cachedResponse,
				ResponseHeaders:    map[string]string{"Content-Type": "application/json"},
				CompletedAt:        &completedAt,
			}
			return existingKey, false, nil
		},
	}

	config := DefaultConfig("test-service", repo)

	router := gin.New()
	router.Use(Middleware(config))
	router.POST("/test", func(c *gin.Context) {
		// This should not be called
		t.Error("Handler should not be called for cached response")
		c.JSON(http.StatusCreated, gin.H{"message": "new"})
	})

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(`{"data":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(HeaderIdempotencyKey, "test-key-123")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != string(cachedResponse) {
		t.Errorf("Expected cached response, got %s", w.Body.String())
	}
}

func TestMiddleware_ParameterMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	completedAt := time.Now().UTC()
	originalFingerprint := ComputeFingerprint([]byte(`{"data":"original"}`))

	repo := &mockKeyRepository{
		acquireLockFunc: func(ctx context.Context, key *IdempotencyKey) (*IdempotencyKey, bool, error) {
			// Return completed key with different fingerprint
			existingKey := &IdempotencyKey{
				ID:                 [12]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
				Key:                key.Key,
				ServiceID:          key.ServiceID,
				RequestPath:        key.RequestPath,
				RequestMethod:      key.RequestMethod,
				RequestFingerprint: originalFingerprint,
				ResponseCode:       http.StatusOK,
				ResponseBody:       []byte(`{"message":"original"}`),
				CompletedAt:        &completedAt,
			}
			return existingKey, false, nil
		},
	}

	config := DefaultConfig("test-service", repo)

	router := gin.New()
	router.Use(Middleware(config))
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"message": "new"})
	})

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(`{"data":"different"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(HeaderIdempotencyKey, "test-key-123")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422, got %d", w.Code)
	}
}

func TestMiddleware_ConcurrentRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	lockedAt := time.Now().UTC()

	repo := &mockKeyRepository{
		acquireLockFunc: func(ctx context.Context, key *IdempotencyKey) (*IdempotencyKey, bool, error) {
			// Return locked key (concurrent request)
			existingKey := &IdempotencyKey{
				ID:                 [12]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
				Key:                key.Key,
				ServiceID:          key.ServiceID,
				RequestPath:        key.RequestPath,
				RequestMethod:      key.RequestMethod,
				RequestFingerprint: key.RequestFingerprint,
				LockedAt:           &lockedAt,
				CompletedAt:        nil, // Not completed
			}
			return existingKey, false, nil
		},
	}

	config := DefaultConfig("test-service", repo)

	router := gin.New()
	router.Use(Middleware(config))
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"message": "new"})
	})

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(`{"data":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(HeaderIdempotencyKey, "test-key-123")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}
}

func TestMiddleware_StorageFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockKeyRepository{
		acquireLockFunc: func(ctx context.Context, key *IdempotencyKey) (*IdempotencyKey, bool, error) {
			return nil, false, errors.New("database connection failed")
		},
	}

	config := DefaultConfig("test-service", repo)

	router := gin.New()
	router.Use(Middleware(config))
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"message": "new"})
	})

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(`{"data":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(HeaderIdempotencyKey, "test-key-123")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}

func TestMiddleware_SkipGETRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockKeyRepository{
		acquireLockFunc: func(ctx context.Context, key *IdempotencyKey) (*IdempotencyKey, bool, error) {
			t.Error("AcquireLock should not be called for GET request")
			return nil, false, errors.New("should not be called")
		},
	}

	config := DefaultConfig("test-service", repo)
	config.OnlyMutating = true

	router := gin.New()
	router.Use(Middleware(config))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(HeaderIdempotencyKey, "test-key-123")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
