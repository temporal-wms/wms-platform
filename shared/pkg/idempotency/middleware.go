package idempotency

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// HeaderIdempotencyKey is the HTTP header name for the idempotency key
	HeaderIdempotencyKey = "Idempotency-Key"

	// ContextKeyIDempotencyKeyID is the context key for storing the idempotency key ID
	ContextKeyIDempotencyKeyID = "idempotency_key_id"
)

// responseWriter wraps gin.ResponseWriter to capture response data
type responseWriter struct {
	gin.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// Middleware returns a Gin middleware for idempotency
func Middleware(config *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip for non-mutating methods if configured
		if config.OnlyMutating && !isMutatingMethod(c.Request.Method) {
			c.Next()
			return
		}

		// Extract idempotency key
		key := c.GetHeader(HeaderIdempotencyKey)
		key = NormalizeKey(key)

		// If no key provided
		if key == "" {
			if config.RequireKey {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error": "Idempotency-Key header is required for this operation",
					"code":  "IDEMPOTENCY_KEY_REQUIRED",
				})
				return
			}
			// Optional mode - proceed without idempotency
			c.Next()
			return
		}

		// Validate key format
		if err := ValidateKeyWithMaxLength(key, config.MaxKeyLength); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Invalid idempotency key: %v", err),
				"code":  "IDEMPOTENCY_KEY_INVALID",
			})
			return
		}

		// Extract user ID if configured
		var userID string
		if config.UserIDExtractor != nil {
			userID = config.UserIDExtractor(c)
		}

		// Read request body for fingerprinting
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			// Restore body for downstream handlers
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}
		fingerprint := ComputeFingerprint(requestBody)

		// Process idempotency logic
		processIdempotency(c, config, key, userID, fingerprint, requestBody)
	}
}

func processIdempotency(c *gin.Context, config *Config, key, userID, fingerprint string, requestBody []byte) {
	ctx := c.Request.Context()
	startTime := time.Now()

	// Create idempotency key model
	idempotencyKey := &IdempotencyKey{
		Key:                key,
		UserID:             userID,
		ServiceID:          config.ServiceName,
		RequestPath:        c.Request.URL.Path,
		RequestMethod:      c.Request.Method,
		RequestFingerprint: fingerprint,
		CreatedAt:          time.Now().UTC(),
		ExpiresAt:          time.Now().UTC().Add(config.RetentionPeriod),
	}

	// Try to acquire lock
	existingKey, isNew, err := config.Repository.AcquireLock(ctx, idempotencyKey)
	if err != nil {
		// Log error
		slog.Error("Failed to acquire idempotency lock",
			"error", err,
			"key", key,
			"service", config.ServiceName,
			"path", c.Request.URL.Path,
		)

		// Record metric
		if config.Metrics != nil {
			config.Metrics.RecordStorageError(config.ServiceName, "acquire_lock")
		}

		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
			"error": "Idempotency storage is temporarily unavailable",
			"code":  "IDEMPOTENCY_STORAGE_UNAVAILABLE",
		})
		return
	}

	// Record lock acquisition duration
	lockDuration := time.Since(startTime).Seconds()
	if config.Metrics != nil {
		config.Metrics.RecordLockAcquisitionDuration(
			config.ServiceName,
			c.Request.URL.Path,
			c.Request.Method,
			lockDuration,
		)
	}

	// Check if request is already completed
	if existingKey.IsCompleted() {
		// Verify request fingerprint matches
		if existingKey.RequestFingerprint != fingerprint {
			slog.Warn("Idempotency parameter mismatch",
				"key", key,
				"service", config.ServiceName,
				"path", c.Request.URL.Path,
				"originalFingerprint", existingKey.RequestFingerprint,
				"newFingerprint", fingerprint,
			)

			// Record metric
			if config.Metrics != nil {
				config.Metrics.RecordParameterMismatch(
					config.ServiceName,
					c.Request.URL.Path,
					c.Request.Method,
				)
			}

			c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
				"error": "Request parameters differ from original request with this idempotency key",
				"code":  "IDEMPOTENCY_PARAMETER_MISMATCH",
			})
			return
		}

		// Return cached response
		slog.Info("Idempotency cache hit",
			"key", key,
			"service", config.ServiceName,
			"path", c.Request.URL.Path,
			"statusCode", existingKey.ResponseCode,
		)

		// Record metric
		if config.Metrics != nil {
			config.Metrics.RecordHit(
				config.ServiceName,
				c.Request.URL.Path,
				c.Request.Method,
			)
		}

		// Set cached response headers
		for k, v := range existingKey.ResponseHeaders {
			c.Header(k, v)
		}

		// Return cached response
		c.Data(existingKey.ResponseCode, "application/json", existingKey.ResponseBody)
		c.Abort()
		return
	}

	// Check if another request is processing (concurrent request)
	if !isNew && existingKey.IsLocked() {
		lockAge := time.Since(*existingKey.LockedAt)
		if lockAge < config.LockTimeout {
			slog.Warn("Concurrent idempotency request",
				"key", key,
				"service", config.ServiceName,
				"path", c.Request.URL.Path,
				"lockAge", lockAge,
			)

			// Record metric
			if config.Metrics != nil {
				config.Metrics.RecordConcurrentCollision(
					config.ServiceName,
					c.Request.URL.Path,
					c.Request.Method,
				)
			}

			c.AbortWithStatusJSON(http.StatusConflict, gin.H{
				"error": "A request with this idempotency key is currently being processed",
				"code":  "IDEMPOTENCY_CONCURRENT_REQUEST",
			})
			return
		}

		// Lock expired, proceed (stale lock cleanup)
		slog.Info("Stale lock detected, proceeding",
			"key", key,
			"service", config.ServiceName,
			"path", c.Request.URL.Path,
			"lockAge", lockAge,
		)
	}

	// Store key ID in context for atomic phases
	c.Set(ContextKeyIDempotencyKeyID, existingKey.ID.Hex())

	// Record cache miss
	if config.Metrics != nil {
		config.Metrics.RecordMiss(
			config.ServiceName,
			c.Request.URL.Path,
			c.Request.Method,
		)
	}

	slog.Info("Processing new idempotency request",
		"key", key,
		"service", config.ServiceName,
		"path", c.Request.URL.Path,
	)

	// Wrap response writer to capture output
	writer := &responseWriter{
		ResponseWriter: c.Writer,
		body:           &bytes.Buffer{},
		statusCode:     http.StatusOK,
	}
	c.Writer = writer

	// Process request
	c.Next()

	// Store response after all handlers execute
	responseBody := writer.body.Bytes()

	// Check response size
	if len(responseBody) > config.MaxResponseSize {
		slog.Warn("Response too large to cache",
			"key", key,
			"service", config.ServiceName,
			"path", c.Request.URL.Path,
			"size", len(responseBody),
			"maxSize", config.MaxResponseSize,
		)

		// Store a marker instead of the full response
		responseBody = []byte(fmt.Sprintf(`{"error":"Response too large to cache","size":%d}`, len(responseBody)))
	}

	// Extract response headers
	headers := extractResponseHeaders(c)

	// Store response
	err = config.Repository.StoreResponse(
		ctx,
		existingKey.ID.Hex(),
		writer.statusCode,
		responseBody,
		headers,
	)

	if err != nil {
		slog.Error("Failed to store idempotency response",
			"error", err,
			"key", key,
			"service", config.ServiceName,
			"path", c.Request.URL.Path,
		)

		// Record metric
		if config.Metrics != nil {
			config.Metrics.RecordStorageError(config.ServiceName, "store_response")
		}
	} else {
		slog.Debug("Stored idempotency response",
			"key", key,
			"service", config.ServiceName,
			"path", c.Request.URL.Path,
			"statusCode", writer.statusCode,
		)
	}
}

// isMutatingMethod returns true if the HTTP method is mutating
func isMutatingMethod(method string) bool {
	return method == http.MethodPost ||
		method == http.MethodPut ||
		method == http.MethodPatch ||
		method == http.MethodDelete
}

// extractResponseHeaders extracts response headers from the context
func extractResponseHeaders(c *gin.Context) map[string]string {
	headers := make(map[string]string)
	for k, v := range c.Writer.Header() {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	return headers
}
