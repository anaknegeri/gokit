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

type S3Storage struct {
	client     *s3.Client
	uploader   *manager.Uploader
	downloader *manager.Downloader
	bucket     string
	basePrefix string
	baseURL    string
	region     string
}

type S3Config struct {
	AWSConfig    aws.Config
	Bucket       string
	BasePrefix   string
	BaseURL      string
	Region       string
	Endpoint     string
	AccessKey    string
	SecretKey    string
	UseSSL       bool
	UsePathStyle bool
}

func NewS3Storage(cfg S3Config) (*S3Storage, error) {
	var s3Client *s3.Client
	var err error

	if cfg.Endpoint != "" {
		customConfig := aws.Config{
			Credentials: credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
			Region:      cfg.Region,
		}

		s3Client = s3.NewFromConfig(customConfig, func(o *s3.Options) {
			o.UsePathStyle = cfg.UsePathStyle
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	} else {
		if cfg.AWSConfig.Region != "" {
			s3Client = s3.NewFromConfig(cfg.AWSConfig)
		} else {
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

func (s *S3Storage) getFullKey(path string) string {
	if s.basePrefix == "" {
		return path
	}
	return filepath.Join(s.basePrefix, path)
}

func (s *S3Storage) getURL(key string) string {
	if s.baseURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(s.baseURL, "/"), strings.TrimLeft(key, "/"))
	}

	if s.region == "" {
		return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.bucket, key)
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, key)
}

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

	buffer := &bytes.Buffer{}
	size, err := io.Copy(buffer, src)
	if err != nil {
		return nil, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			"Failed to read file",
		)
	}

	contentType := http.DetectContentType(buffer.Bytes())
	if strings.HasPrefix(contentType, "application/octet-stream") {
		contentType = getContentTypeByExt(filepath.Ext(file.Filename))
	}

	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return nil, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			"Failed to reset file pointer",
		)
	}

	fullKey := s.getFullKey(path)

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

func (s *S3Storage) Get(ctx context.Context, path string) (io.ReadCloser, *FileInfo, error) {
	fullKey := s.getFullKey(path)

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

	contentType := "application/octet-stream"
	if result.ContentType != nil {
		contentType = *result.ContentType
	}

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

func (s *S3Storage) Delete(ctx context.Context, path string) error {
	fullKey := s.getFullKey(path)

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

func (s *S3Storage) Exists(ctx context.Context, path string) (bool, error) {
	fullKey := s.getFullKey(path)

	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
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

func (s *S3Storage) List(ctx context.Context, path string) ([]FileInfo, error) {
	fullPrefix := s.getFullKey(path)
	if fullPrefix != "" && !strings.HasSuffix(fullPrefix, "/") {
		fullPrefix += "/"
	}

	if path == "" || path == "/" {
		fullPrefix = s.basePrefix
		if fullPrefix != "" && !strings.HasSuffix(fullPrefix, "/") {
			fullPrefix += "/"
		}
	}

	output, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(s.bucket),
		Prefix:    aws.String(fullPrefix),
		Delimiter: aws.String("/"),
	})
	if err != nil {
		return nil, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to list files in S3: %s", path),
		)
	}

	var files []FileInfo

	for _, prefix := range output.CommonPrefixes {
		prefixName := filepath.Base(strings.TrimSuffix(*prefix.Prefix, "/"))

		files = append(files, FileInfo{
			Name:         prefixName,
			Size:         0,
			LastModified: time.Now(),
			URL:          s.getURL(*prefix.Prefix),
			ContentType:  "application/directory",
			IsDirectory:  true,
		})
	}

	for _, obj := range output.Contents {
		key := *obj.Key
		if strings.HasSuffix(key, "/") {
			continue
		}

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

	if len(files) == 0 && !strings.HasSuffix(fullPrefix, "/") {
		fileInfo, err := s.GetInfo(ctx, path)
		if err == nil {
			return []FileInfo{*fileInfo}, nil
		}
	}

	return files, nil
}

func (s *S3Storage) GetInfo(ctx context.Context, path string) (*FileInfo, error) {
	fullKey := s.getFullKey(path)

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

	contentType := "application/octet-stream"
	if headOutput.ContentType != nil {
		contentType = *headOutput.ContentType
	} else {
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

func getInt64Value(val *int64) int64 {
	if val == nil {
		return 0
	}
	return *val
}

func getTimeValue(val *time.Time) time.Time {
	if val == nil {
		return time.Time{}
	}
	return *val
}
