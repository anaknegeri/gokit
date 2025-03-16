// Package errors provides standardized error handling
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
	ErrCodeMethodNotAllowed   = "METHOD_NOT_ALLOWED"

	// Filesystem specific error codes
	ErrCodeFileNotFound       = "FILE_NOT_FOUND"
	ErrCodeFileAlreadyExists  = "FILE_ALREADY_EXISTS"
	ErrCodeFileTooLarge       = "FILE_TOO_LARGE"
	ErrCodeInvalidFileType    = "INVALID_FILE_TYPE"
	ErrCodeStorageUnavailable = "STORAGE_UNAVAILABLE"
	ErrCodePermissionDenied   = "PERMISSION_DENIED"
	ErrCodeQuotaExceeded      = "QUOTA_EXCEEDED"
	ErrCodeInvalidPath        = "INVALID_PATH"

	// Database specific error codes
	ErrCodeDatabaseError       = "DATABASE_ERROR"
	ErrCodeDuplicateEntry      = "DUPLICATE_ENTRY"
	ErrCodeForeignKeyViolation = "FOREIGN_KEY_VIOLATION"
	ErrCodeRecordNotFound      = "RECORD_NOT_FOUND"

	// Authentication specific error codes
	ErrCodeInvalidCredentials = "INVALID_CREDENTIALS"
	ErrCodeTokenExpired       = "TOKEN_EXPIRED"
	ErrCodeInvalidToken       = "INVALID_TOKEN"
	ErrCodeAccountLocked      = "ACCOUNT_LOCKED"
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
	http.StatusMethodNotAllowed:    ErrCodeMethodNotAllowed,
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

// Standard errors for common scenarios

// BadRequestError creates a bad request error
func BadRequestError(message string) *AppError {
	return NewError(http.StatusBadRequest, message)
}

// UnauthorizedError creates an unauthorized error
func UnauthorizedError(message string) *AppError {
	if message == "" {
		message = "Unauthorized access"
	}
	return NewError(http.StatusUnauthorized, message)
}

// ForbiddenError creates a forbidden error
func ForbiddenError(message string) *AppError {
	if message == "" {
		message = "Access forbidden"
	}
	return NewError(http.StatusForbidden, message)
}

// NotFoundError creates a not found error
func NotFoundError(message string) *AppError {
	if message == "" {
		message = "Resource not found"
	}
	return NewError(http.StatusNotFound, message)
}

// ConflictError creates a conflict error
func ConflictError(message string) *AppError {
	return NewError(http.StatusConflict, message)
}

// InternalServerError creates an internal server error
func InternalServerError(message string) *AppError {
	if message == "" {
		message = "Internal server error"
	}
	return NewError(http.StatusInternalServerError, message)
}

// MethodNotAllowedError creates a method not allowed error
func MethodNotAllowedError(message string) *AppError {
	if message == "" {
		message = "Method not allowed"
	}
	return NewError(http.StatusMethodNotAllowed, message)
}

// ServiceUnavailableError creates a service unavailable error
func ServiceUnavailableError(message string) *AppError {
	if message == "" {
		message = "Service temporarily unavailable"
	}
	return NewError(http.StatusServiceUnavailable, message)
}

// File-specific errors

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

// FileAlreadyExistsError creates an error for when a file already exists
func FileAlreadyExistsError(path string) *AppError {
	return NewCustomError(
		http.StatusConflict,
		ErrCodeFileAlreadyExists,
		fmt.Sprintf("File already exists: %s", path),
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

// InvalidPathError creates an error for invalid file paths
func InvalidPathError(path string, reason string) *AppError {
	return NewErrorWithDetails(
		http.StatusBadRequest,
		fmt.Sprintf("Invalid path: %s", path),
		map[string]interface{}{
			"path":   path,
			"reason": reason,
		},
	)
}

// Database-specific errors

// DatabaseError creates a general database error
func DatabaseError(err error) *AppError {
	return WrapErrorWithCustomCode(
		err,
		http.StatusInternalServerError,
		ErrCodeDatabaseError,
		"Database operation failed",
	)
}

// RecordNotFoundError creates a record not found error
func RecordNotFoundError(entity string, id interface{}) *AppError {
	return NewCustomError(
		http.StatusNotFound,
		ErrCodeRecordNotFound,
		fmt.Sprintf("%s with ID %v not found", entity, id),
	)
}

// DuplicateEntryError creates a duplicate entry error
func DuplicateEntryError(entity string, field string, value interface{}) *AppError {
	return NewErrorWithDetails(
		http.StatusConflict,
		fmt.Sprintf("Duplicate %s: %s already exists", entity, field),
		map[string]interface{}{
			"entity": entity,
			"field":  field,
			"value":  value,
		},
	)
}

// ForeignKeyViolationError creates a foreign key violation error
func ForeignKeyViolationError(entity string, relation string) *AppError {
	return NewErrorWithDetails(
		http.StatusConflict,
		fmt.Sprintf("Cannot modify %s due to existing %s references", entity, relation),
		map[string]interface{}{
			"entity":   entity,
			"relation": relation,
		},
	)
}

// Authentication-specific errors

// InvalidCredentialsError creates an invalid credentials error
func InvalidCredentialsError() *AppError {
	return NewCustomError(
		http.StatusUnauthorized,
		ErrCodeInvalidCredentials,
		"Invalid credentials",
	)
}

// TokenExpiredError creates a token expired error
func TokenExpiredError() *AppError {
	return NewCustomError(
		http.StatusUnauthorized,
		ErrCodeTokenExpired,
		"Authentication token has expired",
	)
}

// InvalidTokenError creates an invalid token error
func InvalidTokenError() *AppError {
	return NewCustomError(
		http.StatusUnauthorized,
		ErrCodeInvalidToken,
		"Invalid authentication token",
	)
}

// AccountLockedError creates an account locked error
func AccountLockedError() *AppError {
	return NewCustomError(
		http.StatusForbidden,
		ErrCodeAccountLocked,
		"Account is locked",
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
	case "eqfield":
		return fmt.Sprintf("%s must be equal to %s", fe.Field(), fe.Param())
	case "nefield":
		return fmt.Sprintf("%s must not be equal to %s", fe.Field(), fe.Param())
	case "isbn":
		return fmt.Sprintf("%s must be a valid ISBN", fe.Field())
	case "isbn10":
		return fmt.Sprintf("%s must be a valid ISBN-10", fe.Field())
	case "isbn13":
		return fmt.Sprintf("%s must be a valid ISBN-13", fe.Field())
	case "creditcard":
		return fmt.Sprintf("%s must be a valid credit card number", fe.Field())
	case "hexcolor":
		return fmt.Sprintf("%s must be a valid hex color", fe.Field())
	case "rgb":
		return fmt.Sprintf("%s must be a valid RGB color", fe.Field())
	case "rgba":
		return fmt.Sprintf("%s must be a valid RGBA color", fe.Field())
	case "hsv":
		return fmt.Sprintf("%s must be a valid HSV color", fe.Field())
	case "hsla":
		return fmt.Sprintf("%s must be a valid HSLA color", fe.Field())
	case "e164":
		return fmt.Sprintf("%s must be a valid E.164 formatted phone number", fe.Field())
	case "base64":
		return fmt.Sprintf("%s must be a valid Base64 string", fe.Field())
	case "base64url":
		return fmt.Sprintf("%s must be a valid Base64URL string", fe.Field())
	case "contains":
		return fmt.Sprintf("%s must contain the text '%s'", fe.Field(), fe.Param())
	case "containsany":
		return fmt.Sprintf("%s must contain at least one of the following characters '%s'", fe.Field(), fe.Param())
	case "excludes":
		return fmt.Sprintf("%s may not contain the text '%s'", fe.Field(), fe.Param())
	case "excludesall":
		return fmt.Sprintf("%s may not contain any of the following characters '%s'", fe.Field(), fe.Param())
	case "ip":
		return fmt.Sprintf("%s must be a valid IP address", fe.Field())
	case "ipv4":
		return fmt.Sprintf("%s must be a valid IPv4 address", fe.Field())
	case "ipv6":
		return fmt.Sprintf("%s must be a valid IPv6 address", fe.Field())
	case "mac":
		return fmt.Sprintf("%s must be a valid MAC address", fe.Field())
	default:
		return fmt.Sprintf("%s failed validation for tag %s", fe.Field(), fe.Tag())
	}
}
