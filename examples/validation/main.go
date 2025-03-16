package main

import (
	"fmt"
	"reflect"

	"github.com/anaknegeri/gokit"
)

// User represents a user in the system
type User struct {
	ID        int    `json:"id"`
	FirstName string `json:"firstName" validate:"required"`
	LastName  string `json:"lastName" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8,max=32"`
	Age       int    `json:"age" validate:"gte=18,lte=120"`
	Country   string `json:"country" validate:"required,iso3166_1_alpha2"`
	Website   string `json:"website" validate:"omitempty,url"`
	Role      string `json:"role" validate:"required,oneof=admin user guest"`
}

// Address demonstrates nested struct validation
type Address struct {
	User      User   `json:"user" validate:"required"`
	Street    string `json:"street" validate:"required"`
	City      string `json:"city" validate:"required"`
	ZipCode   string `json:"zipCode" validate:"required,numeric,len=5"`
	Country   string `json:"country" validate:"required,iso3166_1_alpha2"`
	IsBilling bool   `json:"isBilling"`
}

func main() {
	// Initialize the validator
	validator := gokit.NewValidator()

	// Example 1: Valid user
	validUser := User{
		ID:        1,
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john.doe@example.com",
		Password:  "password123",
		Age:       30,
		Country:   "US",
		Website:   "https://example.com",
		Role:      "user",
	}

	err := validator.Struct(validUser)
	if err != nil {
		fmt.Println("Validation error (should not happen):", err)
	} else {
		fmt.Println("Valid user passed validation ✓")
	}

	// Example 2: Invalid user (missing required fields)
	invalidUser := User{
		ID:      2,
		Email:   "invalid-email",
		Age:     16,           // Too young
		Country: "USA",        // Invalid country code
		Role:    "superadmin", // Not in allowed roles
	}

	err = validator.Struct(invalidUser)
	if err != nil {
		// Process validation errors
		appErr := gokit.ValidatorError(err)
		fmt.Println("\nInvalid user validation errors:")
		if details, ok := appErr.Details.([]gokit.ValidationError); ok {
			for i, detail := range details {
				fmt.Printf("%d. Field: %s, Message: %s\n", i+1, detail.Field, detail.Message)
			}
		} else {
			fmt.Println("Validation failed:", appErr.Message)
		}
	}

	// Example 3: Nested struct validation
	address := Address{
		User:      validUser,
		Street:    "123 Main St",
		City:      "New York",
		ZipCode:   "10001",
		Country:   "US",
		IsBilling: true,
	}

	err = validator.Struct(address)
	if err != nil {
		fmt.Println("\nValidation error (should not happen):", err)
	} else {
		fmt.Println("\nValid address passed validation ✓")
	}

	// Example 4: Invalid nested struct
	invalidAddress := Address{
		User:      invalidUser, // Invalid user
		Street:    "",          // Missing required field
		City:      "New York",
		ZipCode:   "ABC12", // Not numeric
		Country:   "USA",   // Invalid country code
		IsBilling: false,
	}

	err = validator.Struct(invalidAddress)
	if err != nil {
		// Process validation errors
		appErr := gokit.ValidatorError(err)
		fmt.Println("\nInvalid address validation errors:")
		if details, ok := appErr.Details.([]gokit.ValidationError); ok {
			for i, detail := range details {
				fmt.Printf("%d. Field: %s, Message: %s\n", i+1, detail.Field, detail.Message)
			}
		} else {
			fmt.Println("Validation failed:", appErr.Message)
		}
	}

	// Example 5: Custom validation
	fmt.Println("\nAdding custom 'username' validation...")

	err = validator.RegisterValidation("username", func(fl interface{}) bool {
		// Get the field value as string
		fieldValue := reflect.ValueOf(fl).MethodByName("Field").Call(nil)[0]
		username := fieldValue.String()

		// Username must be 3-20 characters, alphanumeric
		if len(username) < 3 || len(username) > 20 {
			return false
		}
		for _, r := range username {
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
				return false
			}
		}
		return true
	})

	if err != nil {
		fmt.Println("Error registering custom validation:", err)
	}

	// UserWithUsername demonstrates custom validation
	type UserWithUsername struct {
		Username string `json:"username" validate:"required,username"`
		Email    string `json:"email" validate:"required,email"`
	}

	validUsernameUser := UserWithUsername{
		Username: "john_doe123",
		Email:    "john.doe@example.com",
	}

	invalidUsernameUser := UserWithUsername{
		Username: "jo", // Too short
		Email:    "john.doe@example.com",
	}

	err = validator.Struct(validUsernameUser)
	if err != nil {
		fmt.Println("Validation error (should not happen):", err)
	} else {
		fmt.Println("Valid username passed validation ✓")
	}

	err = validator.Struct(invalidUsernameUser)
	if err != nil {
		// Process validation errors
		appErr := gokit.ValidatorError(err)
		fmt.Println("\nInvalid username validation errors:")
		if details, ok := appErr.Details.([]gokit.ValidationError); ok {
			for i, detail := range details {
				fmt.Printf("%d. Field: %s, Message: %s\n", i+1, detail.Field, detail.Message)
			}
		} else {
			fmt.Println("Validation failed:", appErr.Message)
		}
	}
}
