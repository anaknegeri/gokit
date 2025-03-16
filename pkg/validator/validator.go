// Package validator provides validation utilities for structs
package validator

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator defines the interface for validation
type Validator interface {
	// Struct validates a struct and returns an error if validation fails
	Struct(s interface{}) error

	// RegisterValidation registers a custom validation function
	RegisterValidation(tag string, fn interface{}) error

	// RegisterTagNameFunc sets a function to get the field name from a struct tag
	RegisterTagNameFunc(fn func(fld reflect.StructField) string)
}

// validatorImpl implements the Validator interface
type validatorImpl struct {
	validate *validator.Validate
}

// NewValidator creates a new validator instance
func NewValidator() Validator {
	v := validator.New()

	// By default, use JSON tag names in validation errors
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return &validatorImpl{
		validate: v,
	}
}

// Struct validates a struct and returns an error if validation fails
func (v *validatorImpl) Struct(s interface{}) error {
	return v.validate.Struct(s)
}

// RegisterValidation registers a custom validation function
func (v *validatorImpl) RegisterValidation(tag string, fn interface{}) error {
	validatorFunc, ok := fn.(validator.Func)
	if !ok {
		// Try to adapt the function
		adaptedFunc := validator.Func(func(fl validator.FieldLevel) bool {
			// Call the original function with the FieldLevel
			result := reflect.ValueOf(fn).Call([]reflect.Value{reflect.ValueOf(fl)})
			if len(result) == 1 && result[0].Kind() == reflect.Bool {
				return result[0].Bool()
			}
			return false
		})
		return v.validate.RegisterValidation(tag, adaptedFunc)
	}
	return v.validate.RegisterValidation(tag, validatorFunc)
}

// RegisterTagNameFunc sets a function to get the field name from a struct tag
func (v *validatorImpl) RegisterTagNameFunc(fn func(fld reflect.StructField) string) {
	v.validate.RegisterTagNameFunc(fn)
}
