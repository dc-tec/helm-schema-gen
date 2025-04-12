package jsonschema

import (
	"fmt"
)

func Example() {
	// Sample YAML content as a string with comments
	yamlContent := `
# Configuration for the container image
image:
  # Docker image repository
  repository: nginx
  # Docker image tag
  tag: 1.19.3
  # Kubernetes image pull policy
  pullPolicy: IfNotPresent

# Number of pod replicas
replicaCount: 1

# Resource requirements for the container
resources:
  # Resource limits for the container
  limits:
    # CPU resource limit
    cpu: 100m
    # Memory resource limit
    memory: 128Mi
  # Resource requests for the container
  requests:
    # CPU resource request
    cpu: 100m
    # Memory resource request
    memory: 128Mi

# Service configuration
service:
  # Kubernetes service type
  type: ClusterIP
  # Service port
  port: 80
  # Service annotations
  annotations:
    prometheus.io/scrape: "true"

# Ingress configuration
ingress:
  # Whether to create an Ingress resource
  enabled: false
  # Ingress annotations
  annotations: {}
  # Ingress hosts configuration
  hosts:
    - host: chart-example.local
      paths: ["/"]
  # TLS configuration
  tls: []
`

	// Create a generator with default options
	generator := NewGeneratorWithDefaults()

	// Override some options
	generator.Options.Title = "Sample Helm Chart Schema"
	generator.Options.Description = "JSON Schema for validating values.yaml for a sample Helm chart"
	generator.Options.ExtractDescriptions = true

	// Generate the schema from YAML
	schema, err := generator.GenerateFromYAML([]byte(yamlContent))
	if err != nil {
		fmt.Printf("Error generating schema: %v\n", err)
		return
	}

	// Enhance the schema with Helm-specific optimizations
	optimizedSchema := generator.SpecializeSchemaForHelm(schema)

	// Print the schema as JSON
	fmt.Println(optimizedSchema)
}
