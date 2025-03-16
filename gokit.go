// Package gokit provides a comprehensive toolkit for building Go applications
// including filesystem operations, validation, error handling, pagination, and logging.
package gokit

import (
	"context"
	"reflect"

	"github.com/anaknegeri/gokit/pkg/errors"
	"github.com/anaknegeri/gokit/pkg/filesystem"
	"github.com/anaknegeri/gokit/pkg/logger"
	"github.com/anaknegeri/gokit/pkg/pagination"
	"github.com/anaknegeri/gokit/pkg/response"
	"github.com/anaknegeri/gokit/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Re-export types for filesystem
type (
	// Filesystem types
	FileStorage       = filesystem.Storage
	FileSystemConfig  = filesystem.Config
	FileSystemInfo    = filesystem.FileInfo
	FilesystemHandler = filesystem.FilesystemProvider

	// Pagination types
	PaginationParams = pagination.PaginationParams
	PaginationMeta   = pagination.PaginationMeta
	PaginationResult = pagination.PaginationResult
	Paginator        = pagination.Paginator

	// Error types
	AppError        = errors.AppError
	ValidationError = errors.ValidationError

	// Validator types
	Validator = validator.Validator

	// Logger types
	Logger   = logger.Logger
	LogLevel = logger.LogLevel

	// Response types
	ApiResponse = response.Response
)

// Export error codes
const (
	// Generic error codes
	ErrCodeBadRequest         = errors.ErrCodeBadRequest
	ErrCodeUnauthorized       = errors.ErrCodeUnauthorized
	ErrCodeForbidden          = errors.ErrCodeForbidden
	ErrCodeNotFound           = errors.ErrCodeNotFound
	ErrCodeConflict           = errors.ErrCodeConflict
	ErrCodeValidationError    = errors.ErrCodeValidationError
	ErrCodeInternalError      = errors.ErrCodeInternalError
	ErrCodeServiceUnavailable = errors.ErrCodeServiceUnavailable

	// Filesystem specific error codes
	ErrCodeFileNotFound       = errors.ErrCodeFileNotFound
	ErrCodeFileAlreadyExists  = errors.ErrCodeFileAlreadyExists
	ErrCodeFileTooLarge       = errors.ErrCodeFileTooLarge
	ErrCodeInvalidFileType    = errors.ErrCodeInvalidFileType
	ErrCodeStorageUnavailable = errors.ErrCodeStorageUnavailable
	ErrCodePermissionDenied   = errors.ErrCodePermissionDenied

	// Log levels
	LogLevelDebug = logger.DEBUG
	LogLevelInfo  = logger.INFO
	LogLevelWarn  = logger.WARN
	LogLevelError = logger.ERROR
	LogLevelFatal = logger.FATAL
)

// Filesystem functions

// NewFilesystem creates a new filesystem provider from environment variables
func NewFilesystem(ctx context.Context) (*filesystem.FilesystemProvider, error) {
	return filesystem.NewFilesystemProvider(ctx)
}

// NewFilesystemWithConfig creates a new filesystem provider with the provided config
func NewFilesystemWithConfig(ctx context.Context, config filesystem.Config) (*filesystem.FilesystemProvider, error) {
	return filesystem.NewFilesystemProviderWithConfig(ctx, config)
}

// NewLocalStorage creates a new local storage
func NewLocalStorage(config filesystem.LocalStorageConfig) (filesystem.Storage, error) {
	return filesystem.NewLocalStorage(config)
}

// NewS3Storage creates a new S3 storage
func NewS3Storage(config filesystem.S3Config) (filesystem.Storage, error) {
	return filesystem.NewS3Storage(config)
}

// Pagination functions

// NewPaginator creates a new paginator
func NewPaginator(db *gorm.DB) *pagination.Paginator {
	return pagination.NewPaginator(db)
}

// GetPaginationFromRequest extracts pagination parameters from a request
func GetPaginationFromRequest(c interface {
	QueryInt(string, int) int
}) pagination.PaginationParams {
	return pagination.GetPaginationFromRequest(c)
}

// Validator functions

// NewValidator creates a new validator
func NewValidator() validator.Validator {
	return validator.NewValidator()
}

// Error functions

// New creates a new standard error
func New(message string) error {
	return errors.New(message)
}

// Is checks if an error matches a specific error code
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As attempts to convert an error to a target type
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// NewError creates a new error
func NewError(httpCode int, message string) *errors.AppError {
	return errors.NewError(httpCode, message)
}

// NewErrorWithDetails creates a new error with details
func NewErrorWithDetails(httpCode int, message string, details interface{}) *errors.AppError {
	return errors.NewErrorWithDetails(httpCode, message, details)
}

// WrapError wraps an existing error
func WrapError(err error, httpCode int, message string) *errors.AppError {
	return errors.WrapError(err, httpCode, message)
}

// ValidatorError creates an error from validation errors
func ValidatorError(err error) *errors.AppError {
	return errors.ValidatorError(err)
}

// ValidatorFieldLevel is an alias for validator.FieldLevel
type ValidatorFieldLevel interface {
	Field() reflect.Value
	FieldName() string
	Param() string
}

// FormatErrorResponse formats an error into a response
func FormatErrorResponse(err error) *errors.ErrorResponse {
	return errors.FormatErrorResponse(err)
}

// Logger functions

// NewLogger creates a new logger
func NewLogger() *logger.Logger {
	return logger.NewLogger()
}

// InitLogger initializes a logger from environment variables
func InitLogger() *logger.Logger {
	return logger.InitLogger()
}

// Response functions

// SuccessResponse sends a success response
func SuccessResponse(c *fiber.Ctx, message string, data interface{}, statusCode ...int) error {
	return response.Success(c, message, data, statusCode...)
}

// ErrorResponse sends an error response
func ErrorResponseWithErr(c *fiber.Ctx, err error) error {
	return response.Error(c, err)
}

// CreatedResponse sends a created response
func CreatedResponse(c *fiber.Ctx, message string, data interface{}) error {
	return response.Created(c, message, data)
}

// BadRequestResponse sends a bad request response
func BadRequestResponse(c *fiber.Ctx, message string, details interface{}) error {
	return response.BadRequest(c, message, details)
}

// NotFoundResponse sends a not found response
func NotFoundResponse(c *fiber.Ctx, message string) error {
	return response.NotFound(c, message)
}

// MethodNotAllowedResponse sends a method not allowed response
func MethodNotAllowedResponse(c *fiber.Ctx, message string) error {
	return response.MethodNotAllowed(c, message)
}

// UnauthorizedResponse sends an unauthorized response
func UnauthorizedResponse(c *fiber.Ctx, message string) error {
	return response.Unauthorized(c, message)
}

// ForbiddenResponse sends a forbidden response
func ForbiddenResponse(c *fiber.Ctx, message string) error {
	return response.Forbidden(c, message)
}

// InternalServerErrorResponse sends an internal server error response
func InternalServerErrorResponse(c *fiber.Ctx, message string) error {
	return response.InternalServerError(c, message)
}
