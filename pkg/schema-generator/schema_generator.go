package jsonschema

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dc-tec/helm-schema-gen/pkg/logging"
	"gopkg.in/yaml.v2"
)

// GenerateFromYAML generates a JSON schema from YAML data.
func (g *Generator) GenerateFromYAML(ctx context.Context, yamlData []byte) (*Schema, error) {
	logger := logging.WithComponent(ctx, "json-schema-generator")
	logger.InfoContext(ctx, "generating schema from YAML data")

	// Parse YAML into a map
	var data any
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		logger.ErrorContext(ctx, "failed to unmarshal YAML", "error", err)
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	// Convert YAML map[any]any to map[string]any
	mappedData, err := convertYAMLToStringMap(data)
	if err != nil {
		logger.ErrorContext(ctx, "failed to convert YAML", "error", err)
		return nil, fmt.Errorf("failed to convert YAML: %w", err)
	}

	// Cast to map[string]any
	dataMap, ok := mappedData.(map[string]any)
	if !ok {
		logger.ErrorContext(ctx, "root YAML value must be a map", "type", fmt.Sprintf("%T", mappedData))
		return nil, fmt.Errorf("root YAML value must be a map, got %T", mappedData)
	}

	// Generate schema from the parsed data
	schema, err := g.GenerateFromMap(ctx, dataMap)
	if err != nil {
		logger.ErrorContext(ctx, "failed to generate schema", "error", err)
		return nil, fmt.Errorf("failed to generate schema: %w", err)
	}

	// Extract and apply comments if enabled
	if g.Options.ExtractDescriptions {
		logger.InfoContext(ctx, "extracting descriptions from comments")
		commentExtractor := NewCommentExtractor()

		// Enable debug mode if requested
		if g.Options.Debug {
			commentExtractor.Debug = true
		}

		commentExtractor.ExtractFromYAML(yamlData)

		// Print all comments when in debug mode
		if g.Options.Debug {
			commentExtractor.PrintAllComments()
		}

		// Check for top-level comment
		if topComment := commentExtractor.GetComment(""); topComment != "" && schema.Description == "" {
			schema.Description = topComment
			if g.Options.Debug {
				fmt.Fprintf(os.Stderr, "Applied top-level comment to schema description\n")
			}
		}

		commentExtractor.ApplyCommentsToSchema(schema)
	}

	logger.InfoContext(ctx, "schema generation completed")
	return schema, nil
}

// convertYAMLToStringMap converts YAML decoded map[any]any to map[string]any
func convertYAMLToStringMap(i any) (any, error) {
	switch x := i.(type) {
	case map[any]any:
		m := map[string]any{}
		for k, v := range x {
			str, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("non-string key in YAML: %v", k)
			}
			converted, err := convertYAMLToStringMap(v)
			if err != nil {
				return nil, err
			}
			m[str] = converted
		}
		return m, nil
	case []any:
		for i, v := range x {
			converted, err := convertYAMLToStringMap(v)
			if err != nil {
				return nil, err
			}
			x[i] = converted
		}
	}
	return i, nil
}

// GenerateFromMap generates a JSON schema from a map.
func (g *Generator) GenerateFromMap(ctx context.Context, data map[string]any) (*Schema, error) {
	// Create a root schema
	rootSchema := &Schema{
		Schema:      g.Options.SchemaVersion,
		Title:       g.Options.Title,
		Description: g.Options.Description,
		Type:        TypeObject,
		Properties:  make(map[string]*Schema),
	}

	// Track required properties
	var required []string

	// Process each property in the map
	for key, value := range data {
		propSchema, err := g.inferSchema(ctx, value, key)
		if err != nil {
			return nil, fmt.Errorf("failed to infer schema for property '%s': %w", key, err)
		}

		rootSchema.Properties[key] = propSchema

		// Add to required list if enabled and value is non-nil
		if g.Options.RequireByDefault && value != nil {
			required = append(required, key)
		}
	}

	// Set required properties if any
	if len(required) > 0 {
		rootSchema.Required = required
	}

	return rootSchema, nil
}

// isLikelyYAMLOrJSON checks if a string appears to be a YAML or JSON string
func isLikelyYAMLOrJSON(s string) bool {
	s = strings.TrimSpace(s)
	// Check for JSON-like patterns
	if (strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")) ||
		(strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")) {
		return true
	}

	// Check for YAML-like patterns (key: value pairs)
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// If it has a key-value pattern, it's likely YAML
		if strings.Contains(line, ":") {
			return true
		}
	}

	return false
}
