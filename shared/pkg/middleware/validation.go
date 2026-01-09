package middleware

import (
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/wms-platform/shared/pkg/errors"
)

var (
	validate     *validator.Validate
	validateOnce sync.Once
)

// InitValidator initializes the validator with custom validators
func InitValidator() *validator.Validate {
	validateOnce.Do(func() {
		validate = validator.New()

		// Register custom validators
		_ = validate.RegisterValidation("order_id", validateOrderID)
		_ = validate.RegisterValidation("sku", validateSKU)
		_ = validate.RegisterValidation("wave_id", validateWaveID)
		_ = validate.RegisterValidation("location_id", validateLocationID)
		_ = validate.RegisterValidation("carrier_code", validateCarrierCode)
		_ = validate.RegisterValidation("tracking_number", validateTrackingNumber)
		_ = validate.RegisterValidation("priority", validatePriority)
		_ = validate.RegisterValidation("safe_string", validateSafeString)

		// Use JSON tag names for error messages
		validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return fld.Name
			}
			return name
		})

		// Set as Gin's default validator
		if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
			_ = v.RegisterValidation("order_id", validateOrderID)
			_ = v.RegisterValidation("sku", validateSKU)
			_ = v.RegisterValidation("wave_id", validateWaveID)
			_ = v.RegisterValidation("location_id", validateLocationID)
			_ = v.RegisterValidation("carrier_code", validateCarrierCode)
			_ = v.RegisterValidation("tracking_number", validateTrackingNumber)
			_ = v.RegisterValidation("priority", validatePriority)
			_ = v.RegisterValidation("safe_string", validateSafeString)

			v.RegisterTagNameFunc(func(fld reflect.StructField) string {
				name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
				if name == "-" {
					return fld.Name
				}
				return name
			})
		}
	})

	return validate
}

// GetValidator returns the singleton validator instance
func GetValidator() *validator.Validate {
	if validate == nil {
		return InitValidator()
	}
	return validate
}

// Custom validators

var (
	orderIDRegex   = regexp.MustCompile(`^ORD-[a-zA-Z0-9]{8,}$`)
	skuRegex       = regexp.MustCompile(`^[A-Z0-9][A-Z0-9-]{2,49}$`)
	waveIDRegex    = regexp.MustCompile(`^WAVE-[a-zA-Z0-9]{8,}$`)
	locationRegex  = regexp.MustCompile(`^[A-Z]{1,2}-\d{2}-\d{2}-[A-Z0-9]+$`)
	carrierRegex   = regexp.MustCompile(`^(UPS|FEDEX|USPS|DHL|ONTRAC)$`)
	safeStringRegex = regexp.MustCompile(`^[a-zA-Z0-9\s\-_.,!?@#$%&*()+=:;'"<>\/\[\]{}|\\~\x60]+$`)
)

func validateOrderID(fl validator.FieldLevel) bool {
	return orderIDRegex.MatchString(fl.Field().String())
}

func validateSKU(fl validator.FieldLevel) bool {
	return skuRegex.MatchString(fl.Field().String())
}

func validateWaveID(fl validator.FieldLevel) bool {
	return waveIDRegex.MatchString(fl.Field().String())
}

func validateLocationID(fl validator.FieldLevel) bool {
	return locationRegex.MatchString(fl.Field().String())
}

func validateCarrierCode(fl validator.FieldLevel) bool {
	return carrierRegex.MatchString(fl.Field().String())
}

func validateTrackingNumber(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	// Basic validation: 8-30 alphanumeric characters
	return len(value) >= 8 && len(value) <= 30
}

func validatePriority(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	validPriorities := map[string]bool{
		"same_day": true,
		"next_day": true,
		"standard": true,
		"economy":  true,
	}
	return validPriorities[value]
}

func validateSafeString(fl validator.FieldLevel) bool {
	return safeStringRegex.MatchString(fl.Field().String())
}

// ValidationErrorFormatter formats validation errors into a map
func ValidationErrorFormatter(err error) map[string]string {
	fields := make(map[string]string)

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			field := e.Field()
			fields[field] = formatValidationError(e)
		}
	}

	return fields
}

func formatValidationError(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "is required"
	case "min":
		return "must be at least " + e.Param()
	case "max":
		return "must be at most " + e.Param()
	case "gte":
		return "must be greater than or equal to " + e.Param()
	case "lte":
		return "must be less than or equal to " + e.Param()
	case "email":
		return "must be a valid email address"
	case "uuid":
		return "must be a valid UUID"
	case "order_id":
		return "must be a valid order ID (format: ORD-xxxxxxxx)"
	case "sku":
		return "must be a valid SKU (uppercase alphanumeric with dashes)"
	case "wave_id":
		return "must be a valid wave ID (format: WAVE-xxxxxxxx)"
	case "location_id":
		return "must be a valid location ID (format: A-01-02-B1)"
	case "carrier_code":
		return "must be a valid carrier code (UPS, FEDEX, USPS, DHL, ONTRAC)"
	case "tracking_number":
		return "must be a valid tracking number (8-30 characters)"
	case "priority":
		return "must be one of: same_day, next_day, standard, economy"
	case "safe_string":
		return "contains invalid characters"
	case "oneof":
		return "must be one of: " + e.Param()
	default:
		return "is invalid"
	}
}

// BindAndValidate binds request body and validates it
func BindAndValidate(c *gin.Context, obj interface{}) *errors.AppError {
	if err := c.ShouldBindJSON(obj); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			fields := ValidationErrorFormatter(validationErrors)
			return errors.ErrValidationWithFields("validation failed", fields)
		}
		return errors.ErrBadRequest("invalid request body: " + err.Error())
	}
	return nil
}

// ValidateStruct validates a struct using the validator
func ValidateStruct(obj interface{}) *errors.AppError {
	v := GetValidator()
	if err := v.Struct(obj); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			fields := ValidationErrorFormatter(validationErrors)
			return errors.ErrValidationWithFields("validation failed", fields)
		}
		return errors.ErrBadRequest("validation failed: " + err.Error())
	}
	return nil
}

// SanitizeString removes potentially dangerous characters from a string
func SanitizeString(s string) string {
	// Remove null bytes
	s = strings.ReplaceAll(s, "\x00", "")

	// Trim whitespace
	s = strings.TrimSpace(s)

	return s
}

// InputSanitizer middleware sanitizes string inputs
func InputSanitizer() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Sanitize query parameters
		query := c.Request.URL.Query()
		for key, values := range query {
			for i, v := range values {
				values[i] = SanitizeString(v)
			}
			query[key] = values
		}
		c.Request.URL.RawQuery = query.Encode()

		c.Next()
	}
}

// ContentType middleware ensures proper content type for POST/PUT/PATCH
func ContentType() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			contentType := c.GetHeader("Content-Type")
			if contentType == "" || !strings.HasPrefix(contentType, "application/json") {
				// Allow empty body for some endpoints
				if c.Request.ContentLength > 0 {
					AbortWithAppError(c, &errors.AppError{
						Code:       "INVALID_CONTENT_TYPE",
						Message:    "Content-Type must be application/json",
						HTTPStatus: 415,
					})
					return
				}
			}
		}
		c.Next()
	}
}
