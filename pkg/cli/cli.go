// Package cli provides the command-line interface for helm-schema-gen
package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dc-tec/helm-schema-gen/pkg/logging"
	jsonschema "github.com/dc-tec/helm-schema-gen/pkg/schema-generator"
	"github.com/spf13/cobra"
)

// Configuration options for the CLI
type Options struct {
	// Input/output options
	InputFile  string
	OutputFile string

	// Schema generation options
	SchemaVersion string
	Title         string
	Description   string

	// Schema validation options
	RequireByDefault      bool
	IncludeExamples       bool
	ExtractDescriptions   bool
	ValidateBestPractices bool

	// Application options
	Verbose bool
	Debug   bool
}

// DefaultOptions returns the default configuration options
func DefaultOptions() *Options {
	return &Options{
		InputFile:             "values.yaml",
		OutputFile:            "values.schema.json",
		SchemaVersion:         string(jsonschema.Draft07),
		Title:                 "Helm Values Schema",
		RequireByDefault:      false,
		IncludeExamples:       true,
		ExtractDescriptions:   true,
		ValidateBestPractices: false,
		Verbose:               false,
		Debug:                 false,
	}
}

// NewRootCommand creates and returns the root command for the application
func NewRootCommand() *cobra.Command {
	opts := DefaultOptions()

	rootCmd := &cobra.Command{
		Use:   "helm-schema-gen",
		Short: "Generate JSON Schema for Helm charts",
		Long: `helm-schema-gen generates JSON Schema from Helm chart values.yaml files.
These schemas can be used for validation and providing better IDE support.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			return runGenerateCommand(ctx, opts)
		},
	}

	// Add flags for input/output files
	rootCmd.Flags().StringVarP(&opts.InputFile, "file", "f", opts.InputFile, "Input values.yaml file")
	rootCmd.Flags().StringVarP(&opts.OutputFile, "output", "o", opts.OutputFile, "Output schema file")

	// Add schema generation flags
	rootCmd.Flags().StringVar(&opts.SchemaVersion, "schema-version", opts.SchemaVersion, "JSON Schema version to use")
	rootCmd.Flags().StringVar(&opts.Title, "title", opts.Title, "Schema title")
	rootCmd.Flags().StringVar(&opts.Description, "description", opts.Description, "Schema description")

	// Add validation options
	rootCmd.Flags().BoolVar(&opts.RequireByDefault, "require-all", opts.RequireByDefault, "Make all properties required")
	rootCmd.Flags().BoolVar(&opts.IncludeExamples, "include-examples", opts.IncludeExamples, "Include examples from values")
	rootCmd.Flags().BoolVar(&opts.ExtractDescriptions, "extract-descriptions", opts.ExtractDescriptions, "Extract descriptions from comments")
	rootCmd.Flags().BoolVar(&opts.ValidateBestPractices, "validate", opts.ValidateBestPractices, "Validate schema against Helm best practices")

	// Add application options
	rootCmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", opts.Verbose, "Enable verbose output")
	rootCmd.Flags().BoolVar(&opts.Debug, "debug", opts.Debug, "Enable debug output")

	return rootCmd
}

// ExecuteCLI runs the CLI application
func ExecuteCLI() error {
	rootCmd := NewRootCommand()
	ctx := context.Background()
	rootCmd.SetContext(ctx)
	return rootCmd.Execute()
}

// validatePath checks if a file path is safe by ensuring it doesn't contain suspicious patterns
func validatePath(ctx context.Context, path string) error {
	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return logging.LogError(ctx, fmt.Errorf("path contains forbidden pattern: %s", path), "path contains forbidden pattern")
	}

	// Normalize the path to check for other potential issues
	cleanPath := filepath.Clean(path)
	if cleanPath != path {
		return fmt.Errorf("path contains potentially unsafe elements: %s", path)
	}

	return nil
}

// runGenerateCommand handles the main schema generation logic
func runGenerateCommand(ctx context.Context, opts *Options) error {
	logger := logging.WithComponent(ctx, "cli")

	// Resolve input file path
	inputPath := opts.InputFile
	if !filepath.IsAbs(inputPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		inputPath = filepath.Join(cwd, inputPath)
	}

	// Validate input path for security
	if err := validatePath(ctx, inputPath); err != nil {
		return fmt.Errorf("invalid input file path: %w", err)
	}

	// Check if input file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return fmt.Errorf("input file not found: %s", inputPath)
	}

	// Read input file - #nosec G304 is used because we've validated the path
	yamlData, err := os.ReadFile(inputPath) // #nosec G304
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	if opts.Verbose {
		logger.InfoContext(ctx, "read values file successfully", "path", inputPath, "size", len(yamlData))
	}

	// Configure generator options
	genOpts := jsonschema.GeneratorOptions{
		SchemaVersion:       jsonschema.SchemaVersion(opts.SchemaVersion),
		Title:               opts.Title,
		Description:         opts.Description,
		RequireByDefault:    opts.RequireByDefault,
		IncludeExamples:     opts.IncludeExamples,
		ExtractDescriptions: opts.ExtractDescriptions,
		Debug:               opts.Debug,
	}

	// Create schema generator
	generator := jsonschema.NewGenerator(genOpts)

	// Generate schema from YAML data
	schema, err := generator.GenerateFromYAML(ctx, yamlData)
	if err != nil {
		return fmt.Errorf("schema generation failed: %w", err)
	}

	// Run validation if requested
	if opts.ValidateBestPractices {
		issues := jsonschema.ValidateHelmBestPractices(schema)
		if len(issues) > 0 {
			formattedIssues := jsonschema.FormatValidationIssues(issues)
			fmt.Println("\nHelm Best Practices Validation:")
			fmt.Println(formattedIssues)
		} else if opts.Verbose {
			logger.InfoContext(ctx, "no best practice issues found")
		}
	}

	// Resolve output file path
	outputPath := opts.OutputFile
	if !filepath.IsAbs(outputPath) {
		cwd, err := os.Getwd()
		if err != nil {
			logger.ErrorContext(ctx, "failed to get current directory", "error", err)
			return logging.LogError(ctx, err, "failed to get current directory")
		}
		outputPath = filepath.Join(cwd, outputPath)
	}

	// Validate output path for security
	if err := validatePath(ctx, outputPath); err != nil {
		return fmt.Errorf("invalid output file path: %w", err)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		logger.ErrorContext(ctx, "failed to create output directory", "error", err)
		return logging.LogError(ctx, err, "failed to create output directory")
	}

	// Create output file - #nosec G304 is used because we've validated the path
	f, err := os.Create(outputPath) // #nosec G304
	if err != nil {
		logger.ErrorContext(ctx, "failed to create output file", "error", err)
		return logging.LogError(ctx, err, "failed to create output file")
	}
	defer f.Close()

	// Write schema to file
	_, err = f.WriteString(schema.String())
	if err != nil {
		logger.ErrorContext(ctx, "failed to write schema to file", "error", err)
		return logging.LogError(ctx, err, "failed to write schema to file")
	}

	if opts.Verbose {
		logger.InfoContext(ctx, "schema generation completed", "output", outputPath)
	} else {
		logger.InfoContext(ctx, "schema generated successfully", "output", outputPath)
	}

	return nil
}
