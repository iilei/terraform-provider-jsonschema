# jsonschema-validator CLI

Standalone JSON/JSON5 schema validator with support for all JSON Schema drafts.

## Features

- ✅ **JSON5 Support** - Native support for JSON5 documents AND schemas (unique!)
- ✅ **All Schema Drafts** - Draft 4, 6, 7, 2019-09, 2020-12
- ✅ **Reference Overrides** - Redirect remote `$ref` URLs to local files
- ✅ **Zero-config** - Works without configuration files
- ✅ **Multi-source Config** - Discovers `.jsonschema-validator.yaml`, `pyproject.toml`, `package.json`
- ✅ **Batch Validation** - Validate multiple files with glob patterns
- ✅ **Custom Error Templates** - Go template-based error formatting
- ✅ **CI/CD Ready** - Proper exit codes for automation
- ✅ **Pre-commit Integration** - Native pre-commit hook support

## Installation

### Via Go

```bash
go install github.com/iilei/terraform-provider-jsonschema/cmd/jsonschema-validator@latest
```

### Via Release Binary

Download from [GitHub Releases](https://github.com/iilei/terraform-provider-jsonschema/releases)


## Quick Start

### Simple Validation

```bash
# Validate a single file
jsonschema-validator --schema config.schema.json config.json

# Validate multiple files
jsonschema-validator --schema api.schema.json request1.json request2.json

# JSON5 support (documents AND schemas)
jsonschema-validator --schema app.schema.json5 app.json5
```

### With Configuration File

Create `.jsonschema-validator.yaml`:

```yaml
schema_version: "draft/2020-12"

schemas:
  - path: "config.schema.json"
    documents:
      - "config.json"
      - "config.*.json"
```

Then simply run:

```bash
jsonschema-validator  # Automatically discovers and uses config
```

## Configuration

### Configuration Discovery

The CLI discovers configuration from multiple sources (priority order):

1. **Command-line flags** (highest priority)
2. **Environment variables** (`JSONSCHEMA_VALIDATOR_*`)
3. `.jsonschema-validator.yaml` in current directory
4. `pyproject.toml` section `[tool.jsonschema-validator]`
5. `package.json` field `"jsonschema-validator"`
6. User home `~/.jsonschema-validator.yaml`

### Configuration File Formats

#### `.jsonschema-validator.yaml` (Recommended)

```yaml
# Schema version (draft 4, 6, 7, 2019-09, 2020-12)
schema_version: "draft/2020-12"

# Multiple schema-document mappings
schemas:
  - path: "config.schema.json"
    documents:
      - "config.json"
      - "config.*.json"  # Glob patterns supported

  - path: "api/schemas/request.schema.json"
    documents:
      - "api/requests/*.json"
    # Reference overrides for offline validation
    ref_overrides:
      "https://example.com/user.json": "./schemas/user.json"
      "https://example.com/product.json": "./schemas/product.json"

# Custom error template (Go templates)
error_template: |
  Validation failed with {{.ErrorCount}} error(s):
  {{range .Errors}}
  - {{.DocumentPath}}: {{.Message}}
  {{end}}
```

#### `pyproject.toml` (Python Projects)

```toml
[tool.jsonschema-validator]
schema_version = "draft/2020-12"

[[tool.jsonschema-validator.schemas]]
path = "config.schema.json"
documents = ["config.json", "config.*.json"]

[[tool.jsonschema-validator.schemas]]
path = "api/request.schema.json"
documents = ["api/requests/*.json"]

[tool.jsonschema-validator.schemas.ref_overrides]
"https://example.com/user.json" = "./schemas/user.json"
```

#### `package.json` (Node.js Projects)

```json
{
  "name": "my-project",
  "jsonschema-validator": {
    "schemaVersion": "draft/2020-12",
    "schemas": [
      {
        "path": "config.schema.json",
        "documents": ["config.json"]
      },
      {
        "path": "api/request.schema.json",
        "documents": ["api/requests/*.json"],
        "refOverrides": {
          "https://example.com/user.json": "./schemas/user.json"
        }
      }
    ]
  }
}
```

## Usage Examples

### Basic Validation

```bash
# Single file
jsonschema-validator --schema config.schema.json config.json

# Multiple files
jsonschema-validator --schema api.schema.json req1.json req2.json

# JSON5 support
jsonschema-validator --schema app.schema.json5 app.json5

# Validate from stdin
cat config.json | jsonschema-validator --schema config.schema.json -
```

### With Configuration File

```bash
# Uses .jsonschema-validator.yaml automatically
jsonschema-validator

# Explicit config file
jsonschema-validator --config custom-config.yaml

# Override schema version from config
jsonschema-validator --schema-version "draft/2019-09"
```

### Advanced Options

```bash
# Specify schema draft version
jsonschema-validator \
  --schema-version "draft/2020-12" \
  --schema config.schema.json \
  config.json

# Reference overrides (for offline validation)
jsonschema-validator \
  --schema api.schema.json \
  --ref-override "https://example.com/user.json=./local/user.json" \
  --ref-override "https://example.com/product.json=./local/product.json" \
  request.json

# Custom error template
jsonschema-validator \
  --schema config.schema.json \
  --error-template '{{range .Errors}}{{.DocumentPath}}: {{.Message}}{{end}}' \
  config.json

# JSON output format (for parsing)
jsonschema-validator --format json --schema config.schema.json config.json

# Quiet mode (only errors)
jsonschema-validator --quiet --schema config.schema.json config.json

# Verbose mode (detailed output)
jsonschema-validator --verbose --schema config.schema.json config.json
```

### Environment Variables

```bash
export JSONSCHEMA_VALIDATOR_SCHEMA_VERSION="draft/2020-12"
export JSONSCHEMA_VALIDATOR_SCHEMA="config.schema.json"
jsonschema-validator config.json

# All config options can be set via env vars:
export JSONSCHEMA_VALIDATOR_ERROR_TEMPLATE="..."
export JSONSCHEMA_VALIDATOR_REF_OVERRIDES="url1=path1,url2=path2"
```

## Pre-commit Hook Integration

### Installation

Add to `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/iilei/terraform-provider-jsonschema
    rev: v0.5.0
    hooks:
      - id: jsonschema-validator
```

### Configuration Methods

#### 1. Zero-config (uses .jsonschema-validator.yaml)

```yaml
repos:
  - repo: https://github.com/iilei/terraform-provider-jsonschema
    rev: v0.5.0
    hooks:
      - id: jsonschema-validator
        # Automatically reads .jsonschema-validator.yaml
```

#### 2. Inline configuration

```yaml
repos:
  - repo: https://github.com/iilei/terraform-provider-jsonschema
    rev: v0.5.0
    hooks:
      - id: jsonschema-validator
        name: Validate API requests
        args: ['--schema', 'api/request.schema.json']
        files: '^api/requests/.*\.json$'

      - id: jsonschema-validator
        name: Validate configuration
        args:
          - '--schema'
          - 'config.schema.json5'
          - '--schema-version'
          - 'draft/2020-12'
        files: '^config\.json5$'
```

#### 3. Python project (pyproject.toml)

```yaml
repos:
  - repo: https://github.com/iilei/terraform-provider-jsonschema
    rev: v0.5.0
    hooks:
      - id: jsonschema-validator
        # Automatically reads [tool.jsonschema-validator] from pyproject.toml
```

#### 4. Node.js project (package.json)

```yaml
repos:
  - repo: https://github.com/iilei/terraform-provider-jsonschema
    rev: v0.5.0
    hooks:
      - id: jsonschema-validator
        # Automatically reads "jsonschema-validator" from package.json
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Validate JSON files
on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: Install jsonschema-validator
        run: go install github.com/iilei/terraform-provider-jsonschema/cmd/jsonschema-validator@latest

      - name: Validate JSON files
        run: jsonschema-validator  # Uses .jsonschema-validator.yaml
```

### GitLab CI

```yaml
validate-json:
  image: golang:1.23
  stage: test
  script:
    - go install github.com/iilei/terraform-provider-jsonschema/cmd/jsonschema-validator@latest
    - jsonschema-validator  # Uses .jsonschema-validator.yaml
```

### CircleCI

```yaml
version: 2.1
jobs:
  validate:
    docker:
      - image: cimg/go:1.23
    steps:
      - checkout
      - run:
          name: Install jsonschema-validator
          command: go install github.com/iilei/terraform-provider-jsonschema/cmd/jsonschema-validator@latest
      - run:
          name: Validate JSON files
          command: jsonschema-validator
```

## Command-Line Reference

### Global Flags

```
--config, -c              Path to config file (.jsonschema-validator.yaml)
--schema, -s              Path to JSON Schema file (required if no config)
--schema-version          Schema draft version (draft/2020-12, draft/2019-09, etc.)
--ref-override            Override remote $ref (format: url=path, can be repeated)
--error-template          Custom error message template (Go template syntax)
--format                  Output format: text (default), json
--quiet, -q               Only output errors
--verbose, -v             Verbose output
--version                 Show version information
--help, -h                Show help
```

### Exit Codes

- `0` - All validations passed
- `1` - Validation errors found (schema violations)
- `2` - Usage errors (invalid arguments, missing files, configuration errors)

## Error Message Templates

Customize error output using Go templates:

### Template Variables

- `{{.FullMessage}}` - Complete formatted error message
- `{{.ErrorCount}}` - Number of validation errors
- `{{.Errors}}` - Array of individual errors
  - `{{.DocumentPath}}` - JSON path to the error location
  - `{{.Message}}` - Error message
  - `{{.Value}}` - The invalid value (truncated)
- `{{.SchemaFile}}` - Path to schema file
- `{{.Document}}` - Document content (truncated)

### Template Examples

**Simple list:**
```
{{range .Errors}}
- {{.DocumentPath}}: {{.Message}}
{{end}}
```

**GitHub Actions format:**
```
{{range .Errors}}
::error file={{$.SchemaFile}},line=1::{{.DocumentPath}}: {{.Message}}
{{end}}
```

**JSON format:**
```json
{
  "errors": [
    {{range $i, $e := .Errors}}
    {{if $i}},{{end}}
    {"path": "{{$e.DocumentPath}}", "message": "{{$e.Message}}"}
    {{end}}
  ]
}
```

## Comparison with Terraform Provider

The CLI tool provides the **exact same validation logic** as the Terraform provider:

| Feature | Terraform Provider | CLI Tool |
|---------|-------------------|----------|
| JSON5 support | ✅ | ✅ |
| Schema versions | ✅ All drafts | ✅ All drafts |
| Reference overrides | ✅ | ✅ |
| Error templates | ✅ | ✅ |
| Configuration | HCL | YAML/TOML/JSON |
| Use case | IaC validation | Pre-commit, CI/CD |

## Examples

See [examples/](../../examples/) directory for complete working examples:

- [examples/basic/](../../examples/basic/) - Simple validation examples
- [examples/ref_overrides/](../../examples/ref_overrides/) - Reference override examples
- Pre-commit examples (coming soon)

## Development

```bash
# Build
go build -o jsonschema-validator ./cmd/jsonschema-validator

# Run tests
go test ./cmd/jsonschema-validator/... -v

# Install locally
go install ./cmd/jsonschema-validator
```

## License

Same as parent project - see [LICENSE](../../LICENSE)

## Support

- **Issues**: https://github.com/iilei/terraform-provider-jsonschema/issues
- **Documentation**: https://registry.terraform.io/providers/iilei/jsonschema/latest/docs
- **Examples**: https://github.com/iilei/terraform-provider-jsonschema/tree/master/examples
