package filesystem

import (
	"context"
	"fmt"
	"net/http"

	fserrors "github.com/anaknegeri/gokit/pkg/filesystem/errors"
	"github.com/aws/aws-sdk-go-v2/config"
)

// NewStorageProvider creates a storage provider based on the provided configuration
func NewStorageProvider(ctx context.Context, cfg Config) (*Provider, error) {
	// Validate config
	if errors := cfg.Validate(); len(errors) > 0 {
		return nil, fserrors.NewErrorWithDetails(
			http.StatusBadRequest,
			"Invalid filesystem configuration",
			errors,
		)
	}

	var storage Storage
	switch cfg.StorageType {
	case "s3":
		// Create S3 storage
		var s3Config S3Config

		if cfg.S3Endpoint != "" {
			// S3-compatible service with custom endpoint (like MinIO)
			s3Config = S3Config{
				Endpoint:     cfg.S3Endpoint,
				AccessKey:    cfg.S3AccessKey,
				SecretKey:    cfg.S3SecretKey,
				Bucket:       cfg.S3Bucket,
				BasePrefix:   cfg.S3BasePrefix,
				BaseURL:      cfg.S3BaseURL,
				Region:       cfg.S3Region,
				UseSSL:       cfg.S3UseSSL,
				UsePathStyle: cfg.S3PathStyle,
			}
		} else {
			// Standard AWS S3
			awsCfg, err := config.LoadDefaultConfig(ctx,
				config.WithRegion(cfg.S3Region),
			)
			if err != nil {
				return nil, fserrors.WrapError(
					err,
					http.StatusInternalServerError,
					"Unable to load AWS SDK config",
				)
			}

			s3Config = S3Config{
				AWSConfig:  awsCfg,
				Bucket:     cfg.S3Bucket,
				BasePrefix: cfg.S3BasePrefix,
				BaseURL:    cfg.S3BaseURL,
				Region:     cfg.S3Region,
			}
		}

		s3Storage, err := NewS3Storage(s3Config)
		if err != nil {
			return nil, err
		}
		storage = s3Storage

	case "local", "":
		// Create local storage
		localConfig := LocalStorageConfig{
			BasePath:          cfg.LocalStoragePath,
			BaseURL:           cfg.LocalBaseURL,
			CreateDirectories: cfg.CreateLocalDirs,
		}

		localStorage, err := NewLocalStorage(localConfig)
		if err != nil {
			return nil, fserrors.WrapError(
				err,
				http.StatusInternalServerError,
				"Failed to initialize local storage",
			)
		}
		storage = localStorage

	default:
		return nil, fserrors.NewError(
			http.StatusBadRequest,
			fmt.Sprintf("Unsupported storage type: %s", cfg.StorageType),
		)
	}

	provider := NewProvider(storage)
	return provider, nil
}

// GetUploadHandlerConfig creates a handler configuration from the filesystem config
func GetUploadHandlerConfig(provider *Provider, cfg Config) UploadHandlerConfig {
	handlerConfig := UploadHandlerConfig{
		Provider:     provider,
		BasePath:     "",
		AllowedTypes: cfg.AllowedFileTypes,
		MaxFileSize:  cfg.UploadMaxSizeMB * 1024 * 1024,
		UseUUID:      cfg.UseUUID,
		TimeoutSecs:  cfg.TimeoutSecs,
	}

	return handlerConfig
}
