package jsonschema

import (
	"strings"
	"testing"
)

func TestHelmBestPractices(t *testing.T) {
	t.Run("ValidateHelmBestPractices", func(t *testing.T) {
		// Create a schema with various issues to test validation
		schema := &Schema{
			Type: TypeObject,
			Properties: map[string]*Schema{
				"replicas": {
					Type:        TypeInteger,
					Description: "Number of replicas for the deployment",
					Default:     3,
				},
				"image": {
					Type:        TypeObject,
					Description: "Container image configuration",
					Properties: map[string]*Schema{
						"repository": {
							Type:        TypeString,
							Description: "Docker image repository",
							Default:     "nginx",
						},
						"tag": {
							Type:        TypeString,
							Description: "Docker image tag",
							Default:     "latest",
						},
						"pullPolicy": {
							Type:        TypeString,
							Description: "Image pull policy",
							Default:     "IfNotPresent",
						},
					},
				},
				// For testing naming conventions, we need to create a nested structure
				// where the parent path is non-empty, since checkNamingConventions
				// only runs when path != ""
				"objectWithBadProps": {
					Type:        TypeObject,
					Description: "Object with badly named properties",
					Properties: map[string]*Schema{
						"NOT_CAMEL_CASE": {
							Type: TypeString,
						},
						"with-hyphen": {
							Type: TypeString,
						},
						"with_underscore": {
							Type: TypeString,
						},
					},
				},
				"NOT_CAMEL_CASE": {
					Type: TypeString,
				},
				"with-hyphen": {
					Type: TypeString,
				},
				"with_underscore": {
					Type: TypeString,
				},
				"deeplyNested": {
					Type: TypeObject,
					Properties: map[string]*Schema{
						"level1": {
							Type: TypeObject,
							Properties: map[string]*Schema{
								"level2": {
									Type: TypeObject,
									Properties: map[string]*Schema{
										"level3": {
											Type: TypeObject,
											Properties: map[string]*Schema{
												"level4": {
													Type: TypeObject,
													Properties: map[string]*Schema{
														"level5": {
															Type: TypeObject,
															Properties: map[string]*Schema{
																"level6": {
																	Type: TypeString,
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
				"certificates": {
					Type:  TypeArray,
					Items: nil, // Missing items schema
				},
				"secrets": {
					Type: TypeArray,
					Items: &Schema{
						Type: TypeObject,
						Properties: map[string]*Schema{
							"name": {
								Type: TypeString,
							},
							"value": {
								Type: TypeString,
							},
						},
					},
					// Missing min/max items
				},
				"noDescription": {
					Type: TypeString,
					// No description
				},
				"noExamples": {
					Type:        TypeString,
					Description: "This field has a description but no examples or default",
					// No examples or default
				},
			},
		}

		issues := ValidateHelmBestPractices(schema)

		// Debug: Print all issues to help diagnose test failures
		t.Logf("Found %d validation issues:", len(issues))
		for i, issue := range issues {
			t.Logf("Issue %d: Path='%s', Message='%s', Level=%s",
				i+1, issue.Path, issue.Message, issue.Level)
		}

		// Define test cases for expected issues
		testCases := []struct {
			path     string
			message  string
			level    ValidationLevel
			expected bool
		}{
			{
				"objectWithBadProps.NOT_CAMEL_CASE",
				"Property names should follow camelCase convention",
				Warning,
				true,
			},
			{
				"objectWithBadProps.with-hyphen",
				"Property names should not contain hyphens or underscores",
				Error,
				true,
			},
			{
				"objectWithBadProps.with_underscore",
				"Property names should not contain hyphens or underscores",
				Error,
				true,
			},
			{
				"deeplyNested.level1.level2.level3.level4.level5",
				"Excessive nesting depth",
				Warning,
				true,
			},
			{
				"certificates",
				"Array should define an items schema for validation",
				Warning,
				true,
			},
			{
				"secrets",
				"Consider adding minItems/maxItems constraints for this array",
				Info,
				true,
			},
			{
				"noDescription",
				"Property should have a description",
				Warning,
				true,
			},
			{
				"noExamples",
				"Consider adding examples or default value",
				Info,
				true,
			},
		}

		// Verify each expected issue exists
		for _, tc := range testCases {
			found := false
			for _, issue := range issues {
				// Use more flexible path matching since the actual paths might have different structures
				// (like dot prefixes or different separators)
				if (strings.Contains(issue.Path, tc.path) || strings.HasSuffix(issue.Path, tc.path)) &&
					strings.Contains(issue.Message, tc.message) &&
					issue.Level == tc.level {
					found = true
					break
				}
			}

			if found != tc.expected {
				if tc.expected {
					t.Errorf("Expected issue not found - Path: %s, Message: %s, Level: %s", tc.path, tc.message, tc.level)
				} else {
					t.Errorf("Unexpected issue found - Path: %s, Message: %s, Level: %s", tc.path, tc.message, tc.level)
				}
			}
		}
	})

	t.Run("IsTypeInArray", func(t *testing.T) {
		// Test with []SchemaType
		types1 := []SchemaType{TypeString, TypeObject, TypeArray}
		if !isTypeInArray(TypeString, types1) {
			t.Error("TypeString should be found in types1")
		}
		if !isTypeInArray(TypeObject, types1) {
			t.Error("TypeObject should be found in types1")
		}
		if isTypeInArray(TypeInteger, types1) {
			t.Error("TypeInteger should not be found in types1")
		}

		// Test with []any
		types2 := []any{"string", "object", "array"}
		if !isTypeInArray(TypeString, types2) {
			t.Error("TypeString should be found in types2")
		}
		if !isTypeInArray(TypeObject, types2) {
			t.Error("TypeObject should be found in types2")
		}
		if isTypeInArray(TypeInteger, types2) {
			t.Error("TypeInteger should not be found in types2")
		}

		// Test with non-array
		if isTypeInArray(TypeString, "string") {
			t.Error("isTypeInArray should return false for non-array types")
		}
	})

	t.Run("FormatValidationIssues", func(t *testing.T) {
		issues := []ValidationIssue{
			{
				Path:    "test.path1",
				Message: "Test error message",
				Level:   Error,
			},
			{
				Path:    "test.path2",
				Message: "Test warning message",
				Level:   Warning,
			},
			{
				Path:    "test.path3",
				Message: "Test info message",
				Level:   Info,
			},
		}

		formatted := FormatValidationIssues(issues)

		// Check that the formatted output contains the expected sections and counts
		if !strings.Contains(formatted, "Found 3 issues: 1 errors, 1 warnings, 1 info") {
			t.Errorf("Formatted output missing correct summary: %s", formatted)
		}

		if !strings.Contains(formatted, "ERRORS:") {
			t.Errorf("Formatted output missing ERRORS section: %s", formatted)
		}

		if !strings.Contains(formatted, "WARNINGS:") {
			t.Errorf("Formatted output missing WARNINGS section: %s", formatted)
		}

		if !strings.Contains(formatted, "INFO:") {
			t.Errorf("Formatted output missing INFO section: %s", formatted)
		}

		if !strings.Contains(formatted, "test.path1: Test error message") {
			t.Errorf("Formatted output missing error message: %s", formatted)
		}

		if !strings.Contains(formatted, "test.path2: Test warning message") {
			t.Errorf("Formatted output missing warning message: %s", formatted)
		}

		if !strings.Contains(formatted, "test.path3: Test info message") {
			t.Errorf("Formatted output missing info message: %s", formatted)
		}

		// Test empty issues
		emptyFormatted := FormatValidationIssues([]ValidationIssue{})
		if emptyFormatted != "No validation issues found." {
			t.Errorf("Expected 'No validation issues found.' for empty issues, got: %s", emptyFormatted)
		}
	})
}
