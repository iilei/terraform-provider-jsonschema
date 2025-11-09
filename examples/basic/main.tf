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
