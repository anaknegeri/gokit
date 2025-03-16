# GoKit - A Comprehensive Go Development Toolkit

GoKit is a collection of utilities, helpers, and integrations that make building Go applications faster and easier. It provides a unified API for common tasks such as file storage, validation, error handling, pagination, and logging.

[![Go Reference](https://pkg.go.dev/badge/github.com/anaknegeri/gokit.svg)](https://pkg.go.dev/github.com/anaknegeri/gokit)
[![Go Report Card](https://goreportcard.com/badge/github.com/anaknegeri/gokit)](https://goreportcard.com/report/github.com/anaknegeri/gokit)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

- **📦 File Storage** - Unified interface for local and cloud (S3) file storage
- **✅ Validation** - Struct validation with helpful error messages
- **🚨 Error Handling** - Standardized error system with HTTP integration
- **📄 Pagination** - Easy pagination for database queries
- **📝 Logging** - Flexible logging with multiple output formats
- **🌐 API Responses** - Consistent API response formats
- **🔥 Fiber Integration** - Ready-to-use handlers for the Fiber web framework

## Installation

```bash
go get github.com/anaknegeri/gokit
```

## Quick Start

```go
package main

import (
	"context"
	"log"

	"github.com/anaknegeri/gokit"
	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// Initialize context
	ctx := context.Background()

	// Initialize filesystem
	fs, err := gokit.NewFilesystem(ctx)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Initialize logger
	logger := gokit.InitLogger()
	logger.Info("Application starting")

	// Initialize database and pagination
	db, _ := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	paginator := gokit.NewPaginator(db)

	// Initialize validator
	validator := gokit.NewValidator()

	// Create Fiber app
	app := fiber.New()

	// Set up file upload route
	app.Post("/upload", fs.GetUploadHandler()("uploads"))

	// Start server
	app.Listen(":3000")
}
```

## Modules

### File Storage

GoKit provides a consistent interface for file operations across different storage backends:

```go
// Upload a file
fileInfo, err := fs.Provider.Upload(ctx, fileHeader, "path/to/save.jpg")

// Get a file
file, info, err := fs.Provider.Get(ctx, "path/to/file.jpg")

// Delete a file
err := fs.Provider.Delete(ctx, "path/to/file.jpg")

// Check if a file exists
exists, err := fs.Provider.Exists(ctx, "path/to/file.jpg")

// List files in a directory
files, err := fs.Provider.List(ctx, "directory")

// Get file info without downloading
info, err := fs.Provider.GetInfo(ctx, "path/to/file.jpg")
```

### Validation

Validate structs with detailed error messages:

```go
type User struct {
    Name     string `json:"name" validate:"required"`
    Email    string `json:"email" validate:"required,email"`
    Age      int    `json:"age" validate:"gte=18,lte=120"`
    Password string `json:"password" validate:"required,min=8"`
}

// Create validator
validator := gokit.NewValidator()

// Validate struct
user := User{...}
err := validator.Struct(user)
if err != nil {
    // Format validation errors
    validationErrors := gokit.ValidatorError(err)
    // Handle errors...
}
```

### Error Handling

Standardized error system:

```go
// Create basic errors
err := gokit.NewError(http.StatusBadRequest, "Invalid input")

// Create errors with details
err := gokit.NewErrorWithDetails(
    http.StatusBadRequest,
    "Validation failed",
    validationErrors,
)

// Wrap existing errors
dbError := errors.New("connection timeout")
err := gokit.WrapError(
    dbError,
    http.StatusInternalServerError,
    "Database error",
)

// Domain-specific errors
err := gokit.FileNotFoundError(filepath)
err := gokit.InvalidCredentialsError()
```

### Pagination

Easy pagination for database queries:

```go
// Create paginator
paginator := gokit.NewPaginator(db)

// Get pagination params from request
params := gokit.PaginationParams{
    Page:     c.QueryInt("page", 1),
    PageSize: c.QueryInt("pageSize", 10),
}

// Paginate results
var users []User
result, err := paginator.Paginate(params, &users)
```

### Logging

Flexible logging with multiple output formats:

```go
// Create logger
logger := gokit.NewLogger()

// Log at different levels
logger.Debug("Debug message")
logger.Info("Info message")
logger.Warn("Warning message")
logger.Error("Error message")

// Formatted logging
logger.Infof("User %s logged in", username)

// JSON logging
logger.Infoj(map[string]interface{}{
    "action":   "user_login",
    "username": username,
    "success":  true,
})
```

### API Responses

Consistent API response formats:

```go
// Success response
return gokit.SuccessResponse(c, "User created", user)

// Error response
return gokit.ErrorResponseWithErr(c, err)

// Standard responses
return gokit.NotFoundResponse(c, "User not found")
return gokit.BadRequestResponse(c, "Invalid input", details)
return gokit.UnauthorizedResponse(c, "Invalid credentials")
```

## Configuration

GoKit can be configured using environment variables:

```bash
# File Storage
STORAGE_TYPE=local        # or "s3"
UPLOAD_STORAGE_PATH=./uploads
UPLOAD_MAX_SIZE=20        # Max size in MB
ALLOWED_FILE_TYPES=.jpg,.jpeg,.png,.pdf

# S3 Storage
S3_ENDPOINT=https://s3.amazonaws.com
S3_ACCESS_KEY=your-access-key
S3_SECRET_KEY=your-secret-key
S3_BUCKET=your-bucket
S3_PREFIX=uploads
S3_REGION=us-east-1
S3_USE_SSL=true

# Logging
LOG_LEVEL=info            # debug, info, warn, error, fatal
LOG_OUTPUT=stdout         # stdout, stderr, file
LOG_FILE_PATH=./logs/app.log
LOG_PREFIX=[APP]
```

## Examples

See the [examples](./examples) directory for more comprehensive examples:

- [Simple Usage](./examples/simple/main.go)
- [Fiber Web Server](./examples/fiber/main.go)
- [Validation](./examples/validation/main.go)
- [Error Handling](./examples/errors/main.go)
- [Logging](./examples/logging/main.go)

## License

This project is licensed under the MIT License - see the LICENSE file for details.
