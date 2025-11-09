terraform {
  required_providers {
    jsonschema = {
      source  = "iilei/jsonschema"
      version = "0.0.0-dev" // replace this with actual latest version
    }
  }
}

provider "jsonschema" {}

# Basic validation
data "jsonschema_validator" "config" {
  document = file("${path.module}/valid-config.json")
  schema   = "${path.module}/config.schema.json"
}

output "validated_config" {
  value = jsondecode(data.jsonschema_validator.config.validated)
}

# Example with custom error template
data "jsonschema_validator" "config_with_template" {
  document = file("${path.module}/valid-config.json")
  schema   = "${path.module}/config.schema.json"
  
  error_message_template = <<-EOT
    Validation failed ({{.ErrorCount}} errors):
    {{range .Errors}}• {{.DocumentPath}}: {{.Message}}
    {{end}}
  EOT
}

# Traversal Demo: Schema references with $id fields
# 
# Directory structure:
#   main_schema/config.schema.json (has $id: "./../main_schema/config.schema.json")
#     ↓ $ref
#   traverse_up_dir/test.schema.json (has $id: "test.schema.json")
#
# This demonstrates:
# 1. How $id field affects $ref resolution
# 2. How SchemaFile shows the actual file path you provided
# 3. How SchemaPath shows the resolved schema URI + JSON Pointer fragment

# Valid document - should pass validation
data "jsonschema_validator" "traverse_demo_valid" {
  document = file("${path.module}/traverse-demo-valid.json")
  schema   = "${path.module}/schemas/traverse_tree_demo/main_schema/config.schema.json"
  
  error_message_template = <<-EOT
    ✅ Document is VALID
    Schema File: {{.SchemaFile}}
  EOT
}

# Invalid document - shows how SchemaFile and SchemaPath appear in errors
data "jsonschema_validator" "traverse_demo_invalid" {
  document = file("${path.module}/traverse-demo-invalid.json")
  schema   = "${path.module}/schemas/traverse_tree_demo/main_schema/config.schema.json"
  
  error_message_template = <<-EOT
    ═══════════════════════════════════════════════════════════════
    ❌ VALIDATION ERRORS - Schema Traversal Demo
    ═══════════════════════════════════════════════════════════════
    
    Schema File (as you specified it):
      {{.SchemaFile}}
    
    Total Errors: {{.ErrorCount}}
    
    {{range $i, $e := .Errors}}
    ┌─ Error {{add $i 1}} ────────────────────────────────────────────────
    │
    │ Document Path (where the error is in your JSON):
    │   {{.DocumentPath}}
    │
    │ Schema Path (the actual schema constraint that failed):
    │   {{.SchemaPath}}
    │
    │ Error Message:
    │   {{.Message}}
    │
    │ Invalid Value:
    │   {{.Value}}
    │
    └──────────────────────────────────────────────────────────────
    {{end}}
    
    KEY OBSERVATIONS:
    - SchemaFile: Your input path (relative to Terraform config)
    - SchemaPath: Full file:// URI with fragment, regardless of $id
    - The $id field affects $ref resolution, but not error reporting paths
  EOT
}

output "traverse_demo_summary" {
  description = "Schema traversal demo with $id and $ref"
  value = {
    schema_structure = {
      main_schema = "config.schema.json (has $id with relative path)"
      references = "test.schema.json via $ref with directory traversal"
      note = "SchemaPath shows resolved file:// URI, not the $id value"
    }
    valid_document = data.jsonschema_validator.traverse_demo_valid.validated
    demo_purpose = "See how SchemaFile and SchemaPath differ in error output"
  }
}
