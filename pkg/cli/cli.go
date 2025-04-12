// Package cli provides the command-line interface for helm-schema-gen
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	jsonschema "github.com/dc-tec/helm-schema-gen/pkg/json-schema-generator"
	"github.com/dc-tec/helm-schema-gen/pkg/logging"
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
	RequireByDefault    bool
	IncludeExamples     bool
	ExtractDescriptions bool

	// Application options
	Verbose bool
}

// DefaultOptions returns the default configuration options
func DefaultOptions() *Options {
	return &Options{
		InputFile:           "values.yaml",
		OutputFile:          "values.schema.json",
		SchemaVersion:       string(jsonschema.Draft07),
		Title:               "Helm Values Schema",
		RequireByDefault:    false,
		IncludeExamples:     true,
		ExtractDescriptions: true,
		Verbose:             false,
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
			return runGenerateCommand(opts)
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

	// Add application options
	rootCmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", opts.Verbose, "Enable verbose output")

	return rootCmd
}

// ExecuteCLI runs the CLI application
func ExecuteCLI() error {
	rootCmd := NewRootCommand()
	return rootCmd.Execute()
}

// runGenerateCommand handles the main schema generation logic
func runGenerateCommand(opts *Options) error {
	logger := logging.GetLogger().With("component", "cli")

	// Resolve input file path
	inputPath := opts.InputFile
	if !filepath.IsAbs(inputPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		inputPath = filepath.Join(cwd, inputPath)
	}

	// Check if input file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return fmt.Errorf("input file not found: %s", inputPath)
	}

	// Read input file
	yamlData, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	if opts.Verbose {
		logger.Info("read values file successfully", "path", inputPath, "size", len(yamlData))
	}

	// Configure generator options
	genOpts := jsonschema.GeneratorOptions{
		SchemaVersion:       jsonschema.SchemaVersion(opts.SchemaVersion),
		Title:               opts.Title,
		Description:         opts.Description,
		RequireByDefault:    opts.RequireByDefault,
		IncludeExamples:     opts.IncludeExamples,
		ExtractDescriptions: opts.ExtractDescriptions,
	}

	// Create schema generator
	generator := jsonschema.NewGenerator(genOpts)

	// Generate schema from YAML data
	schema, err := generator.GenerateFromYAML(yamlData)
	if err != nil {
		return fmt.Errorf("schema generation failed: %w", err)
	}

	// Apply Helm-specific optimizations
	optimizedSchema := generator.SpecializeSchemaForHelm(schema)

	// Resolve output file path
	outputPath := opts.OutputFile
	if !filepath.IsAbs(outputPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		outputPath = filepath.Join(cwd, outputPath)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create output file
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	// Write schema to file
	_, err = f.WriteString(optimizedSchema.String())
	if err != nil {
		return fmt.Errorf("failed to write schema to file: %w", err)
	}

	if opts.Verbose {
		logger.Info("schema generation completed", "output", outputPath)
	} else {
		fmt.Printf("Schema generated successfully: %s\n", outputPath)
	}

	return nil
}
