package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/iilei/terraform-provider-jsonschema/internal/provider"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func main() {
	fmt.Println("JSON Schema Validation with Detailed Error Options")
	fmt.Println("=================================================")

	// Test schema
	schemaJSON := `{
		"type": "object",
		"required": ["name", "age", "email"],
		"properties": {
			"name": {"type": "string", "minLength": 3},
			"age": {"type": "integer", "minimum": 0, "maximum": 120},
			"email": {"type": "string", "format": "email"}
		}
	}`

	// Invalid test document
	documentJSON := `{
		"name": "Jo",
		"age": "twenty"
	}`

	// Compile schema using v6 API
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	
	var schemaData interface{}
	if err := json.Unmarshal([]byte(schemaJSON), &schemaData); err != nil {
		log.Fatalf("Schema parsing error: %v", err)
	}
	
	schemaURL := "test://schema.json"
	if err := compiler.AddResource(schemaURL, schemaData); err != nil {
		log.Fatalf("Schema resource error: %v", err)
	}
	
	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		log.Fatalf("Schema compilation error: %v", err)
	}

	// Parse document
	var document interface{}
	if err := json.Unmarshal([]byte(documentJSON), &document); err != nil {
		log.Fatalf("Document parsing error: %v", err)
	}

	// Validate and get error
	validationErr := schema.Validate(document)
	if validationErr == nil {
		fmt.Println("Document is valid!")
		return
	}

	fmt.Println("\n1. Full Error Message:")
	fmt.Println("----------------------")
	basicErr := provider.FormatValidationError(
		validationErr,
		"test://schema.json",
		documentJSON,
		"{{.FullMessage}}",
	)
	fmt.Println(basicErr.Error())

	fmt.Println("\n2. Individual Error Iteration:")
	fmt.Println("------------------------------")
	detailedErr := provider.FormatValidationError(
		validationErr,
		"test://schema.json",
		documentJSON,
		"{{range .Errors}}{{.Path}}: {{.Message}}\n{{end}}",
	)
	fmt.Println(detailedErr.Error())

	fmt.Println("\n3. Error Count and Metadata:")
	fmt.Println("----------------------------")
	
	countErr := provider.FormatValidationError(
		validationErr,
		"test://schema.json",
		documentJSON,
		"Found {{.ErrorCount}} validation errors",
	)
	fmt.Printf("%s\n", countErr.Error())

	fmt.Println("\n4. Detailed Error Information:")
	fmt.Println("------------------------------")
	detailedErr2 := provider.FormatValidationError(
		validationErr,
		"test://schema.json",
		documentJSON,
		"{{range .Errors}}Error at {{.Path}}: {{.Message}} (schema: {{.SchemaPath}})\n{{end}}",
	)
	fmt.Printf("%s\n", detailedErr2.Error())

	fmt.Println("\n5. Complete Context Example:")
	fmt.Println("----------------------------")
	allVarsErr := provider.FormatValidationError(
		validationErr,
		"test://schema.json",
		documentJSON,
		"Schema: {{.Schema}}\nErrors: {{.ErrorCount}}\n{{range .Errors}}- {{.Path}}: {{.Message}}\n{{end}}",
	)
	fmt.Printf("%s\n", allVarsErr.Error())
}