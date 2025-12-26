package api

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/wms-platform/shared/pkg/errors"
)

// ValidationError represents a field validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Tag     string `json:"tag,omitempty"`
}

// BindAndValidate binds request body and validates it
func BindAndValidate(c *gin.Context, obj interface{}) *errors.AppError {
	if err := c.ShouldBindJSON(obj); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			fields := make(map[string]string)
			for _, fieldError := range validationErrors {
				field := getFieldName(fieldError)
				fields[field] = getErrorMessage(fieldError)
			}
			return errors.ErrValidationWithFields("validation failed", fields)
		}
		return errors.ErrBadRequest(fmt.Sprintf("invalid request body: %v", err))
	}
	return nil
}

// BindQueryAndValidate binds query parameters and validates them
func BindQueryAndValidate(c *gin.Context, obj interface{}) *errors.AppError {
	if err := c.ShouldBindQuery(obj); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			fields := make(map[string]string)
			for _, fieldError := range validationErrors {
				field := getFieldName(fieldError)
				fields[field] = getErrorMessage(fieldError)
			}
			return errors.ErrValidationWithFields("validation failed", fields)
		}
		return errors.ErrBadRequest(fmt.Sprintf("invalid query parameters: %v", err))
	}
	return nil
}

// BindURIAndValidate binds URI parameters and validates them
func BindURIAndValidate(c *gin.Context, obj interface{}) *errors.AppError {
	if err := c.ShouldBindUri(obj); err != nil {
		return errors.ErrBadRequest(fmt.Sprintf("invalid URI parameters: %v", err))
	}
	return nil
}

// getFieldName extracts the JSON field name from validator.FieldError
func getFieldName(fe validator.FieldError) string {
	// Try to get JSON tag first
	field := fe.Field()

	// Convert to camelCase (first letter lowercase)
	if len(field) > 0 {
		field = strings.ToLower(field[:1]) + field[1:]
	}

	return field
}

// getErrorMessage returns a human-readable error message for a validation error
func getErrorMessage(fe validator.FieldError) string {
	field := getFieldName(fe)

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "min":
		if fe.Type().String() == "string" {
			return fmt.Sprintf("%s must be at least %s characters", field, fe.Param())
		}
		return fmt.Sprintf("%s must be at least %s", field, fe.Param())
	case "max":
		if fe.Type().String() == "string" {
			return fmt.Sprintf("%s must be at most %s characters", field, fe.Param())
		}
		return fmt.Sprintf("%s must be at most %s", field, fe.Param())
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", field)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, fe.Param())
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", field, fe.Param())
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", field, fe.Param())
	case "lt":
		return fmt.Sprintf("%s must be less than %s", field, fe.Param())
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", field, fe.Param())
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters", field, fe.Param())
	case "alpha":
		return fmt.Sprintf("%s must contain only letters", field)
	case "alphanum":
		return fmt.Sprintf("%s must contain only letters and numbers", field)
	case "numeric":
		return fmt.Sprintf("%s must be a number", field)
	case "datetime":
		return fmt.Sprintf("%s must be a valid datetime in format %s", field, fe.Param())
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}

// ValidateStruct validates a struct and returns AppError
func ValidateStruct(obj interface{}) *errors.AppError {
	validate := validator.New()

	if err := validate.Struct(obj); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			fields := make(map[string]string)
			for _, fieldError := range validationErrors {
				field := getFieldName(fieldError)
				fields[field] = getErrorMessage(fieldError)
			}
			return errors.ErrValidationWithFields("validation failed", fields)
		}
		return errors.ErrBadRequest(fmt.Sprintf("validation error: %v", err))
	}

	return nil
}
