package logs

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Logger provides structured logging
type Logger struct {
	level  Level
	writer io.Writer
	mu     sync.Mutex
}

// global is the package-level logger instance
var global *Logger

// Init initializes the global logger (called at suite startup)
func Init(level Level) {
	global = &Logger{
		level:  level,
		writer: os.Stdout,
	}
}

// InitWithWriter initializes with a custom writer
func InitWithWriter(level Level, writer io.Writer) {
	global = &Logger{
		level:  level,
		writer: writer,
	}
}

// SetLevel changes the minimum log level
func SetLevel(level Level) {
	if global != nil {
		global.mu.Lock()
		global.level = level
		global.mu.Unlock()
	}
}

// GetLevel returns the current log level
func GetLevel() Level {
	if global == nil {
		return LevelInfo
	}
	global.mu.Lock()
	defer global.mu.Unlock()
	return global.level
}

// Trace logs at trace level
func Trace(ctx context.Context, msg string, fields map[string]string) {
	log(ctx, LevelTrace, msg, fields)
}

// Debug logs at debug level
func Debug(ctx context.Context, msg string, fields map[string]string) {
	log(ctx, LevelDebug, msg, fields)
}

// Info logs at info level
func Info(ctx context.Context, msg string, fields map[string]string) {
	log(ctx, LevelInfo, msg, fields)
}

// Warn logs at warn level
func Warn(ctx context.Context, msg string, fields map[string]string) {
	log(ctx, LevelWarn, msg, fields)
}

// Error logs at error level
func Error(ctx context.Context, msg string, fields map[string]string) {
	log(ctx, LevelError, msg, fields)
}

// Fatal logs at fatal level
func Fatal(ctx context.Context, msg string, fields map[string]string) {
	log(ctx, LevelFatal, msg, fields)
}

func log(ctx context.Context, level Level, msg string, fields map[string]string) {
	if global == nil {
		return
	}

	global.mu.Lock()
	defer global.mu.Unlock()

	if level < global.level {
		return
	}

	// Format timestamp
	ts := time.Now().Format(time.RFC3339)

	// Format fields
	fieldStr := ""
	if len(fields) > 0 {
		fieldStr = fmt.Sprintf(" %v", fields)
	}

	// Write log entry
	fmt.Fprintf(global.writer, "[%s] %s %s%s\n", ts, level, msg, fieldStr)
}

// Tracef logs at trace level with formatting
func Tracef(ctx context.Context, format string, args ...interface{}) {
	Trace(ctx, fmt.Sprintf(format, args...), nil)
}

// Debugf logs at debug level with formatting
func Debugf(ctx context.Context, format string, args ...interface{}) {
	Debug(ctx, fmt.Sprintf(format, args...), nil)
}

// Infof logs at info level with formatting
func Infof(ctx context.Context, format string, args ...interface{}) {
	Info(ctx, fmt.Sprintf(format, args...), nil)
}

// Warnf logs at warn level with formatting
func Warnf(ctx context.Context, format string, args ...interface{}) {
	Warn(ctx, fmt.Sprintf(format, args...), nil)
}

// Errorf logs at error level with formatting
func Errorf(ctx context.Context, format string, args ...interface{}) {
	Error(ctx, fmt.Sprintf(format, args...), nil)
}

// Fatalf logs at fatal level with formatting
func Fatalf(ctx context.Context, format string, args ...interface{}) {
	Fatal(ctx, fmt.Sprintf(format, args...), nil)
}
