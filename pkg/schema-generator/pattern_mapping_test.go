package jsonschema

import (
	"testing"
)

func TestPatternMapping(t *testing.T) {
	t.Run("FieldsThatShouldSupportMultipleTypes", func(t *testing.T) {
		// Test cases for fields that should support multiple types
		tests := []struct {
			name          string
			path          string
			shouldMatch   bool
			expectedTypes []SchemaType
		}{
			{
				"Annotations field",
				"metadata.annotations",
				true,
				[]SchemaType{TypeObject, TypeString},
			},
			{
				"Labels field",
				"spec.labels",
				true,
				[]SchemaType{TypeObject, TypeString},
			},
			{
				"NodeSelector field",
				"nodeSelector",
				true,
				[]SchemaType{TypeObject, TypeString},
			},
			{
				"SecurityContext field",
				"pod.securityContext",
				true,
				[]SchemaType{TypeObject, TypeString},
			},
			{
				"Enabled field",
				"service.enabled",
				true,
				[]SchemaType{TypeBoolean, TypeString},
			},
			{
				"Enabled as exact match",
				"enabled",
				true,
				[]SchemaType{TypeBoolean, TypeString},
			},
			{
				"Autoscaling field",
				"deployment.autoscaling",
				true,
				[]SchemaType{TypeBoolean, TypeString},
			},
			{
				"TLS field",
				"ingress.tls",
				true,
				[]SchemaType{TypeBoolean, TypeString},
			},
			{
				"Tolerations field",
				"pod.tolerations",
				true,
				[]SchemaType{TypeNull, TypeArray, TypeString},
			},
			{
				"VolumeMounts field",
				"container.volumeMounts",
				true,
				[]SchemaType{TypeNull, TypeArray, TypeString},
			},
			{
				"Secret name field",
				"tls.secretName",
				true,
				[]SchemaType{TypeNull, TypeString},
			},
			{
				"Service port field",
				"service.port",
				true,
				[]SchemaType{TypeNull, TypeInteger},
			},
			{
				"CPU resource field",
				"resources.limits.cpu",
				true,
				[]SchemaType{TypeString, TypeInteger, TypeNumber},
			},
			{
				"Memory resource field",
				"resources.requests.memory",
				true,
				[]SchemaType{TypeString, TypeInteger, TypeNumber},
			},
			{
				"Config field",
				"app.config",
				true,
				[]SchemaType{TypeString, TypeObject},
			},
			{
				"Regular field should not match",
				"regular.field",
				false,
				nil,
			},
			{
				"Another regular field",
				"foo.bar.baz",
				false,
				nil,
			},
			{
				"JSON field",
				"customResource.json",
				true,
				[]SchemaType{TypeString, TypeObject, TypeArray},
			},
			{
				"Container port",
				"service.containerPort",
				true,
				[]SchemaType{TypeString, TypeObject, TypeArray},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				hasMultipleTypes, types := shouldSupportMultipleTypes(tc.path)

				if hasMultipleTypes != tc.shouldMatch {
					t.Errorf("shouldSupportMultipleTypes(%q) returned hasMultipleTypes=%v, want %v",
						tc.path, hasMultipleTypes, tc.shouldMatch)
				}

				if !tc.shouldMatch {
					if types != nil {
						t.Errorf("shouldSupportMultipleTypes(%q) returned non-nil types for a non-match: %v",
							tc.path, types)
					}
					return
				}

				if len(types) != len(tc.expectedTypes) {
					t.Errorf("shouldSupportMultipleTypes(%q) returned %d types, want %d",
						tc.path, len(types), len(tc.expectedTypes))
					return
				}

				// Check that all expected types are present
				for i, expectedType := range tc.expectedTypes {
					if types[i] != expectedType {
						t.Errorf("shouldSupportMultipleTypes(%q) returned types[%d]=%v, want %v",
							tc.path, i, types[i], expectedType)
					}
				}
			})
		}
	})

	t.Run("CaseInsensitivity", func(t *testing.T) {
		// Test that pattern matching is case-insensitive
		testCases := []struct {
			path          string
			lowerCasePath string
			shouldMatch   bool
			expectedTypes []SchemaType
		}{
			{
				"metadata.ANNOTATIONS",
				"metadata.annotations",
				true,
				[]SchemaType{TypeObject, TypeString},
			},
			{
				"pod.SecurityContext",
				"pod.securitycontext",
				true,
				[]SchemaType{TypeObject, TypeString},
			},
			{
				"service.ENABLED",
				"service.enabled",
				true,
				[]SchemaType{TypeBoolean, TypeString},
			},
			{
				"ENABLED",
				"enabled",
				true,
				[]SchemaType{TypeBoolean, TypeString},
			},
			{
				"pod.Tolerations",
				"pod.tolerations",
				true,
				[]SchemaType{TypeNull, TypeArray, TypeString},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.path, func(t *testing.T) {
				// Check the uppercase/mixed case path
				hasMultipleTypes, types := shouldSupportMultipleTypes(tc.path)

				if !hasMultipleTypes {
					t.Errorf("Case-insensitive test failed: shouldSupportMultipleTypes(%q) returned false", tc.path)
					return
				}

				// Check lowercase path for comparison
				hasMultipleTypesLower, typesLower := shouldSupportMultipleTypes(tc.lowerCasePath)

				if !hasMultipleTypesLower {
					t.Errorf("Case-insensitive reference test failed: shouldSupportMultipleTypes(%q) returned false",
						tc.lowerCasePath)
					return
				}

				// Both should match and have the same types
				if len(types) != len(typesLower) {
					t.Errorf("Case-insensitive test failed: different number of types between %q and %q",
						tc.path, tc.lowerCasePath)
					return
				}

				for i := range types {
					if types[i] != typesLower[i] {
						t.Errorf("Case-insensitive test failed: different types between %q and %q at index %d",
							tc.path, tc.lowerCasePath, i)
					}
				}
			})
		}
	})

	t.Run("MatchTypeVariations", func(t *testing.T) {
		// Test different match types (contains, suffix, exact)

		// "contains" match type
		containsPath := "deployment.podAnnotations.key"
		hasMultipleTypes, types := shouldSupportMultipleTypes(containsPath)
		if !hasMultipleTypes {
			t.Errorf("'contains' match failed for %q", containsPath)
		} else if types[0] != TypeObject || types[1] != TypeString {
			t.Errorf("'contains' match returned unexpected types for %q: %v", containsPath, types)
		}

		// "exact-or-suffix" match type (checking suffix case)
		suffixPath := "service.enabled"
		hasMultipleTypes, types = shouldSupportMultipleTypes(suffixPath)
		if !hasMultipleTypes {
			t.Errorf("'exact-or-suffix' (suffix case) match failed for %q", suffixPath)
		} else if types[0] != TypeBoolean || types[1] != TypeString {
			t.Errorf("'exact-or-suffix' (suffix case) match returned unexpected types for %q: %v", suffixPath, types)
		}

		// "exact-or-suffix" match type (checking exact case)
		exactPath := "enabled"
		hasMultipleTypes, types = shouldSupportMultipleTypes(exactPath)
		if !hasMultipleTypes {
			t.Errorf("'exact-or-suffix' (exact case) match failed for %q", exactPath)
		} else if types[0] != TypeBoolean || types[1] != TypeString {
			t.Errorf("'exact-or-suffix' (exact case) match returned unexpected types for %q: %v", exactPath, types)
		}
	})
}
