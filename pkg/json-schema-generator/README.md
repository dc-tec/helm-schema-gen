# JSON Schema Generator

This package provides functionality for generating JSON Schema documents from YAML data, specifically tailored for Helm chart values.yaml files.

## Features

- Generate JSON Schema from YAML data
- Infer types from values (string, number, boolean, array, object)
- Extract descriptions from YAML comments
- Auto-detect common Helm patterns (image config, resources, etc.)
- Support for different JSON Schema draft versions
- Customizable schema generation options

## Usage

```go
package main

import (
	"fmt"
	"os"

	"github.com/yourusername/helm-schema-gen/pkg/jsonschema"
)

func main() {
	// Read a YAML file
	yamlData, err := os.ReadFile("values.yaml")
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	// Create a generator with default options
	generator := jsonschema.NewGeneratorWithDefaults()

	// Override some options
	generator.Options.Title = "My Helm Chart Schema"
	generator.Options.Description = "JSON Schema for my Helm chart values"
	generator.Options.ExtractDescriptions = true

	// Generate the schema
	schema, err := generator.GenerateFromYAML(yamlData)
	if err != nil {
		fmt.Printf("Error generating schema: %v\n", err)
		return
	}

	// Enhance the schema with Helm-specific optimizations
	optimizedSchema := generator.SpecializeSchemaForHelm(schema)

	// Print the schema
	fmt.Println(optimizedSchema)

	// Or save to a file
	f, err := os.Create("values.schema.json")
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		return
	}
	defer f.Close()

	_, err = f.WriteString(optimizedSchema.String())
	if err != nil {
		fmt.Printf("Error writing schema to file: %v\n", err)
		return
	}
}
```

## Configuration Options

The generator can be configured with various options:

| Option               | Description                         | Default              |
| -------------------- | ----------------------------------- | -------------------- |
| SchemaVersion        | JSON Schema version                 | draft-07             |
| Title                | Schema title                        | "Helm Values Schema" |
| Description          | Schema description                  | ""                   |
| RequireByDefault     | Make all properties required        | false                |
| IncludeExamples      | Include examples from values        | true                 |
| ExtractDescriptions  | Extract descriptions from comments  | true                 |
| UseFullyQualifiedIDs | Use fully qualified IDs for schemas | false                |

## Special Features

### Comment Extraction

The generator can extract descriptions from YAML comments:

```yaml
# This is a description for the replicaCount
replicaCount: 1
```

Will generate:

```json
{
  "properties": {
    "replicaCount": {
      "type": "integer",
      "description": "This is a description for the replicaCount"
    }
  }
}
```

### Pattern Detection

The generator automatically detects common Helm patterns:

1. **Container Images**:

   ```yaml
   image:
     repository: nginx
     tag: latest
   ```

2. **Resource Requirements**:
   ```yaml
   resources:
     limits:
       cpu: 100m
       memory: 128Mi
     requests:
       cpu: 50m
       memory: 64Mi
   ```

And enhances the schema with appropriate descriptions, validations, and more.
