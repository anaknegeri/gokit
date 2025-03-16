// Package response provides standardized API response handling
package response

import (
	"reflect"

	"github.com/anaknegeri/gokit/pkg/errors"
	"github.com/anaknegeri/gokit/pkg/pagination"
	"github.com/gofiber/fiber/v2"
)

// Response represents a standardized API response
type Response struct {
	Success bool        `json:"success"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success sends a successful response with the provided data
func Success(c *fiber.Ctx, message string, data interface{}, statusCode ...int) error {
	code := fiber.StatusOK
	if len(statusCode) > 0 {
		code = statusCode[0]
	}

	return c.Status(code).JSON(Response{
		Success: true,
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// SuccessWithPagination sends a successful paginated response
func SuccessWithPagination(c *fiber.Ctx, message string, paginationResult interface{}, statusCode ...int) error {
	code := fiber.StatusOK
	if len(statusCode) > 0 {
		code = statusCode[0]
	}

	// Extract data and metadata from pagination result if it's from our pagination package
	var data interface{}
	var meta interface{}

	// Check if it's our PaginationResult type
	if pr, ok := paginationResult.(pagination.PaginationResult); ok {
		data = pr.Data
		meta = pr.Meta
	} else {
		// Otherwise, assume it's a custom structure with data and meta fields
		v := reflect.ValueOf(paginationResult)
		if v.Kind() == reflect.Struct {
			dataField := v.FieldByName("data")
			metaField := v.FieldByName("meta")

			if dataField.IsValid() {
				data = dataField.Interface()
			} else {
				data = paginationResult
			}

			if metaField.IsValid() {
				meta = metaField.Interface()
			}
		} else {
			// If not a struct, just use as data
			data = paginationResult
		}
	}

	return c.Status(code).JSON(struct {
		Success bool        `json:"success"`
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data"`
		Meta    interface{} `json:"meta,omitempty"`
	}{
		Success: true,
		Code:    code,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// Error sends an error response
func Error(c *fiber.Ctx, err error) error {
	if appErr, ok := err.(*errors.AppError); ok {
		return c.Status(appErr.HTTPCode).JSON(errors.ErrorResponse{
			Success: false,
			Code:    appErr.HTTPCode,
			Error:   appErr.Code,
			Message: appErr.Message,
			Details: appErr.Details,
		})
	}

	return c.Status(fiber.StatusInternalServerError).JSON(errors.ErrorResponse{
		Success: false,
		Code:    fiber.StatusInternalServerError,
		Error:   errors.ErrCodeInternalError,
		Message: err.Error(),
	})
}

// Created sends a successful created response
func Created(c *fiber.Ctx, message string, data interface{}) error {
	return Success(c, message, data, fiber.StatusCreated)
}

// BadRequest sends a bad request error response
func BadRequest(c *fiber.Ctx, message string, details interface{}) error {
	return c.Status(fiber.StatusBadRequest).JSON(errors.ErrorResponse{
		Success: false,
		Code:    fiber.StatusBadRequest,
		Error:   errors.ErrCodeBadRequest,
		Message: message,
		Details: details,
	})
}

// NotFound sends a not found error response
func NotFound(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusNotFound).JSON(errors.ErrorResponse{
		Success: false,
		Code:    fiber.StatusNotFound,
		Error:   errors.ErrCodeNotFound,
		Message: message,
	})
}

// MethodNotAllowed sends a method not allowed error response
func MethodNotAllowed(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusMethodNotAllowed).JSON(errors.ErrorResponse{
		Success: false,
		Code:    fiber.StatusMethodNotAllowed,
		Error:   errors.ErrCodeMethodNotAllowed,
		Message: message,
	})
}

// Unauthorized sends an unauthorized error response
func Unauthorized(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(errors.ErrorResponse{
		Success: false,
		Code:    fiber.StatusUnauthorized,
		Error:   errors.ErrCodeUnauthorized,
		Message: message,
	})
}

// Forbidden sends a forbidden error response
func Forbidden(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusForbidden).JSON(errors.ErrorResponse{
		Success: false,
		Code:    fiber.StatusForbidden,
		Error:   errors.ErrCodeForbidden,
		Message: message,
	})
}

// InternalServerError sends an internal server error response
func InternalServerError(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusInternalServerError).JSON(errors.ErrorResponse{
		Success: false,
		Code:    fiber.StatusInternalServerError,
		Error:   errors.ErrCodeInternalError,
		Message: message,
	})
}
