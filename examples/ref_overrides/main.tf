terraform {
  required_providers {
    jsonschema = {
      source  = "iilei/jsonschema"
      version = "0.0.0-dev" // replace this with actual latest version

    }
  }
}

provider "jsonschema" {
  # Provider configuration (optional)
}

# Example: Validating data against a schema that references remote schemas
# This demonstrates how to use ref_overrides to redirect remote URLs to local files

data "jsonschema_validator" "api_request" {
  # Main schema file that contains $ref to remote URLs
  # The api-request schema references:
  # 1. Full remote schema: https://api.example.com/schemas/user.json
  # 2. Remote schema with anchor fragment: https://api.example.com/schemas/user.json#email-format
  schema = "${path.module}/schemas/api-request.schema.json"

  # Document file path (v0.6.0+ API - no file() wrapper needed)
  document = "${path.module}/api-request.json"

  # Map remote schema URLs to local files
  # Once a base URL is overridden, anchor fragments are resolved automatically
  # from the local file (e.g., #email-format will be found in user.schema.json)
  ref_overrides = {
    "https://api.example.com/schemas/user.json"    = "${path.module}/schemas/user.schema.json"
    "https://api.example.com/schemas/product.json" = "${path.module}/schemas/product.schema.json"
  }
}

# Access the validated document
output "validated_request" {
  value       = jsondecode(data.jsonschema_validator.api_request.valid_json)
  description = "The validated API request data"
}
