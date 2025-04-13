package jsonschema

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
)

// CommentExtractor extracts comments from YAML files and associates them with paths
type CommentExtractor struct {
	// Map from YAML path to comment
	comments map[string]string
	// Debug mode - print more info
	Debug bool
}

// NewCommentExtractor creates a new comment extractor
func NewCommentExtractor() *CommentExtractor {
	return &CommentExtractor{
		comments: make(map[string]string),
		Debug:    false,
	}
}

// ExtractFromYAML parses a YAML file and extracts comments
func (e *CommentExtractor) ExtractFromYAML(yamlData []byte) {
	scanner := bufio.NewScanner(bytes.NewReader(yamlData))

	var indentToPath = make(map[int][]string) // Map indentation levels to path components
	var pendingComments string
	var topLevelComment string
	var lineIndents = []int{} // Track indentation levels for path management

	lineNum := 0
	foundFirstKey := false

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

			// Some Helm charts use a special syntax like "# -- This is a description"
			// to explicitly mark comments as descriptions
			comment = strings.TrimPrefix(comment, "-- ")
			comment = strings.TrimPrefix(comment, "--")

			// If we haven't found the first key yet, this might be a file-level comment
			if !foundFirstKey {
				if topLevelComment == "" {
					topLevelComment = comment
				} else {
					topLevelComment += "\n" + comment
				}
			}

			// Accumulate comment for the next field
			if pendingComments == "" {
				pendingComments = comment
			} else {
				pendingComments += "\n" + comment
			}
			continue
		}

		// If we have a key-value pair, process it
		if strings.Contains(line, ":") {
			foundFirstKey = true

			// Calculate indentation level
			indent := len(line) - len(strings.TrimLeft(line, " "))

			// Extract the key
			parts := strings.SplitN(strings.TrimSpace(line), ":", 2)
			key := strings.TrimSpace(parts[0])

			// Update the path based on indentation
			// First, find the correct parent level
			var parentLevel int = -1
			for i := len(lineIndents) - 1; i >= 0; i-- {
				if lineIndents[i] < indent {
					parentLevel = lineIndents[i]
					break
				}
			}

			// If we found a parent level, use its path as base
			var currentPath []string
			if parentLevel >= 0 {
				currentPath = append([]string{}, indentToPath[parentLevel]...)
			}

			// Add current key to path
			currentPath = append(currentPath, key)

			// Update indent tracking
			found := false
			for i, lvl := range lineIndents {
				if lvl == indent {
					found = true
					// Replace existing path at this level
					indentToPath[indent] = currentPath
					// Truncate indentation levels to remove deeper levels
					lineIndents = lineIndents[:i+1]
					break
				}
			}

			if !found {
				// New indentation level
				lineIndents = append(lineIndents, indent)
				indentToPath[indent] = currentPath
			}

			// Create dot-notation path
			pathStr := strings.Join(currentPath, ".")

			// If we have pending comments, associate them with this path
			if pendingComments != "" {
				if e.Debug {
					fmt.Fprintf(os.Stderr, "Associated comment with path %s: %s\n", pathStr, pendingComments)
				}
				e.comments[pathStr] = pendingComments
				pendingComments = ""
			}
		}
	}

	// Store the top-level comment as a special entry if we found one
	if topLevelComment != "" {
		e.comments[""] = topLevelComment
		if e.Debug {
			fmt.Fprintf(os.Stderr, "Found top-level comment: %s\n", topLevelComment)
		}
	}
}

// GetComment retrieves a comment for a given path
func (e *CommentExtractor) GetComment(path string) string {
	return e.comments[path]
}

// PrintAllComments prints all extracted comments to stderr for debugging
func (e *CommentExtractor) PrintAllComments() {
	fmt.Fprintf(os.Stderr, "=== Extracted Comments ===\n")
	for path, comment := range e.comments {
		fmt.Fprintf(os.Stderr, "Path: %s\nComment: %s\n\n", path, comment)
	}
	fmt.Fprintf(os.Stderr, "=== End of Comments ===\n")
}

// ApplyCommentsToSchema adds descriptions to schema based on YAML comments
func (e *CommentExtractor) ApplyCommentsToSchema(schema *Schema) {
	if schema == nil {
		return
	}

	if e.Debug {
		fmt.Fprintf(os.Stderr, "Looking for comments for path: %s\n", schema.HelmPath)
	}

	// Apply comment to current schema if applicable
	if comment, ok := e.comments[schema.HelmPath]; ok && schema.Description == "" {
		schema.Description = comment
		if e.Debug {
			fmt.Fprintf(os.Stderr, "Applied comment to %s\n", schema.HelmPath)
		}
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
