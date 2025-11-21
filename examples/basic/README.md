# Basic Examples

This directory contains basic examples demonstrating the terraform-provider-jsonschema functionality.

## Examples

### 1. Basic Validation (`config`)

Simple validation of a JSON document against a schema.

```bash
terraform plan
```

### 2. Custom Error Template (`config_with_template`)

Demonstrates custom error formatting using Go templates with `{{.DocumentPath}}`.

### 3. Schema Traversal Demo with $id and $ref (`traverse_demo`)

**Purpose:** Demonstrates how `SchemaFile` and `SchemaPath` appear in error messages when schemas use `$id` fields and `$ref` to reference schemas across directories.

**Directory Structure:**
```
schemas/traverse_tree_demo/
├── main_schema/
│   └── config.schema.json      # Has $id: "./../main_schema/config.schema.json"
│                                # References test.schema.json via $ref
└── traverse_up_dir/
    └── test.schema.json         # Has $id: "test.schema.json"
                                 # Defines validation constraints
```

**Key Observations:**

- **SchemaFile**: Shows the relative path you provided to Terraform
  ```
  ./schemas/traverse_tree_demo/main_schema/config.schema.json
  ```

- **SchemaPath**: Shows the full `file://` URI with absolute path + JSON Pointer fragment to the actual constraint
  ```
  file:///absolute/path/traverse_up_dir/test.schema.json#/properties/version
  ```

- **$id field behavior**: The `$id` value in schemas affects how `$ref` resolves references, but does NOT affect error reporting paths

**Run the demo:**
```bash
terraform plan
```

**Expected output:**
- `traverse_demo_valid`: Passes validation silently
- `traverse_demo_invalid`: Shows formatted error with both `SchemaFile` and `SchemaPath`

The error template demonstrates:
```hcl
error_message_template = <<-EOT
  Schema File (as you specified it):
    {{.SchemaFile}}

  Schema Path (the actual schema constraint that failed):
    {{.SchemaPath}}
EOT
```

**Understanding the difference:**
1. `{{.SchemaFile}}` - Your input: the relative path to the main schema file as specified in Terraform
2. `{{.SchemaPath}}` - The resolved path: full `file://` URI pointing to where the actual constraint exists (follows `$ref` chain)
3. The `$id` field is used for `$ref` resolution during validation but doesn't change how paths are reported in errors

## Files

- `config.schema.json` - Basic schema for service configuration
- `valid-config.json` - Valid configuration document
- `schemas/traverse_tree_demo/` - Demo directory structure showing `$id` and `$ref` traversal
  - `main_schema/config.schema.json` - Entry point schema with `$id` and `$ref`
  - `traverse_up_dir/test.schema.json` - Referenced schema with validation constraints
- `traverse-demo-valid.json` - Valid document for traversal demo
- `traverse-demo-invalid.json` - Invalid document to trigger validation error (demonstrates paths)
- `main.tf` - Terraform configuration with all examples

## Running Examples

```bash
# Initialize Terraform
terraform init

# See all examples (note: traverse_demo_invalid will show an error by design)
terraform plan

# Focus on the traversal demo error output
terraform plan 2>&1 | grep -A 30 "Schema Traversal Demo"
```

## What You'll Learn

1. **Template Variables**: How to use `{{.SchemaFile}}` vs `{{.SchemaPath}}` in error templates
2. **Schema References**: How `$id` and `$ref` work together across directories
3. **Error Paths**: How error messages report paths (using `file://` URIs, not `$id` values)
4. **JSON Pointer**: How paths use RFC 6901 format with fragments (e.g., `#/properties/version`)
