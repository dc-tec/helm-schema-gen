package jsonschema

import (
	"context"
	"testing"
)

func TestTypeDetection(t *testing.T) {
	// Test utilities for string format detection
	t.Run("StringFormatDetection", func(t *testing.T) {
		tests := []struct {
			input   string
			isDate  bool
			isTime  bool
			isEmail bool
			isURI   bool
		}{
			{"2023-10-15", true, false, false, false},
			{"not-a-date", false, false, false, false},
			{"2023-10-15T14:30:00Z", false, true, false, false},
			{"user@example.com", false, false, true, false},
			{"https://example.com", false, false, false, true},
			{"http://example.com", false, false, false, true},
			{"just some text", false, false, false, false},
		}

		for _, test := range tests {
			t.Run(test.input, func(t *testing.T) {
				if got := isDate(test.input); got != test.isDate {
					t.Errorf("isDate(%q) = %v, want %v", test.input, got, test.isDate)
				}
				if got := isDateTime(test.input); got != test.isTime {
					t.Errorf("isDateTime(%q) = %v, want %v", test.input, got, test.isTime)
				}
				if got := isEmail(test.input); got != test.isEmail {
					t.Errorf("isEmail(%q) = %v, want %v", test.input, got, test.isEmail)
				}
				if got := isURI(test.input); got != test.isURI {
					t.Errorf("isURI(%q) = %v, want %v", test.input, got, test.isURI)
				}
			})
		}
	})

	t.Run("InferSchema", func(t *testing.T) {
		// Create a generator with default options
		generator := NewGenerator(GeneratorOptions{
			IncludeExamples: true,
		})

		ctx := context.Background()

		// Test nil value
		schema, err := generator.inferSchema(ctx, nil, "test.nil")
		if err != nil {
			t.Fatalf("inferSchema(nil) error: %v", err)
		}
		if schema.Type != TypeNull {
			t.Errorf("inferSchema(nil).Type = %v, want %v", schema.Type, TypeNull)
		}

		// Test boolean
		schema, err = generator.inferSchema(ctx, true, "test.bool")
		if err != nil {
			t.Fatalf("inferSchema(bool) error: %v", err)
		}
		if schema.Type != TypeBoolean {
			t.Errorf("inferSchema(bool).Type = %v, want %v", schema.Type, TypeBoolean)
		}
		if schema.Default != true {
			t.Errorf("inferSchema(bool).Default = %v, want %v", schema.Default, true)
		}

		// Test integer
		schema, err = generator.inferSchema(ctx, 42, "test.int")
		if err != nil {
			t.Fatalf("inferSchema(int) error: %v", err)
		}
		if schema.Type != TypeInteger {
			t.Errorf("inferSchema(int).Type = %v, want %v", schema.Type, TypeInteger)
		}
		if schema.Examples[0] != 42 {
			t.Errorf("inferSchema(int).Examples[0] = %v, want %v", schema.Examples[0], 42)
		}

		// Test float
		schema, err = generator.inferSchema(ctx, 3.14, "test.float")
		if err != nil {
			t.Fatalf("inferSchema(float) error: %v", err)
		}
		if schema.Type != TypeNumber {
			t.Errorf("inferSchema(float).Type = %v, want %v", schema.Type, TypeNumber)
		}
		if schema.Examples[0] != 3.14 {
			t.Errorf("inferSchema(float).Examples[0] = %v, want %v", schema.Examples[0], 3.14)
		}

		// Test string with format
		schema, err = generator.inferSchema(ctx, "user@example.com", "test.email")
		if err != nil {
			t.Fatalf("inferSchema(email) error: %v", err)
		}
		if schema.Type != TypeString {
			t.Errorf("inferSchema(email).Type = %v, want %v", schema.Type, TypeString)
		}
		if schema.Format != "email" {
			t.Errorf("inferSchema(email).Format = %q, want %q", schema.Format, "email")
		}

		// Test array
		schema, err = generator.inferSchema(ctx, []any{1, 2, 3}, "test.array")
		if err != nil {
			t.Fatalf("inferSchema(array) error: %v", err)
		}
		if schema.Type != TypeArray {
			t.Errorf("inferSchema(array).Type = %v, want %v", schema.Type, TypeArray)
		}
		if schema.Items == nil {
			t.Fatalf("inferSchema(array).Items is nil, expected schema")
		}
		if schema.Items.Type != TypeInteger {
			t.Errorf("inferSchema(array).Items.Type = %v, want %v", schema.Items.Type, TypeInteger)
		}

		// Test empty array
		schema, err = generator.inferSchema(ctx, []any{}, "test.emptyArray")
		if err != nil {
			t.Fatalf("inferSchema(emptyArray) error: %v", err)
		}
		if schema.Type != TypeArray {
			t.Errorf("inferSchema(emptyArray).Type = %v, want %v", schema.Type, TypeArray)
		}
		if schema.Items != nil {
			t.Errorf("inferSchema(emptyArray).Items = %v, want nil", schema.Items)
		}

		// Test object
		testObj := map[string]any{
			"name":    "Test",
			"enabled": true,
			"count":   42,
		}
		schema, err = generator.inferSchema(ctx, testObj, "test.object")
		if err != nil {
			t.Fatalf("inferSchema(object) error: %v", err)
		}
		if schema.Type != TypeObject {
			t.Errorf("inferSchema(object).Type = %v, want %v", schema.Type, TypeObject)
		}
		if len(schema.Properties) != 3 {
			t.Errorf("inferSchema(object) has %d properties, want 3", len(schema.Properties))
		}
		if schema.Properties["name"].Type != TypeString {
			t.Errorf("inferSchema(object).Properties[name].Type = %v, want %v",
				schema.Properties["name"].Type, TypeString)
		}
		if schema.Properties["enabled"].Type != TypeBoolean {
			t.Errorf("inferSchema(object).Properties[enabled].Type = %v, want %v",
				schema.Properties["enabled"].Type, TypeBoolean)
		}
		if schema.Properties["count"].Type != TypeInteger {
			t.Errorf("inferSchema(object).Properties[count].Type = %v, want %v",
				schema.Properties["count"].Type, TypeInteger)
		}
	})

	t.Run("MixedTypeDetection", func(t *testing.T) {
		// Test if hasMixedTypes properly detects arrays with mixed types
		tests := []struct {
			name     string
			items    []any
			expected bool
		}{
			{
				"Homogeneous integers",
				[]any{1, 2, 3, 4, 5},
				false,
			},
			{
				"Homogeneous strings",
				[]any{"a", "b", "c"},
				false,
			},
			{
				"Mixed types",
				[]any{1, "string", true, 3.14},
				true,
			},
			{
				"Mixed numerics",
				[]any{1, 2.5, 3, 4.1},
				true,
			},
			{
				"Single item",
				[]any{1},
				false,
			},
			{
				"Empty array",
				[]any{},
				false,
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				result := hasMixedTypes(test.items)
				if result != test.expected {
					t.Errorf("hasMixedTypes(%v) = %v, want %v", test.items, result, test.expected)
				}
			})
		}
	})

	t.Run("MultipleTypePaths", func(t *testing.T) {
		// Create a generator with default options
		generator := NewGenerator(GeneratorOptions{})
		ctx := context.Background()

		// Test cases for paths that should support multiple types
		tests := []struct {
			path     string
			value    any
			expected []SchemaType
		}{
			{
				"test.enabled",
				false,
				[]SchemaType{TypeBoolean, TypeString},
			},
			{
				"test.annotations",
				map[string]any{},
				[]SchemaType{TypeObject, TypeString},
			},
			{
				"test.resources.limits.memory",
				"512Mi",
				[]SchemaType{TypeString, TypeInteger, TypeNumber},
			},
			{
				"test.tolerations",
				[]any{},
				[]SchemaType{TypeNull, TypeArray, TypeString},
			},
		}

		for _, test := range tests {
			t.Run(test.path, func(t *testing.T) {
				schema, err := generator.inferSchema(ctx, test.value, test.path)
				if err != nil {
					t.Fatalf("inferSchema error: %v", err)
				}

				types, ok := schema.Type.([]SchemaType)
				if !ok {
					t.Fatalf("schema.Type is not []SchemaType, got: %T", schema.Type)
				}

				if len(types) != len(test.expected) {
					t.Fatalf("schema.Type has %d types, want %d", len(types), len(test.expected))
				}

				// Check that all expected types are present
				for i, expectedType := range test.expected {
					if types[i] != expectedType {
						t.Errorf("schema.Type[%d] = %v, want %v", i, types[i], expectedType)
					}
				}
			})
		}
	})
}
