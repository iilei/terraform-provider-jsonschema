# jsonschema_validator Data Source

The `jsonschema_validator` data source validates JSON, JSON5, YAML, and TOML documents using [JSON Schema](https://json-schema.org/) specifications with enhanced error templating capabilities.

## ⚠️ Breaking Changes in v0.6.0

### File Path API (Major Breaking Change)

**The `document` field now expects a file path instead of raw content.**

| Aspect | Old (v0.5.x) | New (v0.6.0+) |
|--------|--------------|---------------|
| `document` field | Raw content via `file()` function | File path (string) |
| Supported formats | JSON, JSON5 | JSON, JSON5, YAML, TOML |
| Format detection | N/A | Auto-detect from extension |
| Output field | `validated` | `valid_json` |

**Migration Required:**

```hcl
# Before (v0.5.x)
data "jsonschema_validator" "config" {
  document = file("${path.module}/config.json")  # ❌ Remove file() wrapper
  schema   = "${path.module}/config.schema.json"
}

# After (v0.6.0+)
data "jsonschema_validator" "config" {
  document = "${path.module}/config.json"  # ✅ Direct file path
  schema   = "${path.module}/config.schema.json"
}

# Access validated output
locals {
  config = jsondecode(data.jsonschema_validator.config.valid_json)  # ✅ Renamed from 'validated'
}
```

### New Features in v0.6.0

- ✅ **Multi-format support**: Validate YAML and TOML documents against JSON Schema
- ✅ **Auto-detection**: Format determined from file extension (`.json`, `.json5`, `.yaml`, `.yml`, `.toml`)
- ✅ **Force override**: Optional `force_filetype` field to override detection
- ✅ **Cleaner API**: No more `file()` wrapper needed
- ✅ **Better output**: `valid_json` clearly indicates JSON format output

**Template variable names have been clarified for better understanding:**

| Old Name (deprecated) | New Name | Description |
|----------------------|----------|-------------|
| `{{.Schema}}` | `{{.SchemaFile}}` | Path to the schema file |
| `{{.Path}}` | `{{.DocumentPath}}` | JSON Pointer to location in document |
| `{{.SchemaPath}}` | `{{.SchemaPath}}` | *(unchanged)* JSON Pointer to schema constraint |

### Scope of Changes

These variable renames **only affect** custom `error_message_template` configurations. If you're using the default error formatting or haven't customized templates, no action is needed.

### Migration Guide

Update all instances of `error_message_template` (in provider configuration or data source attributes):

**Before (v0.4.x):**
```hcl
data "jsonschema_validator" "example" {
  document = file("config.json")
  schema   = "config.schema.json"

  error_message_template = <<-EOT
    Schema: {{.Schema}}
    Location: {{.Path}}
    Schema Constraint: {{.SchemaPath}}
    Error: {{.Message}}
  EOT
}
```

**After (v0.5.0+):**
```hcl
data "jsonschema_validator" "example" {
  document = file("config.json")
  schema   = "config.schema.json"

  error_message_template = <<-EOT
    Schema: {{.SchemaFile}}
    Location: {{.DocumentPath}}
    Schema Constraint: {{.SchemaPath}}
    Error: {{.Message}}
  EOT
}
```

**Summary:**
- Replace `{{.Schema}}` → `{{.SchemaFile}}`
- Replace `{{.Path}}` → `{{.DocumentPath}}`
- All other variables (`{{.SchemaPath}}`, `{{.Message}}`, `{{.Value}}`, `{{.ErrorCount}}`, etc.) remain unchanged

## Example Usage

### Basic Validation (JSON)

```hcl-terraform
data "jsonschema_validator" "config" {
  document = "${path.module}/config.json"
  schema   = "${path.module}/config.schema.json"
}

# Access validated document as Terraform object
locals {
  config = jsondecode(data.jsonschema_validator.config.valid_json)
  app_name = local.config.application.name
}
```

### YAML Document Validation

```hcl-terraform
data "jsonschema_validator" "k8s_manifest" {
  document = "${path.module}/deployment.yaml"
  schema   = "${path.module}/k8s-schema.json"
}

# YAML is automatically detected from .yaml/.yml extension
locals {
  deployment = jsondecode(data.jsonschema_validator.k8s_manifest.valid_json)
  replicas   = local.deployment.spec.replicas
}
```

### TOML Configuration Validation

```hcl-terraform
data "jsonschema_validator" "app_config" {
  document = "${path.module}/config.toml"
  schema   = "${path.module}/config-schema.json"
}

# TOML is automatically detected from .toml extension
locals {
  config = jsondecode(data.jsonschema_validator.app_config.valid_json)
  port   = local.config.server.port
}
```

### JSON5 Document

```hcl-terraform
# Create a JSON5 document file (supports comments, trailing commas, unquoted keys)
resource "local_file" "json5_config" {
  filename = "${path.module}/config.json5"
  content  = <<-EOT
    {
      // JSON5 comments are supported
      "name": "example-service",
      "ports": [8080, 9090,], // Trailing commas allowed
      "config": {
        enabled: true,  // Unquoted keys supported
        timeout: 30_000 // Numeric separators supported
      }
    }
  EOT
}

data "jsonschema_validator" "json5_example" {
  document = local_file.json5_config.filename
  schema   = "${path.module}/config.schema.json"
}
```

### Force File Type Override

```hcl-terraform
# Validate a .txt file containing JSON
data "jsonschema_validator" "json_in_txt" {
  document       = "${path.module}/data.txt"
  schema         = "${path.module}/schema.json"
  force_filetype = "json"  # Override auto-detection
}

# Validate a file without extension
data "jsonschema_validator" "no_extension" {
  document       = "${path.module}/config"
  schema         = "${path.module}/schema.json"
  force_filetype = "yaml"  # Explicitly specify YAML
}

# Force JSON5 parsing for better error messages (trailing commas, comments, etc.)
data "jsonschema_validator" "relaxed_json" {
  document       = "${path.module}/config.json"
  schema         = "${path.module}/schema.json"
  force_filetype = "json5"  # Use JSON5 parser for relaxed syntax
}
```

### Schema Version Override

```hcl-terraform
data "jsonschema_validator" "legacy_validation" {
  document       = "${path.module}/legacy-data.json"
  schema         = "${path.module}/legacy.schema.json"
  schema_version = "draft-04"  # Override provider default
}
```

### Schema with References

```hcl-terraform
# Schema file with $ref references resolved relative to schema location
data "jsonschema_validator" "with_refs" {
  document = file("config.json")
  schema   = "${path.module}/schemas/main.schema.json"  # Contains $ref: "./types.json"
}
```

### Remote Schema References (ref_overrides)

```hcl-terraform
# Redirect remote schema URLs to local files for offline validation
data "jsonschema_validator" "with_remote_refs" {
  document = "${path.module}/api-request.json"
  schema   = "${path.module}/schemas/api-request.schema.json"

  # Map remote URLs to local files
  # When the schema contains $ref: "https://api.example.com/schemas/user.schema.json",
  # it will use the local file instead
  ref_overrides = {
    "https://api.example.com/schemas/user.schema.json"    = "${path.module}/schemas/user.schema.json"
    "https://api.example.com/schemas/product.schema.json" = "${path.module}/schemas/product.schema.json"
  }
}
```

### Custom Error Message Templates

```hcl-terraform
# Default full message
data "jsonschema_validator" "default_errors" {
  document = "${path.module}/config.json"
  schema   = "${path.module}/config.schema.json"
  error_message_template = "{{.FullMessage}}"
}

# Individual error iteration
data "jsonschema_validator" "individual_errors" {
  document = "${path.module}/config.yaml"
  schema   = "${path.module}/config.schema.json"
  error_message_template = "{{range .Errors}}{{.DocumentPath}}: {{.Message}}\n{{end}}"
}

# Custom format with error count
data "jsonschema_validator" "counted_errors" {
  document = "${path.module}/config.toml"
  schema   = "${path.module}/config.schema.json"
  error_message_template = "Found {{.ErrorCount}} errors:\n{{range .Errors}}• {{.DocumentPath}}: {{.Message}}\n{{end}}"
}

# CI/CD integration format
data "jsonschema_validator" "ci_errors" {
  document = "${path.module}/deployment.yaml"
  schema   = "${path.module}/deployment.schema.json"
  error_message_template = "{{range .Errors}}::error file={{$.SchemaFile}}::{{.Message}}{{if .DocumentPath}} at {{.DocumentPath}}{{end}}\n{{end}}"
}
```

### Advanced Error Templates

```hcl-terraform
# Detailed format with schema information
data "jsonschema_validator" "detailed_format" {
  document = "${path.module}/config.json"
  schema   = "${path.module}/config.schema.json"
  error_message_template = <<-EOT
    Schema: {{.SchemaFile}}
    Errors: {{.ErrorCount}}
    {{range .Errors}}• {{.DocumentPath}}: {{.Message}}
    {{end}}
  EOT
}

# JSON format for structured logging
data "jsonschema_validator" "json_format" {
  document = "${path.module}/config.yaml"
  schema   = "${path.module}/config.schema.json"
  error_message_template = jsonencode({
    "validation_failed": true,
    "schema": "{{.SchemaFile}}",
    "error_count": "{{.ErrorCount}}",
    "errors": "{{range $i, $e := .Errors}}{{if $i}},{{end}}{\"documentPath\":\"{{.DocumentPath}}\",\"message\":\"{{.Message}}\"}{{end}}"
  })
}
```

## Argument Reference

* `document` (Required) - **Path to document file** to validate. Supports JSON, JSON5, YAML, and TOML formats. Format is auto-detected from file extension (`.json`, `.json5`, `.yaml`, `.yml`, `.toml`).
* `schema` (Required) - Path to JSON or JSON5 schema file. Format auto-detected from extension.
* `force_filetype` (Optional) - Override automatic file type detection for the document. Valid values: `"json"`, `"json5"`, `"yaml"`, `"toml"`. Use when file extension doesn't match content format (e.g., `.txt` file containing YAML).
* `schema_version` (Optional) - Schema version override (`"draft-04"` to `"draft/2020-12"`).
* `error_message_template` (Optional) - Custom Go template for error messages. Available variables: `{{.SchemaFile}}`, `{{.Document}}`, `{{.FullMessage}}`, `{{.Errors}}`, `{{.ErrorCount}}`.
* `ref_overrides` (Optional) - Map of remote schema URLs to local file paths. Redirects `$ref` references from remote URLs to local files, enabling offline validation.

## Attributes Reference

* `valid_json` - The validated document in canonical JSON format. Only set when validation succeeds. Contains the document parsed, validated, and re-serialized as standard JSON with resolved `$ref` references. Use `jsondecode()` to access as Terraform objects.

## File Format Support

The provider automatically detects document format from file extension:

| Extension | Format | Parser |
|-----------|--------|--------|
| `.json` | JSON | JSON5 (backward compatible) |
| `.json5` | JSON5 | JSON5 (comments, trailing commas, unquoted keys) |
| `.yaml`, `.yml` | YAML | YAML 1.2 (superset of JSON) |
| `.toml` | TOML | TOML v1.0.0 |
| Other | JSON5 | Fallback to JSON5 for unknown extensions |

**Format Override**: Use `force_filetype` to override detection when:
- File has no extension or wrong extension
- You want to use JSON5 parser for relaxed JSON syntax
- Testing different format parsers

**Schema Format**: Schemas are always JSON/JSON5 and auto-detected from extension. YAML/TOML are converted to JSON internally before validation.

## Schema File Resolution

- **Local files**: Schema paths are resolved relative to the Terraform configuration directory
- **Schema references**: `$ref` URIs in schemas are resolved relative to the schema file's location
- **Relative references**: For example, if your schema is at `./schemas/main.schema.json` and contains `"$ref": "./types.json"`, it resolves to `./schemas/types.json`
- **Absolute references**: Full file paths or URLs in `$ref` are used as-is
- **Remote references with overrides**: When `ref_overrides` is configured, `$ref` URLs matching the map keys are redirected to local files

## Reference Overrides (ref_overrides)

The `ref_overrides` parameter allows you to redirect remote schema URLs to local files, enabling:

- **Offline validation**: No internet connection required
- **Air-gapped environments**: Works in restricted networks
- **Version control**: Keep all schemas in your repository
- **Deterministic builds**: Same inputs always produce same results
- **Performance**: No network latency
- **Security**: No external dependencies or credential management

### How It Works

When the schema compiler encounters a `$ref` to a URL:
1. First checks if the URL exists in `ref_overrides` (uses local file if found)
2. If not found, uses the default loader (file:// URLs)
3. Results are cached for subsequent references

This means you can mix remote refs (overridden to local) with local file:// refs in the same schema.

### Example Schema with Remote References

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "user": {
      "$ref": "https://api.example.com/schemas/user.schema.json"
    }
  }
}
```

With `ref_overrides`:

```hcl
ref_overrides = {
  "https://api.example.com/schemas/user.schema.json" = "./local/user.schema.json"
}
```

The `$ref` will resolve to the local file instead of attempting to fetch from the remote URL.

For a complete example, see `examples/ref_overrides/` in the provider repository.

## JSON5 Features Supported
## Document Format Support

Documents and schemas support multiple formats:

### JSON5 Features

Both document and schema files with `.json` or `.json5` extensions support:

- **Comments**: `// single-line` and `/* multi-line */` comments
- **Trailing commas**: Arrays and objects can have trailing commas
- **Unquoted keys**: Object keys don't require quotes (when they're valid identifiers)
- **Single quotes**: Strings can use single quotes
- **Multi-line strings**: Strings can span multiple lines
- **Numeric literals**: Hexadecimal numbers, leading/trailing decimal points, numeric separators

### YAML Features

YAML documents (`.yaml`, `.yml`) support all YAML 1.2 features:

- **Comments**: `# comment syntax`
- **Anchors and aliases**: `&anchor` and `*alias` for reference reuse
- **Multi-line strings**: Block scalars (`|` and `>`)
- **Native types**: Booleans, numbers, null, dates
- **Superset of JSON**: All JSON documents are valid YAML

### TOML Features

TOML documents (`.toml`) support all TOML v1.0.0 features:

- **Comments**: `# comment syntax`
- **Tables**: `[section]` for configuration sections
- **Inline tables**: `{key = "value", another = 123}`
- **Arrays**: `items = ["one", "two", "three"]`
- **Dates and times**: Native datetime support

## Error Templating

### Template Variables

Available in `error_message_template`:

- `{{.FullMessage}}` - Complete validation error message from jsonschema library
- `{{.ErrorCount}}` - Number of individual validation errors
- `{{.Errors}}` - Array of individual validation errors (for iteration)
- `{{.Document}}` - The document content (truncated if long)
- `{{.SchemaFile}}` - Path to the schema file

### Individual Error Details

Each error in `{{.Errors}}` contains:

- `{{.Message}}` - Human-readable error message
- `{{.DocumentPath}}` - JSON Pointer ([RFC 6901](https://datatracker.ietf.org/doc/html/rfc6901)) to the error location in the document (e.g., `/user/email`, `/items/0`)
- `{{.SchemaPath}}` - Full URI with JSON Pointer fragment to the failing constraint (e.g., `file:///path/to/schema.json#/properties/email/type`)
- `{{.Value}}` - The actual value that failed validation (if available)

**About Paths:**

- **DocumentPath**: Uses [JSON Pointer](https://datatracker.ietf.org/doc/html/rfc6901) syntax (RFC 6901). Empty string `""` represents the root of the document.
  - Examples: `""` (root), `/port` (top-level property), `/users/0/email` (nested array element)

- **SchemaPath**: Contains the full resolved `file://` URI plus JSON Pointer fragment to the exact constraint that failed.
  - Format: `file:///absolute/path/to/schema.json#/properties/port/type`
  - Always shows the **actual file location**, not `$id` values
  - The fragment (after `#`) is a JSON Pointer to the constraint within that schema file
  - Useful for debugging: tells you exactly which file and which constraint failed

**About JSON Pointer:** Path values use [JSON Pointer](https://datatracker.ietf.org/doc/html/rfc6901) syntax, the standard format used by JSON Schema validators. Per RFC 6901:
- Empty string `""` represents the whole document (root)
- Paths starting with `/` reference nested fields (e.g., `/email` for top-level `email` field)
- Array indices are numbers (e.g., `/items/0` for the first item)
- Special characters: `~` encodes as `~0`, `/` encodes as `~1` within field names

**Quick Reference:**

```hcl
# Access all error attributes
{{range .Errors}}
  {{.Message}}       # "at '/email': got number, want string"
  {{.DocumentPath}}  # "/email" (JSON Pointer to document location)
  {{.SchemaPath}}    # "schema.json#/properties/email/type" (JSON Pointer to schema constraint)
  {{.Value}}         # 12345
{{end}}
```

**See also:** Complete working example in `examples/error_attributes/` showing all attributes with realistic validation errors.

### Template Examples

```hcl-terraform
# Simple full message (default)
error_message_template = "{{.FullMessage}}"

# Individual error iteration
error_message_template = "{{range .Errors}}{{.DocumentPath}}: {{.Message}}\n{{end}}"

# Numbered list format (one-based)
error_message_template = "{{.ErrorCount}} validation errors:\n{{range $i, $e := .Errors}}{{add $i 1}}. {{.Message}} (at {{.DocumentPath}})\n{{end}}"

# CI/CD format
error_message_template = "{{range .Errors}}::error file={{$.SchemaFile}}::{{.Message}}{{if .DocumentPath}} at {{.DocumentPath}}{{end}}\n{{end}}"
```

### Template Functions

The following template function is available:

- `add` - Add two integers: `{{add $i 1}}` (useful for one-based indexing)

Templates support standard Go template syntax for formatting error messages.
