package errors

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Error codes for different error types
const (
	// Generic error codes
	ErrCodeBadRequest         = "BAD_REQUEST"
	ErrCodeUnauthorized       = "UNAUTHORIZED"
	ErrCodeForbidden          = "FORBIDDEN"
	ErrCodeNotFound           = "NOT_FOUND"
	ErrCodeConflict           = "CONFLICT"
	ErrCodeValidationError    = "VALIDATION_ERROR"
	ErrCodeInternalError      = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"

	// Filesystem specific error codes
	ErrCodeFileNotFound       = "FILE_NOT_FOUND"
	ErrCodeFileAlreadyExists  = "FILE_ALREADY_EXISTS"
	ErrCodeFileTooLarge       = "FILE_TOO_LARGE"
	ErrCodeInvalidFileType    = "INVALID_FILE_TYPE"
	ErrCodeStorageUnavailable = "STORAGE_UNAVAILABLE"
	ErrCodePermissionDenied   = "PERMISSION_DENIED"
	ErrCodeQuotaExceeded      = "QUOTA_EXCEEDED"
	ErrCodeInvalidPath        = "INVALID_PATH"
)

// Map HTTP status codes to error codes
var statusToErrorCode = map[int]string{
	http.StatusBadRequest:          ErrCodeBadRequest,
	http.StatusUnauthorized:        ErrCodeUnauthorized,
	http.StatusForbidden:           ErrCodeForbidden,
	http.StatusNotFound:            ErrCodeNotFound,
	http.StatusConflict:            ErrCodeConflict,
	http.StatusUnprocessableEntity: ErrCodeValidationError,
	http.StatusInternalServerError: ErrCodeInternalError,
	http.StatusServiceUnavailable:  ErrCodeServiceUnavailable,
}

// AppError represents an application error with detailed information
type AppError struct {
	Code     string      `json:"code"`
	Message  string      `json:"message"`
	Details  interface{} `json:"details,omitempty"`
	HTTPCode int         `json:"-"`
	Internal error       `json:"-"`
}

// Error implements the error interface for AppError
func (e *AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Internal)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error
func (e *AppError) Unwrap() error {
	return e.Internal
}

// ErrorResponse is the structure for API error responses
type ErrorResponse struct {
	Success bool        `json:"success"`
	Code    int         `json:"code"`
	Error   string      `json:"error"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// New creates a new standard error
func New(message string) error {
	return errors.New(message)
}

// NewError creates a new AppError
func NewError(httpCode int, message string) *AppError {
	code, ok := statusToErrorCode[httpCode]
	if !ok {
		code = statusToErrorCode[http.StatusInternalServerError]
	}

	return &AppError{
		Code:     code,
		Message:  message,
		HTTPCode: httpCode,
	}
}

// NewErrorWithDetails creates a new AppError with additional details
func NewErrorWithDetails(httpCode int, message string, details interface{}) *AppError {
	err := NewError(httpCode, message)
	err.Details = details
	return err
}

// NewCustomError creates a new AppError with a custom error code
func NewCustomError(httpCode int, code string, message string) *AppError {
	return &AppError{
		Code:     code,
		Message:  message,
		HTTPCode: httpCode,
	}
}

// WrapError wraps an existing error with additional context
func WrapError(err error, httpCode int, message string) *AppError {
	appErr := NewError(httpCode, message)
	appErr.Internal = err
	return appErr
}

// WrapErrorWithCustomCode wraps an error with a custom error code
func WrapErrorWithCustomCode(err error, httpCode int, code string, message string) *AppError {
	appErr := NewCustomError(httpCode, code, message)
	appErr.Internal = err
	return appErr
}

// Is checks if an error is of a specific type
func Is(err error, target error) bool {
	return errors.Is(err, target)
}

// As attempts to convert an error to a specific type
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// ValidationError represents a field validation error
type ValidationError struct {
	Field   string      `json:"field"`
	Message string      `json:"message"`
	Tag     string      `json:"tag,omitempty"`
	Value   interface{} `json:"value,omitempty"`
	Param   string      `json:"param,omitempty"`
}

// ValidatorError processes validator.ValidationErrors into a consistent format
func ValidatorError(err error) *AppError {
	var validationErrors []ValidationError

	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrs {
			validationErrors = append(validationErrors, ValidationError{
				Field:   formatFieldName(e.Field()),
				Message: generateValidationMessage(e),
				Tag:     e.Tag(),
				Value:   e.Value(),
				Param:   e.Param(),
			})
		}
	}

	return NewErrorWithDetails(
		http.StatusUnprocessableEntity,
		"Validation failed",
		validationErrors,
	)
}

// FormatErrorResponse formats an error into a consistent API response
func FormatErrorResponse(err error) *ErrorResponse {
	if appErr, ok := err.(*AppError); ok {
		return &ErrorResponse{
			Success: false,
			Code:    appErr.HTTPCode,
			Error:   appErr.Code,
			Message: appErr.Message,
			Details: appErr.Details,
		}
	}

	// Default error handling if not an AppError
	return &ErrorResponse{
		Success: false,
		Code:    http.StatusInternalServerError,
		Error:   ErrCodeInternalError,
		Message: err.Error(),
	}
}

// FileNotFoundError creates an error for file not found situations
func FileNotFoundError(path string) *AppError {
	return NewCustomError(
		http.StatusNotFound,
		ErrCodeFileNotFound,
		fmt.Sprintf("File not found: %s", path),
	)
}

// FileTooLargeError creates an error for files that exceed size limits
func FileTooLargeError(size, maxSize int64) *AppError {
	return NewCustomError(
		http.StatusBadRequest,
		ErrCodeFileTooLarge,
		fmt.Sprintf("File size of %d bytes exceeds the maximum allowed size of %d bytes", size, maxSize),
	)
}

// InvalidFileTypeError creates an error for unsupported file types
func InvalidFileTypeError(fileType string, allowedTypes []string) *AppError {
	return NewErrorWithDetails(
		http.StatusBadRequest,
		fmt.Sprintf("File type '%s' is not allowed", fileType),
		map[string]interface{}{
			"fileType":     fileType,
			"allowedTypes": allowedTypes,
		},
	)
}

// StorageUnavailableError creates an error for when storage is unavailable
func StorageUnavailableError(err error) *AppError {
	return WrapErrorWithCustomCode(
		err,
		http.StatusServiceUnavailable,
		ErrCodeStorageUnavailable,
		"Storage service is currently unavailable",
	)
}

// formatFieldName converts field names to camelCase
func formatFieldName(field string) string {
	return strings.ToLower(field[:1]) + field[1:]
}

// generateValidationMessage generates user-friendly validation messages
func generateValidationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fe.Field())
	case "email":
		return "Invalid email format"
	case "min":
		if fe.Kind() == reflect.String {
			return fmt.Sprintf("%s must be at least %s characters long", fe.Field(), fe.Param())
		}
		return fmt.Sprintf("%s must be at least %s", fe.Field(), fe.Param())
	case "max":
		if fe.Kind() == reflect.String {
			return fmt.Sprintf("%s must not exceed %s characters", fe.Field(), fe.Param())
		}
		return fmt.Sprintf("%s must not exceed %s", fe.Field(), fe.Param())
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", fe.Field())
	case "oneof":
		return fmt.Sprintf("%s must be one of [%s]", fe.Field(), fe.Param())
	case "unique":
		return fmt.Sprintf("%s must be unique", fe.Field())
	case "numeric":
		return fmt.Sprintf("%s must be numeric", fe.Field())
	case "json":
		return fmt.Sprintf("%s must be valid JSON", fe.Field())
	case "url":
		return fmt.Sprintf("%s must be a valid URL", fe.Field())
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", fe.Field(), fe.Param())
	case "lt":
		return fmt.Sprintf("%s must be less than %s", fe.Field(), fe.Param())
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", fe.Field(), fe.Param())
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", fe.Field(), fe.Param())
	case "alpha":
		return fmt.Sprintf("%s must contain only letters", fe.Field())
	case "alphanum":
		return fmt.Sprintf("%s must contain only letters and numbers", fe.Field())
	case "datetime":
		return fmt.Sprintf("%s must be a valid datetime", fe.Field())
	case "file":
		return fmt.Sprintf("%s must be a valid file", fe.Field())
	case "image":
		return fmt.Sprintf("%s must be a valid image", fe.Field())
	case "mime":
		return fmt.Sprintf("%s must be of type %s", fe.Field(), fe.Param())
	case "password":
		return fmt.Sprintf("%s must meet password requirements", fe.Field())
	default:
		return fmt.Sprintf("%s failed validation for tag %s", fe.Field(), fe.Tag())
	}
}
