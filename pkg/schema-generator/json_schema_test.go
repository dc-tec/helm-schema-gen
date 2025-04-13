package jsonschema

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSchema_MarshalJSON(t *testing.T) {
	// Create a test schema
	schema := Schema{
		Schema:      Draft07,
		Title:       "Test Schema",
		Description: "Test Description",
		Type:        TypeObject,
		Properties: map[string]*Schema{
			"stringProp": {
				Type:    TypeString,
				Default: "default",
			},
			"integerProp": {
				Type:     TypeInteger,
				Minimum:  floatPtr(0),
				Maximum:  floatPtr(100),
				Examples: []any{42, 64},
			},
		},
		Required: []string{"stringProp"},
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("Failed to marshal schema: %v", err)
	}

	// Unmarshal back to verify
	var unmarshaled Schema
	if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal schema: %v", err)
	}

	// Verify fields
	if unmarshaled.Schema != Draft07 {
		t.Errorf("Expected Schema to be %v, got %v", Draft07, unmarshaled.Schema)
	}

	if unmarshaled.Title != "Test Schema" {
		t.Errorf("Expected Title to be 'Test Schema', got '%s'", unmarshaled.Title)
	}

	if unmarshaled.Type != TypeObject {
		t.Errorf("Expected Type to be %v, got %v", TypeObject, unmarshaled.Type)
	}

	if len(unmarshaled.Properties) != 2 {
		t.Errorf("Expected 2 properties, got %d", len(unmarshaled.Properties))
	}

	if len(unmarshaled.Required) != 1 || unmarshaled.Required[0] != "stringProp" {
		t.Errorf("Expected Required to be ['stringProp'], got %v", unmarshaled.Required)
	}

	// Check property details
	stringProp := unmarshaled.Properties["stringProp"]
	if stringProp == nil {
		t.Fatal("Missing stringProp")
	}
	if stringProp.Type != TypeString {
		t.Errorf("Expected stringProp.Type to be %v, got %v", TypeString, stringProp.Type)
	}
	if stringProp.Default != "default" {
		t.Errorf("Expected stringProp.Default to be 'default', got '%v'", stringProp.Default)
	}

	intProp := unmarshaled.Properties["integerProp"]
	if intProp == nil {
		t.Fatal("Missing integerProp")
	}
	if intProp.Type != TypeInteger {
		t.Errorf("Expected integerProp.Type to be %v, got %v", TypeInteger, intProp.Type)
	}

	if *intProp.Minimum != 0 {
		t.Errorf("Expected integerProp.Minimum to be 0, got %v", *intProp.Minimum)
	}
	if *intProp.Maximum != 100 {
		t.Errorf("Expected integerProp.Maximum to be 100, got %v", *intProp.Maximum)
	}

	if len(intProp.Examples) != 2 {
		t.Errorf("Expected 2 examples, got %d", len(intProp.Examples))
	}
}

func TestSchema_String(t *testing.T) {
	// Create a simple test schema
	schema := Schema{
		Schema:      Draft07,
		Title:       "Test Schema",
		Description: "Test Description",
		Type:        TypeObject,
		Properties: map[string]*Schema{
			"prop": {
				Type: TypeString,
			},
		},
	}

	// Get string representation
	str := schema.String()

	// Verify basic JSON structure
	if !strings.Contains(str, `"$schema": "http://json-schema.org/draft-07/schema#"`) {
		t.Errorf("String representation missing $schema: %s", str)
	}
	if !strings.Contains(str, `"title": "Test Schema"`) {
		t.Errorf("String representation missing title: %s", str)
	}
	if !strings.Contains(str, `"description": "Test Description"`) {
		t.Errorf("String representation missing description: %s", str)
	}
	if !strings.Contains(str, `"type": "object"`) {
		t.Errorf("String representation missing type: %s", str)
	}
	if !strings.Contains(str, `"properties": {`) {
		t.Errorf("String representation missing properties: %s", str)
	}
	if !strings.Contains(str, `"prop": {`) {
		t.Errorf("String representation missing prop: %s", str)
	}
	if !strings.Contains(str, `"type": "string"`) {
		t.Errorf("String representation missing string type: %s", str)
	}

	// Try to parse it as JSON to verify it's valid
	var parsed map[string]any
	if err := json.Unmarshal([]byte(str), &parsed); err != nil {
		t.Errorf("String representation isn't valid JSON: %v\n%s", err, str)
	}
}

func TestDefaultOptions(t *testing.T) {
	options := DefaultOptions()

	// Verify defaults
	if options.SchemaVersion != Draft07 {
		t.Errorf("Expected SchemaVersion to be %v, got %v", Draft07, options.SchemaVersion)
	}
	if options.Title != "Helm Values Schema" {
		t.Errorf("Expected Title to be 'Helm Values Schema', got '%s'", options.Title)
	}
	if options.RequireByDefault {
		t.Error("Expected RequireByDefault to be false")
	}
	if !options.IncludeExamples {
		t.Error("Expected IncludeExamples to be true")
	}
	if !options.ExtractDescriptions {
		t.Error("Expected ExtractDescriptions to be true")
	}
	if options.UseFullyQualifiedIDs {
		t.Error("Expected UseFullyQualifiedIDs to be false")
	}
	if options.Debug {
		t.Error("Expected Debug to be false")
	}
}

func TestNewGenerator(t *testing.T) {
	// Test with custom options
	customOptions := GeneratorOptions{
		SchemaVersion:    Draft2020,
		Title:            "Custom Title",
		Description:      "Custom Description",
		RequireByDefault: true,
		Debug:            true,
	}

	generator := NewGenerator(customOptions)

	// Verify options are set
	if generator.Options.SchemaVersion != Draft2020 {
		t.Errorf("Expected SchemaVersion to be %v, got %v", Draft2020, generator.Options.SchemaVersion)
	}
	if generator.Options.Title != "Custom Title" {
		t.Errorf("Expected Title to be 'Custom Title', got '%s'", generator.Options.Title)
	}
	if generator.Options.Description != "Custom Description" {
		t.Errorf("Expected Description to be 'Custom Description', got '%s'", generator.Options.Description)
	}
	if !generator.Options.RequireByDefault {
		t.Error("Expected RequireByDefault to be true")
	}
	if !generator.Options.Debug {
		t.Error("Expected Debug to be true")
	}

	// Verify schema properties
	if generator.schema.Schema != Draft2020 {
		t.Errorf("Expected schema.Schema to be %v, got %v", Draft2020, generator.schema.Schema)
	}
	if generator.schema.Title != "Custom Title" {
		t.Errorf("Expected schema.Title to be 'Custom Title', got '%s'", generator.schema.Title)
	}
	if generator.schema.Description != "Custom Description" {
		t.Errorf("Expected schema.Description to be 'Custom Description', got '%s'", generator.schema.Description)
	}
}

func TestNewGeneratorWithDefaults(t *testing.T) {
	generator := NewGeneratorWithDefaults()

	// Verify options match defaults
	defaultOptions := DefaultOptions()
	if generator.Options.SchemaVersion != defaultOptions.SchemaVersion {
		t.Errorf("Expected SchemaVersion to match defaults, got %v", generator.Options.SchemaVersion)
	}
	if generator.Options.Title != defaultOptions.Title {
		t.Errorf("Expected Title to match defaults, got '%s'", generator.Options.Title)
	}
	if generator.Options.RequireByDefault != defaultOptions.RequireByDefault {
		t.Errorf("Expected RequireByDefault to match defaults, got %v", generator.Options.RequireByDefault)
	}
	if generator.Options.IncludeExamples != defaultOptions.IncludeExamples {
		t.Errorf("Expected IncludeExamples to match defaults, got %v", generator.Options.IncludeExamples)
	}
	if generator.Options.ExtractDescriptions != defaultOptions.ExtractDescriptions {
		t.Errorf("Expected ExtractDescriptions to match defaults, got %v", generator.Options.ExtractDescriptions)
	}
	if generator.Options.UseFullyQualifiedIDs != defaultOptions.UseFullyQualifiedIDs {
		t.Errorf("Expected UseFullyQualifiedIDs to match defaults, got %v", generator.Options.UseFullyQualifiedIDs)
	}
	if generator.Options.Debug != defaultOptions.Debug {
		t.Errorf("Expected Debug to match defaults, got %v", generator.Options.Debug)
	}

	// Verify schema properties
	if generator.schema.Schema != defaultOptions.SchemaVersion {
		t.Errorf("Expected schema.Schema to match defaults, got %v", generator.schema.Schema)
	}
	if generator.schema.Title != defaultOptions.Title {
		t.Errorf("Expected schema.Title to match defaults, got '%s'", generator.schema.Title)
	}
}

// Helper function to create float pointers
func floatPtr(v float64) *float64 {
	return &v
}
