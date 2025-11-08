# ref_overrides Example

This example demonstrates how to use the `ref_overrides` feature to validate JSON documents against schemas that reference remote URLs, without requiring network access.

## Use Case

When your JSON schema contains `$ref` references to remote URLs (e.g., `https://api.example.com/schemas/user.json`), you can use `ref_overrides` to redirect these to local files. This is useful for:

- **Offline validation**: No internet connection required
- **Air-gapped environments**: Works in restricted networks
- **Version control**: Keep all schemas in your repository
- **Deterministic builds**: Same inputs always produce same results
- **Performance**: No network latency
- **Security**: No external dependencies or credential management

## How It Works

The `ref_overrides` parameter accepts a map of remote URLs to local file paths:

```hcl
data "jsonschema_validator" "example" {
  schema   = "${path.module}/schemas/main.schema.json"
  document = jsonencode({...})
  
  ref_overrides = {
    "https://example.com/remote-schema.json" = "${path.module}/schemas/local-schema.json"
  }
}
```

When the schema compiler encounters a `$ref` to a URL in the `ref_overrides` map, it uses the local file instead of attempting to fetch the remote URL.

## File Structure

```
ref_overrides/
├── main.tf                              # Terraform configuration
├── README.md                            # This file
└── schemas/
    ├── api-request.schema.json          # Main schema with remote $refs
    ├── user.schema.json                 # Local user schema
    └── product.schema.json              # Local product schema
```

## Schema References

The `api-request.schema.json` contains:

```json
{
  "properties": {
    "user": {
      "$ref": "https://api.example.com/schemas/user.json"
    },
    "product": {
      "$ref": "https://api.example.com/schemas/product.json"
    }
  }
}
```

These remote URLs are redirected to local files via `ref_overrides`.

## Running the Example

1. Initialize Terraform:
   ```bash
   terraform init
   ```

2. Validate the configuration:
   ```bash
   terraform validate
   ```

3. Apply to see the validation result:
   ```bash
   terraform apply
   ```

The output will show the validated document in canonical JSON format.

## Testing Validation Errors

Uncomment the `invalid_request` data source in `main.tf` to see how validation errors are reported when the document doesn't match the schema.

## JSON5 Support

Both the main schema and override schemas support JSON5 format, allowing:
- Comments in schema files
- Trailing commas
- Unquoted keys
- Single-quoted strings

## Resolution Order

When the compiler encounters a `$ref`:
1. First checks if the URL is in `ref_overrides` (uses local file)
2. If not found, uses the default loader (file:// URLs)
3. Results are cached for subsequent references

This means you can mix remote refs (overridden to local) with local file:// refs in the same schema.
