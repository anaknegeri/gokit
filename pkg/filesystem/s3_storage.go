package filesystem

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	fserrors "github.com/anaknegeri/gokit/pkg/filesystem/errors"
)

// S3Storage implements the Storage interface for AWS S3 and S3-compatible services
type S3Storage struct {
	client     *s3.Client
	uploader   *manager.Uploader
	downloader *manager.Downloader
	bucket     string
	basePrefix string
	baseURL    string
	region     string
}

// S3Config holds the configuration for S3Storage
type S3Config struct {
	// Standard AWS configuration (optional if using custom endpoint)
	AWSConfig aws.Config

	// Required for both AWS S3 and custom S3-compatible services
	Bucket     string
	BasePrefix string
	BaseURL    string // Custom URL for generating file URLs (optional)
	Region     string // Region (if not using AWSConfig)

	// Optional: For custom S3-compatible endpoints (like MinIO)
	Endpoint     string // Custom endpoint URL (e.g., "http://localhost:9000" for MinIO)
	AccessKey    string // Access key (if not using AWSConfig)
	SecretKey    string // Secret key (if not using AWSConfig)
	UseSSL       bool   // Whether to use SSL for custom endpoint
	UsePathStyle bool   // Whether to use path-style addressing (true for MinIO)
}

// NewS3Storage creates a new S3 storage provider (works with both AWS S3 and S3-compatible services)
func NewS3Storage(cfg S3Config) (*S3Storage, error) {
	var s3Client *s3.Client
	var err error

	// Check if using custom endpoint (like MinIO)
	if cfg.Endpoint != "" {
		// Create custom resolver for MinIO or other S3-compatible services
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               cfg.Endpoint,
				HostnameImmutable: true,
				SigningRegion:     cfg.Region,
			}, nil
		})

		// Use static credentials with custom endpoint
		customConfig := aws.Config{
			Credentials:                 credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
			Region:                      cfg.Region,
			EndpointResolverWithOptions: customResolver,
		}

		// Create client with custom options
		s3Client = s3.NewFromConfig(customConfig, func(o *s3.Options) {
			o.UsePathStyle = cfg.UsePathStyle // MinIO requires path-style addressing
		})
	} else {
		// Use standard AWS configuration if provided
		if cfg.AWSConfig.Region != "" {
			// Use the provided AWS configuration
			s3Client = s3.NewFromConfig(cfg.AWSConfig)
		} else {
			// Load default AWS configuration
			awsCfg, err := config.LoadDefaultConfig(context.TODO(),
				config.WithRegion(cfg.Region),
			)
			if err != nil {
				return nil, fserrors.WrapError(
					err,
					http.StatusInternalServerError,
					"Failed to load AWS configuration",
				)
			}
			s3Client = s3.NewFromConfig(awsCfg)
		}
	}

	// Validate bucket exists
	_, err = s3Client.HeadBucket(context.TODO(), &s3.HeadBucketInput{
		Bucket: aws.String(cfg.Bucket),
	})
	if err != nil {
		return nil, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to access S3 bucket '%s'", cfg.Bucket),
		)
	}

	uploader := manager.NewUploader(s3Client)
	downloader := manager.NewDownloader(s3Client)

	return &S3Storage{
		client:     s3Client,
		uploader:   uploader,
		downloader: downloader,
		bucket:     cfg.Bucket,
		basePrefix: cfg.BasePrefix,
		baseURL:    cfg.BaseURL,
		region:     cfg.Region,
	}, nil
}

// getFullKey returns the full S3 key with base prefix
func (s *S3Storage) getFullKey(path string) string {
	if s.basePrefix == "" {
		return path
	}
	return filepath.Join(s.basePrefix, path)
}

// getURL generates a URL for a file based on configuration
func (s *S3Storage) getURL(key string) string {
	if s.baseURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(s.baseURL, "/"), strings.TrimLeft(key, "/"))
	}

	// Default S3 URL if baseURL is not specified
	if s.region == "" {
		return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.bucket, key)
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, key)
}

// Upload saves a file to S3 storage
func (s *S3Storage) Upload(ctx context.Context, file *multipart.FileHeader, path string) (*FileInfo, error) {
	src, err := file.Open()
	if err != nil {
		return nil, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			"Failed to open uploaded file",
		)
	}
	defer src.Close()

	// Read file into memory to get content type
	buffer := &bytes.Buffer{}
	size, err := io.Copy(buffer, src)
	if err != nil {
		return nil, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			"Failed to read file",
		)
	}

	// Detect content type
	contentType := http.DetectContentType(buffer.Bytes())
	if strings.HasPrefix(contentType, "application/octet-stream") {
		// Use extension to determine content type if not detected
		contentType = getContentTypeByExt(filepath.Ext(file.Filename))
	}

	// Reset file pointer
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return nil, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			"Failed to reset file pointer",
		)
	}

	fullKey := s.getFullKey(path)

	// Check if file already exists
	exists, err := s.Exists(ctx, path)
	if err != nil {
		return nil, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			"Failed to check if file exists",
		)
	}
	if exists {
		return nil, fserrors.NewCustomError(
			http.StatusConflict,
			fserrors.ErrCodeFileAlreadyExists,
			fmt.Sprintf("File already exists: %s", path),
		)
	}

	// Upload the file to S3 with additional metadata
	output, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(fullKey),
		Body:        bytes.NewReader(buffer.Bytes()),
		ContentType: aws.String(contentType),
		Metadata: map[string]string{
			"OriginalFilename": file.Filename,
			"UploadedAt":       time.Now().Format(time.RFC3339),
		},
	})
	if err != nil {
		return nil, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to upload file to S3: %s", err.Error()),
		)
	}

	// Get file URL
	fileURL := output.Location
	if s.baseURL != "" {
		fileURL = s.getURL(fullKey)
	}

	return &FileInfo{
		Name:         filepath.Base(path),
		Size:         size,
		LastModified: time.Now(),
		URL:          fileURL,
		ContentType:  contentType,
		IsDirectory:  false,
	}, nil
}

// Get retrieves a file from S3 storage
func (s *S3Storage) Get(ctx context.Context, path string) (io.ReadCloser, *FileInfo, error) {
	fullKey := s.getFullKey(path)

	// Get file metadata first
	headOutput, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "404") {
			return nil, nil, fserrors.FileNotFoundError(path)
		}
		return nil, nil, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to get file metadata from S3: %s", path),
		)
	}

	// Get the object
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		return nil, nil, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to get file from S3: %s", path),
		)
	}

	// Get content type
	contentType := "application/octet-stream"
	if result.ContentType != nil {
		contentType = *result.ContentType
	}

	// Construct file info
	fileInfo := &FileInfo{
		Name:         filepath.Base(path),
		Size:         getInt64Value(headOutput.ContentLength),
		LastModified: getTimeValue(headOutput.LastModified),
		URL:          s.getURL(fullKey),
		ContentType:  contentType,
		IsDirectory:  false,
	}

	return result.Body, fileInfo, nil
}

// Delete removes a file from S3 storage
func (s *S3Storage) Delete(ctx context.Context, path string) error {
	fullKey := s.getFullKey(path)

	// Check if file exists
	exists, err := s.Exists(ctx, path)
	if err != nil {
		return fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to check if file exists: %s", path),
		)
	}
	if !exists {
		return fserrors.FileNotFoundError(path)
	}

	// Delete the file from S3
	_, err = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		return fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to delete file from S3: %s", path),
		)
	}

	return nil
}

// Exists checks if a file exists in S3 storage
func (s *S3Storage) Exists(ctx context.Context, path string) (bool, error) {
	fullKey := s.getFullKey(path)

	// Try to get file metadata
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		// Check if the error is a "not found" error
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "404") {
			return false, nil
		}

		return false, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to check if file exists in S3: %s", path),
		)
	}

	return true, nil
}

// List returns a list of files from a directory in S3 storage
func (s *S3Storage) List(ctx context.Context, path string) ([]FileInfo, error) {
	fullPrefix := s.getFullKey(path)
	if fullPrefix != "" && !strings.HasSuffix(fullPrefix, "/") {
		fullPrefix += "/"
	}

	// Handle the case where path is empty
	if path == "" || path == "/" {
		fullPrefix = s.basePrefix
		if fullPrefix != "" && !strings.HasSuffix(fullPrefix, "/") {
			fullPrefix += "/"
		}
	}

	// List objects in S3 with the given prefix
	output, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(s.bucket),
		Prefix:    aws.String(fullPrefix),
		Delimiter: aws.String("/"), // Use delimiter to simulate directories
	})
	if err != nil {
		return nil, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to list files in S3: %s", path),
		)
	}

	var files []FileInfo

	// Process "directories" (common prefixes)
	for _, prefix := range output.CommonPrefixes {
		prefixName := filepath.Base(strings.TrimSuffix(*prefix.Prefix, "/"))

		files = append(files, FileInfo{
			Name:         prefixName,
			Size:         0,
			LastModified: time.Now(), // S3 doesn't provide modification time for prefixes
			URL:          s.getURL(*prefix.Prefix),
			ContentType:  "application/directory", // Custom type for directories
			IsDirectory:  true,
		})
	}

	// Process files
	for _, obj := range output.Contents {
		// Skip the directory itself (which might be listed as a file)
		key := *obj.Key
		if strings.HasSuffix(key, "/") {
			continue
		}

		// Skip if key is the same as the prefix (happens when listing a specific file)
		if key == fullPrefix {
			continue
		}

		name := filepath.Base(key)
		contentType := getContentTypeByExt(filepath.Ext(name))

		files = append(files, FileInfo{
			Name:         name,
			Size:         *obj.Size,
			LastModified: *obj.LastModified,
			URL:          s.getURL(key),
			ContentType:  contentType,
			IsDirectory:  false,
		})
	}

	// Special case: If we're looking for a specific file, not a directory
	if len(files) == 0 && !strings.HasSuffix(fullPrefix, "/") {
		// Try to get the file directly
		fileInfo, err := s.GetInfo(ctx, path)
		if err == nil {
			return []FileInfo{*fileInfo}, nil
		}
	}

	return files, nil
}

// GetInfo returns information about a file without fetching its contents
func (s *S3Storage) GetInfo(ctx context.Context, path string) (*FileInfo, error) {
	fullKey := s.getFullKey(path)

	// Get file metadata
	headOutput, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "404") {
			return nil, fserrors.FileNotFoundError(path)
		}
		return nil, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to get file metadata from S3: %s", path),
		)
	}

	// Get content type
	contentType := "application/octet-stream"
	if headOutput.ContentType != nil {
		contentType = *headOutput.ContentType
	} else {
		// Try to determine content type from extension
		contentType = getContentTypeByExt(filepath.Ext(path))
	}

	return &FileInfo{
		Name:         filepath.Base(path),
		Size:         getInt64Value(headOutput.ContentLength),
		LastModified: getTimeValue(headOutput.LastModified),
		URL:          s.getURL(fullKey),
		ContentType:  contentType,
		IsDirectory:  false,
	}, nil
}

// Helper function to get content type from file extension
func getContentTypeByExt(ext string) string {
	ext = strings.ToLower(ext)

	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".txt":
		return "text/plain"
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".zip":
		return "application/zip"
	case ".tar":
		return "application/x-tar"
	case ".gz", ".gzip":
		return "application/gzip"
	case ".mp3":
		return "audio/mpeg"
	case ".mp4":
		return "video/mp4"
	case ".wav":
		return "audio/wav"
	case ".avi":
		return "video/x-msvideo"
	case ".mov":
		return "video/quicktime"
	case ".webm":
		return "video/webm"
	default:
		return "application/octet-stream"
	}
}

// Safe getter for int64 pointers
func getInt64Value(val *int64) int64 {
	if val == nil {
		return 0
	}
	return *val
}

// Safe getter for time pointers
func getTimeValue(val *time.Time) time.Time {
	if val == nil {
		return time.Time{}
	}
	return *val
}
