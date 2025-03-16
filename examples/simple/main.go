package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/anaknegeri/gokit"
	"github.com/anaknegeri/gokit/pkg/filesystem"
)

func main() {
	// Initialize a logger
	logger := gokit.InitLogger()
	logger.Info("Starting simple example application")

	// Create a configuration for local storage
	config := filesystem.Config{
		StorageType:      "local",
		LocalStoragePath: "./uploads",
		CreateLocalDirs:  true,
		UploadMaxSizeMB:  10,
		AllowedFileTypes: []string{".txt", ".jpg", ".png", ".pdf"},
	}

	// Initialize context
	ctx := context.Background()

	// Create filesystem provider with custom config
	fs, err := gokit.NewFilesystemWithConfig(ctx, config)
	if err != nil {
		logger.Fatalf("Failed to create filesystem provider: %v", err)
	}

	// Create a sample file for testing
	testFilePath := "test-file.txt"
	err = os.WriteFile(testFilePath, []byte("This is a test file for GoKit filesystem.\nIt demonstrates basic file operations."), 0644)
	if err != nil {
		logger.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFilePath) // Clean up

	// Use the filesystem for various operations
	demonstrateFilesystem(ctx, fs.Provider, testFilePath, logger)
}

func demonstrateFilesystem(ctx context.Context, fs *filesystem.Provider, testFilePath string, logger *gokit.Logger) {
	// Create directories
	destDir := "example/docs"
	destPath := destDir + "/test-file.txt"
	logger.Infof("Target destination: %s", destPath)

	// For direct access in our example, let's manually copy the file to the uploads directory
	// In a real application, you would typically get the file from a HTTP request
	err := os.MkdirAll("uploads/"+destDir, 0755)
	if err != nil {
		logger.Fatalf("Failed to create directories: %v", err)
	}

	// Copy the file
	source, err := os.Open(testFilePath)
	if err != nil {
		logger.Fatalf("Failed to open source file: %v", err)
	}
	defer source.Close()

	dest, err := os.Create("uploads/" + destPath)
	if err != nil {
		logger.Fatalf("Failed to create destination file: %v", err)
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	if err != nil {
		logger.Fatalf("Failed to copy file: %v", err)
	}
	logger.Info("File copied to uploads directory")

	// Example 1: Check if file exists
	exists, err := fs.Exists(ctx, destPath)
	if err != nil {
		logger.Errorf("Error checking file existence: %v", err)
	} else if exists {
		logger.Infof("File exists at %s: %v", destPath, exists)
	} else {
		logger.Warnf("File does not exist at %s", destPath)
	}

	// Example 2: Get file info
	fileInfo, err := fs.GetInfo(ctx, destPath)
	if err != nil {
		logger.Errorf("Error getting file info: %v", err)
	} else {
		logger.Infof("File info: Name=%s, Size=%d bytes, ContentType=%s",
			fileInfo.Name, fileInfo.Size, fileInfo.ContentType)
	}

	// Example 3: List files in directory
	files, err := fs.List(ctx, "example")
	if err != nil {
		logger.Errorf("Error listing files: %v", err)
	} else {
		logger.Infof("Files in 'example' directory:")
		for _, file := range files {
			isDir := ""
			if file.IsDirectory {
				isDir = " (directory)"
			}
			logger.Infof("- %s%s: %d bytes, Last modified: %s",
				file.Name, isDir, file.Size, file.LastModified.Format(time.RFC3339))
		}
	}

	// Example 4: Read file content
	fileReader, fileInfo, err := fs.Get(ctx, destPath)
	if err != nil {
		logger.Errorf("Error reading file: %v", err)
	} else {
		defer fileReader.Close()
		content, err := io.ReadAll(fileReader)
		if err != nil {
			logger.Errorf("Error reading file content: %v", err)
		} else {
			logger.Infof("File content: %s", string(content))
		}
	}

	// Example 5: Delete file
	fmt.Println("\nPress Enter to delete the test file...")
	fmt.Scanln() // Wait for user input

	err = fs.Delete(ctx, destPath)
	if err != nil {
		logger.Errorf("Error deleting file: %v", err)
	} else {
		logger.Infof("File deleted successfully")
	}

	// Verify deletion
	exists, _ = fs.Exists(ctx, destPath)
	logger.Infof("File still exists: %v", exists)
}
