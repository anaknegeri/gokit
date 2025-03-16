package main

import (
	"fmt"
	"net/http"

	"github.com/anaknegeri/gokit"
	"github.com/anaknegeri/gokit/pkg/errors" // Import directly for demonstration
)

func main() {
	fmt.Println("GoKit Error Handling Examples")
	fmt.Println("============================")

	// Example 1: Basic error creation
	fmt.Println("\n1. Basic error creation:")
	basicError := errors.New("something went wrong")
	fmt.Printf("Basic error: %v\n", basicError)

	// Example 2: Application errors with HTTP codes
	fmt.Println("\n2. Application errors with HTTP codes:")
	appError := gokit.NewError(http.StatusBadRequest, "Invalid input")
	fmt.Printf("App error: %v (Code: %s, HTTP: %d)\n",
		appError, appError.Code, appError.HTTPCode)

	// Example 3: Error with details
	fmt.Println("\n3. Error with details:")
	detailsError := gokit.NewErrorWithDetails(
		http.StatusBadRequest,
		"Invalid query parameters",
		map[string]interface{}{
			"missing": []string{"user_id", "start_date"},
			"invalid": map[string]string{
				"limit": "must be a number",
			},
		},
	)
	fmt.Printf("Error with details: %v\n", detailsError)
	fmt.Printf("Details: %+v\n", detailsError.Details)

	// Example 4: Wrapping errors
	fmt.Println("\n4. Wrapping errors:")
	originalErr := errors.New("connection refused")
	wrappedErr := gokit.WrapError(
		originalErr,
		http.StatusServiceUnavailable,
		"Database connection failed",
	)
	fmt.Printf("Wrapped error: %v\n", wrappedErr)
	fmt.Printf("Original error: %v\n", wrappedErr.Internal)

	// Example 5: Standard error helpers
	fmt.Println("\n5. Standard error helpers:")
	notFoundErr := errors.NotFoundError("User")
	fmt.Printf("Not found error: %v\n", notFoundErr)

	unauthorizedErr := errors.UnauthorizedError("Invalid API key")
	fmt.Printf("Unauthorized error: %v\n", unauthorizedErr)

	badRequestErr := errors.BadRequestError("Missing required field")
	fmt.Printf("Bad request error: %v\n", badRequestErr)

	// Example 6: Domain-specific errors
	fmt.Println("\n6. Domain-specific errors:")
	fileErr := errors.FileNotFoundError("/path/to/file.txt")
	fmt.Printf("File error: %v\n", fileErr)

	dbErr := errors.DatabaseError(errors.New("SQL syntax error"))
	fmt.Printf("Database error: %v\n", dbErr)

	authErr := errors.InvalidCredentialsError()
	fmt.Printf("Auth error: %v\n", authErr)

	// Example 7: Custom error codes
	fmt.Println("\n7. Custom error codes:")
	customErr := errors.NewCustomError(
		http.StatusTeapot,
		"TEAPOT_ERROR",
		"I'm a teapot",
	)
	fmt.Printf("Custom error: %v\n", customErr)

	// Example 8: Error handling in functions
	fmt.Println("\n8. Error handling in functions:")
	if err := processUserData(""); err != nil {
		// Check specific error types
		var appErr *errors.AppError
		if errors.As(err, &appErr) && appErr.Code == errors.ErrCodeBadRequest {
			fmt.Println("Bad request error detected")
		}

		// Get specific error details
		if gokit.As(err, &appErr) {
			fmt.Printf("App error code: %s\n", appErr.Code)
			fmt.Printf("HTTP status: %d\n", appErr.HTTPCode)
			fmt.Printf("Error message: %s\n", appErr.Message)
		}

		// Format for API response
		errResp := gokit.FormatErrorResponse(err)
		fmt.Printf("API response: %+v\n", errResp)
	}

	// Example 9: Validation errors
	fmt.Println("\n9. Validation errors:")
	valErr := createValidationError()
	fmt.Printf("Validation error: %v\n", valErr)

	errorResponse := gokit.FormatErrorResponse(valErr)
	fmt.Printf("Error response: %+v\n", errorResponse)

	if details, ok := errorResponse.Details.([]gokit.ValidationError); ok {
		fmt.Println("Validation details:")
		for i, detail := range details {
			fmt.Printf("%d. Field: %s, Message: %s\n", i+1, detail.Field, detail.Message)
		}
	}
}

// processUserData demonstrates error handling in functions
func processUserData(userID string) error {
	if userID == "" {
		return errors.BadRequestError("User ID is required")
	}

	// Simulate database error
	dbError := gokit.New("connection timeout")
	return gokit.WrapError(
		dbError,
		http.StatusInternalServerError,
		"Failed to fetch user data",
	)
}

// createValidationError simulates a validation error
func createValidationError() *gokit.AppError {
	// Creating a mock validation error
	return errors.NewErrorWithDetails(
		http.StatusUnprocessableEntity,
		"Validation failed",
		[]errors.ValidationError{
			{
				Field:   "email",
				Message: "Invalid email format",
				Tag:     "email",
				Value:   "not-an-email",
			},
			{
				Field:   "age",
				Message: "Age must be at least 18",
				Tag:     "min",
				Value:   16,
				Param:   "18",
			},
			{
				Field:   "password",
				Message: "Password must be at least 8 characters long",
				Tag:     "min",
				Value:   "123",
				Param:   "8",
			},
		},
	)
}
