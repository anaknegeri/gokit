package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/anaknegeri/gokit"
	"github.com/anaknegeri/gokit/pkg/filesystem"
)

var (
	operation   = flag.String("op", "", "Operation: upload, get, exists, list, delete, info")
	src         = flag.String("src", "", "Source file path (for upload)")
	dest        = flag.String("dest", "", "Destination path in storage")
	dir         = flag.String("dir", "", "Directory to list files from")
	storageType = flag.String("storage", "local", "Storage type: local or s3")
	localPath   = flag.String("local-path", "./storage", "Local storage path")
	s3Endpoint  = flag.String("s3-endpoint", "", "S3 endpoint URL")
	s3Region    = flag.String("s3-region", "", "S3 region")
	s3Bucket    = flag.String("s3-bucket", "", "S3 bucket name")
	s3Prefix    = flag.String("s3-prefix", "", "S3 prefix path")
)

func main() {
	flag.Parse()

	// Create configuration
	config := filesystem.DefaultConfig()
	config.StorageType = *storageType

	// Set config based on storage type
	if *storageType == "local" {
		config.LocalStoragePath = *localPath
	} else if *storageType == "s3" {
		if *s3Bucket == "" {
			log.Fatal("S3 bucket name is required for S3 storage")
		}

		config.S3Endpoint = *s3Endpoint
		config.S3Region = *s3Region
		config.S3Bucket = *s3Bucket
		config.S3BasePrefix = *s3Prefix

		// Get S3 credentials from environment
		config.S3AccessKey = os.Getenv("S3_ACCESS_KEY")
		config.S3SecretKey = os.Getenv("S3_SECRET_KEY")

		if config.S3Endpoint != "" && (config.S3AccessKey == "" || config.S3SecretKey == "") {
			log.Fatal("S3_ACCESS_KEY and S3_SECRET_KEY environment variables are required for custom S3 endpoints")
		}
	}

	// Initialize context
	ctx := context.Background()

	// Create provider
	provider, err := gokit.NewFilesystemWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("Error creating storage provider: %v", err)
	}

	// Execute operation
	switch *operation {
	case "upload":
		if *src == "" || *dest == "" {
			log.Fatal("Source and destination paths are required for upload")
		}
		uploadFile(ctx, provider.Provider, *src, *dest)

	case "get":
		if *dest == "" {
			log.Fatal("Destination path is required for get")
		}
		getFile(ctx, provider.Provider, *dest)

	case "exists":
		if *dest == "" {
			log.Fatal("Destination path is required for exists")
		}
		checkExists(ctx, provider.Provider, *dest)

	case "list":
		if *dir == "" {
			*dir = "/"
		}
		listFiles(ctx, provider.Provider, *dir)

	case "delete":
		if *dest == "" {
			log.Fatal("Destination path is required for delete")
		}
		deleteFile(ctx, provider.Provider, *dest)

	case "info":
		if *dest == "" {
			log.Fatal("Destination path is required for info")
		}
		getFileInfo(ctx, provider.Provider, *dest)

	default:
		fmt.Println("GoKit CLI Tool")
		fmt.Println("====================")
		fmt.Println("Usage:")
		flag.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Println("  Upload:  gokit -op upload -src /path/to/file.txt -dest uploads/file.txt")
		fmt.Println("  Get:     gokit -op get -dest uploads/file.txt")
		fmt.Println("  Exists:  gokit -op exists -dest uploads/file.txt")
		fmt.Println("  List:    gokit -op list -dir uploads")
		fmt.Println("  Delete:  gokit -op delete -dest uploads/file.txt")
		fmt.Println("  Info:    gokit -op info -dest uploads/file.txt")
		fmt.Println("\nStorage Types:")
		fmt.Println("  Local:   gokit -storage local -local-path ./storage")
		fmt.Println("  S3:      gokit -storage s3 -s3-bucket my-bucket -s3-region us-east-1")
		fmt.Println("  MinIO:   gokit -storage s3 -s3-endpoint http://localhost:9000 -s3-bucket my-bucket")
	}
}

// uploadFile uploads a file to storage
func uploadFile(ctx context.Context, provider *filesystem.Provider, srcPath, destPath string) {
	// This is a command-line utility and not a web handler, so we need to
	// create our own multipart.FileHeader from the source file

	// Open the file
	file, err := os.Open(srcPath)
	if err != nil {
		log.Fatalf("Error opening source file: %v", err)
	}
	defer file.Close()

	// Get file stats
	stats, err := file.Stat()
	if err != nil {
		log.Fatalf("Error getting file stats: %v", err)
	}

	// Read file content into memory
	content := make([]byte, stats.Size())
	_, err = file.Read(content)
	if err != nil {
		log.Fatalf("Error reading file content: %v", err)
	}

	// Create temporary file for upload
	tempDir, err := os.MkdirTemp("", "gokit")
	if err != nil {
		log.Fatalf("Error creating temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempFile := filepath.Join(tempDir, filepath.Base(srcPath))
	if err := os.WriteFile(tempFile, content, 0644); err != nil {
		log.Fatalf("Error writing temp file: %v", err)
	}

	// Since we don't have an actual HTTP multipart file from a form,
	// we'll have to create a local file and use it instead

	fmt.Printf("Uploading %s to %s...\n", srcPath, destPath)

	// Implementation note: In a real web application, we'd get a proper
	// multipart.FileHeader from the form/request. This CLI implementation
	// is just to demonstrate the concept.

	// For CLI tool, we'll simply copy the file to the destination if it's local storage
	if *storageType == "local" {
		destFullPath := filepath.Join(*localPath, destPath)
		destDir := filepath.Dir(destFullPath)

		if err := os.MkdirAll(destDir, 0755); err != nil {
			log.Fatalf("Error creating destination directory: %v", err)
		}

		if err := copyFile(srcPath, destFullPath); err != nil {
			log.Fatalf("Error copying file: %v", err)
		}

		fmt.Printf("File uploaded successfully to %s\n", destFullPath)
	} else {
		fmt.Println("Direct file upload from CLI is not implemented for non-local storage types.")
		fmt.Println("Use the API or web handlers instead.")
	}
}

// getFile retrieves a file from storage
func getFile(ctx context.Context, provider *filesystem.Provider, path string) {
	fmt.Printf("Getting file: %s\n", path)

	file, info, err := provider.Get(ctx, path)
	if err != nil {
		log.Fatalf("Error getting file: %v", err)
	}
	defer file.Close()

	fmt.Printf("File information:\n")
	fmt.Printf("  Name: %s\n", info.Name)
	fmt.Printf("  Size: %d bytes\n", info.Size)
	fmt.Printf("  Last Modified: %s\n", info.LastModified.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Content Type: %s\n", info.ContentType)

	// Read the content (for small files)
	if info.Size < 1024*1024 { // Only show content for files less than 1MB
		content, err := io.ReadAll(file)
		if err != nil {
			log.Fatalf("Error reading file content: %v", err)
		}

		if isTextFile(info.ContentType) {
			fmt.Printf("\nContent:\n%s\n", string(content))
		} else {
			fmt.Printf("\nBinary file, %d bytes\n", len(content))
		}
	} else {
		fmt.Printf("\nFile too large to display content (%d bytes)\n", info.Size)
	}
}

// checkExists checks if a file exists
func checkExists(ctx context.Context, provider *filesystem.Provider, path string) {
	exists, err := provider.Exists(ctx, path)
	if err != nil {
		log.Fatalf("Error checking file existence: %v", err)
	}

	if exists {
		fmt.Printf("File exists: %s\n", path)
	} else {
		fmt.Printf("File does not exist: %s\n", path)
	}
}

// listFiles lists files in a directory
func listFiles(ctx context.Context, provider *filesystem.Provider, dir string) {
	fmt.Printf("Listing files in: %s\n", dir)

	files, err := provider.List(ctx, dir)
	if err != nil {
		log.Fatalf("Error listing files: %v", err)
	}

	if len(files) == 0 {
		fmt.Println("No files found.")
		return
	}

	fmt.Printf("Found %d files:\n", len(files))
	for i, file := range files {
		fileType := "File"
		if file.IsDirectory {
			fileType = "Directory"
		}

		fmt.Printf("%3d. [%s] %s (%d bytes, modified: %s)\n",
			i+1,
			fileType,
			file.Name,
			file.Size,
			file.LastModified.Format("2006-01-02 15:04:05"))
	}
}

// deleteFile deletes a file
func deleteFile(ctx context.Context, provider *filesystem.Provider, path string) {
	fmt.Printf("Deleting file: %s\n", path)

	if err := provider.Delete(ctx, path); err != nil {
		log.Fatalf("Error deleting file: %v", err)
	}

	fmt.Println("File deleted successfully.")
}

// getFileInfo gets information about a file
func getFileInfo(ctx context.Context, provider *filesystem.Provider, path string) {
	fmt.Printf("Getting file info: %s\n", path)

	info, err := provider.GetInfo(ctx, path)
	if err != nil {
		log.Fatalf("Error getting file info: %v", err)
	}

	fileType := "File"
	if info.IsDirectory {
		fileType = "Directory"
	}

	fmt.Printf("File information:\n")
	fmt.Printf("  Name: %s\n", info.Name)
	fmt.Printf("  Type: %s\n", fileType)
	fmt.Printf("  Size: %d bytes\n", info.Size)
	fmt.Printf("  Last Modified: %s\n", info.LastModified.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Content Type: %s\n", info.ContentType)
	fmt.Printf("  URL: %s\n", info.URL)
}

// Helper functions

// isTextFile checks if a content type is text
func isTextFile(contentType string) bool {
	return strings.HasPrefix(contentType, "text/") ||
		contentType == "application/json" ||
		contentType == "application/xml" ||
		contentType == "application/javascript"
}

// copyFile copies a file from src to dest
func copyFile(src, dest string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
