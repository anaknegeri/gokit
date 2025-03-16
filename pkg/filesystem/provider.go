package filesystem

import (
	"context"
	"net/http"

	fserrors "github.com/anaknegeri/gokit/pkg/filesystem/errors"
)

// FilesystemProvider is a high-level provider that integrates configuration
// and handlers for easy usage in applications
type FilesystemProvider struct {
	Provider      *Provider
	HandlerConfig UploadHandlerConfig
	Config        Config
}

// NewFilesystemProvider creates a new filesystem provider with configuration
// from environment variables
func NewFilesystemProvider(ctx context.Context) (*FilesystemProvider, error) {
	// Load configuration from environment
	config := NewConfigFromEnv()

	// Validate config
	if errors := config.Validate(); len(errors) > 0 {
		return nil, fserrors.NewErrorWithDetails(
			http.StatusBadRequest,
			"Invalid filesystem configuration",
			errors,
		)
	}

	// Create storage provider
	provider, err := NewStorageProvider(ctx, config)
	if err != nil {
		return nil, err
	}

	// Create handler config
	handlerConfig := GetUploadHandlerConfig(provider, config)

	return &FilesystemProvider{
		Provider:      provider,
		HandlerConfig: handlerConfig,
		Config:        config,
	}, nil
}

// NewFilesystemProviderWithConfig creates a new filesystem provider with
// explicit configuration
func NewFilesystemProviderWithConfig(ctx context.Context, config Config) (*FilesystemProvider, error) {
	// Validate config
	if errors := config.Validate(); len(errors) > 0 {
		return nil, fserrors.NewErrorWithDetails(
			http.StatusBadRequest,
			"Invalid filesystem configuration",
			errors,
		)
	}

	// Create storage provider
	provider, err := NewStorageProvider(ctx, config)
	if err != nil {
		return nil, err
	}

	// Create handler config
	handlerConfig := GetUploadHandlerConfig(provider, config)

	return &FilesystemProvider{
		Provider:      provider,
		HandlerConfig: handlerConfig,
		Config:        config,
	}, nil
}

// GetUploadHandler returns a handler for file uploads
// Takes a base path to be prepended to file paths
func (f *FilesystemProvider) GetUploadHandler() func(string) interface{} {
	return func(basePath string) interface{} {
		config := f.HandlerConfig
		config.BasePath = basePath
		return UploadHandler(config)
	}
}

// GetFileHandler returns a handler to serve files
// Takes a base path to be prepended to file paths
func (f *FilesystemProvider) GetFileHandler() func(string) interface{} {
	return func(basePath string) interface{} {
		config := f.HandlerConfig
		config.BasePath = basePath
		return GetFileHandler(config)
	}
}

// GetFileInfoHandler returns a handler to get file information
// Takes a base path to be prepended to file paths
func (f *FilesystemProvider) GetFileInfoHandler() func(string) interface{} {
	return func(basePath string) interface{} {
		config := f.HandlerConfig
		config.BasePath = basePath
		return GetFileInfoHandler(config)
	}
}

// GetDeleteFileHandler returns a handler to delete files
// Takes a base path to be prepended to file paths
func (f *FilesystemProvider) GetDeleteFileHandler() func(string) interface{} {
	return func(basePath string) interface{} {
		config := f.HandlerConfig
		config.BasePath = basePath
		return DeleteFileHandler(config)
	}
}

// GetListFilesHandler returns a handler to list files
// Takes a base path to be prepended to file paths
func (f *FilesystemProvider) GetListFilesHandler() func(string) interface{} {
	return func(basePath string) interface{} {
		config := f.HandlerConfig
		config.BasePath = basePath
		return ListFilesHandler(config)
	}
}
