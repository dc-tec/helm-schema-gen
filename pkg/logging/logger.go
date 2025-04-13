// Package logging provides structured logging functionality using slog.
// It includes utilities for operation tracking, component-based logging,
// and standardized log formatting.
package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// Global logger instance used across the application
var globalLogger *slog.Logger

// Common logging keys used for structured logging
const (
	ComponentKey = "component" // Key used for component/module identification
	OperationKey = "operation" // Key used for operation names
	ErrorKey     = "error"     // Key used for error messages
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// operationContextKey is used to store operation names in context
const operationContextKey = contextKey("operation")

// initLogger creates and configures a new slog.Logger instance with JSON output
// and custom timestamp formatting. It reads the log level from environment variables.
func initLogger() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: getLevelFromEnv(),
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Key = "timestamp"
				if t, ok := a.Value.Any().(time.Time); ok {
					a.Value = slog.StringValue(t.Format(time.RFC3339))
				}
			}
			return a
		},
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	globalLogger = slog.New(handler)
	return globalLogger
}

// GetLogger returns the global logger instance, initializing it if necessary.
func GetLogger() *slog.Logger {
	if globalLogger == nil {
		globalLogger = initLogger()
	}
	return globalLogger
}

// WithComponent creates a new logger with the specified component name added
// to all log entries.
func WithComponent(ctx context.Context, component string) *slog.Logger {
	return GetLogger().With(ComponentKey, component)
}

// WithOperation adds an operation name to the context for tracking purposes.
func WithOperation(ctx context.Context, operation string) context.Context {
	return context.WithValue(ctx, operationContextKey, operation)
}

// GetOperation retrieves the current operation from the context, if any.
func GetOperation(ctx context.Context) (string, bool) {
	op, ok := ctx.Value(operationContextKey).(string)
	return op, ok
}

// LogError logs an error message with the specified error.
func LogError(ctx context.Context, err error, msg string, args ...any) error {
	logger := GetLogger()
	logger.ErrorContext(ctx, msg, append(args, ErrorKey, err)...)
	return fmt.Errorf("%s: %w", msg, err)
}

// LogOperation wraps an operation with standardized logging
func LogOperation[T any](ctx context.Context, operation string, fn func() (T, error)) (T, error) {
	logger := GetLogger()

	// Create a new context with operation info
	opCtx := WithOperation(ctx, operation)

	logger.InfoContext(opCtx, "starting operation", OperationKey, operation)

	result, err := fn()
	if err != nil {
		logger.ErrorContext(opCtx, "operation failed",
			OperationKey, operation,
			ErrorKey, err,
		)
		return result, fmt.Errorf("%s: %w", operation, err)
	}

	logger.InfoContext(opCtx, "operation completed",
		OperationKey, operation,
	)
	return result, nil
}

// getLevelFromEnv reads and parses the LOG_LEVEL environment variable
// to determine the logging level. Defaults to INFO if not set or invalid.
func getLevelFromEnv() slog.Level {
	level := os.Getenv("LOG_LEVEL")
	switch level {
	case "DEBUG", "debug":
		return slog.LevelDebug
	case "INFO", "info":
		return slog.LevelInfo
	case "WARN", "warn":
		return slog.LevelWarn
	case "ERROR", "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
