// Package jsonschema provides functionality for generating JSON Schema from Go values.
package jsonschema

import (
	"encoding/json"
	"fmt"
)

// SchemaVersion represents a JSON Schema specification version
type SchemaVersion string

const (
	// Draft07 represents JSON Schema draft-07
	Draft07 SchemaVersion = "http://json-schema.org/draft-07/schema#"
	// Draft2019 represents JSON Schema 2019-09
	Draft2019 SchemaVersion = "https://json-schema.org/draft/2019-09/schema"
	// Draft2020 represents JSON Schema 2020-12
	Draft2020 SchemaVersion = "https://json-schema.org/draft/2020-12/schema"
)

// SchemaType represents a JSON Schema type
type SchemaType string

const (
	TypeArray   SchemaType = "array"
	TypeBoolean SchemaType = "boolean"
	TypeInteger SchemaType = "integer"
	TypeNull    SchemaType = "null"
	TypeNumber  SchemaType = "number"
	TypeObject  SchemaType = "object"
	TypeString  SchemaType = "string"
)

// Schema represents a JSON Schema document
type Schema struct {
	// Core schema metadata
	ID          string        `json:"$id,omitempty"`
	Schema      SchemaVersion `json:"$schema,omitempty"`
	Title       string        `json:"title,omitempty"`
	Description string        `json:"description,omitempty"`

	// Type information
	Type   any    `json:"type,omitempty"`
	Format string `json:"format,omitempty"`

	// Logical composition
	OneOf []any `json:"oneOf,omitempty"`
	AnyOf []any `json:"anyOf,omitempty"`
	AllOf []any `json:"allOf,omitempty"`

	// Object-specific properties
	Properties map[string]*Schema `json:"properties,omitempty"`
	Required   []string           `json:"required,omitempty"`

	// Array-specific properties
	Items    *Schema `json:"items,omitempty"`
	MinItems *int    `json:"minItems,omitempty"`
	MaxItems *int    `json:"maxItems,omitempty"`

	// Number-specific properties
	Minimum *float64 `json:"minimum,omitempty"`
	Maximum *float64 `json:"maximum,omitempty"`

	// String-specific properties
	MinLength *int   `json:"minLength,omitempty"`
	MaxLength *int   `json:"maxLength,omitempty"`
	Pattern   string `json:"pattern,omitempty"`

	// Common constraints
	Enum     []any `json:"enum,omitempty"`
	Default  any   `json:"default,omitempty"`
	Examples []any `json:"examples,omitempty"`

	// Additional schema features
	Definitions map[string]*Schema `json:"definitions,omitempty"`
	Ref         string             `json:"$ref,omitempty"`

	// Additional metadata for Helm values
	HelmPath string `json:"-"` // Used internally, not rendered in final schema
}

// MarshalJSON customizes the JSON output for the Schema type
func (s Schema) MarshalJSON() ([]byte, error) {
	// Use a separate type to avoid infinite recursion
	type SchemaAlias Schema
	return json.Marshal(SchemaAlias(s))
}

// String returns a JSON string representation of the schema
func (s Schema) String() string {
	bytes, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error marshaling schema: %v", err)
	}
	return string(bytes)
}

// GeneratorOptions configures the behavior of schema generation
type GeneratorOptions struct {
	// SchemaVersion specifies the JSON Schema version to use
	SchemaVersion SchemaVersion

	// Title for the root schema
	Title string

	// Description for the root schema
	Description string

	// RequireByDefault generates required array for all properties
	// when this is true
	RequireByDefault bool

	// IncludeExamples includes example values from the source data
	IncludeExamples bool

	// ExtractDescriptions attempts to extract descriptions from comments
	// or field metadata
	ExtractDescriptions bool

	// UseFullyQualifiedIDs generates fully qualified IDs for all schemas
	UseFullyQualifiedIDs bool

	// Debug enables additional debug output during generation
	Debug bool
}

// DefaultOptions returns the default generator options
func DefaultOptions() GeneratorOptions {
	return GeneratorOptions{
		SchemaVersion:        Draft07,
		Title:                "Helm Values Schema",
		RequireByDefault:     false,
		IncludeExamples:      true,
		ExtractDescriptions:  true,
		UseFullyQualifiedIDs: false,
		Debug:                false,
	}
}

// Generator handles the generation of JSON Schema from different data sources
type Generator struct {
	Options GeneratorOptions
	schema  *Schema
}

// NewGenerator creates a new schema generator with the specified options
func NewGenerator(options GeneratorOptions) *Generator {
	return &Generator{
		Options: options,
		schema: &Schema{
			Schema:      options.SchemaVersion,
			Title:       options.Title,
			Description: options.Description,
		},
	}
}

// NewGeneratorWithDefaults creates a new schema generator with default options
func NewGeneratorWithDefaults() *Generator {
	return NewGenerator(DefaultOptions())
}
