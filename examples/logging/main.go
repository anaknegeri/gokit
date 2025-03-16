package main

import (
	"os"
	"time"

	"github.com/anaknegeri/gokit"
)

func main() {
	// Example 1: Basic logger initialization
	logger := gokit.NewLogger()
	logger.Info("Basic logger initialized")

	// Example 2: Configure logger manually
	logger.SetLevel(uint8(gokit.LogLevelDebug))
	logger.SetPrefix("[GOKIT] ")
	logger.Info("Logger with custom prefix and level")

	// Example 3: Different log levels
	logger.Debug("This is a debug message")
	logger.Info("This is an info message")
	logger.Warn("This is a warning message")
	logger.Error("This is an error message")
	// logger.Fatal("This is a fatal message - would exit the program")

	// Example 4: Formatted logging
	logger.Debugf("Debug with formatting: %s", time.Now().Format(time.RFC3339))
	logger.Infof("User %s logged in from %s", "john_doe", "192.168.1.1")
	logger.Warnf("High CPU usage: %.2f%%", 85.75)
	logger.Errorf("Failed to connect to %s: %v", "database", "connection refused")

	// Example 5: JSON logging
	logger.Infoj(map[string]interface{}{
		"action":   "user_login",
		"user_id":  12345,
		"username": "john_doe",
		"ip":       "192.168.1.1",
		"success":  true,
		"metadata": map[string]interface{}{
			"browser": "Chrome",
			"os":      "Windows",
		},
	})

	// Example 6: Log to file
	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logger.Errorf("Failed to open log file: %v", err)
	} else {
		fileLogger := gokit.NewLogger()
		fileLogger.SetOutput(logFile)
		fileLogger.SetPrefix("[FILE] ")

		fileLogger.Info("This message goes to the file")
		fileLogger.Warnf("Warning logged at %s", time.Now().Format(time.RFC3339))
		fileLogger.Infoj(map[string]interface{}{
			"event":   "file_logging",
			"success": true,
		})

		logger.Info("Check app.log file for file logger output")
	}

	// Example 7: Environment-based initialization
	os.Setenv("LOG_LEVEL", "info")
	os.Setenv("LOG_OUTPUT", "stdout")
	os.Setenv("LOG_PREFIX", "[ENV] ")

	envLogger := gokit.InitLogger()
	envLogger.Debug("This debug message won't appear due to LOG_LEVEL=info")
	envLogger.Info("This info message should appear with [ENV] prefix")
	envLogger.Warn("This warning message should appear with [ENV] prefix")

	// Example 8: Multiple loggers for different components
	authLogger := gokit.NewLogger()
	authLogger.SetPrefix("[AUTH] ")

	dbLogger := gokit.NewLogger()
	dbLogger.SetPrefix("[DB] ")

	cacheLogger := gokit.NewLogger()
	cacheLogger.SetPrefix("[CACHE] ")

	authLogger.Info("User authentication successful")
	dbLogger.Info("Database connection established")
	cacheLogger.Info("Cache initialized")

	// Example 9: Error handling with logger
	if err := simulateError(); err != nil {
		appErr := gokit.NewError(500, "An error occurred in the application")
		logger.Errorf("Application error: %v", appErr)

		wrappedErr := gokit.WrapError(err, 500, "Wrapped error with context")
		logger.Errorf("Wrapped error: %v", wrappedErr)
	}
}

// simulateError simulates an error for demonstration
func simulateError() error {
	return gokit.New("simulated error for demonstration")
}
