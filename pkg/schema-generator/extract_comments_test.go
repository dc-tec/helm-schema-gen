package jsonschema

import (
	"strings"
	"testing"
)

func TestCommentExtraction(t *testing.T) {
	// Create a sample YAML with various comment patterns
	yamlData := `# Top level comment
# that spans multiple lines

# -- This is a special helm-style comment

key1: value1  # This is an inline comment that should be ignored

# Comment for key2
key2: value2

# Multi-line comment for key3
# with additional details
key3:
  # Comment for nested1
  nested1: value3
  # Comment for nested2
  nested2: value4

# Comment for key4 with special -- format
# -- This includes Helm-style comments
key4: value5

myObject:
  # -- First Helm comment
  # -- Second line of the comment
  property1: value
  
  # Regular comment line
  # Second line
  property2: value
`

	// Extract comments
	extractor := NewCommentExtractor()
	extractor.Debug = true
	extractor.ExtractFromYAML([]byte(yamlData))

	// Print all comments for debugging
	for path, comment := range extractor.comments {
		t.Logf("Path: %s\nComment: %s\n", path, comment)
	}

	// Verify we have the expected comments
	testCases := []struct {
		path           string
		expectedPrefix string // Just check the prefix since multiline comments can be tricky to match exactly
	}{
		{"", "Top level comment"},               // Top-level comment
		{"key2", "Comment for key2"},            // Simple single-line comment
		{"key3", "Multi-line comment for key3"}, // Multi-line comment
		{"key3.nested1", "Comment for nested1"}, // Nested comment
		{"key3.nested2", "Comment for nested2"}, // Another nested comment
		{"key4", "Comment for key4 with special -- format\nThis includes Helm-style comments"}, // Helm-style comment
		{"myObject.property1", "First Helm comment\nSecond line of the comment"},               // Helm comment with prefix removed
		{"myObject.property2", "Regular comment line\nSecond line"},                            // Regular multi-line comment
	}

	for _, tc := range testCases {
		comment, found := extractor.comments[tc.path]
		if !found {
			t.Errorf("Comment not found for path %q", tc.path)
			continue
		}

		if !strings.HasPrefix(comment, tc.expectedPrefix) {
			t.Errorf("Comment for path %q does not match expected prefix.\nGot: %q\nExpected prefix: %q",
				tc.path, comment, tc.expectedPrefix)
		}
	}
}
