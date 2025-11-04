package provider

import (
    "fmt"
    "os"
    "path/filepath"
    "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
    "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
    "testing"
    "github.com/titanous/json5"
    "encoding/json"
)


func Test_dataSourceJsonschemaValidatorJson5(t *testing.T) {
    tests := []struct {
        name           string
        document       string
        schemaContent  string
        errorExpected bool
        expectedHash  string
    }{
        {
            name:     "JSON5 with comments and unquoted keys",
            document: `{"test": "value"}`,
            schemaContent: `{
                // This is a JSON5 schema with comments
                type: "object",
                required: ["test"], // trailing comma is valid in JSON5
            }`,
            errorExpected: false,
            // Let's compute the expected hash dynamically
            expectedHash: func() string {
                // First canonicalize the document
                canonDoc, _ := canonicalizeJSON(`{"test": "value"}`)
                
                // Convert JSON5 schema to regular JSON
                var schemaData interface{}
                json5.Unmarshal([]byte(`{
                    type: "object",
                    required: ["test"],
                }`), &schemaData)
                
                // Marshal back to canonical JSON
                canonSchema, _ := json.Marshal(sortKeys(schemaData))
                
                return hash(fmt.Sprintf("%s:%s", canonDoc, string(canonSchema)))
            }(),
        },
        {
            name:     "JSON5 with single quotes and trailing commas in arrays",
            document: `{"test": "value"}`,
            schemaContent: `{
                'type': 'object',
                'properties': {
                    'test': {
                        'type': 'string',
                    },
                },
                'required': [
                    'test',
                ],
            }`,
            errorExpected: false,
            expectedHash: hash(fmt.Sprintf("%s:%s",
                `{"test":"value"}`,
                `{"properties":{"test":{"type":"string"}},"required":["test"],"type":"object"}`)),
        },
        {
            name:     "Invalid JSON5 syntax should fail",
            document: `{"test": "value"}`,
            schemaContent: `{
                // This is invalid JSON5
                type: object, // missing quotes
                required: [test], // missing quotes
            }`,
            errorExpected: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Create temporary directory for schema files
            tmpDir := t.TempDir()
            schemaPath := filepath.Join(tmpDir, "schema.json")
            
            // Write schema to file
            if err := os.WriteFile(schemaPath, []byte(tt.schemaContent), 0644); err != nil {
                t.Fatalf("failed to write schema file: %v", err)
            }
            
            // Create a new ResourceData for testing
            d := schema.TestResourceDataRaw(t, 
                dataSourceJsonschemaValidator().Schema,
                map[string]interface{}{
                    "document": tt.document,
                    "schema":   schemaPath,
                },
            )

            // Call the read function
            err := dataSourceJsonschemaValidatorRead(d, nil)

            // Check error expectation
            if tt.errorExpected {
                if err == nil {
                    t.Errorf("expected error but got none")
                }
                return
            }

            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }

            // Verify the hash
            if got := d.Id(); got != tt.expectedHash {
                t.Errorf("expected hash = %v, got %v", tt.expectedHash, got)
            }

            // Verify the document was validated
            if validated, exists := d.GetOk("validated"); !exists {
                t.Error("validated field not set")
            } else if validated != tt.document {
                t.Errorf("validated = %v, want %v", validated, tt.document)
            }
        })
    }
}


func Test_dataSourceJsonschemaValidatorHash(t *testing.T) {
    tests := []struct {
        name          string
        document      string
        schemaContent string
        want          string // expected hash
    }{
        {
            name:          "Same logical JSON with different key order should produce same hash",
            document:      `{"b": 2, "a": 1}`,
            schemaContent: `{"type": "object", "properties": {"b": {"type": "number"}, "a": {"type": "number"}}}`,
            want:          hash(fmt.Sprintf("%s:%s", 
                `{"a":1,"b":2}`, 
                `{"properties":{"a":{"type":"number"},"b":{"type":"number"}},"type":"object"}`)),
        },
        {
            name:     "Different whitespace should produce same hash",
            document: `{
                "a": 1,
                "b": 2
            }`,
            schemaContent: `{
                "type": "object",
                "properties": {
                    "a": {"type": "number"},
                    "b": {"type": "number"}
                }
            }`,
            want:     hash(fmt.Sprintf("%s:%s", 
                `{"a":1,"b":2}`, 
                `{"properties":{"a":{"type":"number"},"b":{"type":"number"}},"type":"object"}`)),
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Create temporary directory for schema files
            tmpDir := t.TempDir()
            schemaPath := filepath.Join(tmpDir, "schema.json")
            
            // Write schema to file
            if err := os.WriteFile(schemaPath, []byte(tt.schemaContent), 0644); err != nil {
                t.Fatalf("failed to write schema file: %v", err)
            }
            
            // Canonicalize document
            canonicalDoc, err := canonicalizeJSON(tt.document)
            if err != nil {
                t.Fatalf("failed to canonicalize document: %v", err)
            }

            // Canonicalize schema
            var schemaData interface{}
            if err := json.Unmarshal([]byte(tt.schemaContent), &schemaData); err != nil {
                t.Fatalf("failed to unmarshal schema: %v", err)
            }
            canonicalSchema, err := json.Marshal(sortKeys(schemaData))
            if err != nil {
                t.Fatalf("failed to marshal schema: %v", err)
            }

            // Create composite string and hash
            compositeString := fmt.Sprintf("%s:%s", canonicalDoc, string(canonicalSchema))
            got := hash(compositeString)

            if got != tt.want {
                t.Errorf("hash = %v, want %v", got, tt.want)
            }
        })
    }
}
func Test_dataSourceJsonschemaValidatorRead(t *testing.T) {
	// Create temporary directory for schema files
	tmpDir := t.TempDir()
	
	// Write valid schema to file
	validSchemaPath := filepath.Join(tmpDir, "valid_schema.json")
	if err := os.WriteFile(validSchemaPath, []byte(schemaValid), 0644); err != nil {
		t.Fatalf("failed to write valid schema file: %v", err)
	}
	
	// Write empty schema to file
	emptySchemaPath := filepath.Join(tmpDir, "empty_schema.json")
	if err := os.WriteFile(emptySchemaPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to write empty schema file: %v", err)
	}
	
	var cases = []struct {
		document      string
		schemaPath    string
		errorExpected bool
	}{
		{"asd asdasd: ^%^*&^%", emptySchemaPath, true},
		{"{}", validSchemaPath, true},
		{`{"test": "test"}`, validSchemaPath, false},
	}

	for _, tt := range cases {
		t.Run(fmt.Sprintf("%s with error expected %t", tt.document, tt.errorExpected), func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:          func() { testAccPreCheck(t) },
				ProviderFactories: providerFactories,
				Steps: []resource.TestStep{
					{
						Config: makeDataSource(tt.document, tt.schemaPath),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.jsonschema_validator.test", "validated", fmt.Sprintf("%s\n", tt.document)),
						),
					},
				},
				ErrorCheck: func(err error) error {
					if tt.errorExpected {
						if err == nil {
							return fmt.Errorf("error expected")
						} else {
							return nil
						}
					}

					return err
				},
			})
		})
	}
}

func makeDataSource(document string, schemaPath string) string {
	return fmt.Sprintf(`
data "jsonschema_validator" "test" {
  document = <<EOF
%s
EOF
  schema   = "%s"
}
`, document, schemaPath)
}

var schemaValid = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "x-$id": "https://example.com",
  "type": "object",
  "required": ["test"]
}`
