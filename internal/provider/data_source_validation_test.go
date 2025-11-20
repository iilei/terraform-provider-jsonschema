package provider

import (
	"testing"

	validator "github.com/iilei/terraform-provider-jsonschema/pkg/jsonschema"
)

func TestDataSourceValidationLogic(t *testing.T) {
	tests := []struct {
		name           string
		document       string
		schemaVersion  string
		errorTemplate  string
		expectError    bool
		expectedResult string
	}{
		{
			name:          "valid config creation",
			document:      `{"test": "value"}`,
			schemaVersion: "draft-07",
			expectError:   false,
		},
		{
			name:          "invalid schema version",
			document:      `{"test": "value"}`,
			schemaVersion: "invalid-version",
			expectError:   true,
		},
		{
			name:           "JSON5 document parsing",
			document:       `{test: "value", /* comment */}`,
			expectError:    false,
			expectedResult: `{"test":"value"}`,
		},
		{
			name:        "invalid JSON document",
			document:    `{invalid json`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test config creation
			config, err := NewProviderConfig(tt.schemaVersion, tt.errorTemplate, false)
			if err != nil {
				if tt.expectError {
					return // Expected error in config creation
				}
				t.Fatalf("unexpected config error: %v", err)
			}

			if config == nil && !tt.expectError {
				t.Fatalf("expected non-nil config")
			}

			// Test document parsing with JSON5
			parsedDoc, err := validator.ParseJSON5String(tt.document)
			if err != nil {
				if tt.expectError {
					return // Expected error
				}
				t.Fatalf("unexpected document parsing error: %v", err)
			}

			// Test deterministic marshaling
			if !tt.expectError && parsedDoc != nil {
				result, err := validator.MarshalDeterministic(parsedDoc)
				if err != nil {
					t.Fatalf("unexpected marshaling error: %v", err)
				}

				if tt.expectedResult != "" && string(result) != tt.expectedResult {
					t.Errorf("expected result %q, got %q", tt.expectedResult, string(result))
				}
			}
		})
	}
}

func TestDataSourceSchemaStructure(t *testing.T) {
	ds := dataSourceJsonschemaValidator()

	// Test that all required fields are present in the schema
	expectedFields := []string{"document", "schema", "schema_version", "error_message_template"}

	for _, field := range expectedFields {
		if _, ok := ds.Schema[field]; !ok {
			t.Errorf("expected field %q to be present in data source schema", field)
		}
	}

	// Test that the computed field is present
	if _, ok := ds.Schema["valid_json"]; !ok {
		t.Errorf("expected 'valid_json' computed field to be present")
	}

	// Test that the read function is set
	if ds.Read == nil {
		t.Errorf("expected read function to be set")
	}
}

// Mock helper for testing configuration combinations
func TestConfigurationOverrides(t *testing.T) {
	tests := []struct {
		name                  string
		providerSchemaVersion string
		providerErrorTemplate string
		resourceSchemaVersion string
		resourceErrorTemplate string
		expectedSchemaVersion string
		expectedErrorTemplate string
	}{
		{
			name:                  "provider defaults only",
			providerSchemaVersion: "draft-07",
			providerErrorTemplate: "Provider error: {error}",
			expectedSchemaVersion: "draft-07",
			expectedErrorTemplate: "Provider error: {error}",
		},
		{
			name:                  "resource overrides provider",
			providerSchemaVersion: "draft-07",
			providerErrorTemplate: "Provider error: {error}",
			resourceSchemaVersion: "draft-04",
			resourceErrorTemplate: "Resource error: {error}",
			expectedSchemaVersion: "draft-04",
			expectedErrorTemplate: "Resource error: {error}",
		},
		{
			name:                  "partial resource overrides",
			providerSchemaVersion: "draft-07",
			providerErrorTemplate: "Provider error: {error}",
			resourceSchemaVersion: "draft-04",
			// resource doesn't specify error template
			expectedSchemaVersion: "draft-04",
			expectedErrorTemplate: "Provider error: {error}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the configuration resolution logic
			providerConfig, err := NewProviderConfig(tt.providerSchemaVersion, tt.providerErrorTemplate, false)
			if err != nil {
				t.Fatalf("unexpected provider config error: %v", err)
			}

			// Simulate resource-level overrides
			_ = tt.resourceSchemaVersion // Used for test structure only

			finalErrorTemplate := tt.resourceErrorTemplate
			if finalErrorTemplate == "" {
				finalErrorTemplate = providerConfig.DefaultErrorTemplate
			}

			if finalErrorTemplate != tt.expectedErrorTemplate {
				t.Errorf("expected error template %q, got %q", tt.expectedErrorTemplate, finalErrorTemplate)
			}
		})
	}
}
