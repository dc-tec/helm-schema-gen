package jsonschema

import (
	"context"
	"reflect"
	"testing"
)

func TestGenerateFromYAML(t *testing.T) {
	// Create a test context
	ctx := context.Background()

	// Create a generator with test options
	generator := NewGenerator(GeneratorOptions{
		SchemaVersion:       Draft07,
		Title:               "Test Schema",
		ExtractDescriptions: true,
		IncludeExamples:     true,
	})

	// Define test cases
	testCases := []struct {
		name          string
		yaml          string
		shouldSucceed bool
		validateFunc  func(*testing.T, *Schema)
	}{
		{
			name: "Simple YAML",
			yaml: `
# Test values
key1: value1
key2: 42
nested:
  subkey: value
array:
  - item1
  - item2
`,
			shouldSucceed: true,
			validateFunc: func(t *testing.T, schema *Schema) {
				if schema.Type != TypeObject {
					t.Errorf("Expected root schema type to be object, got %v", schema.Type)
				}

				// Check that we have the correct properties
				expectedProps := []string{"key1", "key2", "nested", "array"}
				for _, prop := range expectedProps {
					if _, ok := schema.Properties[prop]; !ok {
						t.Errorf("Missing expected property %s", prop)
					}
				}

				// Check property types
				if schema.Properties["key1"].Type != TypeString {
					t.Errorf("Expected key1 to be string, got %v", schema.Properties["key1"].Type)
				}

				if schema.Properties["key2"].Type != TypeInteger {
					t.Errorf("Expected key2 to be integer, got %v", schema.Properties["key2"].Type)
				}

				if schema.Properties["nested"].Type != TypeObject {
					t.Errorf("Expected nested to be object, got %v", schema.Properties["nested"].Type)
				}

				if schema.Properties["array"].Type != TypeArray {
					t.Errorf("Expected array to be array, got %v", schema.Properties["array"].Type)
				}

				// Check comment extraction
				if schema.Description != "Test values" {
					t.Errorf("Expected description to be 'Test values', got '%s'", schema.Description)
				}
			},
		},
		{
			name: "Invalid YAML",
			yaml: `
invalid: : yaml
  structure:
 -broken
`,
			shouldSucceed: false,
		},
		{
			name:          "Empty YAML",
			yaml:          "",
			shouldSucceed: false,
		},
		{
			name:          "Empty Object YAML",
			yaml:          "{}",
			shouldSucceed: true,
			validateFunc: func(t *testing.T, schema *Schema) {
				if schema.Type != TypeObject {
					t.Errorf("Expected root schema type to be object, got %v", schema.Type)
				}

				// Check that Properties is initialized but empty
				if schema.Properties == nil {
					t.Errorf("Expected Properties to be initialized")
				}

				if len(schema.Properties) != 0 {
					t.Errorf("Expected empty properties for empty object, got %d properties", len(schema.Properties))
				}
			},
		},
		{
			name: "Multiple Type Support",
			yaml: `
# Test multiple type fields
annotations:
  app: nginx
enabled: true
`,
			shouldSucceed: true,
			validateFunc: func(t *testing.T, schema *Schema) {
				// Check that annotations is detected as object/string
				annotationsType, ok := schema.Properties["annotations"].Type.([]SchemaType)
				if !ok {
					t.Errorf("Expected annotations to have multiple types")
				} else if len(annotationsType) != 2 ||
					(annotationsType[0] != TypeObject && annotationsType[1] != TypeObject) ||
					(annotationsType[0] != TypeString && annotationsType[1] != TypeString) {
					t.Errorf("Expected annotations to be [object, string], got %v", annotationsType)
				}

				// Check that enabled is detected as boolean/string
				enabledType, ok := schema.Properties["enabled"].Type.([]SchemaType)
				if !ok {
					t.Errorf("Expected enabled to have multiple types")
				} else if len(enabledType) != 2 ||
					(enabledType[0] != TypeBoolean && enabledType[1] != TypeBoolean) ||
					(enabledType[0] != TypeString && enabledType[1] != TypeString) {
					t.Errorf("Expected enabled to be [boolean, string], got %v", enabledType)
				}
			},
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			schema, err := generator.GenerateFromYAML(ctx, []byte(tc.yaml))

			if tc.shouldSucceed {
				if err != nil {
					t.Fatalf("GenerateFromYAML failed: %v", err)
				}
				if schema == nil {
					t.Fatal("GenerateFromYAML returned nil schema")
				}
				if tc.validateFunc != nil {
					tc.validateFunc(t, schema)
				}
			} else {
				if err == nil {
					t.Fatal("GenerateFromYAML should have failed but didn't")
				}
			}
		})
	}
}

func TestConvertYAMLToStringMap(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected any
		wantErr  bool
	}{
		{
			name: "Simple map",
			input: map[any]any{
				"key1": "value1",
				"key2": 42,
			},
			expected: map[string]any{
				"key1": "value1",
				"key2": 42,
			},
			wantErr: false,
		},
		{
			name: "Nested map",
			input: map[any]any{
				"outer": map[any]any{
					"inner": "value",
				},
			},
			expected: map[string]any{
				"outer": map[string]any{
					"inner": "value",
				},
			},
			wantErr: false,
		},
		{
			name: "Map with array",
			input: map[any]any{
				"array": []any{
					"item1",
					map[any]any{"key": "value"},
				},
			},
			expected: map[string]any{
				"array": []any{
					"item1",
					map[string]any{"key": "value"},
				},
			},
			wantErr: false,
		},
		{
			name: "Non-string key",
			input: map[any]any{
				42: "value",
			},
			wantErr: true,
		},
		{
			name:     "String value",
			input:    "just a string",
			expected: "just a string",
			wantErr:  false,
		},
		{
			name:     "Integer value",
			input:    42,
			expected: 42,
			wantErr:  false,
		},
		{
			name:     "Nil value",
			input:    nil,
			expected: nil,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertYAMLToStringMap(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertYAMLToStringMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("convertYAMLToStringMap() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateFromMap(t *testing.T) {
	ctx := context.Background()

	// Create a generator with test options
	generator := NewGenerator(GeneratorOptions{
		SchemaVersion:    Draft07,
		Title:            "Test Schema",
		RequireByDefault: true,
	})

	testMap := map[string]any{
		"string":  "value",
		"integer": 42,
		"boolean": true,
		"null":    nil,
		"object": map[string]any{
			"nested": "value",
		},
		"array": []any{"item1", "item2"},
	}

	schema, err := generator.GenerateFromMap(ctx, testMap)
	if err != nil {
		t.Fatalf("GenerateFromMap failed: %v", err)
	}

	// Validate schema
	if schema.Type != TypeObject {
		t.Errorf("Expected root schema type to be object, got %v", schema.Type)
	}

	expectedTypes := map[string]SchemaType{
		"string":  TypeString,
		"integer": TypeInteger,
		"boolean": TypeBoolean,
		"null":    TypeNull,
		"object":  TypeObject,
		"array":   TypeArray,
	}

	for prop, expectedType := range expectedTypes {
		if _, ok := schema.Properties[prop]; !ok {
			t.Errorf("Missing expected property %s", prop)
			continue
		}

		propSchema := schema.Properties[prop]
		if prop != "null" && propSchema.Type != expectedType {
			t.Errorf("Property %s: expected type %v, got %v", prop, expectedType, propSchema.Type)
		}
	}

	// Check required properties
	if len(schema.Required) != 5 { // All except null should be required
		t.Errorf("Expected 5 required properties, got %d: %v", len(schema.Required), schema.Required)
	}

	// Test with RequireByDefault = false
	generator.Options.RequireByDefault = false
	schema, err = generator.GenerateFromMap(ctx, testMap)
	if err != nil {
		t.Fatalf("GenerateFromMap failed: %v", err)
	}

	if schema.Required != nil {
		t.Errorf("Expected no required properties when RequireByDefault=false, got %v", schema.Required)
	}
}

func TestIsLikelyYAMLOrJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "JSON Object",
			input:    `{"key": "value"}`,
			expected: true,
		},
		{
			name:     "JSON Array",
			input:    `["item1", "item2"]`,
			expected: true,
		},
		{
			name:     "YAML Key-Value",
			input:    "key: value",
			expected: true,
		},
		{
			name: "Multi-line YAML",
			input: `
key1: value1
key2: value2
`,
			expected: true,
		},
		{
			name:     "YAML with comments",
			input:    "# Comment\nkey: value",
			expected: true,
		},
		{
			name:     "Plain text",
			input:    "This is just plain text",
			expected: false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
		{
			name: "Multi-line text",
			input: `This is
a multi-line
text`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLikelyYAMLOrJSON(tt.input)
			if result != tt.expected {
				t.Errorf("isLikelyYAMLOrJSON(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
