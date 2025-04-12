package jsonschema

import (
	"bufio"
	"bytes"
	"strings"
)

// CommentExtractor extracts comments from YAML files and associates them with paths
type CommentExtractor struct {
	// Map from YAML path to comment
	comments map[string]string
}

// NewCommentExtractor creates a new comment extractor
func NewCommentExtractor() *CommentExtractor {
	return &CommentExtractor{
		comments: make(map[string]string),
	}
}

// ExtractFromYAML parses a YAML file and extracts comments
func (e *CommentExtractor) ExtractFromYAML(yamlData []byte) {
	scanner := bufio.NewScanner(bytes.NewReader(yamlData))

	var currentPath []string
	var currentIndent int
	var lastComment string

	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check if this is a comment line
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			comment := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "#"))

			// Accumulate multi-line comments
			if lastComment != "" {
				lastComment += " " + comment
			} else {
				lastComment = comment
			}
			continue
		}

		// If we have a key-value pair, process it
		if strings.Contains(line, ":") {
			// Calculate indentation level
			indent := len(line) - len(strings.TrimLeft(line, " "))

			// If we're at a shallower indentation, pop elements from path
			if indent < currentIndent {
				// Calculate how many levels we need to pop
				levelsToRemove := (currentIndent - indent) / 2
				if len(currentPath) >= levelsToRemove {
					currentPath = currentPath[:len(currentPath)-levelsToRemove]
				}
			}

			// Extract the key
			parts := strings.SplitN(strings.TrimSpace(line), ":", 2)
			key := strings.TrimSpace(parts[0])

			// Push key to path
			if indent > currentIndent || indent == 0 {
				currentPath = append(currentPath, key)
			} else {
				// Replace the last element
				if len(currentPath) > 0 {
					currentPath[len(currentPath)-1] = key
				} else {
					currentPath = []string{key}
				}
			}

			// Create dot-notation path
			path := strings.Join(currentPath, ".")

			// If we have a comment, associate it with this path
			if lastComment != "" {
				e.comments[path] = lastComment
				lastComment = ""
			}

			// Update current indent
			currentIndent = indent
		}
	}
}

// GetComment retrieves a comment for a given path
func (e *CommentExtractor) GetComment(path string) string {
	return e.comments[path]
}

// ApplyCommentsToSchema adds descriptions to schema based on YAML comments
func (e *CommentExtractor) ApplyCommentsToSchema(schema *Schema) {
	if schema == nil {
		return
	}

	// Apply comment to current schema if applicable
	if comment, ok := e.comments[schema.HelmPath]; ok && schema.Description == "" {
		schema.Description = comment
	}

	// Recursively apply to properties
	if schema.Properties != nil {
		for _, propSchema := range schema.Properties {
			e.ApplyCommentsToSchema(propSchema)
		}
	}

	// Apply to array items
	if schema.Items != nil {
		e.ApplyCommentsToSchema(schema.Items)
	}
}
