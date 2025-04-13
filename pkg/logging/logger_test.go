package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"testing"
)

func TestGetLogger(t *testing.T) {
	// Reset global logger for test
	globalLogger = nil

	logger := GetLogger()
	if logger == nil {
		t.Fatal("GetLogger() returned nil")
	}

	// Calling again should return the same instance
	logger2 := GetLogger()
	if logger != logger2 {
		t.Fatal("GetLogger() returned different instances")
	}
}

func TestWithComponent(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	opts := &slog.HandlerOptions{}
	handler := slog.NewJSONHandler(&buf, opts)
	globalLogger = slog.New(handler)

	ctx := context.Background()
	compLogger := WithComponent(ctx, "test-component")

	compLogger.Info("test message")

	var logEntry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log entry: %v", err)
	}

	component, ok := logEntry[ComponentKey].(string)
	if !ok || component != "test-component" {
		t.Errorf("Expected component 'test-component', got %v", logEntry[ComponentKey])
	}
}

func TestWithOperation(t *testing.T) {
	ctx := context.Background()
	ctx = WithOperation(ctx, "test-operation")

	op, ok := GetOperation(ctx)
	if !ok {
		t.Fatal("GetOperation returned not ok")
	}

	if op != "test-operation" {
		t.Errorf("Expected operation 'test-operation', got %s", op)
	}
}

func TestLogError(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	opts := &slog.HandlerOptions{}
	handler := slog.NewJSONHandler(&buf, opts)
	globalLogger = slog.New(handler)

	ctx := context.Background()
	testErr := errors.New("test error")

	err := LogError(ctx, testErr, "error occurred", "key", "value")

	// Verify the error is wrapped
	if err == nil || err.Error() != "error occurred: test error" {
		t.Errorf("LogError didn't format error correctly: %v", err)
	}

	var logEntry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log entry: %v", err)
	}

	if logEntry["key"] != "value" {
		t.Errorf("Custom field not included in log")
	}

	if logEntry[ErrorKey] != "test error" {
		t.Errorf("Error not included correctly in log")
	}
}

func TestLogOperation(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	opts := &slog.HandlerOptions{}
	handler := slog.NewJSONHandler(&buf, opts)
	globalLogger = slog.New(handler)

	ctx := context.Background()

	// Test success case
	result, err := LogOperation(ctx, "test-op", func() (string, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("LogOperation returned unexpected error: %v", err)
	}

	if result != "success" {
		t.Errorf("LogOperation returned unexpected result: %v", result)
	}

	// Test error case
	_, err = LogOperation(ctx, "test-op-fail", func() (string, error) {
		return "", errors.New("operation failed")
	})

	if err == nil || err.Error() != "test-op-fail: operation failed" {
		t.Errorf("LogOperation didn't handle error correctly: %v", err)
	}
}

func TestGetLevelFromEnv(t *testing.T) {
	tests := []struct {
		envValue string
		expected slog.Level
	}{
		{"DEBUG", slog.LevelDebug},
		{"debug", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{"info", slog.LevelInfo},
		{"WARN", slog.LevelWarn},
		{"warn", slog.LevelWarn},
		{"ERROR", slog.LevelError},
		{"error", slog.LevelError},
		{"invalid", slog.LevelInfo}, // Default
		{"", slog.LevelInfo},        // Default
	}

	for _, tc := range tests {
		t.Run(tc.envValue, func(t *testing.T) {
			// Set environment variable
			if tc.envValue != "" {
				os.Setenv("LOG_LEVEL", tc.envValue)
				defer os.Unsetenv("LOG_LEVEL")
			} else {
				os.Unsetenv("LOG_LEVEL")
			}

			level := getLevelFromEnv()
			if level != tc.expected {
				t.Errorf("getLevelFromEnv with LOG_LEVEL=%q returned %v, expected %v",
					tc.envValue, level, tc.expected)
			}
		})
	}
}
