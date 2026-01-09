package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Standard error codes
const (
	CodeValidationError   = "VALIDATION_ERROR"
	CodeNotFound          = "RESOURCE_NOT_FOUND"
	CodeConflict          = "CONFLICT"
	CodeUnauthorized      = "UNAUTHORIZED"
	CodeForbidden         = "FORBIDDEN"
	CodeInternalError     = "INTERNAL_ERROR"
	CodeBadRequest        = "BAD_REQUEST"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	CodeTimeout           = "TIMEOUT"
	CodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
)

// AppError represents an application error with HTTP status and error code
type AppError struct {
	Code       string            `json:"code"`
	Message    string            `json:"message"`
	Details    map[string]string `json:"details,omitempty"`
	HTTPStatus int               `json:"-"`
	Err        error             `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithDetails adds details to the error
func (e *AppError) WithDetails(details map[string]string) *AppError {
	e.Details = details
	return e
}

// WithDetail adds a single detail to the error
func (e *AppError) WithDetail(key, value string) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]string)
	}
	e.Details[key] = value
	return e
}

// Wrap wraps an existing error
func (e *AppError) Wrap(err error) *AppError {
	e.Err = err
	return e
}

// NewAppError creates a new AppError
func NewAppError(code string, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// Validation errors

// ErrValidation creates a validation error
func ErrValidation(message string) *AppError {
	return NewAppError(CodeValidationError, message, http.StatusBadRequest)
}

// ErrValidationWithFields creates a validation error with field details
func ErrValidationWithFields(message string, fields map[string]string) *AppError {
	return ErrValidation(message).WithDetails(fields)
}

// Resource errors

// ErrNotFound creates a not found error
func ErrNotFound(resource string) *AppError {
	return NewAppError(CodeNotFound, fmt.Sprintf("%s not found", resource), http.StatusNotFound)
}

// ErrNotFoundWithID creates a not found error with ID
func ErrNotFoundWithID(resource, id string) *AppError {
	return ErrNotFound(resource).WithDetail("id", id)
}

// ErrConflict creates a conflict error
func ErrConflict(message string) *AppError {
	return NewAppError(CodeConflict, message, http.StatusConflict)
}

// Authentication/Authorization errors

// ErrUnauthorized creates an unauthorized error
func ErrUnauthorized(message string) *AppError {
	if message == "" {
		message = "authentication required"
	}
	return NewAppError(CodeUnauthorized, message, http.StatusUnauthorized)
}

// ErrForbidden creates a forbidden error
func ErrForbidden(message string) *AppError {
	if message == "" {
		message = "access denied"
	}
	return NewAppError(CodeForbidden, message, http.StatusForbidden)
}

// Internal errors

// ErrInternal creates an internal error
func ErrInternal(message string) *AppError {
	if message == "" {
		message = "an internal error occurred"
	}
	return NewAppError(CodeInternalError, message, http.StatusInternalServerError)
}

// ErrBadRequest creates a bad request error
func ErrBadRequest(message string) *AppError {
	return NewAppError(CodeBadRequest, message, http.StatusBadRequest)
}

// Service errors

// ErrServiceUnavailable creates a service unavailable error
func ErrServiceUnavailable(service string) *AppError {
	return NewAppError(CodeServiceUnavailable, fmt.Sprintf("%s is temporarily unavailable", service), http.StatusServiceUnavailable)
}

// ErrTimeout creates a timeout error
func ErrTimeout(operation string) *AppError {
	return NewAppError(CodeTimeout, fmt.Sprintf("%s timed out", operation), http.StatusGatewayTimeout)
}

// ErrRateLimitExceeded creates a rate limit error
func ErrRateLimitExceeded() *AppError {
	return NewAppError(CodeRateLimitExceeded, "rate limit exceeded", http.StatusTooManyRequests)
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr)
}

// AsAppError converts an error to an AppError if possible
func AsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

// FromError converts a standard error to an AppError
func FromError(err error) *AppError {
	if err == nil {
		return nil
	}

	if appErr, ok := AsAppError(err); ok {
		return appErr
	}

	return ErrInternal("").Wrap(err)
}

// Domain error mappings - common domain errors that should be mapped to AppErrors

// MapDomainError maps common domain error messages to AppErrors
func MapDomainError(err error) *AppError {
	if err == nil {
		return nil
	}

	// Check if it's already an AppError
	if appErr, ok := AsAppError(err); ok {
		return appErr
	}

	msg := err.Error()

	// Map common domain error patterns
	switch {
	case contains(msg, "not found"):
		return ErrNotFound("resource").Wrap(err)
	case contains(msg, "already exists"):
		return ErrConflict(msg).Wrap(err)
	case contains(msg, "invalid"):
		return ErrValidation(msg).Wrap(err)
	case contains(msg, "required"):
		return ErrValidation(msg).Wrap(err)
	case contains(msg, "unauthorized"):
		return ErrUnauthorized(msg).Wrap(err)
	case contains(msg, "forbidden"), contains(msg, "permission denied"):
		return ErrForbidden(msg).Wrap(err)
	case contains(msg, "timeout"):
		return ErrTimeout("operation").Wrap(err)
	default:
		return ErrInternal("").Wrap(err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsIgnoreCase(s, substr))
}

func containsIgnoreCase(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalIgnoreCase(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func equalIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
