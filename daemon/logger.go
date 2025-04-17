package daemon

import (
	"fmt"
)

// Logger defines the logging interface for daemon operations
type Logger interface {
	// Debug logs a debug message
	Debug(format string, args ...interface{})

	// Info logs an info message
	Info(format string, args ...interface{})

	// Success logs a success message
	Success(format string, args ...interface{})

	// Warning logs a warning message
	Warning(format string, args ...interface{})

	// Error logs an error message
	Error(format string, args ...interface{})

	// Fatal logs a fatal message
	Fatal(format string, args ...interface{})
}

// DefaultLogger provides a basic implementation of the Logger interface
type DefaultLogger struct{}

// NewDefaultLogger creates a new default logger
func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{}
}

// Debug logs a debug message
func (l *DefaultLogger) Debug(format string, args ...interface{}) {
	fmt.Printf("[DEBUG] "+format+"\n", args...)
}

// Info logs an info message
func (l *DefaultLogger) Info(format string, args ...interface{}) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}

// Success logs a success message
func (l *DefaultLogger) Success(format string, args ...interface{}) {
	fmt.Printf("[SUCCESS] "+format+"\n", args...)
}

// Warning logs a warning message
func (l *DefaultLogger) Warning(format string, args ...interface{}) {
	fmt.Printf("[WARNING] "+format+"\n", args...)
}

// Error logs an error message
func (l *DefaultLogger) Error(format string, args ...interface{}) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}

// Fatal logs a fatal message
func (l *DefaultLogger) Fatal(format string, args ...interface{}) {
	fmt.Printf("[FATAL] "+format+"\n", args...)
}
