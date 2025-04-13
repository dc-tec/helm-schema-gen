package jsonschema

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/dc-tec/helm-schema-gen/pkg/logging"
)

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

// inferSchema determines the JSON Schema type for a given value
func (g *Generator) inferSchema(ctx context.Context, value any, path string) (*Schema, error) {
	logger := logging.WithComponent(ctx, "json-schema-generator")

	// Handle nil values
	if value == nil {
		schema := &Schema{
			Type:     TypeNull,
			HelmPath: path,
		}

		// Check if this path should support multiple types
		if hasMultipleTypes, types := shouldSupportMultipleTypes(path); hasMultipleTypes {
			schema.Type = types
		}

		return schema, nil
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

		// Check if this path should support multiple types
		if hasMultipleTypes, types := shouldSupportMultipleTypes(path); hasMultipleTypes {
			schema.Type = types
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema.Type = TypeInteger
		if g.Options.IncludeExamples {
			schema.Examples = []any{value}
		}

		// Check if this path should support multiple types
		if hasMultipleTypes, types := shouldSupportMultipleTypes(path); hasMultipleTypes {
			schema.Type = types
		}

	case reflect.Float32, reflect.Float64:
		schema.Type = TypeNumber
		if g.Options.IncludeExamples {
			schema.Examples = []any{value}
		}

		// Check if this path should support multiple types
		if hasMultipleTypes, types := shouldSupportMultipleTypes(path); hasMultipleTypes {
			schema.Type = types
		}

	case reflect.String:
		strValue := value.(string)

		// Check if this path should support multiple types
		if hasMultipleTypes, types := shouldSupportMultipleTypes(path); hasMultipleTypes {
			schema.Type = types
		} else if isLikelyYAMLOrJSON(strValue) {
			// For fields that could be both string and object/array
			schema.Type = []SchemaType{TypeString, TypeObject}
		} else {
			schema.Type = TypeString

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
		}

		// Special case for "-" values in enabled fields
		if (path == "enabled" || strings.HasSuffix(path, ".enabled")) && strValue == "-" {
			schema.Type = []SchemaType{TypeString, TypeBoolean}
		}

		if g.Options.IncludeExamples {
			schema.Examples = []any{strValue}
		}

	case reflect.Slice, reflect.Array:
		schema.Type = TypeArray

		// For empty arrays, we can't infer the items type
		sliceValue := reflect.ValueOf(value)
		if sliceValue.Len() == 0 {
			// For empty arrays, we'll omit the items field entirely
			// This is valid in JSON Schema and means "any type" for array items

			// Check if this path should support multiple types
			if hasMultipleTypes, types := shouldSupportMultipleTypes(path); hasMultipleTypes {
				schema.Type = types
			}
		} else {
			// Check if array has mixed types and handle appropriately
			sliceInterface := make([]any, sliceValue.Len())
			for i := 0; i < sliceValue.Len(); i++ {
				sliceInterface[i] = sliceValue.Index(i).Interface()
			}

			if hasMixedTypes(sliceInterface) {
				// Use our new method for mixed type arrays
				mixedSchema, err := g.InferArrayItemsWithMultipleTypes(ctx, sliceInterface, path)
				if err != nil {
					logger.ErrorContext(ctx, "failed to infer schema for mixed type array", "path", path, "error", err)
					return nil, fmt.Errorf("failed to infer schema for mixed type array: %w", err)
				}
				// We want to keep the array type but use the types from the mixed type handling
				schema.Items = &Schema{
					Type: mixedSchema.Type,
				}
			} else {
				// Get the first item to infer type for homogeneous arrays
				firstItem := sliceValue.Index(0).Interface()
				itemPath := fmt.Sprintf("%s[0]", path)

				itemSchema, err := g.inferSchema(ctx, firstItem, itemPath)
				if err != nil {
					logger.ErrorContext(ctx, "failed to infer schema for array item", "path", itemPath, "error", err)
					return nil, fmt.Errorf("failed to infer schema for array item: %w", err)
				}

				schema.Items = itemSchema
			}

			// Check if this path should support multiple types
			if hasMultipleTypes, types := shouldSupportMultipleTypes(path); hasMultipleTypes {
				schema.Type = types
			}
		}

	case reflect.Map:
		schema.Type = TypeObject
		schema.Properties = make(map[string]*Schema)

		mapValue := value.(map[string]any)
		var required []string

		// First check if this map itself contains keys that suggest it supports multiple types
		for k := range mapValue {
			// Keys that often indicate object/string support
			if strings.HasSuffix(strings.ToLower(k), "annotations") ||
				strings.HasSuffix(strings.ToLower(k), "labels") ||
				strings.HasSuffix(strings.ToLower(k), "nodeselector") ||
				strings.HasSuffix(strings.ToLower(k), "affinity") ||
				strings.HasSuffix(strings.ToLower(k), "selector") {
				schema.Type = []SchemaType{TypeObject, TypeString}
				break
			}

			// Keys that often indicate we should allow null/array/string
			if strings.HasSuffix(strings.ToLower(k), "tolerations") ||
				strings.HasSuffix(strings.ToLower(k), "volumes") ||
				strings.HasSuffix(strings.ToLower(k), "mounts") {
				schema.Type = []SchemaType{TypeObject, TypeString}
				break
			}
		}

		for k, v := range mapValue {
			propertyPath := fmt.Sprintf("%s.%s", path, k)
			propSchema, err := g.inferSchema(ctx, v, propertyPath)
			if err != nil {
				logger.ErrorContext(ctx, "failed to infer schema for property", "property", k, "path", propertyPath, "error", err)
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

		// Check if this path should support multiple types
		if hasMultipleTypes, types := shouldSupportMultipleTypes(path); hasMultipleTypes {
			schema.Type = types
		}

	default:
		// For types we can't determine, don't specify a type
		// which allows any value according to JSON Schema
		logger.InfoContext(ctx, "encountered unknown type", "path", path, "type", valueKind.String())
	}

	return schema, nil
}

// CreateUnionTypeSchema creates a union type schema from a list of types
func (g *Generator) CreateUnionTypeSchema(types []SchemaType, path string) *Schema {
	return &Schema{
		Type:     types,
		HelmPath: path,
	}
}

// hasMixedTypes checks if an array contains elements of different types
func hasMixedTypes(items []any) bool {
	if len(items) <= 1 {
		return false
	}

	var foundTypes = make(map[string]bool)

	for _, item := range items {
		if item == nil {
			foundTypes["null"] = true
			continue
		}

		// Use reflection to determine the type
		itemType := reflect.TypeOf(item)
		itemKind := itemType.Kind()

		// Distinguish between integers and floats
		switch itemKind {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			foundTypes["integer"] = true
		case reflect.Float32, reflect.Float64:
			foundTypes["float"] = true
		default:
			// For other types, use the kind string
			foundTypes[itemKind.String()] = true
		}

		// If we have more than one type, return early
		if len(foundTypes) > 1 {
			return true
		}
	}

	return len(foundTypes) > 1
}

// InferArrayItemsWithMultipleTypes is a method on Generator to handle arrays with mixed types
func (g *Generator) InferArrayItemsWithMultipleTypes(ctx context.Context, items []any, path string) (*Schema, error) {
	// If array contains mixed types (e.g., strings and numbers)
	if hasMixedTypes(items) {
		// Create a type map to track unique types
		typeMap := make(map[string]bool)

		// Analyze all items for their types
		for _, item := range items {
			if item == nil {
				typeMap["null"] = true
				continue
			}

			itemKind := reflect.TypeOf(item).Kind()

			switch itemKind {
			case reflect.Bool:
				typeMap["boolean"] = true
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
				reflect.Float32, reflect.Float64:
				typeMap["number"] = true
			case reflect.String:
				typeMap["string"] = true
			case reflect.Map:
				typeMap["object"] = true
			case reflect.Slice, reflect.Array:
				typeMap["array"] = true
			}
		}

		// Create a list of SchemaType values
		var typeArray []SchemaType
		for typeName := range typeMap {
			switch typeName {
			case "null":
				typeArray = append(typeArray, TypeNull)
			case "boolean":
				typeArray = append(typeArray, TypeBoolean)
			case "integer":
				typeArray = append(typeArray, TypeInteger)
			case "number":
				typeArray = append(typeArray, TypeNumber)
			case "string":
				typeArray = append(typeArray, TypeString)
			case "object":
				typeArray = append(typeArray, TypeObject)
			case "array":
				typeArray = append(typeArray, TypeArray)
			}
		}

		// For mixed types, return a schema with multiple types directly
		return &Schema{
			Type:     typeArray,
			HelmPath: path,
		}, nil
	}

	// If all items are the same type, infer schema from the first item
	if len(items) > 0 {
		itemPath := fmt.Sprintf("%s[0]", path)
		firstItem := items[0]
		return g.inferSchema(ctx, firstItem, itemPath)
	}

	// For empty arrays, return a schema without items field (allows any type)
	return &Schema{
		Type:     TypeArray,
		HelmPath: path,
	}, nil
}
