// Package patternmapping provides functionality for generating JSON Schema from Go values.
package jsonschema

import "strings"

// patternMapping defines a mapping between path patterns and corresponding schema types
type patternMapping struct {
	// patterns are the strings to look for in the path
	patterns []string
	// matchType determines how to match (contains, suffix, exact)
	matchType string
	// types are the schema types to return when a match is found
	types []SchemaType
}

// shouldSupportMultipleTypes checks if a field at the given path should support multiple types
// based on common patterns in Helm chart values.yaml files
func shouldSupportMultipleTypes(path string) (bool, []SchemaType) {
	pathLower := strings.ToLower(path)

	// Define pattern mappings - each entry defines patterns and corresponding types
	patternMappings := []patternMapping{
		{
			// Fields that should support object or string types (typically YAML or JSON strings)
			patterns: []string{
				"annotations", "labels", "nodeselector", "securitycontext",
				"affinity", "strategy", "networkpolicy", "objectselector",
				"poddisruptionbudget", "hostaliases", "matchlabels",
				"nodeaffinity", "podaffinity", "podantiaffinity", "selector",
				"topology", "rules", "expressions", "rollingupdate",
			},
			matchType: "contains",
			types:     []SchemaType{TypeObject, TypeString},
		},
		{
			// Fields that should support string or boolean types
			patterns: []string{
				"autoscaling", "forceupgrade", "createnamespace", "autosync",
				"persistence", "tls", "auth", "hostnetwork", "hostpid", "hostipc",
				"singlenamespace", "debug", "rbac", "monitoring", "istio",
				"serviceaccount", "automounttoken", "priorityclass", "metrics",
				"tracing",
			},
			matchType: "contains",
			types:     []SchemaType{TypeBoolean, TypeString},
		},
		{
			// Special case for enabled fields
			patterns:  []string{"enabled"},
			matchType: "exact-or-suffix",
			types:     []SchemaType{TypeBoolean, TypeString},
		},
		{
			// Fields that should support null, array, or string types
			patterns: []string{
				"tolerations", "topologyspreadconstraints", "volumes",
				"initcontainers", "extracontainers", "volumemounts",
				"imagepullsecrets", "hostalias", "sidecars", "extravolumes",
				"extrainitcontainers", "envfrom", "args", "command", "ports",
				"env", "environment", "secrets", "configmaps", "pods",
				"endpoints", "tls.hosts", "ingress.hosts", "hostAliases",
				"deploymentannotations", "podsecuritycontext", "permissions",
			},
			matchType: "contains",
			types:     []SchemaType{TypeNull, TypeArray, TypeString},
		},
		{
			// Fields that should support null and string
			patterns: []string{
				"secretname", "storageclass", "servicenodeport", "priorityclassname",
				"certname", "keyname", "cabundle", "ingressclassname", "authsecret",
				"namespace", "finalizer", "servicename", "clusterrole", "role",
				"healthcheckpath", "mountpath", "filename", "secretkey", "timezone",
				"bootstrapservers", "topic",
			},
			matchType: "contains",
			types:     []SchemaType{TypeNull, TypeString},
		},
		{
			// Fields that should support null and integer
			patterns: []string{
				"maxunavailable", "nodeport", "replicacount", "replicas",
				"port", "targetport", "containerport", "serviceport", "metricsport",
				"healthport", "readinessport", "maxreplicas", "minreplicas",
				"terminationgraceperiodseconds", "backofflimit", "failurethreshold",
				"successthreshold", "initialdelayseconds", "timeoutseconds",
				"periodseconds", "minavailable", "retention", "timeout", "limit",
				"weight",
			},
			matchType: "contains",
			types:     []SchemaType{TypeNull, TypeInteger},
		},
		{
			// Fields that should support string and object (config blocks)
			patterns: []string{
				"config", "extraenv", "extraenvironmentvars", "extravolumeconfig",
				"configuration", "settings", "options", "parameters", "properties",
				"authentication", "authorization", "security", "networking",
				"customvalues", "extraconfigs",
			},
			matchType: "contains",
			types:     []SchemaType{TypeString, TypeObject},
		},
		{
			// Fields that should support multiple numeric types
			patterns: []string{
				"resources.limits.memory", "resources.requests.memory", "memory",
				"resources.limits.cpu", "resources.requests.cpu", "cpu",
				"resources.limits", "resources.requests",
				"threshold", "percentage", "ratio", "factor", "scalar", "weight",
				"scale", "bytes", "size", "quota", "maxsurge", "minavailable",
				"retention",
			},
			matchType: "contains",
			types:     []SchemaType{TypeString, TypeInteger, TypeNumber},
		},
		{
			// Fields that should support string, integer, and boolean
			patterns: []string{
				"preference", "mode", "state", "status", "level", "type",
				"policy", "protocol",
			},
			matchType: "contains",
			types:     []SchemaType{TypeString, TypeInteger, TypeBoolean},
		},
		{
			// Fields that likely contain JSON
			patterns: []string{
				"json", "raw", "patch", "template", "customdata", "extradata",
				"override", "manifest",
			},
			matchType: "contains",
			types:     []SchemaType{TypeString, TypeObject, TypeArray},
		},
		{
			// Kubernetes API specific fields
			patterns: []string{
				"containerport", "servicetype", "ingresstype", "secrettype",
				"podannotations", "accessmodes", "pathtype", "readinessprobe",
				"livenessprobe", "startupprobe", "volumesource", "volumetype",
				"service.containerport",
			},
			matchType: "contains",
			types:     []SchemaType{TypeString, TypeObject, TypeArray},
		},
	}

	// Check each pattern mapping to see if path matches
	for _, mapping := range patternMappings {
		for _, pattern := range mapping.patterns {
			isMatch := false

			switch mapping.matchType {
			case "contains":
				isMatch = strings.Contains(pathLower, pattern)
			case "suffix":
				isMatch = strings.HasSuffix(pathLower, pattern)
			case "exact":
				isMatch = pathLower == pattern
			case "exact-or-suffix":
				isMatch = pathLower == pattern || strings.HasSuffix(pathLower, "."+pattern)
			}

			if isMatch {
				return true, mapping.types
			}
		}
	}

	// If not a special case, it doesn't need multiple types
	return false, nil
}
