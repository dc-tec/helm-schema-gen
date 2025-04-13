package tool

import (
	"context"

	"github.com/dc-tec/helm-schema-gen/pkg/cli"
	"github.com/dc-tec/helm-schema-gen/pkg/logging"
)

// GenerateSchema runs the schema generation command using the CLI package
func GenerateSchema(ctx context.Context) error {
	logger := logging.WithComponent(ctx, "schema-generator")
	ctx = logging.WithOperation(ctx, "generate-schema")

	logger.InfoContext(ctx, "starting schema generation")

	// Delegate to the CLI package for schema generation
	err := cli.ExecuteCLI()
	if err != nil {
		logger.ErrorContext(ctx, "schema generation failed", "error", err)
		return logging.LogError(ctx, err, "schema generation failed")
	}

	logger.InfoContext(ctx, "schema generation completed successfully")
	return nil
}
