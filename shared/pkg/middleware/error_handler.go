package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wms-platform/shared/pkg/errors"
)

// APIErrorResponse represents a standardized error response
type APIErrorResponse struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Details   map[string]string `json:"details,omitempty"`
	RequestID string            `json:"requestId,omitempty"`
	Timestamp string            `json:"timestamp"`
	Path      string            `json:"path"`
}

// ErrorHandler is a middleware that handles errors and returns standardized responses
func ErrorHandler(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there are any errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			// Get request ID from context
			requestID, _ := c.Get(ContextKeyRequestID)
			reqID, _ := requestID.(string)

			// Convert to AppError
			appErr := errors.MapDomainError(err)

			// Log the error
			logError(logger, c, appErr, reqID)

			// Build response
			response := APIErrorResponse{
				Code:      appErr.Code,
				Message:   appErr.Message,
				Details:   appErr.Details,
				RequestID: reqID,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Path:      c.Request.URL.Path,
			}

			c.JSON(appErr.HTTPStatus, response)
		}
	}
}

// ErrorResponder provides helper methods for sending error responses
type ErrorResponder struct {
	ctx    *gin.Context
	logger *slog.Logger
}

// NewErrorResponder creates a new ErrorResponder
func NewErrorResponder(ctx *gin.Context, logger *slog.Logger) *ErrorResponder {
	return &ErrorResponder{ctx: ctx, logger: logger}
}

// RespondWithError sends an error response
func (r *ErrorResponder) RespondWithError(err error) {
	appErr := errors.MapDomainError(err)
	r.RespondWithAppError(appErr)
}

// RespondWithAppError sends an AppError response
func (r *ErrorResponder) RespondWithAppError(appErr *errors.AppError) {
	requestID, _ := r.ctx.Get(ContextKeyRequestID)
	reqID, _ := requestID.(string)

	logError(r.logger, r.ctx, appErr, reqID)

	response := APIErrorResponse{
		Code:      appErr.Code,
		Message:   appErr.Message,
		Details:   appErr.Details,
		RequestID: reqID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      r.ctx.Request.URL.Path,
	}

	r.ctx.JSON(appErr.HTTPStatus, response)
}

// RespondNotFound sends a 404 response
func (r *ErrorResponder) RespondNotFound(resource string) {
	r.RespondWithAppError(errors.ErrNotFound(resource))
}

// RespondBadRequest sends a 400 response
func (r *ErrorResponder) RespondBadRequest(message string) {
	r.RespondWithAppError(errors.ErrBadRequest(message))
}

// RespondValidationError sends a validation error response
func (r *ErrorResponder) RespondValidationError(message string, fields map[string]string) {
	r.RespondWithAppError(errors.ErrValidationWithFields(message, fields))
}

// RespondInternalError sends a 500 response
func (r *ErrorResponder) RespondInternalError(err error) {
	appErr := errors.ErrInternal("").Wrap(err)
	r.RespondWithAppError(appErr)
}

// RespondConflict sends a 409 response
func (r *ErrorResponder) RespondConflict(message string) {
	r.RespondWithAppError(errors.ErrConflict(message))
}

// RespondUnauthorized sends a 401 response
func (r *ErrorResponder) RespondUnauthorized(message string) {
	r.RespondWithAppError(errors.ErrUnauthorized(message))
}

// RespondForbidden sends a 403 response
func (r *ErrorResponder) RespondForbidden(message string) {
	r.RespondWithAppError(errors.ErrForbidden(message))
}

// RespondServiceUnavailable sends a 503 response
func (r *ErrorResponder) RespondServiceUnavailable(service string) {
	r.RespondWithAppError(errors.ErrServiceUnavailable(service))
}

// Helper function for error response
func logError(logger *slog.Logger, c *gin.Context, appErr *errors.AppError, requestID string) {
	logLevel := slog.LevelError
	if appErr.HTTPStatus < http.StatusInternalServerError {
		logLevel = slog.LevelWarn
	}

	attrs := []any{
		"code", appErr.Code,
		"message", appErr.Message,
		"status", appErr.HTTPStatus,
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
		"requestId", requestID,
		"clientIP", c.ClientIP(),
	}

	if appErr.Err != nil {
		attrs = append(attrs, "error", appErr.Err.Error())
	}

	if appErr.Details != nil {
		attrs = append(attrs, "details", appErr.Details)
	}

	logger.Log(c.Request.Context(), logLevel, "API error", attrs...)
}

// AbortWithError aborts the request with an error
func AbortWithError(c *gin.Context, err error) {
	appErr := errors.MapDomainError(err)
	AbortWithAppError(c, appErr)
}

// AbortWithAppError aborts the request with an AppError
func AbortWithAppError(c *gin.Context, appErr *errors.AppError) {
	requestID, _ := c.Get(ContextKeyRequestID)
	reqID, _ := requestID.(string)

	response := APIErrorResponse{
		Code:      appErr.Code,
		Message:   appErr.Message,
		Details:   appErr.Details,
		RequestID: reqID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      c.Request.URL.Path,
	}

	c.AbortWithStatusJSON(appErr.HTTPStatus, response)
}
