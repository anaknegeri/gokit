// Package logger provides customizable logging functionality
package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// LogLevel defines logging levels
type LogLevel int

// Log levels
const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String returns string representation of log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger is a custom logger implementation
type Logger struct {
	logLevel LogLevel
	output   io.Writer
	prefix   string
}

// NewLogger creates a new logger instance
func NewLogger() *Logger {
	return &Logger{
		logLevel: DEBUG,
		output:   os.Stdout,
		prefix:   "",
	}
}

// InitLogger initializes a logger from environment variables
func InitLogger() *Logger {
	logger := NewLogger()

	// Set log level from environment variable
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		switch strings.ToLower(logLevel) {
		case "debug":
			logger.SetLevel(uint8(DEBUG))
		case "info":
			logger.SetLevel(uint8(INFO))
		case "warn", "warning":
			logger.SetLevel(uint8(WARN))
		case "error":
			logger.SetLevel(uint8(ERROR))
		case "fatal":
			logger.SetLevel(uint8(FATAL))
		}
	}

	// Set output from environment variable
	if logOutput := os.Getenv("LOG_OUTPUT"); logOutput != "" {
		switch strings.ToLower(logOutput) {
		case "stdout":
			logger.SetOutput(os.Stdout)
		case "stderr":
			logger.SetOutput(os.Stderr)
		case "file":
			// Open log file if LOG_FILE_PATH is set
			if logFilePath := os.Getenv("LOG_FILE_PATH"); logFilePath != "" {
				// Ensure directory exists
				dir := filepath.Dir(logFilePath)
				if err := os.MkdirAll(dir, 0755); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to create log directory: %v\n", err)
				} else {
					// Open file for logging
					file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
					} else {
						logger.SetOutput(file)
					}
				}
			}
		}
	}

	// Set prefix from environment variable
	if logPrefix := os.Getenv("LOG_PREFIX"); logPrefix != "" {
		logger.SetPrefix(logPrefix)
	}

	logger.Info("Logger initialized")
	return logger
}

// SetLevel sets the logger level
func (l *Logger) SetLevel(v uint8) {
	if v <= uint8(FATAL) {
		l.logLevel = LogLevel(v)
	}
}

// SetOutput sets the logger output
func (l *Logger) SetOutput(w io.Writer) {
	l.output = w
}

// SetPrefix sets the logger prefix
func (l *Logger) SetPrefix(p string) {
	l.prefix = p
}

// SetHeader is a no-op for compatibility
func (l *Logger) SetHeader(h string) {
	// No-op for compatibility
}

// Output returns the logger output
func (l *Logger) Output() io.Writer {
	return l.output
}

// Prefix returns the logger prefix
func (l *Logger) Prefix() string {
	return l.prefix
}

// Level returns the logger level
func (l *Logger) Level() uint8 {
	return uint8(l.logLevel)
}

// Logf logs a message with specified level and format
func (l *Logger) Logf(level LogLevel, format string, args ...interface{}) {
	l.log(level, format, args...)
}

// log logs a message at the specified level
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.logLevel {
		return
	}

	// Get caller information
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "???"
		line = 0
	}
	file = filepath.Base(file)

	// Format message
	message := format
	if format == "" {
		message = fmt.Sprint(args...)
	} else {
		message = fmt.Sprintf(format, args...)
	}

	// Log to output
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	fmt.Fprintf(l.output, "%s | %s | %s:%d | %s%s\n",
		timestamp, level.String(), file, line, l.prefix, message)

	// If FATAL, exit
	if level == FATAL {
		os.Exit(1)
	}
}

// logJSON logs a JSON object at the specified level
func (l *Logger) logJSON(level LogLevel, j map[string]interface{}) {
	if level < l.logLevel {
		return
	}

	// Get caller information
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "???"
		line = 0
	}
	file = filepath.Base(file)

	// Add metadata to JSON
	j["timestamp"] = time.Now().Format("2006-01-02 15:04:05.000")
	j["level"] = level.String()
	j["file"] = file
	j["line"] = line
	if l.prefix != "" {
		j["prefix"] = l.prefix
	}

	// Convert to JSON
	bytes, err := json.Marshal(j)
	if err != nil {
		fmt.Fprintf(l.output, "ERROR MARSHALING JSON: %v\n", err)
		return
	}

	// Log to output
	fmt.Fprintln(l.output, string(bytes))

	// If FATAL, exit
	if level == FATAL {
		os.Exit(1)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(i ...interface{}) {
	l.log(DEBUG, "", i...)
}

// Debugf logs a debug message with format
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Debugj logs a debug message as JSON
func (l *Logger) Debugj(j map[string]interface{}) {
	l.logJSON(DEBUG, j)
}

// Info logs an info message
func (l *Logger) Info(i ...interface{}) {
	l.log(INFO, "", i...)
}

// Infof logs an info message with format
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Infoj logs an info message as JSON
func (l *Logger) Infoj(j map[string]interface{}) {
	l.logJSON(INFO, j)
}

// Warn logs a warning message
func (l *Logger) Warn(i ...interface{}) {
	l.log(WARN, "", i...)
}

// Warnf logs a warning message with format
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Warnj logs a warning message as JSON
func (l *Logger) Warnj(j map[string]interface{}) {
	l.logJSON(WARN, j)
}

// Error logs an error message
func (l *Logger) Error(i ...interface{}) {
	l.log(ERROR, "", i...)
}

// Errorf logs an error message with format
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// Errorj logs an error message as JSON
func (l *Logger) Errorj(j map[string]interface{}) {
	l.logJSON(ERROR, j)
}

// Fatal logs a fatal message
func (l *Logger) Fatal(i ...interface{}) {
	l.log(FATAL, "", i...)
}

// Fatalf logs a fatal message with format
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log(FATAL, format, args...)
}

// Fatalj logs a fatal message as JSON
func (l *Logger) Fatalj(j map[string]interface{}) {
	l.logJSON(FATAL, j)
}

// Print logs a message
func (l *Logger) Print(i ...interface{}) {
	l.log(INFO, "", i...)
}

// Printf logs a message with format
func (l *Logger) Printf(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Printj logs a message as JSON
func (l *Logger) Printj(j map[string]interface{}) {
	l.logJSON(INFO, j)
}

// Panic logs a panic message
func (l *Logger) Panic(i ...interface{}) {
	l.log(FATAL, "", i...)
	panic(fmt.Sprint(i...))
}

// Panicf logs a panic message with format
func (l *Logger) Panicf(format string, args ...interface{}) {
	l.log(FATAL, format, args...)
	panic(fmt.Sprintf(format, args...))
}

// Panicj logs a panic message as JSON
func (l *Logger) Panicj(j map[string]interface{}) {
	l.logJSON(FATAL, j)
	panic(j)
}
