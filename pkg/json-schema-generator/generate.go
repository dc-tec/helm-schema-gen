package jsonschema

import (
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"gopkg.in/yaml.v2"
)

// GenerateFromYAML generates a JSON schema from YAML data.
func (g *Generator) GenerateFromYAML(yamlData []byte) (*Schema, error) {
	logger := slog.Default().With("component", "json-schema-generator")
	logger.Info("generating schema from YAML data")

	// Parse YAML into a map
	var data map[string]interface{}
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	// Generate schema from the parsed data
	schema, err := g.GenerateFromMap(data)
	if err != nil {
		return nil, fmt.Errorf("failed to generate schema: %w", err)
	}

	// Extract and apply comments if enabled
	if g.Options.ExtractDescriptions {
		commentExtractor := NewCommentExtractor()
		commentExtractor.ExtractFromYAML(yamlData)
		commentExtractor.ApplyCommentsToSchema(schema)
	}

	return schema, nil
}

// GenerateFromMap generates a JSON schema from a map.
func (g *Generator) GenerateFromMap(data map[string]interface{}) (*Schema, error) {
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
		propSchema, err := g.inferSchema(value, key)
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

// inferSchema determines the JSON Schema type for a given value
func (g *Generator) inferSchema(value interface{}, path string) (*Schema, error) {
	// Handle nil values
	if value == nil {
		return &Schema{
			Type:     TypeNull,
			HelmPath: path,
		}, nil
	}

	// Use reflection to determine the type
	valueType := reflect.TypeOf(value)
	valueKind := valueType.Kind()

	schema := &Schema{
		HelmPath: path,
	}

	switch valueKind {
	case reflect.Bool:
		schema.Type = TypeBoolean
		if g.Options.IncludeExamples {
			schema.Default = value
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema.Type = TypeInteger
		if g.Options.IncludeExamples {
			schema.Examples = []interface{}{value}
		}

	case reflect.Float32, reflect.Float64:
		schema.Type = TypeNumber
		if g.Options.IncludeExamples {
			schema.Examples = []interface{}{value}
		}

	case reflect.String:
		schema.Type = TypeString
		strValue := value.(string)

		// Try to infer format
		if isDate(strValue) {
			schema.Format = "date"
		} else if isDateTime(strValue) {
			schema.Format = "date-time"
		} else if isEmail(strValue) {
			schema.Format = "email"
		} else if isURI(strValue) {
			schema.Format = "uri"
		}

		if g.Options.IncludeExamples {
			schema.Examples = []interface{}{strValue}
		}

	case reflect.Slice, reflect.Array:
		schema.Type = TypeArray

		// For empty arrays, we can't infer the items type
		sliceValue := reflect.ValueOf(value)
		if sliceValue.Len() == 0 {
			// Default to allowing any type
			schema.Items = &Schema{
				Type: "",
			}
		} else {
			// Get the first item to infer type
			firstItem := sliceValue.Index(0).Interface()
			itemPath := fmt.Sprintf("%s[0]", path)

			itemSchema, err := g.inferSchema(firstItem, itemPath)
			if err != nil {
				return nil, fmt.Errorf("failed to infer schema for array item: %w", err)
			}

			schema.Items = itemSchema
		}

	case reflect.Map:
		schema.Type = TypeObject
		schema.Properties = make(map[string]*Schema)

		mapValue := value.(map[string]interface{})
		var required []string

		for k, v := range mapValue {
			propertyPath := fmt.Sprintf("%s.%s", path, k)
			propSchema, err := g.inferSchema(v, propertyPath)
			if err != nil {
				return nil, fmt.Errorf("failed to infer schema for property '%s': %w", k, err)
			}

			schema.Properties[k] = propSchema

			// Add to required list if option enabled and value is non-nil
			if g.Options.RequireByDefault && v != nil {
				required = append(required, k)
			}
		}

		// Set required properties if any
		if len(required) > 0 {
			schema.Required = required
		}

	default:
		// For types we can't determine, don't specify a type
		// which allows any value according to JSON Schema
	}

	return schema, nil
}

// isDate checks if a string is a date in format YYYY-MM-DD
func isDate(s string) bool {
	// Very basic check - could be enhanced with regex
	return len(s) == 10 && s[4] == '-' && s[7] == '-'
}

// isDateTime checks if a string is a datetime in ISO 8601 format
func isDateTime(s string) bool {
	// Very basic check - could be enhanced with regex
	return len(s) >= 19 && s[4] == '-' && s[7] == '-' && s[10] == 'T'
}

// isEmail checks if a string appears to be an email address
func isEmail(s string) bool {
	return strings.Contains(s, "@") && strings.Contains(s, ".")
}

// isURI checks if a string appears to be a URI
func isURI(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// SpecializeSchemaForHelm enhances a schema with Helm-specific optimizations
func (g *Generator) SpecializeSchemaForHelm(schema *Schema) *Schema {
	// Check if this is an image configuration
	if g.isImageConfig(schema) {
		return g.createImageSchema(schema.HelmPath)
	}

	// Check if this is a resources configuration
	if g.isResourcesConfig(schema) {
		return g.createResourcesSchema(schema.HelmPath)
	}

	// Check for specialized schema for common patterns like ingress, volumes, etc.
	// ...

	// Recursively process nested properties
	if schema.Type == TypeObject && schema.Properties != nil {
		for key, propSchema := range schema.Properties {
			schema.Properties[key] = g.SpecializeSchemaForHelm(propSchema)
		}
	}

	// Process array items
	if schema.Type == TypeArray && schema.Items != nil {
		schema.Items = g.SpecializeSchemaForHelm(schema.Items)
	}

	return schema
}

// isImageConfig checks if a schema appears to represent a container image configuration
func (g *Generator) isImageConfig(schema *Schema) bool {
	if schema.Type != TypeObject || schema.Properties == nil {
		return false
	}

	// Check for common image properties
	hasRepository := false
	hasTag := false

	for key := range schema.Properties {
		switch key {
		case "repository":
			hasRepository = true
		case "tag":
			hasTag = true
		}
	}

	return hasRepository && hasTag
}

// createImageSchema creates a specialized schema for container image configuration
func (g *Generator) createImageSchema(path string) *Schema {
	return &Schema{
		Type:        TypeObject,
		Description: "Container image configuration",
		HelmPath:    path,
		Properties: map[string]*Schema{
			"repository": {
				Type:        TypeString,
				Description: "Container image repository",
			},
			"tag": {
				Type:        TypeString,
				Description: "Container image tag",
				Default:     "latest",
			},
			"pullPolicy": {
				Type:        TypeString,
				Description: "Image pull policy",
				Enum:        []interface{}{"Always", "IfNotPresent", "Never"},
				Default:     "IfNotPresent",
			},
		},
		Required: []string{"repository"},
	}
}

// isResourcesConfig checks if a schema appears to represent Kubernetes resources
func (g *Generator) isResourcesConfig(schema *Schema) bool {
	if schema.Type != TypeObject || schema.Properties == nil {
		return false
	}

	// Check for limits and requests properties
	hasLimits := false
	hasRequests := false

	for key := range schema.Properties {
		switch key {
		case "limits":
			hasLimits = true
		case "requests":
			hasRequests = true
		}
	}

	return hasLimits || hasRequests
}

// createResourcesSchema creates a specialized schema for Kubernetes resources
func (g *Generator) createResourcesSchema(path string) *Schema {
	resourceSchema := &Schema{
		Type:        TypeObject,
		Properties:  make(map[string]*Schema),
		Description: "CPU/Memory resource requirements",
		HelmPath:    path,
	}

	resourceLimitSchema := &Schema{
		Type:        TypeObject,
		Description: "Resource limits",
		Properties: map[string]*Schema{
			"cpu": {
				Type:        TypeString,
				Description: "CPU limit",
				Examples:    []interface{}{"100m", "0.1"},
			},
			"memory": {
				Type:        TypeString,
				Description: "Memory limit",
				Examples:    []interface{}{"128Mi", "1Gi"},
			},
		},
	}

	resourceRequestSchema := &Schema{
		Type:        TypeObject,
		Description: "Resource requests",
		Properties: map[string]*Schema{
			"cpu": {
				Type:        TypeString,
				Description: "CPU request",
				Examples:    []interface{}{"100m", "0.1"},
			},
			"memory": {
				Type:        TypeString,
				Description: "Memory request",
				Examples:    []interface{}{"128Mi", "1Gi"},
			},
		},
	}

	resourceSchema.Properties["limits"] = resourceLimitSchema
	resourceSchema.Properties["requests"] = resourceRequestSchema

	return resourceSchema
}
