package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/wms-platform/shared/pkg/logging"
)

// CloudEvents WMS extension context keys
const (
	ContextKeyWMSCorrelationID = "wmsCorrelationId"
	ContextKeyWMSWaveNumber    = "wmsWaveNumber"
	ContextKeyWMSWorkflowID    = "wmsWorkflowId"
)

// CloudEvents WMS extension HTTP header names
const (
	HeaderWMSCorrelationID = "X-WMS-Correlation-ID"
	HeaderWMSWaveNumber    = "X-WMS-Wave-Number"
	HeaderWMSWorkflowID    = "X-WMS-Workflow-ID"
)

// CloudEvents middleware extracts WMS CloudEvents extensions from HTTP headers
// and adds them to the request context for downstream logging and propagation.
// These extensions follow the CloudEvents specification and are used for
// distributed tracing and correlation across WMS services.
func CloudEvents() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract WMS CloudEvents extensions from headers
		wmsCorrelationID := c.GetHeader(HeaderWMSCorrelationID)
		wmsWaveNumber := c.GetHeader(HeaderWMSWaveNumber)
		wmsWorkflowID := c.GetHeader(HeaderWMSWorkflowID)

		// Set in Gin context
		if wmsCorrelationID != "" {
			c.Set(ContextKeyWMSCorrelationID, wmsCorrelationID)
		}
		if wmsWaveNumber != "" {
			c.Set(ContextKeyWMSWaveNumber, wmsWaveNumber)
		}
		if wmsWorkflowID != "" {
			c.Set(ContextKeyWMSWorkflowID, wmsWorkflowID)
		}

		// Set in Go context for logging package
		ctx := c.Request.Context()
		if wmsCorrelationID != "" {
			ctx = logging.ContextWithWMSCorrelationID(ctx, wmsCorrelationID)
		}
		if wmsWaveNumber != "" {
			ctx = logging.ContextWithWMSWaveNumber(ctx, wmsWaveNumber)
		}
		if wmsWorkflowID != "" {
			ctx = logging.ContextWithWMSWorkflowID(ctx, wmsWorkflowID)
		}
		c.Request = c.Request.WithContext(ctx)

		// Propagate headers in response (for tracing)
		if wmsCorrelationID != "" {
			c.Header(HeaderWMSCorrelationID, wmsCorrelationID)
		}
		if wmsWaveNumber != "" {
			c.Header(HeaderWMSWaveNumber, wmsWaveNumber)
		}
		if wmsWorkflowID != "" {
			c.Header(HeaderWMSWorkflowID, wmsWorkflowID)
		}

		c.Next()
	}
}

// GetWMSCorrelationID extracts WMS correlation ID from Gin context
func GetWMSCorrelationID(c *gin.Context) string {
	if val, exists := c.Get(ContextKeyWMSCorrelationID); exists {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}

// GetWMSWaveNumber extracts WMS wave number from Gin context
func GetWMSWaveNumber(c *gin.Context) string {
	if val, exists := c.Get(ContextKeyWMSWaveNumber); exists {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}

// GetWMSWorkflowID extracts WMS workflow ID from Gin context
func GetWMSWorkflowID(c *gin.Context) string {
	if val, exists := c.Get(ContextKeyWMSWorkflowID); exists {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}

// CloudEventExtensions holds all WMS CloudEvent extension values
type CloudEventExtensions struct {
	CorrelationID string
	WaveNumber    string
	WorkflowID    string
}

// GetCloudEventExtensions extracts all CloudEvent extensions from Gin context
func GetCloudEventExtensions(c *gin.Context) CloudEventExtensions {
	return CloudEventExtensions{
		CorrelationID: GetWMSCorrelationID(c),
		WaveNumber:    GetWMSWaveNumber(c),
		WorkflowID:    GetWMSWorkflowID(c),
	}
}

// ToLoggingContext converts CloudEventExtensions to logging.CloudEventContext
func (ce CloudEventExtensions) ToLoggingContext() logging.CloudEventContext {
	return logging.CloudEventContext{
		CorrelationID: ce.CorrelationID,
		WaveNumber:    ce.WaveNumber,
		WorkflowID:    ce.WorkflowID,
	}
}

// PropagationCloudEventHeaders returns CloudEvents WMS headers for propagation to downstream services
func PropagationCloudEventHeaders(c *gin.Context) map[string]string {
	headers := make(map[string]string)

	if id := GetWMSCorrelationID(c); id != "" {
		headers[HeaderWMSCorrelationID] = id
	}
	if id := GetWMSWaveNumber(c); id != "" {
		headers[HeaderWMSWaveNumber] = id
	}
	if id := GetWMSWorkflowID(c); id != "" {
		headers[HeaderWMSWorkflowID] = id
	}

	return headers
}

// CloudEventsLogger middleware adds CloudEvents extensions to logs
// This is a specialized Logger middleware that includes WMS CloudEvents extensions
func CloudEventsLogger(logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get CloudEvent extensions
		ext := GetCloudEventExtensions(c)

		// Create enriched logger
		enrichedLogger := logger.WithCloudEventContext(ext.ToLoggingContext())

		// Store enriched logger in context
		c.Set("logger", enrichedLogger)

		c.Next()
	}
}

// GetEnrichedLogger retrieves the CloudEvents-enriched logger from Gin context
func GetEnrichedLogger(c *gin.Context, fallbackLogger *logging.Logger) *logging.Logger {
	if logger, exists := c.Get("logger"); exists {
		if l, ok := logger.(*logging.Logger); ok {
			return l
		}
	}
	return fallbackLogger
}
