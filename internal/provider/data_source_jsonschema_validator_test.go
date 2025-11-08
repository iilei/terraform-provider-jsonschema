package provider

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func Test_dataSourceJsonschemaValidatorRead(t *testing.T) {
	// Create temporary directory for test schema files
	tempDir, err := ioutil.TempDir("", "jsonschema_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Write schema files
	schemaFile := filepath.Join(tempDir, "test.schema.json")
	if err := ioutil.WriteFile(schemaFile, []byte(schemaValid), 0644); err != nil {
		t.Fatal(err)
	}

	json5SchemaFile := filepath.Join(tempDir, "test.schema.json5")
	if err := ioutil.WriteFile(json5SchemaFile, []byte(schemaJSON5), 0644); err != nil {
		t.Fatal(err)
	}

	var cases = []struct {
		name          string
		document      string
		schemaFile    string
		errorExpected bool
		expectedJSON  string
	}{
		{
			name:          "invalid document",
			document:      "asd asdasd: ^%^*&^%",
			schemaFile:    schemaFile,
			errorExpected: true,
		},
		{
			name:          "empty object fails required validation",
			document:      "{}",
			schemaFile:    schemaFile,
			errorExpected: true,
		},
		{
			name:          "valid document",
			document:      `{"test": "test"}`,
			schemaFile:    schemaFile,
			errorExpected: false,
			expectedJSON:  `{"test":"test"}`,
		},
		{
			name:          "JSON5 document with comments",
			document:      `{"test": "test", /* comment */ }`,
			schemaFile:    schemaFile,
			errorExpected: false,
			expectedJSON:  `{"test":"test"}`,
		},
		{
			name:          "JSON5 schema with JSON document",
			document:      `{"name": "example", "age": 25}`,
			schemaFile:    json5SchemaFile,
			errorExpected: false,
			expectedJSON:  `{"age":25,"name":"example"}`,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:          func() { testAccPreCheck(t) },
				ProviderFactories: providerFactories,
				Steps: []resource.TestStep{
					{
						Config: makeDataSourceWithFile(tt.document, tt.schemaFile),
						Check: resource.ComposeAggregateTestCheckFunc(
							func() resource.TestCheckFunc {
								if tt.expectedJSON != "" {
									return resource.TestCheckResourceAttr("data.jsonschema_validator.test", "validated", tt.expectedJSON)
								}
								return resource.TestCheckResourceAttrSet("data.jsonschema_validator.test", "validated")
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
	tempDir, err := ioutil.TempDir("", "jsonschema_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	schemaFile := filepath.Join(tempDir, "test.schema.json")
	if err := ioutil.WriteFile(schemaFile, []byte(schemaValidDraft04), 0644); err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: makeProviderConfigTest(schemaFile),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jsonschema_validator.test", "validated", `{"test":"value"}`),
				),
			},
		},
	})
}

// TestMultipleSchemasInSameDirectory verifies that schemas with different filenames work correctly
// This is a breaking change from the previous implementation that hardcoded "schema.json" in the URL
func TestMultipleSchemasInSameDirectory(t *testing.T) {
	// Create temporary directory for test schema files
	tempDir, err := ioutil.TempDir("", "jsonschema_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create multiple schema files in the same directory with different names
	schema1File := filepath.Join(tempDir, "user.schema.json")
	if err := ioutil.WriteFile(schema1File, []byte(userSchema), 0644); err != nil {
		t.Fatal(err)
	}

	schema2File := filepath.Join(tempDir, "product.schema.json")
	if err := ioutil.WriteFile(schema2File, []byte(productSchema), 0644); err != nil {
		t.Fatal(err)
	}

	var cases = []struct {
		name          string
		document      string
		schemaFile    string
		errorExpected bool
		expectedJSON  string
	}{
		{
			name:          "validate user document",
			document:      `{"name": "John", "email": "john@example.com"}`,
			schemaFile:    schema1File,
			errorExpected: false,
			expectedJSON:  `{"email":"john@example.com","name":"John"}`,
		},
		{
			name:          "validate product document",
			document:      `{"sku": "ABC123", "price": 99.99}`,
			schemaFile:    schema2File,
			errorExpected: false,
			expectedJSON:  `{"price":99.99,"sku":"ABC123"}`,
		},
		{
			name:          "invalid user document against user schema",
			document:      `{"name": "John"}`, // missing email
			schemaFile:    schema1File,
			errorExpected: true,
		},
		{
			name:          "invalid product document against product schema",
			document:      `{"sku": "ABC123"}`, // missing price
			schemaFile:    schema2File,
			errorExpected: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:          func() { testAccPreCheck(t) },
				ProviderFactories: providerFactories,
				Steps: []resource.TestStep{
					{
						Config: makeDataSourceWithFile(tt.document, tt.schemaFile),
						Check: resource.ComposeAggregateTestCheckFunc(
							func() resource.TestCheckFunc {
								if tt.expectedJSON != "" {
									return resource.TestCheckResourceAttr("data.jsonschema_validator.test", "validated", tt.expectedJSON)
								}
								return resource.TestCheckResourceAttrSet("data.jsonschema_validator.test", "validated")
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

func makeDataSourceWithFile(document string, schemaFile string) string {
	return fmt.Sprintf(`
data "jsonschema_validator" "test" {
  document = %q
  schema   = %q
}
`, document, schemaFile)
}

func makeProviderConfigTest(schemaFile string) string {
	return fmt.Sprintf(`
provider "jsonschema" {
  schema_version = "draft-04"
}

data "jsonschema_validator" "test" {
  document = "{\"test\": \"value\"}"
  schema   = %q
}
`, schemaFile)
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
