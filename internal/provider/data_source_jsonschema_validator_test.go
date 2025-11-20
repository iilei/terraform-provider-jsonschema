package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func Test_dataSourceJsonschemaValidatorRead(t *testing.T) {
	// Create temporary directory for test schema files
	tempDir, err := os.MkdirTemp("", "jsonschema_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Write schema files
	schemaFile := filepath.Join(tempDir, "test.schema.json")
	if err := os.WriteFile(schemaFile, []byte(schemaValid), 0644); err != nil {
		t.Fatal(err)
	}

	json5SchemaFile := filepath.Join(tempDir, "test.schema.json5")
	if err := os.WriteFile(json5SchemaFile, []byte(schemaJSON5), 0644); err != nil {
		t.Fatal(err)
	}

	var cases = []struct {
		name             string
		documentContent  string
		documentFileName string
		schemaFile       string
		errorExpected    bool
		expectedJSON     string
	}{
		{
			name:             "invalid document",
			documentContent:  "asd asdasd: ^%^*&^%",
			documentFileName: "invalid.txt",
			schemaFile:       schemaFile,
			errorExpected:    true,
		},
		{
			name:             "empty object fails required validation",
			documentContent:  "{}",
			documentFileName: "empty.json",
			schemaFile:       schemaFile,
			errorExpected:    true,
		},
		{
			name:             "valid document",
			documentContent:  `{"test": "test"}`,
			documentFileName: "valid.json",
			schemaFile:       schemaFile,
			errorExpected:    false,
			expectedJSON:     `{"test":"test"}`,
		},
		{
			name:             "JSON5 document with comments",
			documentContent:  `{"test": "test", /* comment */ }`,
			documentFileName: "valid.json5",
			schemaFile:       schemaFile,
			errorExpected:    false,
			expectedJSON:     `{"test":"test"}`,
		},
		{
			name:             "JSON5 schema with JSON document",
			documentContent:  `{"name": "example", "age": 25}`,
			documentFileName: "person.json",
			schemaFile:       json5SchemaFile,
			errorExpected:    false,
			expectedJSON:     `{"age":25,"name":"example"}`,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			// Create document file
			docFile := filepath.Join(tempDir, tt.documentFileName)
			if err := os.WriteFile(docFile, []byte(tt.documentContent), 0644); err != nil {
				t.Fatal(err)
			}

			resource.Test(t, resource.TestCase{
				PreCheck:          func() { testAccPreCheck(t) },
				ProviderFactories: providerFactories,
				Steps: []resource.TestStep{
					{
						Config: makeDataSourceWithFile(docFile, tt.schemaFile),
						Check: resource.ComposeAggregateTestCheckFunc(
							func() resource.TestCheckFunc {
								if tt.expectedJSON != "" {
									return resource.TestCheckResourceAttr("data.jsonschema_validator.test", "valid_json", tt.expectedJSON)
								}
								return resource.TestCheckResourceAttrSet("data.jsonschema_validator.test", "valid_json")
							}(),
						),
					},
				},
				ErrorCheck: func(err error) error {
					if tt.errorExpected {
						if err == nil {
							return fmt.Errorf("error expected for case: %s", tt.name)
						}
						return nil
					}
					return err
				},
			})
		})
	}
}

func TestProviderConfiguration(t *testing.T) {
	// Create temporary directory for test schema files
	tempDir, err := os.MkdirTemp("", "jsonschema_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	schemaFile := filepath.Join(tempDir, "test.schema.json")
	if err := os.WriteFile(schemaFile, []byte(schemaValidDraft04), 0644); err != nil {
		t.Fatal(err)
	}

	// Create document file
	docFile := filepath.Join(tempDir, "test.json")
	if err := os.WriteFile(docFile, []byte(`{"test":"value"}`), 0644); err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: makeProviderConfigTest(schemaFile, docFile),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jsonschema_validator.test", "valid_json", `{"test":"value"}`),
				),
			},
		},
	})
}

// TestMultipleSchemasInSameDirectory verifies that schemas with different filenames work correctly
// This is a breaking change from the previous implementation that hardcoded "schema.json" in the URL
func TestMultipleSchemasInSameDirectory(t *testing.T) {
	// Create temporary directory for test schema files
	tempDir, err := os.MkdirTemp("", "jsonschema_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create multiple schema files in the same directory with different names
	schema1File := filepath.Join(tempDir, "user.schema.json")
	if err := os.WriteFile(schema1File, []byte(userSchema), 0644); err != nil {
		t.Fatal(err)
	}

	schema2File := filepath.Join(tempDir, "product.schema.json")
	if err := os.WriteFile(schema2File, []byte(productSchema), 0644); err != nil {
		t.Fatal(err)
	}

	var cases = []struct {
		name             string
		documentContent  string
		documentFileName string
		schemaFile       string
		errorExpected    bool
		expectedJSON     string
	}{
		{
			name:             "validate user document",
			documentContent:  `{"name": "John", "email": "john@example.com"}`,
			documentFileName: "user.json",
			schemaFile:       schema1File,
			errorExpected:    false,
			expectedJSON:     `{"email":"john@example.com","name":"John"}`,
		},
		{
			name:             "validate product document",
			documentContent:  `{"sku": "ABC123", "price": 99.99}`,
			documentFileName: "product.json",
			schemaFile:       schema2File,
			errorExpected:    false,
			expectedJSON:     `{"price":99.99,"sku":"ABC123"}`,
		},
		{
			name:             "invalid user document against user schema",
			documentContent:  `{"name": "John"}`, // missing email
			documentFileName: "invalid_user.json",
			schemaFile:       schema1File,
			errorExpected:    true,
		},
		{
			name:             "invalid product document against product schema",
			documentContent:  `{"sku": "ABC123"}`, // missing price
			documentFileName: "invalid_product.json",
			schemaFile:       schema2File,
			errorExpected:    true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			// Create document file
			docFile := filepath.Join(tempDir, tt.documentFileName)
			if err := os.WriteFile(docFile, []byte(tt.documentContent), 0644); err != nil {
				t.Fatal(err)
			}

			resource.Test(t, resource.TestCase{
				PreCheck:          func() { testAccPreCheck(t) },
				ProviderFactories: providerFactories,
				Steps: []resource.TestStep{
					{
						Config: makeDataSourceWithFile(docFile, tt.schemaFile),
						Check: resource.ComposeAggregateTestCheckFunc(
							func() resource.TestCheckFunc {
								if tt.expectedJSON != "" {
									return resource.TestCheckResourceAttr("data.jsonschema_validator.test", "valid_json", tt.expectedJSON)
								}
								return resource.TestCheckResourceAttrSet("data.jsonschema_validator.test", "valid_json")
							}(),
						),
					},
				},
				ErrorCheck: func(err error) error {
					if tt.errorExpected {
						if err == nil {
							return fmt.Errorf("error expected for case: %s", tt.name)
						}
						return nil
					}
					return err
				},
			})
		})
	}
}

func makeDataSourceWithFile(documentPath string, schemaFile string) string {
	return fmt.Sprintf(`
data "jsonschema_validator" "test" {
  document = %q
  schema   = %q
}
`, documentPath, schemaFile)
}

func makeProviderConfigTest(schemaFile string, documentPath string) string {
	return fmt.Sprintf(`
provider "jsonschema" {
  schema_version = "draft-04"
}

data "jsonschema_validator" "test" {
  document = %q
  schema   = %q
}
`, documentPath, schemaFile)
}

var schemaValid = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["test"],
  "properties": {
    "test": {
      "type": "string"
    }
  }
}`

var schemaJSON5 = `{
  // JSON5 schema with comments
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["name", "age"],
  "properties": {
    "name": {
      "type": "string",
    }, // trailing comma allowed
    "age": {
      "type": "number"
    }
  }
}`

var schemaValidDraft04 = `{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "type": "object",
  "required": ["test"],
  "properties": {
    "test": {
      "type": "string"
    }
  }
}`

var userSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["name", "email"],
  "properties": {
    "name": {
      "type": "string"
    },
    "email": {
      "type": "string",
      "format": "email"
    }
  }
}`

var productSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["sku", "price"],
  "properties": {
    "sku": {
      "type": "string"
    },
    "price": {
      "type": "number",
      "minimum": 0
    }
  }
}`

func TestRefOverrides(t *testing.T) {
	// Create temporary directory for test schemas
	tempDir, err := os.MkdirTemp("", "jsonschema_ref_override_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Main schema with remote $refs
	mainSchema := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"user": {
				"$ref": "https://example.com/schemas/user.json"
			},
			"product": {
				"$ref": "https://example.com/schemas/product.json"
			}
		},
		"required": ["user", "product"]
	}`

	// User schema (to override remote URL)
	userSchema := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"name": {
				"type": "string"
			},
			"age": {
				"type": "integer",
				"minimum": 0
			}
		},
		"required": ["name"]
	}`

	// Product schema (to override remote URL)
	productSchema := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"id": {
				"type": "string"
			},
			"price": {
				"type": "number",
				"minimum": 0
			}
		},
		"required": ["id", "price"]
	}`

	// Write schema files
	mainSchemaPath := filepath.Join(tempDir, "main.schema.json")
	if err := os.WriteFile(mainSchemaPath, []byte(mainSchema), 0644); err != nil {
		t.Fatal(err)
	}

	userSchemaPath := filepath.Join(tempDir, "user.schema.json")
	if err := os.WriteFile(userSchemaPath, []byte(userSchema), 0644); err != nil {
		t.Fatal(err)
	}

	productSchemaPath := filepath.Join(tempDir, "product.schema.json")
	if err := os.WriteFile(productSchemaPath, []byte(productSchema), 0644); err != nil {
		t.Fatal(err)
	}

	// Test document
	testDoc := `{
		"user": {
			"name": "John Doe",
			"age": 30
		},
		"product": {
			"id": "prod-123",
			"price": 29.99
		}
	}`

	// Write test document file
	testDocPath := filepath.Join(tempDir, "test.json")
	if err := os.WriteFile(testDocPath, []byte(testDoc), 0644); err != nil {
		t.Fatal(err)
	}

	// Create provider config
	config := &ProviderConfig{
		DefaultSchemaVersion: "draft/2020-12",
		DefaultErrorTemplate: "{{.FullMessage}}",
		DefaultDraft:         nil,
	}

	// Create resource data
	d := dataSourceJsonschemaValidator().TestResourceData()
	d.Set("schema", mainSchemaPath)
	d.Set("document", testDocPath)
	d.Set("ref_overrides", map[string]interface{}{
		"https://example.com/schemas/user.json":    userSchemaPath,
		"https://example.com/schemas/product.json": productSchemaPath,
	})

	// Run the read function
	err = dataSourceJsonschemaValidatorRead(d, config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check that validation succeeded
	validJson := d.Get("valid_json").(string)
	if validJson == "" {
		t.Fatal("Expected valid_json document, got empty string")
	}

	// Check that ID was set
	if d.Id() == "" {
		t.Fatal("Expected resource ID to be set")
	}
}

func TestRefOverridesErrors(t *testing.T) {
	config := &ProviderConfig{
		DefaultSchemaVersion: "draft/2020-12",
		DefaultErrorTemplate: "{{.FullMessage}}",
		DefaultDraft:         nil,
	}

	t.Run("missing override file", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "jsonschema_ref_override_error_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		// Create a simple schema
		mainSchema := `{"type": "object"}`
		mainSchemaPath := filepath.Join(tempDir, "main.schema.json")
		if err := os.WriteFile(mainSchemaPath, []byte(mainSchema), 0644); err != nil {
			t.Fatal(err)
		}

		// Create document file
		docPath := filepath.Join(tempDir, "test.json")
		if err := os.WriteFile(docPath, []byte(`{}`), 0644); err != nil {
			t.Fatal(err)
		}

		d := dataSourceJsonschemaValidator().TestResourceData()
		d.Set("schema", mainSchemaPath)
		d.Set("document", docPath)
		d.Set("ref_overrides", map[string]interface{}{
			"https://example.com/schema.json": "/nonexistent/file.json",
		})

		err = dataSourceJsonschemaValidatorRead(d, config)
		if err == nil {
			t.Fatal("Expected error for missing override file, got nil")
		}
		if !strings.Contains(err.Error(), "ref_override") {
			t.Errorf("Expected error message to contain 'ref_override', got: %v", err)
		}
	})

	t.Run("invalid JSON in override file", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "jsonschema_ref_override_error_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		// Create a simple schema
		mainSchema := `{"type": "object"}`
		mainSchemaPath := filepath.Join(tempDir, "main.schema.json")
		if err := os.WriteFile(mainSchemaPath, []byte(mainSchema), 0644); err != nil {
			t.Fatal(err)
		}

		// Create invalid override file
		invalidOverride := filepath.Join(tempDir, "invalid.json")
		if err := os.WriteFile(invalidOverride, []byte(`{invalid json`), 0644); err != nil {
			t.Fatal(err)
		}

		// Create document file
		docPath := filepath.Join(tempDir, "test.json")
		if err := os.WriteFile(docPath, []byte(`{}`), 0644); err != nil {
			t.Fatal(err)
		}

		d := dataSourceJsonschemaValidator().TestResourceData()
		d.Set("schema", mainSchemaPath)
		d.Set("document", docPath)
		d.Set("ref_overrides", map[string]interface{}{
			"https://example.com/schema.json": invalidOverride,
		})

		err = dataSourceJsonschemaValidatorRead(d, config)
		if err == nil {
			t.Fatal("Expected error for invalid JSON in override file, got nil")
		}
		if !strings.Contains(err.Error(), "ref_override") && !strings.Contains(err.Error(), "parse") {
			t.Errorf("Expected error message about parsing, got: %v", err)
		}
	})
}
