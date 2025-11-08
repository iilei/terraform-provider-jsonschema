terraform {
  required_providers {
    jsonschema = {
      source = "iilei/jsonschema"
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

# Output the validated document
output "validated_request" {
  value       = jsondecode(data.jsonschema_validator.api_request.validated)
  description = "The validated API request in canonical JSON format"
}

# Example showing validation failure
# Uncomment to test error handling
# data "jsonschema_validator" "invalid_request" {
#   schema = "${path.module}/schemas/api-request.schema.json"
#   
#   document = jsonencode({
#     user = {
#       email = "not-an-email"  # Invalid email format
#     }
#     # Missing required 'product' field
#   })
#   
#   ref_overrides = {
#     "https://api.example.com/schemas/user.json"    = "${path.module}/schemas/user.schema.json"
#     "https://api.example.com/schemas/product.json" = "${path.module}/schemas/product.schema.json"
#   }
# }
