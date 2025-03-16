package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	fserrors "github.com/anaknegeri/gokit/pkg/filesystem/errors"
)

// LocalStorage implements the Storage interface for local filesystem
type LocalStorage struct {
	basePath          string
	baseURL           string
	createDirectories bool
}

// LocalStorageConfig holds configuration for the local storage provider
type LocalStorageConfig struct {
	BasePath          string
	BaseURL           string
	CreateDirectories bool
}

// NewLocalStorage creates a new local storage provider
func NewLocalStorage(config LocalStorageConfig) (*LocalStorage, error) {
	basePath := config.BasePath
	if basePath == "" {
		basePath = "./storage/uploads"
	}

	// Ensure the base directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to create base directory: %s", basePath),
		)
	}

	return &LocalStorage{
		basePath:          basePath,
		baseURL:           config.BaseURL,
		createDirectories: config.CreateDirectories,
	}, nil
}

// Upload saves a file to local storage
func (ls *LocalStorage) Upload(ctx context.Context, file *multipart.FileHeader, path string) (*FileInfo, error) {
	fullPath := filepath.Join(ls.basePath, path)

	// Ensure the directory exists if createDirectories is true
	if ls.createDirectories {
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fserrors.WrapError(
				err,
				http.StatusInternalServerError,
				fmt.Sprintf("Failed to create directory: %s", dir),
			)
		}
	} else {
		// Check if directory exists
		dir := filepath.Dir(fullPath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return nil, fserrors.WrapError(
				err,
				http.StatusBadRequest,
				fmt.Sprintf("Directory does not exist: %s", dir),
			)
		}
	}

	// Check if file already exists
	if _, err := os.Stat(fullPath); err == nil {
		return nil, fserrors.NewCustomError(
			http.StatusConflict,
			fserrors.ErrCodeFileAlreadyExists,
			fmt.Sprintf("File already exists: %s", path),
		)
	}

	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		return nil, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			"Failed to open uploaded file",
		)
	}
	defer src.Close()

	// Create the destination file
	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to create destination file: %s", fullPath),
		)
	}
	defer dst.Close()

	// Copy the file contents
	if _, err = io.Copy(dst, src); err != nil {
		return nil, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			"Failed to copy file contents",
		)
	}

	// Get file info
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		return nil, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			"Failed to get file information",
		)
	}

	// Determine content type based on file extension
	contentType := ls.getContentType(filepath.Ext(fullPath))

	// Construct URL
	url := path
	if ls.baseURL != "" {
		url = fmt.Sprintf("%s/%s", strings.TrimRight(ls.baseURL, "/"), strings.TrimLeft(path, "/"))
	}

	return &FileInfo{
		Name:         filepath.Base(path),
		Size:         fileInfo.Size(),
		LastModified: fileInfo.ModTime(),
		URL:          url,
		ContentType:  contentType,
		IsDirectory:  false,
	}, nil
}

// Get retrieves a file from local storage
func (ls *LocalStorage) Get(ctx context.Context, path string) (io.ReadCloser, *FileInfo, error) {
	fullPath := filepath.Join(ls.basePath, path)

	// Check if file exists
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil, fserrors.FileNotFoundError(path)
		}
		return nil, nil, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to access file: %s", path),
		)
	}

	// Check if it's a directory
	if fileInfo.IsDir() {
		return nil, nil, fserrors.NewCustomError(
			http.StatusBadRequest,
			fserrors.ErrCodeInvalidPath,
			fmt.Sprintf("Path is a directory, not a file: %s", path),
		)
	}

	// Open the file
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, nil, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to open file: %s", path),
		)
	}

	// Determine content type based on file extension
	contentType := ls.getContentType(filepath.Ext(fullPath))

	// Construct URL
	url := path
	if ls.baseURL != "" {
		url = fmt.Sprintf("%s/%s", strings.TrimRight(ls.baseURL, "/"), strings.TrimLeft(path, "/"))
	}

	return file, &FileInfo{
		Name:         filepath.Base(path),
		Size:         fileInfo.Size(),
		LastModified: fileInfo.ModTime(),
		URL:          url,
		ContentType:  contentType,
		IsDirectory:  false,
	}, nil
}

// Delete removes a file from local storage
func (ls *LocalStorage) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(ls.basePath, path)

	// Check if file exists
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fserrors.FileNotFoundError(path)
		}
		return fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to access file: %s", path),
		)
	}

	// Check if it's a directory
	if fileInfo.IsDir() {
		return fserrors.NewCustomError(
			http.StatusBadRequest,
			fserrors.ErrCodeInvalidPath,
			fmt.Sprintf("Cannot delete a directory with this method: %s", path),
		)
	}

	// Remove the file
	if err := os.Remove(fullPath); err != nil {
		return fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to delete file: %s", path),
		)
	}

	return nil
}

// Exists checks if a file exists in local storage
func (ls *LocalStorage) Exists(ctx context.Context, path string) (bool, error) {
	fullPath := filepath.Join(ls.basePath, path)

	_, err := os.Stat(fullPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to check file existence: %s", path),
		)
	}

	return true, nil
}

// List returns a list of files from a directory in local storage
func (ls *LocalStorage) List(ctx context.Context, path string) ([]FileInfo, error) {
	fullPath := filepath.Join(ls.basePath, path)

	// Check if directory exists
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fserrors.FileNotFoundError(path)
		}
		return nil, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to access directory: %s", path),
		)
	}

	// If path is a file, return it as a single item
	if !fileInfo.IsDir() {
		contentType := ls.getContentType(filepath.Ext(fullPath))

		// Construct URL
		url := path
		if ls.baseURL != "" {
			url = fmt.Sprintf("%s/%s", strings.TrimRight(ls.baseURL, "/"), strings.TrimLeft(path, "/"))
		}

		return []FileInfo{
			{
				Name:         filepath.Base(path),
				Size:         fileInfo.Size(),
				LastModified: fileInfo.ModTime(),
				URL:          url,
				ContentType:  contentType,
				IsDirectory:  false,
			},
		}, nil
	}

	// Read directory contents
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to read directory: %s", path),
		)
	}

	var files []FileInfo
	for _, entry := range entries {
		entryInfo, err := entry.Info()
		if err != nil {
			// Skip entries with errors
			continue
		}

		relativePath := filepath.Join(path, entry.Name())

		// Construct URL
		url := relativePath
		if ls.baseURL != "" {
			url = fmt.Sprintf("%s/%s", strings.TrimRight(ls.baseURL, "/"), strings.TrimLeft(relativePath, "/"))
		}

		contentType := ""
		if !entryInfo.IsDir() {
			contentType = ls.getContentType(filepath.Ext(entry.Name()))
		}

		files = append(files, FileInfo{
			Name:         entry.Name(),
			Size:         entryInfo.Size(),
			LastModified: entryInfo.ModTime(),
			URL:          url,
			ContentType:  contentType,
			IsDirectory:  entryInfo.IsDir(),
		})
	}

	return files, nil
}

// GetInfo returns information about a file without fetching its contents
func (ls *LocalStorage) GetInfo(ctx context.Context, path string) (*FileInfo, error) {
	fullPath := filepath.Join(ls.basePath, path)

	// Check if file exists
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fserrors.FileNotFoundError(path)
		}
		return nil, fserrors.WrapError(
			err,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to get file information: %s", path),
		)
	}

	contentType := ""
	if !fileInfo.IsDir() {
		contentType = ls.getContentType(filepath.Ext(fullPath))
	}

	// Construct URL
	url := path
	if ls.baseURL != "" {
		url = fmt.Sprintf("%s/%s", strings.TrimRight(ls.baseURL, "/"), strings.TrimLeft(path, "/"))
	}

	return &FileInfo{
		Name:         filepath.Base(path),
		Size:         fileInfo.Size(),
		LastModified: fileInfo.ModTime(),
		URL:          url,
		ContentType:  contentType,
		IsDirectory:  fileInfo.IsDir(),
	}, nil
}

// getContentType returns the MIME content type based on file extension
func (ls *LocalStorage) getContentType(ext string) string {
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
