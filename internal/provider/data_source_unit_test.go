package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestDataSourceJsonschemaValidatorRead(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "jsonschema_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test schema file
	schemaContent := `{
		"type": "object",
		"required": ["name"],
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer", "minimum": 0}
		}
	}`
	schemaFile := filepath.Join(tempDir, "test.schema.json")
	if err := os.WriteFile(schemaFile, []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create JSON5 schema file
	json5SchemaContent := `{
		// JSON5 schema with comments
		type: "object",
		required: ["email"],
		properties: {
			email: {type: "string", format: "email"},
			active: {type: "boolean"}
		}
	}`
	json5SchemaFile := filepath.Join(tempDir, "test.json5.schema")
	if err := os.WriteFile(json5SchemaFile, []byte(json5SchemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Helper to create document files
	createDocFile := func(content string, name string) string {
		docFile := filepath.Join(tempDir, name)
		if err := os.WriteFile(docFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		return docFile
	}

	tests := []struct {
		name                  string
		documentContent       string
		documentFileName      string
		schemaFile            string
		schemaVersionOverride string
		errorMessageTemplate  string
		providerConfig        *ProviderConfig
		expectError           bool
		errorContains         string
		expectedValidJson     string
	}{
		{
			name:             "valid document validation",
			documentContent:  `{"name": "John", "age": 25}`,
			documentFileName: "valid.json",
			schemaFile:       schemaFile,
			providerConfig: &ProviderConfig{
				DefaultErrorTemplate: "JSON Schema validation failed: {error}",
			},
			expectError:       false,
			expectedValidJson: `{"age":25,"name":"John"}`,
		},
		{
			name:             "JSON5 document with JSON schema",
			documentContent:  `{name: "John", age: 25, /* comment */}`,
			documentFileName: "valid.json5",
			schemaFile:       schemaFile,
			providerConfig: &ProviderConfig{
				DefaultErrorTemplate: "JSON Schema validation failed: {error}",
			},
			expectError:       false,
			expectedValidJson: `{"age":25,"name":"John"}`,
		},
		{
			name:             "JSON5 schema validation",
			documentContent:  `{"email": "john@example.com", "active": true}`,
			documentFileName: "email.json",
			schemaFile:       json5SchemaFile,
			providerConfig: &ProviderConfig{
				DefaultErrorTemplate: "JSON Schema validation failed: {error}",
			},
			expectError:       false,
			expectedValidJson: `{"active":true,"email":"john@example.com"}`,
		},
		{
			name:             "validation failure",
			documentContent:  `{"age": 25}`, // missing required "name"
			documentFileName: "invalid.json",
			schemaFile:       schemaFile,
			providerConfig: &ProviderConfig{
				DefaultErrorTemplate: "Validation error: {error}",
			},
			expectError:   true,
			errorContains: "Validation error:",
		},
		{
			name:                  "schema version override",
			documentContent:       `{"name": "John"}`,
			documentFileName:      "john.json",
			schemaFile:            schemaFile,
			schemaVersionOverride: "draft-04",
			providerConfig: &ProviderConfig{
				DefaultSchemaVersion: "draft-07",
				DefaultErrorTemplate: "JSON Schema validation failed: {error}",
			},
			expectError:       false,
			expectedValidJson: `{"name":"John"}`,
		},

		{
			name:                 "custom error template",
			documentContent:      `{"name": 123}`, // invalid type
			documentFileName:     "wrongtype.json",
			schemaFile:           schemaFile,
			errorMessageTemplate: "Custom error: {error}",
			providerConfig: &ProviderConfig{
				DefaultErrorTemplate: "Default: {error}",
			},
			expectError:   true,
			errorContains: "Custom error:",
		},
		{
			name:             "schema file not found error",
			documentContent:  `{"name": "John"}`,
			documentFileName: "test.json",
			schemaFile:       "/nonexistent/schema.json",
			providerConfig: &ProviderConfig{
				DefaultErrorTemplate: "JSON Schema validation failed: {error}",
			},
			expectError:   true,
			errorContains: "failed to parse schema file",
		},
		{
			name:             "invalid JSON document",
			documentContent:  `{invalid json`,
			documentFileName: "invalid_syntax.json",
			schemaFile:       schemaFile,
			providerConfig: &ProviderConfig{
				DefaultErrorTemplate: "JSON Schema validation failed: {error}",
			},
			expectError:   true,
			errorContains: "failed to parse document",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create document file
			var documentPath string
			if tt.documentFileName != "" {
				documentPath = createDocFile(tt.documentContent, tt.documentFileName)
			}

			// Create resource data
			resourceData := schema.TestResourceDataRaw(t, dataSourceJsonschemaValidator().Schema, map[string]interface{}{
				"document":               documentPath,
				"schema":                 tt.schemaFile,
				"schema_version":         tt.schemaVersionOverride,
				"error_message_template": tt.errorMessageTemplate,
			})

			// Call the read function
			err := dataSourceJsonschemaValidatorRead(resourceData, tt.providerConfig)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify valid_json field
			validJson := resourceData.Get("valid_json").(string)
			if tt.expectedValidJson != "" && validJson != tt.expectedValidJson {
				t.Errorf("expected valid_json %q, got %q", tt.expectedValidJson, validJson)
			}

			// Verify ID was set
			if resourceData.Id() == "" {
				t.Errorf("expected resource ID to be set")
			}
		})
	}
}

func TestDataSourceJsonschemaValidatorRead_InvalidProviderConfig(t *testing.T) {
	// Create temporary directory and files
	tempDir, err := os.MkdirTemp("", "jsonschema_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test document file
	docFile := filepath.Join(tempDir, "test.json")
	if err := os.WriteFile(docFile, []byte(`{"test": "value"}`), 0644); err != nil {
		t.Fatal(err)
	}

	resourceData := schema.TestResourceDataRaw(t, dataSourceJsonschemaValidator().Schema, map[string]interface{}{
		"document": docFile,
		"schema":   "/some/path.json",
	})

	// Pass invalid config type
	err = dataSourceJsonschemaValidatorRead(resourceData, "invalid-config")

	if err == nil {
		t.Errorf("expected error for invalid provider config")
	}

	if !strings.Contains(err.Error(), "invalid provider configuration") {
		t.Errorf("expected specific error message, got: %v", err)
	}
}

func TestDataSourceJsonschemaValidatorRead_InvalidDocumentParsing(t *testing.T) {
	// Create temporary schema file
	tempDir, err := os.MkdirTemp("", "jsonschema_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	schemaContent := `{"type": "object"}`
	schemaFile := filepath.Join(tempDir, "test.schema.json")
	if err := os.WriteFile(schemaFile, []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create document file with invalid JSON
	docFile := filepath.Join(tempDir, "invalid.json")
	if err := os.WriteFile(docFile, []byte(`{"test": "\x"}`), 0644); err != nil { // invalid escape sequence
		t.Fatal(err)
	}

	resourceData := schema.TestResourceDataRaw(t, dataSourceJsonschemaValidator().Schema, map[string]interface{}{
		"document": docFile,
		"schema":   schemaFile,
	})

	config := &ProviderConfig{
		DefaultErrorTemplate: "JSON Schema validation failed: {error}",
	}

	err = dataSourceJsonschemaValidatorRead(resourceData, config)

	if err == nil || !strings.Contains(err.Error(), "failed to parse document") {
		t.Errorf("expected 'failed to parse document' error, got: %v", err)
	}
}

func TestDataSourceJsonschemaValidatorRead_InvalidSchemaVersion(t *testing.T) {
	// Create temporary schema file
	tempDir, err := os.MkdirTemp("", "jsonschema_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	schemaContent := `{"type": "object"}`
	schemaFile := filepath.Join(tempDir, "test.schema.json")
	if err := os.WriteFile(schemaFile, []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create document file
	docFile := filepath.Join(tempDir, "test.json")
	if err := os.WriteFile(docFile, []byte(`{"test": "value"}`), 0644); err != nil {
		t.Fatal(err)
	}

	resourceData := schema.TestResourceDataRaw(t, dataSourceJsonschemaValidator().Schema, map[string]interface{}{
		"document":       docFile,
		"schema":         schemaFile,
		"schema_version": "invalid-version",
	})

	config := &ProviderConfig{
		DefaultErrorTemplate: "JSON Schema validation failed: {error}",
	}

	err = dataSourceJsonschemaValidatorRead(resourceData, config)

	if err == nil {
		t.Errorf("expected error for invalid schema version")
	}

	if !strings.Contains(err.Error(), "unsupported JSON Schema version") {
		t.Errorf("expected specific error message, got: %v", err)
	}
}

func TestDataSourceJsonschemaValidatorRead_MissingSchemaFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "jsonschema_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create document file
	docFile := filepath.Join(tempDir, "test.json")
	if err := os.WriteFile(docFile, []byte(`{"test": "value"}`), 0644); err != nil {
		t.Fatal(err)
	}

	resourceData := schema.TestResourceDataRaw(t, dataSourceJsonschemaValidator().Schema, map[string]interface{}{
		"document": docFile,
		"schema":   "/path/to/nonexistent/schema.json",
	})

	config := &ProviderConfig{
		DefaultErrorTemplate: "JSON Schema validation failed: {error}",
	}

	err = dataSourceJsonschemaValidatorRead(resourceData, config)

	if err == nil || !strings.Contains(err.Error(), "failed to parse schema file") {
		t.Errorf("expected 'failed to parse schema file' error, got: %v", err)
	}
}

func TestDataSourceJsonschemaValidatorRead_UnreadableSchemaFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "jsonschema_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file with no read permissions
	schemaFile := filepath.Join(tempDir, "unreadable.schema.json")
	if err := os.WriteFile(schemaFile, []byte(`{"type": "object"}`), 0000); err != nil {
		t.Fatal(err)
	}

	// Create document file
	docFile := filepath.Join(tempDir, "test.json")
	if err := os.WriteFile(docFile, []byte(`{"test": "value"}`), 0644); err != nil {
		t.Fatal(err)
	}

	resourceData := schema.TestResourceDataRaw(t, dataSourceJsonschemaValidator().Schema, map[string]interface{}{
		"document": docFile,
		"schema":   schemaFile,
	})

	config := &ProviderConfig{
		DefaultErrorTemplate: "JSON Schema validation failed: {error}",
	}

	err = dataSourceJsonschemaValidatorRead(resourceData, config)

	if err == nil || !strings.Contains(err.Error(), "failed to parse schema file") {
		t.Errorf("expected 'failed to parse schema file' error, got: %v", err)
	}
}

func TestDataSourceJsonschemaValidatorRead_InvalidSchemaFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "jsonschema_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create schema file with invalid escape sequence
	schemaFile := filepath.Join(tempDir, "invalid.schema.json")
	if err := os.WriteFile(schemaFile, []byte(`{"type": "\x"}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create document file
	docFile := filepath.Join(tempDir, "test.json")
	if err := os.WriteFile(docFile, []byte(`{"test": "value"}`), 0644); err != nil {
		t.Fatal(err)
	}

	resourceData := schema.TestResourceDataRaw(t, dataSourceJsonschemaValidator().Schema, map[string]interface{}{
		"document": docFile,
		"schema":   schemaFile,
	})

	config := &ProviderConfig{
		DefaultErrorTemplate: "JSON Schema validation failed: {error}",
	}

	err = dataSourceJsonschemaValidatorRead(resourceData, config)

	if err == nil || !strings.Contains(err.Error(), "failed to parse schema file") {
		t.Errorf("expected 'failed to parse schema file' error, got: %v", err)
	}
}

func TestHash(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "test",
			expected: "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", // SHA256 of "test"
		},
		{
			input:    "",
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", // SHA256 of empty string
		},
		{
			input:    "hello world",
			expected: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9", // SHA256 of "hello world"
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("hash_%s", tt.input), func(t *testing.T) {
			result := hash(tt.input)
			if result != tt.expected {
				t.Errorf("expected hash %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestDataSourceJsonschemaValidatorRead_ConfigurationCombinations(t *testing.T) {
	// Create temporary directory and schema file
	tempDir, err := os.MkdirTemp("", "jsonschema_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	schemaContent := `{"type": "object", "properties": {"name": {"type": "string"}}}`
	schemaFile := filepath.Join(tempDir, "test.schema.json")
	if err := os.WriteFile(schemaFile, []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create test document file
	docFile := filepath.Join(tempDir, "test.json")
	if err := os.WriteFile(docFile, []byte(`{"name": "test"}`), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		providerConfig *ProviderConfig
		resourceConfig map[string]interface{}
		expectError    bool
	}{
		{
			name: "provider defaults only",
			providerConfig: &ProviderConfig{
				DefaultSchemaVersion: "draft-07",
				DefaultErrorTemplate: "Provider: {error}",
			},
			resourceConfig: map[string]interface{}{
				"document": docFile,
				"schema":   schemaFile,
			},
			expectError: false,
		},
		{
			name: "resource overrides all",
			providerConfig: &ProviderConfig{
				DefaultSchemaVersion: "draft-07",
				DefaultErrorTemplate: "Provider: {error}",
			},
			resourceConfig: map[string]interface{}{
				"document":               docFile,
				"schema":                 schemaFile,
				"schema_version":         "draft-04",
				"base_url":               "https://resource.com/",
				"error_message_template": "Resource: {error}",
			},
			expectError: false,
		},
		{
			name: "empty provider config",
			providerConfig: &ProviderConfig{
				DefaultErrorTemplate: "JSON Schema validation failed: {error}",
			},
			resourceConfig: map[string]interface{}{
				"document": docFile,
				"schema":   schemaFile,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resourceData := schema.TestResourceDataRaw(t, dataSourceJsonschemaValidator().Schema, tt.resourceConfig)

			err := dataSourceJsonschemaValidatorRead(resourceData, tt.providerConfig)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Verify resource ID is set on success
			if !tt.expectError && resourceData.Id() == "" {
				t.Errorf("expected resource ID to be set")
			}
		})
	}
}

func TestDataSourceJsonschemaValidatorRead_RefOverrides(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "jsonschema_test_ref")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a base schema that references a remote URL
	baseSchemaContent := `{
		"type": "object",
		"properties": {
			"user": {
				"$ref": "https://example.com/schemas/user.json"
			}
		}
	}`
	baseSchemaFile := filepath.Join(tempDir, "base.schema.json")
	if err := os.WriteFile(baseSchemaFile, []byte(baseSchemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a local override schema
	overrideSchemaContent := `{
		"type": "object",
		"required": ["name"],
		"properties": {
			"name": {"type": "string"},
			"email": {"type": "string", "format": "email"}
		}
	}`
	overrideSchemaFile := filepath.Join(tempDir, "user.schema.json")
	if err := os.WriteFile(overrideSchemaFile, []byte(overrideSchemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create invalid override file
	invalidOverrideContent := `{invalid json`
	invalidOverrideFile := filepath.Join(tempDir, "invalid.schema.json")
	if err := os.WriteFile(invalidOverrideFile, []byte(invalidOverrideContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create unreadable override file
	unreadableOverrideFile := filepath.Join(tempDir, "unreadable.schema.json")
	if err := os.WriteFile(unreadableOverrideFile, []byte(`{"type": "object"}`), 0000); err != nil {
		t.Fatal(err)
	}

	// Create invalid schema override (not a valid JSON Schema object)
	invalidSchemaOverrideContent := `{"$schema": "invalid-meta-schema-url"}`
	invalidSchemaOverrideFile := filepath.Join(tempDir, "invalid_schema.json")
	if err := os.WriteFile(invalidSchemaOverrideFile, []byte(invalidSchemaOverrideContent), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name            string
		documentContent string
		refOverrides    map[string]interface{}
		expectError     bool
		errorContains   string
	}{
		{
			name:            "valid ref override",
			documentContent: `{"user": {"name": "John", "email": "john@example.com"}}`,
			refOverrides: map[string]interface{}{
				"https://example.com/schemas/user.json": overrideSchemaFile,
			},
			expectError: false,
		},
		{
			name:            "validation fails with override",
			documentContent: `{"user": {"email": "john@example.com"}}`, // missing required "name"
			refOverrides: map[string]interface{}{
				"https://example.com/schemas/user.json": overrideSchemaFile,
			},
			expectError:   true,
			errorContains: "validation",
		},
		{
			name:            "missing override file",
			documentContent: `{"user": {"name": "John"}}`,
			refOverrides: map[string]interface{}{
				"https://example.com/schemas/user.json": "/nonexistent/file.json",
			},
			expectError:   true,
			errorContains: "ref_override: failed to parse local file",
		},
		{
			name:            "invalid override file syntax",
			documentContent: `{"user": {"name": "John"}}`,
			refOverrides: map[string]interface{}{
				"https://example.com/schemas/user.json": invalidOverrideFile,
			},
			expectError:   true,
			errorContains: "ref_override: failed to parse local file",
		},
		{
			name:            "unreadable override file",
			documentContent: `{"user": {"name": "John"}}`,
			refOverrides: map[string]interface{}{
				"https://example.com/schemas/user.json": unreadableOverrideFile,
			},
			expectError:   true,
			errorContains: "ref_override: failed to parse local file",
		},
		{
			name:            "invalid schema structure in override",
			documentContent: `{"user": {"name": "John"}}`,
			refOverrides: map[string]interface{}{
				"https://example.com/schemas/user.json": invalidSchemaOverrideFile,
			},
			expectError:   true,
			errorContains: "failed to compile schema",
		},
	}

	config := &ProviderConfig{
		DefaultErrorTemplate: "{{.FullMessage}}",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create document file
			docFile := filepath.Join(tempDir, "doc_"+tt.name+".json")
			if err := os.WriteFile(docFile, []byte(tt.documentContent), 0644); err != nil {
				t.Fatal(err)
			}

			resourceData := schema.TestResourceDataRaw(t, dataSourceJsonschemaValidator().Schema, map[string]interface{}{
				"document":      docFile,
				"schema":        baseSchemaFile,
				"ref_overrides": tt.refOverrides,
			})

			err := dataSourceJsonschemaValidatorRead(resourceData, config)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestDataSourceJsonschemaValidatorRead_SchemaCompilationErrors(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "jsonschema_test_compile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create schema with invalid JSON Schema syntax
	invalidSchemaContent := `{
		"type": "object",
		"properties": {
			"name": {
				"type": "invalid_type_here"
			}
		}
	}`
	invalidSchemaFile := filepath.Join(tempDir, "invalid.schema.json")
	if err := os.WriteFile(invalidSchemaFile, []byte(invalidSchemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create schema with circular reference
	circularSchemaContent := `{
		"$ref": "#"
	}`
	circularSchemaFile := filepath.Join(tempDir, "circular.schema.json")
	if err := os.WriteFile(circularSchemaFile, []byte(circularSchemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name            string
		schemaFile      string
		documentContent string
		expectError     bool
		errorContains   string
	}{
		{
			name:            "invalid type in schema",
			schemaFile:      invalidSchemaFile,
			documentContent: `{"name": "test"}`,
			expectError:     true,
			errorContains:   "failed to compile schema",
		},
	}

	config := &ProviderConfig{
		DefaultErrorTemplate: "{{.FullMessage}}",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create document file
			docFile := filepath.Join(tempDir, "doc_"+tt.name+".json")
			if err := os.WriteFile(docFile, []byte(tt.documentContent), 0644); err != nil {
				t.Fatal(err)
			}

			resourceData := schema.TestResourceDataRaw(t, dataSourceJsonschemaValidator().Schema, map[string]interface{}{
				"document": docFile,
				"schema":   tt.schemaFile,
			})

			err := dataSourceJsonschemaValidatorRead(resourceData, config)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
			}
		})
	}
}

func TestDataSourceJsonschemaValidatorRead_DraftHandling(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "jsonschema_test_draft")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	schemaContent := `{"type": "object", "properties": {"name": {"type": "string"}}}`
	schemaFile := filepath.Join(tempDir, "test.schema.json")
	if err := os.WriteFile(schemaFile, []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create document file
	docFile := filepath.Join(tempDir, "test.json")
	if err := os.WriteFile(docFile, []byte(`{"name": "test"}`), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		providerConfig *ProviderConfig
		schemaVersion  string
		expectError    bool
	}{
		{
			name: "nil default draft in config",
			providerConfig: &ProviderConfig{
				DefaultErrorTemplate: "{{.FullMessage}}",
				DefaultDraft:         nil,
			},
			schemaVersion: "",
			expectError:   false,
		},
		{
			name: "use default draft when no schema version",
			providerConfig: &ProviderConfig{
				DefaultErrorTemplate: "{{.FullMessage}}",
				DefaultSchemaVersion: "",
				DefaultDraft:         jsonschema.Draft7,
			},
			schemaVersion: "",
			expectError:   false,
		},
		{
			name: "schema version with default draft",
			providerConfig: &ProviderConfig{
				DefaultErrorTemplate: "{{.FullMessage}}",
				DefaultSchemaVersion: "draft-07",
			},
			schemaVersion: "",
			expectError:   false,
		},
		{
			name: "override schema version",
			providerConfig: &ProviderConfig{
				DefaultErrorTemplate: "{{.FullMessage}}",
				DefaultSchemaVersion: "draft-07",
			},
			schemaVersion: "draft-04",
			expectError:   false,
		},
		{
			name: "completely empty config - should use fallback Draft2020",
			providerConfig: &ProviderConfig{
				DefaultErrorTemplate: "{{.FullMessage}}",
				DefaultSchemaVersion: "",
				DefaultDraft:         nil,
			},
			schemaVersion: "",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resourceData := schema.TestResourceDataRaw(t, dataSourceJsonschemaValidator().Schema, map[string]interface{}{
				"document":       docFile,
				"schema":         schemaFile,
				"schema_version": tt.schemaVersion,
			})

			err := dataSourceJsonschemaValidatorRead(resourceData, tt.providerConfig)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
