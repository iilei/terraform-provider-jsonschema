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
  schema = "${path.module}/schemas/api-request.schema.json"
  
  # The JSON document to validate
  document = jsonencode({
    user = {
      name  = "John Doe"
      email = "john@example.com"
      age   = 30
    }
    product = {
      sku   = "PROD-123"
      name  = "Widget"
      price = 29.99
    }
  })
  
  # Map remote schema URLs to local files
  # This allows validation without network access
  ref_overrides = {
    "https://api.example.com/schemas/user.json"    = "${path.module}/schemas/user.schema.json"
    "https://api.example.com/schemas/product.json" = "${path.module}/schemas/product.schema.json"
  }
}

