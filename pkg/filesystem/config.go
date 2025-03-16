package filesystem

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration options for the filesystem
type Config struct {
	// Storage type: "local" or "s3"
	StorageType string

	// Local storage config
	LocalStoragePath string
	LocalBaseURL     string
	CreateLocalDirs  bool

	// S3 config
	S3Endpoint   string
	S3AccessKey  string
	S3SecretKey  string
	S3Bucket     string
	S3BasePrefix string
	S3BaseURL    string
	S3Region     string
	S3UseSSL     bool
	S3PathStyle  bool

	// Upload config
	UploadMaxSizeMB  int
	AllowedFileTypes []string
	UseUUID          bool
	TimeoutSecs      int
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		StorageType:      "local",
		LocalStoragePath: "./storage/uploads",
		CreateLocalDirs:  true,
		UploadMaxSizeMB:  10,
		UseUUID:          true,
		TimeoutSecs:      30,
		AllowedFileTypes: []string{".jpg", ".jpeg", ".png", ".gif", ".pdf", ".doc", ".docx", ".xls", ".xlsx"},
	}
}

// NewConfigFromEnv loads configuration from environment variables
func NewConfigFromEnv() Config {
	config := DefaultConfig()

	// Storage type
	if storageType := os.Getenv("STORAGE_TYPE"); storageType != "" {
		config.StorageType = storageType
	}

	// Local storage config
	if path := os.Getenv("UPLOAD_STORAGE_PATH"); path != "" {
		config.LocalStoragePath = path
	}

	if baseURL := os.Getenv("LOCAL_BASE_URL"); baseURL != "" {
		config.LocalBaseURL = baseURL
	}

	if createDirs := os.Getenv("CREATE_LOCAL_DIRS"); createDirs != "" {
		config.CreateLocalDirs = (createDirs == "true" || createDirs == "1" || createDirs == "yes")
	}

	// S3 config
	config.S3Endpoint = os.Getenv("S3_ENDPOINT")
	config.S3AccessKey = os.Getenv("S3_ACCESS_KEY")
	config.S3SecretKey = os.Getenv("S3_SECRET_KEY")
	config.S3Bucket = os.Getenv("S3_BUCKET")
	config.S3BasePrefix = os.Getenv("S3_PREFIX")
	config.S3BaseURL = os.Getenv("S3_BASE_URL")
	config.S3Region = os.Getenv("S3_REGION")
	config.S3UseSSL = (os.Getenv("S3_USE_SSL") == "true")
	config.S3PathStyle = (os.Getenv("S3_PATH_STYLE") == "true")

	// Upload config
	if maxSize := getEnvAsInt("UPLOAD_MAX_SIZE", 10); maxSize > 0 {
		config.UploadMaxSizeMB = maxSize
	}

	if timeout := getEnvAsInt("UPLOAD_TIMEOUT_SECS", 30); timeout > 0 {
		config.TimeoutSecs = timeout
	}

	if useUUID := os.Getenv("USE_UUID_FILENAMES"); useUUID != "" {
		config.UseUUID = (useUUID == "true" || useUUID == "1" || useUUID == "yes")
	}

	if allowedTypes := os.Getenv("ALLOWED_FILE_TYPES"); allowedTypes != "" {
		types := strings.Split(allowedTypes, ",")
		var cleanTypes []string
		for _, t := range types {
			t = strings.TrimSpace(t)
			if t != "" {
				// Ensure the file type starts with a dot
				if !strings.HasPrefix(t, ".") {
					t = "." + t
				}
				cleanTypes = append(cleanTypes, strings.ToLower(t))
			}
		}
		if len(cleanTypes) > 0 {
			config.AllowedFileTypes = cleanTypes
		}
	}

	return config
}

// Validate checks if the configuration is valid
func (c *Config) Validate() []string {
	var errors []string

	// Check storage type
	if c.StorageType != "local" && c.StorageType != "s3" {
		errors = append(errors, "Invalid storage type. Must be 'local' or 's3'")
	}

	// Check S3 configuration if using S3
	if c.StorageType == "s3" {
		if c.S3Bucket == "" {
			errors = append(errors, "S3 bucket name is required when using S3 storage")
		}

		// If using a custom endpoint, access key and secret key are required
		if c.S3Endpoint != "" {
			if c.S3AccessKey == "" {
				errors = append(errors, "S3 access key is required when using a custom S3 endpoint")
			}
			if c.S3SecretKey == "" {
				errors = append(errors, "S3 secret key is required when using a custom S3 endpoint")
			}
		}
	}

	// Check upload size
	if c.UploadMaxSizeMB <= 0 {
		errors = append(errors, "Upload max size must be greater than 0")
	}

	// Check timeout
	if c.TimeoutSecs <= 0 {
		errors = append(errors, "Timeout seconds must be greater than 0")
	}

	return errors
}

// Helper function to get environment variable as integer
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
