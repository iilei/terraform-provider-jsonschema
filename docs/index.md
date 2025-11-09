# JSON Schema Provider

Terraform provider for validating JSON and JSON5 documents using [JSON Schema](https://json-schema.org/) specifications.

> **Note:** Version 0.x is in initial development. Breaking changes may occur between releases per [semver](https://semver.org/#spec-item-4). Pin versions in production.

## Features

### Core Capabilities

- **JSON5 Support**: Parse and validate both JSON and JSON5 format documents and schemas
- **Multiple Schema Versions**: Support for JSON Schema Draft 4, 6, 7, 2019-09, and 2020-12
- **Consistent Output**: Deterministic JSON formatting for stable Terraform state

### Schema References

- **External Reference Resolution**: Resolves `$ref` URIs including JSON5 files relative to schema location
- **Reference Overrides** (`ref_overrides`): Redirect remote `$ref` URLs to local files for:
  - Offline validation (no internet required)
  - Air-gapped environments
  - Deterministic builds

### Error Handling

- **Enhanced Error Templating**: Go template system with individual error iteration capabilities
- **Consistent Error Ordering**: Deterministic error ordering for reliable testing and CI/CD
- **Flexible Formatting**: CI/CD integration, structured logging, and custom formats

## Provider Configuration

```hcl-terraform
provider "jsonschema" {
  schema_version = "draft/2020-12"  # Optional: JSON Schema version
  error_message_template = "{{.FullMessage}}"  # Optional: Go template for errors
}
```

### Configuration Arguments

- `schema_version` (Optional) - JSON Schema draft version. Defaults to `"draft/2020-12"`.
- `error_message_template` (Optional) - Go template for error messages. Available variables: `{{.SchemaFile}}`, `{{.Document}}`, `{{.FullMessage}}`, `{{.Errors}}`, `{{.ErrorCount}}`. Use `{{range .Errors}}` to iterate over individual errors.

## Basic Example

```hcl-terraform
provider "jsonschema" {
  schema_version = "draft/2020-12"
}

# Validate a JSON document
data "jsonschema_validator" "config" {
  document = file("${path.module}/config.json")
  schema   = "${path.module}/config.schema.json"
}

# Use the validated document
resource "helm_release" "app" {
  name   = "my-app"
  values = [data.jsonschema_validator.config.validated]
}
```

## JSON5 Support Example

```hcl-terraform
# Validate a JSON5 document with JSON5 schema
data "jsonschema_validator" "json5_config" {
  document = <<-EOT
    {
      // JSON5 comments supported
      "name": "my-service",
      "ports": [8080, 8081,], // Trailing commas allowed
      "features": {
        enabled: true,  // Unquoted keys supported
      }
    }
  EOT
  schema = "${path.module}/service.schema.json5"
}
```

## Advanced Configuration

```hcl-terraform
### Schema References

```hcl-terraform
# Schema references are resolved relative to schema file location
# For example, if schema is at "/path/to/schemas/main.schema.json"
# then "$ref": "./types.json" resolves to "/path/to/schemas/types.json"
data "jsonschema_validator" "with_refs" {
  document = file("document.json")
  schema   = "${path.module}/schemas/main.schema.json"  # Contains $ref references
}
```

### Reference Overrides (Offline Validation)

Redirect remote `$ref` URLs to local files for offline validation:

```hcl-terraform
data "jsonschema_validator" "api_request" {
  document = file("api-request.json")
  schema   = "${path.module}/schemas/api-request.schema.json"
  
  # Map remote URLs to local files
  ref_overrides = {
    "https://api.example.com/schemas/user.schema.json"    = "${path.module}/schemas/user.schema.json"
    "https://api.example.com/schemas/product.schema.json" = "${path.module}/schemas/product.schema.json"
  }
}
```

**Benefits:**
- No internet connection required
- Works in air-gapped/restricted networks  
- Deterministic builds
- No proxy settings, authentication, or TLS configuration needed
- No external dependencies in CI/CD

See the complete example in `examples/ref_overrides/` directory.

### Schema Version Overrides

```hcl-terraform
# Override schema version per validation
data "jsonschema_validator" "legacy_config" {
  document       = file("legacy-config.json")
  schema         = "legacy.schema.json"
  schema_version = "draft-04"  # Override provider default
}

# Custom error message template per validation
data "jsonschema_validator" "detailed_validation" {
  document = file("config.json")
  schema   = "config.schema.json"
  error_message_template = "Schema {{.SchemaFile}} failed: {{.FullMessage}}"
}

# Individual error iteration for multiple validation errors
data "jsonschema_validator" "individual_errors" {
  document = file("complex-config.json")
  schema   = "complex.schema.json"
  error_message_template = "{{range .Errors}}{{.DocumentPath}}: {{.Message}}\n{{end}}"
}
```

### Multiple Schema Validations

```hcl-terraform
data "jsonschema_validator" "api_validation" {
  document       = file("api-config.json")
  schema         = "${path.module}/schemas/openapi/config.schema.json"
  schema_version = "draft/2019-09"
}

data "jsonschema_validator" "legacy_validation" {
  document       = file("legacy-config.json") 
  schema         = "${path.module}/schemas/legacy/service.schema.json"
  schema_version = "draft-04"
}
```

## Error Message Templating

Customize validation error messages using Go templates:

```hcl-terraform
# Default full message
data "jsonschema_validator" "config" {
  document = file("config.json")
  schema   = "config.schema.json"
  error_message_template = "{{.FullMessage}}"
}

# Individual error iteration
data "jsonschema_validator" "individual_errors" {
  document = file("api-config.json")
  schema   = "api.schema.json"
  error_message_template = "{{range .Errors}}{{.DocumentPath}}: {{.Message}}\n{{end}}"
}

# Detailed format with error count
data "jsonschema_validator" "detailed_format" {
  document = file("config.json")
  schema   = "config.schema.json"
  error_message_template = <<-EOT
    Found {{.ErrorCount}} validation errors in {{.SchemaFile}}:
    {{range $i, $e := .Errors}}{{add $i 1}}. {{.DocumentPath}}: {{.Message}}
    {{end}}
  EOT
}

# CI/CD friendly format
provider "jsonschema" {
  error_message_template = "{{range .Errors}}::error file={{$.SchemaFile}}::{{.Message}}{{if .DocumentPath}} at {{.DocumentPath}}{{end}}\n{{end}}"
}

# JSON structured output
data "jsonschema_validator" "structured_errors" {
  document = file("config.json")
  schema   = "config.schema.json"
  error_message_template = <<-EOT
    {"validation_failed":true,"schema":"{{.SchemaFile}}","error_count":{{.ErrorCount}},"errors":[{{range $i,$e := .Errors}}{{if $i}},{{end}}{"documentPath":"{{.DocumentPath}}","message":"{{.Message}}"}{{end}}]}
  EOT
}
```
