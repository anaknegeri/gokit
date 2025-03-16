package filesystem

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLocalStorage(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "filesystem-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test config
	config := LocalStorageConfig{
		BasePath:          tempDir,
		CreateDirectories: true,
	}

	// Create local storage
	storage, err := NewLocalStorage(config)
	if err != nil {
		t.Fatalf("Failed to create local storage: %v", err)
	}

	// Create test context
	ctx := context.Background()

	// Create a test file
	testFilePath := filepath.Join(tempDir, "test-file.txt")
	testContent := []byte("Hello, world!")
	if err := os.WriteFile(testFilePath, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test Exists method
	t.Run("Exists", func(t *testing.T) {
		// Should exist
		exists, err := storage.Exists(ctx, "test-file.txt")
		if err != nil {
			t.Fatalf("Error checking if file exists: %v", err)
		}
		if !exists {
			t.Errorf("File should exist but doesn't")
		}

		// Should not exist
		exists, err = storage.Exists(ctx, "non-existent-file.txt")
		if err != nil {
			t.Fatalf("Error checking if file exists: %v", err)
		}
		if exists {
			t.Errorf("File should not exist but does")
		}
	})

	// Test Get method
	t.Run("Get", func(t *testing.T) {
		file, info, err := storage.Get(ctx, "test-file.txt")
		if err != nil {
			t.Fatalf("Error getting file: %v", err)
		}
		defer file.Close()

		if info.Name != "test-file.txt" {
			t.Errorf("Expected name %q, got %q", "test-file.txt", info.Name)
		}

		if info.Size != int64(len(testContent)) {
			t.Errorf("Expected size %d, got %d", len(testContent), info.Size)
		}

		content, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("Error reading file: %v", err)
		}

		if !bytes.Equal(content, testContent) {
			t.Errorf("Expected content %q, got %q", string(testContent), string(content))
		}
	})

	// Test Upload method
	t.Run("Upload", func(t *testing.T) {
		// Create a multipart file header
		buffer := &bytes.Buffer{}
		writer := multipart.NewWriter(buffer)

		fileWriter, err := writer.CreateFormFile("file", "upload-test.txt")
		if err != nil {
			t.Fatalf("Error creating form file: %v", err)
		}

		uploadContent := []byte("This is an uploaded file")
		if _, err := fileWriter.Write(uploadContent); err != nil {
			t.Fatalf("Error writing to form file: %v", err)
		}

		if err := writer.Close(); err != nil {
			t.Fatalf("Error closing writer: %v", err)
		}

		// Create file header
		header := &multipart.FileHeader{
			Filename: "upload-test.txt",
			Size:     int64(len(uploadContent)),
			Header:   make(map[string][]string),
		}

		// Test upload
		fileInfo, err := storage.Upload(ctx, header, "subfolder/upload-test.txt")
		if err != nil {
			t.Fatalf("Error uploading file: %v", err)
		}

		// Verify the file info
		if fileInfo.Name != "upload-test.txt" {
			t.Errorf("Expected file name %q, got %q", "upload-test.txt", fileInfo.Name)
		}

		if fileInfo.Size != int64(len(uploadContent)) {
			t.Errorf("Expected file size %d, got %d", len(uploadContent), fileInfo.Size)
		}

		// Verify the file exists
		exists, err := storage.Exists(ctx, "subfolder/upload-test.txt")
		if err != nil {
			t.Fatalf("Error checking if file exists: %v", err)
		}
		if !exists {
			t.Errorf("Uploaded file should exist but doesn't")
		}
	})

	// Test Delete method
	t.Run("Delete", func(t *testing.T) {
		if err := storage.Delete(ctx, "test-file.txt"); err != nil {
			t.Fatalf("Error deleting file: %v", err)
		}

		exists, err := storage.Exists(ctx, "test-file.txt")
		if err != nil {
			t.Fatalf("Error checking if file exists: %v", err)
		}
		if exists {
			t.Errorf("File should not exist after deletion but does")
		}
	})

	// Test List method
	t.Run("List", func(t *testing.T) {
		// Create some test files in a subdirectory
		subDir := filepath.Join(tempDir, "listtest")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("Error creating subdirectory: %v", err)
		}

		for _, name := range []string{"file1.txt", "file2.txt", "file3.txt"} {
			content := []byte(time.Now().String())
			if err := os.WriteFile(filepath.Join(subDir, name), content, 0644); err != nil {
				t.Fatalf("Error creating test file %s: %v", name, err)
			}
			time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		}

		// Test listing
		files, err := storage.List(ctx, "listtest")
		if err != nil {
			t.Fatalf("Error listing files: %v", err)
		}

		// Check that we got all the files
		if len(files) != 3 {
			t.Errorf("Expected 3 files, got %d", len(files))
		}

		// Check file names
		fileNames := make(map[string]bool)
		for _, file := range files {
			fileNames[file.Name] = true
		}

		for _, name := range []string{"file1.txt", "file2.txt", "file3.txt"} {
			if !fileNames[name] {
				t.Errorf("Expected file %s not found in list", name)
			}
		}
	})

	// Test GetInfo method
	t.Run("GetInfo", func(t *testing.T) {
		// Create a test file with known content
		testFile := "getinfo-test.txt"
		testContent := []byte("GetInfo test content")
		if err := os.WriteFile(filepath.Join(tempDir, testFile), testContent, 0644); err != nil {
			t.Fatalf("Error creating test file: %v", err)
		}

		// Get file info
		info, err := storage.GetInfo(ctx, testFile)
		if err != nil {
			t.Fatalf("Error getting file info: %v", err)
		}

		// Verify info
		if info.Name != testFile {
			t.Errorf("Expected name %q, got %q", testFile, info.Name)
		}

		if info.Size != int64(len(testContent)) {
			t.Errorf("Expected size %d, got %d", len(testContent), info.Size)
		}

		if info.ContentType != "text/plain" {
			t.Errorf("Expected content type %q, got %q", "text/plain", info.ContentType)
		}

		if info.IsDirectory {
			t.Errorf("Expected IsDirectory to be false, got true")
		}
	})
}
