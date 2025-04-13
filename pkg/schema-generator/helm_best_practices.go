package jsonschema

import (
	"fmt"
	"strings"
)

// ValidationLevel represents the severity of a validation issue
type ValidationLevel string

const (
	// Error represents a critical issue that should be fixed
	Error ValidationLevel = "error"
	// Warning represents a potential issue that should be reviewed
	Warning ValidationLevel = "warning"
	// Info represents useful information but not an issue
	Info ValidationLevel = "info"
)

// ValidationIssue represents a single validation issue found in the schema
type ValidationIssue struct {
	Path    string          // The path to the property with the issue
	Message string          // A description of the issue
	Level   ValidationLevel // The severity of the issue
}

// MaxNestingDepth is the recommended maximum nesting depth for Helm values
const MaxNestingDepth = 5

// ValidateHelmBestPractices checks the schema against Helm best practices
func ValidateHelmBestPractices(schema *Schema) []ValidationIssue {
	issues := []ValidationIssue{}

	// Start recursive validation from the root
	validateSchema(schema, "", 0, &issues)

	return issues
}

// validateSchema recursively validates a schema object and its children
func validateSchema(schema *Schema, path string, depth int, issues *[]ValidationIssue) {
	// Check naming conventions
	checkNamingConventions(schema, path, issues)

	// Check nesting depth
	checkNestingDepth(depth, path, issues)

	// Check array structures
	checkArrayStructures(schema, path, issues)

	// Check documentation
	checkDocumentation(schema, path, issues)

	// Recursively check properties if this is an object
	if schema.Properties != nil {
		for propName, propSchema := range schema.Properties {
			propPath := path
			if propPath == "" {
				propPath = propName
			} else {
				propPath = propPath + "." + propName
			}

			validateSchema(propSchema, propPath, depth+1, issues)
		}
	}

	// Recursively check array items if this is an array
	if schema.Type == TypeArray || (schema.Type != nil && isTypeInArray(TypeArray, schema.Type)) {
		if schema.Items != nil {
			itemPath := path + "[]"
			validateSchema(schema.Items, itemPath, depth+1, issues)
		}
	}
}

// isTypeInArray checks if a type is in an array of types
func isTypeInArray(typeToFind SchemaType, types any) bool {
	switch typesVal := types.(type) {
	case []SchemaType:
		for _, t := range typesVal {
			if t == typeToFind {
				return true
			}
		}
	case []any:
		for _, t := range typesVal {
			if str, ok := t.(string); ok && SchemaType(str) == typeToFind {
				return true
			}
		}
	}
	return false
}

// checkNamingConventions validates property names follow Helm conventions
func checkNamingConventions(schema *Schema, path string, issues *[]ValidationIssue) {
	if schema.Properties == nil || path == "" {
		return
	}

	for propName := range schema.Properties {
		// Check for camelCase naming convention
		if strings.ToLower(propName[:1]) != propName[:1] {
			*issues = append(*issues, ValidationIssue{
				Path:    path + "." + propName,
				Message: "Property names should follow camelCase convention",
				Level:   Warning,
			})
		}

		// Check for common naming issues
		if strings.Contains(propName, "-") || strings.Contains(propName, "_") {
			*issues = append(*issues, ValidationIssue{
				Path:    path + "." + propName,
				Message: "Property names should not contain hyphens or underscores",
				Level:   Error,
			})
		}
	}
}

// checkNestingDepth checks if the schema has excessive nesting
func checkNestingDepth(depth int, path string, issues *[]ValidationIssue) {
	if depth > MaxNestingDepth {
		*issues = append(*issues, ValidationIssue{
			Path:    path,
			Message: fmt.Sprintf("Excessive nesting depth (%d levels). Consider flattening the structure or using dot notation for paths.", depth),
			Level:   Warning,
		})
	}
}

// checkArrayStructures validates array structures
func checkArrayStructures(schema *Schema, path string, issues *[]ValidationIssue) {
	if schema.Type == TypeArray || (schema.Type != nil && isTypeInArray(TypeArray, schema.Type)) {
		// Check if items schema is defined
		if schema.Items == nil {
			*issues = append(*issues, ValidationIssue{
				Path:    path,
				Message: "Array should define an items schema for validation",
				Level:   Warning,
			})
		}

		// Check if minimum/maximum items are defined for arrays that should have constraints
		if path != "" &&
			(strings.Contains(path, "secret") ||
				strings.Contains(path, "config") ||
				strings.Contains(path, "certificate")) {

			if schema.MinItems == nil && schema.MaxItems == nil {
				*issues = append(*issues, ValidationIssue{
					Path:    path,
					Message: "Consider adding minItems/maxItems constraints for this array",
					Level:   Info,
				})
			}
		}
	}
}

// checkDocumentation validates schema documentation completeness
func checkDocumentation(schema *Schema, path string, issues *[]ValidationIssue) {
	// Skip root schema
	if path == "" {
		return
	}

	// Check if description is present
	if schema.Description == "" {
		// Higher severity for top-level properties
		level := Info
		if strings.Count(path, ".") <= 1 {
			level = Warning
		}

		*issues = append(*issues, ValidationIssue{
			Path:    path,
			Message: "Property should have a description",
			Level:   level,
		})
	}

	// Check for examples if this is a leaf property (not an object with properties)
	isLeaf := len(schema.Properties) == 0
	if isLeaf && schema.Examples == nil && schema.Default == nil &&
		schema.Type != TypeObject && schema.Type != TypeArray {
		*issues = append(*issues, ValidationIssue{
			Path:    path,
			Message: "Consider adding examples or default value",
			Level:   Info,
		})
	}
}

// FormatValidationIssues formats the validation issues into a readable string
func FormatValidationIssues(issues []ValidationIssue) string {
	if len(issues) == 0 {
		return "No validation issues found."
	}

	var errorCount, warningCount, infoCount int
	var result strings.Builder

	// Count issues by level
	for _, issue := range issues {
		switch issue.Level {
		case Error:
			errorCount++
		case Warning:
			warningCount++
		case Info:
			infoCount++
		}
	}

	// Write summary
	result.WriteString(fmt.Sprintf("Found %d issues: %d errors, %d warnings, %d info\n\n",
		len(issues), errorCount, warningCount, infoCount))

	// Write errors first
	if errorCount > 0 {
		result.WriteString("ERRORS:\n")
		for _, issue := range issues {
			if issue.Level == Error {
				result.WriteString(fmt.Sprintf("- %s: %s\n", issue.Path, issue.Message))
			}
		}
		result.WriteString("\n")
	}

	// Write warnings
	if warningCount > 0 {
		result.WriteString("WARNINGS:\n")
		for _, issue := range issues {
			if issue.Level == Warning {
				result.WriteString(fmt.Sprintf("- %s: %s\n", issue.Path, issue.Message))
			}
		}
		result.WriteString("\n")
	}

	// Write info
	if infoCount > 0 {
		result.WriteString("INFO:\n")
		for _, issue := range issues {
			if issue.Level == Info {
				result.WriteString(fmt.Sprintf("- %s: %s\n", issue.Path, issue.Message))
			}
		}
	}

	return result.String()
}
