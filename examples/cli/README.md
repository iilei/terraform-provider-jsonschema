# CLI Tool Examples

This directory contains example configurations for the `jsonschema-validator` CLI tool.

## Files

- **`.jsonschema-validator.yaml`** - Standalone configuration file (recommended)
- **`pyproject.toml.example`** - Configuration for Python projects
- **`package.json.example`** - Configuration for Node.js projects
- **`.pre-commit-config.yaml.example`** - Pre-commit hook configurations

## Quick Start

### 1. Using Standalone Config File

Copy `.jsonschema-validator.yaml` to your project root:

```bash
cp .jsonschema-validator.yaml /path/to/your/project/
cd /path/to/your/project
jsonschema-validator  # Automatically discovers config
```

### 2. Python Project (pyproject.toml)

Add the configuration section from `pyproject.toml.example` to your `pyproject.toml`:

```toml
[tool.jsonschema-validator]
schema_version = "draft/2020-12"

[[tool.jsonschema-validator.schemas]]
path = "config.schema.json"
documents = ["config.json"]
```

Then run:

```bash
jsonschema-validator  # Reads from pyproject.toml
```

### 3. Node.js Project (package.json)

Add the configuration from `package.json.example` to your `package.json`:

```json
{
  "jsonschema-validator": {
    "schemaVersion": "draft/2020-12",
    "schemas": [
      {"path": "config.schema.json", "documents": ["config.json"]}
    ]
  }
}
```

Then run:

```bash
jsonschema-validator  # Reads from package.json
```

### 4. Pre-commit Hook

Add configuration from `.pre-commit-config.yaml.example` to your `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/iilei/terraform-provider-jsonschema
    rev: v0.5.0
    hooks:
      - id: jsonschema-validator
```

Install and run:

```bash
pre-commit install
pre-commit run jsonschema-validator --all-files
```

## Configuration Options

All configuration options match the Terraform provider for consistency:

### Schema Version

Specify the JSON Schema draft version:

```yaml
schema_version: "draft/2020-12"  # or draft/2019-09, draft-7, draft-6, draft-4
```

### Schema-Document Mappings

Define which schemas validate which documents:

```yaml
schemas:
  - path: "config.schema.json"
    documents:
      - "config.json"          # Single file
      - "config.*.json"        # Glob pattern
      - "configs/**/*.json"    # Recursive glob
```

### Reference Overrides

Redirect remote `$ref` URLs to local files:

```yaml
schemas:
  - path: "api.schema.json"
    documents: ["api/request.json"]
    ref_overrides:
      "https://example.com/user.json": "./schemas/user.json"
```

**Use cases:**
- Offline validation (no internet required)
- Air-gapped environments
- Version-controlled schemas
- Faster validation (no HTTP requests)

### Custom Error Templates

Format validation errors using Go templates:

```yaml
error_template: |
  {{range .Errors}}
  {{.DocumentPath}}: {{.Message}}
  {{end}}
```

**Available variables:**
- `{{.FullMessage}}` - Complete formatted error
- `{{.ErrorCount}}` - Number of errors
- `{{.Errors}}` - Array of errors
  - `{{.DocumentPath}}` - JSON path to error
  - `{{.Message}}` - Error message
  - `{{.Value}}` - Invalid value
- `{{.SchemaFile}}` - Schema file path
- `{{.Document}}` - Document content

## Command-Line Usage

All configurations can be overridden via command line:

```bash
# Override schema version
jsonschema-validator --schema-version "draft/2019-09"

# Specify schema explicitly
jsonschema-validator --schema config.schema.json config.json

# Add reference overrides
jsonschema-validator \
  --schema api.schema.json \
  --ref-override "https://example.com/user.json=./user.json" \
  request.json

# Custom error template
jsonschema-validator \
  --error-template '{{range .Errors}}{{.Message}}{{end}}' \
  --schema config.schema.json \
  config.json
```

## Environment Variables

All options can be set via environment variables:

```bash
export JSONSCHEMA_VALIDATOR_SCHEMA_VERSION="draft/2020-12"
export JSONSCHEMA_VALIDATOR_SCHEMA="config.schema.json"
jsonschema-validator config.json
```

## Configuration Priority

The tool merges configuration from multiple sources (highest to lowest priority):

1. Command-line flags
2. Environment variables (`JSONSCHEMA_VALIDATOR_*`)
3. `.jsonschema-validator.yaml` in current directory
4. `pyproject.toml` section `[tool.jsonschema-validator]`
5. `package.json` field `"jsonschema-validator"`
6. `~/.jsonschema-validator.yaml` in home directory

## JSON5 Support

The validator supports JSON5 for both documents and schemas:

```yaml
schemas:
  - path: "app.schema.json5"
    documents:
      - "app.json5"
      - "configs/*.json5"
```

**JSON5 features supported:**
- Comments (`//` and `/* */`)
- Trailing commas
- Unquoted keys
- Single-quoted strings
- Multi-line strings
- And more...

## CI/CD Integration

### GitHub Actions

```yaml
- name: Validate JSON files
  run: |
    go install github.com/iilei/terraform-provider-jsonschema/cmd/jsonschema-validator@latest
    jsonschema-validator
```

### GitLab CI

```yaml
validate-json:
  script:
    - go install github.com/iilei/terraform-provider-jsonschema/cmd/jsonschema-validator@latest
    - jsonschema-validator
```

## Exit Codes

- `0` - All validations passed
- `1` - Validation errors found
- `2` - Configuration or usage errors

## More Information

See the [main CLI README](../../cmd/jsonschema-validator/README.md) for complete documentation.
