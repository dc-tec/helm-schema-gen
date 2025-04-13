package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	// Verify default values
	if opts.InputFile != "values.yaml" {
		t.Errorf("Expected InputFile to be 'values.yaml', got '%s'", opts.InputFile)
	}

	if opts.OutputFile != "values.schema.json" {
		t.Errorf("Expected OutputFile to be 'values.schema.json', got '%s'", opts.OutputFile)
	}

	if opts.SchemaVersion != "http://json-schema.org/draft-07/schema#" {
		t.Errorf("Expected SchemaVersion to be 'http://json-schema.org/draft-07/schema#', got '%s'", opts.SchemaVersion)
	}

	if opts.Title != "Helm Values Schema" {
		t.Errorf("Expected Title to be 'Helm Values Schema', got '%s'", opts.Title)
	}

	if opts.RequireByDefault {
		t.Error("Expected RequireByDefault to be false")
	}

	if !opts.IncludeExamples {
		t.Error("Expected IncludeExamples to be true")
	}

	if !opts.ExtractDescriptions {
		t.Error("Expected ExtractDescriptions to be true")
	}

	if opts.ValidateBestPractices {
		t.Error("Expected ValidateBestPractices to be false")
	}

	if opts.Verbose {
		t.Error("Expected Verbose to be false")
	}

	if opts.Debug {
		t.Error("Expected Debug to be false")
	}
}

func TestNewRootCommand(t *testing.T) {
	cmd := NewRootCommand()

	// Verify basic command properties
	if cmd.Use != "helm-schema-gen" {
		t.Errorf("Expected command Use to be 'helm-schema-gen', got '%s'", cmd.Use)
	}

	// Verify flags
	checkFlag := func(name, expectedUsage string) {
		flag := cmd.Flags().Lookup(name)
		if flag == nil {
			t.Errorf("Expected flag '%s' not found", name)
			return
		}

		if flag.Usage != expectedUsage {
			t.Errorf("Flag '%s' has unexpected usage: '%s', expected '%s'",
				name, flag.Usage, expectedUsage)
		}
	}

	checkFlag("file", "Input values.yaml file")
	checkFlag("output", "Output schema file")
	checkFlag("schema-version", "JSON Schema version to use")
	checkFlag("title", "Schema title")
	checkFlag("description", "Schema description")
	checkFlag("require-all", "Make all properties required")
	checkFlag("include-examples", "Include examples from values")
	checkFlag("extract-descriptions", "Extract descriptions from comments")
	checkFlag("validate", "Validate schema against Helm best practices")
	checkFlag("verbose", "Enable verbose output")
	checkFlag("debug", "Enable debug output")
}

// In a real-world scenario, we would typically use a testing framework
// that allows mocking or dependency injection for cobra commands.
// For simplicity, we'll perform a basic smoke test for ExecuteCLI.
func TestExecuteCLI_Smoke(t *testing.T) {
	// This test just ensures that ExecuteCLI can run without panic
	// We're not testing the command execution itself, as that requires
	// mocking cmd.Execute() which is difficult to do cleanly.

	// Skip for most test runs as it would try to execute the actual command
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping ExecuteCLI test; set RUN_INTEGRATION_TESTS=true to run")
	}

	// Capture stdout/stderr to prevent output pollution
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	// Run with timeout to ensure it doesn't hang
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan struct{})

	go func() {
		_ = ExecuteCLI() // Ignore error since we're just checking it doesn't panic
		close(done)
	}()

	select {
	case <-ctx.Done():
		// This is expected as ExecuteCLI will hang waiting for input
		// or exit early due to errors, which is fine for a smoke test
	case <-done:
		// Test completed normally
	}

	// Restore stdout/stderr
	w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	// Drain the pipe to prevent deadlock
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
}

// TestHelpers for runGenerateCommand testing
func setupTestFiles(t *testing.T) (string, string, func()) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "helm-schema-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Use the comprehensive values.yaml from testdata
	testdataValuesPath := "../../testdata/values.yaml"

	// Check if the testdata values file exists
	if _, err := os.Stat(testdataValuesPath); os.IsNotExist(err) {
		t.Fatalf("Testdata values file not found at %s", testdataValuesPath)
	}

	// Read the test values file content
	valuesData, err := os.ReadFile(testdataValuesPath)
	if err != nil {
		t.Fatalf("Failed to read testdata values file: %v", err)
	}

	// Create a copy of the values file in the temp directory
	valuesPath := filepath.Join(tempDir, "values.yaml")
	if err := os.WriteFile(valuesPath, valuesData, 0644); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to write test values file: %v", err)
	}

	outputPath := filepath.Join(tempDir, "values.schema.json")

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return valuesPath, outputPath, cleanup
}

func TestRunGenerateCommand(t *testing.T) {
	// Skip actual schema generation and validation in CI
	// This is simplified for the test setup
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping in CI environment")
	}

	// Setup test files
	inputPath, outputPath, cleanup := setupTestFiles(t)
	defer cleanup()

	// Setup test options
	opts := &Options{
		InputFile:             inputPath,
		OutputFile:            outputPath,
		SchemaVersion:         "draft-07",
		Title:                 "Test Schema",
		RequireByDefault:      false,
		IncludeExamples:       true,
		ExtractDescriptions:   true,
		ValidateBestPractices: false,
		Verbose:               true,
		Debug:                 false,
	}

	// Run the command
	ctx := context.Background()
	err := runGenerateCommand(ctx, opts)

	// Verify results
	if err != nil {
		t.Errorf("runGenerateCommand returned unexpected error: %v", err)
	}

	// Check if output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("Output file was not created at %s", outputPath)
	}
}
