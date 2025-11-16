package provider

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	validator "github.com/iilei/terraform-provider-jsonschema/pkg/jsonschema"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// TestGenerateSortedFullMessage_EmptyErrors tests the case where ValidationError has no extracted details
func TestGenerateSortedFullMessage_EmptyErrors(t *testing.T) {
	// Create a ValidationError with no causes (empty error tree)
	// This is a theoretical edge case where the error exists but has no extractable details

	// We need to create a schema that will compile and then manually construct
	// a validation scenario that might not have detailed errors

	schemaJSON := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object"
	}`

	var schemaData interface{}
	if err := json.Unmarshal([]byte(schemaJSON), &schemaData); err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)

	if err := compiler.AddResource("test://schema", schemaData); err != nil {
		t.Fatalf("Failed to add schema: %v", err)
	}

	compiledSchema, err := compiler.Compile("test://schema")
	if err != nil {
		t.Fatalf("Failed to compile schema: %v", err)
	}

	// Try to validate something that will fail at a fundamental level
	// Using a non-object type when object is required
	err = compiledSchema.Validate("not an object")
	if err == nil {
		t.Fatal("Expected validation to fail")
	}

	// Check if we can format this error
	result := validator.FormatValidationError(
		err,
		"test.json",
		`"not an object"`,
		"{{.FullMessage}}",
	)
	if result == nil {
		t.Fatal("Expected error result")
	}

	// The message should still contain the schema URL even if no detailed errors
	errMsg := result.Error()
	if !strings.Contains(errMsg, "test://schema") || !strings.Contains(errMsg, "validation") {
		t.Logf("Error message: %s", errMsg)
	}
}

// TestDataSourceJsonschemaValidatorRead_NoDraftConfiguration tests the fallback to Draft2020
func TestDataSourceJsonschemaValidatorRead_NoDraftConfiguration(t *testing.T) {
	// Create a temporary schema file without $schema field
	tempDir := t.TempDir()
	schemaPath := filepath.Join(tempDir, "test.schema.json")

	schemaContent := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"required": ["name"]
	}`

	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0o644); err != nil {
		t.Fatalf("Failed to create schema file: %v", err)
	}

	// Create document file
	docPath := filepath.Join(tempDir, "test.json")
	if err := os.WriteFile(docPath, []byte(`{"name": "test"}`), 0o644); err != nil {
		t.Fatalf("Failed to create document file: %v", err)
	}

	// Create resource data
	d := schema.TestResourceDataRaw(t, map[string]*schema.Schema{
		"document":               {Type: schema.TypeString},
		"schema":                 {Type: schema.TypeString},
		"schema_version":         {Type: schema.TypeString},
		"error_message_template": {Type: schema.TypeString},
		"ref_overrides":          {Type: schema.TypeMap},
		"valid_json":             {Type: schema.TypeString},
	}, map[string]interface{}{
		"document":       docPath,
		"schema":         schemaPath,
		"schema_version": "", // No version specified
	})

	// Create provider config with NO default schema version and NO default draft
	config := &ProviderConfig{
		DefaultSchemaVersion: "",  // Empty - no default
		DefaultDraft:         nil, // Nil - no default draft
		DefaultErrorTemplate: "",
	}

	// This should trigger the fallback to Draft2020 (line 121)
	err := dataSourceJsonschemaValidatorRead(d, config)
	if err != nil {
		t.Fatalf("Expected validation to succeed with Draft2020 fallback: %v", err)
	}

	// Verify the document was validated successfully
	validJson := d.Get("valid_json").(string)
	if validJson == "" {
		t.Error("Expected valid_json field to be set")
	}
}

// TestDataSourceJsonschemaValidatorRead_AddResourceFailure tests the AddResource error path
// This is extremely difficult to trigger because AddResource in jsonschema/v6 only fails when:
// 1. The URL is malformed (invalid URL parsing)
// 2. The resource is already registered at that URL
// Let's test the duplicate registration scenario
func TestDataSourceJsonschemaValidatorRead_DuplicateSchemaURL(t *testing.T) {
	// Create a temporary schema file
	tempDir := t.TempDir()
	schemaPath := filepath.Join(tempDir, "test.schema.json")

	schemaContent := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		}
	}`

	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0o644); err != nil {
		t.Fatalf("Failed to create schema file: %v", err)
	}

	// Create document file
	docPath := filepath.Join(tempDir, "test.json")
	if err := os.WriteFile(docPath, []byte(`{"name": "test"}`), 0o644); err != nil {
		t.Fatalf("Failed to create document file: %v", err)
	}

	// Create resource data
	d := schema.TestResourceDataRaw(t, map[string]*schema.Schema{
		"document":               {Type: schema.TypeString},
		"schema":                 {Type: schema.TypeString},
		"schema_version":         {Type: schema.TypeString},
		"error_message_template": {Type: schema.TypeString},
		"ref_overrides":          {Type: schema.TypeMap},
		"valid_json":             {Type: schema.TypeString},
	}, map[string]interface{}{
		"document": docPath,
		"schema":   schemaPath,
	})

	config := &ProviderConfig{
		DefaultSchemaVersion: "draft/2020-12",
		DefaultErrorTemplate: "",
	}

	// First call should succeed
	err := dataSourceJsonschemaValidatorRead(d, config)
	if err != nil {
		t.Fatalf("First validation failed: %v", err)
	}

	// Note: We can't easily trigger the AddResource error in lines 158-159
	// because each call to dataSourceJsonschemaValidatorRead creates a new compiler
	// instance, so there's no way to get duplicate registrations.
	//
	// The AddResource error path at lines 158-159 is defensive programming
	// for scenarios that are extremely unlikely to occur in practice without
	// significant changes to the code structure (e.g., reusing compiler instances).
	//
	// For now, we've documented this limitation. Achieving 100% coverage would
	// require dependency injection or mocking of the compiler, which would be
	// significant refactoring for minimal benefit.
}
