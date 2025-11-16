package provider

import (
	"encoding/json"
	"testing"

	validator "github.com/iilei/terraform-provider-jsonschema/pkg/jsonschema"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Test the actual validation schema and compiler behavior
func TestSchemaCompilationAndValidation(t *testing.T) {
	tests := []struct {
		name      string
		schema    string
		document  string
		version   string
		expectErr bool
	}{
		{
			name: "draft-07 validation success",
			schema: `{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"type": "object",
				"required": ["name"],
				"properties": {
					"name": {"type": "string"},
					"age": {"type": "integer", "minimum": 0}
				}
			}`,
			document:  `{"name": "John", "age": 25}`,
			version:   "draft-07",
			expectErr: false,
		},
		{
			name: "draft-04 validation success",
			schema: `{
				"$schema": "http://json-schema.org/draft-04/schema#",
				"type": "object",
				"required": ["test"],
				"properties": {
					"test": {"type": "string"}
				}
			}`,
			document:  `{"test": "value"}`,
			version:   "draft-04",
			expectErr: false,
		},
		{
			name: "validation failure - missing required field",
			schema: `{
				"type": "object",
				"required": ["name"],
				"properties": {
					"name": {"type": "string"}
				}
			}`,
			document:  `{"age": 25}`,
			version:   "draft-07",
			expectErr: true,
		},
		{
			name: "validation failure - type mismatch",
			schema: `{
				"type": "object",
				"properties": {
					"age": {"type": "integer"}
				}
			}`,
			document:  `{"age": "twenty-five"}`,
			version:   "draft-07",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test schema compilation using v6 API
			compiler := jsonschema.NewCompiler()

			// Parse the schema JSON
			var schemaData interface{}
			if err := json.Unmarshal([]byte(tt.schema), &schemaData); err != nil {
				if !tt.expectErr {
					t.Fatalf("failed to parse schema: %v", err)
				}
				return // Expected parsing error
			}

			// Add resource and compile
			schemaURL := "test-schema"
			if err := compiler.AddResource(schemaURL, schemaData); err != nil {
				if !tt.expectErr {
					t.Fatalf("failed to add schema resource: %v", err)
				}
				return // Expected compilation error
			}

			schema, err := compiler.Compile(schemaURL)
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("failed to compile schema: %v", err)
				}
				return // Expected compilation error
			}

			// Parse the document
			document, err := validator.ParseJSON5String(tt.document)
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("failed to parse document: %v", err)
				}
				return // Expected parsing error
			}

			// Validate the document
			err = schema.Validate(document)
			if tt.expectErr && err == nil {
				t.Errorf("expected validation error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}

// Test actual validation errors with custom error templates
func TestValidationErrorTemplateIntegration(t *testing.T) {
	tests := []struct {
		name          string
		schema        string
		document      string
		errorTemplate string
		expectedError string
		version       string
	}{
		{
			name: "simple template with validation error",
			schema: `{
				"type": "object",
				"required": ["name", "age"],
				"properties": {
					"name": {"type": "string"},
					"age": {"type": "integer", "minimum": 0}
				}
			}`,
			document:      `{"name": "John"}`, // Missing required "age"
			errorTemplate: "Config error: {{.FullMessage}}",
			expectedError: "Config error: jsonschema validation failed with 'test://schema.json#'\n- at '': missing property 'age'",
			version:       "draft-07",
		},
		{
			name: "detailed template with all variables",
			schema: `{
				"type": "object",
				"properties": {
					"port": {"type": "integer", "minimum": 1, "maximum": 65535}
				}
			}`,
			document:      `{"port": "8080"}`, // Wrong type - string instead of integer
			errorTemplate: "Schema: {{.SchemaFile}} | Error: {{.FullMessage}} | Document: {{.Document}}",
			expectedError: "Schema: test://schema.json | Error: jsonschema validation failed with 'test://schema.json#'\n- at '/port': got string, want integer | Document: {\"port\": \"8080\"}",
			version:       "draft/2020-12",
		},
		{
			name: "go template syntax",
			schema: `{
				"type": "array",
				"items": {"type": "string"},
				"minItems": 2
			}`,
			document:      `["single"]`, // Array too short
			errorTemplate: "Validation failed in {{.SchemaFile}}: {{.FullMessage}}",
			expectedError: "Validation failed in test://schema.json: jsonschema validation failed with 'test://schema.json#'\n- at '': minItems: got 1, want 2",
			version:       "draft-07",
		},
		{
			name: "ci/cd format template",
			schema: `{
				"type": "object",
				"required": ["version"],
				"properties": {
					"version": {"type": "string", "pattern": "^v[0-9]+\\.[0-9]+\\.[0-9]+$"}
				}
			}`,
			document:      `{"version": "invalid-version"}`, // Invalid version format
			errorTemplate: "::error file={{.SchemaFile}}::{{.FullMessage}}",
			expectedError: "::error file=test://schema.json::jsonschema validation failed with 'test://schema.json#'\n- at '/version': 'invalid-version' does not match pattern '^v[0-9]+\\\\.[0-9]+\\\\.[0-9]+$'",
			version:       "draft-06",
		},
		{
			name: "type mismatch with custom message",
			schema: `{
				"type": "object",
				"properties": {
					"enabled": {"type": "boolean"},
					"timeout": {"type": "number", "minimum": 0}
				}
			}`,
			document:      `{"enabled": "yes", "timeout": -5}`, // Multiple errors
			errorTemplate: "Configuration validation failed: {{.FullMessage}} (check your settings)",
			expectedError: "Configuration validation failed: jsonschema validation failed with 'test://schema.json#'\n- at '/enabled': got string, want boolean\n- at '/timeout': minimum: got -5, want 0 (check your settings)",
			version:       "draft/2019-09",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create provider config with error template
			config, err := NewProviderConfig(
				tt.version,
				tt.errorTemplate,
				true,
			) // Enable detailed errors for testing
			if err != nil {
				t.Fatalf("failed to create provider config: %v", err)
			}

			// Parse the document
			documentData, err := validator.ParseJSON5String(tt.document)
			if err != nil {
				t.Fatalf("failed to parse document: %v", err)
			}

			// Parse the schema
			schemaData, err := validator.ParseJSON5String(tt.schema)
			if err != nil {
				t.Fatalf("failed to parse schema: %v", err)
			}

			// Create compiler with appropriate draft using v6 API
			compiler := jsonschema.NewCompiler()
			if config.DefaultDraft != nil {
				compiler.DefaultDraft(config.DefaultDraft)
			}

			// Convert schema to JSON for compilation
			schemaJSON, err := validator.MarshalDeterministic(schemaData)
			if err != nil {
				t.Fatalf("failed to marshal schema: %v", err)
			}

			// Parse and add schema resource, then compile using v6 API
			var parsedSchema interface{}
			if err := json.Unmarshal(schemaJSON, &parsedSchema); err != nil {
				t.Fatalf("failed to parse schema JSON: %v", err)
			}

			schemaURL := "test://schema.json"
			if err := compiler.AddResource(schemaURL, parsedSchema); err != nil {
				t.Fatalf("failed to add schema resource: %v", err)
			}

			compiledSchema, err := compiler.Compile(schemaURL)
			if err != nil {
				t.Fatalf("failed to compile schema: %v", err)
			}

			// Validate the document (this should fail)
			validationErr := compiledSchema.Validate(documentData)
			if validationErr == nil {
				t.Fatalf("expected validation error but got none")
			}

			// Format the error using our error formatter
			formattedErr := validator.FormatValidationError(
				validationErr,
				"test://schema.json",
				tt.document,
				tt.errorTemplate,
			)
			if formattedErr == nil {
				t.Fatalf("expected formatted error but got nil")
			}

			// Check that the error matches exactly what we expect
			errorMsg := formattedErr.Error()
			if errorMsg == "" {
				t.Fatalf("formatted error message is empty")
			}

			if errorMsg != tt.expectedError {
				t.Errorf("expected error message %q, got: %q", tt.expectedError, errorMsg)
			}

			// Verify the error message was actually formatted (not just the raw validation error)
			if errorMsg == validationErr.Error() {
				t.Errorf("error message was not formatted, got raw validation error: %s", errorMsg)
			}

			t.Logf("Validation error: %s", validationErr.Error())
			t.Logf("Formatted error: %s", errorMsg)
		})
	}
}

// Test real validation with JSON5 features
func TestJSON5ValidationIntegration(t *testing.T) {
	tests := []struct {
		name          string
		schema        string
		document      string
		errorTemplate string
		expectedError string
		expectError   bool
	}{
		{
			name: "valid JSON5 document and schema",
			schema: `{
				// Schema with comments
				/*
					... even multi-line comments
				*/
				"type": "object",
				"required": ["service", "config"],
				"properties": {
					service: {"type": "string"}, // Unquoted key
					config: {
						type: "object",
						properties: {
							port: {"type": "integer"},
							enabled: {"type": "boolean"},
						}
					}
				}
			}`,
			document: `{
				// Service configuration
				"service": "api-server",
				"config": {
					port: 8080,     // Unquoted number
					enabled: true,  // Trailing comma allowed
				},
			}`,
			expectError: false,
		},
		{
			name: "JSON5 validation failure with template",
			schema: `{
				"type": "object",
				"required": ["name"],
				"properties": {
					name: {"type": "string", "minLength": 3}
				}
			}`,
			document: `{
				// Invalid short name
				name: "ab", // Too short (< 3 chars)
			}`,
			errorTemplate: "JSON5 validation error: {{.FullMessage}}",
			expectedError: "JSON5 validation error: jsonschema validation failed with 'test://json5-schema.json#'\n- at '/name': minLength: got 2, want 3",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse JSON5 schema
			schemaData, err := validator.ParseJSON5String(tt.schema)
			if err != nil {
				t.Fatalf("failed to parse JSON5 schema: %v", err)
			}

			// Parse JSON5 document
			documentData, err := validator.ParseJSON5String(tt.document)
			if err != nil {
				t.Fatalf("failed to parse JSON5 document: %v", err)
			}

			// Compile schema using v6 API
			schemaJSON, err := validator.MarshalDeterministic(schemaData)
			if err != nil {
				t.Fatalf("failed to marshal schema: %v", err)
			}

			compiler := jsonschema.NewCompiler()
			var parsedSchema interface{}
			if err := json.Unmarshal(schemaJSON, &parsedSchema); err != nil {
				t.Fatalf("failed to parse schema JSON: %v", err)
			}

			schemaURL := "test://json5-schema.json"
			if err := compiler.AddResource(schemaURL, parsedSchema); err != nil {
				t.Fatalf("failed to add schema resource: %v", err)
			}

			compiledSchema, err := compiler.Compile(schemaURL)
			if err != nil {
				t.Fatalf("failed to compile schema: %v", err)
			}

			// Validate
			validationErr := compiledSchema.Validate(documentData)

			if tt.expectError {
				if validationErr == nil {
					t.Errorf("expected validation error but got none")
					return
				}

				// Test error formatting if template provided
				if tt.errorTemplate != "" && tt.expectedError != "" {
					formattedErr := validator.FormatValidationError(
						validationErr,
						"test://json5-schema.json",
						tt.document,
						tt.errorTemplate,
					)
					if formattedErr == nil {
						t.Errorf("expected formatted error but got nil")
					} else {
						errorMsg := formattedErr.Error()
						if errorMsg == "" {
							t.Errorf("formatted error message is empty")
						}
						if errorMsg != tt.expectedError {
							t.Errorf("expected error message %q, got: %q", tt.expectedError, errorMsg)
						}
						t.Logf("JSON5 validation error: %s", errorMsg)
					}
				}
			} else {
				if validationErr != nil {
					t.Errorf("unexpected validation error: %v", validationErr)
				}
			}
		})
	}
}

// Test configuration resolution between provider and resource levels
func TestConfigurationResolution(t *testing.T) {
	tests := []struct {
		name                       string
		providerSchemaVersion      string
		providerErrorTemplate      string
		resourceSchemaVersion      string
		resourceErrorTemplate      string
		expectedFinalVersion       string
		expectedFinalErrorTemplate string
	}{
		{
			name:                       "all provider defaults",
			providerSchemaVersion:      "draft-07",
			providerErrorTemplate:      "Provider: {{.FullMessage}}",
			expectedFinalVersion:       "draft-07",
			expectedFinalErrorTemplate: "Provider: {{.FullMessage}}",
		},
		{
			name:                       "resource overrides all",
			providerSchemaVersion:      "draft-07",
			providerErrorTemplate:      "Provider: {{.FullMessage}}",
			resourceSchemaVersion:      "draft-04",
			resourceErrorTemplate:      "Resource: {{.FullMessage}}",
			expectedFinalVersion:       "draft-04",
			expectedFinalErrorTemplate: "Resource: {{.FullMessage}}",
		},
		{
			name:                  "partial resource override",
			providerSchemaVersion: "draft-07",
			providerErrorTemplate: "Provider: {{.FullMessage}}",
			resourceSchemaVersion: "draft-04",
			// resource doesn't specify error template
			expectedFinalVersion:       "draft-04",
			expectedFinalErrorTemplate: "Provider: {{.FullMessage}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create provider config
			providerConfig, err := NewProviderConfig(
				tt.providerSchemaVersion,
				tt.providerErrorTemplate,
				false,
			)
			if err != nil {
				t.Fatalf("failed to create provider config: %v", err)
			}

			// Simulate configuration resolution logic (like what happens in the data source)
			finalVersion := tt.resourceSchemaVersion
			if finalVersion == "" {
				finalVersion = providerConfig.DefaultSchemaVersion
			}

			finalErrorTemplate := tt.resourceErrorTemplate
			if finalErrorTemplate == "" {
				finalErrorTemplate = providerConfig.DefaultErrorTemplate
			}

			// Verify the resolution matches expectations
			if finalVersion != tt.expectedFinalVersion {
				t.Errorf("expected final version %q, got %q", tt.expectedFinalVersion, finalVersion)
			}
			if finalErrorTemplate != tt.expectedFinalErrorTemplate {
				t.Errorf(
					"expected final error template %q, got %q",
					tt.expectedFinalErrorTemplate,
					finalErrorTemplate,
				)
			}
		})
	}
}

// Test edge cases for validation
func TestValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		schema    string
		document  string
		expectErr bool
	}{
		{
			name:      "empty schema",
			schema:    `{}`,
			document:  `{"anything": "goes"}`,
			expectErr: false, // Empty schema should allow anything
		},
		{
			name: "schema with $ref (should work with proper base URL)",
			schema: `{
				"type": "object",
				"properties": {
					"test": {"$ref": "#/definitions/testDef"}
				},
				"definitions": {
					"testDef": {"type": "string"}
				}
			}`,
			document:  `{"test": "value"}`,
			expectErr: false,
		},
		{
			name: "complex nested validation",
			schema: `{
				"type": "object",
				"required": ["users"],
				"properties": {
					"users": {
						"type": "array",
						"items": {
							"type": "object",
							"required": ["name", "email"],
							"properties": {
								"name": {"type": "string"},
								"email": {"type": "string", "format": "email"}
							}
						}
					}
				}
			}`,
			document: `{
				"users": [
					{"name": "John", "email": "john@example.com"},
					{"name": "Jane", "email": "jane@example.com"}
				]
			}`,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compile schema using v6 API
			compiler := jsonschema.NewCompiler()

			var schemaData interface{}
			if err := json.Unmarshal([]byte(tt.schema), &schemaData); err != nil {
				if !tt.expectErr {
					t.Fatalf("failed to parse schema: %v", err)
				}
				return
			}

			schemaURL := "test-schema"
			if err := compiler.AddResource(schemaURL, schemaData); err != nil {
				if !tt.expectErr {
					t.Fatalf("failed to add schema resource: %v", err)
				}
				return
			}

			schema, err := compiler.Compile(schemaURL)
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("failed to compile schema: %v", err)
				}
				return
			}

			document, err := validator.ParseJSON5String(tt.document)
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("failed to parse document: %v", err)
				}
				return
			}

			err = schema.Validate(document)
			if tt.expectErr && err == nil {
				t.Errorf("expected validation error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}
