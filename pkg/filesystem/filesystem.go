package filesystem

import (
	"context"
	"io"
	"mime/multipart"
	"time"
)

// FileInfo represents metadata about a file
type FileInfo struct {
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"lastModified"`
	URL          string    `json:"url"`
	ContentType  string    `json:"contentType,omitempty"`
	IsDirectory  bool      `json:"isDirectory,omitempty"`
}

// Storage defines the interface that must be implemented by storage providers
type Storage interface {
	// Upload saves a file to storage and returns file info
	Upload(ctx context.Context, file *multipart.FileHeader, path string) (*FileInfo, error)

	// Get retrieves a file from storage
	Get(ctx context.Context, path string) (io.ReadCloser, *FileInfo, error)

	// Delete removes a file from storage
	Delete(ctx context.Context, path string) error

	// Exists checks if a file exists
	Exists(ctx context.Context, path string) (bool, error)

	// List returns a list of files from a directory
	List(ctx context.Context, path string) ([]FileInfo, error)

	// GetInfo returns information about a file without fetching its contents
	GetInfo(ctx context.Context, path string) (*FileInfo, error)
}

// Provider represents the filesystem provider that wraps a storage implementation
type Provider struct {
	storage Storage
}

// NewProvider creates a new filesystem provider with the specified storage
func NewProvider(storage Storage) *Provider {
	return &Provider{
		storage: storage,
	}
}

// Upload uploads a file to the storage
func (p *Provider) Upload(ctx context.Context, file *multipart.FileHeader, path string) (*FileInfo, error) {
	return p.storage.Upload(ctx, file, path)
}

// Get retrieves a file from storage
func (p *Provider) Get(ctx context.Context, path string) (io.ReadCloser, *FileInfo, error) {
	return p.storage.Get(ctx, path)
}

// Delete removes a file from storage
func (p *Provider) Delete(ctx context.Context, path string) error {
	return p.storage.Delete(ctx, path)
}

// Exists checks if a file exists
func (p *Provider) Exists(ctx context.Context, path string) (bool, error) {
	return p.storage.Exists(ctx, path)
}

// List returns a list of files from a directory
func (p *Provider) List(ctx context.Context, path string) ([]FileInfo, error) {
	return p.storage.List(ctx, path)
}

// GetInfo returns information about a file without fetching its contents
func (p *Provider) GetInfo(ctx context.Context, path string) (*FileInfo, error) {
	return p.storage.GetInfo(ctx, path)
}
