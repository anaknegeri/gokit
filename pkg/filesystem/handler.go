package filesystem

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	fserrors "github.com/anaknegeri/gokit/pkg/filesystem/errors"
)

// UploadHandlerConfig configures the upload handler
type UploadHandlerConfig struct {
	Provider     *Provider
	BasePath     string
	AllowedTypes []string
	MaxFileSize  int
	UseUUID      bool // Use UUID for filenames instead of original name
	TimeoutSecs  int  // Context timeout in seconds
}

// Response is a standardized API response
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// FileResponse is the file data structure for responses
type FileResponse struct {
	Name         string    `json:"name"`
	OriginalName string    `json:"originalName,omitempty"`
	Size         int64     `json:"size"`
	ContentType  string    `json:"contentType,omitempty"`
	URL          string    `json:"url"`
	Path         string    `json:"path"`
	LastModified time.Time `json:"lastModified,omitempty"`
	IsDirectory  bool      `json:"isDirectory,omitempty"`
}

// UploadHandler returns a Fiber handler for file uploads
func UploadHandler(config UploadHandlerConfig) fiber.Handler {
	if config.Provider == nil {
		panic("filesystem provider is required")
	}

	return func(c *fiber.Ctx) error {
		// Set timeout context
		ctx, cancel := context.WithTimeout(c.Context(), time.Duration(config.TimeoutSecs)*time.Second)
		defer cancel()

		// Get the uploaded file
		file, err := c.FormFile("file")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fserrors.FormatErrorResponse(
				fserrors.NewError(
					http.StatusBadRequest,
					"Failed to get uploaded file",
				),
			))
		}

		// Check file size
		if file.Size > int64(config.MaxFileSize) {
			return c.Status(fiber.StatusBadRequest).JSON(fserrors.FormatErrorResponse(
				fserrors.FileTooLargeError(file.Size, int64(config.MaxFileSize)),
			))
		}

		// Check file type if specified
		if len(config.AllowedTypes) > 0 {
			ext := strings.ToLower(filepath.Ext(file.Filename))
			allowed := false
			for _, allowedType := range config.AllowedTypes {
				if ext == allowedType {
					allowed = true
					break
				}
			}
			if !allowed {
				return c.Status(fiber.StatusBadRequest).JSON(fserrors.FormatErrorResponse(
					fserrors.InvalidFileTypeError(ext, config.AllowedTypes),
				))
			}
		}

		// Generate file path
		var filename string
		originalName := file.Filename
		if config.UseUUID {
			ext := filepath.Ext(file.Filename)
			filename = fmt.Sprintf("%s%s", uuid.New().String(), ext)
		} else {
			// Sanitize filename to prevent directory traversal
			filename = sanitizeFilename(file.Filename)
		}

		// Get custom path from form if provided, otherwise use default
		customPath := c.FormValue("path", "")
		if customPath != "" {
			// Sanitize custom path - remove any ".." to prevent directory traversal
			customPath = filepath.Clean(customPath)
			customPath = strings.TrimPrefix(customPath, "../")
		}

		// Combine with base path
		fullPath := filepath.Join(config.BasePath, customPath, filename)

		// Upload the file using the provider
		fileInfo, err := config.Provider.Upload(ctx, file, fullPath)
		if err != nil {
			// Convert to appropriate error response
			if appErr, ok := err.(*fserrors.AppError); ok {
				return c.Status(appErr.HTTPCode).JSON(fserrors.FormatErrorResponse(appErr))
			}

			return c.Status(fiber.StatusInternalServerError).JSON(fserrors.FormatErrorResponse(
				fserrors.WrapError(
					err,
					http.StatusInternalServerError,
					"Failed to upload file",
				),
			))
		}

		// Create response with additional info
		fileResponse := FileResponse{
			Name:         fileInfo.Name,
			OriginalName: originalName,
			Size:         fileInfo.Size,
			URL:          fileInfo.URL,
			Path:         filepath.Join(customPath, filename),
			ContentType:  fileInfo.ContentType,
			LastModified: fileInfo.LastModified,
		}

		return c.Status(fiber.StatusOK).JSON(Response{
			Success: true,
			Message: "File uploaded successfully",
			Data:    fileResponse,
		})
	}
}

// GetFileHandler returns a Fiber handler to serve files
func GetFileHandler(config UploadHandlerConfig) fiber.Handler {
	if config.Provider == nil {
		panic("filesystem provider is required")
	}

	return func(c *fiber.Ctx) error {
		// Set timeout context
		ctx, cancel := context.WithTimeout(c.Context(), time.Duration(config.TimeoutSecs)*time.Second)
		defer cancel()

		// Get the file path from URL parameter
		path := c.Params("*")
		if path == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fserrors.FormatErrorResponse(
				fserrors.NewError(
					http.StatusBadRequest,
					"File path is required",
				),
			))
		}

		// Sanitize path
		path = sanitizePath(path)

		// Combine with base path
		fullPath := filepath.Join(config.BasePath, path)

		// Check if file exists
		exists, err := config.Provider.Exists(ctx, fullPath)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fserrors.FormatErrorResponse(
				fserrors.WrapError(
					err,
					http.StatusInternalServerError,
					"Failed to check file existence",
				),
			))
		}

		if !exists {
			return c.Status(fiber.StatusNotFound).JSON(fserrors.FormatErrorResponse(
				fserrors.FileNotFoundError(path),
			))
		}

		// Get the file from storage
		file, fileInfo, err := config.Provider.Get(ctx, fullPath)
		if err != nil {
			if appErr, ok := err.(*fserrors.AppError); ok {
				return c.Status(appErr.HTTPCode).JSON(fserrors.FormatErrorResponse(appErr))
			}

			return c.Status(fiber.StatusInternalServerError).JSON(fserrors.FormatErrorResponse(
				fserrors.WrapError(
					err,
					http.StatusInternalServerError,
					"Failed to get file",
				),
			))
		}
		defer file.Close()

		// Get query parameters if any
		disposition := c.Query("disposition", "inline") // inline or attachment
		filename := c.Query("filename", fileInfo.Name)  // custom filename if provided

		// Set content type based on fileInfo
		contentType := fileInfo.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		c.Set("Content-Type", contentType)
		c.Set("Content-Disposition", fmt.Sprintf("%s; filename=\"%s\"", disposition, filename))
		c.Set("Cache-Control", "public, max-age=31536000") // 1 year cache

		return c.SendStream(file)
	}
}

// GetFileInfoHandler returns a Fiber handler to get file info without downloading
func GetFileInfoHandler(config UploadHandlerConfig) fiber.Handler {
	if config.Provider == nil {
		panic("filesystem provider is required")
	}

	return func(c *fiber.Ctx) error {
		// Set timeout context
		ctx, cancel := context.WithTimeout(c.Context(), time.Duration(config.TimeoutSecs)*time.Second)
		defer cancel()

		// Get the file path from URL parameter
		path := c.Params("*")
		if path == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fserrors.FormatErrorResponse(
				fserrors.NewError(
					http.StatusBadRequest,
					"File path is required",
				),
			))
		}

		// Sanitize path
		path = sanitizePath(path)

		// Combine with base path
		fullPath := filepath.Join(config.BasePath, path)

		// Get file info
		fileInfo, err := config.Provider.GetInfo(ctx, fullPath)
		if err != nil {
			if appErr, ok := err.(*fserrors.AppError); ok {
				return c.Status(appErr.HTTPCode).JSON(fserrors.FormatErrorResponse(appErr))
			}

			return c.Status(fiber.StatusInternalServerError).JSON(fserrors.FormatErrorResponse(
				fserrors.WrapError(
					err,
					http.StatusInternalServerError,
					"Failed to get file information",
				),
			))
		}

		// Create response
		fileResponse := FileResponse{
			Name:         fileInfo.Name,
			Size:         fileInfo.Size,
			URL:          fileInfo.URL,
			Path:         path,
			ContentType:  fileInfo.ContentType,
			LastModified: fileInfo.LastModified,
			IsDirectory:  fileInfo.IsDirectory,
		}

		return c.Status(fiber.StatusOK).JSON(Response{
			Success: true,
			Data:    fileResponse,
		})
	}
}

// DeleteFileHandler returns a Fiber handler to delete files
func DeleteFileHandler(config UploadHandlerConfig) fiber.Handler {
	if config.Provider == nil {
		panic("filesystem provider is required")
	}

	return func(c *fiber.Ctx) error {
		// Set timeout context
		ctx, cancel := context.WithTimeout(c.Context(), time.Duration(config.TimeoutSecs)*time.Second)
		defer cancel()

		// Get the file path from URL parameter
		path := c.Params("*")
		if path == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fserrors.FormatErrorResponse(
				fserrors.NewError(
					http.StatusBadRequest,
					"File path is required",
				),
			))
		}

		// Sanitize path
		path = sanitizePath(path)

		// Combine with base path
		fullPath := filepath.Join(config.BasePath, path)

		// Check if file exists
		exists, err := config.Provider.Exists(ctx, fullPath)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fserrors.FormatErrorResponse(
				fserrors.WrapError(
					err,
					http.StatusInternalServerError,
					"Failed to check file existence",
				),
			))
		}

		if !exists {
			return c.Status(fiber.StatusNotFound).JSON(fserrors.FormatErrorResponse(
				fserrors.FileNotFoundError(path),
			))
		}

		// Delete the file
		if err := config.Provider.Delete(ctx, fullPath); err != nil {
			if appErr, ok := err.(*fserrors.AppError); ok {
				return c.Status(appErr.HTTPCode).JSON(fserrors.FormatErrorResponse(appErr))
			}

			return c.Status(fiber.StatusInternalServerError).JSON(fserrors.FormatErrorResponse(
				fserrors.WrapError(
					err,
					http.StatusInternalServerError,
					"Failed to delete file",
				),
			))
		}

		return c.Status(fiber.StatusOK).JSON(Response{
			Success: true,
			Message: "File deleted successfully",
			Data: map[string]string{
				"path": path,
			},
		})
	}
}

// ListFilesHandler returns a Fiber handler to list files
func ListFilesHandler(config UploadHandlerConfig) fiber.Handler {
	if config.Provider == nil {
		panic("filesystem provider is required")
	}

	return func(c *fiber.Ctx) error {
		// Set timeout context
		ctx, cancel := context.WithTimeout(c.Context(), time.Duration(config.TimeoutSecs)*time.Second)
		defer cancel()

		// Get the directory path from URL parameter
		path := c.Params("*", "")

		// Sanitize path
		path = sanitizePath(path)

		// Combine with base path
		fullPath := filepath.Join(config.BasePath, path)

		// List files in the directory
		files, err := config.Provider.List(ctx, fullPath)
		if err != nil {
			if appErr, ok := err.(*fserrors.AppError); ok {
				return c.Status(appErr.HTTPCode).JSON(fserrors.FormatErrorResponse(appErr))
			}

			return c.Status(fiber.StatusInternalServerError).JSON(fserrors.FormatErrorResponse(
				fserrors.WrapError(
					err,
					http.StatusInternalServerError,
					"Failed to list files",
				),
			))
		}

		// Convert to response format
		var fileList []FileResponse
		for _, file := range files {
			relativePath := filepath.Join(path, file.Name)
			fileList = append(fileList, FileResponse{
				Name:         file.Name,
				Size:         file.Size,
				URL:          file.URL,
				Path:         relativePath,
				ContentType:  file.ContentType,
				LastModified: file.LastModified,
				IsDirectory:  file.IsDirectory,
			})
		}

		return c.Status(fiber.StatusOK).JSON(Response{
			Success: true,
			Data:    fileList,
		})
	}
}

// sanitizeFilename removes potentially dangerous characters from a filename
func sanitizeFilename(filename string) string {
	// Get only the base name without path components
	filename = filepath.Base(filename)

	// Replace any characters that could be problematic
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		"%", "_",
	)

	return replacer.Replace(filename)
}

// sanitizePath cleans a file path and prevents directory traversal
func sanitizePath(path string) string {
	// Clean the path to resolve any ".." elements
	path = filepath.Clean(path)

	// Remove any leading ".." or "/"
	path = strings.TrimPrefix(path, "../")
	path = strings.TrimPrefix(path, "/")

	return path
}
